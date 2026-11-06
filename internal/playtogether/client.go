package playtogether

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// VoiceMode 是發言規則的模式（docs/voice-chat.md）。
type VoiceMode string

const (
	VoiceOpen      VoiceMode = "open"
	VoiceHostOnly  VoiceMode = "host_only"
	VoiceTurn      VoiceMode = "turn"
	VoiceAllowlist VoiceMode = "allowlist"
)

// VoicePolicy 是遊戲伺服器送給平台的發言規則。
type VoicePolicy struct {
	RoomID              string    `json:"roomId"`
	GameInstanceID      string    `json:"gameInstanceId"`
	Sequence            int64     `json:"sequence"`
	Mode                VoiceMode `json:"mode"`
	AllowedSpeakerIDs   []string  `json:"allowedSpeakerIds"`
	EffectiveAtServerMs int64     `json:"effectiveAtServerMs"`
	ExpiresAtServerMs   int64     `json:"expiresAtServerMs"`
}

// Validate 和 packages/protocol 的 voicePolicySchema 規則相同，送出前先在本地檢查。
func (p VoicePolicy) Validate() error {
	if p.RoomID == "" || p.GameInstanceID == "" {
		return errors.New("roomId 與 gameInstanceId 必填")
	}
	if p.Sequence < 0 {
		return errors.New("sequence 不得為負數")
	}
	if p.EffectiveAtServerMs <= 0 || p.ExpiresAtServerMs <= p.EffectiveAtServerMs {
		return errors.New("expiresAtServerMs 必須晚於 effectiveAtServerMs")
	}
	n := len(p.AllowedSpeakerIDs)
	if n > 64 {
		return errors.New("發言者名單最多 64 人")
	}
	seen := make(map[string]bool, n)
	for _, id := range p.AllowedSpeakerIDs {
		if id == "" || seen[id] {
			return errors.New("發言者名單不得為空字串或重複")
		}
		seen[id] = true
	}
	switch p.Mode {
	case VoiceTurn:
		if n != 1 {
			return errors.New("turn 模式必須剛好一位發言者")
		}
	case VoiceAllowlist:
		if n < 1 {
			return errors.New("allowlist 模式至少要一位發言者")
		}
	case VoiceOpen, VoiceHostOnly:
		if n > 0 {
			return fmt.Errorf("%s 模式不帶發言者名單", p.Mode)
		}
	default:
		return fmt.Errorf("未知的模式：%q", p.Mode)
	}
	return nil
}

// LaunchExchange 是兌換啟動碼的結果。
type LaunchExchange struct {
	GameID   string `json:"gameId"`
	PlayerID string `json:"playerId"`
	Name     string `json:"name"`
	RoomID   string `json:"roomId"`
	// Epoch 是房間目前的局數：房主「重新開始」時加一。請依局數開新局，並拒絕舊局的連線
	Epoch int `json:"epoch"`
	// HostPlayerID 是平台房主的假名 ID。遊戲的房主（開始遊戲、設定）請以此為準：
	// 平台的踢人、重新開始也是這個人操作。房主可能還沒進遊戲。
	HostPlayerID string `json:"hostPlayerId"`
}

// APIError 是平台 API 回傳的錯誤。
type APIError struct {
	Status int
	Code   string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("玩聚平台 API 錯誤：%d %s", e.Status, e.Code)
}

// Client 呼叫玩聚平台 API，所有請求自動簽章。
type Client struct {
	// BaseURL 例如 https://api.playtogether.example
	BaseURL    string
	KeyID      string
	PrivateKey ed25519.PrivateKey
	HTTPClient *http.Client
	// Now 預設為 time.Now，測試時可替換。
	Now func() time.Time
	// Mock 為 true 時不連平台：接受瀏覽器 SDK 模擬模式產生的 "mock-" 啟動碼，發言規則只做本地檢查。
	// 只用於本機開發，正式環境絕對不能開。
	Mock bool
}

