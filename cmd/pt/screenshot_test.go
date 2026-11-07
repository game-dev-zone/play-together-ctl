package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"
)

// shotProject 建立只有 playtogether.json 的專案並切換到那裡
func shotProject(t *testing.T, orientation string) string {
	t.Helper()
	dir := t.TempDir()
	must(t, os.WriteFile(filepath.Join(dir, projectFile), []byte(fmt.Sprintf(`{"name":"shot-game","orientation":%q,"devAddr":"127.0.0.1:18555"}`, orientation)), 0o644))
	t.Chdir(dir)
	return dir
}

func TestScreenshotParseArgs(t *testing.T) {
	dir := shotProject(t, "landscape")
	p, err := findProject()
	must(t, err)

	o, err := parseShotArgs(nil, p)
	must(t, err)
	if o.URL != "http://127.0.0.1:18555/" || o.Wait != 5*time.Second || o.N != 1 || o.Interval != 3*time.Second ||
		o.Orientation != "landscape" || o.Out != filepath.Join(dir, "store", "screenshots") || o.Locale != "" || o.Replace {
		t.Fatalf("預設值：%+v", o)
	}
	o, err = parseShotArgs([]string{"-locale", "en", "-n", "3", "-interval", "2s", "-wait", "8s", "-url", "http://localhost:3000/?demo=1", "-replace"}, p)
	must(t, err)
	if o.Out != filepath.Join(dir, "store", "screenshots", "en") || o.N != 3 || o.Wait != 8*time.Second || o.URL != "http://localhost:3000/?demo=1" || !o.Replace {
		t.Fatalf("參數：%+v", o)
	}
	if o, err = parseShotArgs([]string{"-o", "shots"}, p); err != nil || o.Out != "shots" {
		t.Fatalf("-o：%+v %v", o, err)
	}
	for _, bad := range [][]string{
		{"-orientation", "portrait"}, // 和專案的方向不同
		{"-orientation", "square"},
		{"-n", "0"}, {"-n", "11"},
		{"-wait", "-1s"},
		{"-wait", "100s", "-n", "10", "-interval", "10s"}, // 最長等待超過 2 分鐘
		{"-locale", "ja"},
		{"-url", "ftp://example.com/"}, {"-url", "localhost:8090"},
		{"extra"},
	} {
		if _, err := parseShotArgs(bad, p); err == nil {
			t.Errorf("%v 應該被拒絕", bad)
		}
	}

	// any：預設直式，可以改橫式
	p.Orientation = "any"
	if o, _ := parseShotArgs(nil, p); o.Orientation != "portrait" {
		t.Fatalf("any 預設直式：%s", o.Orientation)
	}
	if o, err := parseShotArgs([]string{"-orientation", "landscape"}, p); err != nil || o.Orientation != "landscape" {
		t.Fatalf("any 可以指定橫式：%v", err)
	}
	// 沒有方向（舊專案）：直式
	p.Orientation = ""
	if o, _ := parseShotArgs(nil, p); o.Orientation != "portrait" {
		t.Fatalf("沒有方向：%s", o.Orientation)
	}
	// 不在專案裡：一定要 -url
	if _, err := parseShotArgs(nil, nil); err == nil || !strings.Contains(err.Error(), "-url") {
		t.Fatalf("不在專案裡：%v", err)
	}
	if o, err := parseShotArgs([]string{"-url", "http://127.0.0.1:9/"}, nil); err != nil || o.Out != filepath.Join(".", "store", "screenshots") {
		t.Fatalf("不在專案裡：%+v %v", o, err)
	}
}

