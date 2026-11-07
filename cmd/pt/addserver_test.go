package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// 單機的既有專案（pt init -game-server none）
func newSinglePlayer(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "solo-game")
	write(t, dir, "web/index.html", `<!doctype html><script src="playtogether-sdk.iife.js"></script><script src="game.js"></script>`)
	write(t, dir, "web/game.js", "// 單機遊戲\n")
	t.Chdir(dir)
	must(t, cmdInit([]string{"-y", "-game-server", "none", "-platform", "https://pt.example"}))
	return dir
}

func TestAddServer(t *testing.T) {
	requireEmbeddedSDK(t)
	dir := newSinglePlayer(t)

	if err := cmdAdd([]string{"nope"}); err == nil {
		t.Fatal("未知的子指令要拒絕")
	}
	if err := cmdAdd([]string{"server", "-server", "rust"}); err == nil {
		t.Fatal("不支援的語言要拒絕")
	}
	if err := cmdAdd([]string{"server", "-server", "node", "-hosting", "playtogether"}); err == nil {
		t.Fatal("玩聚代管只支援 Go")
	}
	if exists(dir, "server") {
		t.Fatal("被拒絕時不能寫檔案")
	}

	must(t, cmdAdd([]string{"server", "-server", "go"}))
	for _, f := range []string{"server/main.go", "server/go.mod", "server/third_party/playtogether-sdk-go/gameserver/gameserver.go",
		netHelperPath, ".claude/skills/playtogether-multiplayer/SKILL.md", ".agents/skills/playtogether-multiplayer/SKILL.md"} {
		if !exists(dir, f) {
			t.Errorf("沒有產生 %s", f)
		}
	}
	if read(t, dir, "web/game.js") != "// 單機遊戲\n" {
		t.Fatal("不能動原本的遊戲程式")
	}
	if exists(dir, "web/playtogether-sdk.iife.js") {
		t.Fatal("既有專案的前端 SDK 由開發者自己管理")
	}
	p, err := findProject()
	must(t, err)
	if p.Server != "go" || !p.existing() || p.gameServerMode() != defaultGameServer || !p.multiplayerEnabled() || p.SDKVersion != sdkVersion() || p.hostedServer() {
		t.Fatalf("playtogether.json：%+v", p)
	}
	m, err := loadManifest(dir)
	must(t, err)
	if f, ok := m.get("server/third_party/playtogether-sdk-go/client.go"); !ok || f.Kind != kindSDK {
		t.Fatal("伺服器的 SDK 要記基線")
	}
	for _, f := range m.Files {
		if !strings.HasPrefix(f.Path, "server/") {
			t.Fatalf("只記伺服器的 SDK：%s", f.Path)
		}
	}

	// 再加一次：拒絕
	if err := cmdAdd([]string{"server", "-server", "go"}); err == nil {
		t.Fatal("已經有伺服器時要拒絕")
	}
	// pt upgrade 仍然不適用既有專案
	if err := cmdUpgrade([]string{"-y"}); err == nil {
		t.Fatal("既有專案不適用 pt upgrade")
	}

	// pt sdk update：只更新伺服器的 SDK；被改過的不覆蓋
	gs := filepath.Join(dir, "server/third_party/playtogether-sdk-go/gameserver/gameserver.go")
	orig := read(t, dir, "server/third_party/playtogether-sdk-go/gameserver/gameserver.go")
	must(t, os.WriteFile(gs, []byte(orig+"// 改過\n"), 0o644))
	if err := cmdSDK([]string{"update"}); err == nil || !strings.Contains(err.Error(), "被改過") {
		t.Fatalf("改過的 SDK 要停下來：%v", err)
	}
	must(t, os.Remove(gs))
	must(t, cmdSDK([]string{"update"}))
	if read(t, dir, "server/third_party/playtogether-sdk-go/gameserver/gameserver.go") != orig {
		t.Fatal("pt sdk update 要補回伺服器的 SDK")
	}
	if exists(dir, "web/playtogether-sdk.iife.js") {
		t.Fatal("pt sdk update 不能寫既有專案的前端 SDK")
	}

	// pt check：還沒填 serverUrl 只是提醒（自己部署）
	if err := cmdCheck([]string{"-local"}); err != nil && !strings.Contains(err.Error(), "serverUrl") {
		t.Fatalf("pt check -local：%v", err)
	}
}

func TestAddServerRefuses(t *testing.T) {
	requireEmbeddedSDK(t)
	// pt new 的專案已經有伺服器
	root := newProject(t, "web-go")
	t.Chdir(root)
	if err := cmdAdd([]string{"server", "-server", "go"}); err == nil || !strings.Contains(err.Error(), "pt new") {
		t.Fatalf("pt new 的專案要拒絕：%v", err)
	}
	// 既有專案已經有自己的伺服器網址
	dir := newSinglePlayer(t)
	var raw map[string]any
	must(t, json.Unmarshal([]byte(read(t, dir, projectFile)), &raw))
	raw["serverUrl"] = "https://game.example"
	b, _ := json.Marshal(raw)
	write(t, dir, projectFile, string(b))
	if err := cmdAdd([]string{"server", "-server", "go"}); err == nil || !strings.Contains(err.Error(), "serverUrl") {
		t.Fatalf("已經有自己的伺服器要拒絕：%v", err)
	}
	// 已經有 server/ 資料夾
	raw["serverUrl"] = ""
	b, _ = json.Marshal(raw)
	write(t, dir, projectFile, string(b))
	write(t, dir, "server/README.md", "我的\n")
	if err := cmdAdd([]string{"server", "-server", "go"}); err == nil || !strings.Contains(err.Error(), "server/") {
		t.Fatalf("已經有 server/ 要拒絕：%v", err)
	}
}

// 加上的伺服器實際通過 pt verify server（Go、Node；代管）。
func TestAddServerVerify(t *testing.T) {
	if testing.Short() {
		t.Skip("要實際跑遊戲伺服器")
	}
	requireEmbeddedSDK(t)
	for _, c := range []struct{ server, hosting string }{{"go", ""}, {"node", ""}, {"go", "playtogether"}} {
		t.Run(c.server+c.hosting, func(t *testing.T) {
			if c.server == "node" {
				if _, err := exec.LookPath("node"); err != nil {
					t.Skip("沒有 node")
				}
			}
			newSinglePlayer(t)
			t.Setenv("GOWORK", "off")
			args := []string{"server", "-server", c.server}
			if c.hosting != "" {
				args = append(args, "-hosting", c.hosting)
			}
			must(t, cmdAdd(args))
			p, _ := findProject()
			if c.hosting != "" && (!p.hostedServer() || serverDir(p.GameServer) != "server") {
				t.Fatalf("代管設定：%+v", p.GameServer)
			}
			if err := cmdVerifyServer([]string{"-json"}); err != nil {
				t.Fatal(err)
			}
		})
	}
}
