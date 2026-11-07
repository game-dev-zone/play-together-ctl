package main

// pt add server：給 pt init 接入的既有專案（多半是單機遊戲）加上玩聚的範本遊戲伺服器，準備改成多人聯機。
//
// 寫入 server/（範本伺服器＋SDK 的遊戲伺服器套件）與 multiplayer/playtogether-net.js（網頁前端的連線小工具），
// 並把 playtogether.json 改成「範本伺服器」：之後 pt dev 會啟動它、pt verify server 可以驗證、pt sdk update 會更新伺服器的 SDK。
// 不動原本的遊戲程式；把遊戲規則搬到伺服器、前端改成只送操作，由開發者或 AI 助手照 playtogether-multiplayer skill 做。

import (
	"bytes"
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

const netHelperPath = "multiplayer/playtogether-net.js"

func cmdAdd(args []string) error {
	if len(args) == 0 || args[0] != "server" {
		return errors.New("用法：pt add server -server go|node [-hosting playtogether]（既有專案加上玩聚的範本遊戲伺服器）")
	}
	fs := flag.NewFlagSet("add server", flag.ExitOnError)
	server := fs.String("server", "", "遊戲伺服器的語言：go 或 node")
	hosting := fs.String("hosting", "", "playtogether＝交給玩聚代管（只支援 go；要平台開通）；不填＝你自己部署")
	_ = fs.Parse(args[1:])
	if fs.NArg() > 0 {
		return errors.New("用法：pt add server -server go|node [-hosting playtogether]")
	}
	if err := requireAssets(); err != nil {
		return err
	}
	p, err := findProject()
	if err != nil {
		return err
	}
	if p == nil {
		return errors.New("找不到 playtogether.json：先用 pt init 把專案接上玩聚")
	}
	if !p.existing() {
		return errors.New("這個專案是 pt new 建立的，已經有範本伺服器（server/）：直接改 server/ 裡的遊戲規則")
	}
	if p.Server != "none" {
		return fmt.Errorf("這個專案已經有範本伺服器（server: %s）", p.Server)
	}
	if p.ServerURL != "" || p.hostedServer() {
		return errors.New("這個專案已經設定了你自己的遊戲伺服器（serverUrl 或 gameServer.hosting）：pt add server 只給還沒有伺服器的專案；要換成範本伺服器，先把 playtogether.json 的 serverUrl 與 gameServer.hosting 清掉")
	}
	if *server == "" {
		*server = "go"
		if interactive() && !ciEnv() {
			if *server, err = choose("遊戲伺服器要用哪個語言？", []option{
				{"go", "Go", "效能好；可以交給玩聚代管"},
				{"node", "Node.js", "和網頁前端同一種語言"},
			}, "go"); err != nil {
				return err
			}
		}
	}
	if *server != "go" && *server != "node" {
		return fmt.Errorf("-server 只能是 go 或 node（目前是 %q）", *server)
	}
	switch *hosting {
	case "":
	case "playtogether":
		if *server != "go" {
			return errors.New("玩聚代管目前只支援 Go 伺服器：用 -server go")
		}
	default:
		return fmt.Errorf("-hosting 只能是 playtogether 或不填（目前是 %q）", *hosting)
	}
	if _, err := os.Stat(p.path("server")); err == nil {
		return errors.New("專案裡已經有 server/ 資料夾：先改名或移走，pt add server 不覆寫既有檔案")
	}

	files, err := addServerFiles(p.Name, p.Client, *server)
	if err != nil {
		return err
	}
	if p.Client == "web" {
		raw, err := embedded.ReadFile("templates/multiplayer/playtogether-net.js")
		if err != nil {
			return err
		}
		files[netHelperPath] = raw
	}
	for rel := range files {
		if _, err := os.Stat(p.path(filepath.FromSlash(rel))); err == nil {
			return fmt.Errorf("%s 已經存在：pt add server 不覆寫既有檔案", rel)
		}
	}
	created, _, err := writeMissing(p.dir, files)
	if err != nil {
		return err
	}

	// playtogether.json：範本伺服器（之後 pt dev、pt verify server、pt sdk update 都適用）
	p.Server = *server
	gs := gameServerConfig{}
	if p.GameServer != nil {
		gs = *p.GameServer
	}
	gs.Mode = defaultGameServer
	if *hosting != "" {
		gs.Hosting, gs.Dir = *hosting, "server"
	}
	p.GameServer = &gs
	p.Multiplayer = &multiplayerConfig{Enabled: true}
	p.SDKVersion = sdkVersion()
	if err := p.save(); err != nil {
		return err
	}
	// 伺服器的 SDK 記基線（pt sdk update 不會蓋掉被改過的檔案）
	m, err := loadManifest(p.dir)
	if err != nil {
		return err
	}
	if m == nil {
		m = &genManifest{ManifestVersion: manifestVersion, TemplateVersion: version}
	}
	sdk, err := serverSDKFiles(p.Client, *server)
	if err != nil {
		return err
	}
	for rel, content := range sdk {
		m.set(rel, kindSDK, sha(content))
	}
	m.SDKVersion = p.SDKVersion
	if err := m.save(p.dir); err != nil {
		return err
	}
	// pt 的 skill（含 playtogether-multiplayer）：缺的補上
	skills := map[string][]byte{}
	if err := addPtSkills(skills); err != nil {
		return err
	}
	added, _, err := writeMissing(p.dir, skills)
	if err != nil {
		return err
	}
	created = append(created, added...)
	created = append(created, manifestPath)
	slices.Sort(created)

	fmt.Printf("已加上玩聚的範本遊戲伺服器（%s）：\n", map[string]string{"go": "Go", "node": "Node.js"}[*server])
	shown := 0
	for _, f := range created {
		if strings.Contains(f, "/third_party/") {
			continue
		}
		fmt.Println("  ✓", f)
		shown++
	}
	fmt.Printf("  ✓ 遊戲伺服器的 SDK（%d 個檔案）\n", len(created)-shown)
	fmt.Println("  ✓ playtogether.json：server、gameServer、multiplayer 改成範本伺服器")
	fmt.Println()
	fmt.Println("範本伺服器現在是示範玩法「一起按」，你的遊戲還是單機的。接下來：")
	fmt.Println("  1. 決定多人怎麼玩（幾個人、同時還是輪流、怎麼分勝負）")
	fmt.Println("  2. 把遊戲規則搬到 server/（伺服器決定分數、勝負、流程），前端改成只送操作、照伺服器的狀態顯示")
	if p.Client == "web" {
		fmt.Printf("     前端連線用 %s：放進會被建置到 %s/ 的地方，在 index.html 載入（用法在檔案開頭）\n", netHelperPath, p.Web)
	} else {
		fmt.Println("     Godot 前端：握手後用 PlayTogether.server_url，POST /join 換 token、POST /action 送操作（見 playtogether-multiplayer skill 的「連線協定」）")
	}
	fmt.Println("  3. pt dev -players 3 開三個玩家視窗實際玩；改完用 pt verify server 驗證平台接口")
	fmt.Println()
	fmt.Println("交給 AI 助手（Claude Code、Codex）：說「幫我把這個遊戲改成多人」，它會照 .claude/skills/playtogether-multiplayer（Codex：.agents/skills/）一步一步做，玩法的決定會先問你。")
	if *hosting == "" {
		fmt.Println("\n上線前：遊戲伺服器要部署到你自己的主機，把網址填進 playtogether.json 的 serverUrl；或改成玩聚代管（Go，gameServer.hosting = playtogether，要平台開通）。")
	}
	return nil
}

// addServerFiles：範本伺服器（server-go／server-node）＋伺服器的 SDK。
func addServerFiles(name, client, server string) (map[string][]byte, error) {
	files := map[string][]byte{}
	data := map[string]string{"Name": name}
	if err := fs2Walk("templates/server-"+server, func(rel string, content []byte) error {
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
		files[path.Join("server", strings.TrimSuffix(rel, ".tmpl"))] = content
		return nil
	}); err != nil {
		return nil, err
	}
	sdk, err := serverSDKFiles(client, server)
	if err != nil {
		return nil, err
	}
	for rel, content := range sdk {
		files[rel] = content
	}
	return files, nil
}

// serverSDKFiles：只有遊戲伺服器的 SDK（既有專案的前端 SDK 由開發者自己管理）。
func serverSDKFiles(client, server string) (map[string][]byte, error) {
	all, err := sdkFiles(client, server, "game")
	if err != nil {
		return nil, err
	}
	out := map[string][]byte{}
	for rel, content := range all {
		if strings.HasPrefix(rel, "server/") {
			out[rel] = content
		}
	}
	return out, nil
}
