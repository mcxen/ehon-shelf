package ehonui

import (
	"os"
	"path/filepath"

	"github.com/egoist/mygo/ui"
)

// The shell carries Inter for Latin, the Apple-like face AvaMedia subsets
// for its desktop, and Noto Sans SC for Simplified Chinese. Both are SIL
// OFL 1.1. Text asks for Inter first and falls through the list.

const (
	fontInter = "Inter"
	fontNoto  = "Noto Sans SC"
	fontStack = "Inter, Noto Sans SC"
)

var fontsLoaded bool

func loadFonts() {
	if fontsLoaded {
		return
	}
	inter := registerFamily("fonts/inter", fontInter)
	noto := registerFamily("fonts/noto-sans-sc", fontNoto)
	fontsLoaded = inter || noto
}

func registerFamily(dir, family string) bool {
	registered := false
	for _, root := range resourceDirs() {
		files, _ := filepath.Glob(filepath.Join(root, dir, "*.ttf"))
		for _, f := range files {
			data, err := os.ReadFile(f)
			if err != nil {
				continue
			}
			if ui.RegisterFont(data, family) == nil {
				registered = true
			}
		}
		if registered {
			return true
		}
	}
	return false
}

func (a *App) fontFamily() string {
	if a.set.FontFamily == "system" || !fontsLoaded {
		return ""
	}
	return fontStack
}
