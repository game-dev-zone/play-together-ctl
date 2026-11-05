package main

import (
	"bytes"
	"flag"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// fakeAI 用假的 Studios 與 skill 清單；回傳的 hits 記錄兩個下載點被連了幾次
func fakeAI(t *testing.T) *int {
	t.Helper()
	hits := new(int)
	m, sm := fakeManifest(), fakeSkillsManifest()
	origC, origS := loadCCGS, loadSkills
	loadCCGS = func() (ccgsManifest, error) { return m, nil }
	loadSkills = func() (ccgsManifest, error) { return sm, nil }
	t.Cleanup(func() { loadCCGS, loadSkills = origC, origS })
	studio, skills := tarball(t, goodEntries(nil)...), skillsTarball(t, nil)
	srvC := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { *hits++; _, _ = w.Write(studio) }))
	srvS := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { *hits++; _, _ = w.Write(skills) }))
	t.Cleanup(srvC.Close)
	t.Cleanup(srvS.Close)
	t.Setenv("PT_CCGS_URL", srvC.URL)
	t.Setenv("PT_SKILLS_URL", srvS.URL)
	return hits
}

func readFile(t *testing.T, path ...string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(path...))
	if err != nil {
		t.Fatalf("少了 %s", filepath.Join(path...))
	}
	return string(b)
}

func exists(path ...string) bool { _, err := os.Stat(filepath.Join(path...)); return err == nil }

func TestResolveAIMode(t *testing.T) {
	cases := []struct {
		mode, ai, skills string
		set              []string
		want             [3]string
		bad              bool
	}{
		{"", "", "", nil, [3]string{"", "", ""}, false}, // 還沒決定：精靈問，非互動用 lightweight
		{"lightweight", "", "", []string{"mode"}, [3]string{"lightweight", "none", "none"}, false},
		{"standard", "", "", []string{"mode"}, [3]string{"standard", "none", "full"}, false},
		{"studio", "", "", []string{"mode"}, [3]string{"studio", "studios", "full"}, false},
		{"", "", "core", []string{"skills"}, [3]string{"custom", "", "core"}, false}, // 舊參數＝自訂，結果和以前相同
		{"", "studios", "", []string{"ai"}, [3]string{"custom", "studios", ""}, false},
		{"custom", "studios", "full", []string{"mode", "ai", "skills"}, [3]string{"custom", "studios", "full"}, false},
		{"standard", "", "core", []string{"mode", "skills"}, [3]string{}, true}, // 模式已經決定元件
		{"studio", "none", "", []string{"mode", "ai"}, [3]string{}, true},
		{"heavy", "", "", []string{"mode"}, [3]string{}, true},
	}
	for _, c := range cases {
		set := map[string]bool{}
		for _, s := range c.set {
			set[s] = true
		}
		m, ai, sk, err := resolveAIMode(c.mode, c.ai, c.skills, set)
		if c.bad {
			if err == nil {
				t.Errorf("%+v 應該報錯", c)
			}
			continue
		}
		if err != nil || [3]string{m, ai, sk} != c.want {
			t.Errorf("%+v：得到 %v %v %v %v", c, m, ai, sk, err)
		}
	}
}

