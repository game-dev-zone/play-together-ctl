package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// pt upload 送出 playtogether.json 的畫面方向：平台依此提示玩家旋轉手機。
func TestUploadSendsOrientation(t *testing.T) {
	requireEmbeddedSDK(t)
	var mu sync.Mutex
	var got map[string]any
	platform := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "POST" && r.URL.Path == "/v1/dev/builds":
			mu.Lock()
			_ = json.NewDecoder(r.Body).Decode(&got)
			mu.Unlock()
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"buildId":"b1","uploadPath":"/v1/dev/uploads/tok"}`))
		case r.Method == "PUT":
			_, _ = w.Write([]byte(`{"artifactStatus":"processing"}`))
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
	must(t, cmdNew([]string{"wide", "-template", "web-go", "-orientation", "landscape", "-platform", platform.URL}))
	t.Chdir(filepath.Join(dir, "wide"))
	run := func(extra ...string) map[string]any {
		t.Helper()
		args := append([]string{"-version", "1.0.0", "-api", platform.URL, "-key-id", "k1", "-key-file", keyFile, "-skip-check", "-wait", "0"}, extra...)
		if err := upload(args); err != nil {
			t.Fatal(err)
		}
		mu.Lock()
		defer mu.Unlock()
		return got
	}
	first := run()
	if o := first["orientation"]; o != "landscape" {
		t.Fatalf("要送出 playtogether.json 的方向：%v", o)
	}
	if m := first["gameServerMode"]; m != "playtogether-managed" {
		t.Fatalf("要送出遊戲伺服器架構：%v", m)
	}
	if o := run("-orientation", "any")["orientation"]; o != "any" {
		t.Fatalf("參數優先：%v", o)
	}
	if err := upload([]string{"-version", "1.0.0", "-api", platform.URL, "-key-id", "k1", "-key-file", keyFile, "-skip-check", "-orientation", "sideways"}); err == nil {
		t.Fatal("不合法的方向要拒絕")
	}
}
