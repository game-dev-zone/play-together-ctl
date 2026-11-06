package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// pt init：既有專案只新增玩聚需要的設定，不覆寫既有檔案；之後 pt check、pt register 用得到；SDK 由開發者自己管理。
func TestInitExistingProject(t *testing.T) {
	requireEmbeddedSDK(t)
	dir := filepath.Join(t.TempDir(), "My Old_Game")
	must(t, os.MkdirAll(filepath.Join(dir, "dist"), 0o755))
	must(t, os.WriteFile(filepath.Join(dir, "dist", "index.html"), []byte(`<!doctype html><script src="playtogether-sdk.iife.js"></script>`), 0o644))
	must(t, os.WriteFile(filepath.Join(dir, "dist", "playtogether-sdk.iife.js"), []byte("// sdk"), 0o644))
	must(t, os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte("我的專案說明\n"), 0o644))
	must(t, os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("node_modules"), 0o644))
	must(t, os.MkdirAll(filepath.Join(dir, "store"), 0o755))
	must(t, os.WriteFile(filepath.Join(dir, "store", "README.md"), []byte("自己的\n"), 0o644))
	t.Chdir(dir)

	must(t, cmdInit([]string{"-y", "-game-server", "none", "-platform", "https://pt.example"}))
	p, err := findProject()
	if err != nil || p == nil {
		t.Fatal(err)
	}
	if p.Name != "my-old-game" || p.Web != "dist" || p.Server != "none" || !p.existing() || p.gameServerMode() != "none" || p.Platform != "https://pt.example" || p.SDKVersion != "" {
		t.Fatalf("playtogether.json：%+v", p)
	}
	for _, f := range []string{"docs/content/CONTENT_PROFILE.md", "docs/content/GAMEPLAY_ECONOMY.md", "store/listing.json"} {
		if _, err := os.Stat(filepath.Join(dir, f)); err != nil {
			t.Errorf("沒有產生 %s", f)
		}
	}
	if raw, _ := os.ReadFile(filepath.Join(dir, "CLAUDE.md")); string(raw) != "我的專案說明\n" {
		t.Fatal("既有的 CLAUDE.md 被改了")
	}
	if raw, _ := os.ReadFile(filepath.Join(dir, "store", "README.md")); string(raw) != "自己的\n" {
		t.Fatal("既有的 store/README.md 被覆寫")
	}
	if raw, _ := os.ReadFile(filepath.Join(dir, ".gitignore")); !strings.Contains(string(raw), "node_modules\n") || !strings.Contains(string(raw), "\nkeys/\n") {
		t.Fatalf(".gitignore：%q", raw)
	}
	for _, f := range []string{"server", "web", "AGENTS.md", "Makefile"} {
		if _, err := os.Stat(filepath.Join(dir, f)); err == nil {
			t.Errorf("既有專案不應產生 %s", f)
		}
	}

	// 宣告與檔案檢查可以通過（store 還沒填只是說明）
	if err := cmdCheck([]string{"-local"}); err != nil {
		t.Fatalf("pt check -local：%v", err)
	}
	// SDK 由開發者自己管理
	if err := cmdSDK([]string{"update"}); err == nil || !strings.Contains(err.Error(), "pt sdk export") {
		t.Fatalf("pt sdk update：%v", err)
	}
	if s := sdkStatus(p, false); !strings.Contains(s, "pt init") {
		t.Fatalf("pt sdk status：%s", s)
	}
	// 再執行一次：不重複接入
	if err := cmdInit([]string{"-y"}); err == nil || !strings.Contains(err.Error(), "已經有 playtogether.json") {
		t.Fatalf("重複執行：%v", err)
	}
	// 子資料夾也不能再接一次
	must(t, os.MkdirAll(filepath.Join(dir, "sub"), 0o755))
	t.Chdir(filepath.Join(dir, "sub"))
	if err := cmdInit([]string{"-y"}); err == nil || !strings.Contains(err.Error(), "上層資料夾") {
		t.Fatalf("子資料夾：%v", err)
	}
}

func TestInitErrors(t *testing.T) {
	requireEmbeddedSDK(t)
	dir := filepath.Join(t.TempDir(), "nothing-here")
	must(t, os.MkdirAll(dir, 0o755))
	t.Chdir(dir)
	if err := cmdInit([]string{"-y"}); err == nil || !strings.Contains(err.Error(), "-web") {
		t.Fatalf("找不到 index.html：%v", err)
	}
	must(t, os.WriteFile(filepath.Join(dir, "index.html"), []byte("<!doctype html>"), 0o644))
	if err := cmdInit([]string{"-y", "-game-server", "playtogether-managed"}); err == nil || !strings.Contains(err.Error(), "pt new") {
		t.Fatalf("managed：%v", err)
	}
	if err := cmdInit([]string{"-y", "-name", "Bad Name"}); err == nil {
		t.Fatal("代號格式錯誤要擋")
	}
	for in, want := range map[string]string{"My Old_Game": "my-old-game", "2048-clone": "clone", "bigwind": "bigwind", "大風吹": ""} {
		if got := initName(in); got != want {
			t.Errorf("initName(%q) = %q, want %q", in, got, want)
		}
	}
}
