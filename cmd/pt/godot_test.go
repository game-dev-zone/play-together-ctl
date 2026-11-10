package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewClientServerFlags(t *testing.T) {
	requireEmbeddedSDK(t)
	t.Chdir(t.TempDir())
	for _, args := range [][]string{{"bad-client", "-client", "unity"}, {"bad-server", "-server", "rust"}, {"bad-tpl", "-template", "godot-rust"}} {
		if err := cmdNew(args); err == nil {
			t.Fatalf("應拒絕 %v", args)
		}
	}
	if err := cmdNew([]string{"old-style", "-template", "godot-node"}); err != nil {
		t.Fatal(err)
	}
	t.Chdir("old-style")
	p, _ := findProject()
	if p.Client != "godot" || p.Server != "node" || p.Web != "build" {
		t.Fatalf("-template godot-node 要對應到 client=godot、server=node：%+v", p)
	}
	if _, err := os.Stat("server/playtogether-server.mjs"); err != nil {
		t.Fatal("Node 伺服器要有 SDK")
	}
}

func TestNewGodot(t *testing.T) {
	requireEmbeddedSDK(t)
	dir := t.TempDir()
	t.Chdir(dir)
	must(t, cmdNew([]string{"party-g", "-client", "godot", "-server", "go", "-platform", "https://pt.example"}))
	root := filepath.Join(dir, "party-g")
	for _, f := range []string{"game/project.godot", "game/export_presets.cfg", "game/main.gd", "game/main.tscn", "game/config.gd",
		"game/addons/playtogether/playtogether.gd", "game/addons/playtogether/plugin.cfg", "game/fonts/NotoSansTC-Common.otf", "game/fonts/OFL.txt",
		"server/main.go", "server/third_party/playtogether-sdk-go/client.go", ".gitignore", "playtogether.json"} {
		if _, err := os.Stat(filepath.Join(root, f)); err != nil {
			t.Fatalf("少了 %s", f)
		}
	}
	read := func(f string) string { b, _ := os.ReadFile(filepath.Join(root, f)); return string(b) }
	if !strings.Contains(read("game/project.godot"), `theme/custom_font="res://fonts/NotoSansTC-Common.otf"`) ||
		!strings.Contains(read("game/project.godot"), "PlayTogether=") {
		t.Fatal("project.godot 要設定中文字型與 PlayTogether autoload")
	}
	if !strings.Contains(read("game/config.gd"), `"https://pt.example"`) {
		t.Fatal("config.gd 要帶入平台網址")
	}
	if !strings.Contains(read(".gitignore"), "build/") || !strings.Contains(read(".gitignore"), "game/.godot/") {
		t.Fatal(".gitignore 要排除匯出結果與 Godot 快取")
	}
	if strings.Contains(read("game/main.gd"), "JavaScriptBridge.eval") {
		t.Fatal("範本不能用 JavaScriptBridge.eval")
	}
}

func TestCheckGodot(t *testing.T) {
	requireEmbeddedSDK(t)
	dir := t.TempDir()
	t.Chdir(dir)
	must(t, cmdNew([]string{"party-g", "-client", "godot"}))
	root := filepath.Join(dir, "party-g")
	t.Chdir(root)
	p, _ := findProject()
	p.Platform, p.ServerURL = "https://pt.example", "https://game.example.com"
	has := func(fs []finding, sub string) bool {
		for _, f := range fs {
			if isErrorLevel(f.level) && strings.Contains(f.msg, sub) {
				return true
			}
		}
		return false
	}
	fs := checkProject(p, filepath.Join(root, "build"))
	for _, want := range []string{"pt build", "SERVER_URL", "PLATFORM_ORIGINS", "playtogether-sdk-godot.iife.js"} {
		if !has(fs, want) {
			t.Fatalf("要提醒 %s：%+v", want, fs)
		}
	}
	preset := filepath.Join(root, "game/export_presets.cfg")
	raw, _ := os.ReadFile(preset)
	must(t, os.WriteFile(preset, []byte(strings.Replace(string(raw), "variant/thread_support=false", "variant/thread_support=true", 1)), 0o644))
	must(t, os.WriteFile(filepath.Join(root, "game/bad.gd"), []byte(`extends Node
func _ready(): JavaScriptBridge.eval("1")`), 0o644))
	must(t, os.WriteFile(filepath.Join(root, "game/p2p.gd"), []byte(`extends Node
var peer := WebRTCPeerConnection.new()`), 0o644))
	must(t, os.WriteFile(filepath.Join(root, "game/mic.gd"), []byte(`extends Node
func _ready(): JavaScriptBridge.get_interface("navigator").mediaDevices.getUserMedia(null)`), 0o644))
	must(t, os.WriteFile(filepath.Join(root, "game/speech.gd"), []byte(`extends Node
var rec := JavaScriptBridge.create_object("webkitSpeechRecognition")`), 0o644))
	fs = checkProject(p, filepath.Join(root, "build"))
	if !has(fs, "執行緒") || !has(fs, "JavaScriptBridge.eval") || !has(fs, "p2p.gd 使用 WebRTC") {
		t.Fatalf("要擋下執行緒支援、eval 與 WebRTC：%+v", fs)
	}
	if !has(fs, "mic.gd 使用麥克風") || !has(fs, "speech.gd 使用麥克風") {
		t.Fatalf("要擋下 GDScript 透過 JavaScriptBridge 使用麥克風與語音辨識：%+v", fs)
	}
}

