// Window chrome adapted from github.com/egoist/kero (MIT).
package ehonui

import (
	"math"
	"runtime"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

const (
	barMid         = 30
	topBar         = 2 * barMid
	headerH        = 2 * (barMid - cardGap)
	cardGap        = 8
	rowH           = 34
	defaultSidebar = 248
	minSidebar     = 200
	maxSidebar     = 420
)

var ownControls = runtime.GOOS != "darwin"

var sidebarMotion = ui.Animation{Duration: 260 * time.Millisecond, Ease: ui.CubicBezier(0.25, 1, 0.5, 1)}

type chrome struct {
	shown     float32
	sideW     float32
	mainPad   float32
	headerPad float32
}

func (ch chrome) buttonsLeft(bar ui.TitleBar) float32 {
	return lead(bar) - (ch.sideW + ch.mainPad)
}

func lead(bar ui.TitleBar) float32 {
	if ownControls {
		return bar.Left + 2*cardGap
	}
	return bar.Left + cardGap
}

func chromeFor(progress, width float32, bar ui.TitleBar) chrome {
	round := func(v float32) float32 { return float32(math.Round(float64(v))) }
	w := round(width * min(max(progress, 0), 1))
	s := float32(0)
	if width > 0 {
		s = w / width
	}
	pad := round(cardGap * (1 - s))
	hidden, shown := lead(bar)+16+60, width+16
	x := round(hidden + (shown-hidden)*s)
	return chrome{shown: s, sideW: w, mainPad: pad, headerPad: x - w - pad}
}

func b2f(b bool) float32 {
	if b {
		return 1
	}
	return 0
}

func iconButton(c *ui.Context, key string, svg *ui.SVG, label string, pal *palette, fn func()) {
	b := ui.ButtonBase(c.Key(key)).Size(28, 26).Radius(6).Center().TextColor(pal.textMuted).Label(label).Tooltip(label)
	if b.Pressed() {
		b.Background(pal.rowHover.Alpha(0.12))
	} else if b.Hovered() {
		b.Background(pal.rowHover)
		b.TextColor(pal.text)
	}
	b.Children(func() { ui.Icon(c, svg).Size(16, 16) })
	if b.Clicked() {
		fn()
	}
}

// Frame hides the title bar and keeps the system's resize edges, shadow and
// corners. Frameless would drop them; the view does not draw a window frame.
func Frame(o mygo.WindowOptions) mygo.WindowOptions { return framed(o) }

// ContentLead is the left inset that clears the traffic lights.
func ContentLead(bar ui.TitleBar) float32 { return lead(bar) }

// BarHeight is the top bar, centered on the traffic lights.
func BarHeight() float32 { return topBar }

// WindowButtons draws minimize, maximize and close where the system buttons
// are hidden. On macOS it draws nothing.
func WindowButtons(c *ui.Context, w *mygo.Window) {
	windowControls(c, paletteFor(c), w, func(k string) string {
		switch k {
		case "window.minimize":
			return "最小化"
		case "window.maximize":
			return "最大化"
		case "window.restore":
			return "还原"
		case "window.close":
			return "关闭"
		default:
			return k
		}
	})
}

func framed(o mygo.WindowOptions) mygo.WindowOptions {
	if ownControls {
		o.TitleBarStyle = mygo.TitleBarHidden
		o.HideWindowButtons = true
		o.AutoHideMenuBar = true
		return o
	}
	o.TitleBarStyle = mygo.TitleBarHiddenInset
	o.TitleBarHeight = topBar
	o.TrafficLightPosition = &mygo.Point{X: 18, Y: barMid - 7}
	return o
}

func windowControls(c *ui.Context, pal *palette, w *mygo.Window, label func(string) string) {
	if !ownControls || w == nil {
		return
	}
	ui.Row(c.Key("window-controls")).Gap(2).AlignItems(ui.Center).Shrink(0).Margin(0, 0, 0, 6).Children(func() {
		iconButton(c, "minimize", minimizeIcon(), label("window.minimize"), pal, w.Minimize)
		maximized := w.IsMaximized()
		text, svg := label("window.maximize"), maximizeIcon()
		if maximized {
			text, svg = label("window.restore"), restoreIcon()
		}
		m := ui.ButtonBase(c.Key("maximize")).Size(28, 26).Radius(6).Center().TextColor(pal.textMuted).Label(text).MaximizeButton()
		if m.Pressed() {
			m.Background(pal.rowHover.Alpha(0.12))
		} else if m.Hovered() {
			m.Background(pal.rowHover).TextColor(pal.text)
		}
		m.Children(func() { ui.Icon(c, svg).Size(16, 16) })
		if m.Clicked() {
			if maximized {
				w.Unmaximize()
			} else {
				w.Maximize()
			}
		}
		b := ui.ButtonBase(c.Key("close-window")).Size(28, 26).Radius(6).Center().TextColor(pal.textMuted).Label(label("window.close")).Tooltip(label("window.close"))
		if b.Hovered() || b.Pressed() {
			b.Background(pal.danger).TextColor(ui.RGB(255, 255, 255))
		}
		b.Children(func() { ui.Icon(c, closeIcon()).Size(14, 14) })
		if b.Clicked() {
			w.Close()
		}
	})
}

func icon(shapes string) *ui.SVG {
	return ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.35" stroke-linecap="round" stroke-linejoin="round">` + shapes + `</svg>`))
}

func minimizeIcon() *ui.SVG { return icon(`<path d="M3.5 8h9"/>`) }
func maximizeIcon() *ui.SVG {
	return icon(`<rect x="3.5" y="3.5" width="9" height="9" rx="1.5"/>`)
}
func restoreIcon() *ui.SVG {
	return icon(`<rect x="3.5" y="5.5" width="7" height="7" rx="1.2"/><path d="M5.5 3.5h5.5a1.5 1.5 0 0 1 1.5 1.5v5.5"/>`)
}
