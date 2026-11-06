package main

// pt init：把不是 pt new 建立的既有專案接上玩聚。只新增玩聚需要的設定，不動原本的程式碼：
//
//	playtogether.json（template: existing、server: none）、docs/content/ 的宣告文件、store/ 上架資料範本、.gitignore 的 keys/、
//	pt 自己的 skill（.claude/skills/、.agents/skills/ 的 playtogether-store：請 AI 助手準備上架資料時用）
//
// 已經存在的檔案一律不覆寫。已經接上玩聚的專案再執行 pt init，只補上 pt 提供、但專案裡缺少的檔案（skill、store/ 的說明）。既有專案的 SDK 由開發者自己管理（pt sdk export 取得），遊戲伺服器是 external 或 none。
// 之後和新專案一樣：pt login -register → pt register → pt upload。

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	texttemplate "text/template"
)

const existingTemplate = "existing"

// existing 是 pt init 接入的既有專案（SDK 與伺服器由開發者自己管理）。
func (p *project) existing() bool { return p.Template == existingTemplate }

const existingSDKNote = "這是既有專案（pt init）：SDK 檔案由你自己管理，pt 不會改寫。要更新時用 pt sdk export <資料夾> 取得新版，再覆蓋到你的專案（例如 web 的 playtogether-sdk.iife.js）"

// initName 從資料夾名稱推一個遊戲代號（小寫英文、數字、連字號）。
func initName(dir string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(filepath.Base(dir)) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case b.Len() > 0 && !strings.HasSuffix(b.String(), "-"):
			b.WriteByte('-')
		}
	}
	s := strings.Trim(b.String(), "-")
	if len(s) > 40 {
		s = strings.Trim(s[:40], "-")
	}
	s = strings.TrimLeft(s, "0123456789-") // 要英文開頭
	return s
}

// findWebDir 找要上傳的資料夾：第一個根目錄有 index.html 的常見資料夾。
func findWebDir(dir string) string {
	for _, d := range []string{"web", "dist", "build", "public", "www", "site", "."} {
		if _, err := os.Stat(filepath.Join(dir, d, "index.html")); err == nil {
			return d
		}
	}
	return ""
}

