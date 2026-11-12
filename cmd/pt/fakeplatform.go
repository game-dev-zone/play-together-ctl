package main

// 本機假平台（pt dev、pt verify server 共用）：實作遊戲伺服器會呼叫的玩聚平台 API，**驗證簽章**，
// 並模擬房主與平台的狀態（踢人、重新開始、關房、平台暫時連不到、時鐘偏差）。
//
// - 只綁 127.0.0.1；啟動碼用 dev- 前綴，真的玩聚平台不會發這種格式。
// - 規則和平台一致：啟動碼 60 秒、只能用一次、重新開始前發的碼失效、被踢的人拿不到碼也兌換不了。
// - 呼叫紀錄只記端點、結果與原因，不記請求內容、標頭與啟動碼。

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/game-dev-zone/play-together-ctl/internal/playtogether"
)

const (
	fakeGameID          = "local-dev-game"
	fakeKeyID           = "local-dev"
	fakeCodeTTL         = 60 * time.Second
	fakeMaxBody         = 64 << 10
	fakeMaxCalls        = 300
	fakeMaxRoundPlayers = 100
)

var (
	fakeRoomIDPattern   = regexp.MustCompile(`^r_[A-Za-z0-9_-]{22}$`)
	fakePlayerIDPattern = regexp.MustCompile(`^p_[A-Za-z0-9_-]{22}$`)
	fakeRoundKeyPattern = regexp.MustCompile(`^[A-Za-z0-9_.:-]{1,64}$`)
	fakePlayerCode      = regexp.MustCompile(`^[A-Za-z0-9_-]{1,32}$`)
)

// 簽章錯誤代碼 → 給開發者看的說明
var fakeSigHints = map[string]string{
	"missing_headers": "沒有簽章標頭：請用玩聚伺服器 SDK 送出請求",
	"bad_timestamp":   "簽章的時間戳記格式錯誤",
	"clock_skew":      "遊戲伺服器的時鐘和平台差超過 5 分鐘：正式主機要開啟 NTP 校時",
	"unknown_key":     "金鑰 ID（PT_KEY_ID）不是登記的金鑰",
	"bad_signature":   "簽章不符：私鑰和登記的公鑰不同，或簽章的內容（方法、路徑、查詢字串、請求內容）和送出的不一致",
	"replay":          "nonce 重複使用：每個請求都要用新的 nonce",
}

type fakePlatform struct {
	mu       sync.Mutex
	pub      ed25519.PublicKey
	nonces   map[string]time.Time
	skew     time.Duration // 模擬平台時鐘和遊戲伺服器的差距
	down     bool          // 模擬平台暫時連不到：遊戲伺服器的 API 回 503
	disabled bool          // 模擬遊戲被平台停用：GET /v1/game/status 回 serving:false，其他 API 和真的平台一樣回 401 unknown_key
	shell    shellConfig   // playtogether.json 的外殼樣式（控制面板的「用平台外殼開啟」預覽）
	// speechFrom 是下載本機語音辨識引擎的玩聚網站（devspeech.go）；空字串時預覽頁的麥克風辨識不能用
	speechFrom string
	rooms      map[string]*fakeRoom
	room       *fakeRoom // 目前的房間
	players    map[string]*fakePlayer
	order      []string // 玩家代號，依加入順序
	codes      map[string]*fakeCode
	rounds     map[string]*fakeRound
	calls      []fakeCall
	seq        int
	polls      map[string][]time.Time // 房號 → 最近幾次查成員的時間
	now        func() time.Time
	ai         http.Handler           // 本機 AI 閘道（ai_services.go）；nil 時 /v1/ai/ 回 404
	speech     map[string]*fakeSpeech // 玩家代號 → 語音指令的狀態（fakespeech.go）
	// serverURL：代管專案在本機的遊戲伺服器網址；發啟動碼時一起給畫面（SDK 放進 launch.serverUrl），和正式平台一樣
	serverURL string
	// 點數下注（fakepoints.go）：pointsStake 是專案有沒有宣告 economy.platformPointsStake；points 用到時才建立
	pointsStake bool
	points      *fakePoints
}

