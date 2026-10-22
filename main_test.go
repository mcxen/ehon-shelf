package main

import (
	"archive/zip"
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
)

// fixture makes a library with one work of three pages in a temporary home.
func fixture(t *testing.T) *app {
	t.Helper()
	home := t.TempDir()
	t.Setenv("EHONSHELF_HOME", home)
	d := filepath.Join(home, "data", "81")
	if err := os.MkdirAll(filepath.Join(d, "pages"), 0o755); err != nil {
		t.Fatal(err)
	}
	img := image.NewRGBA(image.Rect(0, 0, 60, 80))
	for i := range img.Pix {
		img.Pix[i] = 200
	}
	img.Set(1, 1, color.Black)
	var buf bytes.Buffer
	_ = jpeg.Encode(&buf, img, nil)
	m := &Meta{Item: Item{ID: 81, Title: "愛生枕", Artist: "寺沢昌次"}, PageCount: 3, Complete: true}
	for i := 1; i <= 3; i++ {
		f := filepath.Join("pages", "000"+string(rune('0'+i))+".jpg")
		_ = os.WriteFile(filepath.Join(d, f), buf.Bytes(), 0o644)
		m.Pages = append(m.Pages, Page{File: filepath.ToSlash(f)})
	}
	if err := writeJSON(filepath.Join(d, "meta.json"), m); err != nil {
		t.Fatal(err)
	}
	// A cached remote catalog, so the settings view needs no network.
	_ = writeJSON(filepath.Join(home, "data", "remote_catalog.json"), []Item{{ID: 81, Title: "愛生枕", Artist: "寺沢昌次"}, {ID: 255, Title: "秋の寝覚", Artist: "渓斎英泉"}})
	a := newApp()
	a.async = func(fn func()) { fn() }
	a.rescan()
	return a
}

func TestShelfReaderSettings(t *testing.T) {
	a := fixture(t)
	if _, err := os.Stat(filepath.Join(a.conf.DataDir, "81", "cover.jpg")); err != nil {
		t.Fatal("cover.jpg was not generated:", err)
	}
	tt := ui.NewTester(a.viewRoot, 1000, 700)
	if !tt.HasText("愛生枕") || !tt.HasText("共 1 部") {
		t.Fatalf("shelf texts %q", tt.Texts())
	}
	// Search filters the shelf.
	a.query = "不存在"
	a.filter()
	tt.Frame()
	if !tt.HasText("共 0 部") {
		t.Errorf("search: %q", tt.Texts())
	}
	a.query = ""
	a.filter()
	tt.Frame()

	// Open the book directly: this test is about the reader, and the card's
	// click target is keyed by book id, which a text click cannot reach.
	if len(a.books) == 0 {
		t.Fatal("no books")
	}
	a.openBook(&a.books[0])
	tt.Frame()
	if a.view != viewReader || !tt.HasText("第 1 / 3 图") {
		t.Fatalf("reader texts %q", tt.Texts())
	}
	// Right to left by default: the left arrow goes forward.
	tt.Key(0, ui.KeyLeft)
	if a.pos != 1 || !tt.HasText("第 2 / 3 图") {
		t.Errorf("after Left pos=%d %q", a.pos, tt.Texts())
	}
	if err := tt.Click("阅读方向：右至左"); err != nil {
		t.Fatal(err)
	}
	tt.Key(0, ui.KeyLeft)
	if a.pos != 0 {
		t.Errorf("left to right, Left should go back: pos=%d", a.pos)
	}
	if err := tt.Click("单页模式"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("第 1-2 / 3 图") {
		t.Errorf("spread %q", tt.Texts())
	}
	tt.Key(0, ui.KeyRight)
	// The position is remembered in config.json.
	if LoadConfig().Positions["81"] != 2 {
		t.Errorf("position not saved: %v", LoadConfig().Positions)
	}
	// Back to the shelf: the reader's own button is gone, Esc closes it.
	tt.Key(0, ui.KeyEscape)

	// Settings holds the library options. The catalog lives on its own page,
	// opened from the sidebar, and the click loads the cached catalog.
	a.openSettings()
	tt.Frame()
	for _, s := range []string{"数据文件夹", "下载示例数据", "远程目录"} {
		if !tt.HasText(s) {
			t.Errorf("settings lacks %q: %q", s, tt.Texts())
		}
	}
	if err := tt.Click("远程目录"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	for _, s := range []string{"秋の寝覚", "共 2 部，已下载 1 部，已选 0 部"} {
		if !tt.HasText(s) {
			t.Errorf("remote lacks %q: %q", s, tt.Texts())
		}
	}
	if err := tt.Click("全选"); err != nil {
		t.Fatal(err)
	}
	if len(a.selected) != 2 {
		t.Errorf("selected %v", a.selected)
	}
}

func TestParsers(t *testing.T) {
	list := `<a href="/enp/Cover/List/81">x</a><span class="s">愛生枕</span><span class="ss">あいしょうまくら</span><span class="sblue">寺沢昌次</span>`
	if got := ParseList(list); len(got) != 1 || got[0].ID != 81 || got[0].Artist != "寺沢昌次" {
		t.Errorf("ParseList %+v", got)
	}
	pdf := []byte("%PDF /Filter /DCTDecode /Length 4 >>\nstream\n\xff\xd8AB\xff\xd9\nendstream")
	if j := JPEGFromPDF(pdf); !bytes.Equal(j, []byte("\xff\xd8AB\xff\xd9")) {
		t.Errorf("JPEGFromPDF %q", j)
	}
	rules := parseRobots("User-agent: *\nDisallow: /private\n\nUser-agent: Bad\nDisallow: /\n", "NichibunEnpArchiver")
	if len(rules) != 1 || rules[0] != "/private" {
		t.Errorf("robots %v", rules)
	}
	sh := ParseShoshi(`<td>絵師名</td><td>寺沢昌次</td><td>板行年</td><td>西暦</td>`)
	if sh["絵師名"] != "寺沢昌次" || sh["板行年"] != "" {
		t.Errorf("shoshi %v", sh)
	}
}

func TestImportZips(t *testing.T) {
	src, dst := t.TempDir(), t.TempDir()
	zp := filepath.Join(src, "ehonshelf-1-1.zip")
	f, _ := os.Create(zp)
	zw := zip.NewWriter(f)
	for name, body := range map[string]string{"5/meta.json": `{"id":5,"title":"t","pages":[]}`, "5/pages/0001.jpg": "x"} {
		w, _ := zw.Create(name)
		w.Write([]byte(body))
	}
	zw.Close()
	f.Close()
	n, err := ImportZips([]string{zp}, dst)
	if err != nil || n != 1 {
		t.Fatal(n, err)
	}
	if bs := ScanLibrary(dst); len(bs) != 1 || bs[0].ID != 5 {
		t.Fatalf("%+v", bs)
	}
}
