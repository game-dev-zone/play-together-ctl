// Package aigateway 是玩聚 AI 閘道：代管遊戲伺服器經平台 API 代為呼叫 AI 服務。
//
// 這個套件只處理 HTTP 層：服務清單（providers.json）、路徑與標頭白名單、轉送、串流與上限、讀出 token 用量。
// 代號驗證、額度、金鑰解密、用量紀錄由呼叫端處理（平台 API 要查資料庫；pt dev 的本機閘道用開發者的環境變數金鑰）。
//
// 安全原則：
//   - 上游網址固定（providers.json），不接受遊戲指定網址、Host、金鑰或身分標頭；不跟隨轉址。
//   - 只轉送清單上的方法、正規化後的路徑、query 與標頭；編碼字元、`.`、`..`、重複斜線、未知項目一律拒絕。
//   - 請求與回應內容、金鑰、代號都不寫進紀錄。
package aigateway

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"slices"
	"strings"
)

//go:embed providers.json
var providersJSON []byte

// Endpoint 是允許的一個「方法＋路徑」。路徑的段落可以是 `{參數}` 開頭再接固定的字尾（例如 `{model}:generateContent`）。
type Endpoint struct {
	Method string              `json:"method"`
	Path   string              `json:"path"`
	Query  map[string][]string `json:"query"` // 允許的 query 與它的值（每個鍵最多出現一次）
}

// Check 是設定金鑰時呼叫一次的免費 API（確認金鑰有效）。
type Check struct {
	Method  string            `json:"method"`
	Path    string            `json:"path"`
	Query   string            `json:"query"`
	Headers map[string]string `json:"headers"`
}

// Provider 是一個 AI 服務。
type Provider struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Upstream string `json:"upstream"`
	// KeyHeader、KeyScheme：開發者的金鑰放在哪個標頭（KeyScheme 為 Bearer 時寫成 "Bearer <金鑰>"）
	KeyHeader      string            `json:"keyHeader"`
	KeyScheme      string            `json:"keyScheme"`
	Params         map[string]string `json:"params"` // 路徑參數的格式（正規表示式）
	Endpoints      []Endpoint        `json:"endpoints"`
	RequestHeaders []string          `json:"requestHeaders"` // 這個服務另外允許轉送的請求標頭（小寫）
	Check          Check             `json:"check"`

	params map[string]*regexp.Regexp
}

type providerFile struct {
	Version   string     `json:"version"`
	Providers []Provider `json:"providers"`
}

var catalog = func() providerFile {
	var f providerFile
	if err := json.Unmarshal(providersJSON, &f); err != nil {
		panic("aigateway/providers.json 格式錯誤：" + err.Error())
	}
	for i := range f.Providers {
		p := &f.Providers[i]
		u, err := url.Parse(p.Upstream)
		if err != nil || u.Scheme != "https" || u.Host == "" || u.Path != "" || u.RawQuery != "" || u.User != nil {
			panic("aigateway/providers.json：上游網址必須是 https://主機：" + p.ID)
		}
		p.params = map[string]*regexp.Regexp{}
		for k, v := range p.Params {
			p.params[k] = regexp.MustCompile(v)
		}
		for _, e := range p.Endpoints {
			if e.Method != "POST" || !strings.HasPrefix(e.Path, "/") {
				panic("aigateway/providers.json：端點格式錯誤：" + p.ID + " " + e.Path)
			}
		}
	}
	return f
}()

// Version 是服務清單的版本。
func Version() string { return catalog.Version }

// Providers 回傳所有服務（依清單順序）。
func Providers() []Provider { return slices.Clone(catalog.Providers) }

// Lookup 找服務；找不到回 nil。
func Lookup(id string) *Provider {
	for i := range catalog.Providers {
		if catalog.Providers[i].ID == id {
			return &catalog.Providers[i]
		}
	}
	return nil
}

// Known 是不是清單上的服務代號（申報 gameServer.ai 時檢查）。
func Known(id string) bool { return Lookup(id) != nil }

// IDs 回傳所有服務代號。
func IDs() []string {
	out := make([]string, 0, len(catalog.Providers))
	for _, p := range catalog.Providers {
		out = append(out, p.ID)
	}
	return out
}

// segmentPattern：路徑段落允許的字元（Gemini 的 `gemini-2.5-flash:generateContent` 需要 `.` 與 `:`）。
var segmentPattern = regexp.MustCompile(`^[A-Za-z0-9._:-]{1,100}$`)

// matchEndpoint 用正規化後的段落比對允許的端點，回傳端點與用量紀錄用的路徑樣板（參數換成 {名稱}）。
func (p *Provider) matchEndpoint(segs []string) (*Endpoint, bool) {
	for i := range p.Endpoints {
		e := &p.Endpoints[i]
		tpl := strings.Split(strings.TrimPrefix(e.Path, "/"), "/")
		if len(tpl) != len(segs) {
			continue
		}
		ok := true
		for j, t := range tpl {
			if !p.matchSegment(t, segs[j]) {
				ok = false
				break
			}
		}
		if ok {
			return e, true
		}
	}
	return nil, false
}

func (p *Provider) matchSegment(tpl, seg string) bool {
	if !strings.HasPrefix(tpl, "{") {
		return tpl == seg
	}
	end := strings.IndexByte(tpl, '}')
	if end < 0 {
		return false
	}
	name, suffix := tpl[1:end], tpl[end+1:]
	re := p.params[name]
	if re == nil || !strings.HasSuffix(seg, suffix) {
		return false
	}
	val := strings.TrimSuffix(seg, suffix)
	// 參數本身不能含有字尾用的分隔字元（例如 model 裡不能再有 `:`），避免 `a:generateContent:countTokens` 之類的混淆
	return val != "" && !strings.ContainsAny(val, ":/") && re.MatchString(val)
}

// ParseError 是請求不符合服務清單的原因（回給遊戲的錯誤碼與 HTTP 狀態）。
type ParseError struct {
	Status int
	Code   string
	Header string // unknown_header、bad_header：哪一個標頭（名稱不是秘密，回給開發者方便除錯）
}

func (e *ParseError) Error() string { return fmt.Sprintf("%d %s", e.Status, e.Code) }

func perr(status int, code string) *ParseError { return &ParseError{Status: status, Code: code} }
