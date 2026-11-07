package main

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// devHosted：玩聚代管的專案（gameServer.hosting: "playtogether"）在本機的樣子。
//
// 和正式平台一樣分成兩個來源：畫面（上傳資料夾）由 pt dev 提供在 addr，遊戲伺服器（gameServer.dir 的 Go 模組）
// 跑在另一個埠，環境變數照代管容器注入（ADDR、PORT、PT_API_URL、PT_KEY_ID、PT_KEY_FILE、ALLOW_ORIGINS）。
// 假平台發啟動碼時把伺服器網址當成 launch.serverUrl 給畫面，所以前端的程式不用為了本機另外改。
func devHosted(p *project, addr string, players int, smoke, mock bool) error {
	site := "http://" + addr + "/"
	front, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("%s 已經被其他程式使用，請換一個：pt dev -addr 127.0.0.1:18090", addr)
	}
	printSDKNotice(p)
	if p.Client == "godot" {
		if err := buildGodot(p); err != nil {
			front.Close()
			return err
		}
	}
	web := p.path(p.Web)
	if _, err := os.Stat(filepath.Join(web, "index.html")); err != nil {
		front.Close()
		return fmt.Errorf("%s 裡沒有 index.html：先建置畫面（playtogether.json 的 web）", p.Web)
	}
	files := http.FileServer(http.Dir(web))
	frontSrv := &http.Server{ReadHeaderTimeout: 10 * time.Second, Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store") // 改了畫面重新整理就看得到
		files.ServeHTTP(w, r)
	})}
	go func() { _ = frontSrv.Serve(front) }()
	defer frontSrv.Close()

	// 遊戲伺服器：先編譯再執行（go run 不會把結束訊號轉給伺服器）
	srvLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	serverAddr := srvLn.Addr().String()
	srvLn.Close()
	serverURL := "http://" + serverAddr
	tmp, err := os.MkdirTemp("", "pt-hosted-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	bin := filepath.Join(tmp, "server")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	dir := p.path(serverDir(p.GameServer))
	fmt.Println("編譯遊戲伺服器：", dir)
	build := exec.Command("go", "build", "-o", bin, ".")
	build.Dir = dir
	build.Stdout, build.Stderr = os.Stdout, os.Stderr
	if err := build.Run(); err != nil {
		return fmt.Errorf("遊戲伺服器編譯失敗（代管時 pt upload 也會在 %s 執行 go build .）：%w", serverDir(p.GameServer), err)
	}
	_, port, _ := net.SplitHostPort(serverAddr)
	cmd := exec.Command(bin)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "ADDR="+serverAddr, "PORT="+port, "ALLOW_ORIGINS="+strings.TrimSuffix(site, "/"), "TRUST_PROXY=")

	platform, keyNote := "", ""
	if smoke || mock {
		cmd.Env = append(cmd.Env, "PT_MOCK=1")
	} else {
		keyFile, pub, local, cleanup, err := devServerKey(p)
		if err != nil {
			return fmt.Errorf("遊戲伺服器金鑰：%w", err)
		}
		defer cleanup()
		fp := newFakePlatform(pub)
		fp.serverURL = serverURL
		var srv *http.Server
		platform, srv, err = startFakePlatform(fp, site)
		if err != nil {
			return fmt.Errorf("啟動本機假平台失敗：%w", err)
		}
		defer srv.Close()
		cmd.Env = append(cmd.Env, devServerEnv(platform, keyFile)...)
		aiEnv, aiCleanup, err := attachDevAI(fp, p, platform)
		if err != nil {
			return fmt.Errorf("本機 AI 閘道：%w", err)
		}
		defer aiCleanup()
		cmd.Env = append(cmd.Env, aiEnv...)
		keyNote = "專案的 " + p.ServerKeyFile
		if local {
			keyNote = "只供本機的暫存金鑰（沒有 " + p.ServerKeyFile + "；不會登記到玩聚）"
		}
	}
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("啟動遊戲伺服器失敗：%w", err)
	}
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	// 和平台判斷部署完成一樣：/healthz 與 /readyz 都要 200
	if err := waitHealthy(serverURL, done); err != nil {
		stopServer(cmd)
		return err
	}
	if smoke {
		err := smokeCheck(site)
		stopServer(cmd)
		<-done
		if err != nil {
			return fmt.Errorf("啟動驗證沒有通過：%w", err)
		}
		fmt.Println("✓ 啟動驗證通過：遊戲伺服器（模擬模式）的 /healthz、/readyz 正常，首頁與玩聚 SDK 都載入得到")
		return nil
	}
	fmt.Printf("\n畫面：%s　遊戲伺服器（玩聚代管的本機版）：%s\n", site, serverURL)
	if platform == "" {
		fmt.Printf("本機模擬模式：每開一個分頁就是一位玩家（模擬模式沒有 launch.serverUrl）；Ctrl-C 結束\n")
		for i := 0; i < players; i++ {
			openBrowser(site)
		}
	} else {
		fmt.Printf("本機假平台：%s（控制面板：踢人、重新開始、關房、停用遊戲、平台連不到、時鐘偏差、呼叫紀錄）\n", platform)
		fmt.Printf("玩家分頁：%s（換 pt_player 就是另一位玩家；握手時 launch.serverUrl 是 %s）\n", devPlayerURL(site, platform, "p1"), serverURL)
		fmt.Printf("簽章金鑰：%s\n", keyNote)
		fmt.Println("重新整理玩家分頁會拿到新的啟動碼；在控制面板新增玩家。Ctrl-C 結束")
		if players > 0 {
			openBrowser(platform + "/")
		}
		for i := 1; i <= players; i++ {
			openBrowser(devPlayerURL(site, platform, fmt.Sprintf("p%d", i)))
		}
	}
	select {
	case <-stop:
		stopServer(cmd)
		<-done
		return nil
	case err := <-done:
		return fmt.Errorf("遊戲伺服器結束了：%v", err)
	}
}