// 輸出尺寸和 storeImageSpec 的截圖規格一致；手機寬 390 CSS px，擷取範圍 × 倍率 = 輸出像素
func TestScreenshotSize(t *testing.T) {
	for _, c := range []struct {
		orientation    string
		w, h, cssW, cH int
	}{{"portrait", 1080, 1920, 390, 694}, {"landscape", 1920, 1080, 694, 390}} {
		w, h, plan := shotSize(c.orientation)
		if w != c.w || h != c.h || plan.CSSW != c.cssW || plan.CSSH != c.cH {
			t.Errorf("%s：%d×%d（%d×%d）", c.orientation, w, h, plan.CSSW, plan.CSSH)
		}
		if math.Round(plan.Clip[0]*plan.Scale) != float64(w) || math.Round(plan.Clip[1]*plan.Scale) != float64(h) ||
			plan.Clip[0] > float64(plan.CSSW) || plan.Clip[1] > float64(plan.CSSH) {
			t.Errorf("%s：擷取範圍 %v × %v", c.orientation, plan.Clip, plan.Scale)
		}
		sw, sh, _ := storeImageSpec("screenshot", c.orientation, w, h)
		if sw != w || sh != h {
			t.Errorf("%s 和平台規格不同", c.orientation)
		}
	}
	args := strings.Join(chromeArgs("/tmp/u"), " ")
	for _, want := range []string{"--headless=new", "--remote-debugging-pipe", "--user-data-dir=/tmp/u", "--use-angle=swiftshader", "--enable-unsafe-swiftshader", "--disable-background-timer-throttling"} {
		if !strings.Contains(args, want) {
			t.Errorf("Chrome 參數少了 %s：%s", want, args)
		}
	}
	for _, bad := range []string{"--screenshot", "--virtual-time-budget", "--screen-info"} {
		if strings.Contains(args, bad) {
			t.Errorf("不能再用 %s（虛擬時間、每張重新載入）", bad)
		}
	}
}

func TestScreenshotNames(t *testing.T) {
	dir := t.TempDir()
	names, err := nextShotNames(filepath.Join(dir, "missing"), 2, false)
	if err != nil || strings.Join(names, ",") != "01.png,02.png" {
		t.Fatalf("空資料夾：%v %v", names, err)
	}
	for _, f := range []string{"01.png", "03.jpg", "hero.png", "README.md"} {
		must(t, os.WriteFile(filepath.Join(dir, f), []byte("x"), 0o644))
	}
	must(t, os.MkdirAll(filepath.Join(dir, "en"), 0o755))
	if names, err = nextShotNames(dir, 3, false); err != nil || strings.Join(names, ",") != "04.png,05.png,06.png" {
		t.Fatalf("接在最大編號後面：%v %v", names, err)
	}
	if names, err = nextShotNames(dir, 2, true); err != nil || strings.Join(names, ",") != "01.png,02.png" {
		t.Fatalf("-replace：%v %v", names, err)
	}
	// 已有 3 張圖片（含 hero.png）：再加 8 張超過 10 張
	if _, err := nextShotNames(dir, 8, false); err == nil || !strings.Contains(err.Error(), "10") {
		t.Fatalf("超過 10 張：%v", err)
	}
	if _, err := nextShotNames(dir, 7, false); err != nil {
		t.Fatalf("剛好 10 張：%v", err)
	}
	// -replace：01–03 取代既有的 01.png 與 03.jpg，加上 hero.png 共 10 張以內
	if _, err := nextShotNames(dir, 9, true); err != nil {
		t.Fatalf("-replace 9 張：%v", err)
	}
	if _, err := nextShotNames(dir, 10, true); err == nil {
		t.Fatal("-replace 10 張加上 hero.png 超過 10 張")
	}
}

func encodePNG(t *testing.T, img image.Image) []byte {
	t.Helper()
	var b bytes.Buffer
	must(t, png.Encode(&b, img))
	return b.Bytes()
}

func solidPNG(t *testing.T, w, h int, c color.Color) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, c)
		}
	}
	return encodePNG(t, img)
}

func gamePNG(t *testing.T, w, h int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{uint8(x * 255 / w), uint8(y * 255 / h), 120, 255})
		}
	}
	return encodePNG(t, img)
}

func TestScreenshotCheck(t *testing.T) {
	if err := checkShot(gamePNG(t, 108, 192), 108, 192); err != nil {
		t.Fatalf("正常畫面：%v", err)
	}
	if err := checkShot(solidPNG(t, 108, 192, color.Black), 108, 192); err == nil || !strings.Contains(err.Error(), "-wait") || !strings.Contains(err.Error(), "WebGL") {
		t.Fatalf("全黑：%v", err)
	}
	if err := checkShot(solidPNG(t, 108, 192, color.RGBA{250, 250, 250, 255}), 108, 192); err == nil || !strings.Contains(err.Error(), "全白") {
		t.Fatalf("全白：%v", err)
	}
	// 深色畫面上有一些內容：不算全黑
	img := image.NewRGBA(image.Rect(0, 0, 108, 192))
	for y := 0; y < 192; y++ {
		for x := 0; x < 108; x++ {
			c := color.RGBA{5, 5, 8, 255}
			if y > 80 && y < 120 {
				c = color.RGBA{255, 200, 0, 255}
			}
			img.Set(x, y, c)
		}
	}
	if err := checkShot(encodePNG(t, img), 108, 192); err != nil {
		t.Fatalf("深色但有內容：%v", err)
	}
	if err := checkShot(gamePNG(t, 108, 191), 108, 192); err == nil || !strings.Contains(err.Error(), "108×191") {
		t.Fatalf("尺寸不對：%v", err)
	}
	if err := checkShot([]byte("not png"), 108, 192); err == nil {
		t.Fatal("不是 PNG")
	}
}

