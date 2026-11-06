package main

// pt store screenshot：用本機的 Chrome／Chromium 無頭模式擷取實際遊戲畫面，存成上架截圖（store/screenshots/）。
//
// 只用 Chrome 的命令列（--screenshot），不用 CDP／WebSocket（標準函式庫沒有 WebSocket）。
// 手機視窗用 --kiosk 加上 --screen-info 模擬：螢幕是截圖規格的像素（直式 1080×1920、橫式 1920×1080）、
// devicePixelRatio 3，所以遊戲看到的是 360×640（橫式 640×360）CSS px 的手機畫面，輸出剛好是規格尺寸。
// 不用 --window-size＋--force-device-scale-factor：新版無頭模式的視窗寬度最少 500 CSS px、
// 頁面高度會少掉一段（約 87 px），小數倍率也會差一個像素，都不符合手機截圖的需要。
// --screen-info 的 devicePixelRatio 只接受整數，所以倍率固定是 3。

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/png"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// cmdStore：pt store <子指令>
func cmdStore(args []string) error {
	if len(args) == 0 || args[0] != "screenshot" {
		return errors.New("用法：pt store screenshot [-url 網址] [-wait 5s] [-n 1] [-interval 3s] [-o 資料夾] [-replace] [-locale en] [-orientation portrait|landscape] [-chrome 路徑]")
	}
	return cmdStoreScreenshot(args[1:])
}

// shotScale 是模擬手機的 devicePixelRatio（--screen-info 只接受整數）
const shotScale = 3

type shotOptions struct {
	URL         string
	Wait        time.Duration
	Interval    time.Duration
	N           int
	Out         string
	Replace     bool
	Locale      string // "" 是共用（繁中）截圖、"en" 是英文介面專用
	Orientation string // portrait 或 landscape
	Chrome      string
}

// parseShotArgs 解析參數並依 playtogether.json 補上預設值（p 可以是 nil：不在玩聚專案裡）。
func parseShotArgs(args []string, p *project) (shotOptions, error) {
	fs := flag.NewFlagSet("store screenshot", flag.ContinueOnError)
	var o shotOptions
	fs.StringVar(&o.URL, "url", "", "遊戲網址（預設 http://<playtogether.json 的 devAddr>/）")
	fs.DurationVar(&o.Wait, "wait", 5*time.Second, "載入後等多久再擷取")
	fs.IntVar(&o.N, "n", 1, "擷取幾張（每張重新載入；第 i 張多等 (i-1)×-interval）")
	fs.DurationVar(&o.Interval, "interval", 3*time.Second, "多張時每張多等的時間")
	fs.StringVar(&o.Out, "o", "", "輸出資料夾（預設 store/screenshots；-locale en 時是 store/screenshots/en）")
	fs.BoolVar(&o.Replace, "replace", false, "從 01 開始覆寫（預設接在既有截圖後面，不覆寫）")
	fs.StringVar(&o.Locale, "locale", "", "en：英文介面專用的截圖（存到 store/screenshots/en/）")
	fs.StringVar(&o.Orientation, "orientation", "", "portrait（直式 1080×1920）或 landscape（橫式 1920×1080）；預設依 playtogether.json，any 時直式")
	fs.StringVar(&o.Chrome, "chrome", "", "Chrome／Chromium 執行檔路徑（或環境變數 PT_CHROME；預設自動尋找）")
	if err := fs.Parse(reorder(args)); err != nil {
		return o, err
	}
	if fs.NArg() > 0 {
		return o, fmt.Errorf("不認得的參數：%s", strings.Join(fs.Args(), " "))
	}
	if o.N < 1 || o.N > 10 {
		return o, errors.New("-n 要是 1–10（最多 10 張截圖）")
	}
	if o.Wait < 0 || o.Interval < 0 || o.Wait+time.Duration(o.N-1)*o.Interval > 2*time.Minute {
		return o, errors.New("-wait、-interval 不能是負的，最長等待（-wait＋(n-1)×-interval）不能超過 2 分鐘")
	}
	switch o.Locale {
	case "", "zh-TW":
		o.Locale = ""
	case "en":
	default:
		return o, fmt.Errorf("-locale 只能是 en（英文介面專用的截圖）；共用的截圖不用指定（目前是 %q）", o.Locale)
	}
	proj := ""
	if p != nil {
		proj = p.Orientation
	}
	switch {
	case o.Orientation == "":
		o.Orientation = "portrait"
		if proj == "landscape" {
			o.Orientation = "landscape"
		}
	case o.Orientation != "portrait" && o.Orientation != "landscape":
		return o, fmt.Errorf("-orientation 只能是 portrait 或 landscape（目前是 %q）", o.Orientation)
	case (proj == "portrait" || proj == "landscape") && proj != o.Orientation:
		return o, fmt.Errorf("playtogether.json 的畫面方向是 %s（%s），截圖要同一個方向；-orientation %s 只能用在 any 的專案", proj, orientationLabel[proj], o.Orientation)
	}
	if o.URL == "" {
		if p == nil {
			return o, errors.New("不在玩聚專案裡（找不到 playtogether.json）：用 -url 指定遊戲網址")
		}
		o.URL = "http://" + p.DevAddr + "/"
	}
	if u, err := url.Parse(o.URL); err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return o, fmt.Errorf("-url 要是 http(s):// 網址（目前是 %q）", o.URL)
	}
	if o.Out == "" {
		base := "."
		if p != nil {
			base = p.dir
		}
		o.Out = filepath.Join(base, storeDir, "screenshots")
		if o.Locale == "en" {
			o.Out = filepath.Join(o.Out, "en")
		}
	}
	return o, nil
}

