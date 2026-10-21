// Controls adapted from github.com/egoist/kero (MIT).
package ehonui

import (
	"slices"
	"time"

	"github.com/egoist/mygo/ui"
)

const (
	controlH  = 24
	popupH    = 28
	menuPad   = 5
	menuItemH = 24
	menuTextX = 22
)

var menuWidths = map[string]float32{}

func popupFace(c *ui.Context, trig ui.Element, text string, open bool, pal *palette) {
	trig.Height(popupH).Padding(0, 3, 0, 10).Gap(6).Radius(popupH / 2).AlignItems(ui.Center).Cursor(ui.CursorDefault)
	tint := pal.text.Alpha(0.065)
	if pal.dark {
		tint = ui.RGBA(255, 255, 255, 0.09)
	}
	hover := trig.Hovered() || open
	bg, disc := ui.Transparent, tint
	switch {
	case open || trig.Pressed():
		bg, disc = tint.Mix(pal.text, 0.06), ui.Transparent
	case hover:
		bg, disc = tint, ui.Transparent
	}
	anim := ui.Animation{Duration: 120 * time.Millisecond}
	trig.Background(trig.AnimateColor("bg", bg, anim))
	trig.Children(func() {
		ui.Text(c, text).FontSize(13).TextColor(pal.text).SingleLine().Ellipsis("…").MinWidth(0).Shrink(1)
		d := ui.Box(c.Key("disc")).Size(22, 22).Radius(11).Center().Shrink(0)
		d.Background(d.AnimateColor("bg", disc, anim)).Children(func() {
			ui.Icon(c, updownIcon()).Size(16, 16).TextColor(pal.text.Alpha(0.85))
		})
	})
}

func popup[T comparable](c *ui.Context, key string, value *T, options []T, label func(T) string, width float32, pal *palette) ui.Element {
	sel := ui.SelectBase(c.Key(key), value)
	trig := sel.Trigger.MaxWidth(width)
	open := sel.Open()
	popupFace(c, trig, label(*value), open, pal)
	var spot ui.Element
	maxH := float32(0)
	if open {
		spot, maxH = menuSpot(c, key, trig.Bounds(), max(slices.Index(options, *value), 0), len(options))
	}
	sel.Popup(func(panel ui.Element) {
		menuPanel(panel, pal).AttachTo(spot, ui.AnchorTopLeft, ui.AnchorTopLeft)
		menuWidths[key] = panel.Bounds().W
		over := panel.Hovered()
		ui.Scroll(c.Key("items")).MaxHeight(maxH - 2*menuPad).Children(func() {
			for _, o := range options {
				item := sel.Item(o)
				hi := item.Highlighted() && (item.Hovered() || !over && o != *value)
				menuRow(c, item, label(o), o == *value, hi, pal)
			}
		})
	})
	return trig
}

func menuSpot(c *ui.Context, key string, b ui.Rect, at, n int) (ui.Element, float32) {
	const edge = 8
	w, h := c.Size()
	maxH := h - 2*edge
	menuH := min(2*menuPad+float32(n)*menuItemH, maxH)
	y := b.Y - (menuPad + float32(at)*menuItemH + (menuItemH-popupH)/2)
	y = max(edge, min(y, h-edge-menuH))
	x := b.X - (menuPad + menuTextX - 10)
	if mw := max(menuWidths[key], b.W); x+mw > w-edge {
		x = w - edge - mw
	}
	x = max(edge, x)
	var spot ui.Element
	ui.Overlay(c, func() {
		spot = ui.Box(c.Key(key+"-spot")).Absolute().Left(x).Top(y).Size(0, 0).PassThrough()
	})
	return spot, maxH
}