func cmdInit(args []string) error {
	fs := flag.NewFlagSet("init", flag.ExitOnError)
	name := fs.String("name", "", "遊戲代號（英文小寫、數字、連字號；預設用資料夾名稱）")
	client := fs.String("client", "", "畫面：web 或 godot（預設依專案判斷）")
	web := fs.String("web", "", "要上傳的資料夾（裡面要有 index.html；預設自動尋找 web、dist、build、public）")
	gameServer := fs.String("game-server", "", "遊戲伺服器：external（用你自己的）或 none（不需要）")
	profile := fs.String("profile", "", "內容邊界：general、moderate、strong（custom 要在終端機逐項設定）")
	table := fs.String("table-game", "none", "牌桌玩法："+strings.Join(tableCategories, "、"))
	orientation := fs.String("orientation", "", "畫面方向：portrait、landscape 或 any")
	platform := fs.String("platform", "", "平台網址（之後可以在 playtogether.json 改）")
	yes := fs.Bool("y", false, "不詢問，沒指定的項目用預設值")
	_ = fs.Parse(reorder(args))

	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	if _, err := os.Stat(filepath.Join(dir, projectFile)); err == nil {
		return initRefresh(dir)
	}
	if p, err := findProject(); err == nil && p != nil {
		return fmt.Errorf("上層資料夾 %s 已經是玩聚專案：請在那裡執行，或換一個資料夾", p.dir)
	}
	if err := requireAssets(); err != nil {
		return err
	}
	canAsk := !*yes && interactive() && !ciEnv()

	// 遊戲代號
	if *name == "" {
		*name = initName(dir)
		if canAsk {
			if *name, err = askText(fmt.Sprintf("遊戲代號（出現在網址裡；直接按 Enter 用 %s）：", *name), func(s string) error {
				if s == "" || namePattern.MatchString(s) {
					return nil
				}
				return errors.New("只能用小寫英文、數字與連字號（3–40 字，英文開頭）")
			}); err != nil {
				return err
			}
			if *name == "" {
				*name = initName(dir)
			}
		}
	}
	if !namePattern.MatchString(*name) {
		return fmt.Errorf("遊戲代號 %q 不符合格式（小寫英文、數字與連字號，3–40 字，英文開頭）：用 -name 指定", *name)
	}

	// 畫面與上傳資料夾
	if *client == "" {
		*client = "web"
		if _, err := os.Stat(filepath.Join(dir, "project.godot")); err == nil {
			*client = "godot"
		}
	}
	if !slices.Contains(clients, *client) {
		return fmt.Errorf("-client 只能是 web 或 godot（目前是 %q）", *client)
	}
	godotDir := ""
	if *client == "godot" {
		godotDir = "."
		if *web == "" {
			*web = "build" // pt build 匯出到這裡
		}
	}
	if *web == "" {
		*web = findWebDir(dir)
		if *web == "" {
			if !canAsk {
				return errors.New("找不到有 index.html 的資料夾：用 -web <資料夾> 指定要上傳的資料夾（建置後的檔案）")
			}
			if *web, err = askText("要上傳的資料夾（建置後、根目錄有 index.html 的資料夾，例如 dist）：", func(s string) error {
				if _, err := os.Stat(filepath.Join(dir, s, "index.html")); err != nil {
					return fmt.Errorf("%s 裡沒有 index.html", s)
				}
				return nil
			}); err != nil {
				return err
			}
		}
	}
	*web = filepath.ToSlash(filepath.Clean(*web))
	if strings.HasPrefix(*web, "..") || filepath.IsAbs(*web) {
		return errors.New("-web 要是專案裡的資料夾")
	}

	// 遊戲伺服器
	if *gameServer == "" {
		*gameServer = "external"
		if canAsk {
			if *gameServer, err = choose("這個遊戲需要遊戲伺服器嗎？", []option{
				{"external", gameServerLabel["external"], "已經有自己的遊戲伺服器（多人遊戲）：保留原本的架構，補上玩聚的啟動碼兌換"},
				{"none", gameServerLabel["none"], "單人遊戲，或只靠畫面端就能玩"},
			}, "external"); err != nil {
				return err
			}
		}
	}
	if *gameServer != "external" && *gameServer != "none" {
		return errors.New("既有專案的 -game-server 只能是 external（用你自己的遊戲伺服器）或 none；要用玩聚範本伺服器請用 pt new")
	}

	// 畫面方向
	if *orientation == "" {
		*orientation = "any"
		if canAsk {
			if *orientation, err = choose("玩家要用直式還是橫式玩？", []option{
				{"any", orientationLabel["any"], "直式、橫式都可以"},
				{"portrait", orientationLabel["portrait"], "手機直拿"},
				{"landscape", orientationLabel["landscape"], "手機橫拿"},
			}, "any"); err != nil {
				return err
			}
		}
	}
	if !slices.Contains(orientations, *orientation) {
		return fmt.Errorf("-orientation 只能是 portrait、landscape 或 any（目前是 %q）", *orientation)
	}

	// 內容邊界與玩法宣告
	if !slices.Contains(tableCategories, *table) {
		return fmt.Errorf("-table-game 只能是 %s（目前是 %q）", strings.Join(tableCategories, "、"), *table)
	}
	var cp *contentProfile
	switch {
	case canAsk && (*profile == "" || *profile == "custom"):
		if cp, err = askContentProfile(profile); err != nil {
			return err
		}
	case *profile == "custom":
		return errors.New("-profile custom 需要在終端機逐項設定；或先用 general、moderate、strong，之後再改 playtogether.json 的 contentProfile")
	default:
		if *profile == "" {
			*profile = defaultProfile
		}
		if !slices.Contains(profiles, *profile) {
			return fmt.Errorf("-profile 只能是 general、moderate、strong 或 custom（目前是 %q）", *profile)
		}
		c := presetProfile(*profile)
		cp = &c
	}
	pl := defaultPolicy(*cp, *table)

	if *platform == "" {
		*platform = templatePlatform
	}

	// 產生檔案（已經存在的不覆寫）
	data := map[string]string{"Name": *name, "Client": *client, "Server": "none", "Template": existingTemplate, "Platform": *platform,
		"SDKVersion": "", "GodotDir": godotDir, "Orientation": *orientation, "AI": "", "Skills": "", "AIMode": "", "GameServer": *gameServer,
		"Multiplayer": map[bool]string{true: "true", false: ""}[*gameServer != "none"], "PolicyJSON": policyJSON(pl), "PolicyVersion": loadPolicyRules().Version}
	files := map[string][]byte{}
	renderOne := func(src string) ([]byte, error) {
		raw, err := embedded.ReadFile(src)
		if err != nil {
			return nil, err
		}
		if !strings.HasSuffix(src, ".tmpl") {
			return raw, nil
		}
		t, err := texttemplate.New(path.Base(src)).Delims("[[", "]]").Option("missingkey=error").Parse(string(raw))
		if err != nil {
			return nil, fmt.Errorf("範本 %s：%w", src, err)
		}
		var b bytes.Buffer
		if err := t.Execute(&b, data); err != nil {
			return nil, err
		}
		return b.Bytes(), nil
	}
	// playtogether.json：用 pt new 的範本產生後，改成既有專案的設定
	raw, err := renderOne("templates/project/playtogether.json.tmpl")
	if err != nil {
		return err
	}
	var p project
	if err := json.Unmarshal(raw, &p); err != nil {
		return fmt.Errorf("產生 playtogether.json：%w", err)
	}
	p.dir, p.Web, p.Godot, p.AIMode = dir, *web, godotDir, ""
	if err := fs2Walk("templates/project/store", func(rel string, _ []byte) error {
		content, err := renderOne("templates/project/store/" + rel)
		if err != nil {
			return err
		}
		files[path.Join("store", strings.TrimSuffix(rel, ".tmpl"))] = content
		return nil
	}); err != nil {
		return err
	}
	for f, c := range policyDocs(*name, pl) {
		files[f] = []byte(c)
	}
	if err := addPtSkills(files); err != nil {
		return err
	}

	created, kept, err := writeMissing(dir, files)
	if err != nil {
		return err
	}
	if err := p.save(); err != nil {
		return err
	}
	created = append(created, projectFile)
	if added, err := ensureGitignore(dir, "keys/"); err != nil {
		return err
	} else if added {
		created = append(created, ".gitignore（加上 keys/）")
	}
	slices.Sort(created)

	fmt.Printf("已把 %s 接上玩聚（畫面 %s、上傳 %s/、%s、內容邊界 %s）\n", *name, *client, *web, gameServerLabel[*gameServer], profileLabel[*profile])
	for _, f := range created {
		fmt.Println("  ✓", f)
	}
	for _, f := range kept {
		fmt.Println("  ・", f, "（已經存在，沒有改）")
	}
	fmt.Println()
	initHints(&p, dir)
	return nil
}

