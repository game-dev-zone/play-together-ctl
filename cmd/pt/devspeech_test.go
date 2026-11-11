package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
)

// 本機語音辨識的引擎下載：只給清單裡的檔案、雜湊不符不收、下載一次後用快取、沒有來源時說明。
func TestDevSpeechAssets(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CACHE_HOME", home+"/cache")
	t.Setenv("LocalAppData", home+"/cache")
	body := []byte("// worklet\n")
	sum := sha256.Sum256(body)
	speechManifestOnce.Do(func() {})
	speechManifestData = speechManifest{Engine: "test-engine", Files: []speechFile{
		{Path: "speech-worklet.js", Size: int64(len(body)), SHA256: hex.EncodeToString(sum[:])},
		{Path: "speech/tokens.txt", Size: 3, SHA256: strings.Repeat("0", 64)},
	}}
	var hits atomic.Int32
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		switch r.URL.Path {
		case "/speech-worklet.js":
			_, _ = w.Write(body)
		case "/speech/tokens.txt":
			_, _ = w.Write([]byte("abc")) // 雜湊和清單不同（網站的版本不一樣）
		default:
			http.NotFound(w, r)
		}
	}))
	defer site.Close()

	fp := newFakePlatform(nil)
	fp.speechFrom = site.URL
	d := &devPanel{fp: fp, site: "http://127.0.0.1:1/", self: "http://127.0.0.1:2"}
	get := func(p string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		d.ServeHTTP(w, httptest.NewRequest(http.MethodGet, p, nil))
		return w
	}

	if w := get("/speech-worklet.js"); w.Code != 200 || w.Body.String() != string(body) || !strings.HasPrefix(w.Header().Get("Content-Type"), "text/javascript") {
		t.Fatalf("清單裡的檔案要下載後提供：%d %q %s", w.Code, w.Body.String(), w.Header().Get("Content-Type"))
	}
	if w := get("/speech-worklet.js"); w.Code != 200 || hits.Load() != 1 {
		t.Fatalf("第二次要用快取（下載了 %d 次）", hits.Load())
	}
	if w := get("/speech/tokens.txt"); w.Code != http.StatusBadGateway || !strings.Contains(w.Body.String(), "pt upgrade") {
		t.Fatalf("雜湊不符要拒絕並說明：%d %s", w.Code, w.Body.String())
	}
	if w := get("/speech/../../etc/passwd"); w.Code != http.StatusNotFound {
		t.Fatalf("清單以外的路徑要 404：%d", w.Code)
	}
	if w := get("/speech/evil.js"); w.Code != http.StatusNotFound {
		t.Fatalf("清單以外的檔案要 404：%d", w.Code)
	}

	// 沒有來源：說明要填 platform 或 PT_SPEECH_FROM
	fp.speechFrom = ""
	_ = os.RemoveAll(home + "/cache")
	_ = os.RemoveAll(home + "/Library")
	if _, err := ensureSpeechFile(context.Background(), "", speechManifestData.Files[1]); err != errSpeechNoSource {
		t.Fatalf("沒有來源應回 errSpeechNoSource：%v", err)
	}

	// 狀態：格式錯誤的玩家代號
	if w := get("/dev/speech-info?player=../x"); w.Code != 400 {
		t.Fatalf("玩家代號格式錯誤應回 400：%d", w.Code)
	}
	if w := get("/dev/speech-info?player=p1"); w.Code != 200 || !strings.Contains(w.Body.String(), `"allowed":false`) {
		t.Fatalf("狀態：%d %s", w.Code, w.Body.String())
	}
}

func TestSpeechSource(t *testing.T) {
	t.Setenv("PT_SPEECH_FROM", "")
	t.Setenv("PT_API_URL", "")
	if s := speechSource(&project{Platform: "https://pt.example"}); s != "https://pt.example" {
		t.Fatalf("要用 playtogether.json 的 platform：%q", s)
	}
	if s := speechSource(&project{Platform: templatePlatform}); s != "" {
		t.Fatalf("範本預設值不能當來源：%q", s)
	}
	t.Setenv("PT_SPEECH_FROM", "https://dev.pt.example/")
	if s := speechSource(&project{Platform: "https://pt.example"}); s != "https://dev.pt.example" {
		t.Fatalf("PT_SPEECH_FROM 優先：%q", s)
	}
}
