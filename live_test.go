package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestLive downloads a few pages of one work from the real site, pauses, and
// checks the queue survives a restart. Run with EHONSHELF_LIVE=1.
func TestLive(t *testing.T) {
	if os.Getenv("EHONSHELF_LIVE") == "" {
		t.Skip("set EHONSHELF_LIVE=1")
	}
	dir := t.TempDir()
	c := NewCrawler(dir, 1.2)
	items, err := c.ListItems()
	if err != nil || len(items) < 400 {
		t.Fatalf("list: %d %v", len(items), err)
	}
	var it Item
	for _, i := range items {
		if i.ID == 431 {
			it = i
		}
	}
	c.Start([]Item{it})
	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		if s := c.Snapshot(); s.ItemDone >= 2 {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	c.Pause()
	for c.Running() {
		time.Sleep(100 * time.Millisecond)
	}
	s := c.Snapshot()
	t.Logf("status %+v", s)
	if s.State != "paused" || s.ItemTotal == 0 {
		t.Fatalf("status %+v", s)
	}
	b, _ := os.ReadFile(filepath.Join(dir, "431", "pages", "0001.jpg"))
	if len(b) < 10000 || b[0] != 0xff || b[1] != 0xd8 {
		t.Fatalf("page 1 is not a full JPEG: %d bytes", len(b))
	}
	c2 := NewCrawler(dir, 1.2)
	if s2 := c2.Snapshot(); s2.QueueLen != 1 || s2.State != "paused" {
		t.Fatalf("queue not restored: %+v", s2)
	}
}
