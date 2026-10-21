package ehonui

import (
	"log"

	"github.com/egoist/mygo/ui"
)

// Icons are Framework7's line icons and two Lucide glyphs, the same sets
// Kero's sidebar and settings use. They ship in resources/icons.

var iconCache = map[string]*ui.SVG{}

// glyph loads a shipped icon. A missing file must not take the window down:
// icons are decoration, and a test or an installed build without resources
// should still draw the sidebar. The absence is logged once and the row
// simply shows no glyph.
func glyph(name string) *ui.SVG {
	if s, ok := iconCache[name]; ok {
		return s
	}
	b, err := readResource("icons/" + name + ".svg")
	if err != nil {
		log.Printf("icon %s: %v", name, err)
		iconCache[name] = nil
		return nil
	}
	s := ui.MustParseSVG(b)
	iconCache[name] = s
	return s
}

func sidebarIcon() *ui.SVG { return glyph("f7-sidebar-left") }
func plusIcon() *ui.SVG    { return glyph("f7-plus") }
func chevronIcon() *ui.SVG { return glyph("f7-chevron-right") }
func folderIcon() *ui.SVG  { return glyph("f7-folder") }
func searchIcon() *ui.SVG  { return glyph("f7-search") }
func gearIcon() *ui.SVG    { return glyph("f7-gear-alt") }
func closeIcon() *ui.SVG   { return glyph("f7-xmark") }
func updownIcon() *ui.SVG  { return glyph("lucide-chevrons-up-down") }
func checkIcon() *ui.SVG   { return glyph("lucide-check") }

func pageIcon(id string) *ui.SVG {
	switch id {
	case "general":
		return glyph("f7-gear-alt")
	case "appearance":
		return glyph("f7-paintbrush")
	case "updates":
		return glyph("f7-sparkles")
	case "workflow":
		return glyph("f7-chevron-left-slash-chevron-right")
	case "notifications":
		return glyph("f7-bell")
	case "keyboard":
		return glyph("f7-keyboard")
	default:
		return glyph("f7-slider-horizontal-3")
	}
}
