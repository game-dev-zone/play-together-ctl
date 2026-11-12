package main

// pt dev 本機假平台的點數下注與派彩（只用測試點數）：遊戲伺服器呼叫的 /v1/game/points/…，和玩聚平台相同的路徑、簽章、
// 回應欄位、錯誤代碼與上限。依玩法選流程：
//
//	即開型（老虎機）       settle：一次送下注額與派彩
//	有等待時間的局          stake → payoff 或 cancel
//	坐桌型（玩家之間）      table/buyin → table/cashout：同一張桌的轉出總和不超過轉入總和（table_overdraw）
//	機台型（捕魚機）        table/buyin {house: true} → table/cashout：轉出可以多於轉入（一次最多淨贏 1,000，算進每日派彩上限）
//
// 本機的簡化（和玩聚平台不同的地方）：
//   - 點數只存在記憶體，pt dev 結束就清掉；每位玩家第一次用到時有 fakePointsStart 點，控制面板可以再發。
//   - 不做 6 小時逾時（bet_expired、session_expired、平台自動取消與退回）：本機測試很少開這麼久；要測逾時請到玩聚的預覽環境。
//   - 沒有帳號凍結、停權（purchases_suspended）與預覽房間（preview_room）。
//   - 專案沒有宣告 economy.platformPointsStake 時，所有點數 API 一律回 403 declaration_required（平台上沒有宣告就不會有注單）。
//   - 平台核准：控制面板可以切成「未核准」（points_not_approved）或「停權」（points_suspended）；只擋新的操作（查餘額、結算、
//     先扣、轉入），已經成立的注單與入座仍然可以派彩、取消、轉出（結算一定要完成）；重送已經處理過的請求照樣回傳原結果。

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"time"
)

const (
	fakeMaxStake       = 100  // 單筆下注上限
	fakeMaxPayout      = 1000 // 單筆派彩上限（含本金）
	fakeMaxBuyIn       = 300  // 單次入座上限
	fakeMaxPayoutDaily = 5000 // 每位玩家在這款遊戲每天最多拿到的派彩（機台型算淨贏）
	fakeMaxHouseWin    = 1000 // 機台型：一次入座最多淨贏
	fakeDailyStakeCap  = 300  // 每天的下注與入座合計（平台固定）
	fakeResultMaxLen   = 500
	fakePointsStart    = 1000 // 每位玩家一開始的測試點數
	fakePointsGrant    = 500  // 控制面板「發點數」一次的數量
	fakeMaxPointTxs    = 200
)

var (
	fakeGameRefPattern = regexp.MustCompile(`^[A-Za-z0-9._:-]{1,100}$`)
	fakeTaipei         = time.FixedZone("Asia/Taipei", 8*3600)
	// 平台核准的模擬：approved（核准）、pending（未核准）、suspended（停權）
	fakePointsApprovals = map[string]bool{"approved": true, "pending": true, "suspended": true}
)

type fakePoints struct {
	approval string
	balance  map[string]int64 // 玩家代號 → 測試點數
	disabled map[string]bool  // 玩家代號 → 沒有開啟點數玩法
	bets     map[string]*fakeBet
	seats    map[string]*fakeSeat
	txs      []fakePointTx
	txSeq    int
}

type fakeBet struct {
	ID, RoundID, RoomID, Player, Mode, Status string // Mode：settle、staged；Status：staked、settled、cancelled
	Stake                                     int64
	Payout                                    int64
	Hash, PayoutHash                          string
	CreatedAt, SettledAt                      time.Time
	StakeBefore, StakeAfter                   int64
	PayoutBefore, PayoutAfter                 int64
	CancelBefore, CancelAfter                 int64
}

type fakeSeat struct {
	ID, TableID, RoomID, Player, Status string // Status：seated、left
	House                               bool
	BuyIn                               int64
	CashOut                             *int64
	Hash, CashOutHash                   string
	CreatedAt, ClosedAt                 time.Time
	BuyInBefore, BuyInAfter             int64
	CloseBefore, CloseAfter             int64
}

