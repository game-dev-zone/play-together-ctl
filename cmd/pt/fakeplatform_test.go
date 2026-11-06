package main

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/game-dev-zone/play-together-ctl/internal/playtogether"
)

// 本機假平台：簽章錯誤、啟動碼規則、只接受本機的請求。
func TestFakePlatform(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	fp := newFakePlatform(pub)
	srv := httptest.NewServer(fp.handler(nil))
	t.Cleanup(srv.Close)
	c := &playtogether.Client{BaseURL: srv.URL, KeyID: fakeKeyID, PrivateKey: priv}
	ctx := context.Background()
	apiCode := func(err error) string {
		var e *playtogether.APIError
		if errors.As(err, &e) {
			return e.Code
		}
		if err != nil {
			return err.Error()
		}
		return ""
	}

	code, p1, roomID, err := fp.issue("p1")
	must(t, err)
	ex, err := c.ExchangeLaunchCode(ctx, code)
	must(t, err)
	if ex.PlayerID != p1.ID || ex.RoomID != roomID || ex.Epoch != 0 || ex.HostPlayerID != p1.ID {
		t.Fatalf("兌換結果：%+v", ex)
	}
	if _, err := c.ExchangeLaunchCode(ctx, code); apiCode(err) != "invalid_code" {
		t.Fatalf("同一個碼不能用兩次：%v", err)
	}
	// 同一個代號永遠是同一個玩家
	code2, again, _, _ := fp.issue("p1")
	if again.ID != p1.ID {
		t.Fatal("同一個代號要是同一個玩家 ID")
	}
	// 重新開始前發的碼失效
	fp.restart()
	if _, err := c.ExchangeLaunchCode(ctx, code2); apiCode(err) != "invalid_code" {
		t.Fatalf("重新開始前的碼要失效：%v", err)
	}
	// 被踢的人拿不到碼；已經拿到的也不能兌換
	code3, _, _, _ := fp.issue("p2")
	fp.kick("p2")
	if _, _, _, err := fp.issue("p2"); err == nil || err.Error() != "kicked" {
		t.Fatalf("被踢的人不能拿碼：%v", err)
	}
	if _, err := c.ExchangeLaunchCode(ctx, code3); apiCode(err) != "invalid_code" {
		t.Fatalf("被踢的人不能兌換：%v", err)
	}
	m, err := c.GetRoomMembers(ctx, roomID)
	must(t, err)
	if !m.Open || m.Epoch != 1 || len(m.Members) != 1 {
		t.Fatalf("查成員：%+v", m)
	}

	// 簽章錯誤：別的私鑰、別的金鑰 ID、時鐘偏差
	_, other, _ := ed25519.GenerateKey(rand.Reader)
	wrong := &playtogether.Client{BaseURL: srv.URL, KeyID: fakeKeyID, PrivateKey: other}
	if _, err := wrong.GetRoomMembers(ctx, roomID); apiCode(err) != "bad_signature" {
		t.Fatalf("別的私鑰：%v", err)
	}
	typo := &playtogether.Client{BaseURL: srv.URL, KeyID: "typo", PrivateKey: priv}
	if _, err := typo.GetRoomMembers(ctx, roomID); apiCode(err) != "unknown_key" {
		t.Fatalf("別的金鑰 ID：%v", err)
	}
	fp.setSkew(6 * time.Minute)
	if _, err := c.GetRoomMembers(ctx, roomID); apiCode(err) != "clock_skew" {
		t.Fatalf("時鐘偏差：%v", err)
	}
	fp.setSkew(0)
	// 沒有簽章標頭
	res, err := http.Get(srv.URL + "/v1/game/rooms/" + roomID + "/members")
	must(t, err)
	res.Body.Close()
	if res.StatusCode != 401 {
		t.Fatalf("沒有簽章：%d", res.StatusCode)
	}
	// 平台連不到
	fp.setDown(true)
	if _, err := c.GetRoomMembers(ctx, roomID); apiCode(err) != "unavailable" {
		t.Fatalf("平台連不到：%v", err)
	}
	fp.setDown(false)
	// 回報一局：格式、局數、重送
	must(t, c.ReportRoundStart(ctx, roomID, 1, "round-1", []string{p1.ID}))
	must(t, c.ReportRoundStart(ctx, roomID, 1, "round-1", []string{p1.ID}))
	if err := c.ReportRoundStart(ctx, roomID, 0, "round-2", []string{p1.ID}); apiCode(err) != "stale_epoch" {
		t.Fatalf("舊局數：%v", err)
	}
	if err := c.ReportRoundEnd(ctx, roomID, 1, "round-9", playtogether.RoundCompleted); apiCode(err) != "round_not_found" {
		t.Fatalf("沒開始的局：%v", err)
	}
	must(t, c.ReportRoundEnd(ctx, roomID, 1, "round-1", playtogether.RoundCompleted))
	if rs := fp.roundsFor(roomID); len(rs) != 1 || rs[0].Ended != "completed" {
		t.Fatalf("回報紀錄：%+v", rs)
	}
	// 紀錄不含啟動碼
	raw, _ := json.Marshal(fp.state(0))
	if bytes.Contains(raw, []byte(code)) {
		t.Fatal("呼叫紀錄不能包含啟動碼")
	}
}

