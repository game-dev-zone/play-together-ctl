package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"testing"
)

func aiProject(mode, hosting string, ai ...string) *project {
	p := projectFrom(defaultPolicy(presetProfile("general"), "none"))
	p.GameServer = &gameServerConfig{Mode: mode, Hosting: hosting, AI: ai}
	return p
}

func runAI(p *project) levels {
	out := levels{}
	checkAI(p, func(level, format string, a ...any) { out[level] = append(out[level], sprintf(format, a...)) })
	return out
}

func TestCheckAI(t *testing.T) {
	if l := runAI(aiProject("playtogether-managed", "playtogether")); len(l) != 0 {
		t.Fatalf("沒有申報不應有任何訊息：%v", l)
	}
	if l := runAI(aiProject("playtogether-managed", "playtogether", "anthropic", "openai")); !l.only() || len(l[levelInfo]) != 1 {
		t.Fatalf("正確的申報：%v", l)
	}
	if l := runAI(aiProject("playtogether-managed", "playtogether", "deepseek")); !l.only(levelSchema) {
		t.Fatalf("未知的服務要是 schema 錯誤：%v", l)
	}
	if l := runAI(aiProject("playtogether-managed", "playtogether", "openai", "openai")); !l.only(levelSchema) {
		t.Fatalf("重複的服務要是 schema 錯誤：%v", l)
	}
	if l := runAI(aiProject("none", "", "gemini")); !l.only(levelSchema) {
		t.Fatalf("沒有遊戲伺服器不能申報 AI：%v", l)
	}
	if l := runAI(aiProject("external", "", "gemini")); !l.only() || !strings.Contains(strings.Join(l[levelInfo], ""), "不經過玩聚 AI 閘道") {
		t.Fatalf("外部伺服器：%v", l)
	}
}

// 申報 AI 服務是需要平台審核的項目（規則資料 gameServer.ai），沒有申報就不是。
func TestAIRequiresReview(t *testing.T) {
	if l := runPolicy(aiProject("playtogether-managed", "playtogether", "anthropic")); !l.only(levelReview) {
		t.Fatalf("申報 AI 要需要審核：%v", l)
	}
	if l := runPolicy(aiProject("playtogether-managed", "playtogether")); !l.only() {
		t.Fatalf("沒有申報不需要審核：%v", l)
	}
}

func TestDeclarationGameServerAI(t *testing.T) {
	decode := func(p *project) map[string]json.RawMessage {
		raw, ok := declarationBody(p)
		if !ok {
			t.Fatal("沒有宣告")
		}
		var m map[string]json.RawMessage
		if err := json.Unmarshal(raw, &m); err != nil {
			t.Fatal(err)
		}
		return m
	}
	if m := decode(aiProject("playtogether-managed", "playtogether")); m["gameServer"] != nil {
		t.Fatalf("沒有申報就不送 gameServer：%s", m["gameServer"])
	}
	m := decode(aiProject("playtogether-managed", "playtogether", "anthropic", "gemini"))
	if string(m["gameServer"]) != `{"ai":["anthropic","gemini"]}` {
		t.Fatalf("gameServer 區塊：%s", m["gameServer"])
	}
	if len(m) != 6 {
		t.Fatalf("宣告要是五個區塊加 gameServer：%d", len(m))
	}
}