// fakePointTx 是控制面板的點數紀錄。
type fakePointTx struct {
	Seq     int       `json:"seq"`
	At      time.Time `json:"at"`
	Player  string    `json:"player"` // 玩家名稱
	Kind    string    `json:"kind"`   // 下注、派彩、取消、轉入、轉出、發點數
	Ref     string    `json:"ref"`    // betId、sessionId
	Amount  int64     `json:"amount"` // 對玩家的增減
	Balance int64     `json:"balance"`
}

type fakePointsErr struct {
	status     int
	code, hint string
}

func (e *fakePointsErr) Error() string { return e.code }

func pointsErr(status int, code, hint string) *fakePointsErr {
	return &fakePointsErr{status, code, hint}
}

func (f *fakePlatform) pointsLocked() *fakePoints {
	if f.points == nil {
		f.points = &fakePoints{approval: "approved", balance: map[string]int64{}, disabled: map[string]bool{}, bets: map[string]*fakeBet{}, seats: map[string]*fakeSeat{}}
	}
	return f.points
}

func (p *fakePoints) bal(code string) int64 {
	if _, ok := p.balance[code]; !ok {
		p.balance[code] = fakePointsStart
	}
	return p.balance[code]
}

func (f *fakePlatform) addPointTxLocked(code, kind, ref string, amount int64) {
	p := f.pointsLocked()
	p.balance[code] = p.bal(code) + amount
	p.txSeq++
	name := code
	if pl := f.players[code]; pl != nil {
		name = pl.Name
	}
	p.txs = append(p.txs, fakePointTx{Seq: p.txSeq, At: f.now(), Player: name, Kind: kind, Ref: ref, Amount: amount, Balance: p.balance[code]})
	if len(p.txs) > fakeMaxPointTxs {
		p.txs = p.txs[len(p.txs)-fakeMaxPointTxs:]
	}
}

func fakeHash(v map[string]any) string {
	b, _ := json.Marshal(v) // map 的鍵依字母排序，結果固定
	return string(b)
}

func sameTaipeiDay(a, b time.Time) bool {
	return !a.IsZero() && a.In(fakeTaipei).Format(time.DateOnly) == b.In(fakeTaipei).Format(time.DateOnly)
}

// stakeTodayLocked：這位玩家今天的下注與入座（取消的不算；本機不做退回，所以入座都算）。
func (f *fakePlatform) stakeTodayLocked(code string) int64 {
	p, now, sum := f.pointsLocked(), f.now(), int64(0)
	for _, b := range p.bets {
		if b.Player == code && b.Status != "cancelled" && sameTaipeiDay(b.CreatedAt, now) {
			sum += b.Stake
		}
	}
	for _, s := range p.seats {
		if s.Player == code && sameTaipeiDay(s.CreatedAt, now) {
			sum += s.BuyIn
		}
	}
	return sum
}

// payoutTodayLocked：這款遊戲今天派給這位玩家的點數（注單的派彩＋機台型的淨贏）。
func (f *fakePlatform) payoutTodayLocked(code string) int64 {
	p, now, sum := f.pointsLocked(), f.now(), int64(0)
	for _, b := range p.bets {
		if b.Player == code && b.Status == "settled" && sameTaipeiDay(b.SettledAt, now) {
			sum += b.Payout
		}
	}
	for _, s := range p.seats {
		if s.Player == code && s.House && s.Status == "left" && *s.CashOut > s.BuyIn && sameTaipeiDay(s.ClosedAt, now) {
			sum += *s.CashOut - s.BuyIn
		}
	}
	return sum
}

func (f *fakePlatform) stakeLeftLocked(code string) int64 {
	return max(fakeDailyStakeCap-f.stakeTodayLocked(code), 0)
}

