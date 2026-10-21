// EhonShelf: a native bookshelf reader for the 艶本資料データベース of
// 国際日本文化研究センター, with a built-in polite crawler.
package main

import (
	"fmt"
	"image"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/egoist/mygo"
	_ "github.com/egoist/mygo/locales/zh-Hans"
	"github.com/egoist/mygo/ui"
)

// proxyLabel is drawn next to the proxy field. ui.SearchField's own Label
// is not rendered, and a bare box with a grey placeholder was invisible to
// users looking for the setting.
const proxyLabel = "代理（站点限制地区，直连会 403，必须填）"

const (
	viewShelf = iota
	viewReader
	viewSettings
	viewLogs
)

type app struct {
	conf *Config
	cr   *Crawler
	// update runs fn on the main thread and redraws (Window.Update);
	// async runs work off the main thread. Tests make both synchronous.
	update func(fn func())
	async  func(fn func())
	win    *mygo.Window

	view int

	// shelf
	books    []Book
	covers   map[int]*ui.Bitmap
	query    string
	artist   string
	artists  []string
	filtered []int
	grid     ui.GridState
	scanning bool

	// reader
	book    *Book
	pos     int
	rtl     bool
	spread  bool
	zoom    float64
	jump    float64
	pages   map[int]*ui.Bitmap
	loading map[int]bool

	// settings
	proxy         string
	status        Status
	remote        []Item
	remoteFilter  []int
	remoteQuery   string
	remoteMsg     string
	remoteLoading bool
	selected      map[int]bool
	list          ui.ListState
	delay         float64
	confMsg       string
	sampleMsg     string
	sampleBusy    bool

	// logs
	logs      *logBuffer
	logScroll float64
}

func newApp() *app {
	conf := LoadConfig()
	_ = os.MkdirAll(conf.DataDir, 0o755)
	a := &app{conf: conf, covers: map[int]*ui.Bitmap{}, pages: map[int]*ui.Bitmap{}, loading: map[int]bool{},
		selected: map[int]bool{}, proxy: conf.Proxy, artist: "全部绘师", rtl: true, spread: conf.Spread, zoom: 1, delay: conf.Delay,
		logs: newLogBuffer(800)}
	if conf.RTL != nil {
		a.rtl = *conf.RTL
	}
	a.update = func(fn func()) { fn() }
	a.async = func(fn func()) { go fn() }
	a.newCrawler()
	a.logs.Append("EhonShelf 启动，数据目录 %s", conf.DataDir)
	return a
}

func (a *app) newCrawler() {
	a.cr = NewCrawler(a.conf.DataDir, a.conf.Delay)
	a.cr.OnChange = func() {
		s := a.cr.Snapshot()
		a.update(func() {
			done := s.DoneItems != a.status.DoneItems || (s.State != a.status.State && s.State != "running")
			a.status = s
			if done {
				a.rescan()
			}
		})
	}
	a.status = a.cr.Snapshot()
}

// --- library ---

func (a *app) rescan() {
	if a.scanning {
		return
	}
	a.scanning = true
	dir := a.conf.DataDir
	known := map[int]bool{}
	for id := range a.covers {
		known[id] = true
	}
	a.async(func() {
		books := ScanLibrary(dir)
		imgs := map[int]image.Image{}
		for _, b := range books {
			if known[b.ID] || b.Cover == "" {
				continue
			}
			if img, err := loadScaled(b.Cover, 0, 600); err == nil {
				imgs[b.ID] = img
			}
		}
		a.update(func() {
			a.scanning = false
			a.books = books
			for id, img := range imgs {
				a.covers[id] = ui.NewBitmap(img)
			}
			set := map[string]bool{}
			for _, b := range books {
				set[b.Artist] = true
			}
			a.artists = []string{"全部绘师"}
			for k := range set {
				a.artists = append(a.artists, k)
			}
			sort.Strings(a.artists[1:])
			a.filter()
			if len(a.remote) > 0 {
				a.filterRemote()
			}
		})
	})
}

