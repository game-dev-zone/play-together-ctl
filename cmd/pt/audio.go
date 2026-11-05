package main

// pt check 的音檔來源檢查：上傳的每個音檔都要記錄在 art/SOURCES.md（審核準則 1.5；直播使用音樂要能證明授權）。

import (
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

var audioExts = []string{".mp3", ".ogg", ".oga", ".wav", ".m4a", ".aac", ".flac", ".opus", ".weba", ".webm"}

// audioFiles 回傳 root 底下的音檔（相對路徑），略過 skip 裡的資料夾與隱藏資料夾。
func audioFiles(root string, skip []string) []string {
	var out []string
	_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if p != root && (strings.HasPrefix(d.Name(), ".") || slices.Contains(skip, d.Name())) {
				return filepath.SkipDir
			}
			return nil
		}
		if slices.Contains(audioExts, strings.ToLower(filepath.Ext(d.Name()))) {
			rel, _ := filepath.Rel(root, p)
			out = append(out, filepath.ToSlash(rel))
		}
		return nil
	})
	return out
}

// checkAudioSources：網頁版從上傳資料夾找音檔；Godot 的音檔會打包進 index.pck，所以改找 Godot 專案裡的音檔。
func checkAudioSources(p *project, webDir string, add func(level, format string, a ...any)) {
	var files []string
	if p.Client == "godot" {
		files = audioFiles(p.path(p.godotDir()), []string{"addons", "server", "node_modules", "keys", "build", "design", "docs", "production", "art"})
	} else {
		files = audioFiles(webDir, nil)
	}
	if len(files) == 0 {
		return
	}
	raw, err := os.ReadFile(p.path("art/SOURCES.md"))
	if err != nil {
		add(levelSchema, "有 %d 個音檔，但找不到 art/SOURCES.md：每個音效與音樂都要記錄來源與授權（見 PLAYTOGETHER.md「音效與音樂」）", len(files))
		return
	}
	sources := string(raw)
	for _, f := range files {
		if !strings.Contains(sources, filepath.Base(f)) {
			add(levelSchema, "音檔 %s 沒有記錄在 art/SOURCES.md：請寫上來源、授權（證明網址）與日期；直播用的音樂授權不清楚，直播主的錄影可能被聲明版權", f)
		}
	}
}