// pointsHandler 包住每個點數 API：簽章、宣告檢查、錯誤的回應與紀錄。fn 回傳成功的回應與紀錄摘要。
func (f *fakePlatform) pointsHandler(ep string, fn func(r *http.Request, body []byte) (map[string]any, string, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		body, ok := f.verifyLocked(w, r, ep)
		if !ok {
			return
		}
		var out map[string]any
		var summary string
		var err error
		if !f.pointsStake {
			err = pointsErr(403, "declaration_required", "playtogether.json 的 economy.platformPointsStake 不是 true：只有宣告了經平台接口的點數下注才能用（改了要重新啟動 pt dev）")
		} else {
			out, summary, err = fn(r, body)
		}
		if err != nil {
			e, _ := err.(*fakePointsErr)
			if e == nil {
				e = pointsErr(500, "internal", err.Error())
			}
			msg := e.code
			if e.hint != "" {
				msg += "（" + e.hint + "）"
			}
			f.logLocked(ep, e.status, msg, true)
			f.write(w, e.status, map[string]string{"error": e.code})
			return
		}
		out["testPoints"] = true
		f.logLocked(ep, 200, summary, true)
		f.write(w, 200, out)
	}
}

// pointsGateLocked：平台核准（只檢查新的操作）。
func (f *fakePlatform) pointsGateLocked() error {
	switch f.pointsLocked().approval {
	case "pending":
		return pointsErr(403, "points_not_approved", "這款遊戲的點數功能還沒有核准：正式平台要先在開發者頁申請；本機在控制面板「點數（測試）」切回核准")
	case "suspended":
		return pointsErr(403, "points_suspended", "團隊或遊戲的點數功能被停權：新的下注與入座都會被拒絕，已經成立的仍然可以派彩、取消、轉出")
	}
	return nil
}

// roomForPointsLocked：房間存在（交易外的基本檢查，和平台相同）。
func (f *fakePlatform) roomForPointsLocked(roomID string) (*fakeRoom, error) {
	if !fakeRoomIDPattern.MatchString(roomID) {
		return nil, pointsErr(400, "bad_request", "roomId 格式不符")
	}
	room := f.rooms[roomID]
	if room == nil {
		return nil, pointsErr(404, "room_not_found", "")
	}
	return room, nil
}

// betGuardLocked：新的下注或入座：玩家在房間、房間開著、沒被踢、局數相同。回傳玩家代號。
func (f *fakePlatform) betGuardLocked(room *fakeRoom, playerID string, epoch int32) (string, error) {
	p := f.playerByID(playerID)
	if p == nil || !room.Joined[p.Code] {
		return "", pointsErr(409, "player_not_in_room", "這位玩家沒有兌換過這個房間的啟動碼")
	}
	switch {
	case !room.Open:
		return "", pointsErr(409, "room_closed", "")
	case room.Kicked[p.Code]:
		return "", pointsErr(409, "player_not_in_room", "玩家已被房主踢出")
	case int(epoch) != room.Epoch:
		return "", pointsErr(409, "epoch_mismatch", fmt.Sprintf("房間目前是第 %d 局，請求帶的是第 %d 局", room.Epoch, epoch))
	}
	return p.Code, nil
}

// stakeAllowedLocked：玩家開啟了點數玩法、今天還能下注或入座 amount 點。
func (f *fakePlatform) stakeAllowedLocked(code string, amount int64) error {
	if f.pointsLocked().disabled[code] {
		return pointsErr(409, "points_not_enabled", "玩家沒有開啟點數玩法（控制面板可以切換）")
	}
	if f.stakeLeftLocked(code) < amount {
		return pointsErr(409, "daily_limit", fmt.Sprintf("每天的下注與入座合計最多 %d", fakeDailyStakeCap))
	}
	return nil
}

func (f *fakePlatform) payoutAllowedLocked(code string, payout int64) error {
	if payout > 0 && f.payoutTodayLocked(code)+payout > fakeMaxPayoutDaily {
		return pointsErr(409, "game_payout_limit", fmt.Sprintf("這款遊戲每天派給一位玩家最多 %d", fakeMaxPayoutDaily))
	}
	return nil
}

func strictPointsDecode(body []byte, v any) bool {
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()
	return dec.Decode(v) == nil
}

// ---------- 回執 ----------

func (b *fakeBet) stakeReceipt(replayed bool) map[string]any {
	out := map[string]any{"betId": b.ID, "roundId": b.RoundID, "status": "staked", "stake": b.Stake, "beforeBalance": b.StakeBefore, "balance": b.StakeAfter, "replayed": replayed}
	if b.Mode == "settle" {
		out["status"], out["payout"] = "settled", b.Payout
	}
	return out
}