func (a *app) filter() {
	q := strings.ToLower(strings.TrimSpace(a.query))
	a.filtered = a.filtered[:0]
	for i, b := range a.books {
		if a.artist != "全部绘师" && b.Artist != a.artist {
			continue
		}
		if q != "" && !strings.Contains(strings.ToLower(b.Title+b.TitleKana+b.Artist), q) {
			continue
		}
		a.filtered = append(a.filtered, i)
	}
}

func (a *app) have(id int) bool {
	for _, b := range a.books {
		if b.ID == id {
			return b.Complete
		}
	}
	return false
}

// --- views ---

func (a *app) viewRoot(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Children(func() {
		switch a.view {
		case viewReader:
			a.readerView(c)
		case viewSettings:
			a.settingsView(c)
		case viewLogs:
			a.logsView(c)
		default:
			a.shelfView(c)
		}
	})
}

func (a *app) shelfView(c *ui.Context) {
	t := c.Theme()
	ui.Row(c).Gap(12).Padding(10, 16).AlignItems(ui.Center).Children(func() {
		ui.Text(c, "艳本资料书架").FontSize(20).Bold()
		if ui.SearchField(c, &a.query).Label("搜索书名 / 绘师").Placeholder("搜索书名 / 绘师").Width(260).Changed() {
			a.filter()
		}
		if len(a.artists) > 0 {
			if ui.Select(c, &a.artist, a.artists).Width(180).Label("绘师").Changed() {
				a.filter()
			}
		}
		ui.Textf(c, "共 %d 部", len(a.filtered)).TextColor(t.TextMuted)
		ui.Spacer(c)
		if ui.Button(c, "日志").Clicked() {
			a.view = viewLogs
		}
		if ui.Button(c, "设置").Clicked() {
			a.openSettings()
		}
	})
	ui.Divider(c)
	if len(a.books) == 0 {
		ui.Column(c).Grow(1).Center().Gap(12).Children(func() {
			if a.scanning {
				ui.Text(c, "正在读取书库…").TextColor(t.TextMuted)
				return
			}
			ui.Text(c, "书架还是空的").FontSize(18)
			ui.Text(c, "请在「设置」中下载示例数据，或从远程目录选择作品下载。").TextColor(t.TextMuted)
			if ui.PrimaryButton(c, "打开设置").Clicked() {
				a.openSettings()
			}
		})
		return
	}
	ui.GridView(c, &a.grid, len(a.filtered), 170, 290, func(i int) {
		b := &a.books[a.filtered[i]]
		card := ui.Column(c.Key(b.ID)).Grow(1).Gap(4).Padding(6).Radius(8).AlignItems(ui.Center)
		card.Children(func() {
			ui.Box(c).Height(210).FillWidth().Center().Children(func() {
				if bmp := a.covers[b.ID]; bmp != nil {
					img := ui.Image(c, bmp).Fit(ui.Contain).MaxWidth(160).MaxHeight(210).Radius(3).Shadow(2, 3, 8, 0, ui.RGBA(0, 0, 0, 120))
					if !b.Complete {
						img.Opacity(0.45)
					}
				} else {
					ui.Box(c).Size(140, 200).Background(t.Surface).Radius(3)
				}
			})
			ui.Text(c, b.Title).FontSize(13).MaxLines(2).TextAlign(ui.Center)
			info := b.Artist
			if b.Date != "" {
				info += " · " + b.Date
			}
			info += fmt.Sprintf(" · %d 图", b.PageCount)
			if !b.Complete {
				info += " · 未完成"
			}
			ui.Text(c, info).FontSize(11).TextColor(t.TextMuted).MaxLines(2).TextAlign(ui.Center)
		})
		if card.Hovered() {
			card.Background(t.SurfaceHover)
		}
		if card.Clicked() {
			a.openBook(b)
		}
	}).Grow(1).Gap(12).Padding(16)
}

