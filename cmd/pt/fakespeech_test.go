package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/game-dev-zone/play-together-ctl/internal/playtogether"
)

func diceVocabulary() *playtogether.SpeechVocabulary {
	return &playtogether.SpeechVocabulary{Phrases: []string{"{n} 個 {m}", "{n} 個 {m} 齋", "開", "齋"},
		Slots: map[string]playtogether.SpeechSlot{"n": playtogether.SpeechIntegerSlot(1, 30), "m": playtogether.SpeechIntegerSlot(1, 6)}, Strict: true}
}

// 本機假平台的語音指令：授權、控制面板輸入的句子對回詞彙表、畫面輪詢、模擬停止。
func TestFakeSpeech(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	fp := newFakePlatform(pub)
	srv := httptest.NewServer(fp.handler(nil))
	t.Cleanup(srv.Close)
	c := &playtogether.Client{BaseURL: srv.URL, KeyID: fakeKeyID, PrivateKey: priv}
	ctx := context.Background()
	apiCode := func(err error) string {
		var e *playtogether.APIError
		if errors.As(err, &e) {
			return e.Code
		}
		if err != nil {
			return err.Error()
		}
		return ""
	}
	_, p1, roomID, err := fp.issue("p1")
	must(t, err)
	_, _, _, err = fp.issue("p2")
	must(t, err)

	// 畫面輪詢（像瀏覽器 SDK）：要帶本機的 Origin
	poll := func(player string, start bool, origin string) (int, map[string]any) {
		u := srv.URL + "/dev/speech?player=" + player
		if start {
			u += "&start=1"
		}
		req, _ := http.NewRequest("GET", u, nil)
		if origin != "" {
			req.Header.Set("Origin", origin)
		}
		res, err := http.DefaultClient.Do(req)
		must(t, err)
		defer res.Body.Close()
		var out map[string]any
		_ = json.NewDecoder(res.Body).Decode(&out)
		return res.StatusCode, out
	}
	const local = "http://127.0.0.1:8090"
	if st, _ := poll("p1", true, "https://evil.example"); st != 403 {
		t.Fatalf("別的網站不能問：%d", st)
	}
	if st, _ := poll("p1", true, ""); st != 403 {
		t.Fatalf("沒有 Origin 不能問：%d", st)
	}

	// 還沒授權：畫面一開始聽就被停止；控制面板也不能輸入
	if _, out := poll("p1", true, local); out["stop"] != "not_authorized" {
		t.Fatalf("沒有授權應該停止：%v", out)
	}
	if err := fp.say("p1", "四個五"); err == nil {
		t.Fatal("沒有授權不能輸入")
	}

	nowMs := time.Now().UnixMilli()
	policy := playtogether.SpeechPolicy{RoomID: roomID, GameInstanceID: "g1", Epoch: 0, Sequence: 1, AllowedPlayerIDs: []string{p1.ID},
		Mode: "ptt", Audience: "self", Language: "zh-TW", Vocabulary: diceVocabulary(), EffectiveAtServerMs: nowMs, ExpiresAtServerMs: nowMs + 60_000}
	must(t, c.SendSpeechPolicy(ctx, policy))
	must(t, c.SendSpeechPolicy(ctx, policy)) // 同序號同內容是重送
	changed := policy
	changed.Vocabulary = nil
	if err := c.SendSpeechPolicy(ctx, changed); apiCode(err) != "stale_sequence" {
		t.Fatalf("同序號不同內容：%v", err)
	}
	stale := policy
	stale.Sequence, stale.Epoch = 2, 7
	if err := c.SendSpeechPolicy(ctx, stale); apiCode(err) != "stale_epoch" {
		t.Fatalf("局數不對：%v", err)
	}
	gone := policy
	gone.RoomID, gone.Sequence = "r_nope", 3
	if err := c.SendSpeechPolicy(ctx, gone); apiCode(err) != "room_not_found" {
		t.Fatalf("沒有這個房間：%v", err)
	}

	// 已授權但畫面還沒開始聽
	if err := fp.say("p1", "四個五"); err == nil || !strings.Contains(err.Error(), "speech.start") {
		t.Fatalf("畫面還沒開始聽：%v", err)
	}
	if _, out := poll("p1", true, local); out["stop"] != nil || len(out["results"].([]any)) != 0 {
		t.Fatalf("開始聽：%v", out)
	}
	if s := fp.state(0); s.Players[0].Speech != "listening" || s.Players[1].Speech != "" {
		t.Fatalf("面板狀態：%+v", s.Players)
	}
	// 沒被授權的人
	if _, out := poll("p2", true, local); out["stop"] != "not_authorized" {
		t.Fatalf("p2 不在名單：%v", out)
	}

	must(t, fp.say("p1", "四個五"))
	must(t, fp.say("p1", " 兩個一齋。"))
	must(t, fp.say("p1", "12個6"))
	must(t, fp.say("p1", "三十一個六")) // 不在詞彙表（strict）→ 空的結果
	must(t, fp.say("p1", ""))      // 沒聽到
	_, out := poll("p1", false, local)
	raw, _ := json.Marshal(out["results"])
	// 重新編碼後鍵會照字母排序
	want := `[{"alternatives":[{"phrase":"{n} 個 {m}","slots":{"m":5,"n":4},"text":"四個五"}]},` +
		`{"alternatives":[{"phrase":"{n} 個 {m} 齋","slots":{"m":1,"n":2},"text":"兩個一齋。"}]},` +
		`{"alternatives":[{"phrase":"{n} 個 {m}","slots":{"m":6,"n":12},"text":"12個6"}]},` +
		`{"alternatives":[]},{"alternatives":[]}]`
	if string(raw) != want {
		t.Fatalf("結果：\n%s\n應該是：\n%s", raw, want)
	}
	if _, out := poll("p1", false, local); len(out["results"].([]any)) != 0 {
		t.Fatalf("拿過的不再給：%v", out)
	}

	// 模擬停止
	if err := fp.stopSpeech("p1", "whatever"); err == nil {
		t.Fatal("不認得的原因要拒絕")
	}
	must(t, fp.stopSpeech("p1", "mic_denied"))
	if _, out := poll("p1", false, local); out["stop"] != "mic_denied" {
		t.Fatalf("模擬停止：%v", out)
	}

	// 沒有詞彙表：原文照送，沒有 phrase
	free := policy
	free.Sequence, free.Vocabulary = 4, nil
	must(t, c.SendSpeechPolicy(ctx, free))
	poll("p1", true, local)
	must(t, fp.say("p1", "隨便說"))
	_, out = poll("p1", false, local)
	if raw, _ := json.Marshal(out["results"]); string(raw) != `[{"alternatives":[{"text":"隨便說"}]}]` {
		t.Fatalf("沒有詞彙表：%s", raw)
	}

	// 房主重新開始：上一局的授權失效
	fp.restart()
	if _, out := poll("p1", false, local); out["stop"] != "not_authorized" {
		t.Fatalf("重新開始後要停止：%v", out)
	}
}