// waitHealthy 等遊戲伺服器的 /healthz、/readyz 都回 200（最多 30 秒）。
func waitHealthy(serverURL string, done <-chan error) error {
	ok := func(path string) bool {
		res, err := http.Get(serverURL + path)
		if err != nil {
			return false
		}
		res.Body.Close()
		return res.StatusCode == http.StatusOK
	}
	for i := 0; i < 150; i++ {
		select {
		case err := <-done:
			return fmt.Errorf("遊戲伺服器結束了：%v（代管的伺服器要監聽 ADDR 或 PORT）", err)
		case <-time.After(200 * time.Millisecond):
		}
		if ok("/healthz") {
			if ok("/readyz") {
				return nil
			}
			if i > 25 {
				return errors.New("遊戲伺服器的 /readyz 沒有回 200：玩聚代管要 /healthz 與 /readyz 都回 200 才算部署完成")
			}
		}
	}
	return fmt.Errorf("遊戲伺服器 30 秒內 %s/healthz 沒有回 200：玩聚代管的伺服器要監聽 ADDR（或 PORT），並提供 GET /healthz、/readyz", serverURL)
}

// localOrigin 檢查 -server-url 是本機 http 位址，回傳來源（scheme://host:port）。
func localOrigin(s string) (string, error) {
	u, err := url.Parse(s)
	if err != nil || u.Scheme != "http" || (u.Path != "" && u.Path != "/") || u.RawQuery != "" {
		return "", errors.New("要是本機的 http 網址，例如 http://127.0.0.1:9192")
	}
	if h := u.Hostname(); h != "127.0.0.1" && h != "localhost" && h != "::1" {
		return "", errors.New("要是本機的 http 網址（127.0.0.1、localhost）")
	}
	return u.Scheme + "://" + u.Host, nil
}
