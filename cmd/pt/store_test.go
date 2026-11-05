package main

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func pngOf(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	img.Set(0, 0, color.RGBA{1, 2, 3, 255})
	var b bytes.Buffer
	must(t, png.Encode(&b, img))
	return b.Bytes()
}

func fillStore(t *testing.T, root string) {
	t.Helper()
	l := map[string]any{
		"name":               map[string]string{"zh-TW": "大風吹", "en": "Musical Chairs"},
		"summary":            map[string]string{"zh-TW": "聽口令搶位子", "en": "Grab a seat"},
		"description":        map[string]string{"zh-TW": "每一輪主持人喊出口令。\n\n最後坐著的人獲勝。", "en": "Each round the host calls out."},
		"screenshotCaptions": map[string][]string{"zh-TW": {"搶位子"}, "en": {"Grab a seat"}},
		"tags":               []string{"party"}, "players": map[string]int{"min": 2, "max": 8}, "sessionMinutes": 5, "reviewNotes": "兩人以上測試",
	}
	raw, _ := json.MarshalIndent(l, "", "  ")
	must(t, os.WriteFile(filepath.Join(root, "store/listing.json"), raw, 0o644))
	must(t, os.WriteFile(filepath.Join(root, "store/icon.png"), pngOf(t, 600, 600), 0o644))
	must(t, os.WriteFile(filepath.Join(root, "store/screenshots/01.png"), pngOf(t, 720, 1280), 0o644))
	must(t, os.WriteFile(filepath.Join(root, "store/screenshots/02.png"), pngOf(t, 720, 1280), 0o644))
}

type storeFindings []finding

func (fs storeFindings) has(level, sub string) bool {
	for _, f := range fs {
		if f.level == level && strings.Contains(f.msg, sub) {
			return true
		}
	}
	return false
}

func runStore(t *testing.T, root string) (storeFindings, []storeImage, bool) {
	t.Helper()
	p := readProject(t, root)
	p.dir = root
	var fs storeFindings
	_, imgs, _, ready := readStore(&p, func(level, format string, a ...any) {
		fs = append(fs, finding{level, strings.TrimSpace(sprintf(format, a...))})
	})
	return fs, imgs, ready
}

func sprintf(format string, a ...any) string {
	return strings.TrimSpace(strings.ReplaceAll(fmtSprintf(format, a...), "\n", " "))
}

// 新專案的 store/ 範本：還沒準備好只是說明，不是錯誤；填好之後沒有任何問題。
func TestStoreTemplateAndReady(t *testing.T) {
	requireEmbeddedSDK(t)
	dir := t.TempDir()
	t.Chdir(dir)
	must(t, cmdNew([]string{"store-game", "-y"}))
	root := filepath.Join(dir, "store-game")
	for _, f := range []string{"store/listing.json", "store/README.md", "store/screenshots/README.md"} {
		readFile(t, dir, "store-game", f)
	}
	fs, _, ready := runStore(t, root)
	if ready || fs.has(levelSchema, "") || fs.has(levelHard, "") || !fs.has(levelInfo, "還沒準備好") {
		t.Fatalf("範本還沒準備好只說明：%v %v", ready, fs)
	}
	fillStore(t, root)
	fs, imgs, ready := runStore(t, root)
	if !ready || len(fs) != 0 || len(imgs) != 3 {
		t.Fatalf("填好之後要沒有問題：%v %v %d", ready, fs, len(imgs))
	}
	wf := readFile(t, dir, "store-game", "docs/ai/AI_WORKFLOW.md")
	if !strings.Contains(wf, "截圖必須來自實際遊戲畫面") || !strings.Contains(wf, "繁中與英文都要寫") {
		t.Error("AI_WORKFLOW.md 要有上架資料的規則")
	}
}

