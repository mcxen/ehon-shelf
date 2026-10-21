// Package i18n loads the shell's catalogs. The files live in
// resources/locales; a missing key falls back to zh-Hans, then en.
package i18n

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Catalogs maps a language tag to its messages.
type Catalogs map[string]map[string]string

// LoadDir reads every *.json in dir. The file name, without .json, is
// the language tag.
func LoadDir(dir string) (Catalogs, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	out := Catalogs{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		var msgs map[string]string
		if err := json.Unmarshal(b, &msgs); err != nil {
			return nil, fmt.Errorf("%s: %w", e.Name(), err)
		}
		out[strings.TrimSuffix(e.Name(), ".json")] = msgs
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("i18n: no catalogs in %s", dir)
	}
	return out, nil
}

// Text returns the message. A missing key tries zh-Hans, then en, then
// the key itself so a gap stays visible.
func (c Catalogs) Text(lang, key string) string {
	if lang == "" || lang == "system" {
		lang = SystemTag()
	}
	for _, tag := range []string{lang, "zh-Hans", "en"} {
		if msg, ok := c[tag][key]; ok && msg != "" {
			return msg
		}
	}
	return key
}

// SameKeys reports whether every catalog has the same keys.
func (c Catalogs) SameKeys() error {
	var base string
	var want map[string]string
	for tag, msgs := range c {
		if want == nil {
			base, want = tag, msgs
			continue
		}
		for k := range want {
			if _, ok := msgs[k]; !ok {
				return fmt.Errorf("%s missing %s (present in %s)", tag, k, base)
			}
		}
		for k := range msgs {
			if _, ok := want[k]; !ok {
				return fmt.Errorf("%s has extra %s", tag, k)
			}
		}
	}
	return nil
}

// SystemTag is a best effort at the desktop language when the app has
// not chosen one: GODGO_LANG, else LANG, else zh-Hans.
func SystemTag() string {
	if v := os.Getenv("GODGO_LANG"); v != "" {
		return normalize(v)
	}
	if v := os.Getenv("LC_ALL"); v != "" {
		return normalize(v)
	}
	if v := os.Getenv("LANG"); v != "" {
		return normalize(v)
	}
	return "zh-Hans"
}

func normalize(tag string) string {
	tag = strings.Split(tag, ".")[0]
	tag = strings.ReplaceAll(tag, "_", "-")
	switch {
	case strings.HasPrefix(tag, "zh"):
		return "zh-Hans"
	case strings.HasPrefix(tag, "en"):
		return "en"
	default:
		return "en"
	}
}

// Ago is a short "how long ago" for the updates page.
func Ago(lang string, t time.Time, now time.Time) string {
	if t.IsZero() {
		if lang == "en" {
			return "not yet"
		}
		return "还没有"
	}
	d := now.Sub(t)
	zh := lang != "en"
	switch {
	case d < time.Minute:
		if zh {
			return "刚刚"
		}
		return "just now"
	case d < time.Hour:
		n := int(d / time.Minute)
		if zh {
			return fmt.Sprintf("%d 分钟前", n)
		}
		return fmt.Sprintf("%d min ago", n)
	case d < 48*time.Hour:
		n := int(d / time.Hour)
		if zh {
			return fmt.Sprintf("%d 小时前", n)
		}
		return fmt.Sprintf("%d h ago", n)
	default:
		n := int(d / (24 * time.Hour))
		if zh {
			return fmt.Sprintf("%d 天前", n)
		}
		return fmt.Sprintf("%d d ago", n)
	}
}
