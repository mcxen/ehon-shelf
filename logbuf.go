package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"time"
)

// logEntry is one captured log line bound for the in-app log panel.
type logEntry struct {
	When time.Time
	Line string
}

// logBuffer is a bounded, concurrency-safe ring of recent log lines.
//
// The crawler already writes progress through log.Printf; redirecting the
// standard logger's writer into this buffer is what makes those messages
// visible in the UI instead of only on stderr (which a GUI launch discards).
type logBuffer struct {
	mu      sync.Mutex
	entries []logEntry
	max     int
	version int // bumped on every append, so the view can redraw on change
}

func newLogBuffer(max int) *logBuffer {
	if max <= 0 {
		max = 500
	}
	return &logBuffer{max: max}
}

// Write implements io.Writer so it can be handed to log.SetOutput.
func (b *logBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	text := string(p)
	// log.Logger already appends a newline; trim it for display.
	for len(text) > 0 && (text[len(text)-1] == '\n' || text[len(text)-1] == '\r') {
		text = text[:len(text)-1]
	}
	if text == "" {
		return len(p), nil
	}
	b.entries = append(b.entries, logEntry{When: time.Now(), Line: text})
	if len(b.entries) > b.max {
		// Drop oldest in chunks to avoid copying on every line.
		drop := len(b.entries) - b.max
		b.entries = append([]logEntry(nil), b.entries[drop:]...)
	}
	b.version++
	return len(p), nil
}

// Snapshot returns a copy of the captured lines and the current version.
func (b *logBuffer) Snapshot() ([]logEntry, int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]logEntry, len(b.entries))
	copy(out, b.entries)
	return out, b.version
}

// Clear removes everything captured so far.
func (b *logBuffer) Clear() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.entries = nil
	b.version++
}

// Append adds a line that did not come from the standard logger.
func (b *logBuffer) Append(format string, args ...any) {
	b.Write([]byte(fmt.Sprintf(format, args...)))
}

// count reports how many lines are buffered (used by the tab badge).
func (b *logBuffer) count() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.entries)
}

// installLogCapture patches the global logger so every log.Printf in the
// program both reaches stderr (when one exists) and lands in buf.
// It returns a restore function for tests.
func installLogCapture(buf *logBuffer) func() {
	prevWriter := log.Writer()
	prevFlags := log.Flags()
	prevPrefix := log.Prefix()

	log.SetFlags(log.Ltime)
	log.SetPrefix("")
	log.SetOutput(io.MultiWriter(prevWriter, buf))

	return func() {
		log.SetOutput(prevWriter)
		log.SetFlags(prevFlags)
		log.SetPrefix(prevPrefix)
	}
}

// verboseToStderr is false in GUI launches where stderr goes nowhere;
// kept as a named value so the intent is explicit.
var _ = os.Stderr
