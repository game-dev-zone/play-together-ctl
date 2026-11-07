package main

// 專案相關的指令（仿 goctl 的快速建立專案）：pt new、pt dev、pt check、pt sdk update。
// 範本（templates/）與 SDK（assets/sdk，由玩聚從 SDK 原始碼同步）都內嵌在 pt 裡，離線也能用，
// 而且範本和 SDK 的版本永遠一致。只用 Go 標準函式庫。

import (
	"bytes"
	"embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strings"
	texttemplate "text/template"
	"time"
)

//go:embed templates assets policy
var embedded embed.FS

const projectFile = "playtogether.json"

// 畫面（client）與伺服器（server）的組合；舊的 -template 名稱對應到這裡
var (
	clients = []string{"web", "godot"}
	servers = []string{"go", "node"}
	// 畫面方向：portrait（直式）、landscape（橫式）、any（兩者都支援）
	orientations = []string{"portrait", "landscape", "any"}
	templates    = map[string][2]string{
		"web-go": {"web", "go"}, "web-node": {"web", "node"}, "godot-go": {"godot", "go"}, "godot-node": {"godot", "node"},
	}
)

var namePattern = regexp.MustCompile(`^[a-z][a-z0-9-]{1,38}[a-z0-9]$`)

// project 是 playtogether.json：pt 的指令都從這裡讀設定，參數可以覆蓋。
type project struct {
	Name          string `json:"name"`
	Client        string `json:"client"` // web 或 godot
	Server        string `json:"server"` // go 或 node
	Template      string `json:"template"`
	SDKVersion    string `json:"sdkVersion"`
	Platform      string `json:"platform"`
	UploadKeyID   string `json:"uploadKeyId"`
	UploadKeyFile string `json:"uploadKeyFile"`
	ServerKeyID   string `json:"serverKeyId"`
	ServerKeyFile string `json:"serverKeyFile"`
	ServerURL     string `json:"serverUrl"`
	Web           string `json:"web"`
	DevAddr       string `json:"devAddr"`
	// Godot 專案的資料夾（client 為 godot）：預設 game；搭配 Claude Code Game Studios 時是 .（專案根目錄）
	Godot string `json:"godot,omitempty"`
	// 畫面方向：portrait、landscape、any；舊專案沒有這個欄位
	Orientation string `json:"orientation,omitempty"`
	AI          string `json:"ai,omitempty"`     // studios：已加入 Claude Code Game Studios
	Skills      string `json:"skills,omitempty"` // core、full：已加入 awesome-gamedev-agent-skills 精選
	AIMode      string `json:"aiMode,omitempty"` // lightweight、standard、studio、custom（v0.13 起；ai_mode.go）
	// 多人與遊戲伺服器架構（v0.13 起；舊專案沒有時視為多人＋playtogether-managed）。AI 改連線前一定要先確認
	Multiplayer *multiplayerConfig `json:"multiplayer,omitempty"`
	GameServer  *gameServerConfig  `json:"gameServer,omitempty"`
	// 建立或最近一次 pt content sync 時的平台政策版本（policy/rules.json 的 version）
	PolicyVersion string `json:"policyVersion,omitempty"`
	// 政策宣告（v0.15 起；policy.go）：讀成原始 JSON，pt check 再嚴格解析（才能抓到缺欄位、未知鍵、型別錯誤）
	PlatformAudience json.RawMessage `json:"platformAudience,omitempty"`
	ContentProfile   json.RawMessage `json:"contentProfile,omitempty"`
	SensitiveContent json.RawMessage `json:"sensitiveContent,omitempty"`
	Gameplay         json.RawMessage `json:"gameplay,omitempty"`
	Economy          json.RawMessage `json:"economy,omitempty"`
	// v0.14 的內容分級（舊格式，content_rating.go）：只有舊專案有，pt check 依舊規則驗證並提醒改用 v0.15
	ContentRating *contentRating `json:"contentRating,omitempty"`
	// 從 CDN 載入的程式與樣式（pt cdn add 加入）；pt upload 申報給平台
	CDN []cdnAsset `json:"cdn,omitempty"`

	dir string // playtogether.json 所在的資料夾
	raw []byte // playtogether.json 的原始內容（檢查重複鍵、拼錯的宣告鍵）
}

// godotDir 是 Godot 專案相對於專案根目錄的位置
func (p *project) godotDir() string {
	if p.Godot == "" {
		return "game"
	}
	return p.Godot
}

// findProject 從目前資料夾往上找 playtogether.json；找不到回 nil（不是錯誤）。
func findProject() (*project, error) {
	dir, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	for {
		raw, err := os.ReadFile(filepath.Join(dir, projectFile))
		if err == nil {
			var p project
			if err := json.Unmarshal(raw, &p); err != nil {
				return nil, fmt.Errorf("%s 格式錯誤：%w", filepath.Join(dir, projectFile), err)
			}
			p.dir, p.raw = dir, raw
			if p.Web == "" {
				p.Web = "web"
			}
			if p.DevAddr == "" {
				p.DevAddr = "127.0.0.1:8090"
			}
			if p.Client == "" || p.Server == "" { // v0.1 建立的專案只有 template
				if cs, ok := templates[p.Template]; ok {
					p.Client, p.Server = cs[0], cs[1]
				}
			}
			return &p, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return nil, nil
		}
		dir = parent
	}
}

func (p *project) path(rel string) string {
	if rel == "" || filepath.IsAbs(rel) {
		return rel
	}
	return filepath.Join(p.dir, rel)
}

func sdkVersion() string {
	raw, err := embedded.ReadFile("assets/sdk/VERSION")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(raw))
}

func requireAssets() error {
	if sdkVersion() == "" {
		return errors.New("這個 pt 沒有內嵌 SDK：請從 https://github.com/game-dev-zone/play-together-ctl/releases 下載 pt")
	}
	return nil
}

// ---------- pt new ----------

