package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// 舊範本（server/session.go）的專案要提醒：pt sdk update 更新不到平台接口。
func TestLegacyServerHint(t *testing.T) {
	root := newProject(t, "web-go")
	t.Chdir(root)
	p, err := findProject()
	if err != nil || p == nil {
		t.Fatal(err)
	}
	if h := legacyServerHint(p); h != "" {
		t.Fatalf("新範本不應提醒：%s", h)
	}
	must(t, os.WriteFile(filepath.Join(root, "server", "session.go"), []byte("package main\n"), 0o644))
	if h := legacyServerHint(p); !strings.Contains(h, "session.go") {
		t.Fatalf("舊範本沒有提醒：%q", h)
	}
}