func (b *fakeBet) payoutReceipt(replayed bool) map[string]any {
	return map[string]any{"betId": b.ID, "roundId": b.RoundID, "status": "settled", "stake": b.Stake, "payout": b.Payout,
		"beforeBalance": b.PayoutBefore, "balance": b.PayoutAfter, "replayed": replayed}
}

func (b *fakeBet) cancelReceipt(replayed bool) map[string]any {
	return map[string]any{"betId": b.ID, "roundId": b.RoundID, "status": "cancelled", "stake": b.Stake,
		"beforeBalance": b.CancelBefore, "balance": b.CancelAfter, "replayed": replayed}
}

func (s *fakeSeat) buyInReceipt(replayed bool) map[string]any {
	return map[string]any{"sessionId": s.ID, "tableId": s.TableID, "status": "seated", "buyIn": s.BuyIn, "beforeBalance": s.BuyInBefore,
		"balance": s.BuyInAfter, "replayed": replayed, "house": s.House}
}

func (s *fakeSeat) closeReceipt(replayed bool) map[string]any {
	out := map[string]any{"sessionId": s.ID, "tableId": s.TableID, "status": s.Status, "buyIn": s.BuyIn, "beforeBalance": s.CloseBefore,
		"balance": s.CloseAfter, "replayed": replayed, "house": s.House}
	if s.CashOut != nil {
		out["cashOut"] = *s.CashOut
	}
	return out
}

func replayNote(replayed bool) string {
	if replayed {
		return "（重送，回傳原結果）"
	}
	return ""
}

// ---------- API ----------

// GET /v1/game/points/balance?roomId=&playerId=
func (f *fakePlatform) pointsBalance(r *http.Request, _ []byte) (map[string]any, string, error) {
	q := r.URL.Query()
	room, err := f.roomForPointsLocked(q.Get("roomId"))
	if err != nil {
		return nil, "", err
	}
	if !fakePlayerIDPattern.MatchString(q.Get("playerId")) {
		return nil, "", pointsErr(400, "bad_request", "playerId 格式不符")
	}
	if err := f.pointsGateLocked(); err != nil {
		return nil, "", err
	}
	code, err := f.betGuardLocked(room, q.Get("playerId"), int32(room.Epoch))
	if err != nil {
		return nil, "", err
	}
	p := f.pointsLocked()
	balance, enabled, left := p.bal(code), !p.disabled[code], f.stakeLeftLocked(code)
	if !enabled {
		left = 0
	}
	out := map[string]any{"playerId": q.Get("playerId"), "balance": balance, "enabled": enabled, "stakeLeftToday": min(left, balance), "epoch": room.Epoch,
		"maxStake": fakeMaxStake, "maxPayout": fakeMaxPayout, "maxBuyIn": fakeMaxBuyIn}
	return out, fmt.Sprintf("%s：%d 點%s", f.players[code].Name, balance, map[bool]string{true: "", false: "（沒有開啟點數玩法）"}[enabled]), nil
}

type fakeBetIn struct {
	RoomID   string `json:"roomId"`
	PlayerID string `json:"playerId"`
	Epoch    int32  `json:"epoch"`
	BetID    string `json:"betId"`
	RoundID  string `json:"roundId"`
	Stake    int64  `json:"stake"`
	Payout   *int64 `json:"payout"`
	Result   string `json:"result"`
}

