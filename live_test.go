package main

import (
	"os"
	"testing"
)

// TestLiveProxyFetch is the guard that the proxy plumbing actually works end
// to end. Skipped unless EHONSHELF_LIVE=1, because it needs a real proxy and
// the real site:
//
//	EHONSHELF_LIVE=1 EHONSHELF_PROXY=http://127.0.0.1:7897 \
//	  go test -run TestLiveProxyFetch -v
func TestLiveProxyFetch(t *testing.T) {
	if os.Getenv("EHONSHELF_LIVE") != "1" {
		t.Skip("set EHONSHELF_LIVE=1 to hit the real site")
	}
	if HttpProxy == "" {
		t.Fatal("EHONSHELF_PROXY must be set for the live test")
	}
	old := Base
	Base = "https://lapis.nichibun.ac.jp"
	defer func() { Base = old }()

	dir := t.TempDir()
	c := NewCrawler(dir, 1.2)

	// 1. The book list must parse into the full catalogue.
	items, err := c.ListItems()
	if err != nil {
		t.Fatalf("ListItems: %v", err)
	}
	t.Logf("catalogue: %d items", len(items))
	if len(items) < 400 {
		t.Fatalf("expected ~470 items, got %d", len(items))
	}
	first := items[0]
	t.Logf("first item: id=%d title=%q artist=%q", first.ID, first.Title, first.Artist)
	if first.ID == 0 || first.Title == "" {
		t.Fatalf("first item did not parse: %+v", first)
	}

	// 2. The picture list frame must yield page references.
	pics, err := c.getText("/enp/Picture/ListContents/1/1/1")
	if err != nil {
		t.Fatalf("fetch picture list: %v", err)
	}
	refs := ParseVolumePage(pics, 1, 1)
	t.Logf("picture list: %d bytes, %d pages", len(pics), len(refs))
	if len(refs) == 0 {
		t.Fatal("no pages parsed from picture list")
	}
	thumbPath := "/enp/image/2/" + refs[0].name + ".jpg"
	t.Logf("first page: n=%d name=%q thumb=%q", refs[0].n, refs[0].name, thumbPath)

	// 3. A thumbnail must come back as a real JPEG.
	thumb, err := c.get(thumbPath)
	if err != nil {
		t.Fatalf("fetch thumbnail %s: %v", thumbPath, err)
	}
	t.Logf("thumbnail: %d bytes, magic=%q", len(thumb), thumb[:min(4, len(thumb))])
	if len(thumb) < 500 || thumb[0] != 0xFF || thumb[1] != 0xD8 {
		t.Fatalf("thumbnail is not a JPEG: %d bytes", len(thumb))
	}
}