// writeMissing 寫入專案裡還沒有的檔案；已經存在的不覆寫（回傳清單）。
func writeMissing(dir string, files map[string][]byte) (created, kept []string, err error) {
	for rel, content := range files {
		dst := filepath.Join(dir, filepath.FromSlash(rel))
		if _, err := os.Stat(dst); err == nil {
			kept = append(kept, rel)
			continue
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return created, kept, err
		}
		if err := os.WriteFile(dst, content, 0o644); err != nil {
			return created, kept, err
		}
		created = append(created, rel)
	}
	slices.Sort(created)
	slices.Sort(kept)
	return created, kept, nil
}

// initRefresh：已經接上玩聚的專案再執行 pt init，補上 pt 提供、但專案裡缺少的檔案（pt 的 skill、store/ 的說明）。
// 不改 playtogether.json、不覆寫任何已經存在的檔案（開發者改過的 skill 也保留）。
func initRefresh(dir string) error {
	files := map[string][]byte{}
	if err := addPtSkills(files); err != nil {
		return err
	}
	for _, rel := range []string{"README.md", "screenshots/README.md"} {
		raw, err := embedded.ReadFile("templates/project/store/" + rel)
		if err != nil {
			return err
		}
		files[path.Join("store", rel)] = raw
	}
	created, _, err := writeMissing(dir, files)
	if err != nil {
		return err
	}
	if len(created) == 0 {
		fmt.Println("這個資料夾已經接上玩聚（有 playtogether.json），pt 提供的檔案都在，沒有改任何檔案")
		return nil
	}
	fmt.Println("這個資料夾已經接上玩聚（有 playtogether.json）：補上缺少的檔案，其他設定沒有改")
	for _, f := range created {
		fmt.Println("  ✓", f)
	}
	fmt.Println("\n請 AI 助手（Claude Code、Codex）「幫我準備上架資料」，它會照 .claude/skills/playtogether-store（Codex：.agents/skills/）的步驟做")
	return nil
}

