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
	"strings"
	"text/template"
	"time"
)

//go:embed templates assets
var embedded embed.FS

const projectFile = "playtogether.json"

// templates：範本名稱 → 伺服器範本資料夾
var templates = map[string]string{
	"web-go":   "server-go",
	"web-node": "server-node",
}

var namePattern = regexp.MustCompile(`^[a-z][a-z0-9-]{1,38}[a-z0-9]$`)

// project 是 playtogether.json：pt 的指令都從這裡讀設定，參數可以覆蓋。
type project struct {
	Name          string `json:"name"`
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

	dir string // playtogether.json 所在的資料夾
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
	tmpl := fs.String("template", "web-go", "範本：web-go（網頁＋Go 伺服器）或 web-node（網頁＋Node.js 伺服器）")
	platform := fs.String("platform", "https://（平台網址）", "玩聚平台網址（之後可以在 playtogether.json 修改）")
	_ = fs.Parse(reorder(args))
	if fs.NArg() != 1 {
		return errors.New("用法：pt new [-template web-go|web-node] <專案名稱>")
	}
	name := fs.Arg(0)
	if !namePattern.MatchString(name) {
		return errors.New("專案名稱只能用小寫英文、數字與連字號（3–40 字，英文開頭），例如 my-party-game")
	}
	server, ok := templates[*tmpl]
	if !ok {
		return fmt.Errorf("沒有範本 %q：可用 web-go、web-node", *tmpl)
	}
	if err := requireAssets(); err != nil {
		return err
	}
	if _, err := os.Stat(name); err == nil {
		return fmt.Errorf("資料夾 %s 已經存在", name)
	}
	data := map[string]string{"Name": name, "Template": *tmpl, "Platform": *platform, "SDKVersion": sdkVersion()}
	files := map[string][]byte{}
	render := func(src, dst string) error {
		return fs2Walk("templates/"+src, func(rel string, content []byte) error {
			out := path.Join(dst, strings.TrimSuffix(rel, ".tmpl"))
			if strings.HasSuffix(rel, ".tmpl") {
				t, err := template.New(rel).Delims("[[", "]]").Option("missingkey=error").Parse(string(content))
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
	if err := render("project", ""); err != nil {
		return err
	}
	if err := render("web", "web"); err != nil {
		return err
	}
	if err := render(server, "server"); err != nil {
		return err
	}
	files[".gitignore"] = files["gitignore"]
	delete(files, "gitignore")
	sdk, err := sdkFiles(*tmpl)
	if err != nil {
		return err
	}
	for k, v := range sdk {
		files[k] = v
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
	fmt.Printf("已建立 %s（範本 %s、SDK %s）\n\n", name, *tmpl, sdkVersion())
	fmt.Println("下一步：")
	fmt.Printf("  cd %s\n", name)
	fmt.Println("  pt dev        # 本機模擬模式，打開兩個玩家分頁")
	fmt.Println("接上玩聚的步驟見 README.md。")
	return nil
}

// sdkFiles：範本需要的 SDK 檔案（相對專案根目錄的路徑 → 內容）。pt new 與 pt sdk update 共用。
func sdkFiles(tmpl string) (map[string][]byte, error) {
	out := map[string][]byte{}
	iife, err := embedded.ReadFile("assets/sdk/playtogether-sdk.iife.js")
	if err != nil {
		return nil, err
	}
	out["web/playtogether-sdk.iife.js"] = iife
	switch tmpl {
	case "web-go":
		err = fs2Walk("assets/sdk/go", func(rel string, content []byte) error {
			out["server/third_party/playtogether-sdk-go/"+strings.TrimSuffix(rel, ".txt")] = content
			return nil
		})
	case "web-node":
		var mjs []byte
		mjs, err = embedded.ReadFile("assets/sdk/playtogether-server.mjs")
		out["server/playtogether-server.mjs"] = mjs
	default:
		err = fmt.Errorf("不認得的範本 %q", tmpl)
	}
	return out, err
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
	if len(args) != 1 || args[0] != "update" {
		return errors.New("用法：pt sdk update")
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
	files, err := sdkFiles(p.Template)
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
	var cmd *exec.Cmd
	switch p.Template {
	case "web-go":
		cmd = exec.Command("go", "run", ".")
	case "web-node":
		cmd = exec.Command("node", "main.mjs")
	default:
		return fmt.Errorf("pt dev 只支援 web-go、web-node 範本（目前是 %q）", p.Template)
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
		add("error", "%s 根目錄沒有 index.html", webDir)
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
			add("error", "%s 使用 WebRTC（RTCPeerConnection），玩聚不允許遊戲自己建立點對點連線", rel)
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
	bad := false
	for _, f := range fs {
		mark := "！"
		if f.level == "error" {
			mark, bad = "✗", true
		}
		fmt.Println(mark, f.msg)
	}
	if len(fs) == 0 {
		fmt.Println("✓ 檢查通過")
	} else if !bad {
		fmt.Println("✓ 沒有錯誤（有提醒，請確認）")
	}
	return bad
}
