package main

// AI 開發模式與遊戲伺服器架構（pt new -mode、-game-server）。
//
// 開發者用「想要 AI 怎麼幫忙」選擇模式，pt 對應到既有的元件（-skills、-ai studios），
// 並產生所有 AI 工具共用的工作流程契約（docs/ai/AI_WORKFLOW.md、PROJECT_CONTEXT.md）。
// 遊戲伺服器架構寫在 playtogether.json（multiplayer、gameServer），AI 改連線前一定要先確認。

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"slices"
	"strings"
)

// AI 模式：lightweight（只有平台規則）、standard（加精選 skill）、studio（再加 Claude Code Game Studios）、custom（自己指定 -skills、-ai）
var aiModes = []string{"lightweight", "standard", "studio", "custom"}

// 遊戲伺服器模式：none（正式遊玩不需要遊戲伺服器）、playtogether-managed（玩聚支援的範本伺服器架構）、external（開發者自己的遊戲伺服器）
var gameServerModes = []string{"none", "playtogether-managed", "external"}

const defaultGameServer = "playtogether-managed"

type multiplayerConfig struct {
	Enabled bool `json:"enabled"`
}

type gameServerConfig struct {
	Mode string `json:"mode"`
	// Hosting 是 "playtogether" 時，pt upload 會建置遊戲伺服器（Go）並交給玩聚代管（serverbundle.go）
	Hosting string `json:"hosting,omitempty"`
	// Dir 是遊戲伺服器 Go 模組的目錄（相對於專案根目錄；預設 "."）
	Dir string `json:"dir,omitempty"`
	// Data 是遊戲伺服器要讀的資料檔目錄（選填；會放進套件的 data/，伺服器執行時在 /app/data）
	Data string `json:"data,omitempty"`
}

// hostedServer：遊戲伺服器由玩聚代管
func (p *project) hostedServer() bool {
	return p != nil && p.GameServer != nil && p.GameServer.Hosting == "playtogether"
}

// gameServerMode：舊專案沒有 gameServer 欄位，視為原本的行為（範本伺服器）
func (p *project) gameServerMode() string {
	if p.GameServer == nil || p.GameServer.Mode == "" {
		return defaultGameServer
	}
	return p.GameServer.Mode
}

// multiplayerEnabled：舊專案沒有 multiplayer 欄位，視為多人遊戲
func (p *project) multiplayerEnabled() bool {
	if p.Multiplayer == nil {
		return true
	}
	return p.Multiplayer.Enabled
}

// ciEnv：在 CI 裡不進入精靈（CI 不應卡在詢問），測試時替換
var ciEnv = func() bool { return os.Getenv("CI") != "" }

// resolveAIMode 把 -mode 與舊參數 -ai、-skills 對應成最後的 (模式, ai, skills)。
// 回傳的模式是空字串表示還沒決定（終端機會詢問，非互動時用 lightweight）。
func resolveAIMode(mode, ai, skills string, set map[string]bool) (string, string, string, error) {
	if mode != "" && !slices.Contains(aiModes, mode) {
		return "", "", "", fmt.Errorf("-mode 只能是 lightweight、standard、studio 或 custom（目前是 %q）", mode)
	}
	legacy := set["ai"] || set["skills"]
	switch mode {
	case "":
		if legacy {
			return "custom", ai, skills, nil
		}
		return "", ai, skills, nil
	case "custom":
		return "custom", ai, skills, nil
	}
	if legacy {
		return "", "", "", errors.New("-mode 已經決定 skill 與 Claude Code Game Studios，要自己指定 -skills、-ai 請用 -mode custom")
	}
	ai, skills = modeComponents(mode)
	return mode, ai, skills, nil
}

// modeComponents：模式對應的元件
func modeComponents(mode string) (ai, skills string) {
	switch mode {
	case "standard":
		return "none", "full"
	case "studio":
		return "studios", "full"
	}
	return "none", "none"
}

// ---------- Claude Code Game Studios 的設定調整 ----------
//
// 在逐檔驗證 SHA-256 之後、寫入磁碟之前，只改 frontmatter 的一行並加上玩聚的標記；
// 要替換的字串必須剛好出現一次，否則整個安裝失敗（上游改版時不會靜默失效）。

const ptMark = "# PlayTogether（pt new 加入）："

type studiosPatch struct {
	path, old, new, why string
}

var studiosPatches = []studiosPatch{
	{".claude/skills/dev-story/SKILL.md", "disable-model-invocation: true\n",
		ptMark + "AI 可以自動實作 story，開發者不需要輸入指令（docs/ai/AI_WORKFLOW.md）\n",
		"移除 disable-model-invocation，讓 AI 自動實作 story"},
	{".claude/skills/story-done/SKILL.md", "disable-model-invocation: true\n",
		ptMark + "AI 可以自動完成 story 檢查，開發者不需要輸入指令（docs/ai/AI_WORKFLOW.md）\n",
		"移除 disable-model-invocation，讓 AI 自動完成 story 檢查"},
	{".claude/skills/team-release/SKILL.md", "user-invocable: true\n",
		"user-invocable: true\n" + ptMark + "發佈由玩聚流程（pt check、pt upload、審核）負責，AI 不自動使用（docs/ai/AI_WORKFLOW.md）\ndisable-model-invocation: true\n",
		"加上 disable-model-invocation，發佈由玩聚流程負責"},
	{".claude/skills/release-checklist/SKILL.md", "user-invocable: true\n",
		"user-invocable: true\n" + ptMark + "發佈由玩聚流程（pt check、pt upload、審核）負責，AI 不自動使用（docs/ai/AI_WORKFLOW.md）\ndisable-model-invocation: true\n",
		"加上 disable-model-invocation，發佈由玩聚流程負責"},
}