func TestSpeechMatcher(t *testing.T) {
	first := func(n int) string { return zhIntegerReadings(n)[0] }
	got := []string{first(0), first(10), first(11), first(20), first(23), first(100), first(101), first(110), first(1010), first(2020)}
	want := []string{"零", "十", "十一", "二十", "二十三", "一百", "一百零一", "一百一十", "一千零一十", "二千零二十"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("念法：%v", got)
	}
	if r := zhIntegerReadings(200); !reflect.DeepEqual(r, []string{"二百", "兩百"}) {
		t.Fatalf("兩百：%v", r)
	}
	if r := zhIntegerReadings(22); len(r) != 1 {
		t.Fatalf("二十二只有一種念法：%v", r)
	}
	m := newSpeechMatcher(diceVocabulary())
	for text, want := range map[string]string{"二十三個六齋": "{n} 個 {m} 齋", "兩個二": "{n} 個 {m}", "二個兩": "{n} 個 {m}", "開！": "開", "１個１": "{n} 個 {m}"} {
		if got := m[normalizeSpeechText(text)].phrase; got != want {
			t.Errorf("%s → %q，應該是 %q", text, got, want)
		}
	}
	if _, ok := m[normalizeSpeechText("四個七")]; ok {
		t.Error("四個七不在詞彙表")
	}
	enum := newSpeechMatcher(&playtogether.SpeechVocabulary{Phrases: []string{"{a}", "投 {n} 號"},
		Slots: map[string]playtogether.SpeechSlot{"a": playtogether.SpeechEnumSlot("碰", "槓"), "n": playtogether.SpeechIntegerSlot(1, 12)}})
	if got := enum[normalizeSpeechText("投十二號")]; got.phrase != "投 {n} 號" || got.slots["n"] != 12 {
		t.Errorf("投十二號：%+v", got)
	}
	if got := enum["槓"]; got.slots["a"] != "槓" {
		t.Errorf("槓：%+v", got)
	}
}
