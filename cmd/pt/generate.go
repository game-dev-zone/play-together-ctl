package main

// 專案範本的產生：pt new 建立專案、pt upgrade 重新產生範本來比較，都用這裡（結果一樣才能判斷「沒改過」）。
//
// 每個檔案有一個種類（.pt/generated.json 的 kind）：
//   - sdk：玩聚 SDK（pt sdk update 更新）
//   - template：平台規則、AI 契約、Makefile、pt 的 skill 等「跟著 pt 版本走」的檔案（pt upgrade 更新）
//   - starter：遊戲本身的起點（畫面、伺服器、上架資料、素材、playtogether.json、內容宣告文件），建立後就是開發者的，
//     pt upgrade 不會碰（伺服器的平台接口在 SDK 的套件裡，不在這些檔案）
// 第三方內容（Claude Code Game Studios、awesome-gamedev-agent-skills）不在這裡，也不由 pt upgrade 管理。

import (
	"bytes"
	"fmt"
	"path"
	"strings"
	texttemplate "text/template"
)

const (
	kindSDK      = "sdk"
	kindTemplate = "template"
	kindStarter  = "starter"
)

// genInputs：產生範本需要的設定（pt new 從參數與精靈來；pt upgrade 從 playtogether.json 來）。
type genInputs struct {
	Name, Client, Server, Platform, Orientation, AIMode, GameServer, Hosting, Skills string
	Studios                                                                          bool
	Policy                                                                           policy
	CCGS, GDS                                                                        ccgsManifest
}

func (in genInputs) godotDir() string {
	if in.Studios {
		return "." // Claude Code Game Studios 的工具假設 project.godot 在根目錄
	}
	return "game"
}

func (in genInputs) data() map[string]string {
	godotDir := in.godotDir()
	godotPrefix := godotDir + "/"
	if godotDir == "." {
		godotPrefix = ""
	}
	godotPath := godotPrefix
	if godotPath == "" {
		godotPath = "./"
	}
	skills := in.Skills
	if skills == "" {
		skills = "none"
	}
	withSkills := skills != "none"
	data := map[string]string{"Name": in.Name, "Client": in.Client, "Server": in.Server, "Template": in.Client + "-" + in.Server, "Platform": in.Platform, "SDKVersion": sdkVersion(),
		"GodotDir": godotDir, "GodotPrefix": godotPrefix, "GodotPath": godotPath, "AI": map[bool]string{true: "studios", false: ""}[in.Studios],
		"Skills":      map[bool]string{true: skills, false: ""}[withSkills],
		"Orientation": in.Orientation, "OrientationLabel": orientationLabel[in.Orientation],
		"AIMode": in.AIMode, "AIModeLabel": aiModeLabel[in.AIMode], "GameServer": in.GameServer, "Hosting": in.Hosting,
		"PolicyJSON": policyJSON(in.Policy), "PolicyVersion": loadPolicyRules().Version,
		"Multiplayer":   map[bool]string{true: "true", false: ""}[in.GameServer != "none"],
		"SkillsLabel":   map[string]string{"full": "完整（通用＋" + map[string]string{"web": "網頁引擎", "godot": "Godot"}[in.Client] + "）", "core": "通用", "none": ""}[skills],
		"SkillsVersion": in.GDS.Version + "（commit " + short(in.GDS.Commit) + "）", "StudiosVersion": in.CCGS.Version + "（commit " + short(in.CCGS.Commit) + "）"}
	for k, v := range policySummary(in.Policy) {
		data[k] = v
	}
	return data
}

func short(commit string) string {
	if len(commit) > 7 {
		return commit[:7]
	}
	return commit
}

// renderProject 產生專案的範本檔案（不含 SDK 與第三方內容）：相對路徑 → 內容，以及每個檔案的種類。
func renderProject(in genInputs) (files map[string][]byte, kinds map[string]string, err error) {
	files, kinds = map[string][]byte{}, map[string]string{}
	data := in.data()
	render := func(src, dst string, kind func(rel string) string) error {
		return fs2Walk("templates/"+src, func(rel string, content []byte) error {
			out := path.Join(dst, strings.TrimSuffix(rel, ".tmpl"))
			if strings.HasSuffix(rel, ".tmpl") {
				t, err := texttemplate.New(rel).Delims("[[", "]]").Option("missingkey=error").Parse(string(content))
				if err != nil {
					return fmt.Errorf("範本 %s：%w", rel, err)
				}
				var b bytes.Buffer
				if err := t.Execute(&b, data); err != nil {
					return err
				}
				content = b.Bytes()
			}
			files[out] = content
			kinds[out] = kind(strings.TrimSuffix(rel, ".tmpl"))
			return nil
		})
	}
	always := func(k string) func(string) string { return func(string) string { return k } }
	// project：Makefile 與 .gitignore 跟著 pt 走；README、playtogether.json、store/、art/ 是開發者的
	projectKind := func(rel string) string {
		if rel == "Makefile" || rel == "gitignore" {
			return kindTemplate
		}
		return kindStarter
	}
	clientDir := map[string]string{"web": "web", "godot": in.godotDir()}[in.Client]
	parts := []struct {
		src, dst string
		kind     func(string) string
	}{
		{"project", "", projectKind},
		{"ai", "", always(kindTemplate)},
		{"client-" + in.Client, clientDir, always(kindStarter)},
		{"server-" + in.Server, "server", always(kindStarter)},
	}
	if in.Studios {
		parts = append(parts, struct {
			src, dst string
			kind     func(string) string
		}{"ai-studios/docs", "docs", always(kindStarter)}) // CURRENT_TASK.md、DECISIONS.md：工作中的文件
	}
	for _, r := range parts {
		if err := render(r.src, r.dst, r.kind); err != nil {
			return nil, nil, err
		}
	}
	files["docs/content/CONTENT_PROFILE.md"] = []byte(contentProfileDoc(in.Name, in.Policy))
	files["docs/content/GAMEPLAY_ECONOMY.md"] = []byte(gameplayEconomyDoc(in.Name, in.Policy))
	kinds["docs/content/CONTENT_PROFILE.md"], kinds["docs/content/GAMEPLAY_ECONOMY.md"] = kindStarter, kindStarter // pt content sync 管理
	files[".gitignore"], kinds[".gitignore"] = files["gitignore"], kinds["gitignore"]
	delete(files, "gitignore")
	delete(kinds, "gitignore")
	if in.Client == "godot" && in.godotDir() == "." {
		// Godot 專案在根目錄時，素材原始檔不要被匯入、匯出（會變成上傳的一部分）
		files["art/.gdignore"], kinds["art/.gdignore"] = []byte{}, kindStarter
	}
	// pt 自己的 skill（準備上架資料等）：所有 AI 模式都放
	skills, err := ptSkillFiles()
	if err != nil {
		return nil, nil, err
	}
	for dst, content := range skills {
		if _, exists := files[dst]; exists {
			return nil, nil, fmt.Errorf("pt 的 skill %s 和專案裡已有的檔案衝突", dst)
		}
		files[dst], kinds[dst] = content, kindTemplate
	}
	return files, kinds, nil
}