func (a *app) openSettings() {
	a.view = viewSettings
	if len(a.remote) == 0 && !a.remoteLoading {
		a.loadRemote(false)
	}
}

// --- reader ---

func (a *app) openBook(b *Book) {
	a.book = b
	a.pos = a.conf.Positions[strconv.Itoa(b.ID)]
	a.zoom = 1
	a.pages = map[int]*ui.Bitmap{}
	a.loading = map[int]bool{}
	a.view = viewReader
	a.clampPos()
}

func (a *app) step() int {
	if a.spread {
		return 2
	}
	return 1
}

func (a *app) clampPos() {
	n := len(a.book.Pages)
	a.pos = max(0, min(a.pos, n-1))
	a.jump = float64(a.pos + 1)
	a.conf.Positions[strconv.Itoa(a.book.ID)] = a.pos
	_ = a.conf.Save()
}

func (a *app) forward() {
	if a.pos+a.step() < len(a.book.Pages) {
		a.pos += a.step()
		a.clampPos()
	}
}
func (a *app) back() { a.pos -= a.step(); a.clampPos() }

// left/right are visual: reading right to left, the left arrow goes forward.
func (a *app) left() {
	if a.rtl {
		a.forward()
	} else {
		a.back()
	}
}
func (a *app) right() {
	if a.rtl {
		a.back()
	} else {
		a.forward()
	}
}

func (a *app) visible() []int {
	idx := []int{a.pos}
	if a.spread && a.pos+1 < len(a.book.Pages) {
		idx = append(idx, a.pos+1)
	}
	if a.rtl && len(idx) == 2 {
		idx[0], idx[1] = idx[1], idx[0]
	}
	return idx
}

// ensurePages loads the pages in view and the next ones, and frees others.
func (a *app) ensurePages() {
	want := map[int]bool{}
	for i := a.pos - 2; i <= a.pos+3; i++ {
		if i >= 0 && i < len(a.book.Pages) {
			want[i] = true
		}
	}
	for i := range a.pages {
		if !want[i] {
			delete(a.pages, i)
		}
	}
	book := a.book
	for i := range want {
		if a.pages[i] != nil || a.loading[i] {
			continue
		}
		a.loading[i] = true
		path := filepath.Join(book.Dir, filepath.FromSlash(book.Pages[i].File))
		a.async(func() {
			img, err := loadScaled(path, 0, 3200)
			a.update(func() {
				if a.book != book {
					return
				}
				delete(a.loading, i)
				if err == nil {
					a.pages[i] = ui.NewBitmap(img)
				}
			})
		})
	}
}

