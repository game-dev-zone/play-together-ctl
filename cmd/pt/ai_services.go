package main

// 玩聚 AI 閘道（遊戲伺服器使用 Anthropic、OpenAI、Gemini）：
//   - playtogether.json 的 gameServer.ai 申報使用哪些服務（pt check 檢查、pt upload 放進宣告快照，需要平台審核）；
//   - pt dev 提供本機閘道：和平台同一套請求檢查（internal/aigateway），金鑰用開發者自己的環境變數，
//     遊戲伺服器拿到的環境變數（PT_AI_URL、PT_AI_TOKEN_FILE）和玩聚代管時相同，開發時不用改程式。

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/game-dev-zone/play-together-ctl/internal/aigateway"
)

// aiKeyEnv：pt dev 的本機閘道從哪個環境變數讀開發者的金鑰（各服務官方 SDK 的慣用名稱）。
var aiKeyEnv = map[string][]string{
	"anthropic": {"ANTHROPIC_API_KEY"},
	"openai":    {"OPENAI_API_KEY"},
	"gemini":    {"GEMINI_API_KEY", "GOOGLE_API_KEY"},
}

// aiProviders 是申報的 AI 服務（沒有申報時是 nil）。
func (p *project) aiProviders() []string {
	if p == nil || p.GameServer == nil {
		return nil
	}
	return p.GameServer.AI
}

// checkAI 是 pt check 的 AI 服務部分（需要審核的提示由規則資料 rules.json 的 gameServer.ai 產生）。
func checkAI(p *project, add addFunc) {
	ai := p.aiProviders()
	if len(ai) == 0 {
		return
	}
	seen := map[string]bool{}
	for _, id := range ai {
		switch {
		case !aigateway.Known(id):
			add(levelSchema, "playtogether.json 的 gameServer.ai 有未知的服務 %q（可用：%s）", id, strings.Join(aigateway.IDs(), "、"))
		case seen[id]:
			add(levelSchema, "playtogether.json 的 gameServer.ai 有重複的服務 %q", id)
		}
		seen[id] = true
	}
	switch {
	case p.gameServerMode() == "none":
		add(levelSchema, "gameServer.ai 只能用在遊戲伺服器：gameServer.mode 是 none（沒有遊戲伺服器），畫面端不能直接呼叫 AI 服務（金鑰會外洩）")
	case !p.hostedServer():
		add(levelInfo, "遊戲伺服器不是玩聚代管：AI 服務由你的伺服器自己連線（不經過玩聚 AI 閘道），申報與審核相同")
	default:
		add(levelInfo, "上架前在開發者頁「AI 服務」設定金鑰（上架用、預覽用分開；審核中的版本只能用預覽用金鑰）")
	}
}

// devAI 是 pt dev 的本機 AI 閘道（掛在假平台的 /v1/ai/ 底下）。
type devAI struct {
	token    string
	declared []string
	gw       *aigateway.Gateway
	getenv   func(string) string
	log      func(endpoint string, status int, result, hint string)
}

// newDevAI 產生本機 AI 代號，寫成檔案（和代管時的 /run/pt/ai-token 一樣只放代號）。回傳閘道與遊戲伺服器要的環境變數。
func newDevAI(p *project, platform string) (*devAI, []string, func(), error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return nil, nil, nil, err
	}
	token := "ptai_" + base64.RawURLEncoding.EncodeToString(raw)
	dir, err := os.MkdirTemp("", "pt-dev-ai-")
	if err != nil {
		return nil, nil, nil, err
	}
	file := filepath.Join(dir, "ai-token")
	if err := os.WriteFile(file, []byte(token+"\n"), 0o600); err != nil {
		os.RemoveAll(dir)
		return nil, nil, nil, err
	}
	d := &devAI{token: token, declared: p.aiProviders(), gw: &aigateway.Gateway{}, getenv: os.Getenv}
	env := []string{"PT_AI_URL=" + platform + aigateway.Prefix[:len(aigateway.Prefix)-1], "PT_AI_TOKEN_FILE=" + file}
	return d, env, func() { os.RemoveAll(dir) }, nil
}

// keyFor 回傳開發者在環境變數設定的金鑰與變數名稱。
func (d *devAI) keyFor(provider string) (key, name string) {
	names := aiKeyEnv[provider]
	for _, n := range names {
		if v := strings.TrimSpace(d.getenv(n)); v != "" {
			return v, n
		}
	}
	if len(names) > 0 {
		return "", names[0]
	}
	return "", ""
}

