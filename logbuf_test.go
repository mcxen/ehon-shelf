package main

import (
	"log"
	"strings"
	"sync"
	"testing"

	"github.com/egoist/mygo/ui"
)

// The log panel is only useful if every crawler log.Printf lands in the
// buffer, keeps order, and stays bounded. These tests pin that contract,
// plus the view wiring that makes the buffer visible.

func TestLogBufferCapturesStandardLogger(t *testing.T) {
	buf := newLogBuffer(100)
	restore := installLogCapture(buf)
	defer restore()

	log.Printf("HTTP %d %s", 403, "/enp/")
	log.Printf("HTTP %d %s", 200, "/enp/")

	entries, _ := buf.Snapshot()
	if len(entries) != 2 {
		t.Fatalf("want 2 entries, got %d", len(entries))
	}
	if !strings.Contains(entries[0].Line, "HTTP 403") {
		t.Errorf("first entry = %q, want it to mention HTTP 403", entries[0].Line)
	}
	if !strings.Contains(entries[1].Line, "HTTP 200") {
		t.Errorf("second entry = %q, want it to mention HTTP 200", entries[1].Line)
	}
	if entries[0].When.IsZero() {
		t.Error("entries should carry a timestamp")
	}
}

func TestLogBufferTrimsTrailingNewline(t *testing.T) {
	buf := newLogBuffer(10)
	if _, err := buf.Write([]byte("line one\n")); err != nil {
		t.Fatal(err)
	}
	if _, err := buf.Write([]byte("line two\r\n")); err != nil {
		t.Fatal(err)
	}
	entries, _ := buf.Snapshot()
	if len(entries) != 2 {
		t.Fatalf("want 2 entries, got %d", len(entries))
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Line, "\n") || strings.HasSuffix(e.Line, "\r") {
			t.Errorf("entry %q still has a trailing newline", e.Line)
		}
	}
}

func TestLogBufferIsBounded(t *testing.T) {
	const max = 5
	buf := newLogBuffer(max)
	for i := 0; i < 50; i++ {
		buf.Append("line %d", i)
	}
	entries, _ := buf.Snapshot()
	if len(entries) != max {
		t.Fatalf("want %d entries, got %d", max, len(entries))
	}
	if !strings.Contains(entries[0].Line, "line 45") {
		t.Errorf("oldest retained entry = %q, want line 45", entries[0].Line)
	}
	if !strings.Contains(entries[max-1].Line, "line 49") {
		t.Errorf("newest entry = %q, want line 49", entries[max-1].Line)
	}
}

func TestLogBufferVersionBumpsAndClear(t *testing.T) {
	buf := newLogBuffer(10)
	_, v0 := buf.Snapshot()
	buf.Append("a")
	_, v1 := buf.Snapshot()
	buf.Append("b")
	_, v2 := buf.Snapshot()
	buf.Clear()
	_, v3 := buf.Snapshot()

	if !(v0 < v1 && v1 < v2 && v2 < v3) {
		t.Fatalf("version did not advance: %d %d %d %d", v0, v1, v2, v3)
	}
	if buf.count() != 0 {
		t.Errorf("Clear left %d entries", buf.count())
	}
}

func TestLogBufferConcurrentWrites(t *testing.T) {
	buf := newLogBuffer(1000)
	var wg sync.WaitGroup
	for w := 0; w < 8; w++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for i := 0; i < 50; i++ {
				buf.Append("worker %d item %d", id, i)
			}
		}(w)
	}
	wg.Wait()
	entries, _ := buf.Snapshot()
	if len(entries) != 400 {
		t.Fatalf("want 400 entries, got %d", len(entries))
	}
}

func TestInstallLogCaptureRestores(t *testing.T) {
	buf := newLogBuffer(10)
	prev := log.Writer()
	restore := installLogCapture(buf)
	if log.Writer() == prev {
		t.Fatal("log writer was not replaced")
	}
	restore()
	if log.Writer() != prev {
		t.Fatal("log writer was not restored")
	}
}

func TestLogColorClassifiesLines(t *testing.T) {
	// logColor only needs a Theme for the muted fallback; build one via a
	// throwaway render rather than reaching into package internals.
	var tm *ui.Theme
	tt := ui.NewTester(func(c *ui.Context) {
		tm = c.Theme()
		ui.Text(c, "x")
	}, 200, 100)
	tt.Frame()

	errColor := ui.RGBA(220, 80, 80, 255)
	okColor := ui.RGBA(70, 170, 90, 255)

	cases := []struct {
		line string
		want ui.Color
	}{
		{"HTTP 403 /enp/", errColor},
		{"HTTP 429 /enp/", errColor},
		{"error dial tcp", errColor},
		{"放弃 /enp/", errColor},
		{"HTTP 200 /enp/", okColor},
		{"下载完成", okColor},
		{"正在读取书库…", tm.TextMuted},
	}
	for _, tc := range cases {
		if got := logColor(tm, tc.line); got != tc.want {
			t.Errorf("logColor(%q) = %+v, want %+v", tc.line, got, tc.want)
		}
	}
}

// TestLogsViewRenders checks the panel actually shows buffered lines and the
// controls a user needs when a download is being refused.
func TestLogsViewRenders(t *testing.T) {
	a := fixture(t)
	a.view = viewLogs
	a.logs.Clear()
	a.logs.Append("HTTP 403 /enp/1-50/")
	a.logs.Append("重试中")

	tt := ui.NewTester(a.viewRoot, 1000, 700)
	tt.Frame()

	for _, want := range []string{"日志", "2 条", "清空", "复制全部", "HTTP 403", "重试中"} {
		if !tt.HasText(want) {
			t.Errorf("logs view missing %q; texts=%q", want, tt.Texts())
		}
	}
}

// TestLogsViewEmptyState makes sure an empty buffer explains itself instead of
// rendering a blank pane.
func TestLogsViewEmptyState(t *testing.T) {
	a := fixture(t)
	a.view = viewLogs
	a.logs.Clear()

	tt := ui.NewTester(a.viewRoot, 1000, 700)
	tt.Frame()
	if !tt.HasText("还没有日志") {
		t.Errorf("expected empty-state hint; texts=%q", tt.Texts())
	}
}

// TestClearButtonEmptiesBuffer drives the actual control, not just the method.
func TestClearButtonEmptiesBuffer(t *testing.T) {
	a := fixture(t)
	a.view = viewLogs
	a.logs.Clear()
	a.logs.Append("something happened")

	tt := ui.NewTester(a.viewRoot, 1000, 700)
	tt.Frame()
	if err := tt.Click("清空"); err != nil {
		t.Fatalf("click 清空: %v", err)
	}
	tt.Frame()
	if n := a.logs.count(); n != 0 {
		t.Errorf("after clicking 清空 the buffer still has %d entries", n)
	}
	if !tt.HasText("还没有日志") {
		t.Errorf("expected empty state after clear; texts=%q", tt.Texts())
	}
}

// TestShelfHasLogsEntry ensures the panel is reachable from the shelf toolbar.
func TestShelfHasLogsEntry(t *testing.T) {
	a := fixture(t)
	a.view = viewShelf
	a.logs.Clear()

	tt := ui.NewTester(a.viewRoot, 1000, 700)
	tt.Frame()
	if !tt.HasText("日志") {
		t.Errorf("shelf toolbar missing 日志 button; texts=%q", tt.Texts())
	}
	if err := tt.Click("日志"); err != nil {
		t.Fatalf("click 日志: %v", err)
	}
	if a.view != viewLogs {
		t.Errorf("view = %d, want viewLogs", a.view)
	}
}