func (a *app) readerView(c *ui.Context) {
	t := c.Theme()
	if c.Shortcut(0, ui.KeyLeft) {
		a.left()
	}
	if c.Shortcut(0, ui.KeyRight) {
		a.right()
	}
	if c.Shortcut(0, ui.KeyEqual) {
		a.zoom = min(a.zoom+0.5, 6)
	}
	if c.Shortcut(0, ui.KeyMinus) {
		a.zoom = max(1, a.zoom-0.5)
	}
	if c.Shortcut(0, ui.KeyEscape) {
		a.view = viewShelf
		return
	}
	a.ensurePages()
	n := len(a.book.Pages)
	ui.Row(c).Gap(8).Padding(8, 12).AlignItems(ui.Center).Children(func() {
		if ui.Button(c, "返回书架").Clicked() {
			a.view = viewShelf
		}
		ui.Text(c, a.book.Title+"　"+a.book.Artist).Bold().SingleLine().Grow(1).Shrink(1)
		dir := "阅读方向：左至右"
		if a.rtl {
			dir = "阅读方向：右至左"
		}
		if ui.Button(c, dir).Clicked() {
			a.rtl = !a.rtl
			a.conf.RTL = &a.rtl
			_ = a.conf.Save()
		}
		mode := "单页模式"
		if a.spread {
			mode = "双页模式"
		}
		if ui.Button(c, mode).Clicked() {
			a.spread = !a.spread
			a.conf.Spread = a.spread
			_ = a.conf.Save()
		}
		if ui.Button(c, "缩小").Clicked() {
			a.zoom = max(1, a.zoom-0.5)
		}
		if ui.Button(c, "放大").Clicked() {
			a.zoom = min(a.zoom+0.5, 6)
		}
		if ui.Button(c, "适应窗口").Clicked() {
			a.zoom = 1
		}
		label := fmt.Sprintf("第 %d / %d 图", a.pos+1, n)
		if v := a.visible(); len(v) == 2 {
			label = fmt.Sprintf("第 %d-%d / %d 图", a.pos+1, a.pos+2, n)
		}
		ui.Text(c, label)
		if ui.NumberInput(c, &a.jump, 1, float64(max(n, 1)), 1).Label("页码").Width(110).Submitted() {
			a.pos = int(a.jump) - 1
			a.clampPos()
		}
	})
	ui.Row(c).Grow(1).Basis(0).MinHeight(0).Clip().AlignItems(ui.Stretch).Background(ui.RGB(17, 17, 17)).Children(func() {
		if ui.Button(c, "‹").Width(40).Label("左翻页").Clicked() {
			a.left()
		}
		w, h := c.Size()
		if a.zoom <= 1 {
			ui.Row(c).Grow(1).MinWidth(0).AlignItems(ui.Stretch).Justify(ui.Center).Gap(2).Children(func() { a.pageImages(c, 0, 0) })
		} else {
			ph := (h - 60) * float32(a.zoom)
			ui.ScrollBoth(c).Grow(1).Children(func() {
				ui.Row(c).Gap(2).Children(func() { a.pageImages(c, ph, w) })
			})
		}
		if ui.Button(c, "›").Width(40).Label("右翻页").Clicked() {
			a.right()
		}
	})
	_ = t
}

func (a *app) pageImages(c *ui.Context, height, _ float32) {
	for _, i := range a.visible() {
		bmp := a.pages[i]
		if bmp == nil {
			ui.Text(c.Key(i), "加载中…").TextColor(ui.RGB(150, 150, 150)).AlignSelf(ui.Center)
			continue
		}
		img := ui.Image(c.Key(i), bmp)
		if height > 0 {
			bw, bh := bmp.Size()
			img.Size(height*float32(bw)/float32(bh), height)
		} else {
			img.Fit(ui.Contain).Grow(1).Basis(0).MinWidth(0).MinHeight(0)
		}
	}
}

// --- settings ---

func (a *app) loadRemote(refresh bool) {
	a.remoteLoading, a.remoteMsg = true, "正在获取目录…"
	cache := filepath.Join(a.conf.DataDir, "remote_catalog.json")
	cr := a.cr
	a.async(func() {
		var items []Item
		var err error
		if b, e := os.ReadFile(cache); e == nil && !refresh {
			err = jsonUnmarshal(b, &items)
		} else if !refresh {
			// No network needed: start from the catalog built into the app.
			err = jsonUnmarshal(builtinCatalog, &items)
		} else {
			if items, err = cr.ListItems(); err == nil {
				_ = writeJSON(cache, items)
			} else if e := jsonUnmarshal(builtinCatalog, &items); e == nil {
				err = nil
				defer func() { a.update(func() { a.remoteMsg = "联网刷新失败，显示内置目录" }) }()
			}
		}
		a.update(func() {
			a.remoteLoading = false
			if err != nil {
				a.remoteMsg = "获取失败：" + err.Error()
				return
			}
			a.remote = items
			a.filterRemote()
		})
	})
}

func (a *app) filterRemote() {
	q := strings.TrimSpace(a.remoteQuery)
	a.remoteFilter = a.remoteFilter[:0]
	got := 0
	for i, it := range a.remote {
		if a.have(it.ID) {
			got++
		}
		if q == "" || strings.Contains(it.Title+it.TitleKana+it.Artist+strconv.Itoa(it.ID), q) {
			a.remoteFilter = append(a.remoteFilter, i)
		}
	}
	a.remoteMsg = fmt.Sprintf("共 %d 部，已下载 %d 部，已选 %d 部", len(a.remote), got, len(a.selected))
}

