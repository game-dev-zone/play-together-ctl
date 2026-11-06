package main

// pt verify server：用本機假平台實際跑一遍遊戲伺服器和玩聚平台的接口（只支援 playtogether-managed 範本的伺服器）。
// 檢查加入、重新加入、舊局啟動碼、踢人、重新開始、關房、平台連不到、時鐘偏差、回報一局、跨來源、請求大小、限流與簽章。
// 範本的 /join、/action 是範本自己定的介面；external（自己的伺服器）請用 pt dev -no-server 的控制面板與呼叫紀錄確認。

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

const (
	verifyRoomSync   = time.Second // 測試時讓遊戲伺服器每秒查一次房間
	verifyCORSOrigin = "https://v1.games.verify.test"
)

type verifyResult struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Status string `json:"status"` // pass、fail、skip
	Detail string `json:"detail,omitempty"`
}

type verifier struct {
	fp      *fakePlatform
	base    string // 遊戲伺服器
	results []verifyResult
	jsonOut bool
	legacy  bool // 舊範本（server/session.go）：新契約才有的項目只提醒
}

// strict 是 v0.18 起的 HTTP 契約項目；舊範本的專案標成略過並提示改用 SDK 的遊戲伺服器套件。
func (v *verifier) strict(id, title string, ok bool, failDetail string) {
	if !ok && v.legacy {
		v.add(id, title, "skip", failDetail+"（舊範本；改用 SDK 的遊戲伺服器套件後會通過，見文件站 /docs/game-server#migrate）")
		return
	}
	v.check(id, title, ok, failDetail)
}

func (v *verifier) add(id, title, status, detail string) {
	v.results = append(v.results, verifyResult{id, title, status, detail})
	if !v.jsonOut {
		mark := map[string]string{"pass": "✓", "fail": "✗", "skip": "－"}[status]
		line := fmt.Sprintf("  %s %s", mark, title)
		if detail != "" {
			line += "：" + detail
		}
		fmt.Println(line)
	}
}

func (v *verifier) check(id, title string, ok bool, failDetail string) bool {
	if ok {
		v.add(id, title, "pass", "")
	} else {
		v.add(id, title, "fail", failDetail)
	}
	return ok
}

// post 送 JSON 到遊戲伺服器；ip 放在 X-Forwarded-For（測試時伺服器設 TRUST_PROXY=1，每位玩家一個 IP）。
func (v *verifier) post(path, ip string, body []byte) (int, map[string]any) {
	req, _ := http.NewRequest(http.MethodPost, v.base+path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Forwarded-For", ip)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, map[string]any{"error": err.Error()}
	}
	defer res.Body.Close()
	out := map[string]any{}
	_ = json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(&out)
	return res.StatusCode, out
}

func (v *verifier) postJSON(path, ip string, body any) (int, map[string]any) {
	raw, _ := json.Marshal(body)
	return v.post(path, ip, raw)
}

type verifyPlayer struct {
	code, ip, token, id string
}

// join：向假平台拿啟動碼（像平台外殼），再交給遊戲伺服器的 /join。回傳狀態碼與錯誤代碼。
func (v *verifier) join(p *verifyPlayer) (int, string) {
	code, fpPlayer, _, err := v.fp.issue(p.code)
	if err != nil {
		return 0, "假平台不發啟動碼：" + err.Error()
	}
	return v.joinWith(p, code, fpPlayer.ID)
}

func (v *verifier) joinWith(p *verifyPlayer, code, wantID string) (int, string) {
	st, out := v.postJSON("/join", p.ip, map[string]string{"launchCode": code})
	if st == 200 {
		p.token, _ = out["token"].(string)
		p.id, _ = out["playerId"].(string)
		if wantID != "" && p.id != wantID {
			return st, fmt.Sprintf("玩家 ID 是 %q，應該是平台兌換結果的 %q", p.id, wantID)
		}
		return st, ""
	}
	e, _ := out["error"].(string)
	return st, e
}

func (v *verifier) action(p *verifyPlayer, action string) (int, string) {
	st, out := v.postJSON("/action", p.ip, map[string]string{"token": p.token, "action": action})
	e, _ := out["error"].(string)
	return st, e
}