// 格式錯誤是宣告錯誤（-skip-check 不能略過）；禁止的宣稱要平台審核。
func TestStoreValidation(t *testing.T) {
	requireEmbeddedSDK(t)
	dir := t.TempDir()
	t.Chdir(dir)
	must(t, cmdNew([]string{"store-bad", "-y"}))
	root := filepath.Join(dir, "store-bad")
	edit := func(f func(m map[string]any)) {
		fillStore(t, root)
		raw, _ := os.ReadFile(filepath.Join(root, "store/listing.json"))
		m := map[string]any{}
		must(t, json.Unmarshal(raw, &m))
		f(m)
		raw, _ = json.Marshal(m)
		must(t, os.WriteFile(filepath.Join(root, "store/listing.json"), raw, 0o644))
	}
	cases := []struct {
		name  string
		setup func()
		level string
		want  string
	}{
		{"缺英文名稱", func() { edit(func(m map[string]any) { m["name"] = map[string]string{"zh-TW": "大風吹", "en": ""} }) }, levelSchema, "name.en"},
		{"名稱太長", func() {
			edit(func(m map[string]any) { m["name"] = map[string]string{"zh-TW": strings.Repeat("字", 31), "en": "x"} })
		}, levelSchema, "太長"},
		{"不認得的標籤", func() { edit(func(m map[string]any) { m["tags"] = []string{"casino"} }) }, levelSchema, "casino"},
		{"拼錯的鍵", func() { edit(func(m map[string]any) { m["sesionMinutes"] = 5 }) }, levelSchema, "未知的欄位"},
		{"人數", func() { edit(func(m map[string]any) { m["players"] = map[string]int{"min": 9, "max": 2} }) }, levelSchema, "players"},
		{"禁止的宣稱", func() {
			edit(func(m map[string]any) { m["summary"] = map[string]string{"zh-TW": "穩賺不賠", "en": "x"} })
		}, levelReview, "穩賺"},
		{"圖示太小", func() {
			fillStore(t, root)
			must(t, os.WriteFile(filepath.Join(root, "store/icon.png"), pngOf(t, 300, 300), 0o644))
		}, levelSchema, "太小"},
		{"橫式截圖用在直式遊戲", func() {
			fillStore(t, root)
			must(t, os.WriteFile(filepath.Join(root, "store/screenshots/02.png"), pngOf(t, 1280, 720), 0o644))
		}, levelSchema, "不能自動裁切"},
		{"不是圖片", func() {
			fillStore(t, root)
			must(t, os.WriteFile(filepath.Join(root, "store/screenshots/02.png"), []byte("<svg/>"), 0o644))
		}, levelSchema, "不是 PNG 或 JPEG"},
		{"兩個圖示", func() {
			fillStore(t, root)
			must(t, os.WriteFile(filepath.Join(root, "store/icon.jpg"), pngOf(t, 600, 600), 0o644))
		}, levelSchema, "多個 icon"},
	}
	for _, c := range cases {
		c.setup()
		if fs, _, _ := runStore(t, root); !fs.has(c.level, c.want) {
			t.Errorf("%s：要有 %s「%s」：%v", c.name, c.level, c.want, fs)
		}
		os.Remove(filepath.Join(root, "store/icon.jpg"))
	}
}

