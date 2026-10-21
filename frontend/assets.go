package ehonui

import (
	"os"
	"path/filepath"

	"github.com/egoist/mygo"
)

// resourceDirs are where shipped files can be: the app's resource
// directory, then resources/ walked up from the working directory, which
// is where `go test` and `go run` find the source tree.
func resourceDirs() []string {
	var dirs []string
	if dir, err := mygo.App.Path(mygo.PathResources); err == nil && dir != "" {
		dirs = append(dirs, dir)
	}
	wd, err := os.Getwd()
	if err != nil {
		return dirs
	}
	dir := wd
	for range 6 {
		dirs = append(dirs, filepath.Join(dir, "resources"))
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return dirs
}

func readResource(rel string) ([]byte, error) {
	var last error
	for _, dir := range resourceDirs() {
		b, err := os.ReadFile(filepath.Join(dir, rel))
		if err == nil {
			return b, nil
		}
		last = err
	}
	if last == nil {
		last = os.ErrNotExist
	}
	return nil, last
}

func resourceFile(rel string) string {
	for _, dir := range resourceDirs() {
		path := filepath.Join(dir, rel)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}

// moduleRoot is the GodGo source tree, when the process was started
// inside it. An installed app has none, and workflow tasks stay idle.
func moduleRoot() string {
	wd, err := os.Getwd()
	if err != nil {
		return ""
	}
	dir := wd
	for range 6 {
		b, err := os.ReadFile(filepath.Join(dir, "go.mod"))
		if err == nil && len(b) > 0 && bytesContains(b, "module github.com/mcxen/ehon-shelf") {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

func bytesContains(b []byte, s string) bool {
	return len(b) >= len(s) && (string(b) == s || len(s) == 0 || indexBytes(b, s) >= 0)
}

func indexBytes(b []byte, s string) int {
	for i := 0; i+len(s) <= len(b); i++ {
		if string(b[i:i+len(s)]) == s {
			return i
		}
	}
	return -1
}
