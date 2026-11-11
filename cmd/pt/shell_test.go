package main

import (
	"strings"
	"testing"
)

func TestShellNormalize(t *testing.T) {
	for _, c := range []struct {
		in           shellConfig
		style, accnt string
	}{
		{shellConfig{}, "dark", ""},
		{shellConfig{Style: "light"}, "light", ""},
		{shellConfig{Style: "dark", Accent: "#FFB020"}, "dark", "#ffb020"},
		{shellConfig{Style: "light", Accent: "#1d4ed8"}, "light", "#1d4ed8"},
	} {
		s, a, err := c.in.normalize()
		if err != nil || s != c.style || a != c.accnt {
			t.Errorf("%+v → %q %q %v", c.in, s, a, err)
		}
	}
	for _, in := range []shellConfig{
		{Style: "neon"}, {Style: "Dark"}, {Accent: "red"}, {Accent: "#fff"},
		{Style: "dark", Accent: "#1b2333"}, {Style: "light", Accent: "#fff8e0"}, {Style: "minimal", Accent: "#202a38"},
	} {
		if _, _, err := in.normalize(); err == nil {
			t.Errorf("%+v 應該被拒絕", in)
		}
	}
}

func TestCheckShell(t *testing.T) {
	var got []string
	add := func(level, format string, a ...any) { got = append(got, level) }
	checkShell(&project{Shell: &shellConfig{Style: "light", Accent: "#ffffff"}}, add)
	if len(got) != 1 || got[0] != levelSchema {
		t.Fatalf("對比不足應是宣告錯誤：%v", got)
	}
	got = nil
	checkShell(&project{Shell: &shellConfig{Style: "minimal"}}, add)
	checkShell(&project{}, add)
	if len(got) != 0 {
		t.Fatalf("正確或沒有設定不應報錯：%v", got)
	}
	if b, ok := shellBody(&project{Shell: &shellConfig{Style: "dark", Accent: "#FFB020"}}); !ok || b["style"] != "dark" || b["accent"] != "#ffb020" {
		t.Fatalf("上傳內容：%v %v", b, ok)
	}
	if _, ok := shellBody(&project{}); ok {
		t.Fatal("沒有設定時不送 shell")
	}
	if !strings.Contains(func() string { _, _, e := (shellConfig{Style: "light", Accent: "#eeeeee"}).normalize(); return e.Error() }(), "3:1") {
		t.Fatal("錯誤訊息要說明對比門檻")
	}
}