// placeBet：settle 與 stake 共用。mode 是 settle 或 staged（和平台的紀錄相同）。
func (f *fakePlatform) placeBet(body []byte, mode string) (map[string]any, string, error) {
	var in fakeBetIn
	needPayout := mode == "settle"
	if !strictPointsDecode(body, &in) || !fakePlayerIDPattern.MatchString(in.PlayerID) || !fakeGameRefPattern.MatchString(in.BetID) ||
		!fakeGameRefPattern.MatchString(in.RoundID) || len([]rune(in.Result)) > fakeResultMaxLen || needPayout != (in.Payout != nil) {
		return nil, "", pointsErr(400, "bad_request", "格式不符：roomId、playerId、epoch、betId、roundId、stake"+map[bool]string{true: "、payout", false: ""}[needPayout]+"、result（選填）")
	}
	if in.Stake <= 0 || in.Stake > fakeMaxStake || (in.Payout != nil && (*in.Payout < 0 || *in.Payout > fakeMaxPayout)) {
		return nil, "", pointsErr(400, "bad_amount", fmt.Sprintf("下注額要 1～%d，派彩 0～%d", fakeMaxStake, fakeMaxPayout))
	}
	room, err := f.roomForPointsLocked(in.RoomID)
	if err != nil {
		return nil, "", err
	}
	payout := int64(0)
	if in.Payout != nil {
		payout = *in.Payout
	}
	h := fakeHash(map[string]any{"op": mode, "roomId": in.RoomID, "playerId": in.PlayerID, "epoch": in.Epoch, "roundId": in.RoundID,
		"stake": in.Stake, "payout": payout, "result": in.Result})
	p := f.pointsLocked()
	if prev := p.bets[in.BetID]; prev != nil {
		if prev.Hash != h {
			return nil, "", pointsErr(409, "idempotency_conflict", "同一個 betId 送了不同的內容")
		}
		return prev.stakeReceipt(true), "注單 " + in.BetID + replayNote(true), nil
	}
	if err := f.pointsGateLocked(); err != nil {
		return nil, "", err
	}
	code, err := f.betGuardLocked(room, in.PlayerID, in.Epoch)
	if err != nil {
		return nil, "", err
	}
	if err := f.stakeAllowedLocked(code, in.Stake); err != nil {
		return nil, "", err
	}
	if err := f.payoutAllowedLocked(code, payout); err != nil {
		return nil, "", err
	}
	before := p.bal(code)
	if before < in.Stake {
		return nil, "", pointsErr(402, "insufficient_test_points", fmt.Sprintf("剩 %d 點（控制面板可以發點數）", before))
	}
	now := f.now()
	b := &fakeBet{ID: in.BetID, RoundID: in.RoundID, RoomID: in.RoomID, Player: code, Mode: mode, Status: "staked", Stake: in.Stake, Hash: h, CreatedAt: now, StakeBefore: before}
	f.addPointTxLocked(code, "下注", in.BetID, -in.Stake)
	if mode == "settle" {
		if payout > 0 {
			f.addPointTxLocked(code, "派彩", in.BetID, payout)
		}
		b.Status, b.Payout, b.SettledAt = "settled", payout, now
	}
	b.StakeAfter = p.bal(code)
	p.bets[in.BetID] = b
	name := f.players[code].Name
	if mode == "settle" {
		return b.stakeReceipt(false), fmt.Sprintf("%s 下注 %d、派彩 %d，剩 %d 點", name, in.Stake, payout, b.StakeAfter), nil
	}
	return b.stakeReceipt(false), fmt.Sprintf("%s 下注 %d（等派彩或取消），剩 %d 點", name, in.Stake, b.StakeAfter), nil
}

// POST /v1/game/points/payoff {betId, payout, result?}
func (f *fakePlatform) pointsPayoff(_ *http.Request, body []byte) (map[string]any, string, error) {
	var in struct {
		BetID  string `json:"betId"`
		Payout *int64 `json:"payout"`
		Result string `json:"result"`
	}
	if !strictPointsDecode(body, &in) || !fakeGameRefPattern.MatchString(in.BetID) || in.Payout == nil || len([]rune(in.Result)) > fakeResultMaxLen {
		return nil, "", pointsErr(400, "bad_request", "格式不符：betId、payout、result（選填）")
	}
	if *in.Payout < 0 || *in.Payout > fakeMaxPayout {
		return nil, "", pointsErr(400, "bad_amount", fmt.Sprintf("派彩要 0～%d", fakeMaxPayout))
	}
	h := fakeHash(map[string]any{"op": "payoff", "payout": *in.Payout, "result": in.Result})
	p := f.pointsLocked()
	b := p.bets[in.BetID]
	switch {
	case b == nil:
		return nil, "", pointsErr(404, "bet_not_found", "")
	case b.Status == "settled" && b.Mode == "staged" && b.PayoutHash == h:
		return b.payoutReceipt(true), "派彩 " + in.BetID + replayNote(true), nil
	case b.Status == "settled":
		return nil, "", pointsErr(409, "bet_settled", "這張注單已經派彩（一次結算的注單不能再派彩）")
	case b.Status == "cancelled":
		return nil, "", pointsErr(409, "bet_cancelled", "")
	}
	if err := f.payoutAllowedLocked(b.Player, *in.Payout); err != nil {
		return nil, "", err
	}
	b.PayoutBefore = p.bal(b.Player)
	if *in.Payout > 0 {
		f.addPointTxLocked(b.Player, "派彩", b.ID, *in.Payout)
	}
	b.Status, b.Payout, b.PayoutHash, b.SettledAt, b.PayoutAfter = "settled", *in.Payout, h, f.now(), p.bal(b.Player)
	return b.payoutReceipt(false), fmt.Sprintf("%s 派彩 %d，剩 %d 點", f.players[b.Player].Name, *in.Payout, b.PayoutAfter), nil
}

