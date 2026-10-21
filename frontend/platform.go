package ehonui

import (
	"os/exec"
	"runtime"

	"github.com/egoist/mygo"
)

func openPath(path string) error {
	return mygo.Shell.OpenPath(path)
}

func openText(path string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("/usr/bin/open", "-t", path).Start()
	case "windows":
		return exec.Command("notepad.exe", path).Start()
	default:
		return exec.Command("xdg-open", path).Start()
	}
}