type fakeRoom struct {
	ID     string
	Epoch  int
	Open   bool
	Host   string          // 房主的玩家代號
	Joined map[string]bool // 兌換過啟動碼的玩家代號
	Kicked map[string]bool
	// 音高授權（pitch.policy）最後一次的序號與內容；序號只能遞增，同序號同內容視為重送
	PitchSeq  int64
	PitchBody string
	HasPitch  bool
	// 語音指令授權（speech.policy）：最後一次的內容（fakespeech.go）
	Speech      *playtogether.SpeechPolicy
	SpeechBody  string
	speechMatch map[string]speechMatch // 詞彙表展開後的對照，用到時才建立
}

type fakePlayer struct{ Code, ID, Name string }

type fakeCode struct {
	player, roomID string
	epoch          int
	expires        time.Time
	used           bool
}

type fakeRound struct {
	Epoch   int
	Players []string
	Ended   string
}

type fakeCall struct {
	Seq      int       `json:"seq"`
	At       time.Time `json:"at"`
	Endpoint string    `json:"endpoint"`
	Status   int       `json:"status"`
	Result   string    `json:"result"`         // 成功時的摘要或錯誤代碼
	Hint     string    `json:"hint,omitempty"` // 錯誤的說明
	Signed   bool      `json:"signed"`         // 簽章是否通過
}

func newFakePlatform(pub ed25519.PublicKey) *fakePlatform {
	f := &fakePlatform{pub: pub, nonces: map[string]time.Time{}, rooms: map[string]*fakeRoom{}, players: map[string]*fakePlayer{},
		codes: map[string]*fakeCode{}, rounds: map[string]*fakeRound{}, polls: map[string][]time.Time{}, now: time.Now}
	f.newRoomLocked()
	return f
}