func cmdNew(args []string) error {
	fs := flag.NewFlagSet("new", flag.ExitOnError)
	client := fs.String("client", "web", "畫面：web（網頁）或 godot（Godot 4）")
	server := fs.String("server", "go", "遊戲伺服器：go 或 node")
	tmpl := fs.String("template", "", "舊的寫法：web-go、web-node、godot-go、godot-node（等於 -client 加 -server）")
	platform := fs.String("platform", "https://（平台網址）", "玩聚平台網址（之後可以在 playtogether.json 修改）")
	mode := fs.String("mode", "", "AI 開發模式：standard（建議：平台規則＋遊戲開發 skill）、studio（再加上規劃、故事、設計流程；只限 Godot）、lightweight（只有平台規則）、custom（用 -skills、-ai 自己指定）；在終端機執行且沒指定時會詢問，-y、非互動與 CI 預設 lightweight")
	ai := fs.String("ai", "", "自訂（-mode custom）：studios 加入 Claude Code Game Studios（只限 Godot、Godot 專案改放根目錄）或 none")
	skills := fs.String("skills", "", "自訂（-mode custom）：AI 遊戲開發 skill（awesome-gamedev-agent-skills 精選）core、full（core 加上畫面引擎的 skill）或 none")
	gameServer := fs.String("game-server", "", "遊戲伺服器架構：playtogether-managed（預設：玩聚範本伺服器）、external（你已經有自己的遊戲伺服器）、none（單人或畫面端就能玩，正式遊玩不需要遊戲伺服器）")
	orientation := fs.String("orientation", "portrait", "畫面方向：portrait（直式）、landscape（橫式）或 any（兩者都支援）")
	profile := fs.String("profile", "", "內容邊界（開發期間的內容上限，不是正式分級）：general（預設）、moderate、strong、custom（在終端機逐項設定）")
	rating := fs.String("rating", "", "舊的寫法（v0.14）：general、12plus（= moderate）、15plus（= strong）、custom；請改用 -profile")
	table := fs.String("table-game", "none", "牌桌玩法：none（預設）、card、poker、mahjong、dice、board、slot、roulette、baccarat、other（是否能上架由玩聚平台審核決定）")
	yes := fs.Bool("y", false, "不詢問，沒指定的項目用預設值")
	_ = fs.Parse(reorder(args))
	set := map[string]bool{}
	fs.Visit(func(f *flag.Flag) { set[f.Name] = true })
	if fs.NArg() > 1 {
		return errors.New(newUsage)
	}
	checkName := func(name string) error {
		if !namePattern.MatchString(name) {
			return errors.New("專案名稱只能用小寫英文、數字與連字號（3–40 字，英文開頭），例如 my-party-game")
		}
		if _, err := os.Stat(name); err == nil {
			return fmt.Errorf("資料夾 %s 已經存在", name)
		}
		return nil
	}
	if *tmpl != "" {
		cs, ok := templates[*tmpl]
		if !ok {
			return fmt.Errorf("沒有範本 %q：可用 web-go、web-node、godot-go、godot-node", *tmpl)
		}
		*client, *server = cs[0], cs[1]
		set["client"], set["server"] = true, true
	}
	if !slices.Contains(clients, *client) {
		return fmt.Errorf("-client 只能是 web 或 godot（目前是 %q）", *client)
	}
	if !slices.Contains(servers, *server) {
		return fmt.Errorf("-server 只能是 go 或 node（目前是 %q）", *server)
	}
	if !slices.Contains(orientations, *orientation) {
		return fmt.Errorf("-orientation 只能是 portrait（直式）、landscape（橫式）或 any（兩者都支援）（目前是 %q）", *orientation)
	}
	if *ai != "" && *ai != "studios" && *ai != "none" {
		return fmt.Errorf("-ai 只能是 studios 或 none（目前是 %q）", *ai)
	}
	if *skills != "" && !slices.Contains(skillLevels, *skills) {
		return fmt.Errorf("-skills 只能是 none、core 或 full（目前是 %q）", *skills)
	}
	if *gameServer != "" && !slices.Contains(gameServerModes, *gameServer) {
		return fmt.Errorf("-game-server 只能是 playtogether-managed、external 或 none（目前是 %q）", *gameServer)
	}
	if *rating != "" {
		if *profile != "" {
			return errors.New("-profile 和 -rating 只能指定一個（-rating 是舊的寫法）")
		}
		n, ok := legacyProfileNames[*rating]
		if !ok {
			return fmt.Errorf("-rating 只能是 general、12plus、15plus 或 custom（目前是 %q）；請改用 -profile general|moderate|strong|custom", *rating)
		}
		if n != *rating {
			fmt.Printf("提醒：-rating %s 是舊的寫法，改用 -profile %s\n", *rating, n)
		}
		*profile = n
	}
	if *profile != "" && !slices.Contains(profiles, *profile) {
		return fmt.Errorf("-profile 只能是 general、moderate、strong 或 custom（目前是 %q）", *profile)
	}
	if !slices.Contains(tableCategories, *table) {
		return fmt.Errorf("-table-game 只能是 %s（目前是 %q）", strings.Join(tableCategories, "、"), *table)
	}
	aiMode, aiVal, skillsVal, err := resolveAIMode(*mode, *ai, *skills, set)
	if err != nil {
		return err
	}
	*ai, *skills = aiVal, skillsVal
	if *ai == "studios" && *client != "godot" {
		if set["client"] {
			if aiMode == "studio" {
				return errors.New("工作室模式（studio）目前只支援 Godot：請加上 -client godot，網頁專案建議用 -mode standard")
			}
			return errors.New("Claude Code Game Studios 是遊戲引擎專用的範本，目前只搭配 Godot：請加上 -client godot（網頁專案已經附上給 AI 助手的 CLAUDE.md、AGENTS.md）")
		}
		*client, set["client"] = "godot", true // 只指定 studio／-ai studios：一定是 Godot
	}
	if err := requireAssets(); err != nil {
		return err
	}
	ccgs, err := loadCCGS()
	if err != nil {
		return err
	}
	gds, err := loadSkills()
	if err != nil {
		return err
	}

	// 終端機裡逐步詢問沒用參數指定的項目
	name := fs.Arg(0)
	if name != "" {
		if err := checkName(name); err != nil {
			return err
		}
	}
	canAsk := !*yes && interactive() && !ciEnv()
	var cp *contentProfile
	if canAsk && (name == "" || !set["client"] || !set["server"] || !set["orientation"] || aiMode == "" || !set["game-server"] || *profile == "" || *profile == "custom") {
		if name, err = newWizard(name, client, server, orientation, gameServer, profile, &cp, *table, &aiMode, ai, skills, set, ccgs, gds, checkName); err != nil {
			if errors.Is(err, errCanceled) {
				fmt.Println()
				return errors.New("已取消，沒有建立任何檔案")
			}
			return err
		}
		if name == "" {
			return nil
		}
	}
	if name == "" {
		return errors.New(newUsage + "（在終端機執行 pt new 會逐步詢問）")
	}
	if aiMode == "" {
		// -y、非互動、CI：不下載任何第三方內容（保留既有 CI 行為、離線產生、結果固定）
		aiMode = "lightweight"
		*ai, *skills = modeComponents(aiMode)
	}
	if *gameServer == "" {
		*gameServer = defaultGameServer
	}
	if cp == nil {
		// -y、非互動、CI：沒指定時用預設（和精靈的預設相同）；custom 要逐項設定，不能用某個預設頂替
		if *profile == "" {
			*profile = defaultProfile
		}
		if *profile == "custom" {
			return errors.New("-profile custom 需要在終端機逐項設定；或先用 general、moderate、strong 建立，再把 playtogether.json 的 contentProfile 改成 custom 並調整")
		}
		c := presetProfile(*profile)
		cp = &c
	}
	pl := defaultPolicy(*cp, *table)
	studios := *ai == "studios"
	if *ai == "" {
		*ai = "none"
	}
	if *skills == "" {
		*skills = "none"
	}
	withSkills := *skills != "none"
	godotDir := "game"
	if studios {
		godotDir = "." // Claude Code Game Studios 的工具假設 project.godot 在根目錄
	}
	godotPrefix := godotDir + "/"
	if godotDir == "." {
		godotPrefix = ""
	}
	godotPath := godotPrefix
	if godotPath == "" {
		godotPath = "./"
	}
	template := *client + "-" + *server
	data := map[string]string{"Name": name, "Client": *client, "Server": *server, "Template": template, "Platform": *platform, "SDKVersion": sdkVersion(),
		"GodotDir": godotDir, "GodotPrefix": godotPrefix, "GodotPath": godotPath, "AI": map[bool]string{true: "studios", false: ""}[studios],
		"Skills":      map[bool]string{true: *skills, false: ""}[withSkills],
		"Orientation": *orientation, "OrientationLabel": orientationLabel[*orientation],
		"AIMode": aiMode, "AIModeLabel": aiModeLabel[aiMode], "GameServer": *gameServer,
		"PolicyJSON": policyJSON(pl), "PolicyVersion": loadPolicyRules().Version,
		"Multiplayer":   map[bool]string{true: "true", false: ""}[*gameServer != "none"],
		"SkillsLabel":   map[string]string{"full": "完整（通用＋" + map[string]string{"web": "網頁引擎", "godot": "Godot"}[*client] + "）", "core": "通用", "none": ""}[*skills],
		"SkillsVersion": gds.Version + "（commit " + gds.Commit[:7] + "）", "StudiosVersion": ccgs.Version + "（commit " + ccgs.Commit[:7] + "）"}
	for k, v := range policySummary(pl) {
		data[k] = v
	}
	files := map[string][]byte{}
	render := func(src, dst string) error {
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
			return nil
		})
	}
	clientDir := map[string]string{"web": "web", "godot": godotDir}[*client]
	parts := [][2]string{{"project", ""}, {"ai", ""}, {"client-" + *client, clientDir}, {"server-" + *server, "server"}}
	if studios {
		parts = append(parts, [2]string{"ai-studios/docs", "docs"}) // CURRENT_TASK.md、DECISIONS.md
	}
	for _, r := range parts {
		if err := render(r[0], r[1]); err != nil {
			return err
		}
	}
	files["docs/content/CONTENT_PROFILE.md"] = []byte(contentProfileDoc(name, pl))
	files["docs/content/GAMEPLAY_ECONOMY.md"] = []byte(gameplayEconomyDoc(name, pl))
	files[".gitignore"] = files["gitignore"]
	delete(files, "gitignore")
	if *client == "godot" && godotDir == "." {
		// Godot 專案在根目錄時，素材原始檔不要被匯入、匯出（會變成上傳的一部分）
		files["art/.gdignore"] = []byte{}
	}
	sdk, err := sdkFiles(*client, *server, godotDir)
	if err != nil {
		return err
	}
	for k, v := range sdk {
		files[k] = v
	}
	// pt 自己的 skill（準備上架資料等）：所有 AI 模式都放；先放進來，第三方 skill 撞名時會停止
	if err := addPtSkills(files); err != nil {
		return err
	}
	if studios {
		// 先下載並驗證完，才開始寫入檔案（失敗時不留下半個專案）
		fmt.Printf("下載 %s %s（commit %s）並驗證每個檔案…\n", ccgs.Name, ccgs.Version, ccgs.Commit[:7])
		studio, err := fetchCCGS(ccgs)
		if err != nil {
			return err
		}
		if err := addStudiosFiles(files, ccgs, studio); err != nil {
			return err
		}
	}
	var skillNames []string
	if withSkills {
		fmt.Printf("下載 %s（commit %s）並驗證每個檔案…\n", gds.Name, gds.Commit[:7])
		got, err := fetchPinned(gds, "PT_SKILLS_URL")
		if err != nil {
			return err
		}
		if skillNames, err = addSkillsFiles(files, gds, got, *skills, *client); err != nil {
			return err
		}
	}
	for rel, content := range files {
		dst := filepath.Join(name, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(dst, content, 0o644); err != nil {
			return err
		}
	}
	if err := os.MkdirAll(filepath.Join(name, "keys"), 0o700); err != nil {
		return err
	}
	fmt.Printf("已建立 %s（畫面 %s、伺服器 %s、%s、%s、內容邊界 %s、AI 模式 %s、SDK %s）\n\n", name, *client, *server, orientationLabel[*orientation], gameServerLabel[*gameServer], profileLabel[pl.Content.Profile], aiModeLabel[aiMode], sdkVersion())
	if _, review := reviewReasons(pl); len(review) > 0 {
		fmt.Printf("需要玩聚平台人工審核（可以開發與送審，不代表已批准）：%s\n\n", strings.Join(review, "；"))
	}
	fmt.Println("下一步：")
	fmt.Printf("  cd %s\n", name)
	if *client == "godot" {
		fmt.Printf("  pt dev        # 匯出 Godot 網頁版，啟動本機假平台並打開%s與控制面板（需要 Godot 4.3 以上與網頁匯出範本）\n", devTabsLabel[*gameServer != "none"])
		if studios {
			fmt.Println("  用 Godot 編輯器打開專案根目錄修改遊戲")
		} else {
			fmt.Println("  用 Godot 編輯器打開 game/ 修改遊戲")
		}
	} else {
		fmt.Printf("  pt dev        # 啟動遊戲伺服器與本機假平台，打開%s與控制面板\n", devTabsLabel[*gameServer != "none"])
	}
	fmt.Println("  make verify   # 啟動驗證：伺服器測試、檢查要上傳的檔案、在模擬模式實際啟動一次（make 列出所有指令）")
	fmt.Println("接上玩聚的步驟見 README.md。")
	if *gameServer != defaultGameServer {
		fmt.Printf("server/ 是玩聚的%s，正式遊玩不會用到它，不需要部署（見 docs/ai/PROJECT_CONTEXT.md）。\n",
			map[string]string{"none": "本機模擬與參考伺服器", "external": "整合參考與本機模擬伺服器，不是你的正式遊戲伺服器"}[*gameServer])
	}
	if withSkills {
		fmt.Printf("已加入 %d 個遊戲開發 skill。\n", len(skillNames))
	}
	fmt.Println()
	if studios {
		fmt.Println("在專案根目錄執行 claude（Claude Code）開始；也可以用 Codex 等工具（讀 AGENTS.md）。")
	} else {
		fmt.Println("用 Claude Code（claude）、Codex 等 AI 工具在專案資料夾開始。")
	}
	fmt.Println(firstRunHint)
	return nil
}

