package main

// pt new -skills core|full：把 awesome-gamedev-agent-skills（Apache-2.0，github.com/gamedev-skills/awesome-gamedev-agent-skills）
// 的精選 skill 加進新專案，Claude Code（.claude/skills/）與 Codex（.agents/skills/）都讀得到。
//
// 只收說明文件（SKILL.md、references、create-game-assets 的範本），不收任何 scripts、agents、router 或 CI；
// 和玩聚規則衝突的 skill（godot-multiplayer、godot-export、save-systems 等）不收，已知衝突寫在 PLAYTOGETHER.md。
// 版本固定在 assets/ai/gdskills.json 的 commit，逐檔比對 SHA-256（和 -ai studios 相同的檢查）。
// 檔案原樣複製、不修改內容（Apache-2.0 第 4 條），LICENSE 與 NOTICE 放在 THIRD_PARTY/awesome-gamedev-agent-skills/。
// 升級版本見 scripts/gdskills-manifest.py。

import (
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"sort"
	"strings"
)

type skillEntry struct {
	Name  string `json:"name"`
	Dir   string `json:"dir"`   // 原始碼裡的資料夾，例如 skills/disciplines/game-feel
	Group string `json:"group"` // core、godot、web
}

// skillLevels：-skills 的選項
var skillLevels = []string{"none", "core", "full"}

// loadSkills 讀內嵌的清單（測試時換成假的清單）
var loadSkills = loadEmbeddedSkills

func loadEmbeddedSkills() (ccgsManifest, error) {
	var m ccgsManifest
	raw, err := embedded.ReadFile("assets/ai/gdskills.json")
	if err != nil {
		return m, err
	}
	if err := json.Unmarshal(raw, &m); err != nil {
		return m, err
	}
	if len(m.Commit) != 40 || len(m.Files) == 0 || len(m.Skills) == 0 {
		return m, errors.New("內嵌的 awesome-gamedev-agent-skills 清單格式錯誤")
	}
	return m, nil
}

// selectedSkills：core 只裝通用的；full 再加上這個畫面（web／godot）的引擎 skill
func selectedSkills(m ccgsManifest, level, client string) []skillEntry {
	var out []skillEntry
	for _, s := range m.Skills {
		if s.Group == "core" || (level == "full" && s.Group == client) {
			out = append(out, s)
		}
	}
	return out
}

// addSkillsFiles 把驗證過的 skill 放進要寫入的檔案（files：路徑 → 內容），兩個 AI 工具各一份。
func addSkillsFiles(files map[string][]byte, m ccgsManifest, got map[string][]byte, level, client string) ([]string, error) {
	var names []string
	for _, s := range selectedSkills(m, level, client) {
		names = append(names, s.Name)
		prefix := s.Dir + "/"
		for rel, content := range got {
			if !strings.HasPrefix(rel, prefix) {
				continue
			}
			sub := strings.TrimPrefix(rel, prefix)
			for _, root := range []string{".claude/skills", ".agents/skills"} {
				dst := path.Join(root, s.Name, sub)
				if _, exists := files[dst]; exists {
					return nil, fmt.Errorf("%s 的 %s 和專案裡已有的 %s 衝突", m.Name, s.Name, dst)
				}
				files[dst] = content
			}
		}
	}
	sort.Strings(names)
	third := "THIRD_PARTY/awesome-gamedev-agent-skills/"
	files[third+"LICENSE"] = got["LICENSE"]
	files[third+"NOTICE"] = got["NOTICE"]
	files[third+"SOURCE.md"] = []byte(fmt.Sprintf(
		"# %s\n\n- 來源：https://github.com/%s\n- 版本：%s（commit %s）\n- 授權：%s（見同資料夾的 LICENSE、NOTICE）\n- 收錄的 skill（`.claude/skills/` 給 Claude Code、`.agents/skills/` 給 Codex，內容相同）：%s\n\n"+
			"由 `pt new -skills %s` 加入：每個檔案都比對過 SHA-256，內容沒有修改。只收說明文件，沒有收 router、scripts、agents 與和玩聚規則衝突的 skill；"+
			"和玩聚規則的已知衝突寫在專案根目錄的 PLAYTOGETHER.md「第三方遊戲開發 skill」。\n",
		m.Name, m.Repo, m.Version, m.Commit, m.License, strings.Join(names, "、"), level))
	return names, nil
}
