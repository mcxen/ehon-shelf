package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestLiveDownloadOneWork downloads a single work end to end — list, meta,
// pages, images — exactly as the app does when the user picks one item.
// Skipped unless  EHONSHELF_LIVE=1 and EHONSHELF_PROXY are set:
//
//	EHONSHELF_LIVE=1 EHONSHELF_PROXY=http://127.0.0.1:7897 \
//	  go test -run TestLiveDownloadOneWork -v -timeout 600s
func TestLiveDownloadOneWork(t *testing.T) {
	if os.Getenv("EHONSHELF_LIVE") != "1" {
		t.Skip("set EHONSHELF_LIVE=1 to hit the real site")
	}
	if HttpProxy == "" {
		t.Fatal("EHONSHELF_PROXY must be set")
	}
	old := Base
	Base = "https://lapis.nichibun.ac.jp"
	defer func() { Base = old }()

	dir := t.TempDir()
	c := NewCrawler(dir, 1.0)

	items, err := c.ListItems()
	if err != nil {
		t.Fatalf("ListItems: %v", err)
	}
	if len(items) < 400 {
		t.Fatalf("catalogue too small: %d", len(items))
	}

	// Pick a small work (page count is unknown up front, so just take the
	// first and cap the work below).
	target := items[0]
	t.Logf("downloading id=%d %q by %q", target.ID, target.Title, target.Artist)

	done := make(chan error, 1)
	go func() {
		done <- c.crawlItem(target)
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("crawlItem: %v", err)
		}
	case <-time.After(8 * time.Minute):
		t.Fatal("crawlItem timed out after 8 minutes")
	}

	// Verify what actually landed on disk.
	workDir := dir
	var files []string
	_ = filepath.Walk(workDir, func(p string, info os.FileInfo, err error) error {
		if err == nil && info != nil && !info.IsDir() {
			files = append(files, p)
		}
		return nil
	})
	t.Logf("files written: %d", len(files))

	var jpegs, metas int
	var total int64
	for _, f := range files {
		st, _ := os.Stat(f)
		if st != nil {
			total += st.Size()
		}
		switch filepath.Ext(f) {
		case ".jpg":
			jpegs++
		case ".json":
			metas++
		}
	}
	t.Logf("jpeg=%d meta=%d total=%.1f KB", jpegs, metas, float64(total)/1024)
	for _, f := range files {
		rel, _ := filepath.Rel(workDir, f)
		st, _ := os.Stat(f)
		t.Logf("  %-40s %8d", rel, st.Size())
	}
	if jpegs == 0 {
		t.Fatal("no page images were written")
	}
	if total < 50_000 {
		t.Fatalf("downloaded only %d bytes", total)
	}
}
