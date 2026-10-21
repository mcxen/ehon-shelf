"""Polite, resumable crawler for 艶本資料データベース (lapis.nichibun.ac.jp/enp)."""
import html, json, os, re, sys, threading, time, urllib.robotparser
import requests

BASE = "https://lapis.nichibun.ac.jp"
UA = "NichibunEnpArchiver/1.0 (offline mirror for site maintainer; polite 1 req/s)"
MIN_DELAY = 1.0
BIB_KEYS = ("通番","絵師名","よみがな１","作品名","よみがな２","技法","版型","点数","板行年","西暦","作品ﾌｧｲﾙNo.","備考","請求記号","図書ID")


class Paused(Exception):
    pass


def txt(s): return re.sub(r"\s+", " ", html.unescape(re.sub(r"<[^>]+>", " ", s))).strip()


def jpeg_from_pdf(pdf):
    """Return the largest embedded DCTDecode JPEG stream (lossless extraction)."""
    best = None
    for m in re.finditer(rb"/DCTDecode.*?stream\r?\n", pdf, re.S):
        start = m.end(); end = pdf.find(b"endstream", start)
        data = pdf[start:end].rstrip(b"\r\n")
        if data[:2] == b"\xff\xd8" and (best is None or len(data) > len(best)): best = data
    return best


def make_cover(item_dir, width=400):
    """Sharp cover from pages/0001.jpg (resized)."""
    src = os.path.join(item_dir, "pages", "0001.jpg"); dst = os.path.join(item_dir, "cover.jpg")
    if not os.path.exists(src): return None
    if os.path.exists(dst) and os.path.getmtime(dst) >= os.path.getmtime(src): return dst
    from PIL import Image
    im = Image.open(src); im.draft("RGB", (width * 2, width * 4)); im = im.convert("RGB")
    if im.width > width: im = im.resize((width, round(im.height * width / im.width)), Image.LANCZOS)
    im.save(dst + ".part", "JPEG", quality=85, optimize=True); os.replace(dst + ".part", dst)
    return dst