// fakeChrome：測試執行檔被當成 Chrome 啟動時，從 fd 3 讀 CDP 指令、寫回應到 fd 4（每則以 \0 結尾）。
// 模式：ok、blank（擷取全黑）、crash（打開網址後異常結束）、hang（擷取不回應）、naverr（打不開網址）。
// 收到的指令與時間寫到 PT_FAKE_CHROME_LOG（每行一則 JSON）。
func fakeChrome(mode string) {
	in, out := os.NewFile(3, "cdp-in"), os.NewFile(4, "cdp-out")
	if in == nil || out == nil {
		os.Exit(9)
	}
	logf, _ := os.OpenFile(os.Getenv("PT_FAKE_CHROME_LOG"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	br := bufio.NewReader(in)
	for {
		raw, err := br.ReadBytes(0)
		if err != nil {
			return
		}
		var m struct {
			ID     int             `json:"id"`
			Method string          `json:"method"`
			Params json.RawMessage `json:"params"`
		}
		_ = json.Unmarshal(bytes.TrimSuffix(raw, []byte{0}), &m)
		if logf != nil {
			line, _ := json.Marshal(map[string]any{"method": m.Method, "params": m.Params, "at": time.Now().UnixMilli()})
			logf.Write(append(line, '\n'))
		}
		result := map[string]any{}
		switch m.Method {
		case "Target.createTarget":
			result["targetId"] = "T1"
		case "Target.attachToTarget":
			result["sessionId"] = "S1"
		case "Page.navigate":
			if mode == "naverr" {
				result["errorText"] = "net::ERR_CONNECTION_REFUSED"
			}
			if mode == "crash" {
				os.Stderr.WriteString("[fake] GPU process crashed\n")
				os.Exit(3)
			}
		case "Page.captureScreenshot":
			if mode == "hang" {
				continue
			}
			var p struct {
				Clip struct{ Width, Height float64 } `json:"clip"`
			}
			_ = json.Unmarshal(m.Params, &p)
			w, h := 1080, 1920
			if p.Clip.Width > p.Clip.Height {
				w, h = 1920, 1080
			}
			img := image.NewRGBA(image.Rect(0, 0, w, h))
			for y := 0; y < h; y += 1 {
				for x := 0; x < w; x += 1 {
					if mode == "blank" {
						img.Set(x, y, color.Black)
					} else {
						img.Set(x, y, color.RGBA{uint8(x), uint8(y), 120, 255})
					}
				}
			}
			var b bytes.Buffer
			_ = png.Encode(&b, img)
			result["data"] = b.Bytes() // encoding/json 編成 base64
		}
		resp, _ := json.Marshal(map[string]any{"id": m.ID, "result": result})
		out.Write(append(resp, 0))
		if m.Method == "Browser.close" {
			return
		}
	}
}

type cdpCall struct {
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
	At     int64           `json:"at"`
}

func useFakeChrome(t *testing.T, mode string) func() []cdpCall {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("Windows 不支援以管道控制 Chrome")
	}
	log := filepath.Join(t.TempDir(), "calls.log")
	t.Setenv("PT_CHROME", os.Args[0])
	t.Setenv("PT_FAKE_CHROME", mode)
	t.Setenv("PT_FAKE_CHROME_LOG", log)
	return func() []cdpCall {
		raw, _ := os.ReadFile(log)
		var calls []cdpCall
		for _, line := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
			var c cdpCall
			if json.Unmarshal([]byte(line), &c) == nil {
				calls = append(calls, c)
			}
		}
		return calls
	}
}

