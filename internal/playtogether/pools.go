package playtogether

// 比賽水池（原型，只用測試點數；玩聚平台的 dev、stage 環境才開放）。
//
// 遊戲伺服器用平台核准的比賽格式建立比賽，遊戲瀏覽器端用 SDK 的 points.joinPool(poolId) 請平台顯示確認畫面，
// 全部參加者在平台確認後狀態變成 running，這時才開始比賽；結束後用 ReportPoolResult 回報名次（第一位是贏家）或平局。
// 點數的保管、分配與退回都由平台執行：遊戲不能指定金額或收款人，也拿不到玩家的餘額。

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"time"
)

// PoolRequest 是建立比賽的內容。Players 是房間裡的玩家 ID，人數要和格式相同。
type PoolRequest struct {
	RoomID         string
	Format         string // 平台核准的比賽格式代號
	Players        []string
	IdempotencyKey string // 同一個鍵重送回傳同一場比賽
}

// PoolPlayer 是一位參加者：invited（等待確認）、staked（已投入）、declined（拒絕）。
type PoolPlayer struct {
	PlayerID string `json:"playerId"`
	Status   string `json:"status"`
}

// Pool 是比賽的狀態：staking（等待確認）、running（全部投入，可以開始）、settling（結算中）、
// paid（已分配，Winner 是贏家）、refunded（已全額退回，RefundReason 是原因）。
type Pool struct {
	PoolID         string       `json:"poolId"`
	Status         string       `json:"status"`
	RoomID         string       `json:"roomId"`
	Format         string       `json:"format"`
	Stake          int          `json:"stake"`
	Players        []PoolPlayer `json:"players"`
	StakeDeadline  time.Time    `json:"stakeDeadline"`
	ResultDeadline *time.Time   `json:"resultDeadline,omitempty"`
	RefundReason   string       `json:"refundReason,omitempty"`
	Winner         string       `json:"winner,omitempty"`
	TestPoints     bool         `json:"testPoints"`
}

// PoolResult 是比賽結果：Ranks 是全部參加者的名次（第一位是贏家），或 Draw 平局（全額退回）。
// Actions（每人操作量）與 SuspectedThrowing（疑似放水的玩家）是選填的輔助資料，只供平台參考。
type PoolResult struct {
	IdempotencyKey    string
	Ranks             []string
	Draw              bool
	Actions           map[string]int
	SuspectedThrowing []string
}

func (c *Client) poolsDisabled() error {
	return &APIError{Status: http.StatusServiceUnavailable, Code: "points_pool_disabled"}
}

// CreatePool 建立比賽。
func (c *Client) CreatePool(ctx context.Context, in PoolRequest) (*Pool, error) {
	if in.RoomID == "" || in.Format == "" || len(in.Players) < 2 || len(in.IdempotencyKey) < 8 {
		return nil, errors.New("roomId、format 必填，players 至少 2 人，idempotencyKey 至少 8 字")
	}
	if c.Mock {
		return nil, c.poolsDisabled()
	}
	body := map[string]any{"roomId": in.RoomID, "format": in.Format, "players": in.Players, "idempotencyKey": in.IdempotencyKey}
	var out Pool
	if err := c.post(ctx, "/v1/game/pools", body, "", &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetPool 查詢比賽的狀態。
func (c *Client) GetPool(ctx context.Context, poolID string) (*Pool, error) {
	if c.Mock {
		return nil, c.poolsDisabled()
	}
	var out Pool
	if err := c.do(ctx, http.MethodGet, "/v1/game/pools/"+url.PathEscape(poolID), nil, "", &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ReportPoolResult 回報比賽結果。一場比賽只有一個結果：同一個鍵、同樣的內容重送回傳原結果，其他回 409 pool_already_reported。
func (c *Client) ReportPoolResult(ctx context.Context, poolID string, in PoolResult) (*Pool, error) {
	if len(in.IdempotencyKey) < 8 || in.Draw == (len(in.Ranks) > 0) {
		return nil, errors.New("idempotencyKey 至少 8 字；ranks 和 draw 二選一")
	}
	if c.Mock {
		return nil, c.poolsDisabled()
	}
	body := map[string]any{"idempotencyKey": in.IdempotencyKey}
	if in.Draw {
		body["draw"] = true
	} else {
		body["ranks"] = in.Ranks
	}
	if len(in.Actions) > 0 {
		body["actions"] = in.Actions
	}
	if len(in.SuspectedThrowing) > 0 {
		body["suspectedThrowing"] = in.SuspectedThrowing
	}
	var out Pool
	if err := c.post(ctx, "/v1/game/pools/"+url.PathEscape(poolID)+"/result", body, "", &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// CancelPool 取消還沒有結果的比賽（全額退回）。
func (c *Client) CancelPool(ctx context.Context, poolID string) (*Pool, error) {
	if c.Mock {
		return nil, c.poolsDisabled()
	}
	var out Pool
	if err := c.post(ctx, "/v1/game/pools/"+url.PathEscape(poolID)+"/cancel", map[string]any{}, "", &out); err != nil {
		return nil, err
	}
	return &out, nil
}
