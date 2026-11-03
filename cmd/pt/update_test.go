package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewer(t *testing.T) {
	for _, c := range []struct {
		a, b string
		want bool
	}{
		{"v0.6.0", "v0.5.0", true},
		{"v0.10.0", "v0.9.9", true},
		{"v1.0.0", "v0.99.0", true},
		{"v0.5.0", "v0.5.0", false},
		{"v0.4.9", "v0.5.0", false},
		{"v0.5.1", "v0.5.1-0.20261103-abcdef", false},
		{"v0.5.2", "v0.5.1-0.20261103-abcdef", true},
		{"", "v0.5.0", false},
		{"v0.6.0", "dev", false},
		{"v0.6", "v0.5.0", false},
	} {
		if got := newer(c.a, c.b); got != c.want {
			t.Errorf("newer(%q, %q) = %v", c.a, c.b, got)
		}
	}
}

// fakeRelease 模擬 GitHub 的 /releases/latest 轉址，回傳被問了幾次。
func fakeRelease(t *testing.T, location string) *atomic.Int32 {
	t.Helper()
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		if !strings.HasPrefix(r.Header.Get("User-Agent"), "pt/") {
			t.Errorf("User-Agent 要帶 pt 版本：%q", r.Header.Get("User-Agent"))
		}
		w.Header().Set("Location", location)
		w.WriteHeader(http.StatusFound)
	}))
	t.Cleanup(srv.Close)
	old := latestURL
	latestURL = srv.URL + "/releases/latest"
	t.Cleanup(func() { latestURL = old })
	return &hits
}

// withUpdateEnv 設定成「會檢查」的環境：發佈版、終端機、沒有 CI，快取放在暫存資料夾；回傳快取路徑與讀 stderr 的函式。
func withUpdateEnv(t *testing.T, current string) (string, func() string) {
	t.Helper()
	cache := filepath.Join(t.TempDir(), "pt-update.json")
	oldVersion, oldCache, oldTTY := version, updateCache, stderrIsTTY
	version, updateCache, stderrIsTTY = current, func() string { return cache }, func() bool { return true }
	t.Setenv("CI", "")
	t.Setenv("PT_NO_UPDATE_CHECK", "")
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	oldStderr := os.Stderr
	os.Stderr = w
	t.Cleanup(func() { version, updateCache, stderrIsTTY, os.Stderr = oldVersion, oldCache, oldTTY, oldStderr })
	return cache, func() string {
		w.Close()
		os.Stderr = oldStderr
		b, _ := io.ReadAll(r)
		return string(b)
	}
}

func TestUpdateCheckNotifiesAndCaches(t *testing.T) {
	hits := fakeRelease(t, "https://github.com/game-dev-zone/play-together-ctl/releases/tag/v0.6.0")
	cache, stderr := withUpdateEnv(t, "v0.5.0")
	startUpdateCheck()()
	startUpdateCheck()() // 一天內第二次：用快取，不再問
	out := stderr()
	if hits.Load() != 1 {
		t.Fatalf("一天內只能問一次，實際 %d 次", hits.Load())
	}
	if strings.Count(out, "有新版 pt v0.6.0（目前 v0.5.0）") != 2 || !strings.Contains(out, "go install "+installPath+"@latest") || !strings.Contains(out, "pt sdk update") {
		t.Fatalf("提醒內容不對：%s", out)
	}
	if st := readUpdateState(cache); st.Latest != "v0.6.0" {
		t.Fatalf("快取沒有寫入：%+v", st)
	}
}

func TestUpdateCheckQuietWhenCurrent(t *testing.T) {
	fakeRelease(t, "https://github.com/game-dev-zone/play-together-ctl/releases/tag/v0.5.0")
	_, stderr := withUpdateEnv(t, "v0.5.0")
	startUpdateCheck()()
	if out := stderr(); out != "" {
		t.Fatalf("已是最新版不該提醒：%s", out)
	}
}

func TestUpdateCheckStaleCache(t *testing.T) {
	hits := fakeRelease(t, "https://github.com/game-dev-zone/play-together-ctl/releases/tag/v0.7.0")
	cache, stderr := withUpdateEnv(t, "v0.5.0")
	writeUpdateState(cache, updateState{CheckedAt: time.Now().Add(-25 * time.Hour), Latest: "v0.6.0"})
	startUpdateCheck()()
	out := stderr()
	if hits.Load() != 1 || !strings.Contains(out, "v0.7.0") {
		t.Fatalf("快取過期要重新檢查：%d %s", hits.Load(), out)
	}
}

func TestUpdateCheckDisabled(t *testing.T) {
	for name, setup := range map[string]func(){
		"PT_NO_UPDATE_CHECK": func() { t.Setenv("PT_NO_UPDATE_CHECK", "1") },
		"CI":                 func() { t.Setenv("CI", "true") },
		"不是終端機":              func() { stderrIsTTY = func() bool { return false } },
		"自己編譯":               func() { version = "dev" },
	} {
		t.Run(name, func(t *testing.T) {
			hits := fakeRelease(t, "https://github.com/game-dev-zone/play-together-ctl/releases/tag/v9.0.0")
			_, stderr := withUpdateEnv(t, "v0.5.0")
			setup()
			startUpdateCheck()()
			if out := stderr(); out != "" || hits.Load() != 0 {
				t.Fatalf("不該檢查：%d %s", hits.Load(), out)
			}
		})
	}
}

func TestUpdateCheckRejectsBadResponse(t *testing.T) {
	for _, loc := range []string{"", "https://evil.example/tag/latest", "https://github.com/x/releases/tag/v1.0.0;rm", "https://github.com/x/releases/tag/v1.0"} {
		fakeRelease(t, loc)
		cache, stderr := withUpdateEnv(t, "v0.5.0")
		startUpdateCheck()()
		if out := stderr(); out != "" {
			t.Fatalf("轉址 %q 不該產生提醒：%s", loc, out)
		}
		// 失敗後一小時內不再問
		if st := readUpdateState(cache); time.Since(st.CheckedAt) < 22*time.Hour {
			t.Fatalf("失敗後要在一小時後重試：%+v", st)
		}
	}
}

func TestUpdateCheckOffline(t *testing.T) {
	old := latestURL
	latestURL = "http://127.0.0.1:1/releases/latest"
	t.Cleanup(func() { latestURL = old })
	_, stderr := withUpdateEnv(t, "v0.5.0")
	start := time.Now()
	startUpdateCheck()()
	if out := stderr(); out != "" || time.Since(start) > 3*time.Second {
		t.Fatalf("連不上時要安靜且不拖慢指令：%v %s", time.Since(start), out)
	}
}

func TestReadUpdateStateRejectsGarbage(t *testing.T) {
	p := filepath.Join(t.TempDir(), "s.json")
	if err := os.WriteFile(p, []byte(`{"checkedAt":"2026-11-03T00:00:00Z","latest":"v1.0.0\nrm -rf /"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if st := readUpdateState(p); st.Latest != "" {
		t.Fatalf("快取裡不是版本號的內容要丟掉：%q", st.Latest)
	}
}
