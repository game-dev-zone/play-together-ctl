package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// 範本要能建立、編譯、在模擬模式下真的玩：join（模擬啟動碼）→ start → tap。
// 需要內嵌的 SDK（cmd/pt/assets/sdk，由玩聚同步）；沒有時略過。
func requireEmbeddedSDK(t *testing.T) {
	t.Helper()
	if sdkVersion() == "" {
		t.Skip("pt 沒有內嵌 SDK（cmd/pt/assets/sdk）")
	}
}

func freeAddr(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	return ln.Addr().String()
}

func postJSON(t *testing.T, url string, body any, out any) int {
	t.Helper()
	raw, _ := json.Marshal(body)
	res, err := http.Post(url, "application/json", bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if out != nil {
		_ = json.NewDecoder(res.Body).Decode(out)
	}
	return res.StatusCode
}

// playMock 啟動伺服器（模擬模式），用兩位模擬玩家玩一局。
func playMock(t *testing.T, cmd *exec.Cmd, addr string) {
	t.Helper()
	var logs bytes.Buffer
	cmd.Stdout, cmd.Stderr = &logs, &logs
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill(); _ = cmd.Wait() })
	base := "http://" + addr
	for i := 0; ; i++ {
		if res, err := http.Get(base + "/"); err == nil {
			body := new(bytes.Buffer)
			_, _ = body.ReadFrom(res.Body)
			res.Body.Close()
			if !strings.Contains(body.String(), "playtogether-sdk.iife.js") {
				t.Fatalf("首頁沒有載入 SDK：%s", body.String())
			}
			break
		}
		if i > 300 {
			t.Fatalf("伺服器沒有啟動：%s", logs.String())
		}
		time.Sleep(200 * time.Millisecond)
	}
	var a, b struct{ Token, Name, PlayerID string }
	if st := postJSON(t, base+"/join", map[string]string{"launchCode": "mock-aaaa1111bbbb"}, &a); st != 200 || a.Token == "" {
		t.Fatalf("模擬玩家 A 進房失敗：%d %+v %s", st, a, logs.String())
	}
	postJSON(t, base+"/join", map[string]string{"launchCode": "mock-cccc2222dddd"}, &b)
	if a.Name == b.Name {
		t.Fatalf("兩位模擬玩家的暱稱應分得出來：%s", a.Name)
	}
	if st := postJSON(t, base+"/join", map[string]string{"launchCode": "real-code-not-mock"}, nil); st == 200 {
		t.Fatal("模擬模式不能接受非 mock- 的啟動碼")
	}
	postJSON(t, base+"/action", map[string]string{"token": a.Token, "action": "start"}, nil)
	var st struct {
		Playing bool
		Players []struct {
			Name string
			Taps int
		}
	}
	postJSON(t, base+"/action", map[string]string{"token": a.Token, "action": "tap"}, nil)
	postJSON(t, base+"/action", map[string]string{"token": a.Token, "action": "tap"}, &st)
	if !st.Playing || len(st.Players) != 2 || st.Players[0].Taps != 2 || st.Players[0].Name != a.Name {
		t.Fatalf("一局的狀態不對：%+v", st)
	}
	if code := postJSON(t, base+"/action", map[string]string{"token": "bad", "action": "tap"}, nil); code != http.StatusUnauthorized {
		t.Fatalf("錯的 token 要拒絕：%d", code)
	}
}

func newProject(t *testing.T, tmpl string) string {
	t.Helper()
	dir := t.TempDir()
	t.Chdir(dir)
	if err := cmdNew([]string{"my-game", "-template", tmpl, "-platform", "https://pt.example"}); err != nil {
		t.Fatal(err)
	}
	return filepath.Join(dir, "my-game")
}

func TestNewWebGo(t *testing.T) {
	requireEmbeddedSDK(t)
	root := newProject(t, "web-go")
	for _, f := range []string{"playtogether.json", ".gitignore", "README.md", "web/index.html", "web/config.js", "web/playtogether-sdk.iife.js",
		"server/main.go", "server/go.mod", "server/third_party/playtogether-sdk-go/go.mod", "server/third_party/playtogether-sdk-go/client.go"} {
		if _, err := os.Stat(filepath.Join(root, f)); err != nil {
			t.Fatalf("少了 %s", f)
		}
	}
	raw, _ := os.ReadFile(filepath.Join(root, "web/config.js"))
	if !strings.Contains(string(raw), "https://pt.example") {
		t.Fatalf("config.js 要帶入平台網址：%s", raw)
	}
	bin := filepath.Join(t.TempDir(), "server")
	build := exec.Command("go", "build", "-o", bin, ".")
	build.Dir = filepath.Join(root, "server")
	build.Env = append(os.Environ(), "GOWORK=off", "GOFLAGS=-mod=mod", "GOPROXY=off")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("範本的 Go 伺服器編譯失敗：%v\n%s", err, out)
	}
	addr := freeAddr(t)
	cmd := exec.Command(bin)
	cmd.Dir = filepath.Join(root, "server")
	cmd.Env = append(os.Environ(), "PT_MOCK=1", "ADDR="+addr, "WEB_DIR="+filepath.Join(root, "web"))
	playMock(t, cmd, addr)
}