func TestNewAIModes(t *testing.T) {
	requireEmbeddedSDK(t)
	fakeAI(t)
	dir := t.TempDir()
	t.Chdir(dir)

	must(t, cmdNew([]string{"m-light", "-mode", "lightweight"}))
	must(t, cmdNew([]string{"m-std-web", "-mode", "standard"}))
	must(t, cmdNew([]string{"m-std-godot", "-mode", "standard", "-client", "godot"}))
	must(t, cmdNew([]string{"m-studio", "-mode", "studio"})) // 沒指定畫面：自動 Godot
	if err := cmdNew([]string{"m-studio-web", "-mode", "studio", "-client", "web"}); err == nil || !strings.Contains(err.Error(), "standard") {
		t.Fatalf("網頁專案不能用工作室模式，要建議 standard：%v", err)
	}
	if err := cmdNew([]string{"m-mix", "-mode", "standard", "-skills", "core"}); err == nil || !strings.Contains(err.Error(), "custom") {
		t.Fatalf("-mode 與舊參數同時給要報錯：%v", err)
	}

	// 1. 輕量：只有平台規則與工作流程契約
	if exists(dir, "m-light", ".claude") || exists(dir, "m-light", ".agents") || exists(dir, "m-light", "docs/ai/CURRENT_TASK.md") {
		t.Fatal("輕量模式不該有 skill、Studios、CURRENT_TASK")
	}
	// 所有模式都有 AI_WORKFLOW 與 PROJECT_CONTEXT
	for _, p := range []string{"m-light", "m-std-web", "m-std-godot", "m-studio"} {
		for _, f := range []string{"docs/ai/AI_WORKFLOW.md", "docs/ai/PROJECT_CONTEXT.md", "PLAYTOGETHER.md", "CLAUDE.md", "AGENTS.md"} {
			if !exists(dir, p, f) {
				t.Errorf("%s 少了 %s", p, f)
			}
		}
	}
	// 2. 標準：通用＋這個引擎的 skill，沒有 Studios
	if !exists(dir, "m-std-web", ".agents/skills/phaser-core/SKILL.md") || exists(dir, "m-std-web", ".agents/skills/godot-audio") ||
		!exists(dir, "m-std-godot", ".claude/skills/godot-audio/SKILL.md") || exists(dir, "m-std-godot", "project.yaml") {
		t.Fatal("標準模式：通用＋引擎 skill，沒有 Studios")
	}
	// 3. 工作室：Studios＋skill＋四個工作流程文件，設定調整都有標記
	root := filepath.Join(dir, "m-studio")
	p := readProject(t, root)
	if p.Client != "godot" || p.AI != "studios" || p.Skills != "full" || p.AIMode != "studio" {
		t.Fatalf("工作室模式：%+v", p)
	}
	for _, f := range []string{"docs/ai/CURRENT_TASK.md", "docs/ai/DECISIONS.md", ".claude/skills/godot-audio/SKILL.md", ".claude/rules/playtogether.md"} {
		if !exists(root, f) {
			t.Errorf("工作室模式少了 %s", f)
		}
	}
	checkStudiosPatched(t, root)

	// 輕量／標準的 AI_WORKFLOW 不提 Studios；工作室有使用規則與 Codex 邊界
	if wf := readFile(t, dir, "m-std-web", "docs/ai/AI_WORKFLOW.md"); strings.Contains(wf, "Game Studios") || !strings.Contains(wf, ".claude/skills/") {
		t.Fatal("標準模式的 AI_WORKFLOW：提到 skill、不提 Studios")
	}
	wf := readFile(t, root, "docs/ai/AI_WORKFLOW.md")
	for _, want := range []string{"Claude Code Game Studios 的使用規則", "由玩聚負責（不使用）", "/team-release", "Codex 不執行", "docs/ai/DECISIONS.md", "minimal"} {
		if !strings.Contains(wf, want) {
			t.Errorf("工作室的 AI_WORKFLOW 少了 %q", want)
		}
	}
}

// checkStudiosPatched：四個 frontmatter 與 project.yaml 的調整都有玩聚標記，其他內容不變
func checkStudiosPatched(t *testing.T, root string) {
	t.Helper()
	for _, s := range []string{"dev-story", "story-done"} {
		c := readFile(t, root, ".claude/skills", s, "SKILL.md")
		if strings.Contains(c, "disable-model-invocation") || !strings.Contains(c, ptMark) {
			t.Errorf("%s 要讓 AI 自動使用並標記：\n%s", s, c)
		}
	}
	for _, s := range []string{"team-release", "release-checklist"} {
		c := readFile(t, root, ".claude/skills", s, "SKILL.md")
		fm := c[:strings.Index(c[4:], "\n---\n")+4]
		if !strings.Contains(fm, "disable-model-invocation: true") || !strings.Contains(fm, ptMark) {
			t.Errorf("%s 要改成只能手動並標記：\n%s", s, c)
		}
		if strings.Count(c, "disable-model-invocation") != 1 {
			t.Errorf("%s 只能改 frontmatter：\n%s", s, c)
		}
	}
	y := readFile(t, root, "project.yaml")
	if !strings.HasPrefix(y, "schema_version: 1\n") || !strings.Contains(y, "rigor: minimal") || !strings.Contains(y, "automation: guided") ||
		!strings.Contains(y, "check: false") || !strings.Contains(y, "PlayTogether 區塊結束") {
		t.Errorf("project.yaml 要保留上游內容並加上玩聚區塊：\n%s", y)
	}
	src := readFile(t, root, "THIRD_PARTY/claude-code-game-studios/SOURCE.md")
	for _, f := range []string{"dev-story/SKILL.md", "story-done/SKILL.md", "team-release/SKILL.md", "release-checklist/SKILL.md", "`project.yaml`"} {
		if !strings.Contains(src, f) {
			t.Errorf("SOURCE.md 要列出修改的 %s", f)
		}
	}
}

