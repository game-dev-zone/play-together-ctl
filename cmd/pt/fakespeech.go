package main

// pt dev 本機假平台的語音指令：遊戲伺服器的授權（POST /v1/speech/policy）、控制面板輸入的句子、
// 給瀏覽器 SDK 模擬模式的 GET /dev/speech。本機不用麥克風、不做辨識：在控制面板輸入文字當成辨識結果，
// 假平台照授權的詞彙表對回句型與欄位的值（和平台外殼的做法相同）。

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/game-dev-zone/play-together-ctl/internal/playtogether"
)

// fakeSpeechAlt 是一個候選（和 SDK 的 voice.speech 相同的欄位；confidence 由 SDK 補）。
type fakeSpeechAlt struct {
	Text   string         `json:"text"`
	Phrase string         `json:"phrase,omitempty"`
	Slots  map[string]any `json:"slots,omitempty"`
}

type fakeSpeechResult struct {
	Alternatives []fakeSpeechAlt `json:"alternatives"`
}

// fakeSpeech 是每位玩家（代號）的語音指令狀態。
type fakeSpeech struct {
	queue    []fakeSpeechResult // 控制面板輸入、還沒被畫面拿走的句子
	stop     string             // 控制面板要模擬的停止原因（下一次輪詢送出）
	lastPoll time.Time          // 畫面最近一次問（用來顯示「正在聽」）
}

var speechStopReasons = []string{"user", "denied", "not_authorized", "mic_denied", "unsupported", "background", "timeout", "busy", "closed"}

// speechPolicy：語音指令授權。和正式平台相同的檢查：格式（含詞彙表）、生效時間在現在附近、最多 1 小時、房間開著、
// 局數是目前這一局、序號遞增（同序號同內容當成重送）。
func (f *fakePlatform) speechPolicy(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	const ep = "語音指令授權"
	body, ok := f.verifyLocked(w, r, ep)
	if !ok {
		return
	}
	fail := func(code int, errCode, hint string) {
		msg := errCode
		if hint != "" {
			msg += "（" + hint + "）"
		}
		f.logLocked(ep, code, msg, true)
		f.write(w, code, map[string]string{"error": errCode})
	}
	var p playtogether.SpeechPolicy
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&p); err != nil {
		fail(400, "bad_policy", "格式不符")
		return
	}
	if err := p.Validate(); err != nil {
		fail(400, "bad_policy", err.Error())
		return
	}
	nowMs := f.now().Add(f.skew).UnixMilli()
	if p.EffectiveAtServerMs < nowMs-10_000 || p.EffectiveAtServerMs > nowMs+30_000 || p.ExpiresAtServerMs-p.EffectiveAtServerMs > time.Hour.Milliseconds() {
		fail(400, "bad_policy_time", "生效時間要在現在前 10 秒到後 30 秒，有效期間最多 1 小時")
		return
	}
	room := f.rooms[p.RoomID]
	if room == nil || !room.Open {
		fail(404, "room_not_found", "")
		return
	}
	if p.Epoch != room.Epoch {
		fail(409, "stale_epoch", fmt.Sprintf("房間目前是第 %d 局；房主重新開始後要用新的局數重新授權", room.Epoch))
		return
	}
	canon, _ := json.Marshal(p)
	if room.Speech != nil && p.Sequence <= room.Speech.Sequence {
		if p.Sequence == room.Speech.Sequence && string(canon) == room.SpeechBody {
			f.logLocked(ep, 200, fmt.Sprintf("序號 %d 重送", p.Sequence), true)
			f.write(w, 200, map[string]any{"ok": true})
			return
		}
		fail(409, "stale_sequence", fmt.Sprintf("序號要大於上一次的 %d", room.Speech.Sequence))
		return
	}
	room.Speech, room.SpeechBody = &p, string(canon)
	room.speechMatch = nil
	vocab := "沒有詞彙表"
	if p.Vocabulary != nil {
		n, _ := p.Vocabulary.Size()
		vocab = fmt.Sprintf("詞彙表 %d 句", n)
	}
	f.logLocked(ep, 200, fmt.Sprintf("授權 %d 位玩家（序號 %d、%s）；在控制面板的「語音指令」輸入要說的話", len(p.AllowedPlayerIDs), p.Sequence, vocab), true)
	f.write(w, 200, map[string]any{"ok": true})
}

// speechAllowedLocked：這位玩家現在有沒有被授權（目前的房間、這一局、名單、有效時間）。
func (f *fakePlatform) speechAllowedLocked(code string) bool {
	p, room := f.players[code], f.room
	if p == nil || room == nil || !room.Open || room.Kicked[code] || room.Speech == nil || room.Speech.Epoch != room.Epoch {
		return false
	}
	nowMs := f.now().Add(f.skew).UnixMilli()
	return slices.Contains(room.Speech.AllowedPlayerIDs, p.ID) && nowMs >= room.Speech.EffectiveAtServerMs-10_000 && nowMs < room.Speech.ExpiresAtServerMs
}

