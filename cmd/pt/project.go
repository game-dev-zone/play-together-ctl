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

//go:embed templates assets
var embedded embed.FS

const projectFile = "playtogether.json"

// 畫面（client）與伺服器（server）的組合；舊的 -template 名稱對應到這裡
var (
	clients   = []string{"web", "godot"}
	servers   = []string{"go", "node"}
	templates = map[string][2]string{
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
	AI    string `json:"ai,omitempty"` // studios：已加入 Claude Code Game Studios

	dir string // playtogether.json 所在的資料夾
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
			p.dir = dir
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
		return errors.New("這個 pt 沒有內嵌 SDK：請從 https://github.com/Yanghsuanming/play-together-ctl/releases 下載 pt")
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
	ai := fs.String("ai", "", "AI 開發範本：studios（加入 Claude Code Game Studios，只限 Godot）或 none；在終端機執行且沒指定時會詢問")
	yes := fs.Bool("y", false, "不詢問，沒指定的項目用預設值")
	_ = fs.Parse(reorder(args))
	set := map[string]bool{}
	fs.Visit(func(f *flag.Flag) { set[f.Name] = true })
	if fs.NArg() > 1 {
		return errors.New("用法：pt new [-client web|godot] [-server go|node] [-ai studios|none] <專案名稱>")
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
	if *ai != "" && *ai != "studios" && *ai != "none" {
		return fmt.Errorf("-ai 只能是 studios 或 none（目前是 %q）", *ai)
	}
	if *ai == "studios" && *client != "godot" {
		if set["client"] {
			return errors.New("Claude Code Game Studios 是遊戲引擎專用的範本，目前只搭配 Godot：請加上 -client godot（網頁專案已經附上給 AI 助手的 CLAUDE.md、AGENTS.md）")
		}
		*client, set["client"] = "godot", true // 只指定 -ai studios：一定是 Godot
	}
	if err := requireAssets(); err != nil {
		return err
	}
	ccgs, err := loadCCGS()
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
	if !*yes && interactive() && (name == "" || !set["client"] || !set["server"] || !set["ai"]) {
		if name, err = newWizard(name, client, server, ai, set, ccgs, checkName); err != nil {
			if errors.Is(err, errCanceled) {
				fmt.Println("\n已取消，沒有建立任何檔案。")
				return nil
			}
			return err
		}
		if name == "" {
			return nil
		}
	}
	if name == "" {
		return errors.New("用法：pt new [-client web|godot] [-server go|node] [-ai studios|none] <專案名稱>（在終端機執行 pt new 會逐步詢問）")
	}
	studios := *ai == "studios"
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
		"GodotDir": godotDir, "GodotPrefix": godotPrefix, "GodotPath": godotPath, "AI": map[bool]string{true: "studios", false: ""}[studios]}
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
	for _, r := range [][2]string{{"project", ""}, {"ai", ""}, {"client-" + *client, clientDir}, {"server-" + *server, "server"}} {
		if err := render(r[0], r[1]); err != nil {
			return err
		}
	}
	files[".gitignore"] = files["gitignore"]
	delete(files, "gitignore")
	sdk, err := sdkFiles(*client, *server, godotDir)
	if err != nil {
		return err
	}
	for k, v := range sdk {
		files[k] = v
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
	fmt.Printf("已建立 %s（畫面 %s、伺服器 %s、SDK %s）\n\n", name, *client, *server, sdkVersion())
	fmt.Println("下一步：")
	fmt.Printf("  cd %s\n", name)
	if *client == "godot" {
		fmt.Println("  pt dev        # 匯出 Godot 網頁版並用本機模擬模式打開兩個玩家分頁（需要 Godot 4.3 以上與網頁匯出範本）")
		if studios {
			fmt.Println("  用 Godot 編輯器打開專案根目錄修改遊戲")
			fmt.Printf("  claude        # 在專案根目錄啟動 Claude Code，執行 /start 開始（%s 已加入，玩聚規則在 PLAYTOGETHER.md）\n", ccgs.Name)
		} else {
			fmt.Println("  用 Godot 編輯器打開 game/ 修改遊戲")
		}
	} else {
		fmt.Println("  pt dev        # 本機模擬模式，打開兩個玩家分頁")
	}
	fmt.Println("接上玩聚的步驟見 README.md。")
	if !studios {
		fmt.Println("用 AI 助手開發：專案附有 CLAUDE.md、AGENTS.md，平台規則在 PLAYTOGETHER.md。")
		if *client == "godot" {
			fmt.Printf("想用 %s（AI 遊戲開發團隊範本）：建立專案時選擇它，或加上 -ai studios。\n", ccgs.Name)
		}
	}
	return nil
}

// newWizard 逐步詢問專案名稱、畫面、伺服器、AI 開發範本，最後確認。回傳空名稱表示開發者選擇不建立。
func newWizard(name string, client, server, ai *string, set map[string]bool, ccgs ccgsManifest, checkName func(string) error) (string, error) {
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
	if !set["server"] {
		if *server, err = choose("遊戲伺服器（兌換玩家身分、判定勝負）要用哪種語言？", []option{
			{"go", "Go", ""},
			{"node", "Node.js", ""},
		}, "go"); err != nil {
			return "", err
		}
	}
	if !set["ai"] {
		if *client == "godot" {
			if *ai, err = choose("要用哪種方式搭配 AI 開發？", []option{
				{"none", "一般專案", "附上給 AI 助手的規則（PLAYTOGETHER.md、CLAUDE.md、AGENTS.md），Claude Code、Codex 等都適用；Godot 專案在 game/"},
				{"studios", ccgs.Name, fmt.Sprintf("用 Claude Code 當遊戲開發團隊：企劃、美術、程式、QA 等 AI 角色與流程（MIT，%s 版）。\n     從 GitHub 下載並逐檔驗證，加上玩聚規則；Godot 專案放在根目錄", ccgs.Version)},
			}, ""); err != nil {
				return "", err
			}
		} else {
			*ai = "none"
			fmt.Printf("AI 開發：專案會附上給 AI 助手的規則（PLAYTOGETHER.md、CLAUDE.md、AGENTS.md）。%s 目前只搭配 Godot。\n\n", ccgs.Name)
		}
	}
	cmd := fmt.Sprintf("pt new -client %s -server %s -ai %s %s", *client, *server, *ai, name)
	fmt.Printf("即將建立 %s：畫面 %s、伺服器 %s、AI 範本 %s\n", name, *client, *server, map[string]string{"studios": ccgs.Name, "none": "一般"}[*ai])
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
	if len(args) != 1 || args[0] != "update" {
		return errors.New("用法：pt sdk update（在 pt new 建立的專案裡）或 pt sdk export <資料夾>（把 SDK 檔案寫到資料夾，給既有專案用）")
	}
	p, err := findProject()
	if err != nil {
		return err
	}
	if p == nil {
		return errors.New("找不到 playtogether.json：請在用 pt new 建立的專案裡執行")
	}
	if err := requireAssets(); err != nil {
		return err
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
	return nil
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
	players := fs.Int("players", 2, "要打開幾個玩家分頁（0 表示不開瀏覽器）")
	_ = fs.Parse(args)
	p, err := findProject()
	if err != nil {
		return err
	}
	if p == nil {
		return errors.New("找不到 playtogether.json：請在用 pt new 建立的專案裡執行")
	}
	if *addr == "" {
		*addr = p.DevAddr
	}
	if ln, err := net.Listen("tcp", *addr); err != nil {
		return fmt.Errorf("%s 已經被其他程式使用，請換一個：pt dev -addr 127.0.0.1:18090", *addr)
	} else {
		ln.Close()
	}
	if p.Client == "godot" {
		if err := buildGodot(p); err != nil {
			return err
		}
	}
	var cmd *exec.Cmd
	switch p.Server {
	case "go":
		cmd = exec.Command("go", "run", ".")
	case "node":
		cmd = exec.Command("node", "main.mjs")
	default:
		return fmt.Errorf("pt dev 只支援 go、node 伺服器（目前是 %q）", p.Server)
	}
	cmd.Dir = p.path("server")
	// 模擬模式：不需要金鑰、不連玩聚；前端由遊戲伺服器直接提供（同一個來源）
	cmd.Env = append(os.Environ(), "PT_MOCK=1", "ADDR="+*addr, "WEB_DIR="+p.path(p.Web))
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("啟動遊戲伺服器失敗：%w", err)
	}
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	site := "http://" + *addr + "/"
	ready := false
	for i := 0; i < 300 && !ready; i++ { // go run 第一次要編譯，最多等 60 秒
		select {
		case err := <-done:
			return fmt.Errorf("遊戲伺服器結束了：%v", err)
		case <-time.After(200 * time.Millisecond):
		}
		if res, err := http.Get(site); err == nil {
			res.Body.Close()
			ready = true
		}
	}
	if !ready {
		_ = cmd.Process.Kill()
		return errors.New("遊戲伺服器 60 秒內沒有啟動")
	}
	fmt.Printf("\n本機模擬模式：%s（每開一個分頁就是一位玩家；Ctrl-C 結束）\n", site)
	for i := 0; i < *players; i++ {
		openBrowser(site)
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
	level string // error、warn
	msg   string
}

var (
	scriptSrc = regexp.MustCompile(`(?i)<script[^>]+src=["']([^"']+)["']`)
	trackers  = []string{"googletagmanager.com", "google-analytics.com", "doubleclick.net", "googlesyndication.com", "connect.facebook.net",
		"hotjar.com", "clarity.ms", "mixpanel.com", "segment.com", "amplitude.com", "adsbygoogle"}
)

// checkProject：上傳前在本機先擋掉常見的退件與設定錯誤（對照文件站的審核準則與安全要求）。
func checkProject(p *project, webDir string) []finding {
	var out []finding
	add := func(level, format string, a ...any) { out = append(out, finding{level, fmt.Sprintf(format, a...)}) }

	index, err := os.ReadFile(filepath.Join(webDir, "index.html"))
	if err != nil {
		if p != nil && p.Client == "godot" {
			add("error", "還沒有匯出網頁版：先執行 pt build（pt upload 會自動執行）")
		} else {
			add("error", "%s 根目錄沒有 index.html", webDir)
		}
	} else {
		html := string(index)
		if !strings.Contains(html, "playtogether-sdk") {
			add("error", "index.html 沒有載入玩聚 SDK（playtogether-sdk.iife.js 或 playtogether-sdk-godot.iife.js）")
		}
		for _, m := range scriptSrc.FindAllStringSubmatch(html, -1) {
			if strings.HasPrefix(m[1], "http") || strings.HasPrefix(m[1], "//") {
				add("warn", "index.html 從外部載入程式：%s（建議和遊戲檔案放在一起；外部 CDN 請固定版本號）", m[1])
			}
		}
	}

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
			add("error", "上傳資料夾裡有私鑰或密鑰檔案：%s（不能上傳）", rel)
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
				add("error", "%s 含有廣告、追蹤或分析工具（%s），審核不接受", rel, t)
			}
		}
		if strings.Contains(s, "RTCPeerConnection") {
			if isGodotEngineJS(webDir, rel, s) {
				// Godot 的網頁引擎檔本身就含 WebRTC 支援；遊戲有沒有用到改在 GDScript 檢查（checkGodot）
				add("info", "%s 是 Godot 引擎檔，內含 WebRTC 支援程式碼（不影響審核；遊戲本身不要使用 WebRTC）", rel)
			} else {
				add("error", "%s 使用 WebRTC（RTCPeerConnection），玩聚不允許遊戲自己建立點對點連線", rel)
			}
		}
		if strings.Contains(s, "SharedWorker") {
			add("error", "%s 使用 SharedWorker，玩聚不允許", rel)
		}
		return nil
	})
	if files == 0 {
		add("error", "%s 是空的", webDir)
	}
	if total > 50<<20 {
		add("warn", "檔案總共 %.0f MB，超過建議的 50 MB，手機首次載入會很慢", float64(total)/(1<<20))
	}

	if p == nil {
		return out
	}
	if p.Client == "godot" {
		checkGodot(p, add)
	}
	if !strings.HasPrefix(p.Platform, "http") || strings.Contains(p.Platform, "（") {
		add("error", "playtogether.json 的 platform 還沒填（玩聚建立遊戲後會給你平台網址）")
	}
	if p.ServerURL == "" {
		add("error", "playtogether.json 的 serverUrl 還沒填：上傳後遊戲畫面和遊戲伺服器在不同網址，要填遊戲伺服器的對外網址")
	} else if u, err := url.Parse(p.ServerURL); err != nil || u.Host == "" {
		add("error", "playtogether.json 的 serverUrl 格式錯誤：%s", p.ServerURL)
	} else if u.Scheme != "https" && u.Scheme != "wss" && !isLocal(u.Hostname()) {
		add("error", "遊戲伺服器要用 https:// 或 wss://（目前是 %s）", p.ServerURL)
	}
	// 入門範本的 web/config.js：要和 playtogether.json 一致
	if raw, err := os.ReadFile(filepath.Join(webDir, "config.js")); err == nil {
		cfg := string(raw)
		if p.ServerURL != "" && !strings.Contains(cfg, strings.TrimRight(p.ServerURL, "/")) {
			add("error", "web/config.js 的 serverUrl 不是 %s：上傳後前端會找不到遊戲伺服器", p.ServerURL)
		}
		if strings.HasPrefix(p.Platform, "http") && !strings.Contains(cfg, strings.TrimRight(p.Platform, "/")) {
			add("error", "web/config.js 的 platformOrigins 沒有 %s：SDK 會拒絕平台的訊息", p.Platform)
		}
	}
	return out
}

func isLocal(host string) bool {
	return host == "localhost" || host == "127.0.0.1" || host == "::1" || strings.HasSuffix(host, ".localhost")
}

func cmdCheck(args []string) error {
	fs := flag.NewFlagSet("check", flag.ExitOnError)
	_ = fs.Parse(args)
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
	if printFindings(checkProject(p, webDir)) {
		return errors.New("檢查沒有通過")
	}
	return nil
}

// printFindings 印出結果，回傳是否有錯誤。
func printFindings(fs []finding) bool {
	bad, warned := false, false
	for _, f := range fs {
		mark := "・" // info：說明，不需要處理
		switch f.level {
		case "error":
			mark, bad = "✗", true
		case "warn":
			mark, warned = "！", true
		}
		fmt.Println(mark, f.msg)
	}
	if !bad && !warned {
		fmt.Println("✓ 檢查通過")
	} else if !bad {
		fmt.Println("✓ 沒有錯誤（有提醒，請確認）")
	}
	return bad
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
