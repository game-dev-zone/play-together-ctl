package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// pt login（裝置授權）保存送審授權；pt upload 在檢查通過後用它送審版本＋上架資料；-no-submit、-listing-only、pt logout。
func TestLoginAndAutoSubmit(t *testing.T) {
	requireEmbeddedSDK(t)
	creds := filepath.Join(t.TempDir(), "credentials.json")
	t.Setenv("PT_CREDENTIALS_FILE", creds)
	var mu sync.Mutex
	polls := 0
	var submissions []map[string]any
	var auth string
	platform := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		body, _ := io.ReadAll(r.Body)
		switch {
		case r.URL.Path == "/v1/dev/device-codes":
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"deviceCode":"dev-secret","userCode":"ABCD-EFGH","verificationUrl":"http://web/dev/login?code=ABCDEFGH","expiresIn":60,"interval":0.001}`))
		case r.URL.Path == "/v1/dev/device-codes/token":
			polls++
			if polls < 3 {
				w.WriteHeader(http.StatusPreconditionRequired)
				_, _ = w.Write([]byte(`{"error":"authorization_pending"}`))
				return
			}
			_, _ = w.Write([]byte(`{"token":"ptd_secret","expiresAt":"2099-01-01T00:00:00Z","teamId":"t1","teamName":"測試團隊"}`))
		case r.URL.Path == "/v1/dev/builds":
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"buildId":"b1","uploadPath":"/v1/dev/uploads/tok"}`))
		case r.URL.Path == "/v1/dev/uploads/tok":
			_, _ = w.Write([]byte(`{}`))
		case r.URL.Path == "/v1/dev/builds/b1":
			_, _ = w.Write([]byte(`{"artifactStatus":"ready","entryUrl":"http://b.games/","checkReport":{"fileCount":3}}`))
		case r.URL.Path == "/v1/dev/listing-revisions":
			var in map[string]any
			_ = json.Unmarshal(body, &in)
			ups := []map[string]any{}
			for i, im := range in["images"].([]any) {
				m := im.(map[string]any)
				ups = append(ups, map[string]any{"kind": m["kind"], "locale": m["locale"], "position": m["position"], "uploadPath": "/v1/dev/listing-uploads/u" + string(rune('a'+i))})
			}
			raw, _ := json.Marshal(map[string]any{"revisionId": "rev-1", "uploads": ups})
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write(raw)
		case strings.HasPrefix(r.URL.Path, "/v1/dev/listing-uploads/"):
			_, _ = w.Write([]byte(`{}`))
		case r.URL.Path == "/v1/dev/submissions":
			auth = r.Header.Get("Authorization")
			var in map[string]any
			_ = json.Unmarshal(body, &in)
			submissions = append(submissions, in)
			_, _ = w.Write([]byte(`{"buildStatus":"in_review","listingStatus":"in_review","minor":false}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(platform.Close)

	must(t, cmdLogin([]string{"-api", platform.URL, "-no-browser"}))
	info, err := os.Stat(creds)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("送審授權要存成 600：%v %v", info, err)
	}
	if c, ok := credentialFor(platform.URL); !ok || c.Token != "ptd_secret" || c.TeamName != "測試團隊" {
		t.Fatalf("保存的授權：%+v %v", c, ok)
	}

	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	keyFile := filepath.Join(t.TempDir(), "upload.key")
	must(t, os.WriteFile(keyFile, []byte(base64.StdEncoding.EncodeToString(priv.Seed())+"\n"), 0o600))
	dir := t.TempDir()
	t.Chdir(dir)
	must(t, cmdNew([]string{"auto-game", "-y", "-platform", platform.URL}))
	root := filepath.Join(dir, "auto-game")
	t.Chdir(root)
	fillStore(t, root)
	args := []string{"-api", platform.URL, "-key-id", "k1", "-key-file", keyFile, "-skip-check"}

	must(t, upload(append([]string{"-version", "1.0.0"}, args...)))
	if len(submissions) != 1 || submissions[0]["buildId"] != "b1" || submissions[0]["revisionId"] != "rev-1" || auth != "Bearer ptd_secret" {
		t.Fatalf("檢查通過後要用送審授權一起送審：%v %q", submissions, auth)
	}
	must(t, upload(append([]string{"-version", "1.0.1", "-no-submit"}, args...)))
	if len(submissions) != 1 {
		t.Fatal("-no-submit 不送審")
	}
	must(t, upload([]string{"-listing-only", "-api", platform.URL, "-key-id", "k1", "-key-file", keyFile}))
	if len(submissions) != 2 || submissions[1]["buildId"] != "" || submissions[1]["revisionId"] != "rev-1" {
		t.Fatalf("-listing-only 只送上架資料：%v", submissions)
	}
	must(t, cmdLogout([]string{"-api", platform.URL}))
	if _, ok := credentialFor(platform.URL); ok {
		t.Fatal("pt logout 要刪除授權")
	}
	// 沒有登入：照常上傳，不送審
	must(t, upload(append([]string{"-version", "1.0.2"}, args...)))
	if len(submissions) != 2 {
		t.Fatal("沒有登入不送審")
	}
}

// 送審授權只能用 https 傳送（本機開發例外）。
func TestSecurePlatform(t *testing.T) {
	for u, ok := range map[string]bool{"https://pt.example": true, "http://127.0.0.1:8080": true, "http://games.localhost:3000": true,
		"http://pt.example": false, "ftp://pt.example": false, "not a url": false} {
		if got := securePlatform(u) == nil; got != ok {
			t.Errorf("%s：%v，要 %v", u, got, ok)
		}
	}
	if err := cmdLogin([]string{"-api", "http://pt.example", "-no-browser"}); err == nil || !strings.Contains(err.Error(), "https") {
		t.Fatalf("http 的平台要拒絕：%v", err)
	}
}
