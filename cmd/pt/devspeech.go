package main

// pt dev 的本機語音辨識：在「用平台外殼開啟」的預覽頁用麥克風說話，用和玩聚平台相同的辨識程式與詞彙表比對，
// 辨識出的文字走控制面板「輸入一句話」同一條路交給遊戲。
//
// 辨識引擎與模型（語音指令約 18 MB、語音轉文字約 26 MB，只下載用到的）不放進 pt：第一次用時從玩聚網站（playtogether.json 的 platform，或 PT_SPEECH_FROM、PT_API_URL）
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
	"sync/atomic"
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

// speechDownloads：pt 正在從玩聚網站下載的檔案（路徑 → 已下載的位元組），預覽頁用來顯示 pt 這一段的下載進度
// （pt 下載完、核對雜湊後才交給瀏覽器，瀏覽器自己的進度在那之前不會動）。
var speechDownloads sync.Map

type speechProgress struct{ n atomic.Int64 }

func (p *speechProgress) Write(b []byte) (int, error) {
	p.n.Add(int64(len(b)))
	return len(b), nil
}

// speechCacheOK：快取裡的檔案是不是完整而且雜湊正確。雜湊只在檔案變動（大小或修改時間）時重算一次。
var speechHashMemo sync.Map // 本機路徑 → speechHashSeen

type speechHashSeen struct {
	size  int64
	mtime time.Time
	ok    bool
}

func speechCacheOK(local string, f speechFile) bool {
	st, err := os.Stat(local)
	if err != nil || st.Size() != f.Size {
		return false
	}
	if v, ok := speechHashMemo.Load(local); ok {
		if s := v.(speechHashSeen); s.size == st.Size() && s.mtime.Equal(st.ModTime()) {
			return s.ok
		}
	}
	sum, err := fileSHA256(local)
	ok := err == nil && sum == f.SHA256
	speechHashMemo.Store(local, speechHashSeen{size: st.Size(), mtime: st.ModTime(), ok: ok})
	return ok
}

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
	prog := &speechProgress{}
	speechDownloads.Store(f.Path, prog)
	defer speechDownloads.Delete(f.Path)
	n, err := io.Copy(io.MultiWriter(tmp, h, prog), io.LimitReader(res.Body, f.Size+1))
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
	// 兩種引擎分開算有沒有下載過（引擎實際會載入的檔案，大小與雜湊都對）：語音指令（約 18 MB）、語音轉文字（約 26 MB 的模型＋共用的 WASM）。
	// 授權與說明文件（LICENSE、NOTICE、SOURCE）引擎不會載入，不算
	m := loadSpeechManifest()
	kws, asr := m.Engine != "", m.Engine != ""
	if dir, err := speechCacheDir(m.Engine); err == nil {
		for _, sf := range m.Files {
			if isSpeechDocFile(sf.Path) {
				continue
			}
			ok := speechCacheOK(filepath.Join(dir, filepath.FromSlash(sf.Path)), sf)
			switch {
			case isASRFile(sf.Path):
				asr = asr && ok
			case isKWSOnlyFile(sf.Path):
				kws = kws && ok
			default: // 共用的檔案（WASM、收音）
				kws, asr = kws && ok, asr && ok
			}
		}
	} else {
		kws, asr = false, false
	}
	out["engineCached"], out["asrCached"] = kws, asr
	// pt 正在從玩聚網站下載的部分（位元組）
	var loaded, total int64
	speechDownloads.Range(func(k, v any) bool {
		if f, ok := speechFileFor("/" + k.(string)); ok {
			loaded += v.(*speechProgress).n.Load()
			total += f.Size
		}
		return true
	})
	if total > 0 {
		out["downloading"] = map[string]int64{"loaded": loaded, "total": total}
	}
	w.Header().Set("Cache-Control", "no-store")
	f.write(w, 200, out)
}

// isASRFile：只有語音轉文字用的檔案（Worker、模型、簡轉繁）。
func isASRFile(p string) bool {
	base := path.Base(p)
	return strings.HasPrefix(base, "asr-") || base == "sherpa-onnx-asr.js" || base == "zh-convert.js"
}

// isKWSOnlyFile：只有語音指令（關鍵詞偵測）用的檔案。
func isKWSOnlyFile(p string) bool {
	base := path.Base(p)
	return strings.HasPrefix(base, "kws-") || base == "sherpa-onnx-kws.js" || base == "speech-worker.js" || base == "pinyin-zh.json"
}

// isSpeechDocFile：授權與說明文件（引擎不會載入）。
func isSpeechDocFile(p string) bool {
	base := path.Base(p)
	return strings.HasPrefix(base, "LICENSE") || base == "NOTICE.txt" || base == "SOURCE.md"
}