var stateNames = map[string]string{"idle": "空闲", "running": "下载中", "pausing": "正在暂停…", "paused": "已暂停", "done": "已完成", "error": "出错"}

func (a *app) pickFolder() {
	a.async(func() {
		paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{Parent: a.win, Title: "选择数据文件夹", Directory: true, CreateDirectories: true, DefaultPath: a.conf.DataDir})
		if err != nil || len(paths) == 0 {
			return
		}
		a.update(func() { a.setDataDir(paths[0]) })
	})
}

func (a *app) setDataDir(dir string) {
	if a.cr.Running() {
		a.confMsg = "下载进行中，请先暂停"
		return
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		a.confMsg = "无法使用该文件夹：" + err.Error()
		return
	}
	a.conf.DataDir = dir
	_ = a.conf.Save()
	a.newCrawler()
	a.books, a.covers, a.remote = nil, map[int]*ui.Bitmap{}, nil
	a.confMsg = "已切换数据文件夹"
	a.rescan()
	a.loadRemote(false)
}

func (a *app) logsView(c *ui.Context) {
	t := c.Theme()
	muted := t.TextMuted
	entries, _ := a.logs.Snapshot()

	ui.Row(c).Gap(12).Padding(10, 16).AlignItems(ui.Center).Children(func() {
		if ui.Button(c, "返回书架").Clicked() {
			a.view = viewShelf
		}
		ui.Text(c, "运行日志").FontSize(20).Bold()
		ui.Textf(c, "%d 条", len(entries)).TextColor(muted)
		ui.Spacer(c)
		if ui.Button(c, "清空").Clicked() {
			a.logs.Clear()
		}
		if ui.Button(c, "复制全部").Clicked() {
			a.copyLogs(c, entries)
		}
	})
	ui.Divider(c)

	ui.Column(c).Grow(1).Padding(4, 16, 12, 16).Children(func() {
		if len(entries) == 0 {
			ui.Text(c, "还没有日志。下载时会在这里显示进度与 HTTP 状态。").TextColor(muted)
			return
		}
		ui.ScrollBoth(c).Grow(1).Children(func() {
			ui.Column(c).Gap(2).Children(func() {
				for _, e := range entries {
					ui.Textf(c, "%s  %s", e.When.Format("15:04:05"), e.Line).
						FontSize(12).
						TextColor(logColor(t, e.Line)).
						SingleLine()
				}
			})
		})
	})
}

// logColor highlights the lines that matter when diagnosing a blocked download.
func logColor(t *ui.Theme, line string) ui.Color {
	switch {
	case strings.Contains(line, "403"), strings.Contains(line, "429"),
		strings.Contains(line, "error"), strings.Contains(line, "放弃"),
		strings.Contains(line, "失败"):
		return ui.RGBA(220, 80, 80, 255)
	case strings.Contains(line, "200"), strings.Contains(line, "OK"),
		strings.Contains(line, "完成"), strings.Contains(line, "成功"):
		return ui.RGBA(70, 170, 90, 255)
	default:
		return t.TextMuted
	}
}

// copyLogs puts the whole buffer on the clipboard as plain text.
func (a *app) copyLogs(c *ui.Context, entries []logEntry) {
	var b strings.Builder
	for _, e := range entries {
		b.WriteString(e.When.Format("15:04:05"))
		b.WriteString("  ")
		b.WriteString(e.Line)
		b.WriteString("\n")
	}
	c.WriteClipboard(b.String())
}

