package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// 假的 skill 套件：模擬 GitHub 的 tar.gz；scripts/ 與 router 不在清單裡，不能被裝進專案
var fakeSkillsRepo = map[string]string{
	"LICENSE":                               "Apache License\n",
	"NOTICE":                                "notice\n",
	"skills/disciplines/game-feel/SKILL.md": "---\nname: game-feel\n---\nfeel\n",
	"skills/disciplines/game-feel/references/juice.md":   "juice\n",
	"skills/godot/godot-audio/SKILL.md":                  "---\nname: godot-audio\n---\naudio\n",
	"skills/web-engines/phaser-core/SKILL.md":            "---\nname: phaser-core\n---\nphaser\n",
	"skills/disciplines/create-game-assets/scripts/x.py": "import os\n",
	"router/SKILL.md": "router\n",
}

func fakeSkillsManifest() ccgsManifest {
	m := ccgsManifest{Name: "awesome-gamedev-agent-skills", Repo: "example/gds", Version: "2026-01-01", Commit: strings.Repeat("b", 40), License: "Apache-2.0",
		Skills: []skillEntry{
			{"game-feel", "skills/disciplines/game-feel", "core"},
			{"godot-audio", "skills/godot/godot-audio", "godot"},
			{"phaser-core", "skills/web-engines/phaser-core", "web"},
		}}
	for p, c := range fakeSkillsRepo {
		if strings.Contains(p, "scripts/") || strings.HasPrefix(p, "router/") {
			continue // 清單只列說明文件
		}
		sum := sha256.Sum256([]byte(c))
		m.Files = append(m.Files, struct {
			Path   string `json:"path"`
			SHA256 string `json:"sha256"`
			Size   int64  `json:"size"`
		}{p, hex.EncodeToString(sum[:]), int64(len(c))})
	}
	return m
}

func skillsTarball(t *testing.T, override map[string]string) []byte {
	var entries []tarEntry
	for p, c := range fakeSkillsRepo {
		if v, ok := override[p]; ok {
			c = v
		}
		entries = append(entries, tarEntry{name: p, body: c})
	}
	return tarball(t, entries...)
}

func TestEmbeddedSkillsManifest(t *testing.T) {
	m, err := loadEmbeddedSkills()
	must(t, err)
	if m.License != "Apache-2.0" || len(m.Commit) != 40 {
		t.Fatalf("清單格式錯誤：%+v", m.Name)
	}
	listed := map[string]bool{}
	for _, f := range m.Files {
		listed[f.Path] = true
		// 只收說明文件：不能有任何會被執行的檔案
		if !strings.HasSuffix(f.Path, ".md") && !strings.HasSuffix(f.Path, ".json") && f.Path != "LICENSE" && f.Path != "NOTICE" {
			t.Errorf("清單不能包含 %s（只收說明文件）", f.Path)
		}
		if strings.Contains(f.Path, "/scripts/") || strings.HasPrefix(f.Path, "router/") || strings.Contains(f.Path, "/agents/") {
			t.Errorf("清單不能包含 %s", f.Path)
		}
	}
	if !listed["LICENSE"] || !listed["NOTICE"] {
		t.Fatal("Apache-2.0：LICENSE 與 NOTICE 要隨附")
	}
	// 和玩聚規則衝突的 skill 不能收
	for _, s := range m.Skills {
		switch s.Name {
		case "godot-multiplayer", "godot-export", "save-systems", "router", "itch-publish", "steam-publish", "godot-csharp":
			t.Errorf("不能收錄 %s（和玩聚規則衝突）", s.Name)
		}
		if !listed[s.Dir+"/SKILL.md"] {
			t.Errorf("%s 的 SKILL.md 不在清單裡", s.Name)
		}
		if s.Group != "core" && s.Group != "godot" && s.Group != "web" {
			t.Errorf("%s 的 group %q 不對", s.Name, s.Group)
		}
	}
}