// POST /v1/game/points/cancel {betId}
func (f *fakePlatform) pointsCancel(_ *http.Request, body []byte) (map[string]any, string, error) {
	var in struct {
		BetID string `json:"betId"`
	}
	if !strictPointsDecode(body, &in) || !fakeGameRefPattern.MatchString(in.BetID) {
		return nil, "", pointsErr(400, "bad_request", "格式不符：betId")
	}
	p := f.pointsLocked()
	b := p.bets[in.BetID]
	switch {
	case b == nil:
		return nil, "", pointsErr(404, "bet_not_found", "扣款沒有成立，不用處理")
	case b.Status == "cancelled":
		return b.cancelReceipt(true), "取消 " + in.BetID + replayNote(true), nil
	case b.Status == "settled":
		return nil, "", pointsErr(409, "bet_settled", "")
	}
	b.CancelBefore = p.bal(b.Player)
	f.addPointTxLocked(b.Player, "取消", b.ID, b.Stake)
	b.Status, b.SettledAt, b.CancelAfter = "cancelled", f.now(), p.bal(b.Player)
	return b.cancelReceipt(false), fmt.Sprintf("%s 取消注單，退回 %d，剩 %d 點", f.players[b.Player].Name, b.Stake, b.CancelAfter), nil
}

// POST /v1/game/points/table/buyin {roomId, playerId, epoch, tableId, sessionId, amount, house?}
func (f *fakePlatform) pointsBuyIn(_ *http.Request, body []byte) (map[string]any, string, error) {
	var in struct {
		RoomID    string `json:"roomId"`
		PlayerID  string `json:"playerId"`
		Epoch     int32  `json:"epoch"`
		TableID   string `json:"tableId"`
		SessionID string `json:"sessionId"`
		Amount    int64  `json:"amount"`
		House     bool   `json:"house"`
	}
	if !strictPointsDecode(body, &in) || !fakePlayerIDPattern.MatchString(in.PlayerID) || !fakeGameRefPattern.MatchString(in.TableID) || !fakeGameRefPattern.MatchString(in.SessionID) {
		return nil, "", pointsErr(400, "bad_request", "格式不符：roomId、playerId、epoch、tableId、sessionId、amount、house（選填）")
	}
	if in.Amount <= 0 || in.Amount > fakeMaxBuyIn {
		return nil, "", pointsErr(400, "bad_amount", fmt.Sprintf("入座要 1～%d", fakeMaxBuyIn))
	}
	room, err := f.roomForPointsLocked(in.RoomID)
	if err != nil {
		return nil, "", err
	}
	req := map[string]any{"op": "buyin", "roomId": in.RoomID, "playerId": in.PlayerID, "epoch": in.Epoch, "tableId": in.TableID, "amount": in.Amount}
	if in.House {
		req["house"] = true
	}
	h := fakeHash(req)
	p := f.pointsLocked()
	if prev := p.seats[in.SessionID]; prev != nil {
		if prev.Hash != h {
			return nil, "", pointsErr(409, "idempotency_conflict", "同一個 sessionId 送了不同的內容")
		}
		return prev.buyInReceipt(true), "入座 " + in.SessionID + replayNote(true), nil
	}
	if err := f.pointsGateLocked(); err != nil {
		return nil, "", err
	}
	code, err := f.betGuardLocked(room, in.PlayerID, in.Epoch)
	if err != nil {
		return nil, "", err
	}
	for _, s := range p.seats {
		if s.RoomID == in.RoomID && s.TableID == in.TableID && s.House != in.House {
			return nil, "", pointsErr(409, "table_mode_mismatch", "同一張桌不能混用玩家之間的牌桌與機台型（house）")
		}
	}
	if err := f.stakeAllowedLocked(code, in.Amount); err != nil {
		return nil, "", err
	}
	before := p.bal(code)
	if before < in.Amount {
		return nil, "", pointsErr(402, "insufficient_test_points", fmt.Sprintf("剩 %d 點（控制面板可以發點數）", before))
	}
	kind := map[bool]string{true: "機台轉入", false: "牌桌轉入"}[in.House]
	f.addPointTxLocked(code, kind, in.SessionID, -in.Amount)
	s := &fakeSeat{ID: in.SessionID, TableID: in.TableID, RoomID: in.RoomID, Player: code, Status: "seated", House: in.House, BuyIn: in.Amount,
		Hash: h, CreatedAt: f.now(), BuyInBefore: before, BuyInAfter: p.bal(code)}
	p.seats[in.SessionID] = s
	return s.buyInReceipt(false), fmt.Sprintf("%s %s %d（桌 %s），剩 %d 點", f.players[code].Name, kind, in.Amount, in.TableID, s.BuyInAfter), nil
}