// shotSize：截圖的輸出像素與遊戲看到的 CSS 視窗（和 storeImageSpec 的截圖尺寸相同）
func shotSize(orientation string) (w, h, cssW, cssH int) {
	w, h, _ = storeImageSpec("screenshot", orientation, 9, 16)
	return w, h, w / shotScale, h / shotScale
}

var shotName = regexp.MustCompile(`^(\d+)\.(?i:png|jpe?g)$`)

// nextShotNames 決定 n 張截圖的檔名：接在資料夾裡最大的編號後面（01.png、02.png…），replace 時從 01 開始。
// 資料夾裡的截圖（含不是數字檔名的）加起來不能超過 10 張。
func nextShotNames(dir string, n int, replace bool) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	maxNum, images := 0, map[string]bool{}
	for _, e := range entries {
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if e.IsDir() || (ext != ".png" && ext != ".jpg" && ext != ".jpeg") {
			continue
		}
		images[e.Name()] = true
		if m := shotName.FindStringSubmatch(e.Name()); m != nil {
			if v, err := strconv.Atoi(m[1]); err == nil && v > maxNum {
				maxNum = v
			}
		}
	}
	start := maxNum + 1
	if replace {
		start = 1
	}
	var names []string
	total := len(images)
	for i := 0; i < n; i++ {
		name := fmt.Sprintf("%02d.png", start+i)
		names = append(names, name)
		if !images[name] {
			total++
		}
	}
	if replace {
		// 同一個編號的 .jpg 也會被這次的 .png 取代（不然會有兩張同編號的截圖）
		for _, name := range names {
			base := strings.TrimSuffix(name, ".png")
			for _, ext := range []string{".jpg", ".jpeg", ".JPG", ".JPEG"} {
				if images[base+ext] {
					total--
				}
			}
		}
	}
	if total > 10 {
		return nil, fmt.Errorf("%s 最多 10 張截圖（目前 %d 張，再加 %d 張會超過）：先刪掉不要的，或用 -replace 從 01 開始覆寫", dir, len(images), n)
	}
	return names, nil
}