func fakeRandom(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func (f *fakePlatform) newRoomLocked() {
	r := &fakeRoom{ID: "r_" + fakeRandom(16)[:22], Open: true, Joined: map[string]bool{}, Kicked: map[string]bool{}}
	f.rooms[r.ID] = r
	f.room = r
}

// playerLocked 依代號找玩家，沒有就建立（同一個代號永遠是同一個玩家 ID，像平台的假名 ID）。
func (f *fakePlatform) playerLocked(code string) *fakePlayer {
	if p := f.players[code]; p != nil {
		return p
	}
	sum := sha256.Sum256([]byte("player-v1|" + fakeGameID + "|" + code))
	p := &fakePlayer{Code: code, ID: "p_" + base64.RawURLEncoding.EncodeToString(sum[:16]), Name: "測試玩家-" + strings.ToUpper(code)}
	f.players[code] = p
	f.order = append(f.order, code)
	return p
}

func (f *fakePlatform) hostIDLocked(r *fakeRoom) string {
	if p := f.players[r.Host]; p != nil {
		return p.ID
	}
	return ""
}

func (f *fakePlatform) playerByID(id string) *fakePlayer {
	for _, p := range f.players {
		if p.ID == id {
			return p
		}
	}
	return nil
}

func (f *fakePlatform) logLocked(endpoint string, status int, result string, signed bool) {
	f.seq++
	f.calls = append(f.calls, fakeCall{Seq: f.seq, At: f.now(), Endpoint: endpoint, Status: status, Result: result, Hint: fakeSigHints[result], Signed: signed})
	if len(f.calls) > fakeMaxCalls {
		f.calls = f.calls[len(f.calls)-fakeMaxCalls:]
	}
}

// issue 發一個啟動碼（像平台外殼在玩家進房時申請）。
func (f *fakePlatform) issue(player string) (code string, p *fakePlayer, roomID string, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if !fakePlayerCode.MatchString(player) {
		return "", nil, "", errors.New("bad_player")
	}
	r := f.room
	if !r.Open {
		return "", nil, "", errors.New("room_closed")
	}
	if r.Kicked[player] {
		return "", nil, "", errors.New("kicked")
	}
	p = f.playerLocked(player)
	if r.Host == "" {
		r.Host = player
	}
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	code = "dev-" + hex.EncodeToString(b)
	f.codes[code] = &fakeCode{player: player, roomID: r.ID, epoch: r.Epoch, expires: f.now().Add(fakeCodeTTL)}
	return code, p, r.ID, nil
}

// 房主與平台的控制（pt dev 的面板、pt verify server 都用這些）
func (f *fakePlatform) kick(player string) {
	f.mu.Lock()
	f.room.Kicked[player] = true
	f.mu.Unlock()
}

func (f *fakePlatform) unkick(player string) {
	f.mu.Lock()
	delete(f.room.Kicked, player)
	f.mu.Unlock()
}

func (f *fakePlatform) restart() {
	f.mu.Lock()
	f.room.Epoch++
	f.mu.Unlock()
}

func (f *fakePlatform) closeRoom() {
	f.mu.Lock()
	f.room.Open = false
	f.mu.Unlock()
}

func (f *fakePlatform) openNewRoom() {
	f.mu.Lock()
	f.newRoomLocked()
	f.mu.Unlock()
}

// calledEndpoint：遊戲伺服器有沒有呼叫過這個 API（判斷遊戲伺服器套件的版本）。
func (f *fakePlatform) calledEndpoint(endpoint string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, c := range f.calls {
		if c.Endpoint == endpoint {
			return true
		}
	}
	return false
}

func (f *fakePlatform) setDisabled(disabled bool) {
	f.mu.Lock()
	f.disabled = disabled
	f.mu.Unlock()
}

const statusEndpoint = "查遊戲狀態"

// gameStatus：遊戲伺服器定期查詢遊戲狀態。停用時仍驗證得過、回 serving:false（和真的平台一樣）。
func (f *fakePlatform) gameStatus(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.verifyLocked(w, r, statusEndpoint); !ok {
		return
	}
	out := map[string]any{"gameId": "dev-game", "status": "draft", "serving": true, "reason": "", "maintenance": map[string]any{"game": false, "site": ""}}
	result := "服務中"
	if f.disabled {
		out["status"], out["serving"], out["reason"] = "disabled", false, "game_disabled"
		result = "已停用（serving: false）"
	}
	f.logLocked(statusEndpoint, 200, result, true)
	f.write(w, 200, out)
}

func (f *fakePlatform) setDown(down bool) {
	f.mu.Lock()
	f.down = down
	f.mu.Unlock()
}

func (f *fakePlatform) setSkew(d time.Duration) {
	f.mu.Lock()
	f.skew = d
	f.mu.Unlock()
}

// ---------- 遊戲伺服器呼叫的 API ----------

func (f *fakePlatform) handler(panel http.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/launch/exchange", f.exchange)
	mux.HandleFunc("GET /v1/game/rooms/{id}/members", f.members)
	mux.HandleFunc("POST /v1/game/rounds", f.roundReport)
	mux.HandleFunc("POST /v1/voice/policy", f.voicePolicy)
	mux.HandleFunc("POST /v1/pitch/policy", f.pitchPolicy)
	mux.HandleFunc("POST /v1/speech/policy", f.speechPolicy)
	mux.HandleFunc("GET /dev/speech", f.devSpeech)
	mux.HandleFunc("OPTIONS /dev/speech", f.devSpeech)
	mux.HandleFunc("GET /v1/game/status", f.gameStatus)
	mux.HandleFunc("GET /v1/game/points/balance", f.pointsHandler("點數餘額", f.pointsBalance))
	mux.HandleFunc("POST /v1/game/points/settle", f.pointsHandler("點數一次結算", func(_ *http.Request, b []byte) (map[string]any, string, error) { return f.placeBet(b, "settle") }))
	mux.HandleFunc("POST /v1/game/points/stake", f.pointsHandler("點數先扣", func(_ *http.Request, b []byte) (map[string]any, string, error) { return f.placeBet(b, "staged") }))
	mux.HandleFunc("POST /v1/game/points/payoff", f.pointsHandler("點數派彩", f.pointsPayoff))
	mux.HandleFunc("POST /v1/game/points/cancel", f.pointsHandler("點數取消", f.pointsCancel))
	mux.HandleFunc("POST /v1/game/points/table/buyin", f.pointsHandler("牌桌轉入", f.pointsBuyIn))
	mux.HandleFunc("POST /v1/game/points/table/cashout", f.pointsHandler("牌桌轉出", f.pointsCashOut))
	mux.HandleFunc("GET /dev/launch-code", f.launchCode)
	mux.HandleFunc("OPTIONS /dev/launch-code", f.launchCode)
	if panel != nil {
		mux.Handle("/", panel)
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// AI 閘道不經過 ServeMux（ServeMux 會把 ..、// 清理後轉址）：原始的請求目標由 aigateway.Parse 檢查，和平台相同
		if strings.HasPrefix(r.RequestURI, "/v1/ai/") || strings.HasPrefix(r.URL.Path, "/v1/ai/") {
			if f.ai == nil {
				http.NotFound(w, r)
				return
			}
			f.ai.ServeHTTP(w, r)
			return
		}
		mux.ServeHTTP(w, r)
	})
}

