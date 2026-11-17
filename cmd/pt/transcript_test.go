package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func transcriptProject(mode, scope string) *project {
	p := projectFrom(defaultPolicy(presetProfile("general"), "none"))
	p.GameServer = &gameServerConfig{Mode: mode}
	if scope != "-" {
		p.Voice = &voiceConfig{Transcript: scope}
	}
	return p
}

func runTranscript(p *project) levels {
	out := levels{}
	checkTranscript(p, func(level, format string, a ...any) { out[level] = append(out[level], sprintf(format, a...)) })
	return out
}

// pt check 的語音轉文字申報：沒有申報不提示；self、room 可以；其他值、空字串、沒有遊戲伺服器都是格式錯誤。
func TestCheckTranscript(t *testing.T) {
	if l := runTranscript(transcriptProject("playtogether-managed", "-")); len(l) != 0 {
		t.Fatalf("沒有申報不應有任何訊息：%v", l)
	}
	for _, scope := range []string{"self", "room"} {
		if l := runTranscript(transcriptProject("playtogether-managed", scope)); len(l) != 0 {
			t.Fatalf("%s 應該可以：%v", scope, l)
		}
	}
	for name, p := range map[string]*project{
		"未知的範圍":   transcriptProject("playtogether-managed", "everyone"),
		"空字串":     transcriptProject("playtogether-managed", ""),
		"沒有遊戲伺服器": transcriptProject("none", "room"),
	} {
		if l := runTranscript(p); len(l[levelSchema]) != 1 {
			t.Errorf("%s 應該是格式錯誤：%v", name, l)
		}
	}
}

// 有申報時：規則資料產生需要審核的提示，pt upload 的宣告快照帶 voice 區塊；沒有申報時兩者都沒有。
func TestTranscriptDeclaration(t *testing.T) {
	plain := transcriptProject("playtogether-managed", "-")
	if raw, ok := declarationBody(plain); !ok || strings.Contains(string(raw), "voice") {
		t.Fatalf("沒有申報時宣告不含 voice：%s", raw)
	}
	p := transcriptProject("playtogether-managed", "room")
	raw, ok := declarationBody(p)
	if !ok {
		t.Fatal("要有宣告")
	}
	var body map[string]any
	must(t, json.Unmarshal(raw, &body))
	if v, _ := body["voice"].(map[string]any); v["transcript"] != "room" {
		t.Fatalf("宣告要帶 voice.transcript：%s", raw)
	}
	if l := runPolicy(p); !l.only(levelReview) {
		t.Fatalf("申報語音轉文字要需要審核：%v", l)
	}
	if l := runPolicy(plain); !l.only() {
		t.Fatalf("沒有申報不需要審核：%v", l)
	}
}

// pt dev 的假平台：申報了語音轉文字時，控制面板可以模擬玩家不同意，之後拿不到啟動碼（和正式平台相同）；改成同意就恢復。
func TestFakeTranscriptConsent(t *testing.T) {
	fp := newFakePlatform(nil)
	d := &devPanel{fp: fp}
	if err := d.control("transcript-decline", "p1", ""); err == nil {
		t.Fatal("沒有申報時不能模擬")
	}
	fp.transcript = "room"
	if _, _, _, err := fp.issue("p1"); err != nil {
		t.Fatalf("本機預設當成已同意：%v", err)
	}
	must(t, d.control("transcript-decline", "p1", ""))
	if _, _, _, err := fp.issue("p1"); err == nil || err.Error() != "transcript_consent_required" {
		t.Fatalf("不同意的玩家不能拿啟動碼：%v", err)
	}
	if _, _, _, err := fp.issue("p2"); err != nil {
		t.Fatalf("別的玩家不受影響：%v", err)
	}
	must(t, d.control("transcript-agree", "p1", ""))
	if _, _, _, err := fp.issue("p1"); err != nil {
		t.Fatalf("改成同意後可以拿啟動碼：%v", err)
	}
	if err := d.control("transcript-decline", "bad player", ""); err == nil {
		t.Fatal("玩家代號格式要檢查")
	}
}
