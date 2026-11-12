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
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
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
	RoomID            string    `json:"roomId"`
	GameInstanceID    string    `json:"gameInstanceId"`
	Sequence          int64     `json:"sequence"`
	Mode              VoiceMode `json:"mode"`
	AllowedSpeakerIDs []string  `json:"allowedSpeakerIds"`
	// Groups 是分組語音（選填，docs/voice-groups.md）：生效期間組員只在自己的組裡說話，其他人收不到；
	// 組員仍聽得到主頻道。一個人最多在一組。狼人殺的夜晚：Mode allowlist、AllowedSpeakerIDs 空、Groups 放狼人
	Groups              []VoiceGroup `json:"groups,omitempty"`
	EffectiveAtServerMs int64        `json:"effectiveAtServerMs"`
	ExpiresAtServerMs   int64        `json:"expiresAtServerMs"`
}

// VoiceGroup 是一個語音分組。ID 只能用小寫英文、數字、底線、連字號（英文開頭，最多 16 字）
type VoiceGroup struct {
	ID      string   `json:"id"`
	Members []string `json:"members"`
	// Listeners 是旁聽名單（選填，只聽不說，例如死亡玩家的上帝視角）：不能和這一組的組員重疊，可以是別組的組員；
	// 一個人最多旁聽 VoiceMaxListenGroups 組。拿掉旁聽者時平台會換新的組頻道，組員會重新連線（約 1 秒）
	Listeners []string `json:"listeners,omitempty"`
}

// VoiceMaxListenGroups 是一個人最多旁聽幾組（手機上每一組多一條語音連線）
const VoiceMaxListenGroups = 3