// newWizard 逐步詢問專案名稱、畫面、多人與遊戲伺服器、伺服器語言、方向、內容分級、AI 開發模式，最後確認。回傳空名稱表示開發者選擇不建立。
var (
	orientationLabel = map[string]string{"portrait": "直式", "landscape": "橫式", "any": "直式橫式都支援"}
	aiModeLabel      = map[string]string{"lightweight": "輕量", "standard": "標準", "studio": "工作室", "custom": "自訂"}
	devTabsLabel     = map[bool]string{true: "兩個玩家分頁", false: "一個玩家分頁"}
	gameServerLabel  = map[string]string{"playtogether-managed": "多人（玩聚範本伺服器）", "external": "多人（自己的遊戲伺服器）", "none": "不需要遊戲伺服器"}
)

const newUsage = "用法：pt new [-client web|godot] [-server go|node] [-orientation portrait|landscape|any] [-game-server playtogether-managed|external|none] [-profile general|moderate|strong|custom] [-table-game none|card|poker|mahjong|dice|board|slot|roulette|baccarat|other] [-mode standard|studio|lightweight|custom] [-skills none|core|full] [-ai studios|none] <專案名稱>"

func newWizard(name string, client, server, orientation, gameServer, profile *string, cp **contentProfile, table string, aiMode, ai, skills *string, set map[string]bool, ccgs, gds ccgsManifest, checkName func(string) error) (string, error) {
	var err error
	fmt.Println("建立玩聚遊戲專案（直接按 Enter 用預設值，Ctrl-C 取消）")
	fmt.Println()
	if name == "" {
		if name, err = askText("專案名稱（小寫英文、數字、連字號，例如 my-party-game）", checkName); err != nil {
			return "", err
		}
		fmt.Println()
	}
	if !set["client"] {
		if *client, err = choose("遊戲畫面要用什麼做？", []option{
			{"web", "網頁（HTML／JavaScript）", "直接用瀏覽器開發，適合 2D、介面為主的派對遊戲"},
			{"godot", "Godot 4", "用 Godot 編輯器開發，匯出網頁版；需要 Godot 4.3 以上"},
		}, "web"); err != nil {
			return "", err
		}
	}
	if !set["game-server"] {
		multi, err := choose("這是多人一起玩的遊戲嗎？", []option{
			{"yes", "是，多人遊戲", "玩家之間即時互動、比分數；由遊戲伺服器判定結果"},
			{"no", "不是，單人或畫面端就能玩", "解謎、敘事、冒險、節奏、休閒等；正式遊玩不需要遊戲伺服器"},
		}, "yes")
		if err != nil {
			return "", err
		}
		*gameServer = "none"
		if multi == "yes" {
			if *gameServer, err = choose("多人連線用哪種遊戲伺服器？", []option{
				{"playtogether-managed", "玩聚範本伺服器（建議）", "專案附的 server/：兌換玩家身分、判定勝負、斷線重連都已經寫好"},
				{"external", "我已經有自己的遊戲伺服器", "保留你的伺服器架構，只補上玩聚需要的整合；server/ 只當參考與本機模擬"},
			}, "playtogether-managed"); err != nil {
				return "", err
			}
		}
	}
	if !set["server"] {
		q := "遊戲伺服器（兌換玩家身分、判定勝負）要用哪種語言？"
		if *gameServer != defaultGameServer {
			q = "專案附的參考伺服器（本機模擬用）要用哪種語言？"
		}
		if *server, err = choose(q, []option{
			{"go", "Go", ""},
			{"node", "Node.js", ""},
		}, "go"); err != nil {
			return "", err
		}
	}
	if !set["orientation"] {
		if *orientation, err = choose("玩家要用直式還是橫式玩？", []option{
			{"portrait", "直式", "手機直拿，單手就能玩；派對遊戲、答題、卡牌建議用直式"},
			{"landscape", "橫式", "手機橫拿，畫面較寬；動作、賽車、需要大地圖的遊戲"},
			{"any", "直式橫式都支援", "畫面要能隨手機方向調整版面，開發和測試的工作量較大"},
		}, "portrait"); err != nil {
			return "", err
		}
	}
	if *profile == "" || *profile == "custom" {
		if *cp, err = askContentProfile(profile); err != nil {
			return "", err
		}
	} else {
		c := presetProfile(*profile)
		*cp = &c
	}
	if *aiMode == "" {
		engine := map[string]string{"web": "Phaser、PixiJS、three.js", "godot": "Godot"}[*client]
		opts := []option{{"standard", "標準（建議）", "大部分遊戲適合：平台規則＋手感、介面、音效、素材、效能與 " + engine + " 的實作知識。Claude Code、Codex 都能用，額外 token 很少"}}
		if *client == "godot" {
			opts = append(opts, option{"studio", "工作室", "故事、謎題、角色扮演或規模較大的遊戲：先規劃、設計再實作。流程較重、token 用量大，完整流程需要 Claude Code，會安裝在 Claude Code 裡執行的檢查程式（hooks）"})
		}
		opts = append(opts, option{"lightweight", "輕量", "只有平台規則"}, option{"custom", "自訂", "進階：自己選擇遊戲開發 skill 的範圍" + map[bool]string{true: "與是否加入 Claude Code Game Studios", false: ""}[*client == "godot"]})
		if *client != "godot" {
			fmt.Println("（工作室模式目前只支援 Godot）")
		}
		if *aiMode, err = choose("要讓 AI 怎麼幫你開發這個遊戲？", opts, "standard"); err != nil {
			return "", err
		}
		*ai, *skills = modeComponents(*aiMode)
		if *aiMode == "custom" {
			engineSkill := map[string]string{"web": "Phaser、PixiJS、three.js", "godot": "Godot 4.7"}[*client]
			if *skills, err = choose("要加入哪些遊戲開發 skill？（"+gds.Name+" 精選，Claude Code 與 Codex 都能用）", []option{
				{"full", "完整", "手感、手機介面、觸控輸入、音效、素材風格一致、效能，加上 " + engineSkill + " 的 skill；只收說明文件、逐檔驗證"},
				{"core", "只加通用的", "不含引擎的 skill"},
				{"none", "不加入", ""},
			}, "full"); err != nil {
				return "", err
			}
			*ai = "none"
			if *client == "godot" {
				with, err := confirm("要加入 "+ccgs.Name+"（規劃、故事、設計流程；只支援 Claude Code，Godot 專案改放根目錄）嗎？", false)
				if err != nil {
					return "", err
				}
				if with {
					*ai = "studios"
				}
			}
		}
	}
	if *gameServer == "" {
		*gameServer = defaultGameServer
	}
	flags := ""
	if *aiMode == "custom" {
		flags = " -mode custom -skills " + *skills
		if *ai == "studios" {
			flags += " -ai studios"
		}
	} else {
		flags = " -mode " + *aiMode
	}
	if table != "none" {
		flags += " -table-game " + table
	}
	cmd := fmt.Sprintf("pt new -client %s -server %s -orientation %s -game-server %s -profile %s%s %s", *client, *server, *orientation, *gameServer, *profile, flags, name)
	fmt.Printf("即將建立 %s：畫面 %s、%s、伺服器 %s、%s、內容邊界 %s、AI 模式 %s\n", name, *client, gameServerLabel[*gameServer], *server, orientationLabel[*orientation], profileLabel[*profile], aiModeLabel[*aiMode])
	fmt.Printf("（下次可以直接執行：%s）\n", cmd)
	ok, err := confirm("確定建立？", true)
	if err != nil {
		return "", err
	}
	if !ok {
		fmt.Println("沒有建立任何檔案。")
		return "", nil
	}
	fmt.Println()
	return name, nil
}

