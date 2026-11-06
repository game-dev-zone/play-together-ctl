package main

// 範本伺服器的斷線重連（SDK 的 gameserver 套件／createGameServer）：用假的玩聚平台 API 實際跑 Go 與 Node 版伺服器。

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// stubPlatform 是只回應遊戲伺服器會呼叫的三個 API 的假平台（不驗證簽章）。
type stubPlatform struct {
	mu          sync.Mutex
	epoch       int
	open        bool
	kicked      map[string]bool
	failMembers bool
	rounds      []map[string]any
}

func (f *stubPlatform) set(fn func(*stubPlatform)) { f.mu.Lock(); fn(f); f.mu.Unlock() }

func (f *stubPlatform) handler(t *testing.T) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/launch/exchange", func(w http.ResponseWriter, r *http.Request) {
		var in struct{ Code string }
		_ = json.NewDecoder(r.Body).Decode(&in)
		// 啟動碼格式：<玩家>-<任意>；以 old- 開頭表示舊局數的啟動碼
		f.mu.Lock()
		defer f.mu.Unlock()
		epoch, code := f.epoch, in.Code
		if rest, ok := strings.CutPrefix(code, "old-"); ok {
			epoch, code = f.epoch-1, rest
		}
		player, _, _ := strings.Cut(code, "-")
		if f.kicked["p-"+player] {
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"error":"kicked"}`))
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"gameId": "g", "playerId": "p-" + player, "name": "玩家" + player, "roomId": "r_1", "epoch": epoch, "hostPlayerId": "p-a"})
	})
	mux.HandleFunc("GET /v1/game/rooms/r_1/members", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		if f.failMembers {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":"internal"}`))
			return
		}
		members := []map[string]any{}
		for id, k := range f.kicked {
			members = append(members, map[string]any{"playerId": id, "kicked": k})
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"roomId": "r_1", "open": f.open, "epoch": f.epoch, "hostPlayerId": "p-a", "members": members})
	})
	mux.HandleFunc("POST /v1/game/rounds", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		f.mu.Lock()
		f.rounds = append(f.rounds, body)
		f.mu.Unlock()
		_, _ = w.Write([]byte(`{"ok":true}`))
	})
	return mux
}

func (f *stubPlatform) roundEnded(result string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, r := range f.rounds {
		if r["phase"] == "end" && r["result"] == result {
			return true
		}
	}
	return false
}

func TestReconnectSkeleton(t *testing.T) {
	requireEmbeddedSDK(t)
	for _, tmpl := range []string{"web-go", "web-node"} {
		t.Run(tmpl, func(t *testing.T) {
			if tmpl == "web-node" {
				if _, err := exec.LookPath("node"); err != nil {
					t.Skip("沒有 node")
				}
			}
			testReconnect(t, tmpl)
		})
	}
}