// MockCodePrefix 是瀏覽器 SDK 模擬模式產生的啟動碼前綴。
const MockCodePrefix = "mock-"

func (c *Client) post(ctx context.Context, path string, body any, idempotencyKey string, out any) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	return c.do(ctx, http.MethodPost, path, data, idempotencyKey, out)
}

func (c *Client) do(ctx context.Context, method, path string, data []byte, idempotencyKey string, out any) error {
	now := time.Now
	if c.Now != nil {
		now = c.Now
	}
	headers := SignRequest(SignableRequest{Method: method, Path: path, Body: data},
		c.KeyID, c.PrivateKey, now().UnixMilli(), NewNonce())
	req, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(c.BaseURL, "/")+path, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header = headers
	if data != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if idempotencyKey != "" {
		req.Header.Set(HeaderIdempotencyKey, idempotencyKey)
	}
	hc := c.HTTPClient
	if hc == nil {
		hc = &http.Client{Timeout: 5 * time.Second}
	}
	res, err := hc.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		var e struct {
			Error string `json:"error"`
		}
		_ = json.NewDecoder(res.Body).Decode(&e)
		if e.Error == "" {
			e.Error = "unknown"
		}
		return &APIError{Status: res.StatusCode, Code: e.Error}
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(res.Body).Decode(out)
}

// ExchangeLaunchCode 兌換一次性啟動碼，取得玩家假名 ID、暱稱、平台房號。
// 同一位玩家在同一款遊戲的 PlayerID 固定，可用來讓玩家重新整理後回到原座位。
func (c *Client) ExchangeLaunchCode(ctx context.Context, code string) (*LaunchExchange, error) {
	if len(code) < 16 || len(code) > 256 {
		return nil, errors.New("啟動碼格式錯誤")
	}
	if c.Mock {
		if !strings.HasPrefix(code, MockCodePrefix) {
			return nil, errors.New("模擬模式只接受 mock- 開頭的啟動碼")
		}
		// 同一個啟動碼對應同一位假玩家，方便測試重新加入
		suffix := code[len(MockCodePrefix) : len(MockCodePrefix)+8]
		id := "mock-p-" + suffix
		// 名稱帶上代號，開多個分頁時分得出是哪一位
		return &LaunchExchange{GameID: "mock-game", PlayerID: id, Name: "測試玩家-" + strings.ToUpper(suffix[:4]), RoomID: "mock-room", HostPlayerID: id}, nil
	}
	if strings.HasPrefix(code, MockCodePrefix) {
		return nil, errors.New("收到模擬模式的啟動碼，但伺服器不是模擬模式")
	}
	var out LaunchExchange
	if err := c.post(ctx, "/v1/launch/exchange", map[string]string{"code": code}, "", &out); err != nil {
		return nil, err
	}
	if out.GameID == "" || out.PlayerID == "" || out.RoomID == "" {
		return nil, errors.New("平台回應缺少必要欄位")
	}
	return &out, nil
}

// SendVoicePolicy 送出發言規則；同一房間同一序號重送不會重複執行。
func (c *Client) SendVoicePolicy(ctx context.Context, p VoicePolicy) error {
	if p.AllowedSpeakerIDs == nil {
		// 避免序列化成 null，平台會拒絕
		p.AllowedSpeakerIDs = []string{}
	}
	if err := p.Validate(); err != nil {
		return err
	}
	if c.Mock {
		return nil
	}
	key := fmt.Sprintf("voice:%s:%d", p.RoomID, p.Sequence)
	return c.post(ctx, "/v1/voice/policy", p, key, nil)
}

// RoomMembers 是房間目前的狀態：是否開放、局數，以及每位成員是否被房主踢出。
type RoomMembers struct {
	RoomID string `json:"roomId"`
	Open   bool   `json:"open"`
	Epoch  int    `json:"epoch"`
	// HostPlayerID 是平台房主的假名 ID
	HostPlayerID string `json:"hostPlayerId"`
	Members      []struct {
		PlayerID string `json:"playerId"`
		Kicked   bool   `json:"kicked"`
	} `json:"members"`
}