func (a *app) settingsView(c *ui.Context) {
	t := c.Theme()
	muted := t.TextMuted
	section := func(s string) { ui.Text(c, s).FontSize(16).Bold().Margin(10, 0, 2, 0) }
	ui.Row(c).Gap(12).Padding(10, 16).AlignItems(ui.Center).Children(func() {
		if ui.Button(c, "返回书架").Clicked() {
			a.view = viewShelf
		}
		ui.Text(c, "设置").FontSize(20).Bold()
	})
	ui.Divider(c)
	ui.Column(c).Grow(1).Padding(4, 20, 16, 20).Gap(6).Children(func() {
		section("基本设置")
		ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "数据文件夹")
			ui.Text(c, a.conf.DataDir).TextColor(muted).SingleLine().Shrink(1)
			if ui.Button(c, "选择文件夹…").Clicked() {
				a.pickFolder()
			}
		})
		// The site geoblocks whole regions: from a blocked IP every request
		// returns 403 no matter how often it is retried, so the proxy is the
		// one setting a user cannot do without. It goes first, with the same
		// explicit text label the other rows use — ui.SearchField's own Label
		// is not drawn, which made the proxy box look like a stray search box
		// and left people unable to find it.
		ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
			ui.Text(c, proxyLabel)
			ui.SearchField(c, &a.proxy).Placeholder("http://127.0.0.1:7897").Width(260)
			ui.Text(c, "填好后点下面的「保存」").TextColor(muted)
		})
		ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "请求间隔（秒，最小 1）")
			ui.NumberInput(c, &a.delay, MinDelay, 30, 0.1).Label("请求间隔").Width(130)
			if ui.Button(c, "保存").Clicked() {
				// Proxy and delay are both written on save; stopping a running
				// download to change the proxy would lose the queue.
				a.conf.Delay = max(MinDelay, a.delay)
				a.conf.Proxy = strings.TrimSpace(a.proxy)
				a.delay = a.conf.Delay
				HttpProxy = a.conf.Proxy
				_ = a.conf.Save()
				if !a.cr.Running() {
					a.newCrawler()
				}
				a.confMsg = "已保存"
			}
			ui.Text(c, a.confMsg).TextColor(muted)
		})
		section("示例数据")
		ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
			if ui.Button(c, "下载示例数据").Disabled(a.sampleBusy).Clicked() {
				a.downloadSample()
			}
			msg := a.sampleMsg
			if msg == "" {
				msg = "从 GitHub 最新发布版下载 3 部示例作品（约 190 MB）"
			}
			ui.Text(c, msg).TextColor(muted)
		})
		ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
			if ui.Button(c, "导入本地数据包…").Disabled(a.sampleBusy).Clicked() {
				a.importLocal()
			}
			ui.Text(c, "选择用户脚本导出的 ehonshelf-*.zip（可多选），已有文件不会被覆盖").TextColor(muted)
		})
		section("下载进度")
		s := a.status
		line := stateNames[s.State]
		if s.Current != nil {
			line += fmt.Sprintf("：%d %s", s.Current.ID, s.Current.Title)
		}
		ui.Text(c, line+"　"+s.Message)
		prog := func(label string, done, total int, unit string) {
			ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
				ui.Text(c, label).Width(70)
				v := 0.0
				if total > 0 {
					v = float64(done) / float64(total)
				}
				ui.Progress(c, v).Label(label).Width(320)
				ui.Textf(c, "%d / %d %s", done, total, unit).TextColor(muted)
			})
		}
		prog("当前作品", s.ItemDone, s.ItemTotal, "页")
		prog("总体进度", s.DoneItems, s.TotalItems, fmt.Sprintf("部，剩余 %d 部", s.QueueLen))
		ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
			if ui.Button(c, "暂停").Disabled(s.State != "running").Clicked() {
				a.cr.Pause()
			}
			if ui.Button(c, "继续").Disabled(s.State == "running" || s.State == "pausing" || s.QueueLen == 0).Clicked() {
				a.cr.Resume()
			}
			if len(s.Errors) > 0 {
				last := s.Errors[max(0, len(s.Errors)-2):]
				ui.Text(c, fmt.Sprintf("失败 %d 项：%s", len(s.Errors), strings.Join(last, "；"))).TextColor(t.Danger).SingleLine().Shrink(1)
			}
		})
		section("远程目录")
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			if ui.Button(c, "重新获取").Disabled(a.remoteLoading).Clicked() {
				a.loadRemote(true)
			}
			if ui.SearchField(c, &a.remoteQuery).Label("筛选").Placeholder("筛选").Width(200).Changed() {
				a.filterRemote()
			}
			if ui.Button(c, "全选").Clicked() {
				for _, i := range a.remoteFilter {
					a.selected[a.remote[i].ID] = true
				}
				a.filterRemote()
			}
			if ui.Button(c, "全不选").Clicked() {
				a.selected = map[int]bool{}
				a.filterRemote()
			}
			if ui.PrimaryButton(c, "下载所选").Disabled(len(a.selected) == 0).Clicked() {
				var items []Item
				for _, it := range a.remote {
					if a.selected[it.ID] {
						items = append(items, it)
					}
				}
				a.cr.Start(items)
			}
			if ui.Button(c, "下载全部").Disabled(len(a.remote) == 0).Clicked() {
				a.cr.Start(a.remote)
			}
			ui.Text(c, a.remoteMsg).TextColor(muted)
		})
		ui.List(c, &a.list, len(a.remoteFilter), func(i int) {
			it := a.remote[a.remoteFilter[i]]
			ui.Row(c.Key(it.ID)).Gap(10).Padding(3, 8).AlignItems(ui.Center).Children(func() {
				sel := a.selected[it.ID]
				if ui.Checkbox(c, &sel, "").Label(fmt.Sprintf("选择 %d", it.ID)).Changed() {
					if sel {
						a.selected[it.ID] = true
					} else {
						delete(a.selected, it.ID)
					}
					a.filterRemote()
				}
				col := t.Text
				if a.have(it.ID) {
					col = t.Success
				}
				ui.Textf(c, "%d", it.ID).Width(44).TextColor(col)
				ui.Text(c, it.Title).Grow(1).Shrink(1).SingleLine().TextColor(col)
				ui.Text(c, it.Artist).Width(140).SingleLine().TextColor(col)
				st := ""
				if a.have(it.ID) {
					st = "已下载"
				}
				ui.Text(c, st).Width(56).TextColor(col)
			})
		}).Grow(1).MinHeight(200).Border(1, t.Border).Radius(6)
		ui.Text(c, "数据来源：国際日本文化研究センター「艶本資料データベース」。下载按设定间隔逐个请求，遵守 robots.txt，可随时暂停；重启后可继续未完成的队列，已下载的页面会自动跳过。").FontSize(12).TextColor(muted)
	})
}

