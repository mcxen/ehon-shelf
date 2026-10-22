package main

import (
	"errors"
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

	index := make(map[int]Item, len(items))
	for _, it := range items {
		index[it.ID] = it
	}

	start := time.Now()
	var done, skipped, downStreak int
	var failedIDs []int
	for i, it := range items {
		if len(want) > 0 && !want[it.ID] {
			continue
		}
		dir := filepath.Join(dst, strconv.Itoa(it.ID))
		before := dirSize(dir)

		err := c.crawlItem(it)
		after := dirSize(dir)

		if errors.Is(err, errServerDown) {
			// The site is down as a whole. Every remaining work would fail the
			// same way, so stop instead of grinding through the catalogue for
			// hours: the run resumes where it left off when the site is back.
			downStreak++
			if downStreak >= 3 {
				log.Printf("连续 %d 次 5xx，站点已宕机，停止。已下的作品都在 %s，稍后重跑会跳过它们。", downStreak, dst)
				break
			}
		} else {
			downStreak = 0
		}

		switch {
		case err != nil:
			failedIDs = append(failedIDs, it.ID)
			log.Printf("[%d/%d] FAIL id=%d %q: %v", i+1, len(items), it.ID, it.Title, err)
		case after == 0:
			failedIDs = append(failedIDs, it.ID)
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

	// A work that gave up under load usually succeeds on a quieter second
	// pass: the failures are scattered bans, not broken records. Retry them
	// before declaring the run incomplete.
	if len(failedIDs) > 0 {
		log.Printf("--- retrying %d failed works ---", len(failedIDs))
		var stillFailing []int
		for n, id := range failedIDs {
			it, ok := index[id]
			if !ok {
				continue
			}
			dir := filepath.Join(dst, strconv.Itoa(id))
			before := dirSize(dir)
			err := c.crawlItem(it)
			after := dirSize(dir)
			if err != nil || after <= before {
				stillFailing = append(stillFailing, id)
				log.Printf("[retry %d/%d] FAIL id=%d %q: %v", n+1, len(failedIDs), id, it.Title, err)
			} else {
				done++
				log.Printf("[retry %d/%d] ok   id=%d %q %.1f MB", n+1, len(failedIDs), id, it.Title, float64(after-before)/1e6)
			}
		}
		failedIDs = stillFailing
	}

	log.Printf("done: %d downloaded, %d already present, %d still failing in %v",
		done, skipped, len(failedIDs), time.Since(start).Round(time.Second))
	if len(failedIDs) > 0 {
		log.Printf("failed ids: %v", failedIDs)
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
