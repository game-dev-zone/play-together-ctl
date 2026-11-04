package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	testThree  = "https://cdn.jsdelivr.net/npm/three@0.186.1/build/three.module.min.js"
	testPhaser = "https://cdn.jsdelivr.net/npm/phaser@4.2.1/dist/phaser.min.js"
	testCSS    = "https://cdnjs.cloudflare.com/ajax/libs/animate.css/4.1.1/animate.min.css"
)

var testSRI = "sha384-" + strings.Repeat("A", 64)

func TestCDNURLError(t *testing.T) {
	for _, u := range []string{testThree, testPhaser, testCSS} {
		if err := cdnURLError(u); err != nil {
			t.Errorf("%s 應該通過：%v", u, err)
		}
	}
	for _, u := range []string{
		"https://unpkg.com/three@0.186.1/build/three.module.js",
		"https://cdn.jsdelivr.net/npm/three@latest/build/three.module.js",
		"https://cdn.jsdelivr.net/npm/three@0.186/build/three.module.js",
		"https://cdn.jsdelivr.net/npm/three@0.186.1/",
		"https://cdn.jsdelivr.net/npm/three@0.186.1/build/three.wasm",
		"https://cdn.jsdelivr.net/npm/three@0.186.1/a.js?x=1",
		"http://cdn.jsdelivr.net/npm/three@0.186.1/a.js",
		"https://fonts.googleapis.com/css2?family=Noto+Sans+TC",
		"https://cdn.jsdelivr.net/npm/three@0.186.1/a.js; script-src *",
	} {
		if cdnURLError(u) == nil {
			t.Errorf("%s 應該被拒絕", u)
		}
	}
}

func TestCheckCDN(t *testing.T) {
	run := func(t *testing.T, declared []cdnAsset, files map[string]string) []finding {
		t.Helper()
		dir := t.TempDir()
		for name, body := range files {
			must(t, os.MkdirAll(filepath.Dir(filepath.Join(dir, name)), 0o755))
			must(t, os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644))
		}
		var out []finding
		checkCDN(&project{CDN: declared}, dir, func(level, format string, a ...any) {
			out = append(out, finding{level, fmt.Sprintf(format, a...)})
		})
		return out
	}
	ok := []cdnAsset{{testPhaser, testSRI}, {testCSS, testSRI}, {testThree, testSRI}}
	good := map[string]string{
		"index.html": `<script src="` + testPhaser + `" integrity="` + testSRI + `" crossorigin="anonymous" referrerpolicy="no-referrer"></script>
<link rel="stylesheet" href="` + testCSS + `" integrity="` + testSRI + `" crossorigin="anonymous">
<script type="importmap">{"imports": {"three": "` + testThree + `"}, "integrity": {"` + testThree + `": "` + testSRI + `"}}</script>
<script type="module" src="game.js"></script>`,
		"game.js": `import * as THREE from "three";`,
	}
	if fs := run(t, ok, good); hasFinding(fs, "error", "") {
		t.Fatalf("申報且寫好 SRI 應該通過：%v", fs)
	}
	cases := []struct {
		name     string
		declared []cdnAsset
		files    map[string]string
		want     string
	}{
		{"沒有申報", nil, map[string]string{"index.html": `<script src="` + testPhaser + `" integrity="` + testSRI + `" crossorigin="anonymous"></script>`}, "沒有申報"},
		{"沒有 integrity", ok[:1], map[string]string{"index.html": `<script src="` + testPhaser + `" crossorigin="anonymous"></script>`}, "integrity"},
		{"integrity 不同", ok[:1], map[string]string{"index.html": `<script src="` + testPhaser + `" integrity="sha384-` + strings.Repeat("B", 64) + `" crossorigin="anonymous"></script>`}, "integrity"},
		{"沒有 crossorigin", ok[:1], map[string]string{"index.html": `<script src="` + testPhaser + `" integrity="` + testSRI + `"></script>`}, "crossorigin"},
		{"use-credentials", ok[:1], map[string]string{"index.html": `<script src="` + testPhaser + `" integrity="` + testSRI + `" crossorigin="use-credentials"></script>`}, "crossorigin"},
		{"外部字型 preload", nil, map[string]string{"index.html": `<link rel="preload" href="https://fonts.gstatic.com/s/x.woff2" as="font">`}, "字型"},
		{"Google Fonts 樣式", nil, map[string]string{"index.html": `<link rel="stylesheet" href="https://fonts.googleapis.com/css2?family=Noto+Sans+TC">`}, "沒有申報"},
		{"其他網站的程式", nil, map[string]string{"index.html": `<script src="https://evil.example/x.js"></script>`}, "沒有申報"},
		{"JS 直接 import 網址", ok, map[string]string{"index.html": "<p>", "game.js": `import * as THREE from "` + testThree + `";`}, "直接使用 CDN 網址"},
		{"JS 動態載入 unpkg", nil, map[string]string{"index.html": "<p>", "game.js": `await import("https://unpkg.com/three@0.186.1/build/three.module.js")`}, "直接使用 CDN 網址"},
		{"CSS 從 CDN 載字型", nil, map[string]string{"index.html": "<p>", "style.css": `@font-face { src: url(https://cdn.jsdelivr.net/npm/font@1.0.0/a.woff2) }`}, "直接使用 CDN 網址"},
		{"import map 沒有 integrity", ok[2:], map[string]string{"index.html": `<script type="importmap">{"imports": {"three": "` + testThree + `"}}</script>`}, "import map"},
		{"申報格式錯誤", []cdnAsset{{"https://cdn.jsdelivr.net/npm/three@latest/a.js", testSRI}}, map[string]string{"index.html": "<p>"}, "playtogether.json 的 cdn"},
		{"申報沒有 SRI", []cdnAsset{{testThree, ""}}, map[string]string{"index.html": "<p>"}, "SRI"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if fs := run(t, c.declared, c.files); !hasFinding(fs, "error", c.want) {
				t.Fatalf("應該報錯（%s）：%v", c.want, fs)
			}
		})
	}
}

func TestCmdCDNAdd(t *testing.T) {
	dir := t.TempDir()
	must(t, os.WriteFile(filepath.Join(dir, projectFile), []byte(`{"name":"x","client":"web","server":"go","web":"web"}`), 0o644))
	t.Chdir(dir)
	orig := fetchSRI
	fetchSRI = func(string) (string, error) { return testSRI, nil }
	t.Cleanup(func() { fetchSRI = orig })
	if err := cmdCDN([]string{"add", "https://cdn.jsdelivr.net/npm/three@latest/a.js"}); err == nil {
		t.Fatal("不合格的網址要拒絕")
	}
	must(t, cmdCDN([]string{"add", testPhaser}))
	if err := cmdCDN([]string{"add", testPhaser}); err == nil {
		t.Fatal("重複申報要拒絕")
	}
	p, err := findProject()
	must(t, err)
	if len(p.CDN) != 1 || p.CDN[0].URL != testPhaser || p.CDN[0].Integrity != testSRI {
		t.Fatalf("要記到 playtogether.json：%+v", p.CDN)
	}
}
