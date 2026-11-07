package aigateway

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Limits 是轉送的上限。
type Limits struct {
	MaxRequest  int64         // 請求內容
	FirstByte   time.Duration // 送出請求到收到上游回應標頭
	Idle        time.Duration // 回應內容兩次讀取之間
	Total       time.Duration // 整個請求
	MaxResponse int64         // 回應總大小
	MaxEvent    int           // 單一 SSE 事件（含所有行）
}

// DefaultLimits 是平台的設定值。
var DefaultLimits = Limits{
	MaxRequest:  1 << 20,
	FirstByte:   120 * time.Second,
	Idle:        60 * time.Second,
	Total:       5 * time.Minute,
	MaxResponse: 10 << 20,
	MaxEvent:    1 << 20,
}

// 斷路器：同一款遊戲＋服務連續失敗（上游 429、5xx、連不上）幾次就暫停多久。每台平台 API 各自計算。
const (
	breakerThreshold = 5
	breakerCooldown  = 15 * time.Second
)

// usageBufferMax：非串流回應只在這個大小以內才解析 token 用量（避免為了用量把大回應整個留在記憶體）。
const usageBufferMax = 1 << 20

// 回傳給遊戲的回應標頭（白名單）：不轉 Set-Cookie、上游的內部標頭等。
var responseHeaders = []string{"Content-Type", "Retry-After", "Request-Id", "X-Request-Id"}

// Gateway 轉送已經驗證過的請求。零值可用（使用 DefaultLimits 與固定的上游）。
type Gateway struct {
	Limits Limits
	// Client 呼叫上游；nil 時用 NewClient()。
	Client *http.Client
	// UpstreamOverride 只給測試用（把服務指到 httptest 的假上游）；正式程式一律用 providers.json 的固定網址。
	UpstreamOverride map[string]string
	Now              func() time.Time

	once    sync.Once
	mu      sync.Mutex
	breaker map[string]*breakerState
}

type breakerState struct {
	fails     int
	openUntil time.Time
}

var errRedirect = errors.New("AI 閘道不跟隨轉址")