func TestNewLegacyAIFlags(t *testing.T) {
	requireEmbeddedSDK(t)
	fakeAI(t)
	dir := t.TempDir()
	t.Chdir(dir)
	// 4、5. 舊的 -skills core／full：skill 和以前相同，記成自訂，並有工作流程契約
	must(t, cmdNew([]string{"l-core", "-skills", "core"}))
	must(t, cmdNew([]string{"l-full", "-client", "godot", "-skills", "full"}))
	if p := readProject(t, filepath.Join(dir, "l-core")); p.Skills != "core" || p.AIMode != "custom" || exists(dir, "l-core", ".agents/skills/phaser-core") {
		t.Fatalf("-skills core：%+v", p)
	}
	if p := readProject(t, filepath.Join(dir, "l-full")); p.Skills != "full" || p.AIMode != "custom" || !exists(dir, "l-full", ".agents/skills/godot-audio/SKILL.md") {
		t.Fatalf("-skills full：%+v", p)
	}
	// 6. 舊的 -ai studios：自動 Godot、不加 skill，但一樣套用完整的整合層
	must(t, cmdNew([]string{"l-studios", "-ai", "studios"}))
	root := filepath.Join(dir, "l-studios")
	if p := readProject(t, root); p.Client != "godot" || p.AI != "studios" || p.Skills != "" || p.AIMode != "custom" {
		t.Fatalf("-ai studios：%+v", p)
	}
	for _, f := range []string{"docs/ai/AI_WORKFLOW.md", "docs/ai/PROJECT_CONTEXT.md", "docs/ai/CURRENT_TASK.md", "docs/ai/DECISIONS.md"} {
		if !exists(root, f) {
			t.Errorf("-ai studios 少了 %s", f)
		}
	}
	checkStudiosPatched(t, root)
	if exists(root, ".claude/skills/godot-audio") {
		t.Fatal("-ai studios 沒指定 -skills 時不加 skill（和以前相同）")
	}
	// 7. -ai studios -skills full 和 -mode studio 產生相同的檔案（除了 aiMode）
	must(t, cmdNew([]string{"cmp-a", "-ai", "studios", "-skills", "full"}))
	must(t, cmdNew([]string{"cmp-a2", "-mode", "studio"}))
	a, b := snapshot(t, filepath.Join(dir, "cmp-a")), snapshot(t, filepath.Join(dir, "cmp-a2"))
	for _, f := range []string{"playtogether.json", "docs/ai/PROJECT_CONTEXT.md", "README.md"} { // 這幾個會寫出 AI 模式
		delete(a, f)
		delete(b, f)
	}
	if diff := diffSnapshots(a, b); diff != "" {
		t.Fatalf("-ai studios -skills full 應等於 -mode studio：%s", diff)
	}
}

// snapshot：資料夾裡每個檔案的內容（專案名稱換成固定字串，方便比較兩個專案）
func snapshot(t *testing.T, root string) map[string]string {
	t.Helper()
	name := filepath.Base(root)
	out := map[string]string{}
	must(t, filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(root, p)
		b, err := os.ReadFile(p)
		out[filepath.ToSlash(rel)] = strings.ReplaceAll(string(b), name, "<name>")
		return err
	}))
	return out
}

func diffSnapshots(a, b map[string]string) string {
	var diffs []string
	for k, v := range a {
		if w, ok := b[k]; !ok {
			diffs = append(diffs, "只有前者："+k)
		} else if v != w {
			diffs = append(diffs, "內容不同："+k)
		}
	}
	for k := range b {
		if _, ok := a[k]; !ok {
			diffs = append(diffs, "只有後者："+k)
		}
	}
	sort.Strings(diffs)
	return strings.Join(diffs, "；")
}

// 16. 同樣的參數產生兩次，文件完全相同
func TestNewDeterministic(t *testing.T) {
	requireEmbeddedSDK(t)
	fakeAI(t)
	dir := t.TempDir()
	t.Chdir(dir)
	for _, args := range [][]string{{"-mode", "studio"}, {"-mode", "standard", "-game-server", "external"}, {"-mode", "lightweight", "-game-server", "none", "-client", "godot"}} {
		must(t, cmdNew(append([]string{"d-one"}, args...)))
		must(t, cmdNew(append([]string{"d-two"}, args...)))
		if diff := diffSnapshots(snapshot(t, filepath.Join(dir, "d-one")), snapshot(t, filepath.Join(dir, "d-two"))); diff != "" {
			t.Fatalf("%v 兩次產生的結果不同：%s", args, diff)
		}
		must(t, os.RemoveAll(filepath.Join(dir, "d-one")))
		must(t, os.RemoveAll(filepath.Join(dir, "d-two")))
	}
}

