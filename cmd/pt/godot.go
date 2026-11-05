package main

// Godot 專案（pt new -client godot）：找 Godot、確認網頁匯出範本、用無頭模式匯出網頁版（pt build）。

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strconv"
	"strings"
)

// findGodot 依序找：環境變數 GODOT、PATH 裡的 godot／godot4、macOS 的 /Applications/Godot.app。
func findGodot() (string, error) {
	if g := os.Getenv("GODOT"); g != "" {
		if _, err := os.Stat(g); err != nil {
			return "", fmt.Errorf("環境變數 GODOT 指向的檔案不存在：%s", g)
		}
		return g, nil
	}
	for _, name := range []string{"godot", "godot4", "Godot"} {
		if p, err := exec.LookPath(name); err == nil {
			return p, nil
		}
	}
	if runtime.GOOS == "darwin" {
		for _, p := range []string{"/Applications/Godot.app/Contents/MacOS/Godot", filepath.Join(os.Getenv("HOME"), "Applications/Godot.app/Contents/MacOS/Godot")} {
			if _, err := os.Stat(p); err == nil {
				return p, nil
			}
		}
	}
	return "", errors.New("找不到 Godot：請安裝 Godot 4.3 以上（https://godotengine.org），或用環境變數 GODOT 指定執行檔路徑")
}

// godotVersion 回傳例如 "4.7.2.stable"（匯出範本資料夾的名稱）。
func godotVersion(godot string) (string, error) {
	out, err := exec.Command(godot, "--version").Output()
	if err != nil {
		return "", fmt.Errorf("無法執行 Godot（%s）：%w", godot, err)
	}
	v := strings.TrimSpace(string(out))
	parts := strings.Split(v, ".")
	if len(parts) < 4 || !strings.HasPrefix(v, "4.") {
		return "", fmt.Errorf("需要 Godot 4.3 以上（目前是 %s）", v)
	}
	minor := 0
	_, _ = fmt.Sscanf(parts[1], "%d", &minor)
	if minor < 3 {
		return "", fmt.Errorf("需要 Godot 4.3 以上才能關閉網頁版的執行緒支援（目前是 %s）", v)
	}
	return templateVersion(v), nil
}

// templateVersion：「4.7.2.stable.official.ed1daf0bf」→「4.7.2.stable」（數字之後的第一段是發佈狀態：stable、rc1、beta2…）
func templateVersion(v string) string {
	parts := strings.Split(v, ".")
	for i, part := range parts {
		if part == "" || part[0] < '0' || part[0] > '9' {
			return strings.Join(parts[:i+1], ".")
		}
	}
	return v
}

// exportTemplatesDir 是 Godot 放匯出範本的位置（和編輯器「管理匯出範本」相同）。
func exportTemplatesDir(version string) string {
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(os.Getenv("HOME"), "Library/Application Support/Godot/export_templates", version)
	case "windows":
		return filepath.Join(os.Getenv("APPDATA"), "Godot", "export_templates", version)
	default:
		base := os.Getenv("XDG_DATA_HOME")
		if base == "" {
			base = filepath.Join(os.Getenv("HOME"), ".local/share")
		}
		return filepath.Join(base, "godot", "export_templates", version)
	}
}

