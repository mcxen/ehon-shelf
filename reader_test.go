package main

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// The page area used to size itself from c.Size(), which is the whole
// window's content size — toolbar and pager included — so pages were too tall
// and the bottom of every scan was cut off. And because the pages sat in a
// Grow+Basis(0) row, zoom was squeezed back to the viewport and looked like it
// did nothing at all.

func TestReaderRendersWithPagerAndZoomControls(t *testing.T) {
	a := fixture(t)
	openFixtureBook(t, a)

	tt := ui.NewTester(a.viewRoot, 1000, 700)
	tt.Frame()

	for _, want := range []string{"放大", "缩小", "适应窗口", "‹", "›"} {
		if !tt.HasText(want) {
			t.Errorf("reader is missing %q; texts=%q", want, tt.Texts())
		}
	}
}

// Zoom must survive the render cycle: the value the buttons set is what the
// page area is drawn with, so it must not be clamped or reset while building.
func TestReaderZoomButtonsChangeZoom(t *testing.T) {
	a := fixture(t)
	openFixtureBook(t, a)
	a.zoom = 1

	tt := ui.NewTester(a.viewRoot, 1000, 700)
	tt.Frame()
	if err := tt.Click("放大"); err != nil {
		t.Fatalf("click 放大: %v", err)
	}
	if a.zoom <= 1 {
		t.Fatalf("zoom after 放大 = %v, want it above 1", a.zoom)
	}

	zoomed := a.zoom
	tt.Frame()
	if a.zoom != zoomed {
		t.Fatalf("zoom reset during render: %v -> %v", zoomed, a.zoom)
	}

	if err := tt.Click("缩小"); err != nil {
		t.Fatalf("click 缩小: %v", err)
	}
	if a.zoom >= zoomed {
		t.Fatalf("zoom after 缩小 = %v, want it below %v", a.zoom, zoomed)
	}

	if err := tt.Click("适应窗口"); err != nil {
		t.Fatalf("click 适应窗口: %v", err)
	}
	if a.zoom != 1 {
		t.Fatalf("zoom after 适应窗口 = %v, want 1", a.zoom)
	}
}

// Zoom must never go below 1: below that the page would shrink into the
// middle of a black area instead of fitting it.
func TestReaderZoomHasAFloor(t *testing.T) {
	a := fixture(t)
	openFixtureBook(t, a)
	a.zoom = 1

	tt := ui.NewTester(a.viewRoot, 1000, 700)
	for i := 0; i < 5; i++ {
		tt.Frame()
		if err := tt.Click("缩小"); err != nil {
			break
		}
	}
	if a.zoom < 1 {
		t.Fatalf("zoom = %v, want it never below 1", a.zoom)
	}
}

// A page taller than the area must be reachable: the reader scrolls, so at
// any zoom the frame renders without clipping the controls away.
func TestReaderFrameAtHighZoom(t *testing.T) {
	a := fixture(t)
	openFixtureBook(t, a)
	a.zoom = 4

	tt := ui.NewTester(a.viewRoot, 900, 600)
	tt.Frame()

	if !tt.HasText("放大") {
		t.Fatalf("controls vanished at zoom 4; texts=%q", tt.Texts())
	}
}

// openFixtureBook puts the test's sample work on screen, the way clicking a
// cover does. readerView dereferences a.book, so a fixture without one panics.
func openFixtureBook(t *testing.T, a *app) {
	t.Helper()
	if len(a.books) == 0 {
		t.Fatal("fixture scanned no books")
	}
	a.openBook(&a.books[0])
	if a.book == nil {
		t.Fatal("openBook left no current book")
	}
}