// pt upload：送出宣告快照與上架資料（平台沿用或建立修訂、上傳每張圖片）；store/ 還沒準備好時只上傳版本；-listing-only 只上傳上架資料。
func TestUploadSendsDeclarationAndListing(t *testing.T) {
	requireEmbeddedSDK(t)
	var mu sync.Mutex
	var build, listing map[string]any
	puts := 0
	platform := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		body, _ := io.ReadAll(r.Body)
		switch {
		case r.Method == "POST" && r.URL.Path == "/v1/dev/builds":
			_ = json.Unmarshal(body, &build)
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"buildId":"b1","uploadPath":"/v1/dev/uploads/tok"}`))
		case r.Method == "PUT" && r.URL.Path == "/v1/dev/uploads/tok":
			_, _ = w.Write([]byte(`{"artifactStatus":"processing"}`))
		case r.Method == "POST" && r.URL.Path == "/v1/dev/listing-revisions":
			_ = json.Unmarshal(body, &listing)
			ups := []map[string]any{}
			for i, im := range listing["images"].([]any) {
				m := im.(map[string]any)
				ups = append(ups, map[string]any{"kind": m["kind"], "locale": m["locale"], "position": m["position"], "uploadPath": "/v1/dev/listing-uploads/t" + string(rune('a'+i))})
			}
			raw, _ := json.Marshal(map[string]any{"revisionId": "rev-1", "reused": false, "uploads": ups})
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write(raw)
		case r.Method == "PUT" && strings.HasPrefix(r.URL.Path, "/v1/dev/listing-uploads/"):
			puts++
			_, _ = w.Write([]byte(`{"status":"ready"}`))
		case r.Method == "GET" && r.URL.Path == "/v1/dev/status":
			_, _ = w.Write([]byte(`{"game":{"name":"大風吹","status":"draft","rating":"general","contentLabels":[],"warning":"","liveVersion":""},
				"builds":[{"version":"1.0.0","status":"rejected","artifactStatus":"ready","codes":["2.2"],"reason":"手機按鈕太小","policyFindings":{"findings":[{"level":"review","reason":"虛擬下注玩法"}]}}],
				"listingRevisions":[{"id":"rev-1-aaaaaaaa","status":"in_review","minor":false,"codes":[],"reason":"","pendingImages":0,"failedImages":0}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(platform.Close)
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	keyFile := filepath.Join(t.TempDir(), "upload.key")
	must(t, os.WriteFile(keyFile, []byte(base64.StdEncoding.EncodeToString(priv.Seed())+"\n"), 0o600))
	dir := t.TempDir()
	t.Chdir(dir)
	must(t, cmdNew([]string{"up-game", "-y", "-platform", platform.URL}))
	root := filepath.Join(dir, "up-game")
	t.Chdir(root)
	args := []string{"-version", "1.0.0", "-api", platform.URL, "-key-id", "k1", "-key-file", keyFile, "-skip-check", "-wait", "0"}

	// store/ 還沒準備好：只上傳版本（宣告照樣送）
	must(t, upload(args))
	if build["declaration"] == nil || build["policyVersion"] != loadPolicyRules().Version || listing != nil {
		t.Fatalf("要送宣告、不送上架資料：%v %v", build["declaration"], listing)
	}
	decl := build["declaration"].(map[string]any)
	for _, k := range []string{"platformAudience", "contentProfile", "sensitiveContent", "gameplay", "economy"} {
		if decl[k] == nil {
			t.Errorf("宣告少了 %s", k)
		}
	}
	// 準備好之後：一起上傳上架資料與每張圖片
	fillStore(t, root)
	must(t, upload(append([]string{"-version", "1.0.1"}, args[2:]...)))
	if listing == nil || listing["orientation"] != "portrait" || len(listing["images"].([]any)) != 3 || puts != 3 ||
		listing["name"].(map[string]any)["en"] != "Musical Chairs" {
		t.Fatalf("上架資料：%v，上傳 %d 張", listing, puts)
	}
	// -no-listing：不上傳上架資料
	listing, puts = nil, 0
	must(t, upload(append([]string{"-version", "1.0.2", "-no-listing"}, args[2:]...)))
	if listing != nil {
		t.Fatal("-no-listing 不上傳上架資料")
	}
	// -listing-only：只上傳上架資料
	build = nil
	must(t, upload([]string{"-listing-only", "-api", platform.URL, "-key-id", "k1", "-key-file", keyFile}))
	if build != nil || listing == nil || puts != 3 {
		t.Fatalf("-listing-only：%v %v %d", build, listing, puts)
	}
	// 上架資料有格式錯誤：-listing-only 擋下
	must(t, os.WriteFile(filepath.Join(root, "store/icon.png"), pngOf(t, 100, 100), 0o644))
	if err := upload([]string{"-listing-only", "-api", platform.URL, "-key-id", "k1", "-key-file", keyFile}); err == nil {
		t.Fatal("上架資料有錯誤要擋下")
	}
	// 上架資料的格式錯誤是宣告錯誤：-skip-check 也擋下一般上傳
	if err := upload(append([]string{"-version", "1.0.3"}, args[2:]...)); err == nil {
		t.Fatal("上架資料格式錯誤時 -skip-check 也不能上傳")
	}
	// pt status
	p := readProject(t, root)
	p.dir = root
	p.Platform, p.UploadKeyID = platform.URL, "k1"
	must(t, os.MkdirAll(filepath.Join(root, "keys"), 0o700))
	raw, _ := os.ReadFile(keyFile)
	must(t, os.WriteFile(filepath.Join(root, "keys/upload.key"), raw, 0o600))
	must(t, p.save())
	must(t, cmdStatus(nil))
}

var fmtSprintf = fmt.Sprintf