// buildGodot 用 Godot 無頭模式把 Godot 專案（預設 game/）匯出成網頁版到 build/，並放上 SDK 轉接檔。
func buildGodot(p *project) error {
	godot, err := findGodot()
	if err != nil {
		return err
	}
	version, err := godotVersion(godot)
	if err != nil {
		return err
	}
	tpl := filepath.Join(exportTemplatesDir(version), "web_nothreads_release.zip")
	if _, err := os.Stat(tpl); err != nil {
		return fmt.Errorf("沒有 Godot %s 的網頁匯出範本（%s）：請在 Godot 編輯器 → 編輯器 → 管理匯出範本 → 下載並安裝", version, tpl)
	}
	game := p.path(p.godotDir())
	out := p.path(p.Web)
	if err := os.RemoveAll(out); err != nil {
		return err
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		return err
	}
	// 匯出資料夾在 Godot 專案裡面時（Godot 專案在根目錄），讓 Godot 忽略它，否則下次匯出會把自己打包進去。
	// 隱藏檔不會被 pt upload 上傳。
	if err := os.WriteFile(filepath.Join(out, ".gdignore"), nil, 0o644); err != nil {
		return err
	}
	// 第一次（還沒有 .godot/）要先匯入資源：字型等資源還沒匯入時，Godot 啟動載入專案佈景會出錯
	if _, err := os.Stat(filepath.Join(game, ".godot", "imported")); err != nil {
		fmt.Println("第一次建置：匯入 Godot 資源…")
		if logs, err := exec.Command(godot, "--headless", "--path", game, "--import").CombinedOutput(); err != nil {
			return fmt.Errorf("Godot 匯入資源失敗：%v\n%s", err, tail(string(logs), 20))
		}
	}
	fmt.Printf("用 Godot %s 匯出網頁版 → %s\n", version, out)
	cmd := exec.Command(godot, "--headless", "--path", game, "--export-release", "Web", filepath.Join(out, "index.html"))
	logs, err := cmd.CombinedOutput()
	if err != nil || hasGodotError(string(logs)) {
		return fmt.Errorf("Godot 匯出失敗：\n%s", godotErrors(string(logs)))
	}
	if _, err := os.Stat(filepath.Join(out, "index.wasm")); err != nil {
		return fmt.Errorf("Godot 沒有產生 index.wasm：\n%s", tail(string(logs), 20))
	}
	iife, err := embedded.ReadFile("assets/sdk/playtogether-sdk-godot.iife.js")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(out, "playtogether-sdk-godot.iife.js"), iife, 0o644); err != nil {
		return err
	}
	return nil
}

// hasGodotError：Godot 匯出失敗時不一定回傳非零的結束碼，要看輸出裡的錯誤（忽略已知無害的訊息）。
func hasGodotError(logs string) bool {
	for _, line := range strings.Split(logs, "\n") {
		if strings.Contains(line, "ERROR:") || strings.Contains(line, "SCRIPT ERROR") {
			return true
		}
	}
	return false
}

// godotErrors 只取出錯誤行與下一行（位置），沒有錯誤行時給最後 20 行
func godotErrors(logs string) string {
	lines := strings.Split(logs, "\n")
	var out []string
	for i, line := range lines {
		if strings.Contains(line, "ERROR") {
			out = append(out, line)
			if i+1 < len(lines) {
				out = append(out, lines[i+1])
			}
		}
	}
	if len(out) == 0 {
		return tail(logs, 20)
	}
	return strings.Join(out, "\n")
}

func tail(s string, n int) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n")
}

func cmdBuild(args []string) error {
	p, err := findProject()
	if err != nil {
		return err
	}
	if p == nil {
		return errors.New("找不到 playtogether.json：請在用 pt new 建立的專案裡執行")
	}
	printSDKNotice(p)
	if p.Client != "godot" {
		fmt.Println("網頁專案不需要建置：web/ 就是要上傳的檔案")
		return nil
	}
	if err := buildGodot(p); err != nil {
		return err
	}
	fmt.Println("✓ 匯出完成")
	return nil
}

