package main

// pt migrate server-kit：pt v0.18 之前的專案，平台接口寫在 server/session.go、limit.go（Node：session.mjs、limit.mjs），
// 改用 SDK 的遊戲伺服器套件（Go gameserver、Node createGameServer）。遊戲規則要搬到新的寫法（abort → close、
// /action → onAction、自己回報一局 → room.startRound／endRound），這一步需要人（或 AI 助手）改程式，所以分兩步：
//
//	pt migrate server-kit          寫出新範本 server/main.go.pt-new 對照，以及 .pt/migrate-server-kit.md（步驟與舊程式要改的地方）
//	pt migrate server-kit -apply   搬好之後：舊檔備份到 .pt/backup/ 再刪除，檢查伺服器可以編譯；失敗就還原

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const migrateGuide = ".pt/migrate-server-kit.md"

type legacyServer struct {
	main, ext string   // server/main.go、.go
	old       []string // 舊的平台接口檔案（存在的）
	patterns  [][2]string
}

func findLegacyServer(p *project) (*legacyServer, error) {
	var ls legacyServer
	switch p.Server {
	case "go":
		ls = legacyServer{main: "server/main.go", ext: ".go", patterns: [][2]string{
			{"abort()", "abort 改成 Close（不用回傳局 ID，套件會自動回報中止）"},
			{"POST /action", "/action 的處理改寫成 OnAction（套件負責 token、限流、座位）"},
			{"ReportRoundStart", "改成 room.StartRound"},
			{"ReportRoundEnd", "改成 room.EndRound"},
			{"s.hub", "hub 由套件管理：改用 OnAction／Tick 傳進來的 room"},
			{"func cors(", "CORS 由套件處理（ALLOW_ORIGINS）：刪掉"},
			{"actionPerPlayer", "限流由套件處理：刪掉"},
		}}
		for _, f := range []string{"server/session.go", "server/limit.go"} {
			if _, err := os.Stat(p.path(f)); err == nil {
				ls.old = append(ls.old, f)
			}
		}
	case "node":
		ls = legacyServer{main: "server/main.mjs", ext: ".mjs", patterns: [][2]string{
			{"./session.mjs", "刪掉這個 import：改用 createGameServer"},
			{"./limit.mjs", "刪掉這個 import：限流由套件處理"},
			{"abort()", "abort 改成 close（不用回傳局 ID）"},
			{"'/action'", "/action 的處理改寫成 onAction"},
			{"reportRoundStart", "改成 room.startRound"},
			{"reportRoundEnd", "改成 room.endRound"},
		}}
		for _, f := range []string{"server/session.mjs", "server/limit.mjs"} {
			if _, err := os.Stat(p.path(f)); err == nil {
				ls.old = append(ls.old, f)
			}
		}
	default:
		return nil, errors.New("這個專案沒有範本伺服器（server 不是 go 或 node）")
	}
	if len(ls.old) == 0 {
		return nil, nil
	}
	return &ls, nil
}

// serverKitInstalled：專案的 SDK 已經有遊戲伺服器套件（pt sdk update 過）。
func serverKitInstalled(p *project) bool {
	if p.Server == "go" {
		_, err := os.Stat(p.path("server/third_party/playtogether-sdk-go/gameserver"))
		return err == nil
	}
	raw, err := os.ReadFile(p.path("server/playtogether-server.mjs"))
	return err == nil && bytes.Contains(raw, []byte("createGameServer"))
}

// scanLegacy 列出舊 main 檔裡要改的地方（行號＋說明）。
func scanLegacy(content []byte, patterns [][2]string) []string {
	var out []string
	sc := bufio.NewScanner(bytes.NewReader(content))
	sc.Buffer(make([]byte, 1024*1024), 1024*1024)
	for n := 1; sc.Scan(); n++ {
		line := sc.Text()
		for _, pt := range patterns {
			if strings.Contains(line, pt[0]) {
				out = append(out, fmt.Sprintf("第 %d 行 `%s`：%s", n, strings.TrimSpace(line), pt[1]))
			}
		}
	}
	return out
}

func cmdMigrate(args []string) error {
	if len(args) == 0 || args[0] != "server-kit" {
		return errors.New("用法：pt migrate server-kit [-apply]（舊專案改用 SDK 的遊戲伺服器套件）")
	}
	fs := flag.NewFlagSet("migrate server-kit", flag.ExitOnError)
	apply := fs.Bool("apply", false, "遊戲規則搬好之後：備份並刪除舊的平台接口檔案，確認伺服器可以編譯")
	_ = fs.Parse(args[1:])
	if err := requireAssets(); err != nil {
		return err
	}
	p, err := findProject()
	if err != nil {
		return err
	}
	if p == nil {
		return errors.New("找不到 playtogether.json：請在用 pt new 建立的專案裡執行")
	}
	if p.existing() {
		return errors.New("pt init 接入的既有專案沒有範本伺服器，不需要遷移")
	}
	ls, err := findLegacyServer(p)
	if err != nil {
		return err
	}
	if ls == nil {
		fmt.Println("這個專案已經在用遊戲伺服器套件（沒有 session、limit 檔案），不需要遷移。")
		return nil
	}
	if !serverKitInstalled(p) {
		return errors.New("專案的 SDK 還沒有遊戲伺服器套件：先執行 pt sdk update（只加入套件，不會動你的檔案）")
	}
	if *apply {
		return migrateApply(p, ls)
	}
	return migratePrepare(p, ls)
}

