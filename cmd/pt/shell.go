package main

// 平台外殼樣式（playtogether.json 的 shell）：遊戲進行中平台浮動按鈕與選單的配色。
// 只能從平台提供的樣式選，外加一個強調色；按鈕的位置、大小、圖示和選單內容由平台固定。
// 和玩聚平台的檢查相同（伺服器回 400 bad_shell），兩邊要一起改。

import (
	"errors"
	"math"
	"regexp"
	"strconv"
	"strings"
)

type shellConfig struct {
	Style  string `json:"style,omitempty"`  // dark（預設）、light、minimal
	Accent string `json:"accent,omitempty"` // 選填，#rrggbb
}

var shellStyles = []string{"dark", "light", "minimal"}

// shellBase 是每種樣式按鈕的底色，強調色要和它有 3:1 對比（WCAG 1.4.11，非文字的介面元件）
var shellBase = map[string]string{"dark": "#0f1722", "light": "#ffffff", "minimal": "#0f1722"}

var shellAccentRe = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

// normalize 回傳標準化的樣式（空白是 dark）與強調色（小寫；空白表示沒有）。
func (c shellConfig) normalize() (string, string, error) {
	style := c.Style
	if style == "" {
		style = "dark"
	}
	base, ok := shellBase[style]
	if !ok {
		return "", "", errors.New("shell.style 只能是 dark、light 或 minimal（目前是 " + strconv.Quote(c.Style) + "）")
	}
	if c.Accent == "" {
		return style, "", nil
	}
	if !shellAccentRe.MatchString(c.Accent) {
		return "", "", errors.New("shell.accent 要是 #rrggbb 格式（目前是 " + strconv.Quote(c.Accent) + "）")
	}
	accent := strings.ToLower(c.Accent)
	if r := colorContrast(accent, base); r < 3 {
		return "", "", errors.New("shell.accent " + accent + " 和 " + style + " 樣式的底色 " + base + " 對比只有 " +
			strconv.FormatFloat(math.Floor(r*100)/100, 'f', 2, 64) + ":1，至少要 3:1（深色樣式選亮一點的顏色，light 選深一點的）")
	}
	return style, accent, nil
}

func colorLuminance(hex string) float64 {
	v, _ := strconv.ParseUint(hex[1:], 16, 32)
	ch := func(x uint64) float64 {
		f := float64(x) / 255
		if f <= 0.03928 {
			return f / 12.92
		}
		return math.Pow((f+0.055)/1.055, 2.4)
	}
	return 0.2126*ch(v>>16&0xff) + 0.7152*ch(v>>8&0xff) + 0.0722*ch(v&0xff)
}

func colorContrast(a, b string) float64 {
	la, lb := colorLuminance(a), colorLuminance(b)
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}

// checkShell：pt check 檢查 shell（平台也會拒絕，所以是宣告錯誤，-skip-check 不能略過）
func checkShell(p *project, add func(level, format string, a ...any)) {
	if p.Shell == nil {
		return
	}
	if _, _, err := p.Shell.normalize(); err != nil {
		add(levelSchema, "playtogether.json 的 %v", err)
	}
}

// shellBody 是 pt upload 送給平台的 shell；沒設定時不送（平台當成 dark）
func shellBody(p *project) (map[string]string, bool) {
	if p == nil || p.Shell == nil {
		return nil, false
	}
	style, accent, err := p.Shell.normalize()
	if err != nil {
		return nil, false
	}
	out := map[string]string{"style": style}
	if accent != "" {
		out["accent"] = accent
	}
	return out, true
}

// devShell 是 pt dev 外殼預覽用的樣式（設定錯誤時用預設，pt check 會另外報錯）
func devShell(p *project) shellConfig {
	if p == nil || p.Shell == nil {
		return shellConfig{Style: "dark"}
	}
	style, accent, err := p.Shell.normalize()
	if err != nil {
		return shellConfig{Style: "dark"}
	}
	return shellConfig{Style: style, Accent: accent}
}
