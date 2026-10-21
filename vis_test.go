package main

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestSettingsProxyVisibleAtDefaultSize(t *testing.T) {
	a := fixture(t)
	a.view = viewSettings
	for _, sz := range [][2]int{{1000, 700}, {980, 640}, {900, 560}, {860, 500}} {
		tt := ui.NewTester(a.viewRoot, sz[0], sz[1])
		tt.Frame()
		found := false
		for _, x := range tt.Texts() {
			if x == "代理" {
				found = true
			}
		}
		t.Logf("尺寸 %dx%d: 代理输入框可见=%v", sz[0], sz[1], found)
		if !found {
			t.Errorf("尺寸 %dx%d 下看不到代理设置", sz[0], sz[1])
		}
	}
}
