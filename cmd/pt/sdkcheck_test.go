package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompareSDK(t *testing.T) {
	for _, c := range []struct {
		a, b string
		want int
		ok   bool
	}{
		{"0.1.0-20261101", "0.1.0-20261101", 0, true},
		{"0.1.0-20261101", "0.1.0-20261115", -1, true},
		{"0.1.0-20261201", "0.2.0-20261101", -1, true},
		{"0.10.0-20261101", "0.9.0-20261231", 1, true},
		{"0.1.0", "0.1.0-20261101", -1, true},
		{"0.0.1", "0.1.0-20261101", -1, true},
		{"", "0.1.0-20261101", 0, false},
		{"0.1.0-rc1", "0.1.0-20261101", 0, false},
		{"0.1.0-20261101\n", "0.1.0-20261101", 0, false},
	} {
		got, ok := compareSDK(c.a, c.b)
		if got != c.want || ok != c.ok {
			t.Errorf("compareSDK(%q, %q) = %d %v", c.a, c.b, got, ok)
		}
	}
}

// setProjectSDK 把專案記錄的 SDK 版本改成 v。
func setProjectSDK(t *testing.T, root, v string) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(root, projectFile))
	must(t, err)
	must(t, os.WriteFile(filepath.Join(root, projectFile), bytes.Replace(raw, []byte(`"sdkVersion": "`+sdkVersion()+`"`), []byte(`"sdkVersion": "`+v+`"`), 1), 0o644))
}

func TestSDKNotice(t *testing.T) {
	requireEmbeddedSDK(t)
	root := newProject(t, "web-go")
	t.Chdir(root)
	p, _ := findProject()
	if msg := sdkNotice(p); msg != "" {
		t.Fatalf("新專案不該提醒：%s", msg)
	}
	for v, want := range map[string]string{"0.0.1": "pt sdk update", "99.0.0-20991231": "先更新 pt", "weird": "無法辨識"} {
		setProjectSDK(t, root, v)
		p, _ := findProject()
		if msg := sdkNotice(p); !strings.Contains(msg, want) {
			t.Fatalf("SDK %s：提醒要含 %q：%s", v, want, msg)
		}
		// pt check 列為警告，不擋上傳
		if fs := checkProject(p, p.path(p.Web)); !hasFinding(fs, "warn", want) || hasFinding(fs, "error", want) {
			t.Fatalf("SDK %s：pt check 要列為警告：%+v", v, fs)
		}
		// 還原成內嵌版本，下一輪再替換
		raw, _ := os.ReadFile(filepath.Join(root, projectFile))
		must(t, os.WriteFile(filepath.Join(root, projectFile), bytes.Replace(raw, []byte(`"sdkVersion": "`+v+`"`), []byte(`"sdkVersion": "`+sdkVersion()+`"`), 1), 0o644))
	}
}

func TestSDKUpdateRefusesDowngrade(t *testing.T) {
	requireEmbeddedSDK(t)
	root := newProject(t, "web-node")
	t.Chdir(root)
	setProjectSDK(t, root, "99.0.0-20991231")
	if err := cmdSDK([]string{"update"}); err == nil || !strings.Contains(err.Error(), "不會降版") {
		t.Fatalf("專案的 SDK 比 pt 新時不能降版：%v", err)
	}
	if p, _ := findProject(); p.SDKVersion != "99.0.0-20991231" {
		t.Fatalf("拒絕時不能改 playtogether.json：%s", p.SDKVersion)
	}
	if err := cmdSDK([]string{"update", "-force"}); err != nil {
		t.Fatal(err)
	}
	if p, _ := findProject(); p.SDKVersion != sdkVersion() {
		t.Fatalf("-force 要換成內嵌的版本：%s", p.SDKVersion)
	}
}

// fakeSDKRelease：最新版 pt 是 tag，內嵌的 SDK 是 sdk。
func fakeSDKRelease(t *testing.T, tag, sdk string) {
	t.Helper()
	fakeRelease(t, "https://github.com/game-dev-zone/play-together-ctl/releases/tag/"+tag)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/"+tag {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(sdk + "\n"))
	}))
	t.Cleanup(srv.Close)
	old := sdkVersionURL
	sdkVersionURL = func(tag string) string { return srv.URL + "/" + tag }
	t.Cleanup(func() { sdkVersionURL = old })
}

func TestSDKStatus(t *testing.T) {
	requireEmbeddedSDK(t)
	root := newProject(t, "web-go")
	t.Chdir(root)
	p, _ := findProject()

	fakeSDKRelease(t, "v0.6.0", sdkVersion())
	if out := sdkStatus(p, true); !strings.Contains(out, "✓ 專案的 SDK 已經是最新版") {
		t.Fatalf("都是最新版：%s", out)
	}

	fakeSDKRelease(t, "v9.0.0", "9.0.0-20991231")
	out := sdkStatus(p, true)
	if !strings.Contains(out, "最新版 pt（v9.0.0）內嵌：9.0.0-20991231") || !strings.Contains(out, "先更新 pt") || strings.Contains(out, "✓") {
		t.Fatalf("新版 pt 有更新的 SDK 時要提醒先更新 pt：%s", out)
	}

	// 版本一樣但 SDK 檔案被改過
	fakeSDKRelease(t, "v0.6.0", sdkVersion())
	must(t, os.WriteFile(filepath.Join(root, "web/playtogether-sdk.iife.js"), []byte("// 改過"), 0o644))
	if out := sdkStatus(p, true); !strings.Contains(out, "web/playtogether-sdk.iife.js") || !strings.Contains(out, "pt sdk update 會換回") {
		t.Fatalf("要列出被修改的 SDK 檔案：%s", out)
	}
}

func TestLatestSDKRejectsGarbage(t *testing.T) {
	for _, body := range []string{"<html>", "0.1.0; rm -rf /", strings.Repeat("9", 100)} {
		fakeSDKRelease(t, "v0.6.0", body)
		if _, _, err := latestSDK(); err == nil {
			t.Fatalf("SDK 版本 %q 要拒絕", body)
		}
	}
	// 找不到 VERSION 檔
	fakeSDKRelease(t, "v0.6.0", "0.1.0-20261101")
	sdkVersionURL = func(string) string { return "http://127.0.0.1:1/x" }
	if _, _, err := latestSDK(); err == nil {
		t.Fatal("連不上要回傳錯誤")
	}
}

func hasFinding(fs []finding, level, sub string) bool {
	for _, f := range fs {
		if (f.level == level || level == "error" && isErrorLevel(f.level)) && strings.Contains(f.msg, sub) {
			return true
		}
	}
	return false
}