// findChrome 找本機的 Chrome／Chromium：參數 > PT_CHROME > 常見位置。
func findChrome(flagPath string) (string, error) {
	for _, c := range []string{flagPath, os.Getenv("PT_CHROME")} {
		if c == "" {
			continue
		}
		if st, err := os.Stat(c); err == nil && !st.IsDir() {
			return c, nil
		}
		if lp, err := exec.LookPath(c); err == nil {
			return lp, nil
		}
		return "", fmt.Errorf("找不到指定的 Chrome：%s", c)
	}
	var candidates []string
	switch runtime.GOOS {
	case "darwin":
		for _, app := range []string{"Google Chrome.app/Contents/MacOS/Google Chrome", "Chromium.app/Contents/MacOS/Chromium",
			"Google Chrome for Testing.app/Contents/MacOS/Google Chrome for Testing", "Microsoft Edge.app/Contents/MacOS/Microsoft Edge"} {
			candidates = append(candidates, filepath.Join("/Applications", app))
			if home, err := os.UserHomeDir(); err == nil {
				candidates = append(candidates, filepath.Join(home, "Applications", app))
			}
		}
	case "windows":
		for _, env := range []string{"ProgramFiles", "ProgramFiles(x86)", "LocalAppData"} {
			if d := os.Getenv(env); d != "" {
				candidates = append(candidates, filepath.Join(d, `Google\Chrome\Application\chrome.exe`), filepath.Join(d, `Chromium\Application\chrome.exe`))
			}
		}
	}
	for _, c := range candidates {
		if st, err := os.Stat(c); err == nil && !st.IsDir() {
			return c, nil
		}
	}
	for _, name := range []string{"google-chrome", "google-chrome-stable", "chromium", "chromium-browser", "chrome"} {
		if lp, err := exec.LookPath(name); err == nil {
			return lp, nil
		}
	}
	return "", errors.New("找不到 Chrome 或 Chromium：pt store screenshot 用本機瀏覽器的無頭模式擷取畫面。" +
		"請安裝 Google Chrome（macOS 放在 /Applications；Linux 要有 google-chrome、chromium 或 chromium-browser 指令），" +
		"或用 -chrome <執行檔路徑>（環境變數 PT_CHROME）指定；也可以用手機或瀏覽器的手機模擬模式手動截圖")
}

// chromeShotArgs 是擷取一張截圖的 Chrome 參數。
func chromeShotArgs(userDir, out, target string, w, h int, wait time.Duration) []string {
	return []string{
		"--headless=new",
		"--user-data-dir=" + userDir,
		"--no-first-run", "--no-default-browser-check", "--disable-extensions", "--disable-sync",
		"--disable-background-networking", "--disable-component-update", "--mute-audio",
		// 全螢幕（沒有視窗邊框與工具列）＋模擬手機螢幕：遊戲看到的視窗就是 w/3 × h/3 CSS px
		"--kiosk", fmt.Sprintf("--screen-info={%dx%d devicePixelRatio=%d}", w, h, shotScale),
		"--hide-scrollbars",
		// 沒有 GPU 的環境（CI、遠端主機）也能畫 WebGL
		"--use-angle=swiftshader", "--enable-unsafe-swiftshader",
		fmt.Sprintf("--virtual-time-budget=%d", wait.Milliseconds()),
		"--screenshot=" + out,
		target,
	}
}

// runChromeShot 啟動 Chrome 擷取一張截圖（暫存的使用者資料夾，結束後刪除）。
var runChromeShot = func(chrome, out, target string, w, h int, wait time.Duration) error {
	userDir, err := os.MkdirTemp("", "pt-chrome-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(userDir)
	// 虛擬時間通常比實際時間快；網路請求、WebSocket 會讓它變慢，所以實際時間另外設上限
	ctx, cancel := context.WithTimeout(context.Background(), wait+60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, chrome, chromeShotArgs(userDir, out, target, w, h, wait)...)
	var stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stderr, &stderr
	err = cmd.Run()
	if ctx.Err() != nil {
		return fmt.Errorf("Chrome 超過 %s 沒有完成（遊戲一直在載入？）", wait+60*time.Second)
	}
	if _, statErr := os.Stat(out); statErr != nil {
		msg := strings.TrimSpace(stderr.String())
		if lines := strings.Split(msg, "\n"); len(lines) > 5 {
			msg = strings.Join(lines[len(lines)-5:], "\n")
		}
		if err == nil {
			err = errors.New("沒有產生截圖")
		}
		return fmt.Errorf("Chrome 擷取失敗：%v\n%s", err, msg)
	}
	return nil
}

// checkShot 確認截圖的尺寸正確、不是全黑或全白（取樣 32×32 個點）。
func checkShot(data []byte, w, h int) error {
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("截圖不是 PNG：%w", err)
	}
	b := img.Bounds()
	if b.Dx() != w || b.Dy() != h {
		return fmt.Errorf("截圖是 %d×%d，不是需要的 %d×%d：這個 Chrome 可能太舊（不支援 --screen-info），請更新 Chrome 後再試", b.Dx(), b.Dy(), w, h)
	}
	switch blankShot(img) {
	case "black":
		return errors.New("截圖幾乎全黑：可能是 WebGL 沒有啟動，或遊戲還沒載入完；請加大 -wait（例如 -wait 10s）再試")
	case "white":
		return errors.New("截圖幾乎全白：遊戲可能還沒載入完，或網址不對；請確認網址能在瀏覽器打開，再加大 -wait 試試")
	}
	return nil
}

