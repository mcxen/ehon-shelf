// Package ehonui adapts a GodGo-style shell (sidebar, card, chrome) to
// EhonShelf.
//
// The layout, palette, chrome metrics and elevation helpers are adapted from
// github.com/egoist/kero (MIT) via GodGo's frontend. GodGo is a shell with no
// business of its own; here the sidebar carries the bookshelf instead of a
// module list.
package ehonui

import (
	"github.com/egoist/mygo"

	"github.com/mcxen/ehon-shelf/backend/settings"
)

// App is the shell. main keeps its own app and delegates drawing here.
type App struct {
	set settings.Settings

	sections []Section
	current  string
	onSelect func(id string)
	win      *mygo.Window

	// ToggleSidebar is wired by main if it wants a shortcut for it.
	ToggleSidebar func()
	// Extend is the sidebar's plus button hook.
	Extend func()
}

// New returns a shell bound to settings.
func New(s settings.Settings) *App { return &App{set: s} }

// SetSettings replaces the settings the theme and font code read.
func (a *App) SetSettings(s settings.Settings) { a.set = s }

// SetWindow is the window whose buttons the card header draws on Windows and Linux.
func (a *App) SetWindow(w *mygo.Window) { a.win = w }

// Settings returns the current settings.
func (a *App) Settings() settings.Settings { return a.set }