// 瀏覽器取碼只接受本機網頁；控制面板只接受自己送的請求。
func TestFakePlatformOrigins(t *testing.T) {
	pub, _, _ := ed25519.GenerateKey(rand.Reader)
	fp := newFakePlatform(pub)
	self, srv, err := startFakePlatform(fp, "http://127.0.0.1:18090/")
	must(t, err)
	t.Cleanup(func() { srv.Close() })
	get := func(origin string) (int, string) {
		req, _ := http.NewRequest(http.MethodGet, self+"/dev/launch-code?player=p1", nil)
		if origin != "" {
			req.Header.Set("Origin", origin)
		}
		res, err := http.DefaultClient.Do(req)
		must(t, err)
		defer res.Body.Close()
		return res.StatusCode, res.Header.Get("Access-Control-Allow-Origin")
	}
	if st, acao := get("http://127.0.0.1:18090"); st != 200 || acao != "http://127.0.0.1:18090" {
		t.Fatalf("本機網頁：%d %q", st, acao)
	}
	if st, _ := get("http://localhost:3000"); st != 200 {
		t.Fatalf("localhost：%d", st)
	}
	for _, o := range []string{"", "https://evil.example", "http://127.0.0.1.evil.test", "http://192.168.1.2:18090"} {
		if st, _ := get(o); st != 403 {
			t.Fatalf("其他網站 %s 不能取碼：%d", o, st)
		}
	}
	control := func(origin, ctype string) int {
		req, _ := http.NewRequest(http.MethodPost, self+"/dev/control", strings.NewReader(`{"action":"close"}`))
		req.Header.Set("Content-Type", ctype)
		if origin != "" {
			req.Header.Set("Origin", origin)
		}
		res, err := http.DefaultClient.Do(req)
		must(t, err)
		res.Body.Close()
		return res.StatusCode
	}
	if st := control("http://127.0.0.1:18090", "application/json"); st != 403 {
		t.Fatalf("遊戲頁不能操作面板：%d", st)
	}
	if st := control("", "application/json"); st != 403 {
		t.Fatalf("沒有 Origin：%d", st)
	}
	if st := control(self, "text/plain"); st != 403 {
		t.Fatalf("不是 JSON：%d", st)
	}
	if fp.state(0).Room.Open != true {
		t.Fatal("被擋的請求不能關房")
	}
	// DNS rebinding：別的網域解析到 127.0.0.1 時，Host 不是本機
	req, _ := http.NewRequest(http.MethodGet, self+"/dev/state", nil)
	req.Host = "rebind.evil.test"
	res, err := http.DefaultClient.Do(req)
	must(t, err)
	res.Body.Close()
	if res.StatusCode != 403 {
		t.Fatalf("Host 不是本機要拒絕：%d", res.StatusCode)
	}
	if st := control(self, "application/json"); st != 200 || fp.state(0).Room.Open {
		t.Fatalf("面板自己的請求：%d", st)
	}
}

// pt verify server：Go 與 Node 範本都要全部通過；故意弄壞的伺服器要被抓到。
func TestVerifyServerTemplates(t *testing.T) {
	if testing.Short() {
		t.Skip("要實際跑遊戲伺服器約 30 秒")
	}
	requireEmbeddedSDK(t)
	for _, tmpl := range []string{"web-go", "web-node"} {
		t.Run(tmpl, func(t *testing.T) {
			if tmpl == "web-node" {
				if _, err := exec.LookPath("node"); err != nil {
					t.Skip("沒有 node")
				}
			}
			root := newProject(t, tmpl)
			t.Chdir(root)
			t.Setenv("GOWORK", "off")
			if err := cmdVerifyServer([]string{"-json"}); err != nil {
				t.Fatal(err)
			}
		})
	}
	t.Run("被踢的人沒有斷線", func(t *testing.T) {
		root := newProject(t, "web-go")
		path := filepath.Join(root, "server", "session.go")
		raw, err := os.ReadFile(path)
		must(t, err)
		broken := strings.Replace(string(raw), "h.removeSeat(r, mem.PlayerID)", "_ = mem", 1)
		if broken == string(raw) {
			t.Fatal("找不到要弄壞的地方")
		}
		must(t, os.WriteFile(path, []byte(broken), 0o644))
		t.Chdir(root)
		t.Setenv("GOWORK", "off")
		if err := cmdVerifyServer([]string{"-json"}); err == nil {
			t.Fatal("被踢的人沒有斷線時要失敗")
		}
	})
}
