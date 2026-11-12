package playtogether

// 遊戲伺服器的點數下注與派彩（原型，只用測試點數；玩聚平台的 dev、stage 環境才開放）。依玩法選流程：
//
//	即開型（老虎機）       SettleBet：一次送下注額與派彩
//	有等待時間的局          StakeBet 先扣 → PayoffBet 派彩，或 CancelBet 取消（扣款請求逾時沒收到回應時）
//	坐桌型（撲克、麻將）    TableBuyIn 入座轉入 → TableCashOut 離座轉出（同一張桌的轉出總和不超過轉入總和）
//	機台型（捕魚機）        TableBuyIn{House: true} 轉入（rollin）→ TableCashOut 轉出（rollout）：玩家對遊戲、照倍率派彩，
//	                        轉出可以多於轉入（一次最多淨贏 1,000，算進每日派彩上限）
//
// 只限宣告了 economy.platformPointsStake 的版本；玩家要在玩聚「我的點數」開啟；下注與入座要帶房間目前的局數（Epoch，
// 從兌換啟動碼或 GetPointsBalance 取得）；單筆與每日有上限；同一個 BetID／SessionID 重送回傳原結果（Replayed）。

import (
	"context"
	"errors"
	"net/http"
	"net/url"
)

// PointsBalance 是玩家的點數狀態。
type PointsBalance struct {
	PlayerID       string `json:"playerId"`
	Balance        int64  `json:"balance"`
	Enabled        bool   `json:"enabled"`        // 玩家有沒有開啟點數玩法（沒開啟時不能下注或入座）
	StakeLeftToday int64  `json:"stakeLeftToday"` // 今天還能下注或入座的點數（每日上限與餘額取小的）
	Epoch          int32  `json:"epoch"`          // 房間目前的局數
	MaxStake       int64  `json:"maxStake"`
	MaxPayout      int64  `json:"maxPayout"`
	MaxBuyIn       int64  `json:"maxBuyIn"`
	TestPoints     bool   `json:"testPoints"`
}

// Bet 是一張注單。BetID 在這款遊戲裡唯一（1～100 字，英數字與 . _ : -）；RoundID 用來分組（同一局可以有多張注單）。
type Bet struct {
	RoomID   string
	PlayerID string
	Epoch    int32
	BetID    string
	RoundID  string
	Stake    int64  // 下注額
	Payout   int64  // 派彩（含本金；沒中是 0）：只用在 SettleBet
	Result   string // 選填：這局的結果或牌局紀錄代號（最多 500 字）
}

// BetResult 是注單的狀態與異動前後的餘額。Status：staked、settled、cancelled。
type BetResult struct {
	BetID         string `json:"betId"`
	RoundID       string `json:"roundId"`
	Status        string `json:"status"`
	Stake         int64  `json:"stake"`
	Payout        *int64 `json:"payout,omitempty"`
	BeforeBalance int64  `json:"beforeBalance"`
	Balance       int64  `json:"balance"`
	Replayed      bool   `json:"replayed"`
	TestPoints    bool   `json:"testPoints"`
}

// TableBuyIn 是一次入座。SessionID 在這款遊戲裡唯一；TableID 是牌桌（守恆以遊戲＋房間＋TableID 計算）。
// House 為 true 是機台型（rollin／rollout，例如捕魚機）：不受守恆限制，同一張桌不能和玩家之間的牌桌混用（table_mode_mismatch）。
type TableBuyIn struct {
	RoomID    string
	PlayerID  string
	Epoch     int32
	TableID   string
	SessionID string
	Amount    int64
	House     bool
}

// TableResult 是入座的狀態與異動前後的餘額。Status：seated、left。
type TableResult struct {
	SessionID     string `json:"sessionId"`
	TableID       string `json:"tableId"`
	Status        string `json:"status"`
	BuyIn         int64  `json:"buyIn"`
	CashOut       *int64 `json:"cashOut,omitempty"`
	BeforeBalance int64  `json:"beforeBalance"`
	Balance       int64  `json:"balance"`
	Replayed      bool   `json:"replayed"`
	TestPoints    bool   `json:"testPoints"`
	House         bool   `json:"house"` // 機台型
}