// 10–14. 遊戲伺服器架構
func TestNewGameServerModes(t *testing.T) {
	requireEmbeddedSDK(t)
	dir := t.TempDir()
	t.Chdir(dir)
	if err := cmdNew([]string{"gs-bad", "-game-server", "p2p"}); err == nil {
		t.Fatal("不合法的 -game-server 應拒絕")
	}
	must(t, cmdNew([]string{"gs-none", "-game-server", "none"}))
	must(t, cmdNew([]string{"gs-managed"}))
	must(t, cmdNew([]string{"gs-ext", "-game-server", "external", "-client", "godot"}))

	for name, want := range map[string]struct {
		mode  string
		multi bool
	}{"gs-none": {"none", false}, "gs-managed": {"playtogether-managed", true}, "gs-ext": {"external", true}} {
		p := readProject(t, filepath.Join(dir, name))
		if p.gameServerMode() != want.mode || p.multiplayerEnabled() != want.multi {
			t.Errorf("%s：%+v %+v", name, p.Multiplayer, p.GameServer)
		}
		// 範本伺服器一律產生
		if !exists(dir, name, "server") {
			t.Errorf("%s 也要有範本 server/", name)
		}
		ctx := readFile(t, dir, name, "docs/ai/PROJECT_CONTEXT.md")
		if !strings.Contains(ctx, "`"+want.mode+"`") {
			t.Errorf("%s 的 PROJECT_CONTEXT 要寫出 %s", name, want.mode)
		}
		// 14. 所有模式：改連線前先確認宣告的架構，不自動換成引擎內建或範本伺服器
		wf := readFile(t, dir, name, "docs/ai/AI_WORKFLOW.md")
		for _, s := range []string{"修改連線或遊戲伺服器相關程式前，一定先讀它", "Godot Multiplayer", "也不得自動改成 playtogether-managed 遊戲伺服器"} {
			if !strings.Contains(wf, s) {
				t.Errorf("%s 的 AI_WORKFLOW 少了 %q", name, s)
			}
		}
		pg := readFile(t, dir, name, "PLAYTOGETHER.md")
		if strings.Contains(pg, "一律用範本的遊戲伺服器") || !strings.Contains(pg, "遊戲伺服器架構以專案宣告為準") {
			t.Errorf("%s 的 PLAYTOGETHER.md 不能寫死範本伺服器", name)
		}
	}
	// 13. external：不被要求遷移，server/ 只是參考
	ext := readFile(t, dir, "gs-ext", "docs/ai/PROJECT_CONTEXT.md")
	if !strings.Contains(ext, "不是你的正式遊戲伺服器") || !strings.Contains(ext, "你的正式遊戲伺服器") {
		t.Fatal("external 要寫明 server/ 不是正式遊戲伺服器，並留欄位寫自己的伺服器")
	}
	if r := readFile(t, dir, "gs-ext", "README.md"); !strings.Contains(r, "你自己的遊戲伺服器") || strings.Contains(r, "部署遊戲伺服器（`server/`）") {
		t.Fatal("external 的 README 不能要求部署範本伺服器")
	}
	if !strings.Contains(readFile(t, dir, "gs-ext", "PLAYTOGETHER.md"), "不要求遷移到範本伺服器") {
		t.Fatal("external 的 PLAYTOGETHER.md 要寫明不要求遷移")
	}
	// none：不需要部署、不主動加伺服器、可信任狀態的規則
	none := readFile(t, dir, "gs-none", "docs/ai/AI_WORKFLOW.md")
	if !strings.Contains(none, "不要主動加入多人框架或遊戲伺服器") || !strings.Contains(none, "**不要**為此自動加一台遊戲伺服器") {
		t.Fatal("none 的 AI_WORKFLOW 要禁止自動加伺服器")
	}
	if r := readFile(t, dir, "gs-none", "README.md"); !strings.Contains(r, "不需要部署") {
		t.Fatal("none 的 README 要寫明 server/ 不需要部署")
	}
}