// 用假的 Chrome（CDP over pipe）跑完整個指令：同一個頁面、真實時間的間隔、手機模擬、檔名遞增、不覆寫、全黑不存檔
func TestScreenshotCommand(t *testing.T) {
	dir := shotProject(t, "portrait")
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, "<!doctype html>") }))
	defer site.Close()
	calls := useFakeChrome(t, "ok")
	shots := filepath.Join(dir, "store", "screenshots")
	must(t, os.MkdirAll(shots, 0o755))
	must(t, os.WriteFile(filepath.Join(shots, "01.png"), []byte("keep"), 0o644))

	must(t, cmdStoreScreenshot([]string{"-url", site.URL + "/", "-n", "3", "-wait", "200ms", "-interval", "300ms"}))
	var navs, metrics int
	var shotsAt []int64
	var navAt int64
	for _, c := range calls() {
		switch c.Method {
		case "Page.navigate":
			navs++
			navAt = c.At
			if !strings.Contains(string(c.Params), site.URL) {
				t.Errorf("網址：%s", c.Params)
			}
		case "Emulation.setDeviceMetricsOverride":
			metrics++
			var p struct {
				Width, Height     int
				DeviceScaleFactor float64
				Mobile            bool
			}
			must(t, json.Unmarshal(c.Params, &p))
			if p.Width != 390 || p.Height != 694 || !p.Mobile || math.Abs(p.DeviceScaleFactor-1080.0/390) > 1e-9 {
				t.Errorf("手機模擬：%+v", p)
			}
		case "Page.captureScreenshot":
			shotsAt = append(shotsAt, c.At)
			if !strings.Contains(string(c.Params), `"clip"`) {
				t.Errorf("要擷取固定範圍：%s", c.Params)
			}
		}
	}
	if navs != 1 || metrics != 1 || len(shotsAt) != 3 {
		t.Fatalf("同一個頁面擷取 3 張（只打開一次）：navigate %d、metrics %d、擷取 %d", navs, metrics, len(shotsAt))
	}
	// 真實時間：第一張在 -wait 之後，之後每張至少隔 -interval
	if shotsAt[0]-navAt < 200 || shotsAt[1]-shotsAt[0] < 300 || shotsAt[2]-shotsAt[1] < 300 {
		t.Fatalf("等待照真實時間：打開 %d、擷取 %v", navAt, shotsAt)
	}
	if raw, _ := os.ReadFile(filepath.Join(shots, "01.png")); string(raw) != "keep" {
		t.Fatal("不能覆寫既有截圖")
	}
	for _, f := range []string{"02.png", "03.png", "04.png"} {
		raw, err := os.ReadFile(filepath.Join(shots, f))
		if err != nil {
			t.Fatalf("沒有 %s", f)
		}
		if cfg, _, err := image.DecodeConfig(bytes.NewReader(raw)); err != nil || cfg.Width != 1080 || cfg.Height != 1920 {
			t.Fatalf("%s 尺寸：%+v %v", f, cfg, err)
		}
	}
	// 全黑：不存檔、說明加大 -wait；資料夾裡不留下暫存檔
	useFakeChrome(t, "blank")
	if err := cmdStoreScreenshot([]string{"-url", site.URL + "/", "-wait", "0s"}); err == nil || !strings.Contains(err.Error(), "-wait") {
		t.Fatalf("全黑：%v", err)
	}
	entries, _ := os.ReadDir(shots)
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	if strings.Join(names, ",") != "01.png,02.png,03.png,04.png" {
		t.Fatalf("全黑不能存檔、不能留下暫存檔：%v", names)
	}
	// -replace -locale en：從 01 開始，存到 en/
	useFakeChrome(t, "ok")
	must(t, cmdStoreScreenshot([]string{"-url", site.URL + "/", "-locale", "en", "-replace", "-wait", "0s"}))
	if _, err := os.Stat(filepath.Join(shots, "en", "01.png")); err != nil {
		t.Fatal("-locale en 要存到 screenshots/en/01.png")
	}
	// pt check 認得擷取的截圖（尺寸、比例都對）
	var fs storeFindings
	p, _ := findProject()
	fillStore(t, dir)
	must(t, os.Remove(filepath.Join(shots, "02.png")))
	must(t, os.WriteFile(filepath.Join(shots, "01.png"), gamePNG(t, 1080, 1920), 0o644))
	_, imgs, _, ready := readStore(p, func(level, format string, a ...any) { fs = append(fs, finding{level, fmt.Sprintf(format, a...)}) })
	fs = slices.DeleteFunc(fs, func(f finding) bool { return f.level == levelWarn && strings.Contains(f.msg, "沒有封面") }) // 這裡只測截圖
	if len(fs) != 0 || !ready || len(imgs) != 5 {                                                                       // 圖示、共用截圖 01、03、04、英文截圖 01（02 刪掉了）
		t.Errorf("擷取的截圖不該有問題：%v %d", fs, len(imgs))
	}
	// 連不到網址：說明要先啟動遊戲
	site.Close()
	if err := cmdStoreScreenshot([]string{"-url", site.URL + "/"}); err == nil || !strings.Contains(err.Error(), "pt dev") {
		t.Fatalf("連不到：%v", err)
	}
	// 找不到 Chrome
	t.Setenv("PT_CHROME", filepath.Join(dir, "no-chrome"))
	if err := cmdStoreScreenshot(nil); err == nil || !strings.Contains(err.Error(), "Chrome") {
		t.Fatalf("找不到 Chrome：%v", err)
	}
}

