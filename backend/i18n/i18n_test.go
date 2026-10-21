package i18n

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestShippedCatalogsMatch(t *testing.T) {
	c, err := LoadDir(filepath.Join(repoRoot(t), "resources", "locales"))
	if err != nil {
		t.Fatal(err)
	}
	if err := c.SameKeys(); err != nil {
		t.Fatal(err)
	}
	if c.Text("zh-Hans", "settings.title") == "settings.title" {
		t.Fatal("missing settings.title")
	}
	if c.Text("en", "settings.title") == c.Text("zh-Hans", "settings.title") {
		t.Fatal("en and zh-Hans share the settings title")
	}
}

func TestPartialFixtureFallsBack(t *testing.T) {
	c, err := LoadDir(filepath.Join(repoRoot(t), "testdata", "i18n"))
	if err != nil {
		t.Fatal(err)
	}
	if got := c.Text("zh-Hans", "only.english"); got != "English only" {
		t.Fatalf("fallback: %q", got)
	}
	if got := c.Text("zh-Hans", "both"); got != "两者都有" {
		t.Fatalf("zh: %q", got)
	}
}

func TestAgo(t *testing.T) {
	now := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	if Ago("zh-Hans", time.Time{}, now) != "还没有" {
		t.Fatal(Ago("zh-Hans", time.Time{}, now))
	}
	if Ago("en", now.Add(-2*time.Minute), now) != "2 min ago" {
		t.Fatal(Ago("en", now.Add(-2*time.Minute), now))
	}
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