// sdkFiles：專案需要的 SDK 檔案（相對專案根目錄的路徑 → 內容）。pt new 與 pt sdk update 共用。
func sdkFiles(client, server, godotDir string) (map[string][]byte, error) {
	out := map[string][]byte{}
	switch client {
	case "web":
		iife, err := embedded.ReadFile("assets/sdk/playtogether-sdk.iife.js")
		if err != nil {
			return nil, err
		}
		out["web/playtogether-sdk.iife.js"] = iife
	case "godot":
		// 外掛放在 Godot 專案的 addons/；轉接檔匯出時才複製到 build/（pt build）
		if err := fs2Walk("assets/sdk/godot/addons", func(rel string, content []byte) error {
			out[path.Join(godotDir, "addons", rel)] = content
			return nil
		}); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("不認得的畫面類型 %q", client)
	}
	switch server {
	case "none": // pt init 接入的既有專案：伺服器由開發者自己管理
	case "go":
		if err := fs2Walk("assets/sdk/go", func(rel string, content []byte) error {
			out["server/third_party/playtogether-sdk-go/"+strings.TrimSuffix(rel, ".txt")] = content
			return nil
		}); err != nil {
			return nil, err
		}
	case "node":
		mjs, err := embedded.ReadFile("assets/sdk/playtogether-server.mjs")
		if err != nil {
			return nil, err
		}
		out["server/playtogether-server.mjs"] = mjs
	default:
		return nil, fmt.Errorf("不認得的伺服器 %q", server)
	}
	return out, nil
}