// logAI 把本機 AI 閘道的呼叫記到控制面板（不記錄內容與金鑰）。
func (f *fakePlatform) logAI(endpoint string, status int, result, hint string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.logLocked(endpoint, status, result, true)
	if hint != "" {
		f.calls[len(f.calls)-1].Hint = hint
	}
}

func (f *fakePlatform) write(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// verify 驗證簽章；失敗時寫回應與紀錄。成功回傳原始內容，呼叫者要持有 f.mu。
func (f *fakePlatform) verifyLocked(w http.ResponseWriter, r *http.Request, endpoint string) ([]byte, bool) {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, fakeMaxBody))
	if err != nil {
		f.logLocked(endpoint, 413, "too_large", false)
		f.write(w, 413, map[string]string{"error": "too_large"})
		return nil, false
	}
	if f.down {
		f.logLocked(endpoint, 503, "unavailable（模擬平台連不到）", false)
		f.write(w, 503, map[string]string{"error": "unavailable"})
		return nil, false
	}
	lookup := func(_ context.Context, keyID string) (ed25519.PublicKey, error) {
		if keyID != fakeKeyID {
			return nil, nil
		}
		return f.pub, nil
	}
	req := playtogether.SignableRequest{Method: r.Method, Path: r.URL.Path, Query: r.URL.Query(), Body: body}
	if _, err := playtogether.VerifyRequest(r.Context(), req, r.Header, lookup, fakeNonces{f}, f.now().Add(f.skew)); err != nil {
		code := "internal"
		for _, known := range []error{playtogether.ErrMissingHeaders, playtogether.ErrBadTimestamp, playtogether.ErrClockSkew,
			playtogether.ErrUnknownKey, playtogether.ErrBadSignature, playtogether.ErrReplay} {
			if errors.Is(err, known) {
				code = known.Error()
			}
		}
		f.logLocked(endpoint, 401, code, false)
		f.write(w, 401, map[string]string{"error": code})
		return nil, false
	}
	if f.disabled && endpoint != statusEndpoint {
		// 真的平台：停用遊戲的金鑰在其他 API 都無效
		f.logLocked(endpoint, 401, "unknown_key（遊戲已停用）", false)
		f.write(w, 401, map[string]string{"error": "unknown_key"})
		return nil, false
	}
	return body, true
}

// fakeNonces 是 VerifyRequest 用的 nonce 紀錄（呼叫時已持有 f.mu）。
type fakeNonces struct{ f *fakePlatform }