// 單人遊戲（none）不需要遊戲伺服器金鑰：README、Makefile、PROJECT_CONTEXT 都不要求；其他模式照舊兩把
func TestServerKeyByGameServerMode(t *testing.T) {
	requireEmbeddedSDK(t)
	dir := t.TempDir()
	t.Chdir(dir)
	must(t, cmdNew([]string{"k-none", "-game-server", "none"}))
	must(t, cmdNew([]string{"k-managed"}))
	must(t, cmdNew([]string{"k-ext", "-game-server", "external"}))

	readme := readFile(t, dir, "k-none", "README.md")
	setup := readme[strings.Index(readme, "## 接上玩聚"):strings.Index(readme, "## 資料夾")]
	if !strings.Contains(setup, "pt keygen -purpose upload") || !strings.Contains(setup, "不需要遊戲伺服器金鑰") ||
		strings.Contains(setup, "`serverKeyId`、") || strings.Contains(setup, "（`uploadKeyId`、`serverKeyId`") {
		t.Fatalf("none 的 README 只要求上傳金鑰：\n%s", setup)
	}
	mk := readFile(t, dir, "k-none", "Makefile")
	if strings.Contains(mk, "-purpose server") || !strings.Contains(mk, "-purpose upload") {
		t.Fatalf("none 的 make keys 只產生上傳金鑰：\n%s", mk)
	}
	if ctx := readFile(t, dir, "k-none", "docs/ai/PROJECT_CONTEXT.md"); !strings.Contains(ctx, "不需要遊戲伺服器金鑰") {
		t.Fatal("none 的 PROJECT_CONTEXT 要寫明不需要遊戲伺服器金鑰")
	}
	for _, name := range []string{"k-managed", "k-ext"} {
		r := readFile(t, dir, name, "README.md")
		if !strings.Contains(r, "pt keygen -purpose upload") || !strings.Contains(r, "pt keygen -purpose server") || !strings.Contains(r, "`serverKeyId`") {
			t.Errorf("%s 的 README 要要求兩把金鑰", name)
		}
		if m := readFile(t, dir, name, "Makefile"); !strings.Contains(m, "-purpose server") || !strings.Contains(m, "-purpose upload") {
			t.Errorf("%s 的 make keys 要產生兩把金鑰", name)
		}
	}
	if !strings.Contains(readFile(t, dir, "k-ext", "README.md"), "放到你自己的遊戲伺服器") {
		t.Fatal("external 要說明遊戲伺服器金鑰的私鑰放在自己的伺服器")
	}

	// serverUrl 的文案依宣告的架構：none 明確說不需要；其他照舊要求
	if !strings.Contains(setup, "此模式沒有正式遊戲伺服器") || !strings.Contains(setup, "不需要 `serverUrl`") {
		t.Fatalf("none 的 README 要明確說沒有正式遊戲伺服器、不需要 serverUrl：\n%s", setup)
	}
	for _, f := range []string{"README.md", "Makefile", "PLAYTOGETHER.md"} {
		if s := readFile(t, dir, "k-none", f); strings.Contains(s, "serverUrl 都要填好") || strings.Contains(s, "`serverUrl` 都要填好") {
			t.Errorf("none 的 %s 不能說 serverUrl 必填", f)
		}
	}
	if !strings.Contains(mk, "此模式不需要 serverUrl") || !strings.Contains(readFile(t, dir, "k-none", "web/config.js"), "上傳時保持空白") {
		t.Fatal("none 的 make check 說明與 config.js 要說明不需要 serverUrl")
	}
	for _, name := range []string{"k-managed", "k-ext"} {
		if !strings.Contains(readFile(t, dir, name, "Makefile"), "platform、serverUrl 都要填好") ||
			!strings.Contains(readFile(t, dir, name, "README.md"), "`platform`、`serverUrl` 都要填好") ||
			!strings.Contains(readFile(t, dir, name, "web/config.js"), "上傳前要改成你的遊戲伺服器網址") {
			t.Errorf("%s 仍要說明 serverUrl 必填", name)
		}
	}
	// external 的 server/ 仍是參考與本機模擬伺服器
	if !strings.Contains(readFile(t, dir, "k-ext", "docs/ai/PROJECT_CONTEXT.md"), "不是你的正式遊戲伺服器") {
		t.Fatal("external 的 server/ 要寫明是參考與本機模擬")
	}
	// 三種模式都保留範本伺服器與驗證流程
	for _, name := range []string{"k-none", "k-managed", "k-ext"} {
		if !exists(dir, name, "server") || !strings.Contains(readFile(t, dir, name, "Makefile"), "$(PT) dev -smoke") {
			t.Errorf("%s 要保留 server/ 與 pt dev -smoke", name)
		}
	}
	// 玩家分頁數的文案：單人一個、多人兩個
	if !strings.Contains(readFile(t, dir, "k-none", "README.md"), "打開一個玩家分頁") || strings.Contains(readFile(t, dir, "k-none", "README.md"), "兩個玩家") {
		t.Fatal("單人專案的 README 應該寫打開一個玩家分頁")
	}
	if !strings.Contains(readFile(t, dir, "k-managed", "README.md"), "打開兩個玩家分頁") {
		t.Fatal("多人專案的 README 應該寫打開兩個玩家分頁")
	}
}

