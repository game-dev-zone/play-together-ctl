package main

import (
	"bufio"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// 測試預設不是終端機：沒指定的項目用預設值，不詢問
func TestMain(m *testing.M) {
	interactive = func() bool { return false }
	os.Exit(m.Run())
}

// typeIn 模擬開發者在終端機輸入
func typeIn(t *testing.T, lines ...string) {
	t.Helper()
	interactive = func() bool { return true }
	promptIn = bufio.NewReader(strings.NewReader(strings.Join(lines, "\n") + "\n"))
	t.Cleanup(func() { interactive = func() bool { return false } })
}

func readProject(t *testing.T, dir string) project {
	t.Helper()
	var p project
	b, err := os.ReadFile(filepath.Join(dir, projectFile))
	must(t, err)
	must(t, json.Unmarshal(b, &p))
	return p
}

func TestNewWizard(t *testing.T) {
	requireEmbeddedSDK(t)
	m := fakeManifest()
	orig := loadCCGS
	loadCCGS = func() (ccgsManifest, error) { return m, nil }
	t.Cleanup(func() { loadCCGS = orig })
	archive := tarball(t, goodEntries(nil)...)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write(archive) }))
	defer srv.Close()
	t.Setenv("PT_CCGS_URL", srv.URL)
	dir := t.TempDir()
	t.Chdir(dir)

	// 名稱不合格會重問；AI 範本沒有預設，一定要選（亂打會重問）
	typeIn(t, "My Game", "wiz-studio", "2", "", "", "9", "2", "")
	must(t, cmdNew(nil))
	p := readProject(t, filepath.Join(dir, "wiz-studio"))
	if p.Client != "godot" || p.Server != "go" || p.AI != "studios" || p.Godot != "." {
		t.Fatalf("選 Godot＋Claude Code Game Studios：%+v", p)
	}
	if _, err := os.Stat(filepath.Join(dir, "wiz-studio", ".claude/rules/playtogether.md")); err != nil {
		t.Fatal("應加入 Claude Code Game Studios")
	}

	// 參數指定的項目不再詢問：只問 AI 範本
	typeIn(t, "1", "y")
	must(t, cmdNew([]string{"wiz-plain", "-client", "godot", "-server", "node"}))
	if p := readProject(t, filepath.Join(dir, "wiz-plain")); p.Server != "node" || p.AI != "" || p.Godot != "game" {
		t.Fatalf("選一般專案：%+v", p)
	}

	// 網頁專案不問 AI 範本；最後選不建立就不留下資料夾
	typeIn(t, "wiz-web", "", "2", "n")
	must(t, cmdNew(nil))
	if _, err := os.Stat(filepath.Join(dir, "wiz-web")); err == nil {
		t.Fatal("選擇不建立時不能留下資料夾")
	}
	// 輸入中斷（Ctrl-D）視為取消
	typeIn(t, "wiz-eof", "2")
	promptIn = bufio.NewReader(strings.NewReader("wiz-eof\n2\n"))
	if err := cmdNew(nil); err == nil {
		t.Fatal("取消時要回傳錯誤（腳本才知道沒有建立）")
	}
	if _, err := os.Stat(filepath.Join(dir, "wiz-eof")); err == nil {
		t.Fatal("取消時不能留下資料夾")
	}

	// -y 不詢問，用預設值
	typeIn(t)
	must(t, cmdNew([]string{"wiz-yes", "-y"}))
	if p := readProject(t, filepath.Join(dir, "wiz-yes")); p.Client != "web" || p.Server != "go" {
		t.Fatalf("-y 應用預設值：%+v", p)
	}
	// 只指定 -ai studios：一定是 Godot
	typeIn(t, "", "")
	must(t, cmdNew([]string{"wiz-ai", "-ai", "studios"}))
	if p := readProject(t, filepath.Join(dir, "wiz-ai")); p.Client != "godot" || p.AI != "studios" {
		t.Fatalf("-ai studios 應建立 Godot 專案：%+v", p)
	}
}

func TestNewNonInteractive(t *testing.T) {
	requireEmbeddedSDK(t)
	t.Chdir(t.TempDir())
	if err := cmdNew(nil); err == nil || !strings.Contains(err.Error(), "用法") {
		t.Fatalf("不是終端機又沒有名稱應顯示用法：%v", err)
	}
	if err := cmdNew([]string{"Bad Name"}); err == nil {
		t.Fatal("不合格的名稱應拒絕")
	}
}