// until 每 200 毫秒檢查一次，直到成立或逾時。
func until(timeout time.Duration, fn func() bool) bool {
	deadline := time.Now().Add(timeout)
	for {
		if fn() {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func cmdVerify(args []string) error {
	if len(args) == 0 || args[0] != "server" {
		return errors.New("用法：pt verify server [-json]（用本機假平台驗證遊戲伺服器和玩聚平台的接口）")
	}
	return cmdVerifyServer(args[1:])
}

func cmdVerifyServer(args []string) error {
	fs := flag.NewFlagSet("verify server", flag.ExitOnError)
	jsonOut := fs.Bool("json", false, "JSON 輸出")
	startAction := fs.String("start-action", "start", "開始一局的操作（/action 的 action）；改過玩法時用來檢查「回報一局」")
	roundWait := fs.Duration("round-wait", 20*time.Second, "等一局自然結束（回報 completed）最久多久")
	_ = fs.Parse(args)
	p, err := findProject()
	if err != nil {
		return err
	}
	if p == nil {
		return errors.New("找不到 playtogether.json：請在用 pt new 建立的專案裡執行")
	}
	if mode := p.gameServerMode(); mode != defaultGameServer {
		return fmt.Errorf("pt verify server 只支援 %s 範本的伺服器（這個專案是 %s）；自己的遊戲伺服器請用 pt dev -no-server 的控制面板與呼叫紀錄確認",
			defaultGameServer, mode)
	}
	keyFile, pub, _, cleanup, err := devServerKey(p)
	if err != nil {
		return fmt.Errorf("遊戲伺服器金鑰：%w", err)
	}
	defer cleanup()
	fp := newFakePlatform(pub)
	platform, srv, err := startFakePlatform(fp, "")
	if err != nil {
		return err
	}
	defer srv.Close()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	addr := ln.Addr().String()
	ln.Close()
	cmd, tmp, err := verifyServerCmd(p)
	if err != nil {
		return err
	}
	if tmp != "" {
		defer os.RemoveAll(tmp)
	}
	cmd.Dir = p.path("server")
	cmd.Env = append(os.Environ(), "ADDR="+addr, "WEB_DIR="+p.path(p.Web), "ROOM_SYNC="+verifyRoomSync.String(), "SEAT_HOLD=30s",
		"TRUST_PROXY=1", "ALLOW_ORIGINS=https://*.games.verify.test")
	cmd.Env = append(cmd.Env, devServerEnv(platform, keyFile)...)
	var logs lockedBuffer
	cmd.Stdout, cmd.Stderr = &logs, &logs
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("啟動遊戲伺服器失敗：%w", err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	defer func() {
		stopServer(cmd)
		<-done
	}()
	v := &verifier{fp: fp, base: "http://" + addr, jsonOut: *jsonOut, legacy: legacyServerHint(p) != ""}
	up := until(60*time.Second, func() bool {
		select {
		case <-done:
			return true
		default:
		}
		c, err := net.DialTimeout("tcp", addr, 200*time.Millisecond)
		if err == nil {
			c.Close()
		}
		return err == nil
	})
	select {
	case err := <-done:
		done <- err
		return fmt.Errorf("遊戲伺服器結束了：%v\n%s", err, logs.tail())
	default:
	}
	if !up {
		return errors.New("遊戲伺服器 60 秒內沒有啟動\n" + logs.tail())
	}
	if !*jsonOut {
		fmt.Printf("pt verify server：用本機假平台驗證 %s 的遊戲伺服器（%s）\n", p.Name, p.Server)
	}
	if st, _ := v.post("/join", "10.0.0.1", []byte("{}")); st == 404 || st == 405 {
		return errors.New("遊戲伺服器沒有範本的 POST /join：pt verify server 只能檢查 pt new 範本的介面")
	}
	v.run(*startAction, *roundWait)

	pass, fail, skip := 0, 0, 0
	for _, r := range v.results {
		switch r.Status {
		case "pass":
			pass++
		case "fail":
			fail++
		default:
			skip++
		}
	}
	if *jsonOut {
		out, _ := json.MarshalIndent(map[string]any{"ok": fail == 0, "checks": v.results}, "", "  ")
		fmt.Println(string(out))
	} else {
		fmt.Printf("結果：%d 項通過、%d 項失敗、%d 項略過\n", pass, fail, skip)
	}
	if fail > 0 {
		if !*jsonOut {
			fmt.Println("遊戲伺服器最後的輸出：\n" + logs.tail())
		}
		return errors.New("遊戲伺服器的接口檢查沒有通過")
	}
	return nil
}

func verifyServerCmd(p *project) (*exec.Cmd, string, error) {
	switch p.Server {
	case "go":
		tmp, err := os.MkdirTemp("", "pt-verify-")
		if err != nil {
			return nil, "", err
		}
		bin := filepath.Join(tmp, "server")
		if runtime.GOOS == "windows" {
			bin += ".exe"
		}
		build := exec.Command("go", "build", "-o", bin, ".")
		build.Dir = p.path("server")
		if out, err := build.CombinedOutput(); err != nil {
			os.RemoveAll(tmp)
			return nil, "", fmt.Errorf("遊戲伺服器編譯失敗：%w\n%s", err, out)
		}
		return exec.Command(bin), tmp, nil
	case "node":
		return exec.Command("node", "main.mjs"), "", nil
	}
	return nil, "", fmt.Errorf("pt verify server 只支援 go、node 伺服器（目前是 %q）", p.Server)
}

// run 依序跑每個情境。後面的情境依賴前面的狀態，所以前面失敗時後面可能一起失敗。
func (v *verifier) run(startAction string, roundWait time.Duration) {
	fp := v.fp
	wait := 5 * verifyRoomSync

	// 健康檢查
	res, err := http.Get(v.base + "/healthz")
	healthy := false
	if err == nil {
		var out map[string]any
		_ = json.NewDecoder(res.Body).Decode(&out)
		res.Body.Close()
		healthy = res.StatusCode == 200 && out["status"] == "ok"
	}
	v.check("healthz", "健康檢查 GET /healthz 回 200 {\"status\":\"ok\"}", healthy, "沒有回應或格式不對（SDK 的 gameserver 套件／createGameServer 會提供）")

	// 加入與重新加入
	p1 := &verifyPlayer{code: "p1", ip: "10.0.0.1"}
	p2 := &verifyPlayer{code: "p2", ip: "10.0.0.2"}
	st, e := v.join(p1)
	if !v.check("join", "加入：兌換啟動碼後拿到 token，玩家 ID 和平台一致", st == 200 && e == "", fmt.Sprintf("/join 回 %d %s", st, e)) {
		v.lastSigHint()
		return
	}
	if st, e := v.join(p2); st != 200 {
		v.add("join2", "第二位玩家加入", "fail", fmt.Sprintf("/join 回 %d %s", st, e))
		return
	}
	oldToken := p1.token
	st, e = v.join(p1)
	old := &verifyPlayer{ip: p1.ip, token: oldToken}
	oldSt, _ := v.action(old, "state")
	v.check("rejoin", "重新加入：同一位玩家用新的啟動碼回到原座位，舊 token 回 401", st == 200 && e == "" && oldSt == 401,
		fmt.Sprintf("重新加入 %d %s；舊 token %d", st, e, oldSt))

	// 用過的啟動碼：第一次兌換成功，第二次要被拒絕
	code, fpP, _, _ := fp.issue("p1")
	v.joinWith(p1, code, fpP.ID)
	st, e = v.joinWith(&verifyPlayer{code: "p1", ip: p1.ip}, code, "")
	v.check("reuse", "同一個啟動碼不能用兩次（403）", st == 403, fmt.Sprintf("第二次兌換回 %d %s", st, e))

	// 缺少啟動碼
	stBad, outBad := v.post("/join", "10.0.0.8", []byte(`{}`))
	eBad, _ := outBad["error"].(string)
	v.strict("bad-request", "缺少 launchCode 回 400 bad_request", stBad == 400 && eBad == "bad_request", fmt.Sprintf("/join {} 回 %d %s", stBad, eBad))

	// 重新開始：舊局的啟動碼失效，遊戲伺服器結束舊局
	staleCode, _, _, _ := fp.issue("p2")
	fp.restart()
	st, e = v.joinWith(&verifyPlayer{code: "p2", ip: p2.ip}, staleCode, "")
	v.check("stale-code", "重新開始前發的啟動碼被拒絕（403 或 409）", st == 403 || st == 409, fmt.Sprintf("/join 回 %d %s", st, e))
	ended := until(wait, func() bool { s, _ := v.action(p1, "state"); return s == 401 })
	v.check("restart", "房主重新開始後，遊戲伺服器結束舊局（舊 token 回 401）", ended,
		fmt.Sprintf("%s 內舊局的 token 仍然有效：要在查到 epoch 增加時結束房間", wait))

	// 新的一局：回報開始與結束
	for _, p := range []*verifyPlayer{p1, p2} {
		if st, e := v.join(p); st != 200 {
			v.add("join-new-epoch", "重新開始後加入新局", "fail", fmt.Sprintf("%d %s", st, e))
			return
		}
	}
	room := fp.currentRoom()
	v.action(p1, startAction)
	started := until(3*time.Second, func() bool {
		for _, r := range fp.roundsFor(room) {
			if r.Epoch == 1 && len(r.Players) == 2 {
				return true
			}
		}
		return false
	})
	if !started {
		v.add("round-start", "回報一局開始（epoch 與玩家）", "skip",
			fmt.Sprintf("送出 action %q 後沒有收到回報；開局的操作不是 %q 時用 -start-action 指定", startAction, startAction))
	} else {
		v.add("round-start", "回報一局開始（epoch 與玩家）", "pass", "")
		completed := until(roundWait, func() bool {
			v.action(p1, "state")
			v.action(p2, "state")
			for _, r := range fp.roundsFor(room) {
				if r.Ended == "completed" {
					return true
				}
			}
			return false
		})
		if completed {
			v.add("round-end", "一局結束時回報 completed", "pass", "")
		} else {
			v.add("round-end", "一局結束時回報 completed", "skip", fmt.Sprintf("%s 內沒有結束；一局比較長時用 -round-wait 調整", roundWait))
		}
		// 進行中的一局被房主重新開始：回報 aborted
		before := len(fp.roundsFor(room))
		v.action(p1, startAction)
		if until(3*time.Second, func() bool { return len(fp.roundsFor(room)) > before }) {
			fp.restart()
			aborted := until(wait, func() bool {
				for _, r := range fp.roundsFor(room) {
					if r.Ended == "aborted" {
						return true
					}
				}
				return false
			})
			v.check("round-abort", "進行中的一局被重新開始時回報 aborted", aborted, fmt.Sprintf("%s 內沒有收到 aborted", wait))
		} else {
			v.add("round-abort", "進行中的一局被重新開始時回報 aborted", "skip", "第二局沒有開始")
			fp.restart()
		}
	}

	// 踢人
	for _, p := range []*verifyPlayer{p1, p2} {
		until(wait, func() bool { s, _ := v.join(p); return s == 200 })
	}
	fp.kick("p2")
	kicked := until(wait, func() bool { s, _ := v.action(p2, "state"); return s == 401 })
	stillIn, _ := v.action(p1, "state")
	v.check("kick", "房主踢人後，被踢的人的 token 失效，其他人照常", kicked && stillIn == 200,
		fmt.Sprintf("被踢的人%v、其他人 %d（要在查成員看到 kicked 時移出座位）", map[bool]string{true: "已失效", false: "仍然有效"}[kicked], stillIn))

	// 關房
	fp.closeRoom()
	closed := until(wait, func() bool { s, _ := v.action(p1, "state"); return s == 401 })
	v.check("close", "房間關閉後，遊戲伺服器結束房間", closed, fmt.Sprintf("%s 內 token 仍然有效：查到 open 為 false 時要結束", wait))

	// 平台連不到：已在房間的人繼續玩，一段時間後拒絕新加入
	fp.openNewRoom()
	if st, e := v.join(p1); st != 200 {
		v.add("platform-down", "平台連不到時", "fail", fmt.Sprintf("新房間加入失敗：%d %s", st, e))
		return
	}
	downSeq := fp.lastSeq()
	fp.setDown(true)
	keptPlaying := true
	deadline := time.Now().Add(6*verifyRoomSync + 2*time.Second)
	for time.Now().Before(deadline) {
		if s, _ := v.action(p1, "state"); s != 200 {
			keptPlaying = false
		}
		time.Sleep(500 * time.Millisecond)
	}
	p3 := &verifyPlayer{code: "p3", ip: "10.0.0.3"}
	st, e = v.join(p3)
	v.check("platform-down", "平台連不到時，已在房間的人繼續玩、新加入被拒絕（502 或 503）", keptPlaying && (st == 502 || st == 503),
		fmt.Sprintf("已在房間的人%s；新加入回 %d %s", map[bool]string{true: "可以繼續", false: "被中斷"}[keptPlaying], st, e))
	fp.setDown(false)
	time.Sleep(2 * verifyRoomSync) // 等遊戲伺服器下一次查到平台
	if st, e := v.join(p3); st != 200 {
		v.add("platform-up", "平台恢復後可以加入", "fail", fmt.Sprintf("%d %s", st, e))
	}
	upSeq := fp.lastSeq()

	// 時鐘偏差：錯誤要能看出是時鐘問題
	fp.setSkew(6 * time.Minute)
	st, e = v.join(&verifyPlayer{code: "p4", ip: "10.0.0.4"})
	fp.setSkew(0)
	skewSeq := fp.lastSeq()
	v.check("clock-skew", "時鐘偏差超過 5 分鐘時，錯誤訊息帶出 clock_skew", st != 200 && strings.Contains(e, "clock_skew"),
		fmt.Sprintf("/join 回 %d %q（玩家與日誌看不出是時鐘問題；正式主機要開 NTP 校時）", st, e))

	// 跨來源（上傳到玩聚託管後前端和伺服器不同來源）
	preflight := func(origin string) (int, string, string) {
		req, _ := http.NewRequest(http.MethodOptions, v.base+"/join", nil)
		req.Header.Set("Origin", origin)
		req.Header.Set("Access-Control-Request-Method", "POST")
		req.Header.Set("Access-Control-Request-Headers", "content-type")
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			return 0, "", ""
		}
		res.Body.Close()
		return res.StatusCode, res.Header.Get("Access-Control-Allow-Origin"), res.Header.Get("Access-Control-Allow-Headers")
	}
	allowSt, allowed, allowHeaders := preflight(verifyCORSOrigin)
	_, denied, _ := preflight("https://evil.test")
	// 瀏覽器送 JSON 前會先送 OPTIONS：要回 2xx、帶 Allow-Origin，而且允許 Content-Type 標頭
	v.check("cors", "跨來源：允許 ALLOW_ORIGINS 的託管網址（preflight 2xx、允許 Content-Type），拒絕其他來源",
		allowSt >= 200 && allowSt < 300 && allowed == verifyCORSOrigin && strings.Contains(strings.ToLower(allowHeaders), "content-type") && denied == "",
		fmt.Sprintf("允許的來源回 %d %q（Allow-Headers %q）、其他來源回 %q", allowSt, allowed, allowHeaders, denied))

	// 請求太大
	big := append([]byte(`{"token":"`), bytes.Repeat([]byte("x"), 8<<10)...)
	big = append(big, `"}`...)
	st, _ = v.post("/action", "10.0.0.9", big)
	// 中文一個字 3 個位元組：上限要算位元組；有效的 JSON 後面接一大段內容也要擋
	wide, _ := json.Marshal(map[string]string{"token": strings.Repeat("中", 1024)})
	stWide, _ := v.post("/action", "10.0.0.9", wide)
	tail := append([]byte(`{"token":"x"}`), bytes.Repeat([]byte(" x"), 4<<10)...)
	stTail, _ := v.post("/action", "10.0.0.9", tail)
	v.check("too-large", "請求內容太大回 413（依位元組計算、JSON 後面的內容也算）", st == 413 && stWide == 413 && (stTail == 413 || stTail == 400), // JSON 後面接非 JSON 內容：413 或 400 都算擋下
		fmt.Sprintf("英文 8KB 回 %d、中文 3KB 回 %d、JSON 後接 8KB 回 %d", st, stWide, stTail))

	// 限流
	limited := false
	for i := 0; i < 200 && !limited; i++ {
		if s, _ := v.action(p3, "state"); s == 429 {
			limited = true
		}
	}
	v.check("rate-limit", "同一位玩家連續操作會被限流（429）", limited, "連續 200 次都沒有被限流")

	// 查成員的頻率
	poll := fp.state(0).PollSeconds
	v.check("members-poll", "定期查房間成員", poll > 0 && poll <= 3*verifyRoomSync.Seconds(),
		fmt.Sprintf("ROOM_SYNC=%s 時平均間隔 %.1f 秒", verifyRoomSync, poll))

	// 簽章：除了刻意模擬的平台連不到、時鐘偏差，所有呼叫的簽章都要通過
	var bad []string
	for _, c := range fp.callsSince(0) {
		if (c.Seq > downSeq && c.Seq <= upSeq) || (c.Seq > upSeq && c.Seq <= skewSeq) {
			continue
		}
		if c.Status == 401 {
			bad = append(bad, c.Endpoint+"："+c.Result+"（"+c.Hint+"）")
		}
	}
	v.check("signature", "所有請求的簽章都正確", len(bad) == 0, strings.Join(bad, "；"))
}

// lastSigHint 在加入就失敗時，把假平台最後一個錯誤的說明印出來（多半是簽章或金鑰）。
func (v *verifier) lastSigHint() {
	calls := v.fp.callsSince(0)
	for i := len(calls) - 1; i >= 0; i-- {
		if c := calls[i]; c.Status >= 400 {
			detail := c.Result
			if c.Hint != "" {
				detail += "：" + c.Hint
			}
			v.add("platform-error", "假平台最後的錯誤（"+c.Endpoint+"）", "fail", detail)
			return
		}
	}
}

// lockedBuffer 收集遊戲伺服器的輸出（失敗時印最後幾行）。
type lockedBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.buf.Len() > 1<<20 {
		b.buf.Reset()
	}
	return b.buf.Write(p)
}

func (b *lockedBuffer) tail() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	lines := strings.Split(strings.TrimSpace(b.buf.String()), "\n")
	if len(lines) > 20 {
		lines = lines[len(lines)-20:]
	}
	return strings.Join(lines, "\n")
}