func (n fakeNonces) UseOnce(_ context.Context, keyID, nonce string, ttl time.Duration) (bool, error) {
	now := n.f.now()
	for k, exp := range n.f.nonces {
		if now.After(exp) {
			delete(n.f.nonces, k)
		}
	}
	k := keyID + "|" + nonce
	if _, used := n.f.nonces[k]; used {
		return false, nil
	}
	n.f.nonces[k] = now.Add(ttl)
	return true, nil
}

func (f *fakePlatform) exchange(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	const ep = "兌換啟動碼"
	body, ok := f.verifyLocked(w, r, ep)
	if !ok {
		return
	}
	var in struct {
		Code string `json:"code"`
	}
	if json.Unmarshal(body, &in) != nil {
		f.logLocked(ep, 400, "bad_request", true)
		f.write(w, 400, map[string]string{"error": "bad_request"})
		return
	}
	c := f.codes[in.Code]
	reason := ""
	switch {
	case c == nil:
		reason = "不是這個假平台發的啟動碼"
	case c.used:
		reason = "啟動碼已經用過（每個啟動碼只能兌換一次；重新加入要用新的啟動碼）"
	case f.now().After(c.expires):
		reason = "啟動碼已過期（60 秒）"
	case !f.rooms[c.roomID].Open:
		reason = "房間已關閉"
	case f.rooms[c.roomID].Epoch != c.epoch:
		reason = "房主重新開始前發的啟動碼"
	case f.rooms[c.roomID].Kicked[c.player]:
		reason = "玩家已被房主踢出"
	}
	if reason != "" {
		f.logLocked(ep, 409, "invalid_code（"+reason+"）", true)
		f.write(w, 409, map[string]string{"error": "invalid_code"})
		return
	}
	c.used = true
	room := f.rooms[c.roomID]
	room.Joined[c.player] = true
	p := f.players[c.player]
	f.logLocked(ep, 200, fmt.Sprintf("%s 加入（第 %d 局）", p.Name, room.Epoch), true)
	f.write(w, 200, map[string]any{"gameId": fakeGameID, "playerId": p.ID, "name": p.Name, "roomId": room.ID,
		"epoch": room.Epoch, "hostPlayerId": f.hostIDLocked(room)})
}

func (f *fakePlatform) members(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	const ep = "查房間成員"
	if _, ok := f.verifyLocked(w, r, ep); !ok {
		return
	}
	room := f.rooms[r.PathValue("id")]
	if room == nil {
		f.logLocked(ep, 404, "room_not_found", true)
		f.write(w, 404, map[string]string{"error": "room_not_found"})
		return
	}
	ps := append(f.polls[room.ID], f.now())
	if len(ps) > 6 {
		ps = ps[len(ps)-6:]
	}
	f.polls[room.ID] = ps
	list := []map[string]any{}
	kicked := 0
	for _, code := range f.order {
		if room.Joined[code] {
			list = append(list, map[string]any{"playerId": f.players[code].ID, "kicked": room.Kicked[code]})
			if room.Kicked[code] {
				kicked++
			}
		}
	}
	f.logLocked(ep, 200, fmt.Sprintf("%s、第 %d 局、%d 人（%d 人被踢）", map[bool]string{true: "開放中", false: "已關閉"}[room.Open], room.Epoch, len(list), kicked), true)
	f.write(w, 200, map[string]any{"roomId": room.ID, "open": room.Open, "epoch": room.Epoch, "members": list,
		"hostPlayerId": f.hostIDLocked(room)})
}