func TestNewSkills(t *testing.T) {
	requireEmbeddedSDK(t)
	m := fakeSkillsManifest()
	orig := loadSkills
	loadSkills = func() (ccgsManifest, error) { return m, nil }
	t.Cleanup(func() { loadSkills = orig })

	archive := skillsTarball(t, nil)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write(archive) }))
	defer srv.Close()
	t.Setenv("PT_SKILLS_URL", srv.URL)

	dir := t.TempDir()
	t.Chdir(dir)
	if err := cmdNew([]string{"bad-level", "-skills", "all"}); err == nil {
		t.Fatal("-skills 只接受 none、core、full")
	}
	must(t, cmdNew([]string{"g-full", "-client", "godot", "-skills", "full"}))
	must(t, cmdNew([]string{"w-core", "-client", "web", "-skills", "core"}))
	must(t, cmdNew([]string{"w-none", "-client", "web"}))
	exists := func(p string) bool { _, err := os.Stat(filepath.Join(dir, p)); return err == nil }
	read := func(p string) string { b, _ := os.ReadFile(filepath.Join(dir, p)); return string(b) }

	for _, root := range []string{".claude/skills", ".agents/skills"} {
		for _, f := range []string{"game-feel/SKILL.md", "game-feel/references/juice.md", "godot-audio/SKILL.md"} {
			if !exists("g-full/" + root + "/" + f) {
				t.Errorf("Godot full 少了 %s/%s", root, f)
			}
		}
		if exists("g-full/" + root + "/phaser-core") {
			t.Error("Godot 專案不該裝網頁引擎的 skill")
		}
		if !exists("w-core/"+root+"/game-feel/SKILL.md") || exists("w-core/"+root+"/phaser-core") {
			t.Error("core 只裝通用 skill")
		}
	}
	for _, p := range []string{"g-full/.claude/skills/create-game-assets", "g-full/.claude/skills/router"} {
		if exists(p) {
			t.Errorf("清單外的檔案不能被裝進專案：%s", p)
		}
	}
	if !exists("g-full/THIRD_PARTY/awesome-gamedev-agent-skills/LICENSE") || !exists("g-full/THIRD_PARTY/awesome-gamedev-agent-skills/NOTICE") {
		t.Fatal("要附 LICENSE 與 NOTICE")
	}
	if !strings.Contains(read("g-full/THIRD_PARTY/awesome-gamedev-agent-skills/SOURCE.md"), strings.Repeat("b", 40)) {
		t.Fatal("SOURCE.md 要記錄 commit")
	}
	// 內容原樣複製
	if read("g-full/.agents/skills/game-feel/SKILL.md") != fakeSkillsRepo["skills/disciplines/game-feel/SKILL.md"] {
		t.Fatal("skill 內容不能被修改")
	}
	var p project
	must(t, json.Unmarshal([]byte(read("g-full/playtogether.json")), &p))
	if p.Skills != "full" {
		t.Fatalf("playtogether.json 要記錄 skills：%+v", p.Skills)
	}
	pg := read("g-full/PLAYTOGETHER.md")
	for _, want := range []string{"第三方遊戲開發 skill", "不採用、也不要折衷", "ENet", "AudioStreamMicrophone", "user://"} {
		if !strings.Contains(pg, want) {
			t.Errorf("PLAYTOGETHER.md 要寫明已知衝突：少了 %q", want)
		}
	}
	if strings.Contains(read("w-none/PLAYTOGETHER.md"), "第三方遊戲開發 skill") || exists("w-none/.agents") {
		t.Fatal("沒選 skill 的專案不該有 skill 與相關規則")
	}
	// 風格卡：每個專案都有
	if !strings.Contains(read("w-none/art/STYLE.md"), "調色盤") || !strings.Contains(read("w-none/PLAYTOGETHER.md"), "--skip-git-repo-check") {
		t.Fatal("要附 art/STYLE.md，codex exec 範例要能在還沒 git init 的專案執行")
	}

	// 下載的內容被竄改：停止，不留下專案
	archive = skillsTarball(t, map[string]string{"skills/disciplines/game-feel/SKILL.md": "evil\n"})
	if err := cmdNew([]string{"g-bad", "-client", "godot", "-skills", "full"}); err == nil {
		t.Fatal("內容不符應該停止")
	}
	if exists("g-bad") {
		t.Fatal("驗證失敗時不能留下專案資料夾")
	}
}
