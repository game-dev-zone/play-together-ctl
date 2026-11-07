package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// 上架 skill：固定步驟與規則都在（改 skill 時不要漏掉）
func TestPtPublishSkill(t *testing.T) {
	files, err := ptSkillFiles()
	must(t, err)
	a, c := string(files[".agents/skills/playtogether-publish/SKILL.md"]), string(files[".claude/skills/playtogether-publish/SKILL.md"])
	if a == "" || a != c {
		t.Fatal("兩個 AI 工具要有同一份 skill")
	}
	if !strings.HasPrefix(a, "---\nname: playtogether-publish\ndescription: ") {
		t.Fatal("frontmatter 不對")
	}
	for _, want := range []string{
		"幫我上架到玩聚", "平台網址", // 觸發與第 0 步
		"pt init -y", "-game-server", "你不能自己提高內容邊界", // 接上專案
		"pt check -local", "ERROR_HARD_RULE", "REVIEW_REQUIRED", "PlayTogether.init", "pt.launch.code", "-connect", "pt cdn add", // 改到檢查通過
		"playtogether-store",               // 上架資料
		"pt login -register", "背景執行", "短碼", // 授權由開發者本人確認
		"pt register", "uploadKeyId", "pt upload -version", "-replace", "pt status", // 建立、上傳、回報
		"不要讀出、印出、複製或提交私鑰", "不要用 `-skip-check`", "不要執行 `pt logout`", "由玩聚平台審核決定", // 規則
	} {
		if !strings.Contains(a, want) {
			t.Errorf("skill 少了 %q", want)
		}
	}
	// 兩個 skill 的觸發詞不要重疊：「幫我上架」交給 playtogether-publish
	store := string(files[".claude/skills/playtogether-store/SKILL.md"])
	if strings.Contains(store, "「幫我上架」") || !strings.Contains(store, "playtogether-publish") {
		t.Fatal("playtogether-store 不要搶「幫我上架」，要指向 playtogether-publish")
	}
}

func TestSkillsInstall(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	must(t, cmdSkills([]string{"install"}))
	want, err := embedded.ReadFile("templates/skills/playtogether-publish/SKILL.md")
	must(t, err)
	for _, root := range []string{".claude/skills", ".agents/skills"} {
		got, err := os.ReadFile(filepath.Join(home, root, "playtogether-publish", "SKILL.md"))
		if err != nil || string(got) != string(want) {
			t.Fatalf("%s 沒有裝好上架 skill：%v", root, err)
		}
		// 只裝上架 skill；playtogether-store 由 pt init 放進專案
		if _, err := os.Stat(filepath.Join(home, root, "playtogether-store")); err == nil {
			t.Fatalf("%s 不該裝 playtogether-store", root)
		}
	}
	// 被改過的檔案：重跑會更新成 pt 的版本；其他 skill 不動
	f := filepath.Join(home, ".claude/skills/playtogether-publish/SKILL.md")
	must(t, os.WriteFile(f, []byte("舊版\n"), 0o644))
	other := filepath.Join(home, ".claude/skills/mine/SKILL.md")
	must(t, os.MkdirAll(filepath.Dir(other), 0o755))
	must(t, os.WriteFile(other, []byte("自己的\n"), 0o644))
	must(t, cmdSkills([]string{"install"}))
	if got, _ := os.ReadFile(f); string(got) != string(want) {
		t.Fatal("重跑要更新成 pt 的版本")
	}
	// 移除：只刪 pt 的 skill
	must(t, cmdSkills([]string{"uninstall"}))
	if _, err := os.Stat(filepath.Join(home, ".claude/skills/playtogether-publish")); err == nil {
		t.Fatal("uninstall 沒有移除")
	}
	if got, _ := os.ReadFile(other); string(got) != "自己的\n" {
		t.Fatal("uninstall 不能動其他 skill")
	}
	if err := cmdSkills([]string{"bogus"}); err == nil {
		t.Fatal("不認得的子指令要回錯誤")
	}
}
