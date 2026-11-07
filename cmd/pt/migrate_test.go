package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const legacyMain = `package main

func (g *game) abort() string { return g.roundID }

func main() {
	mux.HandleFunc("POST /action", func(w http.ResponseWriter, r *http.Request) {})
	_ = s.hub.pt.ReportRoundStart(ctx, roomID, epoch, roundID, ids)
}
`

func TestMigrateServerKit(t *testing.T) {
	requireEmbeddedSDK(t)
	root := newProject(t, "web-go")
	t.Chdir(root)

	// 新專案：不需要遷移
	must(t, cmdMigrate([]string{"server-kit"}))
	if exists(root, migrateGuide) {
		t.Fatal("沒有舊檔時不能產生遷移說明")
	}

	// 模擬 v0.18 之前的專案
	write(t, root, "server/main.go", legacyMain)
	write(t, root, "server/session.go", "package main // 舊的平台接口\n")
	write(t, root, "server/limit.go", "package main // 舊的限流\n")
	if err := cmdMigrate([]string{"nope"}); err == nil {
		t.Fatal("未知的遷移要拒絕")
	}

	// 第一步：寫出新範本與說明，不動舊檔
	must(t, cmdMigrate([]string{"server-kit"}))
	if !exists(root, "server/main.go.pt-new") || !exists(root, "server/session.go") || read(t, root, "server/main.go") != legacyMain {
		t.Fatal("第一步只產生對照，不能動到舊檔")
	}
	guide := read(t, root, migrateGuide)
	for _, want := range []string{"abort", "POST /action", "ReportRoundStart", "s.hub", "-apply"} {
		if !strings.Contains(guide, want) {
			t.Fatalf("說明要列出 %q：\n%s", want, guide)
		}
	}
	if !strings.Contains(read(t, root, "server/main.go.pt-new"), "gameserver") {
		t.Fatal("對照用的新範本要用套件")
	}

	// -apply 但還沒搬好：編譯失敗 → 還原
	old := buildServer
	t.Cleanup(func() { buildServer = old })
	buildServer = func(*project) error { return errors.New("undefined: hub") }
	if err := cmdMigrate([]string{"server-kit", "-apply"}); err == nil || !strings.Contains(err.Error(), "已經還原") {
		t.Fatalf("編譯失敗要還原：%v", err)
	}
	if !exists(root, "server/session.go") || !exists(root, "server/limit.go") {
		t.Fatal("失敗時舊檔要回到原位")
	}

	// 搬好了：舊檔移到備份，對照檔與說明清掉
	buildServer = func(*project) error { return nil }
	must(t, cmdMigrate([]string{"server-kit", "-apply"}))
	if exists(root, "server/session.go") || exists(root, "server/limit.go") || exists(root, "server/main.go.pt-new") || exists(root, migrateGuide) {
		t.Fatal("成功後要移除舊檔、對照檔與說明")
	}
	backups, _ := filepath.Glob(filepath.Join(root, ".pt", "backup", "*", "server", "session.go"))
	if len(backups) != 1 {
		t.Fatal("舊檔要留一份備份")
	}
	if !strings.Contains(read(t, root, ".gitignore"), ".pt/backup/") {
		t.Fatal("備份不要進版本控制")
	}
}

// 實際用 go build 驗證：新範本的 main.go 換上去之後可以編譯（確認 -apply 的編譯檢查有效）。
func TestMigrateServerKitBuild(t *testing.T) {
	requireEmbeddedSDK(t)
	if testing.Short() {
		t.Skip("需要 go build")
	}
	root := newProject(t, "web-go")
	t.Chdir(root)
	write(t, root, "server/session.go", "package main\n")
	must(t, cmdMigrate([]string{"server-kit"}))
	// 用新範本蓋過去（等於遊戲規則已經搬好）
	must(t, os.Rename(filepath.Join(root, "server/main.go.pt-new"), filepath.Join(root, "server/main.go")))
	must(t, cmdMigrate([]string{"server-kit", "-apply"}))

	// 還沒搬好（舊程式用到 session.go 的東西）：編譯失敗並還原
	write(t, root, "server/session.go", "package main\n\nfunc legacyHelper() int { return 1 }\n")
	write(t, root, "server/use_legacy.go", "package main\n\nvar _ = legacyHelper()\n")
	must(t, cmdMigrate([]string{"server-kit"}))
	if err := cmdMigrate([]string{"server-kit", "-apply"}); err == nil {
		t.Fatal("用到舊檔的程式要編譯失敗")
	}
	if !exists(root, "server/session.go") {
		t.Fatal("失敗時要還原")
	}
}

// Node：還在 import session.mjs 時直接擋下。
func TestMigrateServerKitNode(t *testing.T) {
	requireEmbeddedSDK(t)
	root := newProject(t, "web-node")
	t.Chdir(root)
	write(t, root, "server/session.mjs", "export const createHub = () => {};\n")
	write(t, root, "server/main.mjs", "import { createHub } from './session.mjs';\n")
	must(t, cmdMigrate([]string{"server-kit"}))
	if !exists(root, "server/main.mjs.pt-new") {
		t.Fatal("要寫出新範本")
	}
	if err := cmdMigrate([]string{"server-kit", "-apply"}); err == nil || !strings.Contains(err.Error(), "session.mjs") {
		t.Fatalf("還在 import 舊檔要擋下：%v", err)
	}
	if !exists(root, "server/session.mjs") {
		t.Fatal("不能刪掉")
	}
}