func testReconnect(t *testing.T, tmpl string) {
	root := newProject(t, tmpl)
	f := &stubPlatform{epoch: 1, open: true, kicked: map[string]bool{}}
	platform := httptest.NewServer(f.handler(t))
	t.Cleanup(platform.Close)

	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	keyFile := filepath.Join(t.TempDir(), "server.key")
	must(t, os.WriteFile(keyFile, []byte(base64.StdEncoding.EncodeToString(priv.Seed())+"\n"), 0o600))

	var cmd *exec.Cmd
	if tmpl == "web-go" {
		bin := filepath.Join(t.TempDir(), "server")
		build := exec.Command("go", "build", "-o", bin, ".")
		build.Dir = filepath.Join(root, "server")
		build.Env = append(os.Environ(), "GOWORK=off", "GOFLAGS=-mod=mod", "GOPROXY=off")
		if out, err := build.CombinedOutput(); err != nil {
			t.Fatalf("編譯失敗：%v\n%s", err, out)
		}
		gofmt := exec.Command("gofmt", "-l", ".")
		gofmt.Dir = filepath.Join(root, "server")
		if out, err := gofmt.Output(); err != nil || len(out) > 0 {
			t.Fatalf("範本產生的 Go 程式要符合 gofmt：%v %s", err, out)
		}
		cmd = exec.Command(bin)
	} else {
		cmd = exec.Command("node", "main.mjs")
	}
	addr := freeAddr(t)
	cmd.Dir = filepath.Join(root, "server")
	cmd.Env = append(os.Environ(), "PT_MOCK=", "PT_API_URL="+platform.URL, "PT_KEY_ID=k1", "PT_KEY_FILE="+keyFile,
		"ADDR="+addr, "WEB_DIR="+filepath.Join(root, "web"), "ROOM_SYNC=200ms", "SEAT_HOLD=1s")
	var logs bytes.Buffer
	cmd.Stdout, cmd.Stderr = &logs, &logs
	must(t, cmd.Start())
	t.Cleanup(func() { _ = cmd.Process.Kill(); _ = cmd.Wait() })
	base := "http://" + addr
	for i := 0; ; i++ {
		if res, err := http.Get(base + "/"); err == nil {
			res.Body.Close()
			break
		}
		if i > 300 {
			t.Fatalf("伺服器沒有啟動：%s", logs.String())
		}
		time.Sleep(100 * time.Millisecond)
	}

	type view struct {
		Error   string
		Players []struct {
			ID     string
			Online bool
		}
		Playing bool
	}
	join := func(code string) (int, string) {
		var out struct {
			Token string
			Error string
			State view
		}
		st := postJSON(t, base+"/join", map[string]string{"launchCode": code}, &out)
		return st, out.Token
	}
	act := func(token, action string) (int, view) {
		var v view
		st := postJSON(t, base+"/action", map[string]string{"token": token, "action": action}, &v)
		return st, v
	}
	// eventually：條件在 3 秒內成立；期間 keep 的 token 持續輪詢（保持在線）
	eventually := func(what string, keep []string, cond func() bool) {
		t.Helper()
		for i := 0; i < 60; i++ {
			for _, k := range keep {
				act(k, "state")
			}
			if cond() {
				return
			}
			time.Sleep(50 * time.Millisecond)
		}
		t.Fatalf("%s：3 秒內沒有發生\n%s", what, logs.String())
	}

	// 1. 兩位玩家加入
	st, a1 := join("a-0000000000000001")
	if st != 200 || a1 == "" {
		t.Fatalf("A 加入失敗：%d\n%s", st, logs.String())
	}
	_, b := join("b-0000000000000001")
	if st, v := act(a1, "state"); st != 200 || len(v.Players) != 2 {
		t.Fatalf("房間要有兩位玩家：%d %+v", st, v)
	}

	// 2. A 重新整理：新的啟動碼回到原座位，舊 token 立即失效
	st, a2 := join("a-0000000000000002")
	if st != 200 || a2 == a1 {
		t.Fatalf("重新加入要拿到新 token：%d", st)
	}
	if st, v := act(a1, "state"); st != 401 || v.Error != "session_expired" {
		t.Fatalf("舊 token 要回 401 session_expired：%d %+v", st, v)
	}
	if st, v := act(a2, "state"); st != 200 || len(v.Players) != 2 {
		t.Fatalf("重新加入後座位不變：%d %+v", st, v)
	}

	// 3. 平台踢掉 B：下一次查房間後 B 的 token 失效、座位移除
	f.set(func(f *stubPlatform) { f.kicked["p-b"] = true })
	eventually("被踢的人失效", []string{a2}, func() bool { st, _ := act(b, "state"); return st == 401 })
	if _, v := act(a2, "state"); len(v.Players) != 1 {
		t.Fatalf("被踢的人要移出座位：%+v", v)
	}
	if st, _ := join("b-0000000000000002"); st != 403 {
		t.Fatalf("被踢的人兌換要被拒絕：%d", st)
	}

	// 4. 一局進行中，房主重新開始（局數加一）：房間結束、回報中止、舊 token 失效
	if _, v := act(a2, "start"); !v.Playing {
		t.Fatalf("要開始一局：%+v", v)
	}
	f.set(func(f *stubPlatform) { f.epoch = 2 })
	eventually("重新開始後舊局結束", nil, func() bool { st, _ := act(a2, "state"); return st == 401 })
	eventually("回報中止", nil, func() bool { return f.roundEnded("aborted") })
	st, a3 := join("a-0000000000000003")
	if st != 200 {
		t.Fatalf("新局可以加入：%d", st)
	}
	if st, _ := join("old-c-000000000000001"); st != 409 {
		t.Fatalf("舊局數的啟動碼要回 409：%d", st)
	}

	// 5. 不在一局中、離線超過 SEAT_HOLD 的人讓出座位（A 持續輪詢，C 不再出現）
	_, c := join("c-0000000000000001")
	if _, v := act(a3, "state"); len(v.Players) != 2 {
		t.Fatalf("C 要入座：%+v", v)
	}
	eventually("離線太久讓出座位", []string{a3}, func() bool { _, v := act(a3, "state"); return len(v.Players) == 1 })
	if st, _ := act(c, "state"); st != 401 {
		t.Fatalf("讓出座位後 token 失效：%d", st)
	}

	// 6. 一段時間查不到平台：拒絕新加入，已在房間的人繼續玩
	f.set(func(f *stubPlatform) { f.failMembers = true })
	eventually("查不到平台時拒絕新加入", []string{a3}, func() bool { st, _ := join("d-0000000000000001"); return st == 503 })
	if st, _ := act(a3, "state"); st != 200 {
		t.Fatalf("已在房間的人要能繼續：%d", st)
	}
	f.set(func(f *stubPlatform) { f.failMembers = false })
	eventually("平台恢復後可以加入", []string{a3}, func() bool { st, _ := join("d-0000000000000002"); return st == 200 })

	// 7. 房間關閉：所有連線失效
	f.set(func(f *stubPlatform) { f.open = false })
	eventually("關房後連線失效", nil, func() bool { st, _ := act(a3, "state"); return st == 401 })
}
