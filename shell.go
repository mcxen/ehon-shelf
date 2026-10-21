// Sidebar shell adapted from github.com/egoist/kero (MIT) via GodGo.
package main

import (
	ehonui "github.com/mcxen/ehon-shelf/frontend"

	"github.com/egoist/mygo/ui"
)

// The four views the app has are the four sidebar rows. The reader stays a
// full-window state of its own, because a page needs the width.
const (
	secShelf    = "shelf"
	secRemote   = "remote"
	secLogs     = "logs"
	secSettings = "settings"
)

func (a *app) sections() []ehonui.Section {
	return []ehonui.Section{
		{ID: secShelf, Title: "书架", Icon: "f7-folder"},
		{ID: secRemote, Title: "远程目录", Icon: "f7-sparkles"},
		{ID: secLogs, Title: "日志", Icon: "f7-slider-horizontal-3"},
		{ID: secSettings, Title: "设置", Icon: "f7-gear-alt"},
	}
}

// sectionFor maps the view the rest of the code sets to a sidebar row.
func sectionFor(v int) string {
	switch v {
	case viewSettings:
		return secSettings
	case viewLogs:
		return secLogs
	default:
		return secShelf
	}
}

// viewFor is the reverse: a clicked row sets the view the code already reads.
func viewFor(sec string) int {
	switch sec {
	case secSettings:
		return viewSettings
	case secLogs:
		return viewLogs
	default:
		return viewShelf
	}
}

var (
	_ = secRemote
	_ = ehonui.Section{}
	_ = ui.Context{}
)
