package settings

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestDefaultRoundTrip(t *testing.T) {
	dir := t.TempDir()
	if err := Save(dir, Default()); err != nil {
		t.Fatal(err)
	}
	got, probs, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(probs) != 0 {
		t.Fatalf("problems: %v", probs)
	}
	if got.Language != "system" || !got.Vibrancy || got.FontScale != 1 || got.SidebarWidth != 248 {
		t.Fatalf("%+v", got)
	}
	if got.Shortcut(ShortcutPalette) != "CmdOrCtrl+K" {
		t.Fatal(got.Shortcut(ShortcutPalette))
	}
}

func TestFixtures(t *testing.T) {
	root := repoRoot(t)
	t.Run("valid", func(t *testing.T) {
		s, probs := loadFixture(t, filepath.Join(root, "testdata", "settings", "valid.json"))
		if len(probs) != 0 {
			t.Fatal(probs)
		}
		if s.Appearance != "dark" || s.Language != "zh-Hans" || s.FontFamily != "system" {
			t.Fatalf("%+v", s)
		}
		if s.Shortcut(ShortcutSidebar) != "CmdOrCtrl+Shift+B" {
			t.Fatal(s.Shortcut(ShortcutSidebar))
		}
	})
	t.Run("unknown", func(t *testing.T) {
		_, probs := loadFixture(t, filepath.Join(root, "testdata", "settings", "unknown.json"))
		if !slices.Contains(probs, "unknown:notASetting") {
			t.Fatal(probs)
		}
	})
	t.Run("bad-enum", func(t *testing.T) {
		s, probs := loadFixture(t, filepath.Join(root, "testdata", "settings", "bad-enum.json"))
		if s.Appearance != "system" || s.Background != "rex" {
			t.Fatalf("normalized %+v", s)
		}
		if !slices.Contains(probs, "appearance") || !slices.Contains(probs, "background") {
			t.Fatal(probs)
		}
	})
}

func TestValidAccelerator(t *testing.T) {
	for _, s := range []string{"", "CmdOrCtrl+B", "CmdOrCtrl+,", "Ctrl+Shift+K"} {
		if !ValidAccelerator(s) {
			t.Errorf("want valid: %q", s)
		}
	}
	for _, s := range []string{"B", "CmdOrCtrl+", "not a shortcut", "CmdOrCtrl+设置"} {
		if ValidAccelerator(s) {
			t.Errorf("want invalid: %q", s)
		}
	}
}

func loadFixture(t *testing.T, path string) (Settings, []string) {
	t.Helper()
	dir := t.TempDir()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, fileName), b, 0o644); err != nil {
		t.Fatal(err)
	}
	s, probs, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	return s, probs
}

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for range 8 {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatal("go.mod not found")
	return ""
}
