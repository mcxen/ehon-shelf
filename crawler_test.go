package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// A fake site with the structure of the real one: one work, one volume,
// three pages over two list pages; page 3 has no full-size image.
func fakeSite(t *testing.T, hits *int32) {
	jpg := "\xff\xd8FULL-IMAGE-DATA\xff\xd9"
	mux := http.NewServeMux()
	mux.HandleFunc("/robots.txt", func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, "User-agent: *\nDisallow: /secret\n") })
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(hits, 1)
		if r.Header.Get("User-Agent") != UA {
			t.Errorf("User-Agent %q", r.Header.Get("User-Agent"))
		}
		p := r.URL.Path
		switch {
		case p == "/enp/SearchBooks/BySakuContents":
			fmt.Fprint(w, `<a href="/enp/Cover/List/7">x</a><span class="s">試本</span><span class="ss">しほん</span><span class="sblue">絵師</span>`)
		case p == "/enp/Cover/List/7":
			fmt.Fprint(w, `<a href="/enp/Picture/List/7/1/1">巻1</a>`)
		case p == "/enp/Book/Shoshi/7":
			fmt.Fprint(w, `<td>絵師名</td><td>絵師</td><td>板行年</td><td>文政</td>`)
		case p == "/enp/Picture/ListContents/7/1/1":
			fmt.Fprint(w, `<a href="/enp/Picture/View/7/1/1"><img src="/enp/image/2/007_A_01.jpg"></a><a href="/enp/Picture/View/7/1/2"><img src="/enp/image/2/007_A_02.jpg"></a><a href="/enp/Picture/List/7/1/2">次</a>`)
		case p == "/enp/Picture/ListContents/7/1/2":
			fmt.Fprint(w, `<a href="/enp/Picture/View/7/1/3"><img src="/enp/image/2/007_A_03.jpg"></a>`)
		case p == "/enp/image/1/007_A_03.pdf":
			fmt.Fprint(w, "%PDF no image")
		case strings.HasPrefix(p, "/enp/image/1/"):
			fmt.Fprint(w, "%PDF /DCTDecode >>\nstream\n"+jpg+"\nendstream")
		case strings.HasPrefix(p, "/enp/image/2/"):
			fmt.Fprint(w, "\xff\xd8THUMB\xff\xd9")
		default:
			http.NotFound(w, r)
		}
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	old := Base
	Base = srv.URL
	t.Cleanup(func() { Base = old })
}

func waitIdle(c *Crawler) {
	for i := 0; i < 200 && c.Running(); i++ {
		time.Sleep(50 * time.Millisecond)
	}
}

func TestCrawlFakeSite(t *testing.T) {
	var hits int32
	fakeSite(t, &hits)
	dir := t.TempDir()
	c := NewCrawler(dir, 1)
	c.Delay = 5 * time.Millisecond // the minimum applies to users; tests run fast
	items, err := c.ListItems()
	if err != nil || len(items) != 1 || items[0].Title != "試本" {
		t.Fatalf("list %v %v", items, err)
	}
	c.Start(items)
	waitIdle(c)
	s := c.Snapshot()
	if s.State != "done" || s.DoneItems != 1 || len(s.Errors) > 0 {
		t.Fatalf("status %+v", s)
	}
	m, err := readMeta(filepath.Join(dir, "7", "meta.json"))
	if err != nil || !m.Complete || m.PageCount != 3 || m.Date != "文政" || !m.Pages[2].Lowres {
		t.Fatalf("meta %+v %v", m, err)
	}
	b, _ := os.ReadFile(filepath.Join(dir, "7", "pages", "0001.jpg"))
	if string(b) != "\xff\xd8FULL-IMAGE-DATA\xff\xd9" {
		t.Errorf("page 1 %q", b)
	}
	if _, err := os.Stat(c.queuePath()); !os.IsNotExist(err) {
		t.Error("queue.json should be gone when the queue is empty")
	}
	// A second run skips what exists: only the work's pages are not fetched again.
	before := atomic.LoadInt32(&hits)
	c.Start(items)
	waitIdle(c)
	if got := atomic.LoadInt32(&hits) - before; got != 0 {
		t.Errorf("re-run made %d requests, want 0", got)
	}
}

func TestQueuePersists(t *testing.T) {
	dir := t.TempDir()
	c := NewCrawler(dir, 1)
	c.queue = []Item{{ID: 1}, {ID: 2}}
	c.saveQueue()
	c2 := NewCrawler(dir, 1)
	if s := c2.Snapshot(); s.QueueLen != 2 || s.State != "paused" {
		t.Fatalf("restored %+v", s)
	}
	if NewCrawler(dir, 0.2).Delay != time.Second {
		t.Error("delay below the 1 s minimum")
	}
}
