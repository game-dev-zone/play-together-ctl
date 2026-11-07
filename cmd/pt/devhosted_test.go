package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// 假平台發啟動碼時，代管專案的本機伺服器網址一起給（SDK 放進 launch.serverUrl）；沒有設定時不給。
func TestFakePlatformServerURL(t *testing.T) {
	pub, _, _ := ed25519.GenerateKey(rand.Reader)
	fp := newFakePlatform(pub)
	self, srv, err := startFakePlatform(fp, "http://127.0.0.1:18090/")
	must(t, err)
	t.Cleanup(func() { srv.Close() })
	launch := func() map[string]string {
		req, _ := http.NewRequest(http.MethodGet, self+"/dev/launch-code?player=p1", nil)
		req.Header.Set("Origin", "http://127.0.0.1:18090")
		res, err := http.DefaultClient.Do(req)
		must(t, err)
		defer res.Body.Close()
		var out map[string]string
		must(t, json.NewDecoder(res.Body).Decode(&out))
		return out
	}
	if v, ok := launch()["serverUrl"]; ok {
		t.Fatalf("自己部署的專案不給 serverUrl：%q", v)
	}
	fp.serverURL = "http://127.0.0.1:18092"
	if v := launch()["serverUrl"]; v != "http://127.0.0.1:18092" {
		t.Fatalf("serverUrl：%q", v)
	}
}

func TestLocalOrigin(t *testing.T) {
	for in, want := range map[string]string{
		"http://127.0.0.1:9192": "http://127.0.0.1:9192", "http://localhost:9192/": "http://localhost:9192", "http://[::1]:9192": "http://[::1]:9192",
	} {
		if got, err := localOrigin(in); err != nil || got != want {
			t.Errorf("localOrigin(%q) = %q, %v", in, got, err)
		}
	}
	for _, in := range []string{"https://127.0.0.1:9192", "http://192.168.1.2:9192", "http://127.0.0.1:9192/ws", "http://evil.test", "127.0.0.1:9192"} {
		if _, err := localOrigin(in); err == nil {
			t.Errorf("localOrigin(%q) 應該擋下", in)
		}
	}
}

const hostedServerMain = `package main

import (
	"net/http"
	"os"
)

func main() {
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) })
	if os.Getenv("NO_READYZ") == "" {
		http.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) })
	}
	if os.Getenv("ALLOW_ORIGINS") == "" || os.Getenv("PORT") == "" {
		os.Exit(3) // 代管容器一定有這些環境變數，本機也要給
	}
	_ = http.ListenAndServe(os.Getenv("ADDR"), nil)
}
`

// pt dev 跑代管專案：畫面和遊戲伺服器分開兩個來源，伺服器拿到代管的環境變數，/healthz、/readyz 正常才算啟動。
func TestDevHostedSmoke(t *testing.T) {
	requireEmbeddedSDK(t)
	dir := filepath.Join(t.TempDir(), "hosted-dev")
	must(t, os.MkdirAll(filepath.Join(dir, "web"), 0o755))
	must(t, os.WriteFile(filepath.Join(dir, "web", "index.html"), []byte(`<!doctype html><script src="playtogether-sdk.iife.js"></script><script>pt.launch.serverUrl</script>`), 0o644))
	must(t, os.WriteFile(filepath.Join(dir, "web", "playtogether-sdk.iife.js"), []byte("// sdk"), 0o644))
	must(t, os.MkdirAll(filepath.Join(dir, "srv"), 0o755))
	must(t, os.WriteFile(filepath.Join(dir, "srv", "go.mod"), []byte("module srv\n\ngo 1.24\n"), 0o644))
	must(t, os.WriteFile(filepath.Join(dir, "srv", "main.go"), []byte(hostedServerMain), 0o644))
	t.Chdir(dir)
	must(t, cmdInit([]string{"-y", "-hosting", "playtogether", "-server-dir", "srv", "-platform", "https://pt.example"}))

	if err := cmdDev([]string{"-smoke"}); err != nil {
		t.Fatalf("pt dev -smoke：%v", err)
	}
	if err := cmdDev([]string{"-server-url", "http://127.0.0.1:9"}); err == nil || !strings.Contains(err.Error(), "-no-server") {
		t.Fatalf("-server-url 沒有 -no-server：%v", err)
	}
	t.Setenv("NO_READYZ", "1")
	if err := cmdDev([]string{"-smoke"}); err == nil || !strings.Contains(err.Error(), "/readyz") {
		t.Fatalf("沒有 /readyz 要說清楚：%v", err)
	}
}
