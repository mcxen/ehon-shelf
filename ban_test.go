package main

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// A ban clears on its own but only after minutes, so a long crawl has to wait
// and carry on rather than abort the work. These tests drive the real get()
// against a server that refuses until told otherwise.
//
// Note: get() fetches /robots.txt first, so every fake server here answers
// that normally and only refuses the page paths. Counting raw hits would
// otherwise count the robots request as the first "page" request.

func newFakeSite(pageHandler func(n int32, w http.ResponseWriter)) (*httptest.Server, *int32) {
	var pageHits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/robots.txt" {
			w.Header().Set("Content-Type", "text/plain")
			_, _ = w.Write([]byte("User-agent: *\nAllow: /\n"))
			return
		}
		pageHandler(atomic.AddInt32(&pageHits, 1), w)
	}))
	return srv, &pageHits
}

func TestGetWaitsOutBanThenSucceeds(t *testing.T) {
	if testing.Short() {
		t.Skip("waits for a ban window")
	}
	// Refuse the first page request, serve the second: the shape of a ban
	// whose window has passed.
	srv, pageHits := newFakeSite(func(n int32, w http.ResponseWriter) {
		if n == 1 {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		_, _ = w.Write([]byte("page"))
	})
	defer srv.Close()

	oldBase, oldWait := Base, banWait
	Base = srv.URL
	// Shrink the wait so the test does not literally sleep ten minutes; the
	// code path (wait, retry, succeed) is what matters.
	banWait = 200 * time.Millisecond
	defer func() { Base, banWait = oldBase, oldWait }()

	c := NewCrawler(t.TempDir(), 1)
	body, err := c.get("/enp/Cover/List/1")
	if err != nil {
		t.Fatalf("get after ban: %v", err)
	}
	if string(body) != "page" {
		t.Fatalf("body = %q, want %q", body, "page")
	}
	if n := atomic.LoadInt32(pageHits); n < 2 {
		t.Fatalf("server saw %d page requests, want at least 2 (the retry)", n)
	}
}

func TestGetGivesUpAfterMaxBanRetries(t *testing.T) {
	srv, _ := newFakeSite(func(n int32, w http.ResponseWriter) {
		w.WriteHeader(http.StatusForbidden)
	})
	defer srv.Close()

	oldBase, oldWait := Base, banWait
	Base = srv.URL
	banWait = time.Millisecond
	defer func() { Base, banWait = oldBase, oldWait }()

	c := NewCrawler(t.TempDir(), 1)
	if _, err := c.get("/enp/Cover/List/1"); err == nil {
		t.Fatal("a permanently refused client must eventually give up")
	}
}

// Once a ban is seen the crawl slows down for the rest of the run, otherwise
// the very next page trips the limiter again.
func TestBanSlowsTheCrawlerForTheRestOfTheRun(t *testing.T) {
	srv, _ := newFakeSite(func(n int32, w http.ResponseWriter) {
		if n == 1 {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		_, _ = w.Write([]byte("ok"))
	})
	defer srv.Close()

	oldBase, oldWait := Base, banWait
	Base = srv.URL
	banWait = time.Millisecond
	defer func() { Base, banWait = oldBase, oldWait }()

	c := NewCrawler(t.TempDir(), 1)
	// Start below the safe pace to prove the ban raises it.
	c.Delay = 500 * time.Millisecond

	if _, err := c.get("/enp/Cover/List/1"); err != nil {
		t.Fatalf("get: %v", err)
	}
	if c.Delay < safeDelay {
		t.Fatalf("delay after ban = %v, want at least %v", c.Delay, safeDelay)
	}
}

// A pause during the ban wait must not be swallowed: the wait is long enough
// that the user has to be able to stop it.
func TestBanWaitIsInterruptibleByPause(t *testing.T) {
	srv, _ := newFakeSite(func(n int32, w http.ResponseWriter) {
		w.WriteHeader(http.StatusForbidden)
	})
	defer srv.Close()

	oldBase, oldWait := Base, banWait
	Base = srv.URL
	banWait = 30 * time.Second
	defer func() { Base, banWait = oldBase, oldWait }()

	c := NewCrawler(t.TempDir(), 1)
	// Pause() only arms when a run is in flight (that is what the UI does), so
	// mark the crawler as running the way Start() would.
	c.mu.Lock()
	c.running = true
	c.pause = true
	c.mu.Unlock()

	start := time.Now()
	_, err := c.get("/enp/Cover/List/1")
	elapsed := time.Since(start)

	if err != errPaused {
		t.Fatalf("err = %v, want errPaused", err)
	}
	if elapsed > 5*time.Second {
		t.Fatalf("pause took %v to interrupt the ban wait", elapsed)
	}
}
