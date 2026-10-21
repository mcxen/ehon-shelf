// ==UserScript==
// @name         EhonShelf 数据导出
// @namespace    https://github.com/mcxen/ehon-shelf
// @version      0.2.3
// @description  在浏览器中把「艶本資料データベース」的作品导出为 EhonShelf 可导入的数据包（zip）
// @author       mcxen
// @match        https://lapis.nichibun.ac.jp/enp/*
// @require      https://cdn.jsdelivr.net/npm/fflate@0.8.2/umd/index.js
// @grant        none
// @run-at       document-idle
// @downloadURL  https://raw.githubusercontent.com/mcxen/ehon-shelf/main/userscript/ehonshelf-export.user.js
// @updateURL    https://raw.githubusercontent.com/mcxen/ehon-shelf/main/userscript/ehonshelf-export.user.js
// ==/UserScript==

(function () {
  'use strict';

  // ---------- parsers (ported from crawler.go) ----------
  const BIB_KEYS = ['通番', '絵師名', 'よみがな１', '作品名', 'よみがな２', '技法', '版型', '点数', '板行年', '西暦', '作品ﾌｧｲﾙNo.', '備考', '請求記号', '図書ID'];
  const ENTITIES = { amp: '&', lt: '<', gt: '>', quot: '"', apos: "'", nbsp: '\u00a0' };

  function unescapeHTML(s) {
    return s.replace(/&(#x[0-9a-fA-F]+|#\d+|[a-zA-Z]+);/g, (m, e) => {
      if (e[0] === '#') {
        const cp = e[1] === 'x' || e[1] === 'X' ? parseInt(e.slice(2), 16) : parseInt(e.slice(1), 10);
        return isFinite(cp) ? String.fromCodePoint(cp) : m;
      }
      return Object.prototype.hasOwnProperty.call(ENTITIES, e) ? ENTITIES[e] : m;
    });
  }

  // Go's \s does not include U+00A0, so collapse only ASCII whitespace.
  function txt(s) {
    return unescapeHTML(s.replace(/<[^>]+>/g, ' ')).replace(/[\t\n\f\r ]+/g, ' ').trim();
  }

  function parseList(h) {
    const re = /href="\/enp\/Cover\/List\/(\d+)"[\s\S]*?class="s">([\s\S]*?)<\/span>[\s\S]*?class="ss">([\s\S]*?)<\/span>[\s\S]*?class="sblue">([\s\S]*?)<\/span>/g;
    const out = [];
    let m;
    while ((m = re.exec(h))) out.push({ id: +m[1], title: txt(m[2]), title_kana: txt(m[3]), artist: txt(m[4]) });
    return out;
  }

  function parseShoshi(h) {
    const cells = [];
    const re = /<td[^>]*>([\s\S]*?)<\/td>/g;
    let m;
    while ((m = re.exec(h))) cells.push(txt(m[1]));
    const d = {};
    for (let i = 0; i + 1 < cells.length; i++) {
      const k = cells[i];
      let v = cells[i + 1];
      if (BIB_KEYS.includes(k) && !(k in d)) {
        if (BIB_KEYS.includes(v)) v = '';
        d[k] = v;
      }
    }
    return d;
  }

  function parseVolumes(h, id) {
    const re = new RegExp('/enp/Picture/List/' + id + '/(\\d+)/1"', 'g');
    const vs = new Set();
    let m;
    while ((m = re.exec(h))) vs.add(+m[1]);
    const out = [...vs].sort((a, b) => a - b);
    return out.length ? out : [1];
  }

  function parseVolumePage(h, id, vol) {
    let re = new RegExp('href="/enp/Picture/View/' + id + '/' + vol + '/(\\d+)"\\s*>?\\s*<img src="/enp/image/2/([^"]+)\\.jpg"', 'g');
    let ms = [...h.matchAll(re)];
    if (!ms.length) {
      re = new RegExp('View/' + id + '/' + vol + '/(\\d+)"[\\s\\S]{0,200}?image/2/([^"]+)\\.jpg', 'g');
      ms = [...h.matchAll(re)];
    }
    return ms.map((m) => ({ n: +m[1], name: m[2] }));
  }

  function hasNextListPage(h, id, vol, pg) {
    return h.includes('Picture/List/' + id + '/' + vol + '/' + (pg + 1) + '"');
  }

  // Largest embedded DCTDecode JPEG stream, unchanged.
  function jpegFromPDF(pdf) {
    let s = '';
    for (let i = 0; i < pdf.length; i += 0x8000) s += String.fromCharCode.apply(null, pdf.subarray(i, i + 0x8000));
    const re = /\/DCTDecode[\s\S]*?stream\r?\n/g;
    let best = null, m;
    while ((m = re.exec(s))) {
      const start = m.index + m[0].length;
      const end = s.indexOf('endstream', start);
      if (end < 0) continue;
      let e = end;
      while (e > start && (pdf[e - 1] === 0x0a || pdf[e - 1] === 0x0d)) e--;
      if (e - start > 2 && pdf[start] === 0xff && pdf[start + 1] === 0xd8 && (!best || e - start > best.length)) best = pdf.subarray(start, e);
    }
    return best;
  }

  function pad4(n) { return String(n).padStart(4, '0'); }

  // meta.json with the same key order as Meta in store.go.
  function buildMeta(item, bib, vols, pages) {
    const date = bib['板行年'] || bib['西暦'] || '';
    return {
      id: item.id, title: item.title, title_kana: item.title_kana, artist: item.artist,
      source: 'https://lapis.nichibun.ac.jp/enp/Cover/List/' + item.id,
      bibliographic: bib, volumes: vols, pages: pages, date: date,
      page_count: pages.length, complete: true,
    };
  }

  function parseRange(s, all) {
    s = (s || '').trim();
    if (!s || s === 'all' || s === '全部') return all.slice();
    const want = new Set();
    for (const part of s.split(/[,，\s]+/)) {
      if (!part) continue;
      const m = part.match(/^(\d+)\s*[-~～]\s*(\d+)$/);
      if (m) { for (let i = +m[1]; i <= +m[2]; i++) want.add(i); }
      else if (/^\d+$/.test(part)) want.add(+part);
    }
    return all.filter((it) => want.has(it.id));
  }

  const api = { txt, parseList, parseShoshi, parseVolumes, parseVolumePage, hasNextListPage, jpegFromPDF, buildMeta, parseRange, BIB_KEYS };
  if (typeof module !== 'undefined' && module.exports) { module.exports = api; return; }
  if (typeof document === 'undefined') return;

  // ---------- polite fetcher ----------
  const MIN_GAP = 1000;
  let lastReq = 0;
  class Refused extends Error {}
  const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

  async function get(path, kind) {
    if (!path.startsWith('/enp/')) throw new Error('仅允许同源请求：' + path);
    const wait = lastReq + MIN_GAP - Date.now();
    if (wait > 0) await sleep(wait);
    lastReq = Date.now();
    const r = await fetch(path, { credentials: 'same-origin' });
    lastReq = Date.now();
    if (r.status === 403 || r.status === 429) throw new Refused('服务器返回 ' + r.status + '，已停止。请稍后再继续。');
    if (!r.ok) throw new Error('HTTP ' + r.status + ' ' + path);
    return kind === 'text' ? r.text() : new Uint8Array(await r.arrayBuffer());
  }

  // ---------- state ----------
  const LS_DONE = 'ehonshelf.exported';
  const LS_OPT = 'ehonshelf.options';
  const loadDone = () => new Set(JSON.parse(localStorage.getItem(LS_DONE) || '[]'));
  const saveDone = (s) => localStorage.setItem(LS_DONE, JSON.stringify([...s].sort((a, b) => a - b)));
  const opts = Object.assign({ range: 'all', batch: 5 }, JSON.parse(localStorage.getItem(LS_OPT) || '{}'));

  let running = false, stopReq = false;

  // ---------- UI ----------
  const box = document.createElement('div');
  box.style.cssText = 'position:fixed;right:12px;bottom:12px;z-index:2147483647;background:#fff;color:#222;border:1px solid #999;border-radius:6px;padding:10px;font:13px/1.5 sans-serif;width:300px;box-shadow:0 2px 8px rgba(0,0,0,.25)';
  box.innerHTML =
    '<div style="display:flex;justify-content:space-between;font-weight:bold;margin-bottom:6px"><span>EhonShelf 数据导出</span><a href="#" data-k="fold" style="text-decoration:none">收起</a></div>' +
    '<div data-k="body">' +
    '<label>作品编号（如 1-50,81 或 all）<input data-k="range" style="width:100%;box-sizing:border-box"></label>' +
    '<label>每个数据包作品数 <input data-k="batch" type="number" min="1" max="50" style="width:60px"></label>' +
    '<div style="margin:6px 0;display:flex;gap:6px;flex-wrap:wrap">' +
    '<button data-k="start">开始</button><button data-k="pause" disabled>暂停</button><button data-k="reset">清除已导出记录</button></div>' +
    '<progress data-k="bar" value="0" max="1" style="width:100%"></progress>' +
    '<div data-k="msg" style="white-space:pre-wrap;color:#555">就绪。每秒最多 1 个请求，导出的作品会记录在本浏览器中，下次自动跳过。</div>' +
    '</div>';
  document.body.appendChild(box);
  const $ = (k) => box.querySelector('[data-k="' + k + '"]');
  $('range').value = opts.range;
  $('batch').value = opts.batch;
  const say = (s) => { $('msg').textContent = s; };
  $('fold').onclick = (e) => { e.preventDefault(); const b = $('body'); const hide = b.style.display !== 'none'; b.style.display = hide ? 'none' : ''; $('fold').textContent = hide ? '展开' : '收起'; };
  $('reset').onclick = () => { if (!running && confirm('确定清除已导出记录？')) { localStorage.removeItem(LS_DONE); say('已清除已导出记录。'); } };
  $('pause').onclick = () => { stopReq = true; $('pause').disabled = true; say('正在暂停，当前作品完成前已完成的作品会先打包下载…'); };
  $('start').onclick = () => { if (!running) run().catch((e) => say('出错：' + e.message)); };

  function saveZip(files, ids) {
    const data = fflate.zipSync(files, { level: 0 });
    const blob = new Blob([data], { type: 'application/zip' });
    const a = document.createElement('a');
    a.href = URL.createObjectURL(blob);
    a.download = 'ehonshelf-' + ids[0] + '-' + ids[ids.length - 1] + '.zip';
    document.body.appendChild(a);
    a.click();
    a.remove();
    setTimeout(() => URL.revokeObjectURL(a.href), 60000);
  }

  async function exportWork(item, files, onPage) {
    const id = item.id;
    const cover = await get('/enp/Cover/List/' + id, 'text');
    const vols = parseVolumes(cover, id);
    const bib = parseShoshi(await get('/enp/Book/Shoshi/' + id, 'text'));
    const pages = [];
    for (const v of vols) {
      const seen = new Set();
      const refs = [];
      for (let pg = 1; ; pg++) {
        if (stopReq) return false;
        const h = await get('/enp/Picture/ListContents/' + id + '/' + v + '/' + pg, 'text');
        for (const r of parseVolumePage(h, id, v)) {
          const k = r.n + '\u0000' + r.name;
          if (!seen.has(k)) { seen.add(k); refs.push(r); }
        }
        if (!hasNextListPage(h, id, v, pg)) break;
      }
      refs.sort((a, b) => a.n - b.n || (a.name < b.name ? -1 : a.name > b.name ? 1 : 0));
      for (const r of refs) pages.push({ vol: v, n: r.n, name: r.name, pdf: '/enp/image/1/' + r.name + '.pdf', thumb: '/enp/image/2/' + r.name + '.jpg', file: '' });
    }
    pages.forEach((p, i) => { p.file = 'pages/' + pad4(i + 1) + '.jpg'; });
    const out = {};
    for (let i = 0; i < pages.length; i++) {
      if (stopReq) return false;
      const p = pages[i];
      onPage(i, pages.length);
      let jpg = jpegFromPDF(await get(p.pdf));
      if (!jpg) { jpg = await get(p.thumb); p.lowres = true; }
      out[id + '/' + p.file] = jpg;
    }
    if (pages.length) out[id + '/cover_thumb.jpg'] = await get(pages[0].thumb);
    const meta = buildMeta(item, bib, vols, pages);
    out[id + '/meta.json'] = new TextEncoder().encode(JSON.stringify(meta, null, 1));
    Object.assign(files, out);
    return true;
  }

  async function run() {
    opts.range = $('range').value.trim() || 'all';
    opts.batch = Math.max(1, Math.min(50, parseInt($('batch').value, 10) || 5));
    localStorage.setItem(LS_OPT, JSON.stringify(opts));
    running = true; stopReq = false;
    $('start').disabled = true; $('pause').disabled = false;
    let files = {}, ids = [];
    const done = loadDone();
    const flush = () => {
      if (!ids.length) return;
      saveZip(files, ids);
      ids.forEach((i) => done.add(i));
      saveDone(done);
      files = {}; ids = [];
    };
    try {
      say('读取作品目录…');
      const all = parseList(await get('/enp/SearchBooks/BySakuContents', 'text'));
      if (!all.length) throw new Error('目录为空，网站结构可能已变化');
      const todo = parseRange(opts.range, all).filter((it) => !done.has(it.id));
      $('bar').max = Math.max(1, todo.length); $('bar').value = 0;
      for (let k = 0; k < todo.length; k++) {
        if (stopReq) break;
        const it = todo[k];
        const ok = await exportWork(it, files, (i, n) => say('(' + (k + 1) + '/' + todo.length + ') ' + it.id + ' ' + it.title + '\n第 ' + (i + 1) + ' / ' + n + ' 页'));
        if (!ok) break;
        ids.push(it.id);
        $('bar').value = k + 1;
        if (ids.length >= opts.batch) flush();
      }
      flush();
      say(stopReq ? '已暂停。再次点击「开始」可继续，已导出的作品会跳过。' : '完成：共处理 ' + todo.length + ' 部作品。');
    } catch (e) {
      try { flush(); } catch (_) { /* ignore */ }
      say((e instanceof Refused ? '' : '出错：') + e.message + '\n已完成的作品已打包下载，可稍后点击「开始」继续。');
    } finally {
      running = false;
      $('start').disabled = false; $('pause').disabled = true;
    }
  }
})();