func migratePrepare(p *project, ls *legacyServer) error {
	in, err := inputsFromProject(p)
	if err != nil {
		return err
	}
	files, _, err := renderProject(in)
	if err != nil {
		return err
	}
	ref := files[ls.main]
	if ref == nil {
		return fmt.Errorf("這個 pt 的範本沒有 %s", ls.main)
	}
	if err := os.WriteFile(p.path(ls.main+".pt-new"), ref, 0o644); err != nil {
		return err
	}
	old, err := os.ReadFile(p.path(ls.main))
	if err != nil {
		return err
	}
	hits := scanLegacy(old, ls.patterns)
	var b strings.Builder
	fmt.Fprintf(&b, "# 改用 SDK 的遊戲伺服器套件（pt migrate server-kit，%s）\n\n", version)
	fmt.Fprintf(&b, "舊的平台接口：%s。它們仍可使用，但 pt sdk update 更新不到（限流、413、/healthz、斷線重連等之後的修正都不會套用）。\n\n", strings.Join(ls.old, "、"))
	b.WriteString("## 步驟\n\n")
	fmt.Fprintf(&b, "1. 對照新範本 `%s.pt-new`：平台接口都在套件裡，範本只剩遊戲規則。\n", ls.main)
	fmt.Fprintf(&b, "2. 把你的遊戲搬過去（直接改 `%s`，或把規則搬進 `.pt-new` 再改名蓋過去）：view、canFree、leave 不變；abort 改成 close；/action 的處理改寫成 onAction；自己回報一局的地方改成 room.startRound、room.endRound。\n", ls.main)
	b.WriteString("3. 執行 `pt migrate server-kit -apply`：舊檔備份到 `.pt/backup/` 後刪除，並確認伺服器可以編譯（失敗會還原）。\n")
	b.WriteString("4. 執行 `make verify`（含 `pt verify server`）確認平台接口正常，再用 `pt dev` 實際玩一局。\n")
	b.WriteString("\n詳細說明：玩聚開發者文件站 /docs/game-server「舊專案改用套件」。\n")
	if len(hits) > 0 {
		fmt.Fprintf(&b, "\n## `%s` 裡要改的地方\n\n", ls.main)
		for _, h := range hits {
			b.WriteString("- " + h + "\n")
		}
	}
	if err := os.MkdirAll(p.path(".pt"), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(p.path(filepath.FromSlash(migrateGuide)), []byte(b.String()), 0o644); err != nil {
		return err
	}
	fmt.Printf("已寫出：\n    %s.pt-new（新範本，對照用）\n    %s（步驟與 %d 處要改的地方）\n\n", ls.main, migrateGuide, len(hits))
	fmt.Printf("把遊戲規則搬到新的寫法後，執行 pt migrate server-kit -apply。用 AI 助手的話，請它照 %s 做。\n", migrateGuide)
	return nil
}

// buildServer 檢查伺服器可以編譯（Go：go build；Node：node --check）。測試時換掉。
var buildServer = func(p *project) error {
	var cmd *exec.Cmd
	if p.Server == "go" {
		cmd = exec.Command("go", "build", "-o", os.DevNull, ".")
	} else {
		cmd = exec.Command("node", "--check", "main.mjs")
	}
	cmd.Dir = p.path("server")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w\n%s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func migrateApply(p *project, ls *legacyServer) error {
	main, err := os.ReadFile(p.path(ls.main))
	if err != nil {
		return err
	}
	if p.Server == "node" {
		for _, imp := range []string{"./session.mjs", "./limit.mjs"} {
			if bytes.Contains(main, []byte(imp)) {
				return fmt.Errorf("%s 還在 import %s：先照 %s 把遊戲規則搬到新的寫法", ls.main, imp, migrateGuide)
			}
		}
	}
	backup := filepath.Join(p.path(".pt"), "backup", time.Now().Format("20060102-150405"))
	moved := map[string]string{}
	restore := func() {
		for src, dst := range moved {
			_ = os.Rename(dst, src)
		}
	}
	for _, f := range ls.old {
		src := p.path(filepath.FromSlash(f))
		dst := filepath.Join(backup, filepath.FromSlash(f))
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			restore()
			return err
		}
		if err := os.Rename(src, dst); err != nil {
			restore()
			return err
		}
		moved[src] = dst
	}
	if err := buildServer(p); err != nil {
		restore()
		return fmt.Errorf("拿掉 %s 之後伺服器編譯失敗，已經還原：\n%v\n先照 %s 把遊戲規則搬到新的寫法", strings.Join(ls.old, "、"), err, migrateGuide)
	}
	_ = os.Remove(p.path(ls.main + ".pt-new"))
	_ = os.Remove(p.path(filepath.FromSlash(migrateGuide)))
	rel, _ := filepath.Rel(p.dir, backup)
	fmt.Printf("已改用遊戲伺服器套件：%s 移到 %s（確認沒問題後可以刪掉）。\n", strings.Join(ls.old, "、"), rel)
	fmt.Println("接著執行 make verify（含 pt verify server）確認平台接口正常，再用 pt dev 實際玩一局。")
	return nil
}
