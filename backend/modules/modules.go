// Package modules is the shell's sidebar: the pieces a project built on
// GodGo is made of, in groups. Adding a module is a registration here
// plus the text keys in resources/locales.
package modules

// Module is one row of the sidebar.
type Module struct {
	ID      string
	Group   string
	Title   string // i18n key
	Summary string
	Body    string
}

// Group is a section of the sidebar, in the order it is shown.
type Group struct {
	ID    string
	Title string
}

// Groups is the sidebar's sections, top to bottom.
func Groups() []Group {
	return []Group{
		{ID: "workbench", Title: "group.workbench"},
		{ID: "resources", Title: "group.resources"},
		{ID: "develop", Title: "group.develop"},
	}
}

// All is every module, in the order its group shows them.
func All() []Module {
	return []Module{
		{ID: "overview", Group: "workbench", Title: "module.overview.title", Summary: "module.overview.summary", Body: "module.overview.body"},
		{ID: "spec", Group: "workbench", Title: "module.spec.title", Summary: "module.spec.summary", Body: "module.spec.body"},
		{ID: "workflow", Group: "workbench", Title: "module.workflow.title", Summary: "module.workflow.summary", Body: "module.workflow.body"},
		{ID: "updates", Group: "workbench", Title: "module.updates.title", Summary: "module.updates.summary", Body: "module.updates.body"},
		{ID: "fonts", Group: "resources", Title: "module.fonts.title", Summary: "module.fonts.summary", Body: "module.fonts.body"},
		{ID: "icons", Group: "resources", Title: "module.icons.title", Summary: "module.icons.summary", Body: "module.icons.body"},
		{ID: "testdata", Group: "resources", Title: "module.testdata.title", Summary: "module.testdata.summary", Body: "module.testdata.body"},
		{ID: "backend", Group: "develop", Title: "module.backend.title", Summary: "module.backend.summary", Body: "module.backend.body"},
		{ID: "frontend", Group: "develop", Title: "module.frontend.title", Summary: "module.frontend.summary", Body: "module.frontend.body"},
	}
}

// ByID returns the module, and whether it is registered.
func ByID(id string) (Module, bool) {
	for _, m := range All() {
		if m.ID == id {
			return m, true
		}
	}
	return Module{}, false
}

// In returns the modules of one group.
func In(group string) []Module {
	var out []Module
	for _, m := range All() {
		if m.Group == group {
			out = append(out, m)
		}
	}
	return out
}
