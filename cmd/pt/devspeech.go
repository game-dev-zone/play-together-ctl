package main

// pt dev 的本機語音辨識：在「用平台外殼開啟」的預覽頁用麥克風說話，用和玩聚平台相同的辨識程式與詞彙表比對，
// 辨識出的文字走控制面板「輸入一句話」同一條路交給遊戲。
//
// 辨識引擎與模型（約 18 MB）不放進 pt：第一次用時從玩聚網站（playtogether.json 的 platform，或 PT_SPEECH_FROM、PT_API_URL）
// 下載到本機快取，逐一核對 speech-manifest.json 的 SHA-256，之後離線也能用。只提供清單裡的檔案。

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type speechFile struct {
	Path   string `json:"path"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}

type speechManifest struct {
	Engine string       `json:"engine"`
	Files  []speechFile `json:"files"`
}

var (
	speechManifestOnce sync.Once
	speechManifestData speechManifest
	speechFetchLocks   sync.Map // 檔案路徑 → *sync.Mutex（同一個檔案同時只下載一次）
)

func loadSpeechManifest() speechManifest {
	speechManifestOnce.Do(func() {
		raw, err := embedded.ReadFile("assets/dev/speech-manifest.json")
		if err == nil {
			_ = json.Unmarshal(raw, &speechManifestData)
		}
	})
	return speechManifestData
}

// speechSource 是下載辨識引擎的玩聚網站：PT_SPEECH_FROM、PT_API_URL、playtogether.json 的 platform，依序取第一個有效的。
func speechSource(p *project) string {
	for _, s := range []string{os.Getenv("PT_SPEECH_FROM"), os.Getenv("PT_API_URL"), func() string {
		if p == nil {
			return ""
		}
		return p.Platform
	}()} {
		s = strings.TrimRight(strings.TrimSpace(s), "/")
		if s != "" && validPlatform(s) == nil {
			return s
		}
	}
	return ""
}

func speechCacheDir(engine string) (string, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "playtogether", "speech", engine), nil
}

// speechFileFor：清單裡的檔案（網址路徑，例如 /speech/kws-zh-tokens.txt）。
func speechFileFor(urlPath string) (speechFile, bool) {
	rel := strings.TrimPrefix(urlPath, "/")
	for _, f := range loadSpeechManifest().Files {
		if f.Path == rel {
			return f, true
		}
	}
	return speechFile{}, false
}

func fileSHA256(p string) (string, error) {
	fh, err := os.Open(p)
	if err != nil {
		return "", err
	}
	defer fh.Close()
	h := sha256.New()
	if _, err := io.Copy(h, fh); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

var errSpeechNoSource = errors.New("no_source")

// ensureSpeechFile 確保快取裡有這個檔案而且雜湊正確；沒有就從玩聚網站下載。回傳本機路徑。
func ensureSpeechFile(ctx context.Context, source string, f speechFile) (string, error) {
	m := loadSpeechManifest()
	dir, err := speechCacheDir(m.Engine)
	if err != nil {
		return "", err
	}
	local := filepath.Join(dir, filepath.FromSlash(f.Path))
	lock, _ := speechFetchLocks.LoadOrStore(f.Path, &sync.Mutex{})
	lock.(*sync.Mutex).Lock()
	defer lock.(*sync.Mutex).Unlock()
	if sum, err := fileSHA256(local); err == nil && sum == f.SHA256 {
		return local, nil
	}
	if source == "" {
		return "", errSpeechNoSource
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, source+"/"+f.Path, nil)
	if err != nil {
		return "", err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("下載 %s 失敗：%w", f.Path, err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("下載 %s 失敗：HTTP %d", f.Path, res.StatusCode)
	}
	if err := os.MkdirAll(filepath.Dir(local), 0o755); err != nil {
		return "", err
	}
	tmp, err := os.CreateTemp(filepath.Dir(local), ".download-*")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmp.Name())
	h := sha256.New()
	n, err := io.Copy(io.MultiWriter(tmp, h), io.LimitReader(res.Body, f.Size+1))
	tmp.Close()
	if err != nil {
		return "", fmt.Errorf("下載 %s 失敗：%w", f.Path, err)
	}
	if n != f.Size || hex.EncodeToString(h.Sum(nil)) != f.SHA256 {
		return "", fmt.Errorf("%s 和這個 pt 版本的辨識引擎不同（玩聚網站的版本比較新或比較舊）：請更新 pt（pt upgrade），或改用 PT_SPEECH_FROM 指定版本相同的玩聚網站", f.Path)
	}
	if err := os.Rename(tmp.Name(), local); err != nil {
		return "", err
	}
	return local, nil
}

// serveSpeechAsset：GET /speech/*、/speech-worklet.js（只有清單裡的檔案）。
func (d *devPanel) serveSpeechAsset(w http.ResponseWriter, r *http.Request) bool {
	if r.Method != http.MethodGet || !(strings.HasPrefix(r.URL.Path, "/speech/") || r.URL.Path == "/speech-worklet.js") {
		return false
	}
	f, ok := speechFileFor(r.URL.Path)
	if !ok {
		http.NotFound(w, r)
		return true
	}
	local, err := ensureSpeechFile(r.Context(), d.fp.speechFrom, f)
	if err != nil {
		code := http.StatusBadGateway
		msg := err.Error()
		if errors.Is(err, errSpeechNoSource) {
			code, msg = http.StatusServiceUnavailable, "沒有玩聚網站可以下載辨識引擎：請在 playtogether.json 填 platform，或設定 PT_SPEECH_FROM"
		}
		http.Error(w, msg, code)
		return true
	}
	ct := mime.TypeByExtension(path.Ext(f.Path))
	switch path.Ext(f.Path) {
	case ".wasm":
		ct = "application/wasm"
	case ".js":
		ct = "text/javascript; charset=utf-8"
	case ".onnx":
		ct = "application/octet-stream"
	case ".txt", ".md":
		ct = "text/plain; charset=utf-8"
	}
	if ct != "" {
		w.Header().Set("Content-Type", ct)
	}
	w.Header().Set("Cache-Control", "no-cache")
	http.ServeFile(w, r, local)
	return true
}

// speechInfo：GET /dev/speech-info?player=，預覽頁用：這位玩家有沒有授權、畫面是不是正在聽、詞彙表、引擎下載狀態。
func (d *devPanel) speechInfo(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("player")
	if !fakePlayerCode.MatchString(code) {
		d.fp.write(w, 400, map[string]string{"error": "bad_player"})
		return
	}
	f := d.fp
	f.mu.Lock()
	out := map[string]any{"allowed": f.speechAllowedLocked(code), "listening": f.speechListeningLocked(code), "source": f.speechFrom}
	if f.room != nil && f.room.Speech != nil {
		out["sequence"], out["language"], out["vocabulary"] = f.room.Speech.Sequence, f.room.Speech.Language, f.room.Speech.Vocabulary
	}
	f.mu.Unlock()
	m := loadSpeechManifest()
	cached := m.Engine != ""
	if dir, err := speechCacheDir(m.Engine); err == nil {
		for _, sf := range m.Files {
			if st, err := os.Stat(filepath.Join(dir, filepath.FromSlash(sf.Path))); err != nil || st.Size() != sf.Size {
				cached = false
				break
			}
		}
	}
	out["engineCached"] = cached
	w.Header().Set("Cache-Control", "no-store")
	f.write(w, 200, out)
}
