package main

// pt dev 的本機假平台：啟動假平台與控制面板、準備遊戲伺服器的金鑰。假平台本身在 fakeplatform.go。

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"
)

// devPanel 是控制面板（假平台網址的 /）與它用的 API。
type devPanel struct {
	fp   *fakePlatform
	site string // 遊戲首頁
	self string // 假平台網址
}

func (d *devPanel) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/":
		page, err := embedded.ReadFile("assets/dev/panel.html")
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write(page)
	case r.Method == http.MethodGet && r.URL.Path == "/dev/state":
		after := 0
		_ = json.Unmarshal([]byte(r.URL.Query().Get("after")), &after)
		s := d.fp.state(after)
		d.fp.write(w, 200, struct {
			fakeState
			Site string `json:"site"`
			Self string `json:"self"`
		}{s, d.site, d.self})
	case r.Method == http.MethodPost && r.URL.Path == "/dev/control":
		// 只接受面板自己送的（擋住其他網站對本機的跨站請求）
		if r.Header.Get("Origin") != d.self || !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
			d.fp.write(w, 403, map[string]string{"error": "origin"})
			return
		}
		var in struct{ Action, Player string }
		if json.NewDecoder(http.MaxBytesReader(w, r.Body, 1024)).Decode(&in) != nil {
			d.fp.write(w, 400, map[string]string{"error": "bad_request"})
			return
		}
		if err := d.control(in.Action, in.Player); err != nil {
			d.fp.write(w, 400, map[string]string{"error": err.Error()})
			return
		}
		d.fp.write(w, 200, map[string]bool{"ok": true})
	default:
		http.NotFound(w, r)
	}
}

func (d *devPanel) control(action, player string) error {
	f := d.fp
	switch action {
	case "kick", "unkick":
		if !fakePlayerCode.MatchString(player) {
			return errors.New("bad_player")
		}
		if action == "kick" {
			f.kick(player)
		} else {
			f.unkick(player)
		}
	case "restart":
		f.restart()
	case "close":
		f.closeRoom()
	case "new-room":
		f.openNewRoom()
	case "down":
		f.mu.Lock()
		f.down = !f.down
		f.mu.Unlock()
	case "disable":
		f.mu.Lock()
		f.disabled = !f.disabled
		f.mu.Unlock()
	case "skew":
		f.mu.Lock()
		if f.skew == 0 {
			f.skew = 6 * time.Minute
		} else {
			f.skew = 0
		}
		f.mu.Unlock()
	default:
		return errors.New("bad_action")
	}
	return nil
}

// startFakePlatform 在 127.0.0.1 的空埠啟動假平台（含控制面板）。回傳假平台網址。
func startFakePlatform(fp *fakePlatform, site string) (string, *http.Server, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", nil, err
	}
	self := "http://" + ln.Addr().String()
	_, port, _ := net.SplitHostPort(ln.Addr().String())
	hosts := map[string]bool{"127.0.0.1:" + port: true, "localhost:" + port: true}
	inner := fp.handler(&devPanel{fp: fp, site: site, self: self})
	// 只接受 Host 是本機的請求：擋住 DNS rebinding（別的網域解析到 127.0.0.1 後讀取面板或操作假平台）
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !hosts[r.Host] {
			http.Error(w, "forbidden host", http.StatusForbidden)
			return
		}
		inner.ServeHTTP(w, r)
	})
	srv := &http.Server{Handler: handler, ReadHeaderTimeout: 10 * time.Second}
	go func() { _ = srv.Serve(ln) }()
	return self, srv, nil
}

// devServerKey 準備遊戲伺服器的私鑰：專案已經有 keys/server.key 就用它（順便確認檔案格式正確），
// 沒有就產生一把只供本機的暫存金鑰（不寫進專案、不登記到玩聚）。cleanup 刪除暫存金鑰。
func devServerKey(p *project) (keyFile string, pub ed25519.PublicKey, local bool, cleanup func(), err error) {
	if p.ServerKeyFile != "" {
		path := p.path(p.ServerKeyFile)
		if _, statErr := os.Stat(path); statErr == nil {
			priv, err := readKey(path)
			if err != nil {
				return "", nil, false, nil, err
			}
			abs, _ := filepath.Abs(path)
			return abs, priv.Public().(ed25519.PublicKey), false, func() {}, nil
		}
	}
	dir, err := os.MkdirTemp("", "pt-dev-key-")
	if err != nil {
		return "", nil, false, nil, err
	}
	pubKey, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", nil, false, nil, err
	}
	keyFile = filepath.Join(dir, "server.key")
	if err := os.WriteFile(keyFile, []byte(base64.StdEncoding.EncodeToString(priv.Seed())+"\n"), 0o600); err != nil {
		return "", nil, false, nil, err
	}
	return keyFile, pubKey, true, func() { os.RemoveAll(dir) }, nil
}

// devPlayerURL 是玩家分頁的網址：瀏覽器 SDK 看到 pt_dev、pt_player 就向假平台取啟動碼。
func devPlayerURL(site, platform, player string) string {
	u, err := url.Parse(site)
	if err != nil {
		return site
	}
	q := u.Query()
	q.Set("pt_dev", platform)
	q.Set("pt_player", player)
	u.RawQuery = q.Encode()
	return u.String()
}

// devServerEnv 是遊戲伺服器連到假平台需要的環境變數（PT_MOCK 清空，避免沿用終端機裡的設定）。
func devServerEnv(platform, keyFile string) []string {
	return []string{"PT_MOCK=", "PT_API_URL=" + platform, "PT_KEY_ID=" + fakeKeyID, "PT_KEY_FILE=" + keyFile}
}

// devPlatformOnly（pt dev -no-server）：只啟動假平台與控制面板，印出自己的遊戲伺服器要設定的環境變數。
func devPlatformOnly(p *project, site, addr string, players int) error {
	if site == "" {
		site = "http://" + addr + "/"
	}
	keyFile, pub, local, cleanup, err := devServerKey(p)
	if err != nil {
		return fmt.Errorf("遊戲伺服器金鑰：%w", err)
	}
	defer cleanup()
	fp := newFakePlatform(pub)
	platform, srv, err := startFakePlatform(fp, site)
	if err != nil {
		return fmt.Errorf("啟動本機假平台失敗：%w", err)
	}
	defer srv.Close()
	fmt.Printf("本機假平台：%s（控制面板與呼叫紀錄）\n", platform)
	fmt.Println("請用下面的環境變數啟動你的遊戲伺服器（只供本機；不要設 PT_MOCK）：")
	for _, kv := range devServerEnv(platform, keyFile)[1:] {
		fmt.Println("  " + kv)
	}
	if local {
		fmt.Printf("（沒有 %s，用只供本機的暫存金鑰；pt dev 結束時刪除）\n", p.ServerKeyFile)
	}
	fmt.Printf("玩家分頁：%s（換 pt_player 就是另一位玩家）\n", devPlayerURL(site, platform, "p1"))
	fmt.Println("你的遊戲前端要用玩聚瀏覽器 SDK 0.1.0-20261106 以上（pt sdk update）。Ctrl-C 結束")
	if players > 0 {
		openBrowser(platform + "/")
	}
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	<-stop
	return nil
}
