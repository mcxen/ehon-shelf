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
	bar := c.TitleBar()

	row := ui.Row(c).Fill().AlignItems(ui.Stretch)
	backdrop(c, pal, row, a.set)
	row.Children(func() {
		ui.Column(c).Width(a.set.SidebarWidth).Shrink(0).Children(func() {
			a.sidebar(c, pal, bar)
		})
		// The card meets the sidebar. The 8 DIP gap is only the top, right and bottom.
		main := ui.Column(c).Grow(1).MinWidth(0).Padding(cardGap, cardGap, cardGap, 0).DragWindow()
		main.Children(func() {
			a.card(c, pal, body)
		})
	})
}

func (a *App) sidebar(c *ui.Context, pal *palette, bar ui.TitleBar) {
	ui.Row(c).Height(topBar).Shrink(0).AlignItems(ui.Center).Padding(0, 12, 0, lead(bar)).DragWindow().Children(func() {
		ui.Text(c, "艳本资料书架").FontSize(13).FontWeight(600).TextColor(pal.text).SingleLine()
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
	row := ui.Row(c.Key(s.ID)).Height(rowH).Shrink(0).Radius(8).Padding(0, 10).
		AlignItems(ui.Center).Gap(10).Background(bg).Cursor(ui.CursorPointer)
	if sel {
		chosen(row, pal)
	}
	if !sel && row.Hovered() {
		row.Background(pal.rowHover)
	}
	row.Children(func() {
		if g := glyph(s.Icon); g != nil {
			ic := ui.Icon(c, g).Size(15, 15).TextColor(pal.textMuted)
			if sel {
				ic.TextColor(pal.accent)
			}
		}
		ui.Text(c, s.Title).FontSize(13).TextColor(pal.text).SingleLine().Shrink(1)
	})
	if (row.Pressed() || row.Clicked()) && !sel {
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
		ui.Row(c).Height(headerH).Shrink(0).AlignItems(ui.Center).Padding(0, 12, 0, 20).DragWindow().Children(func() {
			ui.Text(c, title).FontSize(15).FontWeight(650).TextColor(pal.text)
			ui.Spacer(c)
			WindowButtons(c, a.win)
		})
		// The body scrolls itself. A grid or a list inside another scroll
		// never gets a height, so the shelf could not fill the card.
		ui.Column(c).Grow(1).MinHeight(0).Padding(4, 16, 16, 16).Children(func() {
			body(c)
		})
	})
}