// projectYAMLBlock 加在 Studios 的 project.yaml 最後：開發者不需要回答 Studios 的內部設定
const projectYAMLBlock = `
# ---- PlayTogether（pt new 加入）：開發者不需要回答 Claude Code Game Studios 的內部設定 ----
modes:
  rigor: minimal        # AI 可依專案複雜度自行調整（docs/ai/AI_WORKFLOW.md）
  automation: guided    # 小決定 AI 說明後直接做，改變產品方向的才問開發者
updates:
  check: false          # 升級一律透過新版 pt（固定版本與逐檔雜湊驗證）
# ---- PlayTogether 區塊結束 ----
`

var yamlTopKey = regexp.MustCompile(`(?m)^(modes|updates):`)

// patchFrontmatter 只在檔案開頭的 frontmatter（--- 之間）替換，old 必須剛好出現一次
func patchFrontmatter(content []byte, old, new string) ([]byte, error) {
	s := string(content)
	if !strings.HasPrefix(s, "---\n") {
		return nil, errors.New("沒有 frontmatter")
	}
	end := strings.Index(s[4:], "\n---\n")
	if end < 0 {
		return nil, errors.New("frontmatter 沒有結尾")
	}
	fm := s[4 : 4+end+1]
	if n := strings.Count(fm, old); n != 1 {
		return nil, fmt.Errorf("frontmatter 裡 %q 出現 %d 次（應該剛好一次）", strings.TrimSpace(old), n)
	}
	return []byte(s[:4] + strings.Replace(fm, old, new, 1) + s[4+end+1:]), nil
}

// applyStudiosPatches 套用上面的調整；任何一項失敗就整個安裝失敗。回傳修改說明（寫進 SOURCE.md）。
func applyStudiosPatches(files map[string][]byte, m ccgsManifest) ([]string, error) {
	var notes []string
	for _, p := range studiosPatches {
		content, ok := files[p.path]
		if !ok {
			return nil, fmt.Errorf("%s 缺少 %s：這個版本無法套用玩聚的設定，停止安裝", m.Name, p.path)
		}
		out, err := patchFrontmatter(content, p.old, p.new)
		if err != nil {
			return nil, fmt.Errorf("%s 的 %s 無法套用玩聚的設定（%v）：停止安裝，沒有寫入任何檔案", m.Name, p.path, err)
		}
		files[p.path] = out
		notes = append(notes, fmt.Sprintf("`%s`：%s", p.path, p.why))
	}
	y, ok := files["project.yaml"]
	if !ok {
		return nil, fmt.Errorf("%s 缺少 project.yaml：停止安裝", m.Name)
	}
	if yamlTopKey.Match(y) {
		return nil, fmt.Errorf("%s 的 project.yaml 已經有 modes 或 updates：無法套用玩聚的設定，停止安裝", m.Name)
	}
	if len(y) > 0 && y[len(y)-1] != '\n' {
		y = append(y, '\n')
	}
	files["project.yaml"] = append(append([]byte{}, y...), projectYAMLBlock...)
	notes = append(notes, "`project.yaml`：最後加上玩聚區塊（`modes.rigor: minimal`、`modes.automation: guided`、`updates.check: false`），開發者不需要回答內部設定，也不會被引導繞過 pt 升級")
	return notes, nil
}

// firstRunHint：pt new 結束時與 AI 第一次回覆時的提示（CLAUDE.md、AGENTS.md 也引用同一段）
const firstRunHint = `這個專案已準備好使用 AI 協助開發。
你可以直接用自然語言描述想做的遊戲或功能，例如：
  - 幫我做一款 4 人合作遊戲
  - 我要做心理恐怖解謎遊戲
  - 幫我建立第一個可玩原型
  - 幫我改善目前操作手感
  - 幫我檢查目前專案
AI 會自動選擇適合的設計與實作流程。`

// checkGameServer 依宣告的架構判斷上架資格（不是看「有沒有遊戲伺服器」）；回傳是否需要檢查 serverUrl。
func checkGameServer(p *project, add func(level, format string, a ...any)) bool {
	mode := p.gameServerMode()
	if !slices.Contains(gameServerModes, mode) {
		add(levelSchema, "playtogether.json 的 gameServer.mode 只能是 playtogether-managed、external 或 none（目前是 %q）", mode)
		return false
	}
	if mode == "none" {
		if p.multiplayerEnabled() {
			add(levelHard, "多人遊戲（multiplayer.enabled: true）需要經核准的連線架構：gameServer.mode 要是 playtogether-managed 或 external（目前是 none）；不要自己加上引擎內建的多人連線")
			return false
		}
		if p.ServerURL != "" {
			add("info", "gameServer.mode 是 none：正式遊玩不需要遊戲伺服器，serverUrl 可以留空（目前是 %s）", p.ServerURL)
		}
		return false
	}
	return true
}

// devPlayers：pt dev 預設開幾個玩家分頁。沒有明確指定（-players 小於 0）時，多人專案 2 個、單人專案 1 個；明確指定的值一律照用。
func devPlayers(p *project, flagValue int) int {
	if flagValue >= 0 {
		return flagValue
	}
	if p.multiplayerEnabled() {
		return 2
	}
	return 1
}