var voiceGroupIDRe = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,15}$`)

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
		// 有分組時主頻道可以沒有人能說話（例如狼人殺的夜晚）
		if n < 1 && len(p.Groups) == 0 {
			return errors.New("allowlist 模式至少要一位發言者")
		}
	case VoiceOpen, VoiceHostOnly:
		if n > 0 {
			return fmt.Errorf("%s 模式不帶發言者名單", p.Mode)
		}
	default:
		return fmt.Errorf("未知的模式：%q", p.Mode)
	}
	if len(p.Groups) > 8 {
		return errors.New("分組最多 8 組")
	}
	ids, members := map[string]bool{}, map[string]bool{}
	for _, g := range p.Groups {
		if !voiceGroupIDRe.MatchString(g.ID) || ids[g.ID] {
			return errors.New("分組 id 只能用小寫英文、數字、底線、連字號（英文開頭，最多 16 字），而且不能重複")
		}
		ids[g.ID] = true
		if len(g.Members) < 1 || len(g.Members) > 64 {
			return errors.New("每組 1–64 人")
		}
		for _, m := range g.Members {
			if m == "" || members[m] {
				return errors.New("組員不得為空字串，一個人最多在一組")
			}
			members[m] = true
		}
	}
	listens := map[string]int{}
	for _, g := range p.Groups {
		if len(g.Listeners) > 64 {
			return errors.New("每組旁聽最多 64 人")
		}
		seen := map[string]bool{}
		for _, l := range g.Listeners {
			if l == "" || seen[l] || slices.Contains(g.Members, l) {
				return errors.New("旁聽名單不得為空字串、不得重複，也不能和同一組的組員重疊")
			}
			seen[l] = true
			if listens[l]++; listens[l] > VoiceMaxListenGroups {
				return fmt.Errorf("一個人最多旁聽 %d 組", VoiceMaxListenGroups)
			}
		}
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
	// Entitlements 是這位玩家在這款遊戲擁有的永久解鎖 SKU（平台點數，docs/points.md）。
	// 平台是權益的唯一來源：不要快取超過一個房間的時間
	Entitlements []string `json:"entitlements,omitempty"`
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
	return c.doQuery(ctx, method, path, nil, data, idempotencyKey, out)
}

// doQuery 和 do 相同，另外帶查詢字串（查詢字串也在簽章範圍內）。
func (c *Client) doQuery(ctx context.Context, method, path string, query url.Values, data []byte, idempotencyKey string, out any) error {
	now := time.Now
	if c.Now != nil {
		now = c.Now
	}
	headers := SignRequest(SignableRequest{Method: method, Path: path, Query: query, Body: data},
		c.KeyID, c.PrivateKey, now().UnixMilli(), NewNonce())
	target := strings.TrimRight(c.BaseURL, "/") + path
	if len(query) > 0 {
		target += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, method, target, bytes.NewReader(data))
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

// PitchPolicy 是遊戲伺服器送給平台的音高授權（docs/voice-chat.md「三之一」），和發言規則分開。
// Epoch 必須是房間目前的局數（兌換啟動碼回傳的 epoch；房主重新開始後要重新授權）。
// Mode："ptt"（只在玩家有發言權且按住說話時分析）或 "hands_free"（授權期間持續分析，玩家不用加入語音、不用按住；
// 聲音不傳給其他人，平台另外取得玩家同意）。Audience 只有 "self"（只給唱歌者自己的畫面）。
// 有效期間最多 1 小時；名單 1–64 人。
type PitchPolicy struct {
	RoomID              string   `json:"roomId"`
	GameInstanceID      string   `json:"gameInstanceId"`
	Epoch               int      `json:"epoch"`
	Sequence            int64    `json:"sequence"`
	AllowedPlayerIDs    []string `json:"allowedPlayerIds"`
	Mode                string   `json:"mode"`
	Audience            string   `json:"audience"`
	EffectiveAtServerMs int64    `json:"effectiveAtServerMs"`
	ExpiresAtServerMs   int64    `json:"expiresAtServerMs"`
}

// Validate 和 packages/protocol 的 pitchPolicySchema 規則相同。
func (p PitchPolicy) Validate() error {
	if p.RoomID == "" || p.GameInstanceID == "" {
		return errors.New("roomId 與 gameInstanceId 必填")
	}
	if p.Epoch < 0 || p.Sequence < 0 {
		return errors.New("epoch 與 sequence 不得為負數")
	}
	if p.EffectiveAtServerMs <= 0 || p.ExpiresAtServerMs <= p.EffectiveAtServerMs {
		return errors.New("expiresAtServerMs 必須晚於 effectiveAtServerMs")
	}
	if p.Mode != "ptt" && p.Mode != "hands_free" {
		return fmt.Errorf("未知的模式：%q（只有 ptt、hands_free）", p.Mode)
	}
	if p.Audience != "self" {
		return fmt.Errorf("未知的對象：%q（目前只有 self）", p.Audience)
	}
	n := len(p.AllowedPlayerIDs)
	if n < 1 || n > 64 {
		return errors.New("名單要 1–64 人")
	}
	seen := make(map[string]bool, n)
	for _, id := range p.AllowedPlayerIDs {
		if id == "" || seen[id] {
			return errors.New("名單不得為空字串或重複")
		}
		seen[id] = true
	}
	return nil
}

// SendPitchPolicy 送出音高授權；同一房間同一序號重送不會重複執行。
func (c *Client) SendPitchPolicy(ctx context.Context, p PitchPolicy) error {
	if err := p.Validate(); err != nil {
		return err
	}
	if c.Mock {
		return nil
	}
	key := fmt.Sprintf("pitch:%s:%d", p.RoomID, p.Sequence)
	return c.post(ctx, "/v1/pitch/policy", p, key, nil)
}

// SpeechPolicy 是遊戲伺服器送給平台的語音指令授權（docs/speech-commands.md），和發言規則、音高授權分開。
// 玩家用說的操作（例如「四個五」「投三號」）：平台在玩家手機上辨識，只把文字交給這位玩家自己的遊戲畫面。
// Epoch 必須是房間目前的局數（房主重新開始後要重新授權）。Mode 是 "ptt" 或 "hands_free"（和 PitchPolicy 相同）、Audience 只有 "self"、
// Language 只有 "zh-TW"。有效期間最多 1 小時；名單 1–64 人。Vocabulary 選填（強烈建議提供，準確度差很多）。
type SpeechPolicy struct {
	RoomID              string            `json:"roomId"`
	GameInstanceID      string            `json:"gameInstanceId"`
	Epoch               int               `json:"epoch"`
	Sequence            int64             `json:"sequence"`
	AllowedPlayerIDs    []string          `json:"allowedPlayerIds"`
	Mode                string            `json:"mode"`
	Audience            string            `json:"audience"`
	Language            string            `json:"language"`
	Vocabulary          *SpeechVocabulary `json:"vocabulary,omitempty"`
	EffectiveAtServerMs int64             `json:"effectiveAtServerMs"`
	ExpiresAtServerMs   int64             `json:"expiresAtServerMs"`
}

// SpeechVocabulary 是語音指令的詞彙表：Phrases 是句型（詞之間用空白分開，{名稱} 是欄位，例如 "{n} 個 {m} 齋"），
// Slots 定義欄位。Strict 為 true 時平台只回符合詞彙表的結果；false 時只用來偏重。展開後最多 SpeechMaxPhrases 句。
type SpeechVocabulary struct {
	Phrases []string              `json:"phrases"`
	Slots   map[string]SpeechSlot `json:"slots,omitempty"`
	Strict  bool                  `json:"strict"`
}

// SpeechSlot 是句型裡的欄位。Type "integer"：Min–Max 的整數（平台負責念法，例如「二十三」「兩」）；
// Type "enum"：Values 裡的詞（例如「碰」「槓」「胡」）。
type SpeechSlot struct {
	Type   string   `json:"type"`
	Min    *int     `json:"min,omitempty"`
	Max    *int     `json:"max,omitempty"`
	Values []string `json:"values,omitempty"`
}

// SpeechIntegerSlot 建立整數欄位。
func SpeechIntegerSlot(min, max int) SpeechSlot {
	return SpeechSlot{Type: "integer", Min: &min, Max: &max}
}

// SpeechEnumSlot 建立固定幾個詞的欄位。
func SpeechEnumSlot(values ...string) SpeechSlot { return SpeechSlot{Type: "enum", Values: values} }

// SpeechMaxPhrases 是詞彙表展開後的句數上限。
const SpeechMaxPhrases = 5000

var (
	speechSlotName = regexp.MustCompile(`^[a-z][a-z0-9_]{0,15}$`)
	speechSlotRef  = regexp.MustCompile(`\{([^{}]*)\}`)
)

func hasControl(s string) bool {
	return strings.ContainsFunc(s, func(r rune) bool { return r < 0x20 || r == 0x7f })
}

// Size 回傳詞彙表展開後有幾句；格式不對時回錯誤。規則和 packages/protocol 的 speechVocabularySchema 相同。
func (v SpeechVocabulary) Size() (int, error) {
	if len(v.Phrases) < 1 || len(v.Phrases) > 50 {
		return 0, errors.New("句型要 1–50 個")
	}
	if len(v.Slots) > 8 {
		return 0, errors.New("欄位最多 8 個")
	}
	sizes := make(map[string]int, len(v.Slots))
	for name, slot := range v.Slots {
		if !speechSlotName.MatchString(name) {
			return 0, fmt.Errorf("欄位名稱 %q 要是小寫英文開頭、最多 16 個字元", name)
		}
		switch slot.Type {
		case "integer":
			if slot.Min == nil || slot.Max == nil || len(slot.Values) != 0 || *slot.Min < 0 || *slot.Max > 9999 || *slot.Max < *slot.Min {
				return 0, fmt.Errorf("整數欄位 %q 要有 0–9999 的 min、max，而且 max 不小於 min", name)
			}
			sizes[name] = *slot.Max - *slot.Min + 1
		case "enum":
			if slot.Min != nil || slot.Max != nil || len(slot.Values) < 1 || len(slot.Values) > 200 {
				return 0, fmt.Errorf("欄位 %q 的 values 要 1–200 個", name)
			}
			for _, val := range slot.Values {
				if val == "" || utf8.RuneCountInString(val) > 20 || hasControl(val) || strings.ContainsFunc(val, unicode.IsSpace) || strings.ContainsAny(val, "{}") {
					return 0, fmt.Errorf("欄位 %q 的值 %q 不能是空的、超過 20 個字，或含空白與大括號", name, val)
				}
			}
			sizes[name] = len(slot.Values)
		default:
			return 0, fmt.Errorf("欄位 %q 的 type 只能是 integer 或 enum", name)
		}
	}
	total := 0
	seen := make(map[string]bool, len(v.Phrases))
	for _, phrase := range v.Phrases {
		if phrase == "" || utf8.RuneCountInString(phrase) > 60 || hasControl(phrase) {
			return 0, fmt.Errorf("句型 %q 不能是空的、超過 60 個字或含控制字元", phrase)
		}
		if seen[phrase] {
			return 0, fmt.Errorf("句型 %q 重複", phrase)
		}
		seen[phrase] = true
		n := 1
		for _, m := range speechSlotRef.FindAllStringSubmatch(phrase, -1) {
			size, ok := sizes[m[1]]
			if !ok {
				return 0, fmt.Errorf("句型 %q 用到沒有定義的欄位 %q", phrase, m[1])
			}
			if n *= size; n > SpeechMaxPhrases {
				return 0, fmt.Errorf("詞彙表展開後超過 %d 句", SpeechMaxPhrases)
			}
		}
		if strings.ContainsAny(speechSlotRef.ReplaceAllString(phrase, ""), "{}") {
			return 0, fmt.Errorf("句型 %q 的大括號沒有成對", phrase)
		}
		if total += n; total > SpeechMaxPhrases {
			return 0, fmt.Errorf("詞彙表展開後超過 %d 句", SpeechMaxPhrases)
		}
	}
	return total, nil
}

// Validate 和 packages/protocol 的 speechPolicySchema 規則相同。
func (p SpeechPolicy) Validate() error {
	base := PitchPolicy{RoomID: p.RoomID, GameInstanceID: p.GameInstanceID, Epoch: p.Epoch, Sequence: p.Sequence,
		AllowedPlayerIDs: p.AllowedPlayerIDs, Mode: p.Mode, Audience: p.Audience,
		EffectiveAtServerMs: p.EffectiveAtServerMs, ExpiresAtServerMs: p.ExpiresAtServerMs}
	if err := base.Validate(); err != nil {
		return err
	}
	if p.Language != "zh-TW" {
		return fmt.Errorf("未知的語言：%q（目前只有 zh-TW）", p.Language)
	}
	if p.Vocabulary != nil {
		if _, err := p.Vocabulary.Size(); err != nil {
			return err
		}
	}
	return nil
}

// SendSpeechPolicy 送出語音指令授權；同一房間同一序號重送不會重複執行。
func (c *Client) SendSpeechPolicy(ctx context.Context, p SpeechPolicy) error {
	if err := p.Validate(); err != nil {
		return err
	}
	if c.Mock {
		return nil
	}
	key := fmt.Sprintf("speech:%s:%d", p.RoomID, p.Sequence)
	return c.post(ctx, "/v1/speech/policy", p, key, nil)
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