func (f *fakePlatform) roundReport(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	const ep = "回報一局"
	body, ok := f.verifyLocked(w, r, ep)
	if !ok {
		return
	}
	var in struct {
		RoomID    string   `json:"roomId"`
		Epoch     *int     `json:"epoch"`
		RoundID   string   `json:"roundId"`
		Phase     string   `json:"phase"`
		PlayerIDs []string `json:"playerIds"`
		Result    string   `json:"result"`
	}
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()
	valid := dec.Decode(&in) == nil && fakeRoomIDPattern.MatchString(in.RoomID) && fakeRoundKeyPattern.MatchString(in.RoundID) && in.Epoch != nil && *in.Epoch >= 0
	switch {
	case !valid:
	case in.Phase == "start":
		valid = len(in.PlayerIDs) > 0 && len(in.PlayerIDs) <= fakeMaxRoundPlayers && in.Result == "" &&
			!slices.ContainsFunc(in.PlayerIDs, func(id string) bool { return !fakePlayerIDPattern.MatchString(id) })
	case in.Phase == "end":
		valid = len(in.PlayerIDs) == 0 && (in.Result == "completed" || in.Result == "aborted")
	default:
		valid = false
	}
	if !valid {
		f.logLocked(ep, 400, "bad_request（格式不符：roomId、roundId、epoch、phase、playerIds、result）", true)
		f.write(w, 400, map[string]string{"error": "bad_request"})
		return
	}
	room := f.rooms[in.RoomID]
	if room == nil {
		f.logLocked(ep, 404, "room_not_found", true)
		f.write(w, 404, map[string]string{"error": "room_not_found"})
		return
	}
	key := in.RoomID + "|" + in.RoundID
	rd := f.rounds[key]
	if in.Phase == "end" {
		if rd == nil {
			f.logLocked(ep, 404, "round_not_found（沒有回報過這一局的開始）", true)
			f.write(w, 404, map[string]string{"error": "round_not_found"})
			return
		}
		recorded := rd.Ended == ""
		if recorded {
			rd.Ended = in.Result
		}
		f.logLocked(ep, 200, map[bool]string{true: "一局結束：" + in.Result, false: "一局結束（重送，不重複記錄）"}[recorded], true)
		f.write(w, 200, map[string]any{"ok": true, "recorded": recorded})
		return
	}
	if !room.Open {
		f.logLocked(ep, 409, "room_closed", true)
		f.write(w, 409, map[string]string{"error": "room_closed"})
		return
	}
	if *in.Epoch != room.Epoch {
		f.logLocked(ep, 409, fmt.Sprintf("stale_epoch（回報第 %d 局，目前是第 %d 局）", *in.Epoch, room.Epoch), true)
		f.write(w, 409, map[string]string{"error": "stale_epoch"})
		return
	}
	if rd != nil {
		f.logLocked(ep, 200, "一局開始（重送，不重複記錄）", true)
		f.write(w, 200, map[string]any{"ok": true, "recorded": false})
		return
	}
	f.rounds[key] = &fakeRound{Epoch: *in.Epoch, Players: in.PlayerIDs}
	f.logLocked(ep, 200, fmt.Sprintf("一局開始：%d 位玩家", len(in.PlayerIDs)), true)
	f.write(w, 200, map[string]any{"ok": true, "recorded": true})
}

func (f *fakePlatform) voicePolicy(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	const ep = "發言規則"
	body, ok := f.verifyLocked(w, r, ep)
	if !ok {
		return
	}
	var p playtogether.VoicePolicy
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&p); err != nil {
		f.logLocked(ep, 400, "bad_request", true)
		f.write(w, 400, map[string]string{"error": "bad_request"})
		return
	}
	if err := p.Validate(); err != nil {
		f.logLocked(ep, 400, "bad_policy（"+err.Error()+"）", true)
		f.write(w, 400, map[string]string{"error": "bad_policy"})
		return
	}
	if f.rooms[p.RoomID] == nil {
		f.logLocked(ep, 404, "room_not_found", true)
		f.write(w, 404, map[string]string{"error": "room_not_found"})
		return
	}
	groups := ""
	if len(p.Groups) > 0 {
		groups = fmt.Sprintf("、分組 %d 組", len(p.Groups))
	}
	f.logLocked(ep, 200, fmt.Sprintf("%s（序號 %d%s）；本機不實際切換語音", p.Mode, p.Sequence, groups), true)
	f.write(w, 200, map[string]any{"ok": true})
}