func fs2Walk(root string, fn func(rel string, content []byte) error) error {
	return fs.WalkDir(embedded, root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		content, err := embedded.ReadFile(p)
		if err != nil {
			return err
		}
		return fn(strings.TrimPrefix(p, root+"/"), content)
	})
}

// reorder 讓旗標可以放在位置參數後面（pt new my-game -template web-node）。
func reorder(args []string) []string {
	var flags, rest []string
	for i := 0; i < len(args); i++ {
		if strings.HasPrefix(args[i], "-") {
			flags = append(flags, args[i])
			if !strings.Contains(args[i], "=") && i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				flags = append(flags, args[i+1])
				i++
			}
			continue
		}
		rest = append(rest, args[i])
	}
	return append(flags, rest...)
}

// ---------- pt sdk update ----------

func cmdSDK(args []string) error {
	if len(args) == 2 && args[0] == "export" {
		return exportSDK(args[1])
	}
	if len(args) == 0 || (len(args) == 1 && args[0] == "status") {
		if err := requireAssets(); err != nil {
			return err
		}
		p, err := findProject()
		if err != nil {
			return err
		}
		fmt.Print(sdkStatus(p, true))
		if hint := legacyServerHint(p); hint != "" {
			fmt.Println(hint)
		}
		return nil
	}
	force := len(args) == 2 && args[0] == "update" && args[1] == "-force"
	if !force && (len(args) != 1 || args[0] != "update") {
		return errors.New("用法：pt sdk status（檢查有沒有新版）、pt sdk update（在 pt new 建立的專案裡）或 pt sdk export <資料夾>（把 SDK 檔案寫到資料夾，給既有專案用）")
	}
	p, err := findProject()
	if err != nil {
		return err
	}
	if p == nil {
		return errors.New("找不到 playtogether.json：請在用 pt new 建立的專案裡執行")
	}
	if p.existing() {
		return errors.New(existingSDKNote)
	}
	if err := requireAssets(); err != nil {
		return err
	}
	// 不要用舊的 pt 把專案的 SDK 降版
	if c, ok := compareSDK(p.SDKVersion, sdkVersion()); ok && c > 0 && !force {
		return fmt.Errorf("專案的 SDK %s 比這個 pt 內嵌的 %s 新，不會降版：請先更新 pt（go install %s@latest）；確定要降版請加 -force", p.SDKVersion, sdkVersion(), installPath)
	}
	files, err := sdkFiles(p.Client, p.Server, p.godotDir())
	if err != nil {
		return err
	}
	for rel, content := range files {
		if err := os.MkdirAll(filepath.Dir(p.path(rel)), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(p.path(rel), content, 0o644); err != nil {
			return err
		}
	}
	old := p.SDKVersion
	p.SDKVersion = sdkVersion()
	if err := p.save(); err != nil {
		return err
	}
	fmt.Printf("SDK 已更新：%s → %s（%d 個檔案）\n", old, p.SDKVersion, len(files))
	if hint := legacyServerHint(p); hint != "" {
		fmt.Println(hint)
	}
	return nil
}

// legacyServerHint：v0.18 之前的範本把平台接口寫在 server/session.go（session.mjs）裡，pt sdk update 更新不到。
func legacyServerHint(p *project) string {
	if p == nil {
		return ""
	}
	name := map[string]string{"go": "session.go", "node": "session.mjs"}[p.Server]
	if name == "" {
		return ""
	}
	if _, err := os.Stat(p.path(filepath.Join("server", name))); err != nil {
		return ""
	}
	return fmt.Sprintf("提醒：server/%s 是舊範本的平台接口，pt sdk update 更新不到它（限流、413、/healthz 等之後的修正都不會套用）。"+
		"建議改用 SDK 的遊戲伺服器套件，步驟見玩聚開發者文件站 /docs/game-server「舊專案改用套件」。", name)
}

func (p *project) save() error {
	raw, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(p.dir, projectFile), append(raw, '\n'), 0o644)
}

// ---------- pt dev ----------

