package main

// 新版檢查：最多一天問一次 GitHub 最新的 pt 版本，有新版時在指令結束後提醒一行。
// 只讀 GitHub 的「最新版本」轉址（不用 API、不送任何專案資料），不會自動下載或替換執行檔。
// 不檢查的情況：PT_NO_UPDATE_CHECK=1、CI、輸出不是終端機、自己編譯的 dev 版本。

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	repoURL        = "https://github.com/game-dev-zone/play-together-ctl"
	installPath    = "github.com/game-dev-zone/play-together-ctl/cmd/pt"
	updateInterval = 24 * time.Hour
)

var (
	// 測試時換掉
	latestURL   = repoURL + "/releases/latest"
	updateCache = func() string {
		d, err := os.UserCacheDir()
		if err != nil {
			return ""
		}
		return filepath.Join(d, "playtogether", "pt-update.json")
	}
	stderrIsTTY = func() bool {
		st, err := os.Stderr.Stat()
		return err == nil && st.Mode()&os.ModeCharDevice != 0 && !sameAsDevNull(st)
	}
	now           = time.Now
	tagPattern    = regexp.MustCompile(`^v\d+\.\d+\.\d+$`)
	updateTimeout = 2 * time.Second
)

func sameAsDevNull(st os.FileInfo) bool {
	null, err := os.Stat(os.DevNull)
	return err == nil && os.SameFile(st, null)
}

type updateState struct {
	CheckedAt time.Time `json:"checkedAt"`
	Latest    string    `json:"latest"`
}

// fetchLatest 讀 /releases/latest 轉到的 /releases/tag/vX.Y.Z，取出版本號。
func fetchLatest() (string, error) {
	client := &http.Client{Timeout: updateTimeout, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	req, err := http.NewRequest(http.MethodHead, latestURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "pt/"+version)
	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	res.Body.Close()
	loc := res.Header.Get("Location")
	tag := loc[strings.LastIndex(loc, "/")+1:]
	if res.StatusCode/100 != 3 || !tagPattern.MatchString(tag) {
		return "", fmt.Errorf("無法取得最新版本（%d）", res.StatusCode)
	}
	return tag, nil
}

// newer 回傳 a 是否比 b 新（都是 vX.Y.Z；格式不對一律當作不新）。
func newer(a, b string) bool {
	pa, okA := parseVersion(a)
	pb, okB := parseVersion(b)
	if !okA || !okB {
		return false
	}
	for i := range pa {
		if pa[i] != pb[i] {
			return pa[i] > pb[i]
		}
	}
	return false
}

func parseVersion(v string) ([3]int, bool) {
	var out [3]int
	v = strings.TrimPrefix(v, "v")
	if i := strings.IndexAny(v, "-+"); i >= 0 { // v0.5.0-rc1、go install 的偽版本
		v = v[:i]
	}
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return out, false
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return out, false
		}
		out[i] = n
	}
	return out, true
}

func readUpdateState(path string) updateState {
	var st updateState
	if raw, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(raw, &st)
	}
	if !tagPattern.MatchString(st.Latest) {
		st.Latest = ""
	}
	return st
}

func writeUpdateState(path string, st updateState) {
	if os.MkdirAll(filepath.Dir(path), 0o755) != nil {
		return
	}
	raw, _ := json.Marshal(st)
	_ = os.WriteFile(path, raw, 0o644)
}

func updateCheckDisabled() bool {
	if v := os.Getenv("PT_NO_UPDATE_CHECK"); v != "" && v != "0" {
		return true
	}
	if os.Getenv("CI") != "" {
		return true
	}
	return !strings.HasPrefix(version, "v") || !stderrIsTTY()
}

// startUpdateCheck 在指令開始時呼叫：快取過期就在背景問一次。回傳的函式在指令結束時呼叫，
// 最多再等 updateTimeout，有新版就印出提醒。
func startUpdateCheck() func() {
	if updateCheckDisabled() {
		return func() {}
	}
	path := updateCache()
	if path == "" {
		return func() {}
	}
	st := readUpdateState(path)
	var done chan struct{}
	if now().Sub(st.CheckedAt) >= updateInterval || now().Before(st.CheckedAt) {
		done = make(chan struct{})
		go func() {
			defer close(done)
			latest, err := fetchLatest()
			if err != nil {
				// 連不上：一小時後再試，不要每次指令都等
				writeUpdateState(path, updateState{CheckedAt: now().Add(-updateInterval + time.Hour), Latest: st.Latest})
				return
			}
			st = updateState{CheckedAt: now(), Latest: latest}
			writeUpdateState(path, st)
		}()
	}
	return func() {
		if done != nil {
			select {
			case <-done:
			case <-time.After(updateTimeout):
				return
			}
		}
		if newer(st.Latest, version) {
			fmt.Fprint(os.Stderr, updateNotice(version, st.Latest))
		}
	}
}

func updateNotice(current, latest string) string {
	return fmt.Sprintf(`
有新版 pt %s（目前 %s）。更新方式：
  go install %s@latest
  或下載：%s/releases/latest
更新後在專案裡執行 pt sdk update，把專案裡的 SDK 換成新版。（關閉這個提醒：PT_NO_UPDATE_CHECK=1）
`, latest, current, installPath, repoURL)
}

// versionInfo 給 pt version：立即檢查一次（不看快取），回報是否有新版。
func versionInfo() string {
	out := fmt.Sprintf("pt %s（玩聚開發者工具），內嵌 SDK %s\n", version, sdkVersion())
	if !strings.HasPrefix(version, "v") {
		return out + "這是自己編譯的版本，不檢查新版。\n"
	}
	if v := os.Getenv("PT_NO_UPDATE_CHECK"); v != "" && v != "0" {
		return out
	}
	latest, err := fetchLatest()
	if err != nil {
		return out + "無法檢查新版：" + err.Error() + "\n"
	}
	if path := updateCache(); path != "" {
		writeUpdateState(path, updateState{CheckedAt: now(), Latest: latest})
	}
	if newer(latest, version) {
		return out + strings.TrimPrefix(updateNotice(version, latest), "\n")
	}
	return out + "已經是最新版。\n"
}
