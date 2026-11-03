package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewOrientation(t *testing.T) {
	requireEmbeddedSDK(t)
	for _, client := range []string{"web", "godot"} {
		for _, o := range orientations {
			t.Run(client+"-"+o, func(t *testing.T) {
				dir := t.TempDir()
				t.Chdir(dir)
				must(t, cmdNew([]string{"or-game", "-client", client, "-server", "go", "-orientation", o, "-y"}))
				root := filepath.Join(dir, "or-game")
				read := func(rel string) string {
					b, err := os.ReadFile(filepath.Join(root, rel))
					must(t, err)
					return string(b)
				}
				p := readProject(t, root)
				if p.Orientation != o {
					t.Fatalf("playtogether.json 要記錄方向 %s：%+v", o, p)
				}
				pg := read("PLAYTOGETHER.md")
				if !strings.Contains(pg, "畫面方向："+orientationLabel[o]) {
					t.Fatalf("PLAYTOGETHER.md 要寫明方向：%s", pg)
				}
				if client == "godot" {
					proj := read("game/project.godot")
					want := map[string]string{"portrait": "viewport_width=390\nwindow/size/viewport_height=844", "landscape": "viewport_width=844\nwindow/size/viewport_height=390", "any": "viewport_width=390\nwindow/size/viewport_height=844"}[o]
					hand := map[string]string{"portrait": "orientation=1", "landscape": "orientation=0", "any": "orientation=6"}[o]
					if !strings.Contains(proj, want) || !strings.Contains(proj, "window/handheld/"+hand) {
						t.Fatalf("project.godot 的畫面大小與方向不對：\n%s", proj)
					}
					if !strings.Contains(read("game/config.gd"), `ORIENTATION := "`+o+`"`) || !strings.Contains(read("game/main.gd"), "set_orientation(Config.ORIENTATION)") {
						t.Fatal("config.gd 要有 ORIENTATION，main.gd 要告訴平台方向")
					}
				} else {
					css, html := read("web/style.css"), read("web/index.html")
					if strings.Contains(css, "[[") || strings.Contains(html, "[[") {
						t.Fatal("範本沒有展開")
					}
					if got := strings.Contains(css, "@media (orientation: landscape)"); got != (o != "portrait") {
						t.Fatalf("%s：橫拿版面只在 landscape、any 出現", o)
					}
					if got := strings.Contains(html, `id="rotate"`); got != (o == "landscape") {
						t.Fatalf("%s：「請轉成橫式」只在 landscape 出現", o)
					}
					if !strings.Contains(read("web/config.js"), "orientation: '"+o+"'") || !strings.Contains(read("web/game.js"), "pt.ui.orientation(") {
						t.Fatal("config.js 要有 orientation，game.js 要告訴平台方向")
					}
				}
				// 新專案的方向設定一致：pt check 不該有方向的警告
				t.Chdir(root)
				pp, _ := findProject()
				var msgs []string
				checkOrientation(pp, pp.path(pp.Web), func(level, format string, a ...any) { msgs = append(msgs, level+" "+format) })
				if len(msgs) > 0 {
					t.Fatalf("新專案不該有方向警告：%v", msgs)
				}
			})
		}
	}
}

func TestCheckOrientationMismatch(t *testing.T) {
	requireEmbeddedSDK(t)
	dir := t.TempDir()
	t.Chdir(dir)
	must(t, cmdNew([]string{"or-g", "-client", "godot", "-server", "go", "-orientation", "portrait", "-y"}))
	must(t, cmdNew([]string{"or-w", "-client", "web", "-server", "go", "-orientation", "landscape", "-y"}))
	collect := func(root string) []finding {
		t.Chdir(root)
		p, _ := findProject()
		var out []finding
		checkOrientation(p, p.path(p.Web), func(level, format string, a ...any) {
			out = append(out, finding{level, format})
		})
		return out
	}
	// Godot：只改 playtogether.json 成橫式，畫面大小與 config.gd 還是直式
	g := filepath.Join(dir, "or-g")
	raw, _ := os.ReadFile(filepath.Join(g, projectFile))
	must(t, os.WriteFile(filepath.Join(g, projectFile), []byte(strings.Replace(string(raw), `"orientation": "portrait"`, `"orientation": "landscape"`, 1)), 0o644))
	if fs := collect(g); !hasFinding(fs, "warn", "畫面大小") || !hasFinding(fs, "warn", "兩邊要一致") {
		t.Fatalf("Godot 的方向不一致要警告：%+v", fs)
	}
	// 網頁：config.js 被改成直式
	w := filepath.Join(dir, "or-w")
	cfg, _ := os.ReadFile(filepath.Join(w, "web/config.js"))
	must(t, os.WriteFile(filepath.Join(w, "web/config.js"), []byte(strings.Replace(string(cfg), "orientation: 'landscape'", "orientation: 'portrait'", 1)), 0o644))
	if fs := collect(w); !hasFinding(fs, "warn", "兩邊要一致") {
		t.Fatalf("網頁的方向不一致要警告：%+v", fs)
	}
	// 不認得的值是錯誤
	raw, _ = os.ReadFile(filepath.Join(w, projectFile))
	must(t, os.WriteFile(filepath.Join(w, projectFile), []byte(strings.Replace(string(raw), `"orientation": "landscape"`, `"orientation": "upside"`, 1)), 0o644))
	if fs := collect(w); !hasFinding(fs, "error", "orientation 只能是") {
		t.Fatalf("不認得的方向要是錯誤：%+v", fs)
	}
}