// blankShot 取樣判斷圖片是不是幾乎全黑或全白（回傳 "black"、"white" 或 ""）。
func blankShot(img image.Image) string {
	b := img.Bounds()
	const grid = 32
	dark, light, total := 0, 0, 0
	for i := 0; i < grid; i++ {
		for j := 0; j < grid; j++ {
			x := b.Min.X + (2*i+1)*b.Dx()/(2*grid)
			y := b.Min.Y + (2*j+1)*b.Dy()/(2*grid)
			r, g, bl, _ := img.At(x, y).RGBA()
			hi, lo := max(r, g, bl)>>8, min(r, g, bl)>>8
			total++
			if hi <= 16 {
				dark++
			}
			if lo >= 240 {
				light++
			}
		}
	}
	switch {
	case dark*100 >= total*99:
		return "black"
	case light*100 >= total*99:
		return "white"
	}
	return ""
}

func cmdStoreScreenshot(args []string) error {
	p, err := findProject()
	if err != nil {
		return err
	}
	o, err := parseShotArgs(args, p)
	if err != nil {
		return err
	}
	chrome, err := findChrome(o.Chrome)
	if err != nil {
		return err
	}
	{
		client := http.Client{Timeout: 5 * time.Second}
		res, err := client.Get(o.URL)
		if err != nil {
			return fmt.Errorf("連不到 %s：先啟動遊戲（pt new 的專案用 pt dev -mock；用 pt dev 時把它印出的玩家分頁網址傳給 -url；其他專案用自己的啟動方式），再執行 pt store screenshot", o.URL)
		}
		res.Body.Close()
	}
	if err := os.MkdirAll(o.Out, 0o755); err != nil {
		return err
	}
	names, err := nextShotNames(o.Out, o.N, o.Replace)
	if err != nil {
		return err
	}
	w, h, cssW, cssH := shotSize(o.Orientation)
	fmt.Printf("用 %s 擷取 %s（%s，手機畫面 %d×%d，輸出 %d×%d）\n", filepath.Base(chrome), o.URL, orientationLabel[o.Orientation], cssW, cssH, w, h)
	var saved []string
	for i, name := range names {
		wait := o.Wait + time.Duration(i)*o.Interval
		dst := filepath.Join(o.Out, name)
		tmp, err := os.CreateTemp(o.Out, ".pt-shot-*.png")
		if err != nil {
			return err
		}
		tmp.Close()
		os.Remove(tmp.Name()) // Chrome 自己建立檔案
		err = runChromeShot(chrome, tmp.Name(), o.URL, w, h, wait)
		var data []byte
		if err == nil {
			data, err = os.ReadFile(tmp.Name())
		}
		if err == nil {
			err = checkShot(data, w, h)
		}
		if err != nil {
			os.Remove(tmp.Name())
			if len(saved) > 0 {
				fmt.Printf("已存 %s\n", strings.Join(saved, "、"))
			}
			return fmt.Errorf("第 %d 張（等待 %s）沒有存檔：%w", i+1, wait, err)
		}
		if o.Replace {
			base := strings.TrimSuffix(dst, ".png")
			for _, ext := range []string{".jpg", ".jpeg", ".JPG", ".JPEG"} {
				os.Remove(base + ext)
			}
		}
		if err := os.Rename(tmp.Name(), dst); err != nil {
			os.Remove(tmp.Name())
			return err
		}
		rel := dst
		if p != nil {
			if r, err := filepath.Rel(p.dir, dst); err == nil {
				rel = r
			}
		}
		saved = append(saved, filepath.ToSlash(rel))
		fmt.Printf("  ✓ %s（等待 %s）\n", filepath.ToSlash(rel), wait)
	}
	fmt.Println("請親自看過每張截圖（畫面正確、沒有錯誤訊息或除錯資訊），不好的刪掉重拍；截圖說明寫在 store/listing.json 的 screenshotCaptions。")
	return nil
}