// ensureGitignore 確認 .gitignore 有這一行（私鑰不進版控）。
func ensureGitignore(dir, line string) (bool, error) {
	f := filepath.Join(dir, ".gitignore")
	raw, err := os.ReadFile(f)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return false, err
	}
	for _, l := range strings.Split(string(raw), "\n") {
		if strings.TrimSpace(l) == line || strings.TrimSpace(l) == "/"+line {
			return false, nil
		}
	}
	s := string(raw)
	if s != "" && !strings.HasSuffix(s, "\n") {
		s += "\n"
	}
	s += "# 玩聚的私鑰（pt register／pt keygen 產生），不要提交\n" + line + "\n"
	return true, os.WriteFile(f, []byte(s), 0o644)
}

// initHints 印出接下來要做的事；先檢查常見的缺口（SDK、平台網址）。
func initHints(p *project, dir string) {
	var todo []string
	if p.Client == "web" {
		html, err := os.ReadFile(filepath.Join(dir, p.Web, "index.html"))
		if err == nil && !strings.Contains(string(html), "playtogether-sdk") {
			todo = append(todo, fmt.Sprintf("%s/index.html 還沒有載入玩聚瀏覽器 SDK：pt sdk export sdk 取得 sdk/browser/playtogether-sdk.iife.js，放進 %s/ 並在 index.html 載入（見開發者文件「快速開始」第 3 步）", p.Web, p.Web))
		}
	}
	if p.gameServerMode() == "external" {
		todo = append(todo, "你的遊戲伺服器要兌換玩聚的啟動碼（伺服器 SDK：pt sdk export 的 go／node；文件站「遊戲伺服器」的 external 最低要求），部署後把網址填進 playtogether.json 的 serverUrl")
		todo = append(todo, "上傳到玩聚後，畫面在 https://<版本>.games.（平台網域），和遊戲伺服器不同網址：前端要連完整的伺服器網址（不能用 location.host），伺服器要允許這個來源（CORS／WebSocket Origin）")
	}
	if strings.Contains(p.Platform, "（") {
		todo = append(todo, "把 playtogether.json 的 platform 改成玩聚的網址（見開發者文件「快速開始」的「平台網址」）")
	}
	if len(todo) > 0 {
		fmt.Println("還要處理：")
		for _, t := range todo {
			fmt.Println("  -", t)
		}
		fmt.Println()
	}
	fmt.Println("下一步：")
	fmt.Println("  pt check -local          檢查宣告與要上傳的檔案")
	fmt.Println("  pt login -register       團隊擁有者或管理者：在瀏覽器同意建立遊戲")
	fmt.Println("  pt register              建立草稿遊戲、登記金鑰（金鑰 ID 寫回 playtogether.json）")
	fmt.Println("  pt upload -version 1.0.0 上傳版本（先在 store/ 填好上架資料）")
	fmt.Println()
	fmt.Println("上架資料可以請 AI 助手（Claude Code、Codex）「幫我準備上架資料」：它會照 .claude/skills/playtogether-store（Codex：.agents/skills/）的步驟起草文字、準備圖示、用 pt store screenshot 擷取截圖。")
}
