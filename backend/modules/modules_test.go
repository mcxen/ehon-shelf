package modules

import "testing"

func TestGroupsCoverEveryModule(t *testing.T) {
	known := map[string]bool{}
	for _, g := range Groups() {
		if g.ID == "" || g.Title == "" {
			t.Fatalf("bad group %+v", g)
		}
		if known[g.ID] {
			t.Fatal("duplicate group", g.ID)
		}
		known[g.ID] = true
		if len(In(g.ID)) == 0 {
			t.Fatal("empty group", g.ID)
		}
	}
	seen := map[string]bool{}
	for _, m := range All() {
		if seen[m.ID] {
			t.Fatal("duplicate module", m.ID)
		}
		seen[m.ID] = true
		if !known[m.Group] || m.Title == "" || m.Summary == "" || m.Body == "" {
			t.Fatalf("bad module %+v", m)
		}
	}
	if _, ok := ByID("overview"); !ok {
		t.Fatal("overview missing")
	}
}