// checkGodot：Godot 專案專屬的檢查（匯出設定、GDScript）。
func checkGodot(p *project, add func(level, format string, a ...any)) {
	gdir := p.godotDir()
	game := p.path(gdir)
	preset, err := os.ReadFile(filepath.Join(game, "export_presets.cfg"))
	if err != nil {
		add(levelBuild, "找不到 %s（網頁匯出設定）", path.Join(gdir, "export_presets.cfg"))
	} else {
		cfg := string(preset)
		if !strings.Contains(cfg, "variant/thread_support=false") {
			add(levelBuild, "Godot 網頁匯出要關閉執行緒支援（variant/thread_support=false）：開啟時需要跨來源隔離，在玩聚的 iframe 裡無法使用")
		}
		if strings.Contains(cfg, "vram_texture_compression/for_mobile=true") {
			add(levelWarn, "行動裝置 VRAM 壓縮已開啟：需要先在專案設定啟用 ETC2/ASTC 匯入，否則匯出會失敗")
		}
		if !strings.Contains(cfg, "playtogether-sdk-godot.iife.js") {
			add(levelBuild, "匯出設定的 Head Include 沒有載入 playtogether-sdk-godot.iife.js")
		}
	}
	_ = filepath.WalkDir(game, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			if d != nil && d.IsDir() && path != game && (strings.HasPrefix(d.Name(), ".") || slices.Contains([]string{"addons", "server", "node_modules", "keys", "build", "design", "docs", "production"}, d.Name())) {
				return filepath.SkipDir
			}
			return err
		}
		if !strings.HasSuffix(d.Name(), ".gd") {
			return nil
		}
		raw, err := os.ReadFile(path)
		rel, _ := filepath.Rel(p.dir, path)
		if err == nil && strings.Contains(string(raw), "JavaScriptBridge.eval") {
			add(levelHard, "%s 使用 JavaScriptBridge.eval：需要 unsafe-eval，請改用 JavaScriptBridge.get_interface", rel)
		}
		if err == nil && strings.Contains(string(raw), "WebRTC") {
			add(levelHard, "%s 使用 WebRTC（WebRTCPeerConnection／WebRTCMultiplayerPeer）：玩聚不允許遊戲自己建立點對點連線，請透過遊戲伺服器", rel)
		}
		return nil
	})
	if raw, err := os.ReadFile(filepath.Join(game, "config.gd")); err == nil {
		cfg := string(raw)
		if p.ServerURL != "" && !strings.Contains(cfg, strings.TrimRight(p.ServerURL, "/")) {
			add(levelBuild, "%s 的 SERVER_URL 不是 %s：上傳後遊戲會找不到遊戲伺服器", path.Join(gdir, "config.gd"), p.ServerURL)
		}
		if strings.HasPrefix(p.Platform, "http") && !strings.Contains(cfg, strings.TrimRight(p.Platform, "/")) {
			add(levelBuild, "%s 的 PLATFORM_ORIGINS 沒有 %s：SDK 會拒絕平台的訊息", path.Join(gdir, "config.gd"), p.Platform)
		}
	}
	web := p.path(p.Web)
	if _, err := os.Stat(filepath.Join(web, "playtogether-sdk-godot.iife.js")); err != nil {
		add(levelBuild, "%s 裡沒有 playtogether-sdk-godot.iife.js：請用 pt build 匯出（會自動放上）", p.Web)
	}
}

var (
	viewportSize   = regexp.MustCompile(`(?m)^window/size/viewport_(width|height)=(\d+)`)
	orientationVal = regexp.MustCompile(`(?:ORIENTATION\s*:?=\s*|orientation:\s*)["']([a-z]+)["']`)
)

// checkOrientation：playtogether.json 的 orientation 要和遊戲的設定一致（Godot 的畫面大小、config.gd、web/config.js）。
func checkOrientation(p *project, webDir string, add func(level, format string, a ...any)) {
	if p.Orientation == "" {
		return // pt v0.8 以前建立的專案沒有這個欄位
	}
	if !slices.Contains(orientations, p.Orientation) {
		add(levelBuild, "playtogether.json 的 orientation 只能是 portrait、landscape 或 any（目前是 %q）", p.Orientation)
		return
	}
	declared := func(file, raw string) {
		if m := orientationVal.FindStringSubmatch(raw); m != nil && m[1] != p.Orientation {
			add(levelWarn, "%s 的畫面方向是 %s，playtogether.json 是 %s：兩邊要一致", file, m[1], p.Orientation)
		}
	}
	if p.Client == "godot" {
		gdir := p.godotDir()
		if raw, err := os.ReadFile(filepath.Join(p.path(gdir), "config.gd")); err == nil {
			declared(path.Join(gdir, "config.gd"), string(raw))
		}
		if raw, err := os.ReadFile(filepath.Join(p.path(gdir), "project.godot")); err == nil {
			size := map[string]int{}
			for _, m := range viewportSize.FindAllStringSubmatch(string(raw), -1) {
				size[m[1]], _ = strconv.Atoi(m[2])
			}
			w, h := size["width"], size["height"]
			if (p.Orientation == "portrait" && w > h) || (p.Orientation == "landscape" && h > w) {
				add(levelWarn, "Godot 的畫面大小是 %d×%d，和 playtogether.json 的方向（%s）不符：專案設定 → 顯示 → 視窗", w, h, p.Orientation)
			}
		}
		return
	}
	if raw, err := os.ReadFile(filepath.Join(webDir, "config.js")); err == nil {
		declared("web/config.js", string(raw))
	}
}