// POST /v1/game/points/table/cashout {sessionId, amount}
func (f *fakePlatform) pointsCashOut(_ *http.Request, body []byte) (map[string]any, string, error) {
	var in struct {
		SessionID string `json:"sessionId"`
		Amount    *int64 `json:"amount"`
	}
	if !strictPointsDecode(body, &in) || !fakeGameRefPattern.MatchString(in.SessionID) || in.Amount == nil {
		return nil, "", pointsErr(400, "bad_request", "格式不符：sessionId、amount")
	}
	amount := *in.Amount
	if amount < 0 {
		return nil, "", pointsErr(400, "bad_amount", "轉出不能是負數")
	}
	h := fakeHash(map[string]any{"op": "cashout", "amount": amount})
	p := f.pointsLocked()
	s := p.seats[in.SessionID]
	switch {
	case s == nil:
		return nil, "", pointsErr(404, "session_not_found", "")
	case s.Status == "left" && s.CashOutHash == h:
		return s.closeReceipt(true), "轉出 " + in.SessionID + replayNote(true), nil
	case s.Status == "left":
		return nil, "", pointsErr(409, "session_closed", "這次入座已經轉出")
	}
	if s.House {
		if amount > s.BuyIn+fakeMaxHouseWin {
			return nil, "", pointsErr(400, "bad_amount", fmt.Sprintf("機台型一次最多淨贏 %d：轉出不能超過 %d", fakeMaxHouseWin, s.BuyIn+fakeMaxHouseWin))
		}
		if amount > s.BuyIn {
			if err := f.payoutAllowedLocked(s.Player, amount-s.BuyIn); err != nil {
				return nil, "", err
			}
		}
	} else {
		var buyins, cashouts int64
		for _, o := range p.seats {
			if o.RoomID == s.RoomID && o.TableID == s.TableID && !o.House {
				buyins += o.BuyIn
				if o.Status == "left" {
					cashouts += *o.CashOut
				}
			}
		}
		if amount > buyins-cashouts {
			return nil, "", pointsErr(409, "table_overdraw", fmt.Sprintf("桌 %s 目前剩 %d（轉入總和 %d − 已轉出 %d）：玩家之間的牌桌轉出不能超過轉入", s.TableID, buyins-cashouts, buyins, cashouts))
		}
	}
	s.CloseBefore = p.bal(s.Player)
	kind := map[bool]string{true: "機台轉出", false: "牌桌轉出"}[s.House]
	if amount > 0 {
		f.addPointTxLocked(s.Player, kind, s.ID, amount)
	}
	s.Status, s.CashOut, s.CashOutHash, s.ClosedAt, s.CloseAfter = "left", &amount, h, f.now(), p.bal(s.Player)
	return s.closeReceipt(false), fmt.Sprintf("%s %s %d（轉入 %d），剩 %d 點", f.players[s.Player].Name, kind, amount, s.BuyIn, s.CloseAfter), nil
}