func TestNewWebNode(t *testing.T) {
	requireEmbeddedSDK(t)
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("沒有 node")
	}
	root := newProject(t, "web-node")
	addr := freeAddr(t)
	cmd := exec.Command("node", "main.mjs")
	cmd.Dir = filepath.Join(root, "server")
	cmd.Env = append(os.Environ(), "PT_MOCK=1", "ADDR="+addr, "WEB_DIR="+filepath.Join(root, "web"))
	playMock(t, cmd, addr)
}

func TestNewRejects(t *testing.T) {
	requireEmbeddedSDK(t)
	t.Chdir(t.TempDir())
	for _, args := range [][]string{{"My Game"}, {"a"}, {"ok-name", "-template", "unity"}} {
		if err := cmdNew(args); err == nil {
			t.Fatalf("應拒絕 %v", args)
		}
	}
	if err := cmdNew([]string{"dup-game"}); err != nil {
		t.Fatal(err)
	}
	if err := cmdNew([]string{"dup-game"}); err == nil {
		t.Fatal("資料夾已存在時要拒絕，不能覆蓋")
	}
}

// pt sdk update 換掉專案裡的 SDK 檔案並記錄版本。
func TestSDKUpdate(t *testing.T) {
	requireEmbeddedSDK(t)
	root := newProject(t, "web-node")
	t.Chdir(root)
	stale := filepath.Join(root, "server/playtogether-server.mjs")
	must(t, os.WriteFile(stale, []byte("// old"), 0o644))
	raw, _ := os.ReadFile(filepath.Join(root, projectFile))
	must(t, os.WriteFile(filepath.Join(root, projectFile), bytes.Replace(raw, []byte(sdkVersion()), []byte("0.0.1"), 1), 0o644))
	if err := cmdSDK([]string{"update"}); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(stale); string(b) == "// old" {
		t.Fatal("SDK 檔案沒有更新")
	}
	p, _ := findProject()
	if p.SDKVersion != sdkVersion() || p.ServerURL != "" || p.Template != "web-node" {
		t.Fatalf("playtogether.json 更新後要保留其他設定：%+v", p)
	}
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func TestCheck(t *testing.T) {
	requireEmbeddedSDK(t)
	root := newProject(t, "web-go")
	t.Chdir(root)
	p, _ := findProject()
	web := filepath.Join(root, "web")
	has := func(fs []finding, level, sub string) bool {
		for _, f := range fs {
			if (f.level == level || level == "error" && isErrorLevel(f.level)) && strings.Contains(f.msg, sub) {
				return true
			}
		}
		return false
	}
	// 新專案：serverUrl 還沒填
	if fs := checkProject(p, web); !has(fs, "error", "serverUrl 還沒填") {
		t.Fatalf("要提醒填 serverUrl：%+v", fs)
	}
	// 填了 playtogether.json 但 config.js 沒改：照抄相對網址的問題
	p.ServerURL = "https://game.example.com"
	if fs := checkProject(p, web); !has(fs, "error", "web/config.js 的 serverUrl") {
		t.Fatalf("config.js 和 playtogether.json 不一致要擋：%+v", fs)
	}
	cfg, _ := os.ReadFile(filepath.Join(web, "config.js"))
	must(t, os.WriteFile(filepath.Join(web, "config.js"), bytes.Replace(cfg, []byte("serverUrl: ''"), []byte("serverUrl: 'https://game.example.com'"), 1), 0o644))
	if fs := checkProject(p, web); len(fs) != 0 {
		t.Fatalf("設定正確時要通過：%+v", fs)
	}
	p.ServerURL = "http://game.example.com"
	if fs := checkProject(p, web); !has(fs, "error", "https://") {
		t.Fatal("正式的遊戲伺服器要 https")
	}
	p.ServerURL = "https://game.example.com"
	// 審核會擋的東西
	must(t, os.WriteFile(filepath.Join(web, "upload.key"), []byte("secret"), 0o600))
	must(t, os.WriteFile(filepath.Join(web, "ads.js"), []byte(`load("https://www.googletagmanager.com/gtag/js")`), 0o644))
	must(t, os.WriteFile(filepath.Join(web, "p2p.js"), []byte(`new RTCPeerConnection()`), 0o644))
	fs := checkProject(p, web)
	for _, want := range []string{"私鑰", "googletagmanager", "WebRTC"} {
		if !has(fs, "error", want) {
			t.Fatalf("要擋下 %s：%+v", want, fs)
		}
	}
	must(t, os.Remove(filepath.Join(web, "index.html")))
	if fs := checkProject(p, web); !has(fs, "error", "index.html") {
		t.Fatal("沒有 index.html 要擋")
	}
}

func TestReorder(t *testing.T) {
	got := fmt.Sprint(reorder([]string{"my-game", "-template", "web-node", "-x=1"}))
	if got != "[-template web-node -x=1 my-game]" {
		t.Fatal(got)
	}
}

func TestNewMakefile(t *testing.T) {
	requireEmbeddedSDK(t)
	for _, tc := range []struct {
		tmpl  string
		godot bool
	}{{"web-go", false}, {"godot-node", true}} {
		root := newProject(t, tc.tmpl)
		raw, err := os.ReadFile(filepath.Join(root, "Makefile"))
		if err != nil {
			t.Fatalf("%s 沒有 Makefile", tc.tmpl)
		}
		mk := string(raw)
		for _, target := range []string{"help:", "dev:", "server:", "test:", "check:", "verify:", "keys:", "upload:"} {
			if !strings.Contains(mk, "\n"+target) {
				t.Fatalf("%s 的 Makefile 少了 %s", tc.tmpl, target)
			}
		}
		if strings.Contains(mk, "\nbuild:") != tc.godot || strings.Contains(mk, "\nclean:") != tc.godot {
			t.Fatalf("%s：只有 Godot 專案要有 build、clean", tc.tmpl)
		}
		if !strings.Contains(mk, "\t$(PT) dev -smoke") || !strings.Contains(mk, "\t$(PT) check -local") {
			t.Fatalf("make verify 要跑啟動驗證與部署前檢查：%s", mk)
		}
		if _, err := exec.LookPath("make"); err == nil {
			for _, args := range [][]string{{"-n", "verify"}, {"-s", "help"}} {
				c := exec.Command("make", args...)
				c.Dir = root
				if out, err := c.CombinedOutput(); err != nil {
					t.Fatalf("%s：make %v 失敗：%v\n%s", tc.tmpl, args, err, out)
				}
			}
			c := exec.Command("make", "upload")
			c.Dir = root
			if out, err := c.CombinedOutput(); err == nil || !strings.Contains(string(out), "請指定版本") {
				t.Fatalf("沒有 VERSION 時 make upload 要擋下：%s", out)
			}
		}
	}
}

func TestCheckLocal(t *testing.T) {
	requireEmbeddedSDK(t)
	root := newProject(t, "web-go")
	t.Chdir(root)
	if err := cmdCheck(nil); err == nil {
		t.Fatal("新專案沒填 platform、serverUrl，pt check 要失敗")
	}
	if err := cmdCheck([]string{"-local"}); err != nil {
		t.Fatalf("pt check -local 不該因為還沒部署而失敗：%v", err)
	}
	// -local 只放寬「還沒填」：私鑰放進上傳資料夾仍然是錯誤
	if err := os.WriteFile(filepath.Join(root, "web", "server.key"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := cmdCheck([]string{"-local"}); err == nil {
		t.Fatal("pt check -local 仍要擋下私鑰")
	}
}

func TestDevSmoke(t *testing.T) {
	requireEmbeddedSDK(t)
	t.Setenv("GOWORK", "off")
	t.Setenv("GOFLAGS", "-mod=mod")
	t.Setenv("GOPROXY", "off")
	root := newProject(t, "web-go")
	t.Chdir(root)
	if err := cmdDev([]string{"-smoke"}); err != nil {
		t.Fatalf("pt dev -smoke 應通過：%v", err)
	}
	// 首頁沒有載入 SDK：要失敗
	idx := filepath.Join(root, "web", "index.html")
	raw, _ := os.ReadFile(idx)
	if err := os.WriteFile(idx, []byte(strings.ReplaceAll(string(raw), "playtogether-sdk", "other")), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := cmdDev([]string{"-smoke"}); err == nil || !strings.Contains(err.Error(), "SDK") {
		t.Fatalf("首頁沒有 SDK 時 pt dev -smoke 要失敗：%v", err)
	}
}