// pitchPolicy：音高授權。和正式平台相同的檢查：格式、生效時間在現在附近、最多 1 小時、房間開著、局數是目前這一局、
// 序號遞增（同序號同內容當成重送）。本機不實際分析聲音：畫面端的 SDK 在 pt dev 會送假的音高。
func (f *fakePlatform) pitchPolicy(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	const ep = "音高授權"
	body, ok := f.verifyLocked(w, r, ep)
	if !ok {
		return
	}
	fail := func(code int, errCode, hint string) {
		msg := errCode
		if hint != "" {
			msg += "（" + hint + "）"
		}
		f.logLocked(ep, code, msg, true)
		f.write(w, code, map[string]string{"error": errCode})
	}
	var p playtogether.PitchPolicy
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&p); err != nil {
		fail(400, "bad_policy", "格式不符")
		return
	}
	if err := p.Validate(); err != nil {
		fail(400, "bad_policy", err.Error())
		return
	}
	nowMs := f.now().Add(f.skew).UnixMilli()
	if p.EffectiveAtServerMs < nowMs-10_000 || p.EffectiveAtServerMs > nowMs+30_000 || p.ExpiresAtServerMs-p.EffectiveAtServerMs > time.Hour.Milliseconds() {
		fail(400, "bad_policy_time", "生效時間要在現在前 10 秒到後 30 秒，有效期間最多 1 小時")
		return
	}
	room := f.rooms[p.RoomID]
	if room == nil || !room.Open {
		fail(404, "room_not_found", "")
		return
	}
	if p.Epoch != room.Epoch {
		fail(409, "stale_epoch", fmt.Sprintf("房間目前是第 %d 局；房主重新開始後要用新的局數重新授權", room.Epoch))
		return
	}
	canon, _ := json.Marshal(p)
	if room.HasPitch && p.Sequence <= room.PitchSeq {
		if p.Sequence == room.PitchSeq && string(canon) == room.PitchBody {
			f.logLocked(ep, 200, fmt.Sprintf("序號 %d 重送", p.Sequence), true)
			f.write(w, 200, map[string]any{"ok": true})
			return
		}
		fail(409, "stale_sequence", fmt.Sprintf("序號要大於上一次的 %d", room.PitchSeq))
		return
	}
	room.HasPitch, room.PitchSeq, room.PitchBody = true, p.Sequence, string(canon)
	f.logLocked(ep, 200, fmt.Sprintf("授權 %d 位玩家（序號 %d）；本機的畫面端會收到假的音高", len(p.AllowedPlayerIDs), p.Sequence), true)
	f.write(w, 200, map[string]any{"ok": true})
}

