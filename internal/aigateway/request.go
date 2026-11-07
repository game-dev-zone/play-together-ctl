package aigateway

import (
	"mime"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strings"
)

// Prefix 是閘道的路由前綴：POST /v1/ai/{provider}/{path…}。
const Prefix = "/v1/ai/"

// Request 是通過服務清單檢查的請求。
type Request struct {
	Provider *Provider
	Endpoint *Endpoint
	// Path 是轉送給上游的路徑（例如 /v1/messages）；Template 是用量紀錄用的樣板（參數換成 {名稱}）
	Path, Template string
	// RawQuery 是重新編碼的 query（只有允許的鍵）
	RawQuery string
	// Headers 是要轉送的請求標頭（白名單）；金鑰標頭由 Forward 換上開發者的金鑰
	Headers http.Header
	// Token 是遊戲送來的 AI 代號（x-api-key、Authorization: Bearer 或 x-goog-api-key）；不能寫進任何紀錄
	Token string
}

// 遊戲送 AI 代號的標頭（官方 SDK 各自的寫法）。這些標頭一律不轉送。
var tokenHeaders = []string{"x-api-key", "authorization", "x-goog-api-key"}

// 所有服務都轉送的標頭。
var commonHeaders = []string{"content-type", "accept"}

// 可以忽略（不轉送、也不拒絕）的標頭：連線層、官方 SDK 自動加的環境資訊，
// 以及平台自己的反向代理（Cloudflare Tunnel、Caddy）一定會加上的轉送資訊。這些都不會送到 AI 服務。
var ignoredHeaders = []string{"user-agent", "accept-encoding", "content-length", "connection", "keep-alive", "te", "x-goog-api-client",
	"x-forwarded-for", "x-forwarded-proto", "x-forwarded-host", "x-real-ip", "via", "cdn-loop", "true-client-ip"}

// x-stainless-：OpenAI、Anthropic 官方 SDK 的環境資訊（作業系統、重試次數等）；cf-：Cloudflare 加的（cf-ray、cf-warp-tag-id…）
var ignoredPrefixes = []string{"x-stainless-", "cf-"}

var headerValuePatterns = map[string]*regexp.Regexp{
	"anthropic-version": regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}$`),
	"anthropic-beta":    regexp.MustCompile(`^[a-z0-9-]{1,80}( *, *[a-z0-9-]{1,80}){0,9}$`),
	"accept":            regexp.MustCompile(`^[A-Za-z0-9*/+.,;= -]{1,200}$`),
}

// Parse 用服務清單檢查請求的方法、路徑、query、標頭與 Content-Type（不讀內容）。
// 路徑檢查用原始的請求目標（RequestURI），不用解碼、清理過的 r.URL.Path，避免編碼字元或 `..` 被正規化後通過。
func Parse(r *http.Request) (*Request, *ParseError) {
	if r.Method == http.MethodConnect || !strings.HasPrefix(r.RequestURI, "/") || r.URL.IsAbs() {
		return nil, perr(http.StatusBadRequest, "bad_path")
	}
	rawPath, rawQuery, _ := strings.Cut(r.RequestURI, "?")
	if !strings.HasPrefix(rawPath, Prefix) {
		return nil, perr(http.StatusNotFound, "unknown_path")
	}
	// 編碼字元（%2F、%2E…）、反斜線、#：一律拒絕，不嘗試解碼
	if strings.ContainsAny(rawPath, "%\\#") || r.URL.RawPath != "" || r.URL.Path != rawPath {
		return nil, perr(http.StatusBadRequest, "bad_path")
	}
	segs := strings.Split(rawPath[len(Prefix):], "/")
	for _, s := range segs {
		// 空段落（重複斜線、結尾斜線）、. 與 ..、不允許的字元
		if s == "" || s == "." || s == ".." || !segmentPattern.MatchString(s) {
			return nil, perr(http.StatusBadRequest, "bad_path")
		}
	}
	p := Lookup(segs[0])
	if p == nil || len(segs) < 2 {
		return nil, perr(http.StatusNotFound, "unknown_path")
	}
	ep, ok := p.matchEndpoint(segs[1:])
	if !ok {
		return nil, perr(http.StatusNotFound, "unknown_path")
	}
	if r.Method != ep.Method {
		return nil, perr(http.StatusMethodNotAllowed, "method_not_allowed")
	}
	req := &Request{Provider: p, Endpoint: ep, Path: "/" + strings.Join(segs[1:], "/"), Template: ep.Path, Headers: http.Header{}}

	// query：只允許端點列出的鍵與值，每個鍵一次（Gemini 的 ?key= 也在這裡被拒絕）
	if rawQuery != "" {
		if strings.ContainsAny(rawQuery, ";%+") {
			return nil, perr(http.StatusBadRequest, "bad_query")
		}
		vals, err := url.ParseQuery(rawQuery)
		if err != nil {
			return nil, perr(http.StatusBadRequest, "bad_query")
		}
		for k, vs := range vals {
			allowed, ok := ep.Query[k]
			if !ok || len(vs) != 1 || !slices.Contains(allowed, vs[0]) {
				return nil, perr(http.StatusBadRequest, "bad_query")
			}
		}
		req.RawQuery = vals.Encode()
	}

	// 標頭：AI 代號、白名單、可以忽略的；其他（帳號範圍、身分、轉送來源…）一律拒絕
	for name, vs := range r.Header {
		lower := strings.ToLower(name)
		switch {
		case slices.Contains(tokenHeaders, lower):
			if len(vs) != 1 {
				return nil, perr(http.StatusUnauthorized, "invalid_token")
			}
			t := vs[0]
			if lower == "authorization" {
				var ok bool
				if t, ok = strings.CutPrefix(t, "Bearer "); !ok {
					return nil, perr(http.StatusUnauthorized, "invalid_token")
				}
			}
			t = strings.TrimSpace(t)
			if req.Token != "" && req.Token != t {
				return nil, perr(http.StatusUnauthorized, "invalid_token") // 兩個標頭的代號不同
			}
			req.Token = t
		case slices.Contains(commonHeaders, lower) || slices.Contains(p.RequestHeaders, lower):
			if len(vs) != 1 || len(vs[0]) > 300 {
				return nil, &ParseError{Status: http.StatusBadRequest, Code: "bad_header", Header: lower}
			}
			if re := headerValuePatterns[lower]; re != nil && !re.MatchString(vs[0]) {
				return nil, &ParseError{Status: http.StatusBadRequest, Code: "bad_header", Header: lower}
			}
			req.Headers.Set(name, vs[0])
		case slices.Contains(ignoredHeaders, lower) || slices.ContainsFunc(ignoredPrefixes, func(p string) bool { return strings.HasPrefix(lower, p) }):
		default:
			return nil, &ParseError{Status: http.StatusBadRequest, Code: "unknown_header", Header: lower}
		}
	}
	if req.Token == "" {
		return nil, perr(http.StatusUnauthorized, "invalid_token")
	}
	mt, params, err := mime.ParseMediaType(req.Headers.Get("Content-Type"))
	if err != nil || mt != "application/json" {
		return nil, perr(http.StatusUnsupportedMediaType, "bad_content_type")
	}
	for k, v := range params {
		if k != "charset" || !strings.EqualFold(v, "utf-8") {
			return nil, perr(http.StatusUnsupportedMediaType, "bad_content_type")
		}
	}
	return req, nil
}