func TestTemplateVersion(t *testing.T) {
	for in, want := range map[string]string{"4.7.2.stable.official.ed1daf0bf": "4.7.2.stable", "4.3.stable.official.77dcf97d8": "4.3.stable", "4.5.rc1.official.abc": "4.5.rc1"} {
		if got := templateVersion(in); got != want {
			t.Fatalf("%s → %s，預期 %s", in, got, want)
		}
	}
}

// 有 Godot 時（CI 的 godot 工作）：實際匯出網頁版。
func TestGodotBuild(t *testing.T) {
	requireEmbeddedSDK(t)
	if _, err := findGodot(); err != nil {
		t.Skip("沒有 Godot")
	}
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("沒有 go")
	}
	dir := t.TempDir()
	t.Chdir(dir)
	must(t, cmdNew([]string{"party-g", "-client", "godot"}))
	t.Chdir(filepath.Join(dir, "party-g"))
	if err := cmdBuild(nil); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"index.html", "index.wasm", "index.pck", "index.js", "playtogether-sdk-godot.iife.js"} {
		if _, err := os.Stat(filepath.Join("build", f)); err != nil {
			t.Fatalf("匯出結果少了 %s", f)
		}
	}
	html, _ := os.ReadFile("build/index.html")
	if !strings.Contains(string(html), "playtogether-sdk-godot.iife.js") {
		t.Fatal("index.html 要載入 SDK 轉接檔")
	}
}

func TestSDKExport(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "sdk")
	if err := exportSDK(dir); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"browser/playtogether-sdk.iife.js", "browser/playtogether-sdk.js", "godot/playtogether-sdk-godot.iife.js", "godot/addons/playtogether/plugin.cfg", "go/go.mod", "go/client.go", "node/playtogether-server.mjs", "VERSION"} {
		if _, err := os.Stat(filepath.Join(dir, f)); err != nil {
			t.Errorf("缺少 %s", f)
		}
	}
	// 已有檔案的資料夾要拒絕，避免覆蓋
	if err := exportSDK(dir); err == nil {
		t.Fatal("非空資料夾應該拒絕")
	}
}

// Godot 引擎檔內建 WebRTC 程式碼不能擋上傳；GDScript 真的用到 WebRTC 才擋
func TestGodotEngineWebRTC(t *testing.T) {
	dir := t.TempDir()
	must(t, os.WriteFile(filepath.Join(dir, "index.html"), []byte(`<script src="playtogether-sdk-godot.iife.js"></script>`), 0o644))
	must(t, os.WriteFile(filepath.Join(dir, "index.js"), []byte("var Engine=1;function GodotRTCPeerConnection(){new RTCPeerConnection()}"), 0o644))
	must(t, os.WriteFile(filepath.Join(dir, "index.wasm"), []byte("x"), 0o644))
	must(t, os.WriteFile(filepath.Join(dir, "game.js"), []byte("new RTCPeerConnection()"), 0o644))
	var engine, other string
	for _, f := range checkProject(nil, dir) {
		if strings.Contains(f.msg, "index.js") {
			engine = f.level
		}
		if strings.Contains(f.msg, "game.js") && strings.Contains(f.msg, "WebRTC") {
			other = f.level
		}
	}
	if engine != "info" || other != levelHard {
		t.Fatalf("引擎檔應是 info、其他檔案仍是 error：%q %q", engine, other)
	}
}
