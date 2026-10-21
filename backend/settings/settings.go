// Package settings is GodGo's configuration: one JSON file the shell
// writes from its Settings window and reads on the next launch.
package settings

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
)

const fileName = "settings.json"

// Shortcut ids the shell binds.
const (
	ShortcutSidebar  = "sidebar.toggle"
	ShortcutSettings = "settings.open"
	ShortcutPalette  = "palette.open"
	ShortcutUpdates  = "updates.check"
)

// DefaultShortcuts are the accelerators of a new install. A shortcut
// stored as "" unbinds it.
var DefaultShortcuts = map[string]string{
	ShortcutSidebar:  "CmdOrCtrl+B",
	ShortcutSettings: "CmdOrCtrl+,",
	ShortcutPalette:  "CmdOrCtrl+K",
	ShortcutUpdates:  "CmdOrCtrl+U",
}

// Languages the shell is written in.
var Languages = []string{"zh-Hans", "en"}

// Backgrounds are the washes the window can show. The ids match the
// washes drawn by the shell, which follow Kero's.
var Backgrounds = []string{"rex", "mist", "meadow", "dusk", "plain"}

// FontScales are the interface sizes the appearance page offers.
var FontScales = []float64{0.9, 1, 1.1, 1.25}

// Settings are the user's choices. The zero value of a field is never a
// choice of its own: Load starts from Default, so a key the file omits
// keeps its default.
type Settings struct {
	Language    string `json:"language"`
	ConfirmQuit bool   `json:"confirmQuit"`
	RestorePage bool   `json:"restorePage"`

	Appearance string  `json:"appearance"`
	Vibrancy   bool    `json:"vibrancy"`
	Background string  `json:"background"`
	FontFamily string  `json:"fontFamily"` // "" the fonts the shell carries; "system" the desktop's
	FontScale  float64 `json:"fontScale"`

	AutoCheck   bool `json:"autoCheck"`
	AutoInstall bool `json:"autoInstall"`

	RunOnSave bool `json:"runOnSave"`

	NotifyUpdate bool `json:"notifyUpdate"`
	NotifyDone   bool `json:"notifyDone"`
	NotifyFail   bool `json:"notifyFail"`
	NotifySound  bool `json:"notifySound"`

	Shortcuts map[string]string `json:"shortcuts,omitempty"`

	SidebarHidden bool            `json:"sidebarHidden"`
	SidebarWidth  float32         `json:"sidebarWidth"`
	Collapsed     map[string]bool `json:"collapsed,omitempty"`
	LastModule    string          `json:"lastModule"`
}

// Default returns the settings of a new install.
func Default() Settings {
	return Settings{
		Language:     "system",
		ConfirmQuit:  true,
		RestorePage:  true,
		Appearance:   "system",
		Vibrancy:     true,
		Background:   "rex",
		FontScale:    1,
		AutoCheck:    true,
		NotifyUpdate: true,
		NotifyDone:   true,
		NotifyFail:   true,
		SidebarWidth: 248,
		LastModule:   "overview",
	}
}

// Shortcut returns the accelerator of id: the user's, else the default.
func (s Settings) Shortcut(id string) string {
	if s.Shortcuts != nil {
		if v, ok := s.Shortcuts[id]; ok {
			return v
		}
	}
	return DefaultShortcuts[id]
}

// Clone returns a copy that shares nothing with s.
func (s Settings) Clone() Settings {
	if s.Shortcuts != nil {
		m := make(map[string]string, len(s.Shortcuts))
		for k, v := range s.Shortcuts {
			m[k] = v
		}
		s.Shortcuts = m
	}
	if s.Collapsed != nil {
		m := make(map[string]bool, len(s.Collapsed))
		for k, v := range s.Collapsed {
			m[k] = v
		}
		s.Collapsed = m
	}
	return s
}

// normalize puts values back into range and reports what it changed.
func (s *Settings) normalize() []string {
	d := Default()
	var probs []string
	oneOf := func(name string, v *string, def string, ok ...string) {
		if slices.Contains(ok, *v) {
			return
		}
		probs = append(probs, name)
		*v = def
	}
	oneOf("language", &s.Language, d.Language, append([]string{"system"}, Languages...)...)
	oneOf("appearance", &s.Appearance, d.Appearance, "system", "light", "dark")
	oneOf("background", &s.Background, d.Background, Backgrounds...)
	oneOf("fontFamily", &s.FontFamily, d.FontFamily, "", "system")
	if s.FontScale < 0.8 || s.FontScale > 1.5 {
		probs = append(probs, "fontScale")
		s.FontScale = d.FontScale
	}
	if s.SidebarWidth < 200 || s.SidebarWidth > 420 {
		probs = append(probs, "sidebarWidth")
		s.SidebarWidth = d.SidebarWidth
	}
	if s.LastModule == "" {
		s.LastModule = d.LastModule
	}
	for id, v := range s.Shortcuts {
		if _, known := DefaultShortcuts[id]; !known || !ValidAccelerator(v) {
			probs = append(probs, "shortcuts."+id)
			delete(s.Shortcuts, id)
		}
	}
	if len(s.Shortcuts) == 0 {
		s.Shortcuts = nil
	}
	slices.Sort(probs)
	return probs
}

// ValidAccelerator reports whether s is a shortcut MyGo accepts, or empty,
// which unbinds one. Modifiers are CmdOrCtrl, Cmd, Ctrl, Alt, Shift and
// Super; the last piece is the key, and "," is the comma key.
func ValidAccelerator(s string) bool {
	if s == "" {
		return true
	}
	if len(s) > 48 || strings.ContainsAny(s, " \t\n") {
		return false
	}
	parts := strings.Split(s, "+")
	if len(parts) < 2 {
		return false
	}
	mods := map[string]bool{"CmdOrCtrl": true, "Cmd": true, "Ctrl": true, "Alt": true, "Shift": true, "Super": true}
	for _, p := range parts[:len(parts)-1] {
		if !mods[p] {
			return false
		}
	}
	key := parts[len(parts)-1]
	if key == "" {
		return false
	}
	for _, r := range key {
		if r > 127 {
			return false
		}
	}
	return true
}

// Load reads the settings in dir. Missing file returns the defaults and
// no problems. Problems are unknown keys and values normalize replaced.
func Load(dir string) (Settings, []string, error) {
	b, err := os.ReadFile(filepath.Join(dir, fileName))
	if os.IsNotExist(err) {
		return Default(), nil, nil
	}
	if err != nil {
		return Default(), nil, err
	}
	s := Default()
	if err := json.Unmarshal(b, &s); err != nil {
		return Default(), []string{"json"}, err
	}
	probs := append(unknownKeys(b), s.normalize()...)
	slices.Sort(probs)
	return s, probs, nil
}

// Save writes s into dir, atomically.
func Save(dir string, s Settings) error {
	s.normalize()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	path := filepath.Join(dir, fileName)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func unknownKeys(b []byte) []string {
	var m map[string]json.RawMessage
	if json.Unmarshal(b, &m) != nil {
		return nil
	}
	known := map[string]bool{}
	t := reflect.TypeOf(Settings{})
	for i := range t.NumField() {
		name := strings.Split(t.Field(i).Tag.Get("json"), ",")[0]
		if name != "" && name != "-" {
			known[name] = true
		}
	}
	var probs []string
	for k := range m {
		if !known[k] {
			probs = append(probs, "unknown:"+k)
		}
	}
	slices.Sort(probs)
	return probs
}