func cmdDev(args []string) error {
	fs := flag.NewFlagSet("dev", flag.ExitOnError)
	addr := fs.String("addr", "", "本機位址（預設 playtogether.json 的 devAddr）")
	players := fs.Int("players", -1, "要打開幾個玩家分頁（預設：多人專案 2、單人專案 1；0 表示不開瀏覽器）")
	smoke := fs.Bool("smoke", false, "啟動驗證：啟動遊戲伺服器、確認首頁與 SDK 載入得到，然後結束（不開瀏覽器）")
	mock := fs.Bool("mock", false, "改用舊的模擬模式（不經過本機假平台、不簽章；之後的版本會移除）")
	noServer := fs.Bool("no-server", false, "只啟動本機假平台與控制面板，不啟動專案的伺服器（用自己的遊戲伺服器時）")
	site := fs.String("site", "", "搭配 -no-server：遊戲首頁網址（預設 http://<devAddr>/）")
	_ = fs.Parse(args)
	p, err := findProject()
	if err != nil {
		return err
	}
	if p == nil {
		return errors.New("找不到 playtogether.json：請在用 pt new 建立的專案裡執行")
	}
	*players = devPlayers(p, *players)
	if *addr == "" && *smoke {
		// 啟動驗證用隨機的空埠，不和正在跑的 pt dev 搶
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			return err
		}
		*addr = ln.Addr().String()
		ln.Close()
	}
	if *addr == "" {
		*addr = p.DevAddr
	}
	if *noServer {
		return devPlatformOnly(p, *site, *addr, *players)
	}
	if ln, err := net.Listen("tcp", *addr); err != nil {
		return fmt.Errorf("%s 已經被其他程式使用，請換一個：pt dev -addr 127.0.0.1:18090", *addr)
	} else {
		ln.Close()
	}
	printSDKNotice(p)
	if p.Client == "godot" {
		if err := buildGodot(p); err != nil {
			return err
		}
	}
	var cmd *exec.Cmd
	switch p.Server {
	case "go":
		cmd = exec.Command("go", "run", ".")
		if *smoke {
			// 先編譯再直接執行：go run 不會把結束訊號轉給伺服器
			tmp, err := os.MkdirTemp("", "pt-smoke-")
			if err != nil {
				return err
			}
			defer os.RemoveAll(tmp)
			bin := filepath.Join(tmp, "server")
			if runtime.GOOS == "windows" {
				bin += ".exe"
			}
			build := exec.Command("go", "build", "-o", bin, ".")
			build.Dir = p.path("server")
			build.Stdout, build.Stderr = os.Stdout, os.Stderr
			if err := build.Run(); err != nil {
				return fmt.Errorf("遊戲伺服器編譯失敗：%w", err)
			}
			cmd = exec.Command(bin)
		}
	case "node":
		cmd = exec.Command("node", "main.mjs")
	case "none":
		return fmt.Errorf("這個專案的遊戲伺服器由你自己管理：先啟動你的伺服器與前端，再用 pt dev -no-server -site <你的遊戲網址>（本機假平台會印出伺服器要設定的環境變數）")
	default:
		return fmt.Errorf("pt dev 只支援 go、node 伺服器（目前是 %q）", p.Server)
	}
	cmd.Dir = p.path("server")
	site0 := "http://" + *addr + "/"
	// 前端由遊戲伺服器直接提供（同一個來源）。預設接本機假平台（驗證簽章、可以模擬踢人與關房）；
	// 啟動驗證（-smoke）與 -mock 用舊的模擬模式：不需要金鑰、不連任何平台
	cmd.Env = append(os.Environ(), "ADDR="+*addr, "WEB_DIR="+p.path(p.Web))
	platform := ""
	keyNote := ""
	if *smoke || *mock {
		cmd.Env = append(cmd.Env, "PT_MOCK=1")
	} else {
		keyFile, pub, local, cleanup, err := devServerKey(p)
		if err != nil {
			return fmt.Errorf("遊戲伺服器金鑰：%w", err)
		}
		defer cleanup()
		fp := newFakePlatform(pub)
		var srv *http.Server
		platform, srv, err = startFakePlatform(fp, site0)
		if err != nil {
			return fmt.Errorf("啟動本機假平台失敗：%w", err)
		}
		defer srv.Close()
		cmd.Env = append(cmd.Env, devServerEnv(platform, keyFile)...)
		aiEnv, aiCleanup, err := attachDevAI(fp, p, platform)
		if err != nil {
			return fmt.Errorf("本機 AI 閘道：%w", err)
		}
		defer aiCleanup()
		cmd.Env = append(cmd.Env, aiEnv...)
		keyNote = "專案的 " + p.ServerKeyFile
		if local {
			keyNote = "只供本機的暫存金鑰（沒有 " + p.ServerKeyFile + "；不會登記到玩聚）"
		}
	}
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("啟動遊戲伺服器失敗：%w", err)
	}
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	siteURL := site0
	ready := false
	for i := 0; i < 300 && !ready; i++ { // go run 第一次要編譯，最多等 60 秒
		select {
		case err := <-done:
			return fmt.Errorf("遊戲伺服器結束了：%v", err)
		case <-time.After(200 * time.Millisecond):
		}
		if res, err := http.Get(siteURL); err == nil {
			res.Body.Close()
			ready = true
		}
	}
	if !ready {
		_ = cmd.Process.Kill()
		return errors.New("遊戲伺服器 60 秒內沒有啟動")
	}
	if *smoke {
		err := smokeCheck(siteURL)
		stopServer(cmd)
		<-done
		if err != nil {
			return fmt.Errorf("啟動驗證沒有通過：%w", err)
		}
		fmt.Println("✓ 啟動驗證通過：遊戲伺服器在模擬模式啟動，首頁與玩聚 SDK 都載入得到")
		return nil
	}
	if platform == "" {
		fmt.Printf("\n本機模擬模式：%s（每開一個分頁就是一位玩家；Ctrl-C 結束）\n", siteURL)
		for i := 0; i < *players; i++ {
			openBrowser(siteURL)
		}
	} else {
		fmt.Printf("\n本機假平台：%s（控制面板：踢人、重新開始、關房、平台連不到、時鐘偏差、呼叫紀錄）\n", platform)
		fmt.Printf("玩家分頁：%s（換 pt_player 就是另一位玩家；直接開 %s 不會經過假平台，會兌換失敗）\n", devPlayerURL(siteURL, platform, "p1"), siteURL)
		fmt.Printf("簽章金鑰：%s\n", keyNote)
		fmt.Println("重新整理玩家分頁會回到原座位；在控制面板新增玩家。Ctrl-C 結束")
		if *players > 0 {
			openBrowser(platform + "/")
		}
		for i := 1; i <= *players; i++ {
			openBrowser(devPlayerURL(siteURL, platform, fmt.Sprintf("p%d", i)))
		}
	}
	select {
	case <-stop:
		_ = cmd.Process.Signal(os.Interrupt)
		<-done
		return nil
	case err := <-done:
		return err
	}
}

// smokeCheck 確認首頁回 200、有載入玩聚 SDK，而且 SDK 檔案本身也拿得到。
func smokeCheck(site string) error {
	get := func(u string) (string, error) {
		res, err := http.Get(u)
		if err != nil {
			return "", err
		}
		defer res.Body.Close()
		b, err := io.ReadAll(io.LimitReader(res.Body, 4<<20))
		if err != nil {
			return "", err
		}
		if res.StatusCode != http.StatusOK {
			return "", fmt.Errorf("%s 回應 %d", u, res.StatusCode)
		}
		return string(b), nil
	}
	html, err := get(site)
	if err != nil {
		return err
	}
	sdk := ""
	for _, m := range scriptSrc.FindAllStringSubmatch(html, -1) {
		if strings.Contains(m[1], "playtogether-sdk") {
			sdk = m[1]
		}
	}
	if sdk == "" {
		return errors.New("首頁沒有載入玩聚 SDK（playtogether-sdk.iife.js 或 playtogether-sdk-godot.iife.js）")
	}
	u, err := url.Parse(site)
	if err != nil {
		return err
	}
	ref, err := url.Parse(sdk)
	if err != nil {
		return err
	}
	if _, err := get(u.ResolveReference(ref).String()); err != nil {
		return fmt.Errorf("玩聚 SDK 載入失敗：%w", err)
	}
	return nil
}

// stopServer：先請伺服器自己結束（Windows 不支援送 Ctrl-C，直接結束）。
func stopServer(cmd *exec.Cmd) {
	if runtime.GOOS == "windows" || cmd.Process.Signal(os.Interrupt) != nil {
		_ = cmd.Process.Kill()
	}
}

func openBrowser(u string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", u)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", u)
	default:
		cmd = exec.Command("xdg-open", u)
	}
	_ = cmd.Start()
}

// ---------- pt check ----------

type finding struct {
	level string // hard、schema、build、review、warn、info（policy.go 的 level 常數）
	msg   string
}

var (
	scriptSrc     = regexp.MustCompile(`(?i)<script[^>]+src=["']([^"']+)["']`)
	privateKeyPEM = regexp.MustCompile(`-----BEGIN [A-Z ]*PRIVATE KEY-----`)
	trackers      = []string{"googletagmanager.com", "google-analytics.com", "doubleclick.net", "googlesyndication.com", "connect.facebook.net",
		"hotjar.com", "clarity.ms", "mixpanel.com", "segment.com", "amplitude.com", "adsbygoogle"}
)

