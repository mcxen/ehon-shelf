package main

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

// TestLiveDownloadFiveWorks downloads five works end to end and reports what
// landed on disk for each, so a silently-empty save is impossible to miss.
// Skipped unless EHONSHELF_LIVE=1 and EHONSHELF_PROXY are set:
//
//	EHONSHELF_LIVE=1 EHONSHELF_PROXY=http://127.0.0.1:7897 \
//	  go test -run TestLiveDownloadFiveWorks -v -timeout 1800s
func TestLiveDownloadFiveWorks(t *testing.T) {
	if os.Getenv("EHONSHELF_LIVE") != "1" {
		t.Skip("set EHONSHELF_LIVE=1 to hit the real site")
	}
	if HttpProxy == "" {
		t.Fatal("EHONSHELF_PROXY must be set")
	}
	old := Base
	Base = "https://lapis.nichibun.ac.jp"
	defer func() { Base = old }()

	const want = 5
	dir := t.TempDir()
	c := NewCrawler(dir, 1.0)

	items, err := c.ListItems()
	if err != nil {
		t.Fatalf("ListItems: %v", err)
	}
	if len(items) < want {
		t.Fatalf("catalogue too small: %d", len(items))
	}

	// Spread the sample across the catalogue instead of taking the first five,
	// so a failure in one region of the list cannot hide behind the others.
	step := len(items) / want
	var picked []Item
	for i := 0; i < want; i++ {
		picked = append(picked, items[i*step])
	}
	for _, it := range picked {
		t.Logf("selected id=%d %q by %q", it.ID, it.Title, it.Artist)
	}

	type result struct {
		Item
		Err      error
		Pages    int
		Jpegs    int
		MetaOK   bool
		CoverOK  bool
		Complete bool
		Bytes    int64
	}

	var results []result
	for _, it := range picked {
		start := time.Now()
		t.Logf("--- START id=%d %q ---", it.ID, it.Title)
		r := result{Item: it}
		// A single banned request legitimately takes banWait (10 min) to
		// resume, so this guard must sit well above banWait * maxBanRetries;
		// a shorter one aborts the run in the middle of a legitimate wait and
		// reports the bug it is meant to catch.
		guard := banWait*time.Duration(maxBanRetries) + 5*time.Minute
		done := make(chan error, 1)
		go func(target Item) { done <- c.crawlItem(target) }(it)
		select {
		case r.Err = <-done:
		case <-time.After(guard):
			r.Err = errTimeout
		}

		// Inspect the work directory the crawler actually writes to.
		workDir := filepath.Join(dir, strconv.Itoa(it.ID))
		_ = filepath.Walk(workDir, func(p string, info os.FileInfo, err error) error {
			if err != nil || info == nil || info.IsDir() {
				return nil
			}
			r.Bytes += info.Size()
			switch filepath.Base(p) {
			case "meta.json":
				r.MetaOK = true
			case "cover.jpg":
				r.CoverOK = true
			}
			if filepath.Ext(p) == ".jpg" && filepath.Base(filepath.Dir(p)) == "pages" {
				r.Jpegs++
			}
			return nil
		})
		if meta, _ := readMeta(filepath.Join(workDir, "meta.json")); meta != nil {
			r.Pages = len(meta.Pages)
			r.Complete = meta.Complete
		}
		results = append(results, r)

		status := "OK"
		if r.Err != nil {
			status = "FAIL: " + r.Err.Error()
		}
		t.Logf("[%s] id=%d title=%q pages=%d jpegs=%d meta=%v cover=%v complete=%v bytes=%d (%.1fs)",
			status, it.ID, it.Title, r.Pages, r.Jpegs, r.MetaOK, r.CoverOK, r.Complete,
			r.Bytes, time.Since(start).Seconds())
	}

	// Aggregate check: every work must have saved real image data.
	failed, lowres, ok := 0, 0, 0
	for _, r := range results {
		switch {
		case r.Err != nil || r.Jpegs == 0 || r.Bytes < 100_000:
			failed++
		case !r.Complete || r.Jpegs < r.Pages:
			lowres++
		default:
			ok++
		}
	}
	t.Logf("summary: %d/%d fully downloaded, %d partial, %d failed", ok, want, lowres, failed)
	if failed > 0 {
		t.Fatalf("%d of %d works failed to download", failed, want)
	}
}

var errTimeout = &timeoutErr{}

type timeoutErr struct{}

func (*timeoutErr) Error() string { return "download timed out" }