// Chrome 異常結束、不回應、打不開網址：清楚報錯，而且一定會結束
func TestScreenshotChromeFailures(t *testing.T) {
	plan := func() shotPlan {
		_, _, sp := shotSize("portrait")
		sp.URL, sp.N, sp.ExtraTimeout = "http://127.0.0.1:1/", 2, 2*time.Second
		return sp
	}
	never := func(int, []byte) error { t.Error("不該拍到截圖"); return nil }

	useFakeChrome(t, "crash")
	err := captureWithChrome(os.Args[0], plan(), never)
	if err == nil || !strings.Contains(err.Error(), "Chrome 異常結束") || !strings.Contains(err.Error(), "GPU process crashed") {
		t.Fatalf("異常結束要附上 Chrome 的輸出：%v", err)
	}
	useFakeChrome(t, "naverr")
	if err := captureWithChrome(os.Args[0], plan(), never); err == nil || !strings.Contains(err.Error(), "ERR_CONNECTION_REFUSED") {
		t.Fatalf("打不開網址：%v", err)
	}
	useFakeChrome(t, "hang")
	start := time.Now()
	if err := captureWithChrome(os.Args[0], plan(), never); err == nil || !strings.Contains(err.Error(), "還沒完成") {
		t.Fatalf("不回應要逾時：%v", err)
	}
	if time.Since(start) > 15*time.Second {
		t.Fatalf("逾時後要結束 Chrome：花了 %s", time.Since(start))
	}
	// onShot 回傳錯誤時停止，不再擷取
	calls := useFakeChrome(t, "ok")
	sp := plan()
	sp.N, sp.Interval = 3, 10*time.Millisecond
	n := 0
	err = captureWithChrome(os.Args[0], sp, func(i int, _ []byte) error { n++; return fmt.Errorf("停") })
	if err == nil || n != 1 {
		t.Fatalf("onShot 錯誤：%v %d", err, n)
	}
	closed := false
	for _, c := range calls() {
		closed = closed || c.Method == "Browser.close"
	}
	if !closed {
		t.Fatal("結束時要關閉 Chrome")
	}
}

// pt check：截圖方向和專案不符時，提示用 pt store screenshot 重新擷取
func TestScreenshotCheckHint(t *testing.T) {
	dir := shotProject(t, "portrait")
	shots := filepath.Join(dir, "store", "screenshots")
	must(t, os.MkdirAll(filepath.Join(shots, "en"), 0o755))
	fillStore(t, dir)
	must(t, os.Remove(filepath.Join(shots, "02.png")))
	must(t, os.WriteFile(filepath.Join(shots, "01.png"), pngOf(t, 1920, 1080), 0o644))
	must(t, os.WriteFile(filepath.Join(shots, "en", "01.png"), pngOf(t, 1920, 1080), 0o644))
	must(t, os.WriteFile(filepath.Join(dir, "store", "icon.png"), pngOf(t, 1920, 1080), 0o644))
	p, _ := findProject()
	var fs storeFindings
	readStore(p, func(level, format string, a ...any) { fs = append(fs, finding{level, fmt.Sprintf(format, a...)}) })
	if !fs.has(levelSchema, "screenshots/01.png 的比例") || !fs.has(levelSchema, "pt store screenshot 重新擷取") || !fs.has(levelSchema, "pt store screenshot -locale en") {
		t.Fatalf("要提示 pt store screenshot：%v", fs)
	}
	for _, f := range fs {
		if strings.Contains(f.msg, "icon.png") && strings.Contains(f.msg, "pt store screenshot") {
			t.Fatalf("圖示不該提示截圖指令：%v", f)
		}
	}
}