// checkProject：上傳前在本機先擋掉常見的退件與設定錯誤（對照文件站的審核準則與安全要求）。
func checkProject(p *project, webDir string) []finding {
	var out []finding
	add := func(level, format string, a ...any) { out = append(out, finding{level, fmt.Sprintf(format, a...)}) }

	index, err := os.ReadFile(filepath.Join(webDir, "index.html"))
	if err != nil {
		if p != nil && p.Client == "godot" {
			add(levelBuild, "還沒有匯出網頁版：先執行 pt build（pt upload 會自動執行）")
		} else {
			add(levelBuild, "%s 根目錄沒有 index.html", webDir)
		}
	} else {
		html := string(index)
		if !strings.Contains(html, "playtogether-sdk") {
			add(levelBuild, "index.html 沒有載入玩聚 SDK（playtogether-sdk.iife.js 或 playtogether-sdk-godot.iife.js）")
		}
	}
	// 外部程式與樣式：一律要申報、寫 SRI（cdn.go）
	checkCDN(p, webDir, add)
	// 玩聚代管的遊戲伺服器（serverbundle.go）
	checkServerHosting(p, add)

	var total int64
	files := 0
	_ = filepath.WalkDir(webDir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		files++
		info, err := d.Info()
		if err == nil {
			total += info.Size()
		}
		rel, _ := filepath.Rel(webDir, p)
		base := strings.ToLower(d.Name())
		if strings.HasSuffix(base, ".key") || strings.HasSuffix(base, ".pem") || base == ".env" || strings.HasPrefix(base, ".env.") {
			add(levelHard, "上傳資料夾裡有私鑰或密鑰檔案：%s（不能上傳）", rel)
		}
		ext := strings.ToLower(filepath.Ext(base))
		if ext != ".js" && ext != ".mjs" && ext != ".html" {
			return nil
		}
		raw, err := os.ReadFile(p)
		if err != nil {
			return nil
		}
		s := string(raw)
		for _, t := range trackers {
			if strings.Contains(s, t) {
				add(levelHard, "%s 含有廣告、追蹤或分析工具（%s）：遊戲不能自己加入廣告、追蹤、分析 SDK", rel, t)
			}
		}
		if strings.Contains(s, "RTCPeerConnection") {
			if isGodotEngineJS(webDir, rel, s) {
				// Godot 的網頁引擎檔本身就含 WebRTC 支援；遊戲有沒有用到改在 GDScript 檢查（checkGodot）
				add("info", "%s 是 Godot 引擎檔，內含 WebRTC 支援程式碼（不影響審核；遊戲本身不要使用 WebRTC）", rel)
			} else {
				add(levelHard, "%s 使用 WebRTC（RTCPeerConnection），玩聚不允許遊戲自己建立點對點連線", rel)
			}
		}
		if strings.Contains(s, "SharedWorker") {
			add(levelHard, "%s 使用 SharedWorker，玩聚不允許", rel)
		}
		if privateKeyPEM.MatchString(s) {
			add(levelHard, "%s 裡有私鑰（-----BEGIN … PRIVATE KEY-----）：私鑰只能放在遊戲伺服器，不能進前端或上傳", rel)
		}
		if strings.Contains(s, "PaymentRequest") || strings.Contains(s, "js.stripe.com") {
			add(levelHard, "%s 使用付款功能（PaymentRequest 或金流 SDK）：遊戲 iframe 沒有付款能力，不能自己接付款（商業模式涉及真實金錢時宣告 economy.realMoney，由平台審核）", rel)
		}
		if strings.Contains(s, "getUserMedia") && !isGodotEngineJS(webDir, rel, s) {
			add(levelHard, "%s 使用 getUserMedia：遊戲不能自己使用麥克風或相機（語音由玩聚平台外殼處理）", rel)
		}
		return nil
	})
	if files == 0 {
		add(levelBuild, "%s 是空的", webDir)
	}
	if total > 50<<20 {
		add(levelWarn, "檔案總共 %.0f MB，超過建議的 50 MB，手機首次載入會很慢", float64(total)/(1<<20))
	}

	if p == nil {
		return out
	}
	if p.Client == "godot" {
		checkGodot(p, add)
	}
	if msg := sdkNotice(p); msg != "" {
		add(levelWarn, "%s", msg)
	}
	checkOrientation(p, webDir, add)
	checkAudioSources(p, webDir, add)
	if !strings.HasPrefix(p.Platform, "http") || strings.Contains(p.Platform, "（") {
		add(levelBuild, "playtogether.json 的 platform 還沒填：填玩聚網站的網址（平台 API 在同一個網址的 /v1 底下）")
	} else if err := validPlatform(strings.TrimRight(p.Platform, "/")); err != nil {
		add(levelBuild, "playtogether.json 的 platform：%v", err)
	}
	checkPolicy(p, add)
	checkStore(p, add)
	checkAI(p, add)
	if checkGameServer(p, add) {
		if p.ServerURL == "" && p.hostedServer() {
			// 玩聚代管：每個房間的伺服器網址由 SDK 握手提供（launch.serverUrl），不用填；但前端要真的用它
			if p.Client != "godot" && !webUsesLaunchServerURL(webDir) {
				add(levelWarn, "遊戲伺服器由玩聚代管，但前端沒有讀 pt.launch.serverUrl：上傳後會把請求送到遊戲畫面的網址（405）。握手後改用 pt.launch.serverUrl 當遊戲伺服器網址（pt new 的範本 game.js 有範例）")
			}
		} else if p.ServerURL == "" {
			msg := "playtogether.json 的 serverUrl 還沒填：上傳後遊戲畫面和遊戲伺服器在不同網址，要填遊戲伺服器的對外網址"
			if p.gameServerMode() == "external" {
				msg = "playtogether.json 的 serverUrl 還沒填：要填你自己的遊戲伺服器的對外網址"
			}
			add(levelBuild, "%s", msg)
		} else if u, err := url.Parse(p.ServerURL); err != nil || u.Host == "" {
			add(levelBuild, "playtogether.json 的 serverUrl 格式錯誤：%s", p.ServerURL)
		} else if u.Scheme != "https" && u.Scheme != "wss" && !isLocal(u.Hostname()) {
			add(levelHard, "遊戲伺服器要用 https:// 或 wss://（目前是 %s）", p.ServerURL)
		}
	}
	// 入門範本的 web/config.js：要和 playtogether.json 一致（pt init 接入的既有專案有自己的 config.js 格式，不檢查）
	if raw, err := os.ReadFile(filepath.Join(webDir, "config.js")); err == nil && !p.existing() {
		cfg := string(raw)
		if p.ServerURL != "" && !strings.Contains(cfg, strings.TrimRight(p.ServerURL, "/")) {
			add(levelBuild, "web/config.js 的 serverUrl 不是 %s：上傳後前端會找不到遊戲伺服器", p.ServerURL)
		}
		if strings.HasPrefix(p.Platform, "http") && !strings.Contains(cfg, strings.TrimRight(p.Platform, "/")) {
			add(levelBuild, "web/config.js 的 platformOrigins 沒有 %s：SDK 會拒絕平台的訊息", p.Platform)
		}
	}
	return out
}

func isLocal(host string) bool {
	return host == "localhost" || host == "127.0.0.1" || host == "::1" || strings.HasSuffix(host, ".localhost")
}

