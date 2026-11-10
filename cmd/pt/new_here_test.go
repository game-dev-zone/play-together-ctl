package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// pt new .：在目前的空資料夾建立專案（先開好資料夾、在裡面打開 AI 助手的流程）。
func TestNewHere(t *testing.T) {
	requireEmbeddedSDK(t)
	dir := filepath.Join(t.TempDir(), "Tap Duel")
	must(t, os.MkdirAll(filepath.Join(dir, ".git"), 0o755))
	must(t, os.MkdirAll(filepath.Join(dir, ".claude"), 0o755))
	t.Chdir(dir)

	if err := cmdNew([]string{"my-game", "-name", "x-game", "-y"}); err == nil || !strings.Contains(err.Error(), "pt new .") {
		t.Fatalf("-name 只用在 pt new .：%v", err)
	}
	must(t, cmdNew([]string{".", "-y", "-platform", "https://pt.example", "-hosting", "playtogether"}))
	p, err := findProject()
	must(t, err)
	if p == nil || p.dir != dir || p.Name != "tap-duel" || !p.hostedServer() {
		t.Fatalf("要建立在目前的資料夾、名稱用資料夾名稱、玩聚代管：%+v", p)
	}
	for _, f := range []string{"CLAUDE.md", "server/main.go", "web/index.html", ".pt/generated.json", ".gitignore"} {
		if !exists(dir, f) {
			t.Errorf("沒有產生 %s", f)
		}
	}
	if exists(dir, "tap-duel") {
		t.Fatal("不能另外建子資料夾")
	}
	if fi, err := os.Stat(filepath.Join(dir, "keys")); err != nil || fi.Mode().Perm() != 0o700 {
		t.Fatal("keys/ 要建立在目前的資料夾")
	}
	// 專案裡不放開新遊戲的 skill（只裝在使用者層級）
	if exists(dir, ".claude/skills/playtogether-new") || !exists(dir, ".claude/skills/playtogether-publish/SKILL.md") {
		t.Fatal("專案的 skill 不對")
	}
	// 已經是專案：拒絕
	if err := cmdNew([]string{".", "-y"}); err == nil || !strings.Contains(err.Error(), "playtogether.json") {
		t.Fatalf("已經是專案要拒絕：%v", err)
	}
}

func TestNewHereRefuses(t *testing.T) {
	requireEmbeddedSDK(t)
	// 不是空資料夾
	dir := filepath.Join(t.TempDir(), "old-game")
	write(t, dir, "index.html", "<!doctype html>")
	t.Chdir(dir)
	if err := cmdNew([]string{".", "-y"}); err == nil || !strings.Contains(err.Error(), "pt init") {
		t.Fatalf("不是空資料夾要拒絕並提示 pt init：%v", err)
	}
	if exists(dir, "playtogether.json") {
		t.Fatal("被拒絕時不能寫檔案")
	}
	// 資料夾名稱不符合格式：要 -name
	dir = filepath.Join(t.TempDir(), "123")
	must(t, os.MkdirAll(dir, 0o755))
	t.Chdir(dir)
	if err := cmdNew([]string{".", "-y"}); err == nil || !strings.Contains(err.Error(), "-name") {
		t.Fatalf("名稱不符合格式要提示 -name：%v", err)
	}
	must(t, cmdNew([]string{".", "-y", "-name", "quiz-party"}))
	if p, _ := findProject(); p == nil || p.Name != "quiz-party" {
		t.Fatalf("-name：%+v", p)
	}
	// 隱藏檔和範本衝突（例如自己的 .gitignore）：不覆寫
	dir = filepath.Join(t.TempDir(), "with-ignore")
	write(t, dir, ".gitignore", "node_modules\n")
	t.Chdir(dir)
	if err := cmdNew([]string{".", "-y"}); err == nil || !strings.Contains(err.Error(), ".gitignore") {
		t.Fatalf("不能覆寫既有的 .gitignore：%v", err)
	}
}

// pt skills install 裝開新遊戲與上架兩個 skill；pt install skills 也可以。
func TestGlobalSkillsIncludeNew(t *testing.T) {
	files, err := globalSkillFiles()
	must(t, err)
	for _, name := range []string{"playtogether-new", "playtogether-publish"} {
		for _, root := range ptSkillRoots {
			if _, ok := files[filepath.Join(filepath.FromSlash(root), name, "SKILL.md")]; !ok {
				t.Errorf("沒有 %s/%s", root, name)
			}
		}
	}
	raw, _ := embedded.ReadFile("templates/skills/playtogether-new/SKILL.md")
	for _, want := range []string{"name: playtogether-new", "pt new .", "玩聚代管", "CLAUDE.md", "make verify", "playtogether-publish", "依類型建議做法", "AVG"} {
		if !strings.Contains(string(raw), want) {
			t.Errorf("playtogether-new 要提到 %q", want)
		}
	}
}