// pt dev 的玩家分頁數：沒指定時多人 2、單人 1；明確指定的值照用（單人也可以 -players 2）
func TestDevPlayersDefault(t *testing.T) {
	multi := &project{Multiplayer: &multiplayerConfig{true}, GameServer: &gameServerConfig{"playtogether-managed"}}
	single := &project{Multiplayer: &multiplayerConfig{false}, GameServer: &gameServerConfig{"none"}}
	legacy := &project{} // 舊專案：視為多人
	for _, c := range []struct {
		name string
		p    *project
		flag int
		want int
	}{
		{"多人預設", multi, -1, 2},
		{"單人預設", single, -1, 1},
		{"舊專案預設", legacy, -1, 2},
		{"單人明確指定 2", single, 2, 2},
		{"多人明確指定 1", multi, 1, 1},
		{"不開瀏覽器", single, 0, 0},
	} {
		if got := devPlayers(c.p, c.flag); got != c.want {
			t.Errorf("%s：得到 %d，應為 %d", c.name, got, c.want)
		}
	}
	// 旗標的預設值是「沒指定」，才分得出開發者有沒有明確給 -players
	fs := flag.NewFlagSet("dev", flag.ContinueOnError)
	players := fs.Int("players", -1, "")
	must(t, fs.Parse([]string{"-players", "2"}))
	if devPlayers(single, *players) != 2 {
		t.Fatal("單人專案明確指定 -players 2 要開兩個")
	}
}

// 12a. pt check 依宣告的架構判斷上架資格
func TestCheckGameServer(t *testing.T) {
	cases := []struct {
		name      string
		multi     *multiplayerConfig
		gs        *gameServerConfig
		serverURL string
		wantErr   bool
		wantURL   bool // 是否接著檢查 serverUrl
	}{
		{"單人＋none", &multiplayerConfig{false}, &gameServerConfig{"none"}, "", false, false},
		{"單人＋none 但填了網址", &multiplayerConfig{false}, &gameServerConfig{"none"}, "https://x.example", false, false},
		{"多人＋範本", &multiplayerConfig{true}, &gameServerConfig{"playtogether-managed"}, "", false, true},
		{"多人＋external", &multiplayerConfig{true}, &gameServerConfig{"external"}, "", false, true},
		{"多人＋none 不成立", &multiplayerConfig{true}, &gameServerConfig{"none"}, "", true, false},
		{"單人＋範本", &multiplayerConfig{false}, &gameServerConfig{"playtogether-managed"}, "", false, true},
		{"舊專案（沒有欄位）", nil, nil, "", false, true},
		{"不合法的值", nil, &gameServerConfig{"p2p"}, "", true, false},
	}
	for _, c := range cases {
		p := &project{Multiplayer: c.multi, GameServer: c.gs, ServerURL: c.serverURL}
		var errs []string
		gotURL := checkGameServer(p, func(level, format string, a ...any) {
			if level == "error" {
				errs = append(errs, format)
			}
		})
		if (len(errs) > 0) != c.wantErr || gotURL != c.wantURL {
			t.Errorf("%s：錯誤 %v、檢查 serverUrl %v", c.name, errs, gotURL)
		}
	}

	// 實際的 pt check：none 不要求 serverUrl；範本伺服器照舊要求
	requireEmbeddedSDK(t)
	dir := t.TempDir()
	t.Chdir(dir)
	must(t, cmdNew([]string{"c-none", "-game-server", "none", "-platform", "https://pt.example"}))
	must(t, cmdNew([]string{"c-managed", "-platform", "https://pt.example"}))
	for name, wantServerURLErr := range map[string]bool{"c-none": false, "c-managed": true} {
		t.Chdir(filepath.Join(dir, name))
		p, _ := findProject()
		found := false
		for _, f := range checkProject(p, p.path(p.Web)) {
			if f.level == "error" && strings.Contains(f.msg, "serverUrl") {
				found = true
			}
		}
		if found != wantServerURLErr {
			t.Errorf("%s：serverUrl 錯誤 %v，應為 %v", name, found, wantServerURLErr)
		}
	}
}

