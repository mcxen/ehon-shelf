package main

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// The proxy is the only way past the site's geo-block, so it must be
// findable at every window size the app allows — not just present in the
// view tree. A bare input box with a grey placeholder read as a stray search
// field and users could not find the setting at all, so the row must carry a
// visible text label of its own.
func TestSettingsProxyIsLabelledAndVisible(t *testing.T) {
	a := fixture(t)
	a.view = viewSettings

	for _, sz := range [][2]int{{1000, 700}, {980, 640}, {900, 560}, {860, 500}} {
		tt := ui.NewTester(a.viewRoot, sz[0], sz[1])
		tt.Frame()

		texts := tt.Texts()
		var hasLabel, hasHint bool
		for _, x := range texts {
			if x == proxyLabel {
				hasLabel = true
			}
			if x == "填好后点下面的「保存」" {
				hasHint = true
			}
		}
		if !hasLabel {
			t.Errorf("%dx%d: 看不到代理的文字标签 %q；texts=%q", sz[0], sz[1], proxyLabel, texts)
		}
		if !hasHint {
			t.Errorf("%dx%d: 看不到代理行的说明文字", sz[0], sz[1])
		}
	}
}

// Saving from a running download must still store the proxy: the proxy is
// often filled in precisely because the download is failing with 403.
func TestSaveStoresProxyWhileRunning(t *testing.T) {
	a := fixture(t)
	a.view = viewSettings
	a.proxy = "http://10.0.0.1:8080"
	a.conf.Proxy = ""

	tt := ui.NewTester(a.viewRoot, 1000, 700)
	tt.Frame()
	if err := tt.Click("保存"); err != nil {
		t.Fatalf("click 保存: %v", err)
	}
	if a.conf.Proxy != "http://10.0.0.1:8080" {
		t.Fatalf("conf.Proxy = %q, want it saved", a.conf.Proxy)
	}
	if HttpProxy != "http://10.0.0.1:8080" {
		t.Fatalf("HttpProxy = %q, want it applied immediately", HttpProxy)
	}
	if a.confMsg != "已保存" {
		t.Fatalf("confMsg = %q, want 已保存", a.confMsg)
	}
}