func (a *app) downloadSample() {
	a.sampleBusy, a.sampleMsg = true, "正在下载示例数据…"
	dir := a.conf.DataDir
	a.async(func() {
		err := DownloadSample(dir, func(m string) { a.update(func() { a.sampleMsg = m }) })
		a.update(func() {
			a.sampleBusy = false
			if err != nil {
				a.sampleMsg = "下载失败：" + err.Error()
				return
			}
			a.sampleMsg = "示例数据已导入"
			a.rescan()
		})
	})
}

// importLocal imports zip files chosen by the user (e.g. from the userscript).
func (a *app) importLocal() {
	dir := a.conf.DataDir
	a.async(func() {
		paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{Parent: a.win, Title: "选择数据包", Multiple: true,
			DefaultPath: DownloadsDir(), Filters: []mygo.FileFilter{{Name: "ZIP 数据包", Extensions: []string{"zip"}}}})
		if err != nil || len(paths) == 0 {
			return
		}
		a.update(func() { a.sampleBusy, a.sampleMsg = true, "正在导入…" })
		n, err := ImportZips(paths, dir)
		a.update(func() {
			a.sampleBusy = false
			if err != nil {
				a.sampleMsg = "导入失败：" + err.Error()
			} else {
				a.sampleMsg = fmt.Sprintf("已导入 %d 个数据包", n)
			}
			a.rescan()
		})
	})
}