// launchCode：瀏覽器 SDK 的模擬模式向假平台取啟動碼（網址帶 pt_dev、pt_player 時）。只允許本機的網頁呼叫。
func (f *fakePlatform) launchCode(w http.ResponseWriter, r *http.Request) {
	// 瀏覽器 SDK 是跨來源的 fetch，一定帶 Origin；沒有 Origin（例如別的網站放 <img>）或不是本機網頁就拒絕
	o := r.Header.Get("Origin")
	if !loopbackOrigin(o) {
		f.write(w, 403, map[string]string{"error": "origin"})
		return
	}
	w.Header().Set("Access-Control-Allow-Origin", o)
	w.Header().Set("Vary", "Origin")
	if r.Method == http.MethodOptions {
		w.WriteHeader(204)
		return
	}
	f.mu.Lock()
	disabled := f.disabled
	f.mu.Unlock()
	if disabled { // 停用的遊戲：平台不發啟動碼
		f.write(w, 403, map[string]string{"error": "game_disabled"})
		return
	}
	code, p, roomID, err := f.issue(r.URL.Query().Get("player"))
	f.mu.Lock()
	if err != nil {
		f.logLocked("發啟動碼（瀏覽器）", 403, err.Error(), false)
	} else {
		f.logLocked("發啟動碼（瀏覽器）", 200, p.Name, false)
	}
	f.mu.Unlock()
	if err != nil {
		f.write(w, map[bool]int{true: 400, false: 403}[err.Error() == "bad_player"], map[string]string{"error": err.Error()})
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	out := map[string]string{"launchCode": code, "name": p.Name, "roomId": roomID}
	if f.serverURL != "" {
		out["serverUrl"] = f.serverURL
	}
	f.write(w, 200, out)
}

// loopbackOrigin：只接受本機的 http 來源。
func loopbackOrigin(o string) bool {
	u, err := url.Parse(o)
	if err != nil || u.Scheme != "http" {
		return false
	}
	ip := net.ParseIP(u.Hostname())
	return u.Hostname() == "localhost" || (ip != nil && ip.IsLoopback())
}

// ---------- 給面板與 pt verify server 的狀態 ----------

type fakeState struct {
	Room struct {
		ID    string `json:"id"`
		Epoch int    `json:"epoch"`
		Open  bool   `json:"open"`
	} `json:"room"`
	Players []struct {
		Code   string `json:"code"`
		ID     string `json:"id"`
		Name   string `json:"name"`
		Host   bool   `json:"host"`
		Joined bool   `json:"joined"`
		Kicked bool   `json:"kicked"`
		// 語音指令："listening"（畫面正在聽）、"allowed"（已授權、畫面還沒開始聽）或空字串
		Speech string `json:"speech"`
	} `json:"players"`
	Down        bool            `json:"down"`
	Disabled    bool            `json:"disabled"`
	SkewSeconds int             `json:"skewSeconds"`
	PollSeconds float64         `json:"pollSeconds"` // 最近幾次查成員的平均間隔；0 表示還沒有
	Calls       []fakeCall      `json:"calls"`
	Points      fakePointsState `json:"points"` // 點數下注（測試點數，fakepoints.go）
}

func (f *fakePlatform) state(after int) fakeState {
	f.mu.Lock()
	defer f.mu.Unlock()
	var s fakeState
	s.Room.ID, s.Room.Epoch, s.Room.Open = f.room.ID, f.room.Epoch, f.room.Open
	for _, code := range f.order {
		p := f.players[code]
		speech := ""
		if f.speechAllowedLocked(code) {
			speech = "allowed"
			if f.speechListeningLocked(code) {
				speech = "listening"
			}
		}
		s.Players = append(s.Players, struct {
			Code   string `json:"code"`
			ID     string `json:"id"`
			Name   string `json:"name"`
			Host   bool   `json:"host"`
			Joined bool   `json:"joined"`
			Kicked bool   `json:"kicked"`
			Speech string `json:"speech"`
		}{code, p.ID, p.Name, f.room.Host == code, f.room.Joined[code], f.room.Kicked[code], speech})
	}
	s.Down, s.Disabled, s.SkewSeconds = f.down, f.disabled, int(f.skew.Seconds())
	if ps := f.polls[f.room.ID]; len(ps) >= 2 {
		s.PollSeconds = ps[len(ps)-1].Sub(ps[0]).Seconds() / float64(len(ps)-1)
	}
	s.Points = f.pointsStateLocked()
	for _, c := range f.calls {
		if c.Seq > after {
			s.Calls = append(s.Calls, c)
		}
	}
	return s
}

// roundsFor 列出某個房間回報過的局（pt verify server 用）。
func (f *fakePlatform) roundsFor(roomID string) []fakeRound {
	f.mu.Lock()
	defer f.mu.Unlock()
	keys := []string{}
	for k := range f.rounds {
		if strings.HasPrefix(k, roomID+"|") {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	out := []fakeRound{}
	for _, k := range keys {
		out = append(out, *f.rounds[k])
	}
	return out
}

// callsSince 列出某個序號之後的呼叫（pt verify server 用）。
func (f *fakePlatform) callsSince(seq int) []fakeCall {
	return f.state(seq).Calls
}

func (f *fakePlatform) lastSeq() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.seq
}

func (f *fakePlatform) currentRoom() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.room.ID
}
