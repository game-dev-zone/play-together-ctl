package main

// SDK 版本檢查：專案的 SDK（playtogether.json 的 sdkVersion）和這個 pt 內嵌的 SDK 比較。
// SDK 只隨 pt 發佈，所以「最新的 SDK」就是最新版 pt 內嵌的 SDK：pt sdk status 會一併查。

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var (
	sdkVersionPattern = regexp.MustCompile(`^\d+\.\d+\.\d+(-\d{8})?$`)
	// 測試時換掉：某個 pt 版本內嵌的 SDK 版本
	sdkVersionURL = func(tag string) string {
		return "https://raw.githubusercontent.com/game-dev-zone/play-together-ctl/" + tag + "/cmd/pt/assets/sdk/VERSION"
	}
)

// compareSDK 比較 SDK 版本（主.次.修訂-日期）：a 比 b 舊回傳 -1，一樣 0，比較新 1；格式不對回傳 0 與 false。
func compareSDK(a, b string) (int, bool) {
	pa, okA := parseSDK(a)
	pb, okB := parseSDK(b)
	if !okA || !okB {
		return 0, false
	}
	for i := range pa {
		if pa[i] != pb[i] {
			if pa[i] < pb[i] {
				return -1, true
			}
			return 1, true
		}
	}
	return 0, true
}

func parseSDK(v string) ([4]int, bool) {
	var out [4]int
	if !sdkVersionPattern.MatchString(v) {
		return out, false
	}
	main, date, _ := strings.Cut(v, "-")
	for i, s := range strings.Split(main, ".") {
		out[i], _ = strconv.Atoi(s)
	}
	if date != "" {
		out[3], _ = strconv.Atoi(date)
	}
	return out, true
}

// sdkNotice：專案的 SDK 和這個 pt 內嵌的不一樣時，回傳一行提醒（一樣或無法比較時回傳空字串）。
func sdkNotice(p *project) string {
	if p == nil || p.SDKVersion == "" || sdkVersion() == "" {
		return ""
	}
	c, ok := compareSDK(p.SDKVersion, sdkVersion())
	switch {
	case !ok && p.SDKVersion != sdkVersion():
		return fmt.Sprintf("專案的 SDK 版本 %q 無法辨識，這個 pt 內嵌 %s：執行 pt sdk update 換成內嵌的版本", p.SDKVersion, sdkVersion())
	case c < 0:
		return fmt.Sprintf("專案的 SDK 是 %s，這個 pt 內嵌較新的 %s：執行 pt sdk update 更新（pt sdk status 看詳細）", p.SDKVersion, sdkVersion())
	case c > 0:
		return fmt.Sprintf("專案的 SDK %s 比這個 pt 內嵌的 %s 新：請先更新 pt（go install %s@latest）", p.SDKVersion, sdkVersion(), installPath)
	}
	return ""
}

// printSDKNotice 給 pt dev、pt build：有提醒就印一行。
func printSDKNotice(p *project) {
	if msg := sdkNotice(p); msg != "" {
		fmt.Println("！" + msg)
	}
}

// modifiedSDKFiles 列出專案裡和這個 pt 內嵌內容不同（或不見了）的 SDK 檔案；只有版本相同時才有意義。
func modifiedSDKFiles(p *project) ([]string, error) {
	files, err := sdkFiles(p.Client, p.Server, p.godotDir())
	if err != nil {
		return nil, err
	}
	var out []string
	for rel, want := range files {
		got, err := os.ReadFile(p.path(rel))
		if err != nil || !bytes.Equal(got, want) {
			out = append(out, rel)
		}
	}
	sort.Strings(out)
	return out, nil
}

// latestSDK 查最新版 pt 內嵌的 SDK 版本（先查最新 pt 版本，再讀那個版本的 VERSION 檔）。
func latestSDK() (ptTag, sdk string, err error) {
	ptTag, err = fetchLatest()
	if err != nil {
		return "", "", err
	}
	client := &http.Client{Timeout: updateTimeout}
	req, err := http.NewRequest(http.MethodGet, sdkVersionURL(ptTag), nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("User-Agent", "pt/"+version)
	res, err := client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(res.Body, 64))
	if err != nil {
		return "", "", err
	}
	sdk = strings.TrimSpace(string(raw))
	if res.StatusCode != http.StatusOK || !sdkVersionPattern.MatchString(sdk) {
		return "", "", fmt.Errorf("無法取得 %s 內嵌的 SDK 版本（%d）", ptTag, res.StatusCode)
	}
	return ptTag, sdk, nil
}

// sdkStatus 給 pt sdk status：專案、這個 pt、最新版 pt 的 SDK 版本，以及建議的下一步。
func sdkStatus(p *project, online bool) string {
	var b strings.Builder
	embedded := sdkVersion()
	if p != nil {
		recorded := p.SDKVersion
		if recorded == "" {
			recorded = "（沒有記錄）"
		}
		fmt.Fprintf(&b, "專案的 SDK：%s（playtogether.json）\n", recorded)
	}
	fmt.Fprintf(&b, "這個 pt（%s）內嵌：%s\n", version, embedded)
	latestTag, latest := "", ""
	if online {
		var err error
		if latestTag, latest, err = latestSDK(); err != nil {
			fmt.Fprintf(&b, "最新版 pt：無法檢查（%v）\n", err)
		} else {
			fmt.Fprintf(&b, "最新版 pt（%s）內嵌：%s\n", latestTag, latest)
		}
	}
	b.WriteString("\n")
	ptOld := false
	if latest != "" {
		if c, ok := compareSDK(embedded, latest); ok && c < 0 {
			ptOld = true
		}
	}
	if p != nil && p.existing() {
		b.WriteString(existingSDKNote + "\n")
	} else if p != nil {
		if msg := sdkNotice(p); msg != "" {
			b.WriteString("！" + msg + "\n")
		} else if mod, err := modifiedSDKFiles(p); err == nil && len(mod) > 0 {
			fmt.Fprintf(&b, "！這些 SDK 檔案和 %s 的內容不同（被修改或刪除）：\n", embedded)
			for _, f := range mod {
				b.WriteString("    " + f + "\n")
			}
			b.WriteString("  SDK 檔案不要直接修改；pt sdk update 會換回原本的內容。\n")
		} else if !ptOld {
			b.WriteString("✓ 專案的 SDK 已經是最新版\n")
		}
	}
	if ptOld {
		fmt.Fprintf(&b, "！有更新的 SDK %s：先更新 pt（go install %s@latest，或到 %s/releases/latest 下載），", latest, installPath, repoURL)
		if p != nil {
			b.WriteString("再執行 pt sdk update\n")
		} else {
			b.WriteString("再用 pt sdk export 匯出\n")
		}
	}
	return b.String()
}
