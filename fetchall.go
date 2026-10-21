package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// A headless full download, reusing the app's crawler unchanged: same pacing,
// same ban handling, same skip-what-exists logic. Building with -tags fetchall
// and running it downloads the catalogue to a directory, typically an SMB
// share, and can be interrupted and restarted at any point.
//
//	go build -tags fetchall -o fetchall .
//	./fetchall -out /Volumes/TO3TNS/EhonShelf -proxy http://127.0.0.1:7897
func fetchallMain() {
	log.SetFlags(log.Ltime)

	out := flag.String("out", "", "destination directory (required)")
	proxy := flag.String("proxy", os.Getenv("EHONSHELF_PROXY"), "HTTP proxy")
	delay := flag.Float64("delay", 0, "seconds between requests (0 = the crawler default)")
	only := flag.String("only", "", "comma separated ids (default: the whole catalogue)")
	flag.Parse()

	if *out == "" {
		log.Fatal("-out is required")
	}
	dst, err := filepath.Abs(*out)
	if err != nil {
		log.Fatalf("resolve %s: %v", *out, err)
	}
	if err := os.MkdirAll(dst, 0o755); err != nil {
		log.Fatalf("create %s: %v", dst, err)
	}

	HttpProxy = strings.TrimSpace(*proxy)
	Base = "https://lapis.nichibun.ac.jp"

	want := map[int]bool{}
	for _, s := range strings.Split(*only, ",") {
		if n, err := strconv.Atoi(strings.TrimSpace(s)); err == nil {
			want[n] = true
		}
	}

	c := NewCrawler(dst, *delay)
	if c.Delay < safeDelay {
		c.Delay = safeDelay
	}
	// Page-level progress comes from the crawler's logger. installLogCapture
	// tees it to stderr as well as the buffer, so a long run leaves a trail
	// in the terminal instead of silence.
	restore := installLogCapture(newLogBuffer(200))
	defer restore()
	log.Printf("destination %s, %v between requests, proxy %q", dst, c.Delay, HttpProxy)

	items, err := c.ListItems()
	if err != nil {
		log.Fatalf("read the catalogue: %v", err)
	}
	log.Printf("%d works in the catalogue", len(items))

	start := time.Now()
	var done, skipped, failed int
	for i, it := range items {
		if len(want) > 0 && !want[it.ID] {
			continue
		}
		dir := filepath.Join(dst, strconv.Itoa(it.ID))
		before := dirSize(dir)

		err := c.crawlItem(it)
		after := dirSize(dir)

		switch {
		case err != nil:
			failed++
			log.Printf("[%d/%d] FAIL id=%d %q: %v", i+1, len(items), it.ID, it.Title, err)
		case after == 0:
			failed++
			log.Printf("[%d/%d] EMPTY id=%d %q: nothing was saved", i+1, len(items), it.ID, it.Title)
		case after == before:
			skipped++
			log.Printf("[%d/%d] have id=%d %q", i+1, len(items), it.ID, it.Title)
		default:
			done++
			log.Printf("[%d/%d] ok   id=%d %q %.1f MB (%.0fs elapsed)",
				i+1, len(items), it.ID, it.Title, float64(after-before)/1e6,
				time.Since(start).Seconds())
		}
	}

	log.Printf("done: %d downloaded, %d already present, %d failed in %v",
		done, skipped, failed, time.Since(start).Round(time.Second))
	if failed > 0 {
		os.Exit(1)
	}
	_ = fmt.Sprint()
}

// dirSize sums the bytes under a directory, so a run can tell "already
// present" from "downloaded just now" without a second bookkeeping file.
func dirSize(dir string) int64 {
	var n int64
	_ = filepath.Walk(dir, func(_ string, info os.FileInfo, err error) error {
		if err == nil && info != nil && !info.IsDir() {
			n += info.Size()
		}
		return nil
	})
	return n
}
