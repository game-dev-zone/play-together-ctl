package main

// 音高玩法與語音指令的防呆（pt check 的 WARNING）：用說的、唱的玩法常見的漏洞。
// 只掃開發者自己的程式（不含 SDK 檔案、addons、third_party、node_modules），只看有沒有用到、有沒有處理，不解析語法。

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// sourceText 把目錄底下符合副檔名的檔案內容串起來（略過 SDK 與第三方目錄）。
func sourceText(dir string, exts ...string) string {
	var b strings.Builder
	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		name := d.Name()
		if d.IsDir() {
			switch name {
			case "node_modules", "addons", ".godot", "third_party", ".git", "dist", "build":
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasPrefix(name, "playtogether-sdk") || strings.HasPrefix(name, "playtogether-server") {
			return nil
		}
		for _, e := range exts {
			if strings.HasSuffix(name, e) {
				if raw, err := os.ReadFile(path); err == nil && len(raw) < 4<<20 {
					b.Write(raw)
					b.WriteByte('\n')
				}
				break
			}
		}
		return nil
	})
	return b.String()
}

func containsAny(s string, subs ...string) bool {
	for _, x := range subs {
		if strings.Contains(s, x) {
			return true
		}
	}
	return false
}

// checkMicUsage：畫面用了音高或語音指令時，要處理停止；伺服器要授權，而且建議用套件自動續約。
func checkMicUsage(p *project, webDir string, add func(level, format string, args ...any)) {
	if p == nil {
		return
	}
	var client string
	godot := p.Client == "godot"
	if godot {
		client = sourceText(p.path(p.godotDir()), ".gd")
	} else {
		client = sourceText(webDir, ".js", ".mjs", ".ts", ".html")
	}
	type kind struct {
		name                  string
		start, startGD        []string
		handled, handledGD    []string
		allow, direct         []string
		allowHint, eventsHint string
	}
	kinds := []kind{
		{name: "音高", start: []string{"pitch.start("}, startGD: []string{"pitch_start("},
			handled: []string{"pitch.on('state'", `pitch.on("state"`}, handledGD: []string{"pitch_stopped"},
			allow: []string{"AllowPitch(", "allowPitch("}, direct: []string{"SendPitchPolicy(", ".pitchPolicy("},
			allowHint: "room.AllowPitch(玩家, \"ptt\")／StopPitch()（Node：room.allowPitch／stopPitch）", eventsHint: "pt.pitch.on('state')（Godot：pitch_stopped）"},
		{name: "語音指令", start: []string{"speech.start("}, startGD: []string{"speech_start("},
			handled: []string{"speech.on('state'", `speech.on("state"`}, handledGD: []string{"speech_stopped"},
			allow: []string{"AllowSpeech(", "allowSpeech("}, direct: []string{"SendSpeechPolicy(", ".speechPolicy("},
			allowHint: "room.AllowSpeech(玩家, \"ptt\", \"zh-TW\", 詞彙表)／StopSpeech()（Node：room.allowSpeech／stopSpeech）", eventsHint: "pt.speech.on('state')（Godot：speech_stopped）"},
	}
	var server string
	if p.gameServerMode() != "none" {
		// 伺服器的位置依專案而不同（server/、根目錄的 Go 模組），整個專案一起看；比對的字樣只會出現在伺服器程式
		server = sourceText(p.dir, ".go", ".mjs", ".js", ".ts")
	}
	for _, k := range kinds {
		start, handled := k.start, k.handled
		if godot {
			start, handled = k.startGD, k.handledGD
		}
		if !containsAny(client, start...) {
			continue
		}
		if !containsAny(client, handled...) {
			add(levelWarn, "畫面用了%s，但沒有處理停止（%s）：授權到期、玩家沒確認繼續、免按住沒聲音（timeout）、拒絕（denied）時玩家會卡住。收到 stopped 時顯示原因，並在玩家按按鈕時再呼叫 start", k.name, k.eventsHint)
		}
		if p.gameServerMode() == "none" {
			continue // 沒有遊戲伺服器時，ai_mode 的檢查會說明不能用
		}
		switch {
		case containsAny(server, k.allow...):
			// 用套件：自動續約，不用處理到期
		case containsAny(server, k.direct...):
			add(levelWarn, "遊戲伺服器自己送%s授權：要自己在到期前續約（最長 1 小時）、序號遞增，而且每間房每分鐘最多 30 次。建議改用套件的 %s，會自動續約與重試", k.name, k.allowHint)
		default:
			add(levelWarn, "畫面用了%s，但遊戲伺服器沒有授權：玩家會收到 not_authorized。一局開始時呼叫 %s", k.name, k.allowHint)
		}
	}
}
