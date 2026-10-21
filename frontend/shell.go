// Sidebar layout adapted from github.com/egoist/kero (MIT) via GodGo.
package ehonui

import (
	"github.com/egoist/mygo/ui"
)

// Section is one row of the sidebar.
type Section struct {
	ID    string
	Title string
	Icon  string
}

// SetSections replaces the sidebar rows.
func (a *App) SetSections(s []Section) { a.sections = s }

// SetSection selects a row.
func (a *App) SetSection(id string) { a.current = id }

// Current returns the selected row's id.
func (a *App) Current() string { return a.current }

// OnSelect is called when the user clicks a sidebar row.
func (a *App) OnSelect(fn func(id string)) { a.onSelect = fn }

// View draws the shell: the sidebar on the left and a card on the right, whose
// body is filled by body.
func (a *App) View(c *ui.Context, body func(*ui.Context)) {
	pal := paletteFor(c)
	c.SetTheme(a.theme(c, pal))

	row := ui.Row(c).Fill().AlignItems(ui.Stretch)
	backdrop(c, pal, row, a.set)
	row.Children(func() {
		ui.Column(c).Width(a.set.SidebarWidth).Shrink(0).Children(func() {
			a.sidebar(c, pal)
		})
		main := ui.Column(c).Grow(1).MinWidth(0).Padding(cardGap, cardGap, cardGap, cardGap).DragWindow()
		main.Children(func() {
			a.card(c, pal, body)
		})
	})
}

func (a *App) sidebar(c *ui.Context, pal *palette) {
	ui.Row(c).Height(topBar).Shrink(0).AlignItems(ui.Center).Padding(0, 12, 0, 12).DragWindow().Children(func() {
		ui.Text(c, "艳本资料书架").FontSize(13).FontWeight(600).TextColor(pal.text)
	})
	ui.Scroll(c.Key("sections")).Grow(1).Padding(0, 10, 16, 10).Children(func() {
		ui.Column(c).Gap(2).Children(func() {
			for _, s := range a.sections {
				a.sidebarRow(c, pal, s)
			}
		})
	})
}

func (a *App) sidebarRow(c *ui.Context, pal *palette, s Section) {
	sel := s.ID == a.current
	bg := ui.Transparent
	if sel {
		bg = pal.rowSelected
	}
	row := ui.Row(c.Key(s.ID)).Height(rowH).Shrink(0).Radius(7).Padding(0, 9, 0, 9).
		AlignItems(ui.Center).Gap(9).Background(bg).Cursor(ui.CursorPointer)
	if !sel && row.Hovered() {
		row.Background(pal.rowHover)
	}
	row.Children(func() {
		if g := glyph(s.Icon); g != nil {
			ui.Icon(c, g).Size(15, 15).TextColor(pal.textMuted)
		}
		ui.Text(c, s.Title).FontSize(13).TextColor(pal.text).SingleLine().Shrink(1)
	})
	if row.Clicked() {
		a.current = s.ID
		if a.onSelect != nil {
			a.onSelect(s.ID)
		}
	}
}

func (a *App) card(c *ui.Context, pal *palette, body func(*ui.Context)) {
	card := raised(ui.Column(c).Grow(1).MinHeight(0).Radius(14).Background(pal.card), pal).Clip()
	card.Children(func() {
		title := ""
		for _, s := range a.sections {
			if s.ID == a.current {
				title = s.Title
			}
		}
		ui.Row(c).Height(headerH).Shrink(0).AlignItems(ui.Center).Padding(0, 16, 0, 16).DragWindow().Children(func() {
			ui.Text(c, title).FontSize(14).FontWeight(600).TextColor(pal.text)
		})
		ui.Scroll(c.Key("body")).Grow(1).MinHeight(0).Padding(6, 18, 20, 18).Children(func() {
			body(c)
		})
	})
}