func (f *fakePlatform) speechStateLocked(code string) *fakeSpeech {
	if f.speech == nil {
		f.speech = map[string]*fakeSpeech{}
	}
	s := f.speech[code]
	if s == nil {
		s = &fakeSpeech{}
		f.speech[code] = s
	}
	return s
}

// devSpeech：瀏覽器 SDK 的模擬模式在 speech.start() 之後每 0.4 秒問一次（網址帶 pt_dev、pt_player 時）。
// 回 {"results": [...]}（控制面板輸入的句子）或 {"stop": 原因}（沒有授權、控制面板模擬的停止）。只允許本機的網頁呼叫。
func (f *fakePlatform) devSpeech(w http.ResponseWriter, r *http.Request) {
	o := r.Header.Get("Origin")
	if !loopbackOrigin(o) {
		f.write(w, 403, map[string]string{"error": "origin"})
		return
	}
	w.Header().Set("Access-Control-Allow-Origin", o)
	w.Header().Set("Vary", "Origin")
	if r.Method == http.MethodOptions {
		w.WriteHeader(204)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	code := r.URL.Query().Get("player")
	if !fakePlayerCode.MatchString(code) {
		f.write(w, 400, map[string]string{"error": "bad_player"})
		return
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	s := f.speechStateLocked(code)
	start := r.URL.Query().Get("start") == "1"
	if start {
		s.queue, s.stop = nil, "" // 新的一次：不重播之前輸入的句子
	}
	if !f.speechAllowedLocked(code) {
		s.queue, s.lastPoll = nil, time.Time{}
		if start {
			f.logLocked("語音指令（瀏覽器）", 403, "not_authorized（遊戲伺服器還沒用 speech.policy 授權這位玩家，或授權已過期、不是這一局）", false)
		}
		f.write(w, 200, map[string]string{"stop": "not_authorized"})
		return
	}
	if s.stop != "" {
		reason := s.stop
		s.stop, s.queue, s.lastPoll = "", nil, time.Time{}
		f.write(w, 200, map[string]string{"stop": reason})
		return
	}
	s.lastPoll = f.now()
	results := s.queue
	s.queue = nil
	if results == nil {
		results = []fakeSpeechResult{}
	}
	f.write(w, 200, map[string]any{"results": results})
}

// speechListeningLocked：這位玩家的畫面是不是正在聽（最近 2 秒內問過）。
func (f *fakePlatform) speechListeningLocked(code string) bool {
	s := f.speech[code]
	return s != nil && !s.lastPoll.IsZero() && f.now().Sub(s.lastPoll) < 2*time.Second
}

// say：控制面板輸入一句話，當成這位玩家的辨識結果。照授權的詞彙表對回句型與欄位；
// 詞彙表是 strict 而且對不上時，和平台一樣送出「沒聽到」（空的候選）。
func (f *fakePlatform) say(code, text string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if !f.speechAllowedLocked(code) {
		return errors.New("這位玩家沒有語音指令的授權（遊戲伺服器要先送 speech.policy）")
	}
	if !f.speechListeningLocked(code) {
		return errors.New("這位玩家的畫面還沒開始聽（遊戲要先呼叫 speech.start）")
	}
	text = strings.TrimSpace(text)
	if utf8.RuneCountInString(text) > 100 {
		return errors.New("最多 100 個字")
	}
	room := f.room
	res := fakeSpeechResult{Alternatives: []fakeSpeechAlt{}}
	note := "沒聽到（空的結果）"
	if text != "" {
		alt := fakeSpeechAlt{Text: text}
		note = "「" + text + "」"
		if v := room.Speech.Vocabulary; v != nil {
			if room.speechMatch == nil {
				room.speechMatch = newSpeechMatcher(v)
			}
			if m, ok := room.speechMatch[normalizeSpeechText(text)]; ok {
				alt.Phrase, alt.Slots = m.phrase, m.slots
				note += " → " + m.phrase
			} else if v.Strict {
				res.Alternatives, note = []fakeSpeechAlt{}, note+" 不在詞彙表（strict）：送出空的結果"
				alt.Text = ""
			} else {
				note += "（不在詞彙表）"
			}
		}
		if alt.Text != "" {
			res.Alternatives = append(res.Alternatives, alt)
		}
	}
	s := f.speechStateLocked(code)
	if len(s.queue) < 20 {
		s.queue = append(s.queue, res)
	}
	f.logLocked("語音指令（控制面板）", 200, f.players[code].Name+"："+note, false)
	return nil
}

// stopSpeech：控制面板模擬停止原因（玩家不同意、麥克風被拒等）。
func (f *fakePlatform) stopSpeech(code, reason string) error {
	if !slices.Contains(speechStopReasons, reason) {
		return errors.New("bad_reason")
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if !f.speechListeningLocked(code) {
		return errors.New("這位玩家的畫面還沒開始聽（遊戲要先呼叫 speech.start）")
	}
	f.speechStateLocked(code).stop = reason
	f.logLocked("語音指令（控制面板）", 200, f.players[code].Name+"：模擬停止（"+reason+"）", false)
	return nil
}

// ---------- 詞彙表：展開句型、整數的念法、把文字對回句型（和平台外殼相同的規則） ----------

type speechMatch struct {
	phrase string
	slots  map[string]any
}

var zhDigits = []string{"零", "一", "二", "三", "四", "五", "六", "七", "八", "九"}

// zhIntegerReadings：0–9999 的華語念法。第一個是標準念法（二十三、一百零五），之後是口語的「兩」。
func zhIntegerReadings(n int) []string {
	if n < 0 || n > 9999 {
		return nil
	}
	if n < 10 {
		if n == 2 {
			return []string{"二", "兩"}
		}
		return []string{zhDigits[n]}
	}
	read := func(liang bool) string {
		out, pendingZero, rest := "", false, n
		for _, u := range []struct {
			value int
			unit  string
		}{{1000, "千"}, {100, "百"}, {10, "十"}} {
			d := rest / u.value
			rest %= u.value
			if d == 0 {
				if out != "" {
					pendingZero = true
				}
				continue
			}
			if pendingZero {
				out += "零"
			}
			pendingZero = false
			switch {
			case u.value == 10 && d == 1 && out == "": // 十幾不念「一十幾」
				out += u.unit
			case liang && d == 2 && u.value >= 100:
				out += "兩" + u.unit
			default:
				out += zhDigits[d] + u.unit
			}
		}
		if rest > 0 {
			if pendingZero {
				out += "零"
			}
			out += zhDigits[rest]
		}
		return out
	}
	standard, spoken := read(false), read(true)
	if spoken == standard {
		return []string{standard}
	}
	return []string{standard, spoken}
}

// normalizeSpeechText：比對用，去掉空白與標點、全形換半形、英文小寫。
func normalizeSpeechText(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= 0xFF01 && r <= 0xFF5E { // 全形英數與標點 → 半形
			r -= 0xFEE0
		}
		if unicode.IsSpace(r) || unicode.IsPunct(r) || unicode.IsSymbol(r) {
			continue
		}
		b.WriteRune(unicode.ToLower(r))
	}
	return b.String()
}

var speechSlotRef = regexp.MustCompile(`\{([^{}]*)\}`)

// newSpeechMatcher 把詞彙表展開成「念法 → 句型與欄位的值」。詞彙表已經通過 Validate（展開後最多 5000 句）。
func newSpeechMatcher(v *playtogether.SpeechVocabulary) map[string]speechMatch {
	type choice struct {
		value any
		texts []string
	}
	index := map[string]speechMatch{}
	for _, phrase := range v.Phrases {
		names := []string{}
		for _, m := range speechSlotRef.FindAllStringSubmatch(phrase, -1) {
			if !slices.Contains(names, m[1]) {
				names = append(names, m[1])
			}
		}
		choices := make([][]choice, len(names))
		for i, name := range names {
			slot := v.Slots[name]
			if slot.Type == "enum" {
				for _, val := range slot.Values {
					choices[i] = append(choices[i], choice{val, []string{val}})
				}
				continue
			}
			for n := *slot.Min; n <= *slot.Max; n++ {
				choices[i] = append(choices[i], choice{n, append(zhIntegerReadings(n), strconv.Itoa(n))})
			}
		}
		var pick func(i int, slots map[string]any, texts []string)
		pick = func(i int, slots map[string]any, texts []string) {
			if i == len(names) {
				for _, t := range texts {
					if key := normalizeSpeechText(t); key != "" {
						if _, dup := index[key]; !dup {
							index[key] = speechMatch{phrase, slots}
						}
					}
				}
				return
			}
			for _, c := range choices[i] {
				next := map[string]any{}
				for k, val := range slots {
					next[k] = val
				}
				next[names[i]] = c.value
				// 每種念法都算（「兩個二」「二個兩」都對得上）
				out := make([]string, 0, len(texts)*len(c.texts))
				for _, t := range texts {
					for _, x := range c.texts {
						out = append(out, strings.ReplaceAll(t, "{"+names[i]+"}", x))
					}
				}
				pick(i+1, next, out)
			}
		}
		pick(0, map[string]any{}, []string{phrase})
	}
	return index
}