// GetPointsBalance 取得房間裡一位玩家的點數狀態。
func (c *Client) GetPointsBalance(ctx context.Context, roomID, playerID string) (*PointsBalance, error) {
	if c.Mock {
		return nil, c.poolsDisabled()
	}
	var out PointsBalance
	q := url.Values{"roomId": {roomID}, "playerId": {playerID}} // 查詢字串在簽章範圍內
	if err := c.doQuery(ctx, http.MethodGet, "/v1/game/points/balance", q, nil, "", &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func betBody(b Bet) map[string]any {
	body := map[string]any{"roomId": b.RoomID, "playerId": b.PlayerID, "epoch": b.Epoch, "betId": b.BetID, "roundId": b.RoundID, "stake": b.Stake}
	if b.Result != "" {
		body["result"] = b.Result
	}
	return body
}

func (b Bet) valid() error {
	if b.RoomID == "" || b.PlayerID == "" || b.BetID == "" || b.RoundID == "" || b.Stake < 0 || b.Payout < 0 {
		return errors.New("roomId、playerId、betId、roundId 必填，金額不能是負數")
	}
	return nil
}

// SettleBet 一次結算（即開型）：同一筆交易扣下注額（> 0）、發派彩（≥ 0）。不能零下注直接派彩（免費旋轉的獎金算進觸發它的注單）。
// 常見的錯誤代碼：bet_expired、session_expired、session_closed、insufficient_test_points、points_not_enabled、daily_limit、game_payout_limit、bad_amount、epoch_mismatch、
// declaration_required、idempotency_conflict、preview_room、player_not_in_room、room_closed、purchases_suspended、points_pool_disabled。
func (c *Client) SettleBet(ctx context.Context, b Bet) (*BetResult, error) {
	if err := b.valid(); err != nil {
		return nil, err
	}
	if b.Stake == 0 { // 和平台相同的錯誤代碼
		return nil, &APIError{Status: http.StatusBadRequest, Code: "bad_amount"}
	}
	if c.Mock {
		return nil, c.poolsDisabled()
	}
	body := betBody(b)
	body["payout"] = b.Payout
	var out BetResult
	if err := c.post(ctx, "/v1/game/points/settle", body, "", &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// StakeBet 先扣（有等待時間的局）：之後用 PayoffBet 派彩或 CancelBet 取消；6 小時內沒有結束，平台自動取消。
func (c *Client) StakeBet(ctx context.Context, b Bet) (*BetResult, error) {
	if err := b.valid(); err != nil {
		return nil, err
	}
	if b.Stake == 0 {
		return nil, errors.New("stake 要大於 0")
	}
	if c.Mock {
		return nil, c.poolsDisabled()
	}
	var out BetResult
	if err := c.post(ctx, "/v1/game/points/stake", betBody(b), "", &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// PayoffBet 派彩（沒中送 0）。玩家離開或房間關閉後仍然可以派彩。已經派彩回 bet_settled，已經取消回 bet_cancelled。
func (c *Client) PayoffBet(ctx context.Context, betID string, payout int64, result string) (*BetResult, error) {
	if betID == "" || payout < 0 {
		return nil, errors.New("betId 必填，payout 不能是負數")
	}
	if c.Mock {
		return nil, c.poolsDisabled()
	}
	body := map[string]any{"betId": betID, "payout": payout}
	if result != "" {
		body["result"] = result
	}
	var out BetResult
	if err := c.post(ctx, "/v1/game/points/payoff", body, "", &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// CancelBet 取消先扣的注單（退還下注額）。注單不存在回 bet_not_found（扣款沒成立，不用處理）；已經派彩回 bet_settled。
func (c *Client) CancelBet(ctx context.Context, betID string) (*BetResult, error) {
	if betID == "" {
		return nil, errors.New("betId 必填")
	}
	if c.Mock {
		return nil, c.poolsDisabled()
	}
	var out BetResult
	if err := c.post(ctx, "/v1/game/points/cancel", map[string]any{"betId": betID}, "", &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// TableBuyIn 入座轉入（坐桌型）：6 小時內沒有轉出，平台把轉入退回。
func (c *Client) TableBuyIn(ctx context.Context, in TableBuyIn) (*TableResult, error) {
	if in.RoomID == "" || in.PlayerID == "" || in.TableID == "" || in.SessionID == "" || in.Amount <= 0 {
		return nil, errors.New("roomId、playerId、tableId、sessionId 必填，amount 要大於 0")
	}
	if c.Mock {
		return nil, c.poolsDisabled()
	}
	body := map[string]any{"roomId": in.RoomID, "playerId": in.PlayerID, "epoch": in.Epoch, "tableId": in.TableID, "sessionId": in.SessionID, "amount": in.Amount}
	if in.House {
		body["house"] = true
	}
	var out TableResult
	if err := c.post(ctx, "/v1/game/points/table/buyin", body, "", &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// TableCashOut 離座轉出桌上剩下的點數（可以是 0）。同一張桌的轉出總和超過轉入總和回 table_overdraw；
// 機台型：轉出超過轉入＋1,000 回 bad_amount，淨贏超過這款遊戲今天的派彩上限回 game_payout_limit。
func (c *Client) TableCashOut(ctx context.Context, sessionID string, amount int64) (*TableResult, error) {
	if sessionID == "" || amount < 0 {
		return nil, errors.New("sessionId 必填，amount 不能是負數")
	}
	if c.Mock {
		return nil, c.poolsDisabled()
	}
	var out TableResult
	if err := c.post(ctx, "/v1/game/points/table/cashout", map[string]any{"sessionId": sessionID, "amount": amount}, "", &out); err != nil {
		return nil, err
	}
	return &out, nil
}
