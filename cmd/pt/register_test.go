package main

import (
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

// pt login -register 另外保存建立遊戲的授權；pt register 產生金鑰、建立遊戲、寫回金鑰 ID；重試時接手已建立的遊戲。
func TestRegister(t *testing.T) {
	requireEmbeddedSDK(t)
	t.Setenv("PT_CREDENTIALS_FILE", filepath.Join(t.TempDir(), "credentials.json"))
	var mu sync.Mutex
	var scopes []any
	var created, registered map[string]string
	var auth string
	conflict := false // 模擬上次已經建立、這次回 id_unavailable
	platform := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		body, _ := io.ReadAll(r.Body)
		switch {
		case r.URL.Path == "/v1/dev/device-codes":
			var in map[string]any
			_ = json.Unmarshal(body, &in)
			scopes, _ = in["scopes"].([]any)
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"deviceCode":"d","userCode":"ABCD-EFGH","verificationUrl":"http://web/dev/login","expiresIn":60,"interval":0.001}`))
		case r.URL.Path == "/v1/dev/device-codes/token":
			_, _ = w.Write([]byte(`{"token":"ptd_reg","expiresAt":"2099-01-01T00:00:00Z","teamId":"t1","teamName":"測試團隊","scopes":["submit","register"]}`))
		case r.Method == http.MethodPost && r.URL.Path == "/v1/dev/games":
			auth = r.Header.Get("Authorization")
			created = map[string]string{} // 新的 map：Unmarshal 會沿用舊的 map
			_ = json.Unmarshal(body, &created)
			if conflict {
				w.WriteHeader(http.StatusConflict)
				_, _ = w.Write([]byte(`{"error":"id_unavailable"}`))
				return
			}
			registered = created
			w.WriteHeader(http.StatusCreated)
			out := map[string]any{"gameId": created["id"], "uploadKeyId": created["id"] + "-upload-abc", "serverKeyId": nil}
			if created["serverPublicKey"] != "" {
				out["serverKeyId"] = created["id"] + "-server-abc"
			}
			raw, _ := json.Marshal(out)
			_, _ = w.Write(raw)
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v1/dev/games/"):
			if registered == nil {
				http.NotFound(w, r)
				return
			}
			raw, _ := json.Marshal(map[string]any{"id": registered["id"], "keys": []map[string]any{
				{"id": registered["id"] + "-upload-old", "purpose": "upload", "publicKey": registered["uploadPublicKey"], "status": "active"},
				{"id": registered["id"] + "-server-old", "purpose": "server", "publicKey": registered["serverPublicKey"], "status": "active"},
			}})
			_, _ = w.Write(raw)
		default:
			http.NotFound(w, r)
		}
	}))
	defer platform.Close()

	root := newProject(t, "web-go")
	t.Chdir(root)
	p, _ := findProject()
	p.Platform = platform.URL
	must(t, p.save())

	// 還沒有建立遊戲的授權
	if err := cmdRegister(nil); err == nil || !strings.Contains(err.Error(), "pt login -register") {
		t.Fatalf("沒有授權：%v", err)
	}
	// 一般的 pt login 不會帶 register
	must(t, cmdLogin([]string{"-no-browser"}))
	if scopes != nil {
		t.Fatalf("一般登入不要求 register：%v", scopes)
	}
	if err := cmdRegister(nil); err == nil {
		t.Fatal("送審授權不能用來建立遊戲")
	}
	must(t, cmdLogin([]string{"-register", "-no-browser"}))
	if len(scopes) != 2 || scopes[1] != "register" {
		t.Fatalf("要求的範圍：%v", scopes)
	}
	creds := loadCredentials()
	if _, ok := creds[platform.URL]; !ok {
		t.Fatal("送審授權被覆蓋了")
	}
	if _, ok := creds[registerKey(platform.URL)]; !ok {
		t.Fatal("沒有保存建立遊戲的授權")
	}

	// 私鑰格式錯誤：停下來，不覆寫
	keyFile := filepath.Join(root, "keys", "upload.key")
	must(t, os.MkdirAll(filepath.Dir(keyFile), 0o700))
	must(t, os.WriteFile(keyFile, []byte("not a key\n"), 0o600))
	if err := cmdRegister(nil); err == nil || !strings.Contains(err.Error(), "不是 pt keygen") {
		t.Fatalf("壞掉的私鑰：%v", err)
	}
	if raw, _ := os.ReadFile(keyFile); string(raw) != "not a key\n" {
		t.Fatal("壞掉的私鑰被覆寫")
	}
	must(t, os.Remove(keyFile))

	// 建立：產生兩把私鑰（權限 600），只送公鑰，金鑰 ID 寫回
	must(t, cmdRegister([]string{"-title", "我的遊戲"}))
	if auth != "Bearer ptd_reg" || created["id"] != "my-game" || created["name"] != "我的遊戲" || created["serverPublicKey"] == "" {
		t.Fatalf("送出的內容：%s %v", auth, created)
	}
	for _, f := range []string{"upload.key", "server.key"} {
		info, err := os.Stat(filepath.Join(root, "keys", f))
		if err != nil || info.Mode().Perm() != 0o600 {
			t.Fatalf("%s：%v %v", f, err, info)
		}
		raw, _ := os.ReadFile(filepath.Join(root, "keys", f))
		if strings.Contains(created["uploadPublicKey"]+created["serverPublicKey"], strings.TrimSpace(string(raw))) {
			t.Fatal("私鑰被送出")
		}
	}
	p, _ = findProject()
	if p.UploadKeyID != "my-game-upload-abc" || p.ServerKeyID != "my-game-server-abc" {
		t.Fatalf("playtogether.json：%s %s", p.UploadKeyID, p.ServerKeyID)
	}
	if err := cmdRegister(nil); err == nil || !strings.Contains(err.Error(), "已經登記過") {
		t.Fatalf("重複登記：%v", err)
	}

	// 上次建立成功但沒收到回應：用公鑰比對接手
	p.UploadKeyID, p.ServerKeyID = "", ""
	must(t, p.save())
	conflict = true
	must(t, cmdRegister(nil))
	p, _ = findProject()
	if p.UploadKeyID != "my-game-upload-old" || p.ServerKeyID != "my-game-server-old" {
		t.Fatalf("接手：%s %s", p.UploadKeyID, p.ServerKeyID)
	}
	// 用途對不上：不接手（公鑰相同但用途錯置）
	p.UploadKeyID, p.ServerKeyID = "", ""
	must(t, p.save())
	mu.Lock()
	registered["serverPublicKey"], registered["uploadPublicKey"] = registered["uploadPublicKey"], registered["serverPublicKey"]
	mu.Unlock()
	if err := cmdRegister(nil); err == nil {
		t.Fatal("用途錯置時不能接手")
	}
	mu.Lock()
	registered["serverPublicKey"], registered["uploadPublicKey"] = registered["uploadPublicKey"], registered["serverPublicKey"]
	mu.Unlock()
	// 代號被別人用了（查不到自己的遊戲）
	p.UploadKeyID = ""
	must(t, p.save())
	must(t, os.Remove(filepath.Join(root, "keys", "upload.key"))) // 新的公鑰對不上
	if err := cmdRegister(nil); err == nil || !strings.Contains(err.Error(), "不能使用") {
		t.Fatalf("代號不能用：%v", err)
	}
}
