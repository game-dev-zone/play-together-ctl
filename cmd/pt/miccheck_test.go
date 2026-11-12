package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// 音高、語音指令的防呆：沒處理停止、伺服器沒授權、自己送授權（建議改用套件）都要提醒；用套件時不提醒。
func TestCheckMicUsage(t *testing.T) {
	write := func(t *testing.T, dir, rel, body string) {
		t.Helper()
		p := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	run := func(p *project, web string) []string {
		var out []string
		checkMicUsage(p, web, func(level, format string, a ...any) {
			if level != levelWarn {
				t.Fatalf("應該是 WARNING：%s", level)
			}
			out = append(out, fmt.Sprintf(format, a...))
		})
		return out
	}
	has := func(list []string, s string) bool {
		for _, x := range list {
			if strings.Contains(x, s) {
				return true
			}
		}
		return false
	}

	t.Run("沒用到就不檢查（SDK 檔案裡的字樣不算）", func(t *testing.T) {
		dir := t.TempDir()
		write(t, dir, "web/playtogether-sdk.iife.js", "pitch.start(")
		if got := run(&project{dir: dir, Client: "web"}, filepath.Join(dir, "web")); len(got) != 0 {
			t.Fatalf("得到 %v", got)
		}
	})

	t.Run("沒處理停止、伺服器沒授權", func(t *testing.T) {
		dir := t.TempDir()
		write(t, dir, "web/game.js", "pt.pitch.start({ mode: 'ptt' });")
		write(t, dir, "server/main.go", "package main")
		got := run(&project{dir: dir, Client: "web"}, filepath.Join(dir, "web"))
		if len(got) != 2 || !has(got, "沒有處理停止") || !has(got, "沒有授權") {
			t.Fatalf("得到 %v", got)
		}
	})

	t.Run("伺服器自己送授權：建議改用套件", func(t *testing.T) {
		dir := t.TempDir()
		write(t, dir, "web/game.js", "pt.pitch.on('state', s => {}); pt.pitch.start();")
		write(t, dir, "server/main.go", "pt.SendPitchPolicy(ctx, policy)")
		got := run(&project{dir: dir, Client: "web"}, filepath.Join(dir, "web"))
		if len(got) != 1 || !has(got, "自己送音高授權") {
			t.Fatalf("得到 %v", got)
		}
	})

	t.Run("用套件自動續約、有處理停止：不提醒（Godot 與語音指令）", func(t *testing.T) {
		dir := t.TempDir()
		write(t, dir, "main.gd", "PlayTogether.speech_start()\nPlayTogether.speech_stopped.connect(_on_stop)")
		write(t, dir, "addons/playtogether/playtogether.gd", "pitch_start(")
		write(t, dir, "server/main.mjs", "room.allowSpeech(ids, 'ptt', 'zh-TW', vocab)")
		got := run(&project{dir: dir, Client: "godot"}, filepath.Join(dir, "web"))
		if len(got) != 0 {
			t.Fatalf("得到 %v", got)
		}
	})
}
