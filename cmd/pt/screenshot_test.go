package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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

// 輸出尺寸和 storeImageSpec 的截圖規格一致；CSS 視窗是整數倍
func TestScreenshotSize(t *testing.T) {
	for _, c := range []struct {
		orientation    string
		w, h, cssW, cH int
	}{{"portrait", 1080, 1920, 360, 640}, {"landscape", 1920, 1080, 640, 360}} {
		w, h, cssW, cssH := shotSize(c.orientation)
		if w != c.w || h != c.h || cssW != c.cssW || cssH != c.cH || cssW*shotScale != w || cssH*shotScale != h {
			t.Errorf("%s：%d×%d（%d×%d）", c.orientation, w, h, cssW, cssH)
		}
		sw, sh, _ := storeImageSpec("screenshot", c.orientation, w, h)
		if sw != w || sh != h {
			t.Errorf("%s 和平台規格不同", c.orientation)
		}
	}
	args := strings.Join(chromeShotArgs("/tmp/u", "/tmp/o.png", "http://x/", 1080, 1920, 5*time.Second), " ")
	for _, want := range []string{"--headless=new", "--screen-info={1080x1920 devicePixelRatio=3}", "--kiosk", "--virtual-time-budget=5000", "--screenshot=/tmp/o.png", "--user-data-dir=/tmp/u", "--use-angle=swiftshader", "http://x/"} {
		if !strings.Contains(args, want) {
			t.Errorf("Chrome 參數少了 %s：%s", want, args)
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

// 用假的 Chrome 跑完整個指令：檔名遞增、不覆寫、全黑不存檔、連不到網址時說明怎麼啟動
func TestScreenshotCommand(t *testing.T) {
	dir := shotProject(t, "portrait")
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, "<!doctype html>") }))
	defer site.Close()
	t.Setenv("PT_CHROME", os.Args[0]) // 任何存在的執行檔；實際不會執行
	var waits []time.Duration
	blank := false
	orig := runChromeShot
	t.Cleanup(func() { runChromeShot = orig })
	runChromeShot = func(chrome, out, target string, w, h int, wait time.Duration) error {
		waits = append(waits, wait)
		if target != site.URL+"/" || w != 1080 || h != 1920 {
			t.Errorf("Chrome 參數：%s %d×%d", target, w, h)
		}
		if blank {
			return os.WriteFile(out, solidPNG(t, w, h, color.Black), 0o644)
		}
		return os.WriteFile(out, gamePNG(t, w, h), 0o644)
	}
	shots := filepath.Join(dir, "store", "screenshots")
	must(t, os.MkdirAll(shots, 0o755))
	must(t, os.WriteFile(filepath.Join(shots, "01.png"), []byte("keep"), 0o644))

	must(t, cmdStoreScreenshot([]string{"-url", site.URL + "/", "-n", "2", "-wait", "1s", "-interval", "2s"}))
	if len(waits) != 2 || waits[0] != time.Second || waits[1] != 3*time.Second {
		t.Fatalf("等待時間要遞增：%v", waits)
	}
	if raw, _ := os.ReadFile(filepath.Join(shots, "01.png")); string(raw) != "keep" {
		t.Fatal("不能覆寫既有截圖")
	}
	for _, f := range []string{"02.png", "03.png"} {
		raw, err := os.ReadFile(filepath.Join(shots, f))
		if err != nil {
			t.Fatalf("沒有 %s", f)
		}
		if cfg, _, err := image.DecodeConfig(bytes.NewReader(raw)); err != nil || cfg.Width != 1080 || cfg.Height != 1920 {
			t.Fatalf("%s 尺寸：%+v %v", f, cfg, err)
		}
	}
	// 全黑：不存檔、說明加大 -wait；資料夾裡不留下暫存檔
	blank = true
	if err := cmdStoreScreenshot([]string{"-url", site.URL + "/"}); err == nil || !strings.Contains(err.Error(), "-wait") {
		t.Fatalf("全黑：%v", err)
	}
	entries, _ := os.ReadDir(shots)
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	if strings.Join(names, ",") != "01.png,02.png,03.png" {
		t.Fatalf("全黑不能存檔、不能留下暫存檔：%v", names)
	}
	// -replace -locale en：從 01 開始，存到 en/
	blank = false
	must(t, cmdStoreScreenshot([]string{"-url", site.URL + "/", "-locale", "en", "-replace"}))
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
	if len(fs) != 0 || !ready || len(imgs) != 4 { // 圖示、共用截圖 01、03、英文截圖 01（02 刪掉了）
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

// 實際用本機的 Chrome 擷取一個 WebGL 頁面（PT_TEST_CHROME=1 才跑；PT_TEST_SHOT_DIR 可以保留輸出）
func TestScreenshotChrome(t *testing.T) {
	if os.Getenv("PT_TEST_CHROME") != "1" {
		t.Skip("設定 PT_TEST_CHROME=1 才實際啟動 Chrome")
	}
	if _, err := findChrome(""); err != nil {
		t.Fatal(err)
	}
	// 依視窗大小畫 WebGL 背景與 2D 文字；底部的紅條確認畫面高度完整（沒有被瀏覽器工具列吃掉）
	page := `<!doctype html><html><head><meta name="viewport" content="width=device-width,initial-scale=1"></head>
<body style="margin:0;background:#000;overflow:hidden">
<canvas id="gl" style="position:fixed;inset:0;width:100vw;height:100vh"></canvas>
<canvas id="ui" style="position:fixed;inset:0;width:100vw;height:100vh"></canvas>
<script>
const W = innerWidth, H = innerHeight, d = devicePixelRatio;
for (const c of document.querySelectorAll('canvas')) { c.width = W * d; c.height = H * d; }
const gl = document.getElementById('gl').getContext('webgl');
if (gl) { gl.clearColor(0.2, 0.55, 0.85, 1); gl.clear(gl.COLOR_BUFFER_BIT); }
const x = document.getElementById('ui').getContext('2d');
x.scale(d, d);
x.fillStyle = '#ffd34d'; x.beginPath(); x.arc(W / 2, H / 2, Math.min(W, H) / 4, 0, Math.PI * 2); x.fill();
x.fillStyle = '#d93a3a'; x.fillRect(0, H - 20, W, 20);
x.fillStyle = '#fff'; x.font = '20px sans-serif'; x.fillText(W + '×' + H + ' @' + d + (gl ? ' WebGL' : ' 沒有 WebGL'), 12, 32);
</script></body></html>`
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, page)
	}))
	defer site.Close()
	for _, orientation := range []string{"portrait", "landscape"} {
		dir := shotProject(t, orientation)
		must(t, cmdStoreScreenshot([]string{"-url", site.URL + "/", "-wait", "1s"}))
		f := filepath.Join(dir, "store", "screenshots", "01.png")
		raw, err := os.ReadFile(f)
		must(t, err)
		img, err := png.Decode(bytes.NewReader(raw))
		must(t, err)
		w, h, _, _ := shotSize(orientation)
		if img.Bounds().Dx() != w || img.Bounds().Dy() != h {
			t.Fatalf("%s：%v", orientation, img.Bounds())
		}
		// WebGL 背景（藍）、中間的圓（黃）、最底下一列是紅條（畫面高度完整）
		near := func(x, y int, r, g, b uint32) bool {
			cr, cg, cb, _ := img.At(x, y).RGBA()
			diff := func(a, b uint32) uint32 {
				if a > b {
					return a - b
				}
				return b - a
			}
			return diff(cr>>8, r)+diff(cg>>8, g)+diff(cb>>8, b) < 60
		}
		if !near(w-5, 100, 51, 140, 217) {
			t.Errorf("%s：WebGL 背景不是藍色：%v", orientation, img.At(w-5, 100))
		}
		if !near(w/2, h/2, 255, 211, 77) {
			t.Errorf("%s：中間不是黃色的圓：%v", orientation, img.At(w/2, h/2))
		}
		if !near(w/2, h-2, 217, 58, 58) {
			t.Errorf("%s：最底下不是紅條（畫面高度被截掉？）：%v", orientation, img.At(w/2, h-2))
		}
		if keep := os.Getenv("PT_TEST_SHOT_DIR"); keep != "" {
			must(t, os.MkdirAll(keep, 0o755))
			must(t, os.WriteFile(filepath.Join(keep, "chrome-"+orientation+".png"), raw, 0o644))
		}
	}
}
