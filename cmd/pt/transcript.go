package main

// 語音轉文字的申報（玩聚平台 docs/ASR_INTEGRATION_AUDIT.md 第 0 節第 7 點）：
//   - playtogether.json 的 voice.transcript："self"（逐字稿只給說話者自己的畫面）或 "room"（遊戲可以分享給同房間的玩家）
//   - pt check 檢查值與架構（授權由遊戲伺服器送，所以要有遊戲伺服器）；需要審核的提示由規則資料 rules.json 的 voice.transcript 產生
//   - pt upload 放進宣告快照的 voice 區塊；平台據此在玩家進房前要求同意（不同意就不能加入），授權範圍要和申報相同
//   - pt dev 的假平台照同樣的規則檢查語音轉文字的授權（fakespeech.go）

// voiceConfig 是 playtogether.json 的 voice 區塊。
type voiceConfig struct {
	// Transcript 是語音轉文字的範圍："self" 或 "room"；空字串＝不使用
	Transcript string `json:"transcript,omitempty"`
}

// transcriptScope 回傳申報的語音轉文字範圍（沒有申報時是空字串）。
func (p *project) transcriptScope() string {
	if p == nil || p.Voice == nil {
		return ""
	}
	return p.Voice.Transcript
}

// checkTranscript 是 pt check 的語音轉文字部分。
func checkTranscript(p *project, add addFunc) {
	if p == nil || p.Voice == nil {
		return
	}
	switch p.Voice.Transcript {
	case "":
		add(levelSchema, "playtogether.json 的 voice.transcript 是空的：不用語音轉文字就拿掉 voice 區塊，要用就填 self 或 room")
		return
	case "self", "room":
	default:
		add(levelSchema, "playtogether.json 的 voice.transcript 只能是 self（逐字稿只給說話者自己的畫面）或 room（遊戲可以分享給同房間的玩家），現在是 %q", p.Voice.Transcript)
		return
	}
	if p.GameServer != nil && p.GameServer.Mode == "none" {
		add(levelSchema, "voice.transcript 要由遊戲伺服器授權（room.AllowTranscript／room.allowTranscript），但 gameServer.mode 是 none（沒有遊戲伺服器）")
	}
}
