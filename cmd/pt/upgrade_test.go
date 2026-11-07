package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func read(t *testing.T, root, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func write(t *testing.T, root, rel, content string) {
	t.Helper()
	must(t, os.MkdirAll(filepath.Dir(filepath.Join(root, filepath.FromSlash(rel))), 0o755))
	must(t, os.WriteFile(filepath.Join(root, filepath.FromSlash(rel)), []byte(content), 0o644))
}

// pt new 記下 sdk 與 template 的基線；遊戲本身的檔案（starter）不記。
func TestManifestOnNew(t *testing.T) {
	requireEmbeddedSDK(t)
	root := newProject(t, "web-go")
	m, err := loadManifest(root)
	must(t, err)
	if m == nil || m.TemplateVersion != version || m.SDKVersion != sdkVersion() {
		t.Fatalf("要有 %s：%+v", manifestPath, m)
	}
	for rel, kind := range map[string]string{"Makefile": kindTemplate, "CLAUDE.md": kindTemplate, ".gitignore": kindTemplate,
		"web/playtogether-sdk.iife.js": kindSDK, "server/third_party/playtogether-sdk-go/client.go": kindSDK} {
		f, ok := m.get(rel)
		if !ok || f.Kind != kind || f.BaselineSHA256 != sha([]byte(read(t, root, rel))) {
			t.Fatalf("%s 要記成 %s 並有正確的基線：%+v", rel, kind, f)
		}
	}
	for _, rel := range []string{"playtogether.json", "server/main.go", "web/game.js", "store/listing.json", "README.md", "docs/content/CONTENT_PROFILE.md"} {
		if _, ok := m.get(rel); ok {
			t.Fatalf("%s 是遊戲本身的檔案，不能交給 pt upgrade", rel)
		}
	}
}

func TestUpgrade(t *testing.T) {
	requireEmbeddedSDK(t)
	root := newProject(t, "web-go")
	t.Chdir(root)

	// 剛建立：沒有要更新的
	must(t, cmdUpgrade([]string{"-y"}))

	m, _ := loadManifest(root)
	// 1. 舊版範本、沒改過（磁碟＝基線）→ 換新
	write(t, root, "Makefile", "# 舊版範本\n")
	m.set("Makefile", kindTemplate, sha([]byte("# 舊版範本\n")))
	// 2. 改過 → 寫 .pt-new，原檔不動
	write(t, root, "CLAUDE.md", "# 我自己的規則\n")
	// 3. 刪掉的範本檔 → 不補回
	must(t, os.Remove(filepath.Join(root, "AGENTS.md")))
	// 4. 新的範本檔（基線沒有、磁碟沒有）→ 新增
	m.remove("PLAYTOGETHER.md")
	must(t, os.Remove(filepath.Join(root, "PLAYTOGETHER.md")))
	// 5. 新範本拿掉的檔案：沒改過的 -delete-removed 才刪；改過的保留
	write(t, root, "docs/ai/OLD.md", "舊文件\n")
	m.set("docs/ai/OLD.md", kindTemplate, sha([]byte("舊文件\n")))
	write(t, root, "docs/ai/OLD2.md", "我改過\n")
	m.set("docs/ai/OLD2.md", kindTemplate, sha([]byte("原本的\n")))
	must(t, m.save(root))
	// 6. 遊戲本身的檔案：一律不碰
	write(t, root, "server/main.go", "package main // 我的遊戲\n")
	write(t, root, "playtogether.json", strings.Replace(read(t, root, "playtogether.json"), `"name"`, `"name"`, 1))

	// -dry-run 不寫任何檔案
	must(t, cmdUpgrade([]string{"-dry-run"}))
	if read(t, root, "Makefile") != "# 舊版範本\n" || exists(root, "CLAUDE.md.pt-new") {
		t.Fatal("-dry-run 不能寫檔案")
	}
	// 非互動又沒有 -y：只列計畫
	if err := cmdUpgrade(nil); err == nil || !strings.Contains(err.Error(), "-y") {
		t.Fatalf("非互動要加 -y：%v", err)
	}

	must(t, cmdUpgrade([]string{"-y", "-delete-removed"}))
	if read(t, root, "Makefile") == "# 舊版範本\n" {
		t.Fatal("沒改過的範本要換新")
	}
	if read(t, root, "CLAUDE.md") != "# 我自己的規則\n" || !exists(root, "CLAUDE.md.pt-new") {
		t.Fatal("改過的要保留並寫 .pt-new")
	}
	if exists(root, "AGENTS.md") {
		t.Fatal("刪掉的範本檔不能補回")
	}
	if !exists(root, "PLAYTOGETHER.md") {
		t.Fatal("新的範本檔要新增")
	}
	if exists(root, "docs/ai/OLD.md") || !exists(root, "docs/ai/OLD2.md") {
		t.Fatal("-delete-removed 只刪沒改過的")
	}
	if read(t, root, "server/main.go") != "package main // 我的遊戲\n" {
		t.Fatal("遊戲本身的檔案不能動")
	}
	m, _ = loadManifest(root)
	if f, _ := m.get("Makefile"); f.BaselineSHA256 != sha([]byte(read(t, root, "Makefile"))) {
		t.Fatal("換新後基線要更新")
	}
	if f, _ := m.get("CLAUDE.md"); f.BaselineSHA256 == sha([]byte(read(t, root, "CLAUDE.md"))) {
		t.Fatal("有衝突的檔案基線不能變成你的版本（否則下次會被覆蓋）")
	}

	// 合併：照新版改好 → 下次當作沒改過
	write(t, root, "CLAUDE.md", read(t, root, "CLAUDE.md.pt-new"))
	must(t, os.Remove(filepath.Join(root, "CLAUDE.md.pt-new")))
	must(t, cmdUpgrade([]string{"-y"}))
	m, _ = loadManifest(root)
	if f, _ := m.get("CLAUDE.md"); f.BaselineSHA256 != sha([]byte(read(t, root, "CLAUDE.md"))) || exists(root, "CLAUDE.md.pt-new") {
		t.Fatal("合併成新版後基線要更新")
	}
}

// 沒有 .pt/generated.json 的舊專案：先 -init；和範本不同的一律當作改過（不會被覆蓋）。
func TestUpgradeInit(t *testing.T) {
	requireEmbeddedSDK(t)
	root := newProject(t, "web-node")
	t.Chdir(root)
	must(t, os.Remove(filepath.Join(root, manifestPath)))
	write(t, root, "CLAUDE.md", "# 舊版或我改過\n")

	if err := cmdUpgrade([]string{"-y"}); err == nil || !strings.Contains(err.Error(), "-init") {
		t.Fatalf("沒有基線要先 -init：%v", err)
	}
	must(t, cmdUpgrade([]string{"-init"}))
	if err := cmdUpgrade([]string{"-init"}); err == nil {
		t.Fatal("已經有基線時 -init 要拒絕")
	}
	m, _ := loadManifest(root)
	if f, ok := m.get("CLAUDE.md"); !ok || f.BaselineSHA256 != "" {
		t.Fatalf("和範本不同的基線要留空：%+v", f)
	}
	if f, _ := m.get("Makefile"); f.BaselineSHA256 == "" {
		t.Fatal("和範本一樣的要有基線")
	}
	if f, ok := m.get("server/playtogether-server.mjs"); !ok || f.Kind != kindSDK || f.BaselineSHA256 == "" {
		t.Fatalf("SDK 也要記：%+v", f)
	}
	must(t, cmdUpgrade([]string{"-y"}))
	if read(t, root, "CLAUDE.md") != "# 舊版或我改過\n" || !exists(root, "CLAUDE.md.pt-new") {
		t.Fatal("基線不明的檔案不能覆蓋")
	}
}

// pt init 接入的既有專案不適用。
func TestUpgradeExistingProject(t *testing.T) {
	p := &project{Template: existingTemplate}
	if !p.existing() {
		t.Skip()
	}
	dir := t.TempDir()
	t.Chdir(dir)
	write(t, dir, projectFile, `{"name":"x","template":"`+existingTemplate+`","client":"web","server":"none"}`)
	if err := cmdUpgrade([]string{"-y"}); err == nil || !strings.Contains(err.Error(), "pt init") {
		t.Fatalf("既有專案要拒絕：%v", err)
	}
}