// 實際用本機的 Chrome 擷取（PT_TEST_CHROME=1 才跑；PT_TEST_SHOT_DIR 可以保留輸出）：
// WebGL、手機視窗大小、畫面高度完整，以及等待是真實時間（頁面 3 秒後才變色，第二張要拍到變色後的畫面）。
func TestScreenshotChrome(t *testing.T) {
	if os.Getenv("PT_TEST_CHROME") != "1" {
		t.Skip("設定 PT_TEST_CHROME=1 才實際啟動 Chrome")
	}
	if _, err := findChrome(""); err != nil {
		t.Fatal(err)
	}
	// 依視窗大小畫 WebGL 背景與 2D 文字；底部的紅條確認畫面高度完整。
	// 頁面打開 3 秒（真實時間，用 Date.now 判斷）後背景從藍色變成綠色。
	page := `<!doctype html><html><head><meta name="viewport" content="width=device-width,initial-scale=1"></head>
<body style="margin:0;background:#000;overflow:hidden">
<canvas id="gl" style="position:fixed;inset:0;width:100vw;height:100vh"></canvas>
<canvas id="ui" style="position:fixed;inset:0;width:100vw;height:100vh"></canvas>
<script>
const W = innerWidth, H = innerHeight, d = devicePixelRatio, t0 = Date.now();
for (const c of document.querySelectorAll('canvas')) { c.width = Math.round(W * d); c.height = Math.round(H * d); }
const gl = document.getElementById('gl').getContext('webgl');
const x = document.getElementById('ui').getContext('2d');
x.scale(d, d);
function frame() {
  const late = Date.now() - t0 >= 3000;
  if (gl) { if (late) gl.clearColor(0.2, 0.75, 0.3, 1); else gl.clearColor(0.2, 0.55, 0.85, 1); gl.clear(gl.COLOR_BUFFER_BIT); }
  x.clearRect(0, 0, W, H);
  x.fillStyle = '#ffd34d'; x.beginPath(); x.arc(W / 2, H / 2, Math.min(W, H) / 4, 0, Math.PI * 2); x.fill();
  x.fillStyle = '#d93a3a'; x.fillRect(0, H - 20, W, 20);
  x.fillStyle = '#fff'; x.font = '20px sans-serif';
  x.fillText(W + '×' + H + ' @' + d.toFixed(3) + (gl ? ' WebGL' : ' 沒有 WebGL') + ' ' + ((Date.now() - t0) / 1000).toFixed(1) + 's', 12, 32);
  requestAnimationFrame(frame);
}
frame();
</script></body></html>`
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, page)
	}))
	defer site.Close()
	blue, green := [3]uint32{51, 140, 217}, [3]uint32{51, 191, 77}
	for _, orientation := range []string{"portrait", "landscape"} {
		dir := shotProject(t, orientation)
		must(t, cmdStoreScreenshot([]string{"-url", site.URL + "/", "-wait", "1s", "-n", "2", "-interval", "3s"}))
		w, h, _ := shotSize(orientation)
		for i, bg := range [][3]uint32{blue, green} {
			f := filepath.Join(dir, "store", "screenshots", fmt.Sprintf("%02d.png", i+1))
			raw, err := os.ReadFile(f)
			must(t, err)
			img, err := png.Decode(bytes.NewReader(raw))
			must(t, err)
			if img.Bounds().Dx() != w || img.Bounds().Dy() != h {
				t.Fatalf("%s：%v", orientation, img.Bounds())
			}
			near := func(x, y int, c [3]uint32) bool {
				cr, cg, cb, _ := img.At(x, y).RGBA()
				diff := func(a, b uint32) uint32 {
					if a > b {
						return a - b
					}
					return b - a
				}
				return diff(cr>>8, c[0])+diff(cg>>8, c[1])+diff(cb>>8, c[2]) < 60
			}
			// 第一張：WebGL 背景是藍色；第二張（真實時間 3 秒後）：綠色
			if !near(w-5, 100, bg) {
				t.Errorf("%s 第 %d 張：背景要是 %v（真實時間），實際 %v", orientation, i+1, bg, img.At(w-5, 100))
			}
			if !near(w/2, h/2, [3]uint32{255, 211, 77}) {
				t.Errorf("%s：中間不是黃色的圓：%v", orientation, img.At(w/2, h/2))
			}
			if !near(w/2, h-2, [3]uint32{217, 58, 58}) {
				t.Errorf("%s：最底下不是紅條（畫面高度被截掉？）：%v", orientation, img.At(w/2, h-2))
			}
			if keep := os.Getenv("PT_TEST_SHOT_DIR"); keep != "" {
				must(t, os.MkdirAll(keep, 0o755))
				must(t, os.WriteFile(filepath.Join(keep, fmt.Sprintf("chrome-%s-%d.png", orientation, i+1)), raw, 0o644))
			}
		}
	}
}