// keyStatus 是 pt dev 啟動時的提示：每個申報的服務有沒有設定金鑰。
func (d *devAI) keyStatus() []string {
	var out []string
	for _, id := range d.declared {
		if key, name := d.keyFor(id); key != "" {
			out = append(out, fmt.Sprintf("%s：使用 %s", id, name))
		} else if name != "" {
			out = append(out, fmt.Sprintf("%s：沒有設定 %s（呼叫會回 503 ai_key_missing）", id, name))
		}
	}
	return out
}

func (d *devAI) fail(w http.ResponseWriter, endpoint string, status int, code, hint string, extra map[string]string) {
	body := map[string]string{"error": code}
	for k, v := range extra {
		body[k] = v
	}
	if hint != "" {
		body["message"] = hint // 只有本機閘道會附說明；平台只回錯誤代碼
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
	if d.log != nil {
		d.log(endpoint, status, code, hint)
	}
}

var aiParseHints = map[string]string{
	"bad_path":           "路徑格式不對（不能有 %、反斜線、重複斜線、. 或 ..）",
	"unknown_path":       "玩聚 AI 閘道沒有開放這個服務或 API（只開放文字生成、計算 token 等；檔案上傳、批次、微調不開放）",
	"method_not_allowed": "只接受 POST",
	"bad_query":          "不接受這個 query 參數（金鑰不能放在網址；Gemini 串流只接受 alt=sse）",
	"unknown_header":     "不轉送這個標頭（帳號範圍、身分、Cookie 等）：請移除",
	"bad_header":         "標頭的值格式不對或重複",
	"invalid_token":      "AI 代號不對：用 SDK 的 AI 小工具讀 PT_AI_TOKEN_FILE，當成官方 SDK 的 apiKey",
	"bad_content_type":   "Content-Type 要是 application/json",
}

func (d *devAI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	endpoint, _, _ := strings.Cut(r.RequestURI, "?")
	req, perr := aigateway.Parse(r)
	if perr != nil {
		var extra map[string]string
		if perr.Header != "" {
			extra = map[string]string{"header": perr.Header}
		}
		d.fail(w, endpoint, perr.Status, perr.Code, aiParseHints[perr.Code], extra)
		return
	}
	provider := req.Provider.ID
	if subtle.ConstantTimeCompare([]byte(req.Token), []byte(d.token)) != 1 {
		d.fail(w, endpoint, http.StatusUnauthorized, "invalid_token", aiParseHints["invalid_token"], nil)
		return
	}
	if r.ContentLength > d.gw.MaxRequest() {
		d.fail(w, endpoint, http.StatusRequestEntityTooLarge, "too_large", "請求內容不能超過 1 MB", nil)
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, d.gw.MaxRequest()))
	if err != nil {
		var mbe *http.MaxBytesError
		if errors.As(err, &mbe) {
			d.fail(w, endpoint, http.StatusRequestEntityTooLarge, "too_large", "請求內容不能超過 1 MB", nil)
		} else {
			d.fail(w, endpoint, http.StatusBadRequest, "bad_request", "", nil)
		}
		return
	}
	if len(body) == 0 {
		d.fail(w, endpoint, http.StatusBadRequest, "bad_request", "請求內容是空的", nil)
		return
	}
	if !slices.Contains(d.declared, provider) {
		d.fail(w, endpoint, http.StatusForbidden, "provider_not_declared",
			fmt.Sprintf("playtogether.json 的 gameServer.ai 沒有申報 %s：加上後重新啟動 pt dev（上架時一樣會被拒絕）", provider), nil)
		return
	}
	key, name := d.keyFor(provider)
	if key == "" {
		d.fail(w, endpoint, http.StatusServiceUnavailable, "ai_key_missing",
			fmt.Sprintf("本機開發要在啟動 pt dev 的終端機設定 %s（你自己的金鑰；上架後改用開發者頁設定的金鑰）", name), nil)
		return
	}
	res := d.gw.Forward(w, r, req, body, key)
	if d.log != nil {
		result := "ok"
		if res.Error != "" {
			result = res.Error
		} else if res.Status >= 400 {
			result = fmt.Sprintf("上游回 %d", res.Status)
		}
		d.log(endpoint, res.Status, result, "")
	}
	if res.Aborted {
		panic(http.ErrAbortHandler) // 回應標頭已經送出：中斷連線，讓遊戲知道回應不完整（和平台相同）
	}
}