// GetRoomMembers 查詢房間狀態。遊戲伺服器可以定期查詢（例如每 10 秒），
// 把被踢出的玩家斷線、在局數改變時開新局，並在房間關閉後結束遊戲。
func (c *Client) GetRoomMembers(ctx context.Context, roomID string) (*RoomMembers, error) {
	if c.Mock {
		return &RoomMembers{RoomID: roomID, Open: true}, nil
	}
	var out RoomMembers
	if err := c.do(ctx, http.MethodGet, "/v1/game/rooms/"+url.PathEscape(roomID)+"/members", nil, "", &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// RoundResult 是一局結束的方式。
type RoundResult string

const (
	RoundCompleted RoundResult = "completed" // 正常玩完
	RoundAborted   RoundResult = "aborted"   // 中斷：人數不足、房主重新開始、房間關閉等
)

// ReportRoundStart 回報一局開始與這一局的玩家（只用於平台的成效分析：完成率、連玩率、從連結到開局的時間）。
// roundID 由遊戲自己決定，同一房間內不可重複（英數字與 _ . : -，最多 64 字）；同一局重送不會重複記錄。
// epoch 必須是目前的局數（ExchangeLaunchCode、GetRoomMembers 回傳），舊局數會被拒絕（stale_epoch）。
func (c *Client) ReportRoundStart(ctx context.Context, roomID string, epoch int, roundID string, playerIDs []string) error {
	if roundID == "" || len(playerIDs) == 0 {
		return errors.New("roundID 與玩家不可為空")
	}
	if c.Mock {
		return nil
	}
	body := map[string]any{"roomId": roomID, "epoch": epoch, "roundId": roundID, "phase": "start", "playerIds": playerIDs}
	return c.post(ctx, "/v1/game/rounds", body, "round:"+roomID+":"+roundID+":start", nil)
}

// ReportRoundEnd 回報一局結束。result 用 RoundCompleted 或 RoundAborted。
func (c *Client) ReportRoundEnd(ctx context.Context, roomID string, epoch int, roundID string, result RoundResult) error {
	if result != RoundCompleted && result != RoundAborted {
		return errors.New("result 必須是 completed 或 aborted")
	}
	if c.Mock {
		return nil
	}
	body := map[string]any{"roomId": roomID, "epoch": epoch, "roundId": roundID, "phase": "end", "result": result}
	return c.post(ctx, "/v1/game/rounds", body, "round:"+roomID+":"+roundID+":end", nil)
}

// GameStatus 是遊戲在玩聚上的狀態（GET /v1/game/status）。
type GameStatus struct {
	GameID string `json:"gameId"`
	// Status：draft、live、unlisted、disabled
	Status string `json:"status"`
	// Serving 為 false 時遊戲伺服器應停止服務（遊戲被停用）；Reason 是原因代碼（game_disabled）
	Serving     bool   `json:"serving"`
	Reason      string `json:"reason"`
	Maintenance struct {
		Game bool   `json:"game"` // 單一遊戲維護中（平台不讓新玩家進房，已在房間的照常）
		Site string `json:"site"` // 全站維護：""、no_new_rooms、site_down
	} `json:"maintenance"`
}

// GetGameStatus 查詢遊戲狀態。遊戲被停用時仍可呼叫（回 Serving=false）；金鑰撤銷時回 401 unknown_key。
// 建議每分鐘查一次（gameserver 套件會自動查）。
func (c *Client) GetGameStatus(ctx context.Context) (*GameStatus, error) {
	if c.Mock {
		return &GameStatus{Status: "live", Serving: true}, nil
	}
	var out GameStatus
	if err := c.do(ctx, http.MethodGet, "/v1/game/status", nil, "", &out); err != nil {
		return nil, err
	}
	return &out, nil
}