// 本機 AI 閘道：代號、申報、金鑰、請求檢查和平台相同，金鑰換成開發者的環境變數，代號不會送到上游。
func TestDevAIGateway(t *testing.T) {
	var gotKey, gotToken, gotPath, gotCF string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey, gotToken, gotPath, gotCF = r.Header.Get("X-Api-Key"), r.Header.Get("Authorization"), r.URL.Path, r.Header.Get("Cf-Ray")
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"id":"msg_1","usage":{"input_tokens":3,"output_tokens":5}}`)
	}))
	defer upstream.Close()

	p := aiProject("playtogether-managed", "playtogether", "anthropic")
	fp := newFakePlatform(nil)
	platform, srv, err := startFakePlatform(fp, "")
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()
	env, cleanup, err := attachDevAI(fp, p, platform)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	d := fp.ai.(*devAI)
	d.gw.UpstreamOverride = map[string]string{"anthropic": upstream.URL, "openai": upstream.URL}
	keys := map[string]string{}
	d.getenv = func(k string) string { return keys[k] }

	var base, tokenFile string
	for _, kv := range env {
		k, v, _ := strings.Cut(kv, "=")
		switch k {
		case "PT_AI_URL":
			base = v
		case "PT_AI_TOKEN_FILE":
			tokenFile = v
		}
	}
	if base != platform+"/v1/ai" || tokenFile == "" {
		t.Fatalf("環境變數：%v", env)
	}
	if !slices.Contains(env, "ANTHROPIC_API_KEY=") || !slices.Contains(env, "OPENAI_API_KEY=") {
		t.Fatalf("要清空遊戲伺服器的金鑰環境變數：%v", env)
	}
	raw, err := os.ReadFile(tokenFile)
	if err != nil {
		t.Fatal(err)
	}
	token := strings.TrimSpace(string(raw))
	if !strings.HasPrefix(token, "ptai_") || len(token) != 48 {
		t.Fatalf("代號格式：%q", token)
	}

	call := func(path, tok string, hdr map[string]string) (int, map[string]string) {
		req, _ := http.NewRequest("POST", base+path, strings.NewReader(`{"model":"x"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Api-Key", tok)
		for k, v := range hdr {
			req.Header.Set(k, v)
		}
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		var m map[string]string
		b, _ := io.ReadAll(res.Body)
		_ = json.Unmarshal(b, &m)
		return res.StatusCode, m
	}

	if st, m := call("/anthropic/v1/messages", "ptai_wrong", nil); st != 401 || m["error"] != "invalid_token" {
		t.Fatalf("錯的代號：%d %v", st, m)
	}
	if st, m := call("/openai/v1/chat/completions", token, nil); st != 403 || m["error"] != "provider_not_declared" {
		t.Fatalf("沒有申報的服務：%d %v", st, m)
	}
	if st, m := call("/anthropic/v1/messages", token, nil); st != 503 || m["error"] != "ai_key_missing" || !strings.Contains(m["message"], "ANTHROPIC_API_KEY") {
		t.Fatalf("沒有設定金鑰：%d %v", st, m)
	}
	if st, m := call("/anthropic/v1/files", token, nil); st != 404 || m["error"] != "unknown_path" {
		t.Fatalf("沒有開放的 API：%d %v", st, m)
	}
	if st, m := call("/anthropic/v1/messages", token, map[string]string{"Cookie": "a=b"}); st != 400 || m["header"] != "cookie" {
		t.Fatalf("不轉送的標頭：%d %v", st, m)
	}
	if gotKey != "" {
		t.Fatal("被拒絕的請求不應送到上游")
	}

	keys["ANTHROPIC_API_KEY"] = "sk-ant-developer"
	st, m := call("/anthropic/v1/messages", token, map[string]string{"Anthropic-Version": "2023-06-01", "Cf-Ray": "abc"})
	if st != 200 || m["id"] != "msg_1" {
		t.Fatalf("轉送：%d %v", st, m)
	}
	if gotKey != "sk-ant-developer" || gotToken != "" || gotPath != "/v1/messages" || gotCF != "" {
		t.Fatalf("上游收到的：key=%q auth=%q path=%q cf=%q", gotKey, gotToken, gotPath, gotCF)
	}
	calls := fp.callsSince(0)
	if len(calls) == 0 || calls[len(calls)-1].Endpoint != "/v1/ai/anthropic/v1/messages" || calls[len(calls)-1].Status != 200 {
		t.Fatalf("控制面板的呼叫紀錄：%+v", calls)
	}
	for _, c := range calls {
		if strings.Contains(c.Endpoint+c.Result+c.Hint, token) || strings.Contains(c.Endpoint+c.Result+c.Hint, "sk-ant") {
			t.Fatal("呼叫紀錄不能有代號或金鑰")
		}
	}
}

// 沒有遊戲伺服器的專案不掛閘道，/v1/ai/ 回 404。
func TestDevAINoServer(t *testing.T) {
	fp := newFakePlatform(nil)
	env, cleanup, err := attachDevAI(fp, aiProject("none", ""), "http://127.0.0.1:1")
	if err != nil || env != nil || fp.ai != nil {
		t.Fatalf("不應掛上閘道：%v %v", env, err)
	}
	cleanup()
	w := httptest.NewRecorder()
	fp.handler(nil).ServeHTTP(w, httptest.NewRequest("POST", "/v1/ai/anthropic/v1/messages", nil))
	if w.Code != 404 {
		t.Fatalf("應為 404：%d", w.Code)
	}
}