func menuPanel(panel ui.Element, pal *palette) ui.Element {
	if pal.dark {
		return panel.Padding(menuPad).Radius(13).Background(pal.popover).
			InsetShadow(0, 0, 0, 1, ui.RGBA(255, 255, 255, 0.12)).
			Shadow(0, 4, 16, 0, ui.RGBA(0, 0, 0, 0.4))
	}
	return panel.Padding(menuPad).Radius(13).
		LinearGradient(ui.LinearGradient{Angle: 180, From: ui.Hex("#fbfbfb"), To: ui.Hex("#f4f4f5")}).
		InsetShadow(0, 0, 0, 0.5, ui.RGBA(0, 0, 0, 0.3)).
		Shadow(0, 5, 10, 0, ui.RGBA(0, 0, 0, 0.22))
}

func menuRow(c *ui.Context, item ui.Element, label string, current, highlighted bool, pal *palette) {
	item.Height(menuItemH).Padding(0, 12, 0, 6).Gap(5).Radius(7).AlignItems(ui.Center)
	fg := pal.text
	if highlighted {
		item.Background(pal.accent)
		fg = ui.Hex("#ffffff")
	}
	item.TextColor(fg).Children(func() {
		mark := ui.Icon(c, checkIcon()).Size(12, 12).Shrink(0)
		if !current {
			mark.Opacity(0)
		}
		ui.Text(c, label).FontSize(13).SingleLine().Ellipsis("…").MinWidth(0)
	})
}

func field(frame ui.Element, focused bool, pal *palette) ui.Element {
	frame.Height(controlH+2).Padding(0, 8).Gap(6).Radius(7).AlignItems(ui.Center)
	stroke := ui.RGBA(0, 0, 0, 0.13)
	bg := ui.Hex("#ffffff")
	if pal.dark {
		stroke, bg = ui.RGBA(255, 255, 255, 0.13), ui.RGBA(255, 255, 255, 0.06)
	}
	frame.Background(bg).InsetShadow(0, 0, 0, 1, stroke)
	glow := frame.Animate("glow", b2f(focused), 120*time.Millisecond)
	if glow > 0 {
		frame.Shadow(0, 0, 0, 3.5*glow, pal.accent.Alpha(0.32*glow)).InsetShadow(0, 0, 0, 1, pal.accent.Alpha(0.75*glow))
	}
	return frame
}

func textField(c *ui.Context, key string, value *string, placeholder string, width float32, pal *palette) ui.Element {
	var in ui.Element
	frame := ui.Row(c.Key(key)).Cursor(ui.CursorText)
	if width > 0 {
		frame.Width(width)
	} else {
		frame.Grow(1)
	}
	frame.Children(func() {
		in = ui.TextInputBase(c.Key("in"), value).Grow(1).MinWidth(0).FontSize(13).Placeholder(placeholder).Label(placeholder)
	})
	field(frame, in.Focused(), pal)
	return in
}

func segmented(c *ui.Context, key string, value *string, ids, labels []string) {
	pal := paletteFor(c)
	i := max(slices.Index(ids, *value), 0)
	track, thumb := ui.RGBA(0, 0, 0, 0.06), ui.Hex("#ffffff")
	if pal.dark {
		track, thumb = ui.RGBA(255, 255, 255, 0.07), ui.RGBA(255, 255, 255, 0.2)
	}
	seg := ui.SegmentedBase(c.Key(key), &i, len(labels))
	seg.Track.Height(24).Padding(2).Radius(7).Background(track).AlignItems(ui.Center).Children(func() {
		for n, label := range labels {
			if n > 0 {
				sep := ui.Box(c).Width(1).Height(12).Shrink(0)
				if n != i && n-1 != i {
					sep.Background(pal.text.Alpha(0.14))
				}
			}
			s := seg.Segment(n).FillHeight().Padding(0, 12).Radius(5).Center()
			if n == i {
				s.Background(thumb)
				if pal.dark {
					s.Shadow(0, 1, 1, 0, ui.RGBA(0, 0, 0, 0.25))
				} else {
					s.Shadow(0, 0.5, 1.5, 0, ui.RGBA(0, 0, 0, 0.16))
				}
			}
			s.Children(func() {
				ui.Text(c, label).FontSize(12.5).SingleLine().TextColor(pal.text)
			})
		}
	})
	seg.Track.OnChange(func() {
		if i >= 0 && i < len(ids) {
			*value = ids[i]
		}
	})
}