// 18. -y、非互動與 CI：沒指定 AI 模式時用 lightweight，不連網；明確指定仍然優先
func TestNewNoNetworkByDefault(t *testing.T) {
	requireEmbeddedSDK(t)
	hits := fakeAI(t)
	dir := t.TempDir()
	t.Chdir(dir)
	must(t, cmdNew([]string{"n-yes", "-y", "-client", "godot"}))
	must(t, cmdNew([]string{"n-script", "-client", "godot"})) // 非互動
	ciEnv = func() bool { return true }
	t.Cleanup(func() { ciEnv = func() bool { return false } })
	typeIn(t) // 有終端機但在 CI：不進入精靈（沒有輸入也不會卡住）
	must(t, cmdNew([]string{"n-ci"}))
	if *hits != 0 {
		t.Fatalf("沒指定 AI 模式時不能連網（連了 %d 次）", *hits)
	}
	for _, n := range []string{"n-yes", "n-script", "n-ci"} {
		if p := readProject(t, filepath.Join(dir, n)); p.AIMode != "lightweight" || p.Skills != "" || p.AI != "" {
			t.Errorf("%s 應該是 lightweight：%+v", n, p)
		}
	}
	must(t, cmdNew([]string{"n-explicit", "-mode", "standard"}))
	if *hits == 0 || !exists(dir, "n-explicit", ".agents/skills/game-feel/SKILL.md") {
		t.Fatal("在 CI 明確指定 -mode standard 仍要下載 skill")
	}
}

// 17. 失敗時不留下專案：Studios 的設定調整找不到要改的地方（上游改版）
func TestStudiosPatchFailures(t *testing.T) {
	for name, c := range map[string]struct{ content, old string }{
		"沒有 frontmatter":  {"# skill\ndisable-model-invocation: true\n", "disable-model-invocation: true\n"},
		"找不到":             {"---\nname: x\n---\nbody\n", "disable-model-invocation: true\n"},
		"出現兩次":            {"---\nuser-invocable: true\nuser-invocable: true\n---\n", "user-invocable: true\n"},
		"只在內文出現":          {"---\nname: x\n---\nuser-invocable: true\n", "user-invocable: true\n"},
		"frontmatter 沒結尾": {"---\nuser-invocable: true\n", "user-invocable: true\n"},
	} {
		if _, err := patchFrontmatter([]byte(c.content), c.old, "x\n"); err == nil {
			t.Errorf("%s：應該失敗", name)
		}
	}
	out, err := patchFrontmatter([]byte("---\na: 1\nuser-invocable: true\n---\nuser-invocable: true\n"), "user-invocable: true\n", "user-invocable: false\n")
	if err != nil || string(out) != "---\na: 1\nuser-invocable: false\n---\nuser-invocable: true\n" {
		t.Fatalf("只改 frontmatter：%q %v", out, err)
	}

	requireEmbeddedSDK(t)
	dir := t.TempDir()
	t.Chdir(dir)
	for name, override := range map[string]map[string]string{
		"上游已經是可自動使用":            {".claude/skills/dev-story/SKILL.md": "---\nname: dev-story\nuser-invocable: true\n---\n"},
		"project.yaml 已有 modes": {"project.yaml": "schema_version: 1\nmodes:\n  rigor: full\n"},
	} {
		// 清單和下載內容一致（雜湊通過），但無法套用玩聚的設定
		files := map[string]string{}
		for k, v := range fakeStudio {
			files[k] = v
		}
		for k, v := range override {
			files[k] = v
		}
		m := manifestFor(files)
		orig := loadCCGS
		loadCCGS = func() (ccgsManifest, error) { return m, nil }
		var entries []tarEntry
		for k, v := range files {
			entries = append(entries, tarEntry{name: k, body: v})
		}
		archive := tarball(t, entries...)
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write(archive) }))
		t.Setenv("PT_CCGS_URL", srv.URL)
		err := cmdNew([]string{"p-bad", "-ai", "studios"})
		srv.Close()
		loadCCGS = orig
		if err == nil || !strings.Contains(err.Error(), "停止安裝") {
			t.Errorf("%s：應停止安裝：%v", name, err)
		}
		if exists(dir, "p-bad") {
			t.Fatalf("%s：失敗時不能留下專案資料夾", name)
		}
	}
}

func manifestFor(files map[string]string) ccgsManifest {
	orig := fakeStudio
	fakeStudio = files
	defer func() { fakeStudio = orig }()
	return fakeManifest()
}