// ---------- 控制面板 ----------

// pointsControl：發點數、切換玩家的點數玩法、平台核准的模擬。
func (f *fakePlatform) pointsControl(action, player, text string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	p := f.pointsLocked()
	switch action {
	case "points-grant", "points-toggle":
		if f.players[player] == nil {
			return fmt.Errorf("bad_player")
		}
		if action == "points-grant" {
			p.bal(player)
			f.addPointTxLocked(player, "發點數", "", fakePointsGrant)
		} else {
			p.disabled[player] = !p.disabled[player]
		}
	case "points-approval":
		if !fakePointsApprovals[text] {
			return fmt.Errorf("bad_approval")
		}
		p.approval = text
	default:
		return fmt.Errorf("bad_action")
	}
	return nil
}

type fakePointsState struct {
	Declared bool               `json:"declared"`
	Approval string             `json:"approval"`
	Players  []fakePointsPlayer `json:"players"`
	Open     []map[string]any   `json:"open"` // 還沒結束的注單與入座
	Txs      []fakePointTx      `json:"txs"`
	Limits   map[string]int64   `json:"limits"`
}

type fakePointsPlayer struct {
	Code       string `json:"code"`
	Name       string `json:"name"`
	Balance    int64  `json:"balance"`
	Enabled    bool   `json:"enabled"`
	StakeToday int64  `json:"stakeToday"`
}

func (f *fakePlatform) pointsStateLocked() fakePointsState {
	p := f.pointsLocked()
	s := fakePointsState{Declared: f.pointsStake, Approval: p.approval, Players: []fakePointsPlayer{}, Open: []map[string]any{}, Txs: []fakePointTx{},
		Limits: map[string]int64{"stake": fakeMaxStake, "payout": fakeMaxPayout, "buyIn": fakeMaxBuyIn, "dailyStake": fakeDailyStakeCap, "dailyPayout": fakeMaxPayoutDaily, "houseWin": fakeMaxHouseWin}}
	for _, code := range f.order {
		s.Players = append(s.Players, fakePointsPlayer{Code: code, Name: f.players[code].Name, Balance: p.bal(code), Enabled: !p.disabled[code], StakeToday: f.stakeTodayLocked(code)})
	}
	for _, b := range p.bets {
		if b.Status == "staked" {
			s.Open = append(s.Open, map[string]any{"kind": "注單", "ref": b.ID, "player": f.players[b.Player].Name, "amount": b.Stake, "at": b.CreatedAt})
		}
	}
	for _, t := range p.seats {
		if t.Status == "seated" {
			s.Open = append(s.Open, map[string]any{"kind": map[bool]string{true: "機台", false: "牌桌"}[t.House] + " " + t.TableID, "ref": t.ID, "player": f.players[t.Player].Name, "amount": t.BuyIn, "at": t.CreatedAt})
		}
	}
	sort.Slice(s.Open, func(i, j int) bool { return s.Open[i]["at"].(time.Time).Before(s.Open[j]["at"].(time.Time)) })
	for i := len(p.txs) - 1; i >= 0 && len(s.Txs) < 50; i-- {
		s.Txs = append(s.Txs, p.txs[i])
	}
	return s
}

// devPointsStake：專案有沒有宣告 economy.platformPointsStake（pt dev 啟動時讀一次）。
func devPointsStake(p *project) bool {
	if p == nil || p.Economy == nil {
		return false
	}
	var e struct {
		PlatformPointsStake *bool `json:"platformPointsStake"`
	}
	return json.Unmarshal(p.Economy, &e) == nil && e.PlatformPointsStake != nil && *e.PlatformPointsStake
}