// firstRun seeds an empty library from a sample-data.zip next to the executable.
func (a *app) firstRun() {
	ents, _ := os.ReadDir(a.conf.DataDir)
	for _, e := range ents {
		if _, err := strconv.Atoi(e.Name()); err == nil {
			return
		}
	}
	exe, err := os.Executable()
	if err != nil {
		return
	}
	for _, p := range []string{filepath.Join(filepath.Dir(exe), "sample-data.zip"), filepath.Join(filepath.Dir(exe), "..", "Resources", "sample-data.zip")} {
		if _, err := os.Stat(p); err == nil {
			if err := ImportSample(p, a.conf.DataDir); err != nil {
				log.Print(err)
			}
			return
		}
	}
}

func (a *app) menu() *mygo.Menu {
	do := func(fn func()) func(*mygo.MenuItem, *mygo.Window) {
		return func(*mygo.MenuItem, *mygo.Window) { a.update(fn) }
	}
	return mygo.NewMenu([]*mygo.MenuItem{
		{Role: mygo.RoleAppMenu},
		{Label: "文件", Submenu: []*mygo.MenuItem{
			{Label: "书架", Accelerator: "CmdOrCtrl+1", Click: do(func() { a.view = viewShelf })},
			{Label: "设置", Accelerator: "CmdOrCtrl+,", Click: do(a.openSettings)},
			{Label: "运行日志", Accelerator: "CmdOrCtrl+L", Click: do(func() { a.view = viewLogs })},
			{Label: "重新读取书库", Accelerator: "CmdOrCtrl+R", Click: do(a.rescan)},
			mygo.Separator(),
			{Role: mygo.RoleQuit, Label: "退出"},
		}},
		{Role: mygo.RoleEditMenu, Label: "编辑"},
		{Label: "阅读", Submenu: []*mygo.MenuItem{
			{Label: "切换阅读方向", Click: do(func() { a.rtl = !a.rtl; a.conf.RTL = &a.rtl; _ = a.conf.Save() })},
			{Label: "切换单页 / 双页", Click: do(func() { a.spread = !a.spread; a.conf.Spread = a.spread; _ = a.conf.Save() })},
			{Label: "适应窗口", Accelerator: "CmdOrCtrl+0", Click: do(func() { a.zoom = 1 })},
			mygo.Separator(),
			{Role: mygo.RoleToggleFullScreen},
		}},
		{Role: mygo.RoleWindowMenu, Label: "窗口"},
	})
}

func main() {
	// The app is written in Simplified Chinese; MyGo's own menus and
	// controls follow it.
	mygo.App.SetLanguages("zh-Hans")
	a := newApp()
	// Prefer the proxy saved in settings; EHONSHELF_PROXY is the fallback
	// already picked up by crawler.go.
	if p := strings.TrimSpace(a.conf.Proxy); p != "" {
		HttpProxy = p
	}
	// Route every log.Printf (crawler progress, HTTP status, retries) into
	// the in-app log panel. A windowed launch has no visible stderr, so
	// without this the download diagnostics are invisible to the user.
	restoreLog := installLogCapture(a.logs)
	defer restoreLog()
	a.firstRun()
	mygo.App.WhenReady(func() {
		mygo.App.SetMenu(a.menu())
		a.win = mygo.NewWindow(mygo.WindowOptions{
			Title:     "艳本资料书架",
			Width:     1200,
			Height:    820,
			MinWidth:  720,
			MinHeight: 480,
			StateKey:  "main",
			Content:   ui.View(a.viewRoot),
		})
		a.update = a.win.Update
		a.rescan()
		// EHONSHELF_VIEW=settings opens the settings first (for screenshots and support).
		switch os.Getenv("EHONSHELF_VIEW") {
		case "settings":
			a.openSettings()
		case "logs":
			a.view = viewLogs
		}
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
