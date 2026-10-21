// Colors adapted from github.com/egoist/kero (MIT).
package ehonui

import (
	"github.com/egoist/mygo/ui"

	"github.com/mcxen/ehon-shelf/backend/settings"
)

// palette is the shell's colors, for the light and the dark appearance.
// The values follow Kero's, so the sidebar and the settings window sit
// on the same washes and the same row treatment.
type palette struct {
	dark bool

	sidebar      ui.Color
	text         ui.Color
	textMuted    ui.Color
	textFaint    ui.Color
	rowHover     ui.Color
	rowSelected  ui.Color
	rowSelBorder ui.Color

	card        ui.Color
	cardShadow  ui.Color
	cardShadow2 ui.Color
	floatShadow ui.Color
	edge        ui.Color

	control       ui.Color
	controlHover  ui.Color
	controlShadow bool
	divider       ui.Color
	chip          ui.Color

	attention ui.Color
	done      ui.Color
	danger    ui.Color
	accent    ui.Color

	overlay ui.Color
	popover ui.Color
	kbd     ui.Color
}

var lightPalette = palette{
	sidebar:      ui.Hex("#eceef4"),
	text:         ui.Hex("#1d1d1f"),
	textMuted:    ui.Hex("#6e6e78"),
	textFaint:    ui.Hex("#a3a3ad"),
	rowHover:     ui.RGBA(0, 0, 0, 0.04),
	rowSelected:  ui.Hex("#ffffff"),
	rowSelBorder: ui.RGBA(0, 0, 0, 0.05),

	card:          ui.Hex("#ffffff"),
	cardShadow:    ui.RGBA(0, 0, 0, 0.07),
	cardShadow2:   ui.RGBA(0, 0, 0, 0.035),
	floatShadow:   ui.RGBA(20, 20, 40, 0.14),
	control:       ui.Hex("#ffffff"),
	controlHover:  ui.Hex("#f7f7f9"),
	controlShadow: true,
	divider:       ui.RGBA(0, 0, 0, 0.06),
	chip:          ui.RGBA(0, 0, 0, 0.045),

	attention: ui.Hex("#c27b0f"),
	done:      ui.Hex("#1d1d1f"),
	danger:    ui.Hex("#d93a35"),
	accent:    ui.Hex("#3b6cf6"),

	overlay: ui.RGBA(10, 10, 30, 0.12),
	popover: ui.Hex("#ffffff"),
	kbd:     ui.RGBA(0, 0, 0, 0.04),
}

var darkPalette = palette{
	dark:         true,
	sidebar:      ui.Hex("#141418"),
	text:         ui.Hex("#ececf0"),
	textMuted:    ui.Hex("#a9a9b4"),
	textFaint:    ui.Hex("#83838f"),
	rowHover:     ui.RGBA(255, 255, 255, 0.05),
	rowSelected:  ui.RGBA(255, 255, 255, 0.10),
	rowSelBorder: ui.RGBA(255, 255, 255, 0.06),

	card:         ui.Hex("#232328"),
	cardShadow:   ui.RGBA(0, 0, 0, 0.35),
	cardShadow2:  ui.RGBA(0, 0, 0, 0.2),
	floatShadow:  ui.RGBA(0, 0, 0, 0.45),
	edge:         ui.RGBA(255, 255, 255, 0.075),
	control:      ui.Hex("#404047"),
	controlHover: ui.Hex("#4a4a52"),
	divider:      ui.RGBA(255, 255, 255, 0.07),
	chip:         ui.RGBA(255, 255, 255, 0.07),

	attention: ui.Hex("#f0a33a"),
	done:      ui.Hex("#ececf0"),
	danger:    ui.Hex("#ff6961"),
	accent:    ui.Hex("#6b93ff"),

	overlay: ui.RGBA(0, 0, 0, 0.35),
	popover: ui.Hex("#232327"),
	kbd:     ui.RGBA(255, 255, 255, 0.06),
}

func paletteFor(c *ui.Context) *palette {
	if c.Theme().Dark {
		return &darkPalette
	}
	return &lightPalette
}

func backdrop(c *ui.Context, pal *palette, row ui.Element, s settings.Settings) {
	base, glow := washes(s.Background, pal.dark)
	if s.Background == "plain" {
		bg := pal.sidebar
		if c.Vibrancy() {
			bg = bg.Alpha(0.82)
		}
		c.Root().Background(bg)
		return
	}
	alpha := float32(1)
	if c.Vibrancy() {
		alpha = 0.95
	}
	stops := make([]ui.GradientStop, len(base.Stops))
	for i, st := range base.Stops {
		stops[i] = ui.GradientStop{Color: st.Color.Alpha(alpha), At: st.At}
	}
	base.Stops = stops
	c.Root().LinearGradient(base)
	row.LinearGradient(glow)
}

func washes(id string, dark bool) (base, glow ui.LinearGradient) {
	w, ok := washTable[id]
	if !ok {
		w = washTable["rex"]
	}
	stops := func(hexes ...string) []ui.GradientStop {
		out := make([]ui.GradientStop, len(hexes))
		for i, h := range hexes {
			out[i] = ui.GradientStop{Color: ui.Hex(h), At: float32(i) / float32(len(hexes)-1)}
		}
		return out
	}
	glowOf := func(c ui.Color) ui.LinearGradient {
		return ui.LinearGradient{Angle: 225, Oklab: true, Stops: []ui.GradientStop{{Color: c, At: 0}, {Color: c.Alpha(0), At: 0.5}}}
	}
	if dark {
		return ui.LinearGradient{Angle: 180, Oklab: true, Stops: stops(w.dark...)}, glowOf(w.darkGlow)
	}
	return ui.LinearGradient{Angle: 180, Oklab: true, Stops: stops(w.light...)}, glowOf(w.lightGlow)
}

type wash struct {
	light, dark         []string
	lightGlow, darkGlow ui.Color
}

var washTable = map[string]wash{
	"rex": {
		light: []string{"#ebe0f3", "#f3e7f1", "#f7eee4", "#f6efcf"}, lightGlow: ui.RGBA(247, 220, 241, 0.75),
		dark: []string{"#17141f", "#191620", "#1b1916"}, darkGlow: ui.RGBA(120, 70, 120, 0.22),
	},
	"mist": {
		light: []string{"#e3e9f4", "#ebeff6", "#f1f2f3", "#f2f1ec"}, lightGlow: ui.RGBA(210, 225, 248, 0.75),
		dark: []string{"#12151b", "#14171d", "#17181b"}, darkGlow: ui.RGBA(70, 100, 150, 0.22),
	},
	"meadow": {
		light: []string{"#e2eee5", "#ebf2e7", "#f3f2e2", "#f5f0d6"}, lightGlow: ui.RGBA(214, 240, 222, 0.75),
		dark: []string{"#111814", "#141a16", "#181a14"}, darkGlow: ui.RGBA(70, 130, 95, 0.2),
	},
	"dusk": {
		light: []string{"#e4dbf3", "#efdeee", "#f6e3dc", "#f7e6cf"}, lightGlow: ui.RGBA(246, 206, 222, 0.8),
		dark: []string{"#191321", "#1e161e", "#221a17"}, darkGlow: ui.RGBA(150, 80, 110, 0.25),
	},
	"plain": {
		light: []string{"#eceef4", "#eceef4"}, lightGlow: ui.RGBA(0, 0, 0, 0),
		dark: []string{"#141418", "#141418"}, darkGlow: ui.RGBA(0, 0, 0, 0),
	},
}
