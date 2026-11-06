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
	if mode := os.Getenv("PT_FAKE_CHROME"); mode != "" {
		fakeChrome(mode) // 這個測試執行檔被當成 Chrome 啟動（screenshot_test.go）
		os.Exit(0)
	}
	interactive = func() bool { return false }
	ciEnv = func() bool { return false } // GitHub Actions 會設定 CI；需要時各測試自己打開
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
	sm := fakeSkillsManifest()
	origSkills := loadSkills
	loadSkills = func() (ccgsManifest, error) { return sm, nil }
	t.Cleanup(func() { loadSkills = origSkills })
	skillsArchive := skillsTarball(t, nil)
	skillsSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write(skillsArchive) }))
	defer skillsSrv.Close()
	t.Setenv("PT_SKILLS_URL", skillsSrv.URL)
	dir := t.TempDir()
	t.Chdir(dir)

	// 順序：名稱 → 畫面 → 多人？→ 伺服器種類 → 伺服器語言 → 方向 → 內容邊界 → AI 模式 → 確認
	// 名稱不合格會重問；AI 模式亂打會重問，預設標準（加入完整的遊戲開發 skill，不加 Studios）
	typeIn(t, "My Game", "wiz-godot", "2", "", "", "", "2", "", "9", "", "")
	must(t, cmdNew(nil))
	p := readProject(t, filepath.Join(dir, "wiz-godot"))
	if p.Client != "godot" || p.Server != "go" || p.AI != "" || p.Godot != "game" || p.Orientation != "landscape" || p.Skills != "full" ||
		p.AIMode != "standard" || p.gameServerMode() != "playtogether-managed" || !p.multiplayerEnabled() || projectPolicy(t, p).Content.Profile != "general" {
		t.Fatalf("選 Godot＋標準模式：%+v", p)
	}
	if _, err := os.Stat(filepath.Join(dir, "wiz-godot", ".agents/skills/godot-audio/SKILL.md")); err != nil {
		t.Fatal("標準模式應加入 Godot 的遊戲開發 skill")
	}
	if _, err := os.Stat(filepath.Join(dir, "wiz-godot", ".claude/rules/playtogether.md")); err == nil {
		t.Fatal("標準模式不該加入 Claude Code Game Studios")
	}

	// 參數指定的項目不再詢問：問多人、伺服器種類、方向、內容邊界（3＝Strong）、AI 模式（3＝輕量）
	typeIn(t, "", "", "", "3", "3", "y")
	must(t, cmdNew([]string{"wiz-plain", "-client", "godot", "-server", "node"}))
	if p := readProject(t, filepath.Join(dir, "wiz-plain")); p.Server != "node" || p.AI != "" || p.Godot != "game" || p.Orientation != "portrait" || p.Skills != "" || p.AIMode != "lightweight" ||
		projectPolicy(t, p).Content != presetProfile("strong") {
		t.Fatalf("選輕量模式、Strong：%+v", p)
	}

	// 工作室（只限 Godot）：加入 Studios 與完整 skill
	typeIn(t, "wiz-studio", "2", "", "", "", "", "", "2", "")
	must(t, cmdNew(nil))
	if p := readProject(t, filepath.Join(dir, "wiz-studio")); p.AI != "studios" || p.Skills != "full" || p.AIMode != "studio" || p.Godot != "." {
		t.Fatalf("選工作室模式：%+v", p)
	}
	// 自訂：問 skill 範圍與是否加入 Studios
	typeIn(t, "wiz-custom", "2", "", "", "", "", "", "4", "2", "y", "")
	must(t, cmdNew(nil))
	if p := readProject(t, filepath.Join(dir, "wiz-custom")); p.AI != "studios" || p.Skills != "core" || p.AIMode != "custom" {
		t.Fatalf("選自訂：%+v", p)
	}
	// 網頁專案沒有工作室選項：輸入 studio 會重問；選單是 標準、輕量、自訂
	typeIn(t, "wiz-webstd", "", "", "", "", "", "", "studio", "1", "")
	must(t, cmdNew(nil))
	if p := readProject(t, filepath.Join(dir, "wiz-webstd")); p.Client != "web" || p.AI != "" || p.AIMode != "standard" {
		t.Fatalf("網頁專案不能選工作室：%+v", p)
	}
	// 多人＋自己的遊戲伺服器
	typeIn(t, "wiz-ext", "", "", "2", "", "", "", "2", "")
	must(t, cmdNew(nil))
	if p := readProject(t, filepath.Join(dir, "wiz-ext")); p.gameServerMode() != "external" || !p.multiplayerEnabled() || p.AIMode != "lightweight" {
		t.Fatalf("選自己的遊戲伺服器：%+v", p)
	}

	// 單人遊戲：不需要遊戲伺服器；最後選不建立就不留下資料夾
	typeIn(t, "wiz-web", "", "2", "2", "3", "", "2", "n")
	must(t, cmdNew(nil))
	if _, err := os.Stat(filepath.Join(dir, "wiz-web")); err == nil {
		t.Fatal("選擇不建立時不能留下資料夾")
	}
	// 內容邊界選 Custom：暴力選 moderate、血腥選 mild，其他按 Enter（從 general 的值開始）
	typeIn(t, "wiz-solo", "", "2", "", "", "4", "3", "2", "", "", "", "", "", "2", "")
	must(t, cmdNew(nil))
	p = readProject(t, filepath.Join(dir, "wiz-solo"))
	if p.gameServerMode() != "none" || p.multiplayerEnabled() {
		t.Fatalf("單人遊戲應該是 none：%+v", p)
	}
	if r := projectPolicy(t, p).Content; r.Profile != "custom" || r.Violence != "moderate" || r.Blood != "mild" || r.Horror != "mild" || r.Alcohol != "none" {
		t.Fatalf("自訂內容邊界：%+v", r)
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

	// -y 不詢問，用預設值（輕量、不下載任何東西）
	typeIn(t)
	must(t, cmdNew([]string{"wiz-yes", "-y"}))
	if p := readProject(t, filepath.Join(dir, "wiz-yes")); p.Client != "web" || p.Server != "go" || p.Orientation != "portrait" || p.AIMode != "lightweight" || p.Skills != "" ||
		projectPolicy(t, p).Content.Profile != "general" {
		t.Fatalf("-y 應用預設值（內容邊界和精靈的預設相同，general）：%+v", p)
	}
	// 只指定 -ai studios（舊參數＝自訂）：一定是 Godot，不再問 AI 模式
	typeIn(t, "", "", "", "", "", "")
	must(t, cmdNew([]string{"wiz-ai", "-ai", "studios"}))
	if p := readProject(t, filepath.Join(dir, "wiz-ai")); p.Client != "godot" || p.AI != "studios" || p.AIMode != "custom" || p.Skills != "" {
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
	if err := cmdNew([]string{"ok-name", "-orientation", "sideways"}); err == nil || !strings.Contains(err.Error(), "-orientation") {
		t.Fatalf("不合格的方向應拒絕：%v", err)
	}
}
