package main

import (
	"bytes"
	"image"
	"image/jpeg"
	"os"
	"path/filepath"
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

// The edge pads sit over the page well. Right to left, the left pad goes
// forward and the right pad goes back, without the keys.
func TestReaderTurnPads(t *testing.T) {
	a := fixture(t)
	openFixtureBook(t, a)

	tt := ui.NewTester(a.viewRoot, 1000, 700)
	tt.Frame()
	if err := tt.Click("左翻页"); err != nil {
		t.Fatal(err)
	}
	if a.pos != 1 {
		t.Fatalf("left pad pos=%d, want 1", a.pos)
	}
	if err := tt.Click("右翻页"); err != nil {
		t.Fatal(err)
	}
	if a.pos != 0 {
		t.Fatalf("right pad pos=%d, want 0", a.pos)
	}
}

// At zoom 1 the wheel turns a page. Past that it pans, so the position stays.
func TestReaderWheelTurnsWhenFitted(t *testing.T) {
	a := fixture(t)
	openFixtureBook(t, a)

	tt := ui.NewTester(a.viewRoot, 1000, 700)
	tt.Frame()
	tt.Frame()
	tt.Wheel(500, 400, 0, 40)
	if a.pos != 1 {
		t.Fatalf("wheel pos=%d, want 1", a.pos)
	}
	a.zoom = 2
	tt.Frame()
	tt.Wheel(500, 400, 0, 40)
	if a.pos != 1 {
		t.Fatalf("wheel while zoomed pos=%d, want it to stay 1", a.pos)
	}
}

// At zoom 1 the whole scan stays inside the well: a portrait page fills the
// height under the toolbar, and a wide page shrinks to the width instead of
// running off the sides.
func TestReaderPageFitsTheWell(t *testing.T) {
	const winW, winH = 1200, 800
	const bar, inset = 60, 20
	a := fixture(t)
	openFixtureBook(t, a)
	tt := ui.NewTester(a.viewRoot, winW, winH)
	tt.Frame()
	tt.Frame()

	x0, y0, x1, y1 := grayBox(t, tt)
	pw, ph := x1-x0+1, y1-y0+1
	if ph < 680 || ph > 720 {
		t.Fatalf("portrait %dx%d at %d,%d, want the height under the toolbar", pw, ph, x0, y0)
	}
	if pw*4 < ph*3-ph/10 || pw*4 > ph*3+ph/10 {
		t.Fatalf("portrait %dx%d is not 3:4", pw, ph)
	}
	if mid := (x0 + x1 + 1) / 2; mid < winW/2-15 || mid > winW/2+15 {
		t.Fatalf("portrait not centered, x %d-%d", x0, x1)
	}
	if y0 < bar || y1 >= winH-inset/2 {
		t.Fatalf("portrait runs into the toolbar or past the window, y %d-%d", y0, y1)
	}

	a.spread = true
	tt.Frame()
	sx0, sy0, sx1, sy1 := grayBox(t, tt)
	sw, sh := sx1-sx0+1, sy1-sy0+1
	if sh < 680 || sh > 720 || sw < 1000 || sw > winW-2*inset {
		t.Fatalf("spread %dx%d at %d,%d, want both pages inside the well", sw, sh, sx0, sy0)
	}

	writeFixturePages(t, a, 200, 80, 210)
	a.spread = false
	a.zoom = 1
	tt.Frame()
	tt.Frame()
	x0, y0, x1, y1 = grayBox(t, tt)
	pw, ph = x1-x0+1, y1-y0+1
	if pw*2 < ph*5-pw/10 || pw*2 > ph*5+pw/10 {
		t.Fatalf("wide %dx%d is not 5:2, so the sides are still clipped", pw, ph)
	}
	if ph > 600 {
		t.Fatalf("wide height %d still fills the well", ph)
	}
	if x0 < inset/2 || x1 > winW-inset/2 || y0 < bar || y1 >= winH-inset/2 {
		t.Fatalf("wide sits outside the window: %d,%d %dx%d", x0, y0, pw, ph)
	}
	top, bottom := y0-bar, winH-(y1+1)
	if top < bottom-40 || top > bottom+40 {
		t.Fatalf("wide is not centered in the well: top %d, bottom %d", top, bottom)
	}

	// Zoom grows that fitted size, so the scan leaves the well and can be panned.
	fittedBottom := y1
	a.zoom = 2
	tt.Frame()
	_, zy0, _, zy1 := grayBox(t, tt)
	if zy1 < fittedBottom+80 || zy0 > bar+40 {
		t.Fatalf("zoomed wide y %d-%d did not grow past the fitted page", zy0, zy1)
	}
}

// grayBox is the bounding box of the flat gray scan. Header text is near
// white, and the well is near black, so neither is part of the box.
func grayBox(t *testing.T, tt *ui.Tester) (x0, y0, x1, y1 int) {
	t.Helper()
	img := tt.Image()
	b := img.Bounds()
	x0, y0, x1, y1 = b.Max.X, b.Max.Y, 0, 0
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, _ := img.At(x, y).RGBA()
			rr, gg, bb := int(r>>8), int(g>>8), int(bl>>8)
			dg, db := rr-gg, gg-bb
			if dg < 0 {
				dg = -dg
			}
			if db < 0 {
				db = -db
			}
			if rr > 160 && rr < 230 && dg < 12 && db < 12 && y > 70 {
				if x < x0 {
					x0 = x
				}
				if y < y0 {
					y0 = y
				}
				if x > x1 {
					x1 = x
				}
				if y > y1 {
					y1 = y
				}
			}
		}
	}
	if x1 < x0 || y1 < y0 {
		t.Fatal("no scan pixels")
	}
	return x0, y0, x1, y1
}

func writeFixturePages(t *testing.T, a *app, w, h int, pix byte) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for i := range img.Pix {
		img.Pix[i] = pix
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 95}); err != nil {
		t.Fatal(err)
	}
	for _, p := range a.book.Pages {
		path := filepath.Join(a.book.Dir, filepath.FromSlash(p.File))
		if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	a.pages = map[int]*ui.Bitmap{}
	a.loading = map[int]bool{}
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