// NewClient 是呼叫上游用的 client：不跟隨轉址、不用環境變數的代理、不自動解壓縮（要求 identity）、TLS 1.2 以上。
func NewClient() *http.Client {
	tr := &http.Transport{
		Proxy:                  nil,
		DialContext:            (&net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
		TLSHandshakeTimeout:    10 * time.Second,
		TLSClientConfig:        &tls.Config{MinVersion: tls.VersionTLS12},
		MaxIdleConns:           64,
		MaxIdleConnsPerHost:    32,
		IdleConnTimeout:        90 * time.Second,
		DisableCompression:     true,
		MaxResponseHeaderBytes: 64 << 10,
		ForceAttemptHTTP2:      true,
	}
	return &http.Client{Transport: tr, CheckRedirect: func(*http.Request, []*http.Request) error { return errRedirect }}
}

func (g *Gateway) init() {
	g.once.Do(func() {
		if g.Limits == (Limits{}) {
			g.Limits = DefaultLimits
		}
		if g.Client == nil {
			g.Client = NewClient()
		}
		if g.Now == nil {
			g.Now = time.Now
		}
		g.breaker = map[string]*breakerState{}
	})
}

// MaxRequest 是請求內容上限。
func (g *Gateway) MaxRequest() int64 {
	g.init()
	return g.Limits.MaxRequest
}

func (g *Gateway) upstream(p *Provider) string {
	if u, ok := g.UpstreamOverride[p.ID]; ok {
		return u
	}
	return p.Upstream
}

// BreakerOpen：這款遊戲＋服務是否在暫停中，回傳剩下的秒數。
func (g *Gateway) BreakerOpen(gameID, provider string) (bool, time.Duration) {
	g.init()
	g.mu.Lock()
	defer g.mu.Unlock()
	b := g.breaker[gameID+"\x00"+provider]
	if b == nil {
		return false, 0
	}
	if left := b.openUntil.Sub(g.Now()); left > 0 {
		return true, left
	}
	return false, 0
}

// Report 記錄一次上游結果：連續失敗達門檻就暫停。
func (g *Gateway) Report(gameID, provider string, failure bool) {
	g.init()
	g.mu.Lock()
	defer g.mu.Unlock()
	k := gameID + "\x00" + provider
	b := g.breaker[k]
	if !failure {
		delete(g.breaker, k)
		return
	}
	if b == nil {
		if len(g.breaker) > 10_000 {
			g.breaker = map[string]*breakerState{}
		}
		b = &breakerState{}
		g.breaker[k] = b
	}
	b.fails++
	if b.fails >= breakerThreshold {
		b.openUntil = g.Now().Add(breakerCooldown)
		b.fails = 0
	}
}

// Result 是一次轉送的結果（用量紀錄用；不含任何內容）。
type Result struct {
	Status       int // 回給遊戲的狀態碼（上游的狀態碼，或閘道自己的錯誤）
	Bytes        int64
	InputTokens  *int64
	OutputTokens *int64
	// Error 是閘道判斷的問題：upstream_timeout、upstream_error、upstream_too_large、event_too_large、bad_upstream_encoding、client_gone
	Error string
	// UpstreamFailure：計入斷路器（上游 429、5xx、連不上、逾時）
	UpstreamFailure bool
	// Aborted：回應標頭已經送出後才發現問題，呼叫端清理完要 panic(http.ErrAbortHandler) 中斷連線，讓遊戲知道回應不完整
	Aborted bool
}

// Forward 把請求送到固定的上游、換上開發者的金鑰，並把回應串流回去。
// 呼叫端已經驗證代號、額度，body 已經讀完（≤ MaxRequest）。key 只放在這個請求的標頭裡。
func (g *Gateway) Forward(w http.ResponseWriter, r *http.Request, req *Request, body []byte, key string) Result {
	g.init()
	lim := g.Limits
	ctx, cancel := context.WithTimeout(r.Context(), lim.Total)
	defer cancel()
	var timedOut atomic.Bool
	timer := time.AfterFunc(lim.FirstByte, func() { timedOut.Store(true); cancel() })
	defer timer.Stop()

	target := g.upstream(req.Provider) + req.Path
	if req.RawQuery != "" {
		target += "?" + req.RawQuery
	}
	out, err := http.NewRequestWithContext(ctx, http.MethodPost, target, bytes.NewReader(body))
	if err != nil {
		return g.fail(w, http.StatusBadGateway, "upstream_error", true)
	}
	out.Header = req.Headers.Clone()
	if req.Provider.KeyScheme != "" {
		out.Header.Set(req.Provider.KeyHeader, req.Provider.KeyScheme+" "+key)
	} else {
		out.Header.Set(req.Provider.KeyHeader, key)
	}
	out.Header.Set("Accept-Encoding", "identity")
	out.Header.Set("User-Agent", "playtogether-ai-gateway/1")
	res, err := g.Client.Do(out)
	if err != nil {
		switch {
		case r.Context().Err() != nil:
			return Result{Status: 499, Error: "client_gone"}
		case timedOut.Load() || errors.Is(err, context.DeadlineExceeded):
			return g.fail(w, http.StatusGatewayTimeout, "upstream_timeout", true)
		default:
			return g.fail(w, http.StatusBadGateway, "upstream_error", true) // 連不上、TLS 錯誤、上游轉址
		}
	}
	defer res.Body.Close()
	if ce := res.Header.Get("Content-Encoding"); ce != "" && !strings.EqualFold(ce, "identity") {
		return g.fail(w, http.StatusBadGateway, "bad_upstream_encoding", true)
	}
	if res.ContentLength > lim.MaxResponse {
		return g.fail(w, http.StatusBadGateway, "upstream_too_large", true)
	}
	timer.Reset(lim.Idle)
	idle := &idleReader{r: res.Body, reset: func() { timer.Reset(lim.Idle) }}

	for _, h := range responseHeaders {
		if v := res.Header.Get(h); v != "" {
			w.Header().Set(h, v)
		}
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(res.StatusCode)
	result := Result{Status: res.StatusCode, UpstreamFailure: res.StatusCode == http.StatusTooManyRequests || res.StatusCode >= 500}
	rc := http.NewResponseController(w)
	u := &usage{}
	mt, _, _ := mime.ParseMediaType(res.Header.Get("Content-Type"))
	var problem string
	if mt == "text/event-stream" {
		problem = copySSE(w, rc, idle, lim, &result.Bytes, u)
	} else {
		problem = copyRaw(w, rc, idle, lim, &result.Bytes, u, mt == "application/json")
	}
	if problem != "" {
		result.Aborted = true
		switch {
		case problem == "read" && r.Context().Err() != nil, problem == "write":
			result.Error = "client_gone"
		case problem == "read" && timedOut.Load(), problem == "read" && ctx.Err() != nil:
			result.Error, result.UpstreamFailure = "upstream_timeout", true
		case problem == "read":
			result.Error, result.UpstreamFailure = "upstream_error", true
		default:
			result.Error = problem
		}
	}
	result.InputTokens, result.OutputTokens = u.in, u.out
	return result
}

func (g *Gateway) fail(w http.ResponseWriter, status int, code string, upstream bool) Result {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_, _ = io.WriteString(w, `{"error":"`+code+`"}`+"\n")
	return Result{Status: status, Error: code, UpstreamFailure: upstream}
}

// idleReader 每次讀到資料就重設閒置計時器。
type idleReader struct {
	r     io.Reader
	reset func()
}

func (i *idleReader) Read(p []byte) (int, error) {
	n, err := i.r.Read(p)
	if n > 0 {
		i.reset()
	}
	return n, err
}

// copyRaw 原樣串流（逐段寫出並 flush）。回傳問題（空字串＝正常結束）：read、write、upstream_too_large。
func copyRaw(w io.Writer, rc *http.ResponseController, body io.Reader, lim Limits, total *int64, u *usage, isJSON bool) string {
	buf := make([]byte, 32<<10)
	var keep []byte
	for {
		n, err := body.Read(buf)
		if n > 0 {
			*total += int64(n)
			if *total > lim.MaxResponse {
				return "upstream_too_large"
			}
			if isJSON && len(keep)+n <= usageBufferMax {
				keep = append(keep, buf[:n]...)
			} else {
				isJSON, keep = false, nil
			}
			if _, werr := w.Write(buf[:n]); werr != nil {
				return "write"
			}
			_ = rc.Flush()
		}
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return "read"
		}
	}
	if isJSON {
		u.parse(keep)
	}
	return ""
}

// copySSE 逐行轉送 SSE，每個事件結束（空行）時 flush；限制單一事件大小與總大小，並從 data 讀出 token 用量。
// 很長的一行（超過讀取緩衝）分段讀，仍算在同一個事件裡。
func copySSE(w io.Writer, rc *http.ResponseController, body io.Reader, lim Limits, total *int64, u *usage) string {
	br := bufio.NewReaderSize(body, 64<<10)
	event := 0
	var data []byte
	partial, inData := false, false // partial：上一段還沒讀到行尾；inData：目前這一行是 data:
	for {
		line, err := br.ReadSlice('\n')
		if len(line) > 0 {
			*total += int64(len(line))
			event += len(line)
			if *total > lim.MaxResponse {
				return "upstream_too_large"
			}
			if event > lim.MaxEvent {
				return "event_too_large"
			}
			end := false
			if !partial {
				d, ok := bytes.CutPrefix(line, []byte("data:"))
				inData = ok
				if ok {
					data = append(data, d...)
				}
				end = err == nil && len(bytes.TrimRight(line, "\r\n")) == 0
			} else if inData {
				data = append(data, line...)
			}
			if _, werr := w.Write(line); werr != nil {
				return "write"
			}
			if end {
				if bytes.Contains(data, []byte("sage")) { // usage、usageMetadata
					u.parse(data)
				}
				data, event = data[:0], 0
				_ = rc.Flush()
			}
		}
		partial = errors.Is(err, bufio.ErrBufferFull)
		switch {
		case err == nil, partial:
		case errors.Is(err, io.EOF):
			_ = rc.Flush()
			return ""
		default:
			return "read"
		}
	}
}

// usage 是讀到的 token 數（各服務格式不同；串流時取看過的最大值，因為多數服務送的是累計值）。
type usage struct{ in, out *int64 }

func (u *usage) set(dst **int64, v any) {
	f, ok := v.(float64)
	if !ok || f < 0 || f > 1e12 {
		return
	}
	n := int64(f)
	if *dst == nil || n > **dst {
		*dst = &n
	}
}

func (u *usage) add(m map[string]any) {
	if m == nil {
		return
	}
	if us, ok := m["usage"].(map[string]any); ok {
		u.set(&u.in, us["input_tokens"])   // Anthropic、OpenAI Responses
		u.set(&u.out, us["output_tokens"]) //
		u.set(&u.in, us["prompt_tokens"])  // OpenAI Chat Completions、Embeddings
		u.set(&u.out, us["completion_tokens"])
	}
	if um, ok := m["usageMetadata"].(map[string]any); ok { // Gemini
		u.set(&u.in, um["promptTokenCount"])
		u.set(&u.out, um["candidatesTokenCount"])
		u.set(&u.in, um["totalTokens"]) // countTokens
	}
	for _, k := range []string{"message", "response"} { // Anthropic message_start、OpenAI Responses 串流的 response.completed
		if inner, ok := m[k].(map[string]any); ok {
			if us, ok := inner["usage"].(map[string]any); ok {
				u.add(map[string]any{"usage": us})
			}
		}
	}
	if n, ok := m["input_tokens"]; ok { // Anthropic count_tokens
		u.set(&u.in, n)
	}
}

func (u *usage) parse(b []byte) {
	b = bytes.TrimSpace(b)
	if len(b) == 0 || b[0] != '{' {
		return
	}
	var m map[string]any
	if json.Unmarshal(b, &m) == nil {
		u.add(m)
	}
}

// ErrKeyRejected：服務回應金鑰無效（401、403）。
var ErrKeyRejected = errors.New("AI 服務回應金鑰無效")

// CheckKey 呼叫服務的免費 API 一次，確認金鑰有效（設定金鑰時用）。回傳 ErrKeyRejected 或暫時性的錯誤。
// 不讀回應內容（只看狀態碼），金鑰只放在這個請求的標頭。
func (g *Gateway) CheckKey(ctx context.Context, provider, key string) error {
	g.init()
	p := Lookup(provider)
	if p == nil {
		return errors.New("未知的服務")
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	target := g.upstream(p) + p.Check.Path
	if p.Check.Query != "" {
		target += "?" + p.Check.Query
	}
	req, err := http.NewRequestWithContext(ctx, p.Check.Method, target, nil)
	if err != nil {
		return err
	}
	for k, v := range p.Check.Headers {
		req.Header.Set(k, v)
	}
	if p.KeyScheme != "" {
		req.Header.Set(p.KeyHeader, p.KeyScheme+" "+key)
	} else {
		req.Header.Set(p.KeyHeader, key)
	}
	req.Header.Set("Accept-Encoding", "identity")
	req.Header.Set("User-Agent", "playtogether-ai-gateway/1")
	res, err := g.Client.Do(req)
	if err != nil {
		return errors.New("連不到 AI 服務") // 不回傳原始錯誤（可能含網址）
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(res.Body, 64<<10))
	res.Body.Close()
	switch {
	case res.StatusCode == http.StatusOK:
		return nil
	case res.StatusCode == http.StatusUnauthorized || res.StatusCode == http.StatusForbidden:
		return ErrKeyRejected
	case res.StatusCode == http.StatusBadRequest && p.ID == "gemini":
		return ErrKeyRejected // Gemini 對無效金鑰回 400 API_KEY_INVALID
	default:
		return errors.New("AI 服務暫時無法確認金鑰")
	}
}