// 15. 被排除的 skill 不會裝進來（清單外的資料夾、router）
func TestExcludedSkillsStayOut(t *testing.T) {
	requireEmbeddedSDK(t)
	fakeAI(t)
	dir := t.TempDir()
	t.Chdir(dir)
	must(t, cmdNew([]string{"x-std", "-mode", "standard", "-client", "godot"}))
	for _, p := range []string{"router", "create-game-assets", "godot-multiplayer", "godot-export", "save-systems"} {
		if exists(dir, "x-std", ".claude/skills", p) || exists(dir, "x-std", ".agents/skills", p) {
			t.Errorf("不該安裝 %s", p)
		}
	}
	// 內嵌的正式清單也不含衝突的 skill
	m, err := loadEmbeddedSkills()
	must(t, err)
	for _, s := range m.Skills {
		switch s.Name {
		case "godot-multiplayer", "godot-export", "save-systems", "router":
			t.Errorf("正式清單不該收 %s", s.Name)
		}
	}
	if pg := readFile(t, dir, "x-std", "PLAYTOGETHER.md"); !strings.Contains(pg, "godot-multiplayer、godot-export、save-systems") {
		t.Fatal("PLAYTOGETHER.md 要說明被排除的 skill 不要自己裝回來")
	}
}

// 20. 入口檔的閱讀順序與優先順序；開發者看到的文件不要求記指令
func TestEntryPointPrecedence(t *testing.T) {
	requireEmbeddedSDK(t)
	fakeAI(t)
	dir := t.TempDir()
	t.Chdir(dir)
	must(t, cmdNew([]string{"e-std", "-mode", "standard"}))
	must(t, cmdNew([]string{"e-studio", "-mode", "studio"}))
	order := func(s string, parts ...string) bool {
		last := -1
		for _, p := range parts {
			i := strings.Index(s, p)
			if i <= last {
				return false
			}
			last = i
		}
		return true
	}
	for _, p := range []string{"e-std", "e-studio"} {
		agents := readFile(t, dir, p, "AGENTS.md")
		if !order(agents, "PLAYTOGETHER.md", "docs/ai/AI_WORKFLOW.md", "docs/ai/PROJECT_CONTEXT.md", "docs/ai/CURRENT_TASK.md") || !strings.Contains(agents, "以它為準") {
			t.Errorf("%s 的 AGENTS.md 閱讀順序不對：\n%s", p, agents)
		}
		claude := readFile(t, dir, p, "CLAUDE.md")
		if !order(claude, "@PLAYTOGETHER.md", "@docs/ai/AI_WORKFLOW.md", "@docs/ai/PROJECT_CONTEXT.md") || !strings.Contains(claude, "自動路由") {
			t.Errorf("%s 的 CLAUDE.md 要負責自動路由並依序匯入：\n%s", p, claude)
		}
		wf := readFile(t, dir, p, "docs/ai/AI_WORKFLOW.md")
		if !order(wf, "1. `PLAYTOGETHER.md`（平台規則）", "2. 專案特定限制", "4. 遊戲願景") || !strings.Contains(wf, "`PLAYTOGETHER.md` 永遠優先") {
			t.Errorf("%s 的權威順序不對", p)
		}
		for _, f := range []string{"README.md", "CLAUDE.md"} {
			s := readFile(t, dir, p, f)
			if p == "e-studio" && f == "CLAUDE.md" {
				s = s[:strings.Index(s, "PlayTogether 區塊結束")] // Studios 原本的 CLAUDE.md 不是我們寫的
			}
			for _, cmd := range []string{"/start", "/dev-story", "/setup-engine", "/settings"} {
				if strings.Contains(s, cmd) {
					t.Errorf("%s 的 %s 不該要求開發者輸入 %s", p, f, cmd)
				}
			}
		}
	}
	if a := readFile(t, dir, "e-studio", "AGENTS.md"); !strings.Contains(a, "Codex 不執行它的流程") {
		t.Fatal("工作室專案的 AGENTS.md 要寫明 Codex 邊界")
	}
	if a := readFile(t, dir, "e-std", "AGENTS.md"); strings.Contains(a, "Game Studios") {
		t.Fatal("沒有 Studios 的專案不該提到它")
	}
	// Studios 的 CLAUDE.md：玩聚區塊在最前面、Studios 原本內容在後面
	c := readFile(t, dir, "e-studio", "CLAUDE.md")
	if !bytes.HasPrefix([]byte(c), []byte("<!-- 由 pt new 加入（PlayTogether）")) || strings.Index(c, "studio rules") < strings.Index(c, "PlayTogether 區塊結束") {
		t.Fatal("玩聚區塊要在 Studios 的 CLAUDE.md 之前並明確標記")
	}
}
