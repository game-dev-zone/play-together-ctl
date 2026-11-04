package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// 範本附預設音效（程式合成、沒有音檔）；pt check 要擋下沒有記錄來源的音檔。
func TestAudioSources(t *testing.T) {
	requireEmbeddedSDK(t)
	for _, client := range []string{"web", "godot"} {
		t.Run(client, func(t *testing.T) {
			dir := t.TempDir()
			t.Chdir(dir)
			must(t, cmdNew([]string{"snd", "-client", client}))
			root := filepath.Join(dir, "snd")
			t.Chdir(root)
			p, _ := findProject()
			read := func(rel string) string { b, _ := os.ReadFile(filepath.Join(root, rel)); return string(b) }

			// 範本：程式合成的音效、遵守平台靜音，並記錄在 art/SOURCES.md
			if client == "web" {
				if !strings.Contains(read("web/index.html"), `src="sfx.js"`) || !strings.Contains(read("web/game.js"), "pt.on('mute'") || !strings.Contains(read("web/game.js"), "sfx.unlock()") {
					t.Fatal("網頁版要載入 sfx.js、在點擊時 unlock、接上平台靜音")
				}
			} else if gd := read(filepath.Join(p.godotDir(), "main.gd")); !strings.Contains(gd, "_build_sfx()") || !strings.Contains(gd, "mute_changed") {
				t.Fatal("Godot 版要產生音效並接上平台靜音")
			}
			if !strings.Contains(read("art/SOURCES.md"), "程式合成") || !strings.Contains(read("PLAYTOGETHER.md"), "## 音效與音樂") {
				t.Fatal("SOURCES.md 要記錄程式合成音效，PLAYTOGETHER.md 要有音效規則")
			}

			// 沒有音檔：不報錯
			webDir := p.path(p.Web)
			if hasFinding(checkProject(p, webDir), "error", "音檔") {
				t.Fatal("沒有音檔不該報錯")
			}
			// 加一個沒記錄的音檔 → error；記錄後 → 通過
			audioDir := filepath.Join(root, "web", "sounds")
			if client == "godot" {
				audioDir = filepath.Join(p.path(p.godotDir()), "assets")
			}
			must(t, os.MkdirAll(audioDir, 0o755))
			must(t, os.WriteFile(filepath.Join(audioDir, "bgm_loop.ogg"), []byte("x"), 0o644))
			if !hasFinding(checkProject(p, webDir), "error", "bgm_loop.ogg") {
				t.Fatal("沒記錄來源的音檔要報錯")
			}
			f, _ := os.OpenFile(filepath.Join(root, "art/SOURCES.md"), os.O_APPEND|os.O_WRONLY, 0)
			_, _ = f.WriteString("| `bgm_loop.ogg` | Kenney | CC0（https://kenney.nl/support） | 2026-11-04 | 背景音樂 |\n")
			f.Close()
			if hasFinding(checkProject(p, webDir), "error", "bgm_loop.ogg") {
				t.Fatal("記錄之後不該報錯")
			}
			// 沒有 SOURCES.md
			must(t, os.Remove(filepath.Join(root, "art/SOURCES.md")))
			if !hasFinding(checkProject(p, webDir), "error", "art/SOURCES.md") {
				t.Fatal("有音檔但沒有 SOURCES.md 要報錯")
			}
		})
	}
}