class Crawler:
    def __init__(self, data_dir, delay=1.2, log=print):
        self.data_dir = data_dir; self.delay = max(MIN_DELAY, delay); self.log = log
        self.s = requests.Session(); self.s.headers["User-Agent"] = UA
        self._last = 0.0; self.rp = None
        self.pause_ev = threading.Event(); self.lock = threading.Lock(); self.thread = None
        self.status = {"state": "idle", "queue": [], "current": None, "item_done": 0, "item_total": 0,
                       "done_items": 0, "total_items": 0, "message": "", "errors": []}

    # --- HTTP ---
    def _robots(self):
        if self.rp is None:
            rp = urllib.robotparser.RobotFileParser(BASE + "/robots.txt")
            try:
                r = self.s.get(BASE + "/robots.txt", timeout=30)
                rp.parse(r.text.splitlines() if r.status_code == 200 else [])
            except requests.RequestException:
                rp.parse([])
            self.rp = rp
        return self.rp

    def get(self, path, binary=False):
        url = BASE + path
        if not self._robots().can_fetch(UA, url):
            raise RuntimeError("robots.txt disallows " + url)
        for attempt in range(6):
            if self.pause_ev.is_set(): raise Paused()
            wait = self.delay - (time.time() - self._last)
            if wait > 0: time.sleep(wait)
            self._last = time.time()
            try:
                r = self.s.get(url, timeout=60)
                if r.status_code == 200:
                    return r.content if binary else r.content.decode("utf-8", "replace")
                if r.status_code == 404: return None
                self.log(f"HTTP {r.status_code} {url}")
            except requests.RequestException as e:
                self.log(f"error {e} {url}")
            time.sleep(min(120, 5 * 2 ** attempt))
        raise RuntimeError("giving up " + url)

    # --- parsing ---
    def list_items(self):
        h = self.get("/enp/SearchBooks/BySakuContents") or ""
        items = []
        for m in re.finditer(r'href="/enp/Cover/List/(\d+)".*?class="s">(.*?)</span>.*?class="ss">(.*?)</span>.*?class="sblue">(.*?)</span>', h, re.S):
            items.append({"id": int(m[1]), "title": txt(m[2]), "title_kana": txt(m[3]), "artist": txt(m[4])})
        return items

    def shoshi(self, iid):
        h = self.get(f"/enp/Book/Shoshi/{iid}") or ""
        cells = [txt(c) for c in re.findall(r"<td[^>]*>(.*?)</td>", h, re.S)]
        d = {}
        for k, v in zip(cells, cells[1:]):
            if k in BIB_KEYS and k not in d: d[k] = "" if v in BIB_KEYS else v
        return d

    def volume_pages(self, iid, vol):
        names, pg = [], 1
        while True:
            h = self.get(f"/enp/Picture/ListContents/{iid}/{vol}/{pg}") or ""
            found = re.findall(r'href="/enp/Picture/View/%d/%d/(\d+)"\s*>?\s*<img src="/enp/image/2/([^"]+)\.jpg"' % (iid, vol), h)
            if not found:
                found = re.findall(r'View/%d/%d/(\d+)".{0,200}?image/2/([^"]+)\.jpg' % (iid, vol), h, re.S)
            for n, f in found:
                if (int(n), f) not in names: names.append((int(n), f))
            if f'Picture/List/{iid}/{vol}/{pg+1}"' not in h: break
            pg += 1
        return sorted(names)

    def crawl_item(self, it):
        iid = it["id"]; d = os.path.join(self.data_dir, str(iid)); os.makedirs(os.path.join(d, "pages"), exist_ok=True)
        mp = os.path.join(d, "meta.json")
        meta = json.load(open(mp, encoding="utf-8")) if os.path.exists(mp) else None
        if not meta or not meta.get("pages"):
            self.status["message"] = "读取目录信息"
            cover = self.get(f"/enp/Cover/List/{iid}") or ""
            vols = sorted({int(v) for v in re.findall(r'/enp/Picture/List/%d/(\d+)/1"' % iid, cover)}) or [1]
            meta = dict(it, source=f"{BASE}/enp/Cover/List/{iid}", bibliographic=self.shoshi(iid), volumes=vols, pages=[])
            for v in vols:
                for n, f in self.volume_pages(iid, v):
                    meta["pages"].append({"vol": v, "n": n, "name": f, "pdf": f"/enp/image/1/{f}.pdf", "thumb": f"/enp/image/2/{f}.jpg"})
            for i, p in enumerate(meta["pages"], 1): p["file"] = f"pages/{i:04d}.jpg"
            bib = meta["bibliographic"]; meta["date"] = bib.get("板行年") or bib.get("西暦", "")
            meta["page_count"] = len(meta["pages"])
            json.dump(meta, open(mp, "w", encoding="utf-8"), ensure_ascii=False, indent=1)
        self.status["item_total"] = len(meta["pages"])
        for i, p in enumerate(meta["pages"]):
            self.status["item_done"] = i
            fp = os.path.join(d, p["file"])
            if os.path.exists(fp) and os.path.getsize(fp) > 0: continue
            self.status["message"] = f"下载 {p['file']}"
            pdf = self.get(p["pdf"], binary=True)
            jpg = jpeg_from_pdf(pdf) if pdf else None
            if not jpg:
                self.log(f"no full-res for {p['name']}, using thumbnail")
                jpg = self.get(p["thumb"], binary=True); p["lowres"] = True
            open(fp + ".part", "wb").write(jpg); os.replace(fp + ".part", fp)
        self.status["item_done"] = len(meta["pages"])
        make_cover(d)
        meta["complete"] = True
        json.dump(meta, open(mp, "w", encoding="utf-8"), ensure_ascii=False, indent=1)
        return meta

    # --- background job ---
    def running(self): return self.thread is not None and self.thread.is_alive()

    def start(self, items):
        with self.lock:
            if self.running(): return False
            self.pause_ev.clear()
            self.status.update(state="running", queue=[dict(i) for i in items], done_items=0,
                               total_items=len(items), errors=[], message="")
            self.thread = threading.Thread(target=self._run, daemon=True); self.thread.start()
            return True

    def pause(self):
        if self.running(): self.pause_ev.set(); self.status["state"] = "pausing"

    def resume(self):
        return self.start(self.status["queue"]) if self.status["queue"] else False

    def _run(self):
        st = self.status
        try:
            while st["queue"]:
                it = st["queue"][0]; st["current"] = it; st["item_done"] = 0; st["item_total"] = 0
                try:
                    self.crawl_item(it)
                except Paused:
                    raise
                except Exception as e:  # keep going with next item
                    st["errors"].append(f"{it['id']}: {e}"); self.log(f"item {it['id']} failed: {e}")
                st["queue"].pop(0); st["done_items"] += 1
            st.update(state="done", current=None, message="全部完成")
        except Paused:
            st.update(state="paused", message="已暂停")
        except Exception as e:
            st.update(state="error", message=str(e))