func cmdCheck(args []string) error {
	fs := flag.NewFlagSet("check", flag.ExitOnError)
	local := fs.Bool("local", false, "本機開發：platform、serverUrl（宣告的遊戲伺服器架構需要時）還沒填只提醒，不算錯誤（上傳前仍要填）")
	asJSON := fs.Bool("json", false, "用 JSON 輸出結果（給 CI 與 AI 工具讀）")
	_ = fs.Parse(reorder(args))
	p, err := findProject()
	if err != nil {
		return err
	}
	webDir := fs.Arg(0)
	if webDir == "" {
		if p == nil {
			return errors.New("用法：pt check <建置資料夾>（在 pt new 建立的專案裡可以省略）")
		}
		webDir = p.path(p.Web)
	}
	fs2 := checkProject(p, webDir)
	if *local {
		for i, f := range fs2 {
			if f.level == levelBuild && strings.Contains(f.msg, "還沒填") {
				fs2[i].level = levelInfo
			}
		}
	}
	var r checkResult
	if *asJSON {
		r = summarize(fs2)
		out := []map[string]string{}
		for _, f := range fs2 {
			out = append(out, map[string]string{"severity": severityName[f.level], "message": f.msg})
		}
		raw, _ := json.MarshalIndent(map[string]any{"result": r.overall(), "findings": out}, "", "  ")
		fmt.Println(string(raw))
	} else {
		r = printFindings(fs2)
	}
	if r.blocking() || r.build {
		return errors.New("檢查沒有通過")
	}
	return nil
}

// checkResult 是各等級是否出現。
type checkResult struct{ hard, schema, build, review, warn bool }

// blocking：硬性規則與宣告錯誤，-skip-check 也不能略過。
func (r checkResult) blocking() bool { return r.hard || r.schema }

func (r checkResult) overall() string {
	switch {
	case r.hard:
		return "ERROR_HARD_RULE"
	case r.schema:
		return "ERROR_SCHEMA"
	case r.build:
		return "ERROR_BUILD"
	case r.review:
		return "REVIEW_REQUIRED"
	case r.warn:
		return "WARNING"
	}
	return "PASS"
}

var severityName = map[string]string{levelHard: "ERROR_HARD_RULE", levelSchema: "ERROR_SCHEMA", levelBuild: "ERROR_BUILD",
	levelReview: "REVIEW_REQUIRED", levelWarn: "WARNING", levelInfo: "INFO"}

func summarize(fs []finding) checkResult {
	var r checkResult
	for _, f := range fs {
		switch f.level {
		case levelHard:
			r.hard = true
		case levelSchema:
			r.schema = true
		case levelBuild:
			r.build = true
		case levelReview:
			r.review = true
		case levelWarn:
			r.warn = true
		}
	}
	return r
}

// printFindings 依等級分組印出結果，最後一行是整體結果。
func printFindings(fs []finding) checkResult {
	groups := []struct{ level, mark, title string }{
		{levelHard, "✗", "平台硬性規則（-skip-check 不能略過）"},
		{levelSchema, "✗", "宣告錯誤（-skip-check 不能略過）"},
		{levelBuild, "✗", "建置與本機設定"},
		{levelReview, "?", "需要玩聚平台人工審核（可以送審，不代表已批准）"},
		{levelWarn, "！", "提醒"},
		{levelInfo, "・", "說明"},
	}
	for _, g := range groups {
		first := true
		for _, f := range fs {
			if f.level != g.level {
				continue
			}
			if first {
				fmt.Printf("%s：\n", g.title)
				first = false
			}
			fmt.Println(" ", g.mark, f.msg)
		}
	}
	r := summarize(fs)
	switch r.overall() {
	case "PASS":
		fmt.Println("✓ 檢查通過（PASS）")
	case "WARNING":
		fmt.Println("✓ 沒有錯誤（WARNING：有提醒，請確認）")
	case "REVIEW_REQUIRED":
		fmt.Println("✓ 沒有錯誤（REVIEW_REQUIRED：上面的項目需要玩聚平台人工審核，結果由平台決定）")
	default:
		fmt.Printf("✗ 檢查沒有通過（%s）\n", r.overall())
	}
	return r
}

// exportSDK 把內嵌的 SDK 全部寫到 dir（不是 pt new 建立的既有專案用）：
// browser/、godot/、go/、node/ 與 VERSION。資料夾已經有東西時拒絕，避免覆蓋開發者的檔案。
func exportSDK(dir string) error {
	if err := requireAssets(); err != nil {
		return err
	}
	if entries, err := os.ReadDir(dir); err == nil && len(entries) > 0 {
		return fmt.Errorf("%s 已經有檔案：請指定新的或空的資料夾", dir)
	}
	files := map[string][]byte{}
	read := func(src, dst string) error {
		b, err := embedded.ReadFile("assets/sdk/" + src)
		if err != nil {
			return err
		}
		files[dst] = b
		return nil
	}
	for src, dst := range map[string]string{
		"playtogether-sdk.iife.js":       "browser/playtogether-sdk.iife.js",
		"playtogether-sdk.js":            "browser/playtogether-sdk.js",
		"playtogether-sdk-godot.iife.js": "godot/playtogether-sdk-godot.iife.js",
		"playtogether-server.mjs":        "node/playtogether-server.mjs",
		"VERSION":                        "VERSION",
	} {
		if err := read(src, dst); err != nil {
			return err
		}
	}
	if err := fs2Walk("assets/sdk/godot/addons", func(rel string, content []byte) error {
		files["godot/addons/"+rel] = content
		return nil
	}); err != nil {
		return err
	}
	if err := fs2Walk("assets/sdk/go", func(rel string, content []byte) error {
		files["go/"+strings.TrimSuffix(rel, ".txt")] = content
		return nil
	}); err != nil {
		return err
	}
	for rel, content := range files {
		path := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(path, content, 0o644); err != nil {
			return err
		}
	}
	fmt.Printf("SDK %s 已寫到 %s（%d 個檔案）\n", sdkVersion(), dir, len(files))
	fmt.Println("  browser/  網頁遊戲：<script src=\"playtogether-sdk.iife.js\">，或 ES module playtogether-sdk.js")
	fmt.Println("  godot/    Godot：addons/playtogether 複製到專案，playtogether-sdk-godot.iife.js 放在匯出的 index.html 旁")
	fmt.Println("  go/       Go 遊戲伺服器：go.mod 加 replace github.com/Yanghsuanming/play-together/sdk-go => <這個資料夾>/go")
	fmt.Println("  node/     Node.js 遊戲伺服器：直接 import playtogether-server.mjs")
	return nil
}

// isGodotEngineJS：Godot 網頁匯出的引擎載入檔（和 .wasm 同名、內容是 Godot 引擎）
func isGodotEngineJS(webDir, rel, content string) bool {
	if !strings.HasSuffix(rel, ".js") || strings.ContainsAny(rel, `/\`) {
		return false
	}
	if _, err := os.Stat(filepath.Join(webDir, strings.TrimSuffix(rel, ".js")+".wasm")); err != nil {
		return false
	}
	return strings.Contains(content, "GodotRTCPeerConnection") && strings.Contains(content, "Engine")
}

// webUsesLaunchServerURL：前端的 JavaScript（不含 SDK 檔案）有沒有讀 launch.serverUrl。
func webUsesLaunchServerURL(webDir string) bool {
	found := false
	_ = filepath.WalkDir(webDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || found {
			return nil
		}
		if d.IsDir() {
			if d.Name() == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		name := d.Name()
		if !(strings.HasSuffix(name, ".js") || strings.HasSuffix(name, ".mjs") || strings.HasSuffix(name, ".ts")) || strings.HasPrefix(name, "playtogether-sdk") {
			return nil
		}
		if b, err := os.ReadFile(path); err == nil && strings.Contains(string(b), "launch.serverUrl") {
			found = true
		}
		return nil
	})
	return found
}
