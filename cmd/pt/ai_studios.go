package main

// pt new -client godot -ai studios：把 Claude Code Game Studios（MIT，github.com/donchitos/claude-code-game-studios）
// 加進新專案，並疊上玩聚的規則。
//
// 安全：範本裡的 hooks 與 scripts 會在開發者電腦上由 Claude Code 執行，所以
//   - 版本固定在 assets/ai/ccgs.json 的 commit，下載後逐一比對 SHA-256，任何一個不符就整個停止；
//   - 只安裝清單裡的檔案，下載內容裡其他的檔案一律忽略；
//   - 全部下載並驗證完才寫入磁碟。
// 升級版本見 scripts/ccgs-manifest.py。

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
	"time"
)

type ccgsManifest struct {
	Name    string `json:"name"`
	Repo    string `json:"repo"`
	Version string `json:"version"`
	Commit  string `json:"commit"`
	License string `json:"license"`
	// Skills：只有 awesome-gamedev-agent-skills 的清單有（ai_skills.go）
	Skills []skillEntry `json:"skills,omitempty"`
	Files  []struct {
		Path   string `json:"path"`
		SHA256 string `json:"sha256"`
		Size   int64  `json:"size"`
	} `json:"files"`
}

// loadCCGS 讀內嵌的清單（測試時換成假的清單）
var loadCCGS = loadEmbeddedCCGS

func loadEmbeddedCCGS() (ccgsManifest, error) {
	var m ccgsManifest
	raw, err := embedded.ReadFile("assets/ai/ccgs.json")
	if err != nil {
		return m, err
	}
	if err := json.Unmarshal(raw, &m); err != nil {
		return m, err
	}
	if len(m.Commit) != 40 || len(m.Files) == 0 {
		return m, errors.New("內嵌的 Claude Code Game Studios 清單格式錯誤")
	}
	return m, nil
}

// maxCCGSDownload 是下載壓縮檔的上限（目前約 3 MB）
const maxCCGSDownload = 64 << 20

// fetchCCGS 下載固定 commit 的原始碼壓縮檔，回傳通過驗證的檔案（路徑 → 內容）。
// 網址可以用環境變數 PT_CCGS_URL 換成鏡像（內容仍要通過清單的 SHA-256）。
func fetchCCGS(m ccgsManifest) (map[string][]byte, error) { return fetchPinned(m, "PT_CCGS_URL") }

// fetchPinned 下載清單裡固定 commit 的 GitHub 原始碼壓縮檔並逐檔驗證；urlEnv 是可以換成鏡像網址的環境變數。
func fetchPinned(m ccgsManifest, urlEnv string) (map[string][]byte, error) {
	url := os.Getenv(urlEnv)
	if url == "" {
		url = "https://codeload.github.com/" + m.Repo + "/tar.gz/" + m.Commit
	}
	client := &http.Client{Timeout: 3 * time.Minute}
	res, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("下載 %s 失敗：%w", m.Name, err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("下載 %s 失敗：HTTP %d", m.Name, res.StatusCode)
	}
	return verifyCCGSArchive(m, io.LimitReader(res.Body, maxCCGSDownload))
}

// verifyCCGSArchive 讀 GitHub 的 tar.gz（第一層是「repo-commit/」資料夾），只取清單裡的檔案並比對大小與 SHA-256。
func verifyCCGSArchive(m ccgsManifest, r io.Reader) (map[string][]byte, error) {
	want := map[string]int{}
	for i, f := range m.Files {
		want[f.Path] = i
	}
	gz, err := gzip.NewReader(r)
	if err != nil {
		return nil, fmt.Errorf("%s 的壓縮檔格式錯誤：%w", m.Name, err)
	}
	tr := tar.NewReader(gz)
	got := map[string][]byte{}
	for {
		h, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("%s 的壓縮檔格式錯誤：%w", m.Name, err)
		}
		_, rel, ok := strings.Cut(h.Name, "/")
		if !ok {
			continue
		}
		i, listed := want[rel]
		if !listed {
			continue
		}
		f := m.Files[i]
		if h.Typeflag != tar.TypeReg {
			return nil, fmt.Errorf("%s 的 %s 不是一般檔案：停止安裝", m.Name, rel)
		}
		data, err := io.ReadAll(io.LimitReader(tr, f.Size+1))
		if err != nil {
			return nil, err
		}
		sum := sha256.Sum256(data)
		if int64(len(data)) != f.Size || hex.EncodeToString(sum[:]) != f.SHA256 {
			return nil, fmt.Errorf("%s 的 %s 和 %s 版（%s）不符：停止安裝，沒有寫入任何檔案", m.Name, rel, m.Version, m.Commit[:7])
		}
		got[rel] = data
	}
	for _, f := range m.Files {
		if _, ok := got[f.Path]; !ok {
			return nil, fmt.Errorf("%s 的壓縮檔缺少 %s：停止安裝，沒有寫入任何檔案", m.Name, f.Path)
		}
	}
	return got, nil
}

// addStudiosFiles 把驗證過的範本與玩聚的規則加進要寫入的檔案（files：路徑 → 內容）。
func addStudiosFiles(files map[string][]byte, m ccgsManifest, studio map[string][]byte) error {
	for rel, content := range studio {
		switch rel {
		case "CLAUDE.md":
			// 玩聚規則放在最前面，優先於範本的預設
			header, err := embedded.ReadFile("templates/ai-studios/claude-header.md")
			if err != nil {
				return err
			}
			files[rel] = append(append([]byte{}, header...), content...)
		case ".gitignore":
			// 範本會忽略 export_presets.cfg，但玩聚的網頁匯出設定要進版控
			files[rel] = append(append(append([]byte{}, files[rel]...), "\n# ---- Claude Code Game Studios ----\n"...), content...)
			files[rel] = append(files[rel], "\n# 玩聚：網頁匯出設定要進版控（pt build 需要）\n!export_presets.cfg\n"...)
		case "LICENSE":
			files["THIRD_PARTY/claude-code-game-studios/LICENSE"] = content
		default:
			if _, exists := files[rel]; exists {
				return fmt.Errorf("%s 和玩聚專案的 %s 衝突", m.Name, rel)
			}
			files[rel] = content
		}
	}
	files["THIRD_PARTY/claude-code-game-studios/SOURCE.md"] = []byte(fmt.Sprintf(
		"# %s\n\n- 來源：https://github.com/%s\n- 版本：%s（commit %s）\n- 授權：%s（見同資料夾的 LICENSE）\n\n由 `pt new -ai studios` 加入：每個檔案都比對過 SHA-256。`.claude/rules/playtogether.md` 與 `CLAUDE.md` 開頭的玩聚規則是 pt 加上的。\n",
		m.Name, m.Repo, m.Version, m.Commit, m.License))
	if err := fs2Walk("templates/ai-studios/claude", func(rel string, content []byte) error {
		files[path.Join(".claude", rel)] = content
		return nil
	}); err != nil {
		return err
	}
	// Godot 專案在根目錄：讓 Godot 不要匯入（與匯出）這些資料夾
	for _, dir := range []string{"server", "keys", "design", "docs", "production", "THIRD_PARTY"} {
		files[path.Join(dir, ".gdignore")] = nil
	}
	return nil
}
