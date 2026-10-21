import json, os, shutil, sys, threading, time, urllib.request, webbrowser, zipfile
from http.server import ThreadingHTTPServer, SimpleHTTPRequestHandler
from urllib.parse import urlparse, unquote
from .crawler import Crawler, make_cover, MIN_DELAY

REPO = "mcxen/ehon-shelf"
SAMPLE_URL = f"https://github.com/{REPO}/releases/latest/download/sample-data.zip"
HOME = os.path.join(os.path.expanduser("~"), "EhonShelf")
CONF = os.path.join(HOME, "config.json")
BUNDLE = getattr(sys, "_MEIPASS", os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
WEB = os.path.join(os.path.dirname(os.path.abspath(__file__)), "web")
if not os.path.isdir(WEB): WEB = os.path.join(BUNDLE, "ehon_shelf", "web")


def load_conf():
    c = {"data_dir": os.path.join(HOME, "data"), "delay": 1.2}
    if os.path.exists(CONF): c.update(json.load(open(CONF, encoding="utf-8")))
    return c


def save_conf(c):
    os.makedirs(HOME, exist_ok=True); json.dump(c, open(CONF, "w", encoding="utf-8"), ensure_ascii=False, indent=1)


class App:
    def __init__(self):
        self.conf = load_conf(); os.makedirs(self.conf["data_dir"], exist_ok=True)
        self.crawler = Crawler(self.conf["data_dir"], self.conf["delay"])
        self.sample = {"state": "idle", "message": ""}

    @property
    def data_dir(self): return self.conf["data_dir"]

    def library(self):
        out = []
        for name in os.listdir(self.data_dir):
            mp = os.path.join(self.data_dir, name, "meta.json")
            if not name.isdigit() or not os.path.exists(mp): continue
            try: m = json.load(open(mp, encoding="utf-8"))
            except Exception: continue
            d = os.path.join(self.data_dir, name)
            try: make_cover(d)
            except Exception: pass
            cover = "cover.jpg" if os.path.exists(os.path.join(d, "cover.jpg")) else "cover_thumb.jpg"
            out.append({k: m.get(k) for k in ("id", "title", "title_kana", "artist", "date", "page_count", "complete")}
                       | {"cover": f"data/{name}/{cover}"})
        return sorted(out, key=lambda b: b["id"])

    def remote(self, refresh=False):
        cache = os.path.join(self.data_dir, "remote_catalog.json")
        if os.path.exists(cache) and not refresh:
            return json.load(open(cache, encoding="utf-8"))
        items = self.crawler.list_items()
        json.dump(items, open(cache, "w", encoding="utf-8"), ensure_ascii=False, indent=1)
        return items

    def import_sample(self, path):
        with zipfile.ZipFile(path) as z:
            for n in z.namelist():
                if n.startswith("/") or ".." in n: continue
                tgt = os.path.join(self.data_dir, n)
                if n.endswith("/") or os.path.exists(tgt): continue
                os.makedirs(os.path.dirname(tgt), exist_ok=True)
                with z.open(n) as s, open(tgt, "wb") as f: shutil.copyfileobj(s, f)

    def download_sample(self):
        def run():
            self.sample.update(state="running", message="正在下载示例数据")
            tmp = os.path.join(self.data_dir, "sample-data.zip.part")
            try:
                req = urllib.request.Request(SAMPLE_URL, headers={"User-Agent": "EhonShelf"})
                with urllib.request.urlopen(req, timeout=60) as r, open(tmp, "wb") as f:
                    total = int(r.headers.get("Content-Length") or 0); got = 0
                    while chunk := r.read(1 << 20):
                        f.write(chunk); got += len(chunk)
                        self.sample["message"] = f"已下载 {got >> 20} MB" + (f" / {total >> 20} MB" if total else "")
                self.import_sample(tmp); self.sample.update(state="done", message="示例数据已导入")
            except Exception as e:
                self.sample.update(state="error", message=f"失败: {e}")
            finally:
                if os.path.exists(tmp): os.remove(tmp)
        threading.Thread(target=run, daemon=True).start()

    def set_conf(self, body):
        if self.crawler.running(): raise ValueError("下载进行中，请先暂停")
        if "delay" in body: self.conf["delay"] = max(MIN_DELAY, float(body["delay"]))
        if body.get("data_dir"):
            d = os.path.abspath(os.path.expanduser(body["data_dir"])); os.makedirs(d, exist_ok=True)
            self.conf["data_dir"] = d
        save_conf(self.conf)
        self.crawler = Crawler(self.conf["data_dir"], self.conf["delay"])


def first_run(app):
    """Seed an empty library from a bundled sample-data.zip, if any."""
    if any(n.isdigit() for n in os.listdir(app.data_dir)): return
    for p in (os.path.join(BUNDLE, "sample-data.zip"), os.path.join(os.path.dirname(sys.executable), "sample-data.zip")):
        if os.path.exists(p):
            print("导入内置示例数据:", p); app.import_sample(p); return


def make_handler(app):
    class H(SimpleHTTPRequestHandler):
        def log_message(self, *a): pass

        def send_json(self, obj, code=200):
            b = json.dumps(obj, ensure_ascii=False).encode()
            self.send_response(code); self.send_header("Content-Type", "application/json; charset=utf-8")
            self.send_header("Content-Length", str(len(b))); self.send_header("Cache-Control", "no-store")
            self.end_headers(); self.wfile.write(b)

        def translate_path(self, path):
            p = unquote(urlparse(path).path)
            if p.startswith("/data/"):
                root, rel = app.data_dir, p[len("/data/"):]
            else:
                root, rel = WEB, p.lstrip("/") or "index.html"
            full = os.path.realpath(os.path.join(root, rel))
            return full if full.startswith(os.path.realpath(root)) else os.path.join(WEB, "404")

        def do_GET(self):
            p = urlparse(self.path).path
            try:
                if p == "/api/library": return self.send_json(app.library())
                if p == "/api/config": return self.send_json(app.conf | {"min_delay": MIN_DELAY})
                if p == "/api/remote": return self.send_json(app.remote("refresh" in self.path))
                if p == "/api/status":
                    s = dict(app.crawler.status); s["queue_len"] = len(s.pop("queue"))
                    return self.send_json(s | {"sample": app.sample})
            except Exception as e:
                return self.send_json({"error": str(e)}, 500)
            return super().do_GET()

        def do_POST(self):
            p = urlparse(self.path).path
            n = int(self.headers.get("Content-Length") or 0)
            body = json.loads(self.rfile.read(n) or b"{}")
            try:
                if p == "/api/config": app.set_conf(body); return self.send_json(app.conf)
                if p == "/api/download":
                    items = app.remote()
                    if not body.get("all"):
                        ids = set(map(int, body.get("ids", []))); items = [i for i in items if i["id"] in ids]
                    return self.send_json({"ok": app.crawler.start(items)})
                if p == "/api/pause": app.crawler.pause(); return self.send_json({"ok": True})
                if p == "/api/resume": return self.send_json({"ok": app.crawler.resume()})
                if p == "/api/sample": app.download_sample(); return self.send_json({"ok": True})
            except Exception as e:
                return self.send_json({"error": str(e)}, 400)
            self.send_json({"error": "not found"}, 404)
    return H


def main(argv=None):
    import argparse
    ap = argparse.ArgumentParser(description="绘本书架 (Ehon Shelf)")
    ap.add_argument("--port", type=int, default=8765); ap.add_argument("--no-browser", action="store_true")
    a = ap.parse_args(argv)
    app = App(); first_run(app)
    srv = None
    for port in range(a.port, a.port + 20):
        try: srv = ThreadingHTTPServer(("127.0.0.1", port), make_handler(app)); break
        except OSError: continue
    url = f"http://127.0.0.1:{port}/"
    print(f"绘本书架已启动: {url}  数据目录: {app.data_dir}  (Ctrl+C 退出)", flush=True)
    if not a.no_browser: threading.Timer(0.8, webbrowser.open, (url,)).start()
    try: srv.serve_forever()
    except KeyboardInterrupt: pass
