package main

// 從 CDN 載入程式與樣式（玩聚 2026-11-04 開放）：
//   - 只接受 jsDelivr 的 npm 套件與 cdnjs，精確版本（x.y.z）的完整檔案網址（.js、.mjs、.css）；
//   - 每個檔案記錄在 playtogether.json 的 cdn（網址＋SRI），pt upload 會申報，平台上傳時自己下載驗證；
//   - 頁面上一律用 <script src integrity crossorigin> 或 <link rel=stylesheet integrity crossorigin>（import map 也要寫 integrity）。
// 字型、圖片、音效、WASM、Worker 不能從 CDN 載入，一律打包進上傳的檔案。
// 規則和玩聚平台上傳時的檢查相同；平台改規則時這裡要一起改。

import (
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type cdnAsset struct {
	URL       string `json:"url"`
	Integrity string `json:"integrity"`
}

const maxCDNAssets = 10

var (
	cdnSemver = `\d+\.\d+\.\d+(?:-[0-9A-Za-z.]+)?`
	cdnFile   = `[A-Za-z0-9_][A-Za-z0-9._/-]*\.(?:m?js|css)`
	cdnPaths  = map[string]*regexp.Regexp{
		"cdn.jsdelivr.net":     regexp.MustCompile(`^/npm/(?:@[a-z0-9][a-z0-9._-]*/)?[a-z0-9][a-z0-9._-]*@` + cdnSemver + `/` + cdnFile + `$`),
		"cdnjs.cloudflare.com": regexp.MustCompile(`^/ajax/libs/[A-Za-z0-9][A-Za-z0-9._-]*/` + cdnSemver + `/` + cdnFile + `$`),
	}
	sriPattern = regexp.MustCompile(`^(sha384|sha512)-([A-Za-z0-9+/]+={0,2})$`)
	// 頁面裡任何指向常見 CDN 的網址（不管寫在哪裡）
	cdnMention = regexp.MustCompile(`(?:https?:)?//(?:cdn\.jsdelivr\.net|cdnjs\.cloudflare\.com|unpkg\.com|esm\.sh|cdn\.skypack\.dev|fonts\.googleapis\.com|fonts\.gstatic\.com)[^\s"'<>)]*`)
	htmlTag    = regexp.MustCompile(`(?is)<(script|link)\b([^>]*)>`)
	htmlAttr   = regexp.MustCompile(`(?is)([a-z-]+)\s*=\s*(?:"([^"]*)"|'([^']*)'|([^\s>]+))`)
	importMap  = regexp.MustCompile(`(?is)<script\b[^>]*type\s*=\s*["']?importmap["']?[^>]*>(.*?)</script>`)
)

func cdnURLError(raw string) error {
	if raw == "" || len(raw) > 300 || strings.ContainsAny(raw, " \t\r\n;,'\"\\%") {
		return errors.New("網址含不允許的字元")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return err
	}
	re, ok := cdnPaths[u.Host]
	switch {
	case u.Scheme != "https":
		return errors.New("只接受 https")
	case !ok:
		return errors.New("只接受 cdn.jsdelivr.net（/npm/）與 cdnjs.cloudflare.com（/ajax/libs/）")
	case u.User != nil, u.RawQuery != "", u.ForceQuery, u.Fragment != "":
		return errors.New("不能有 query 或 #")
	case strings.Contains(u.Path, "..") || strings.Contains(u.Path, "//"):
		return errors.New("路徑不合格")
	case !re.MatchString(u.Path):
		return errors.New("要是精確版本（例如 @0.186.1）的完整檔案網址（.js、.mjs、.css），不能用 latest、版本範圍或目錄")
	}
	return nil
}

func sriError(s string) error {
	m := sriPattern.FindStringSubmatch(s)
	if m == nil {
		return errors.New("SRI 要是 sha384-… 或 sha512-…")
	}
	b, err := base64.StdEncoding.DecodeString(m[2])
	if err != nil || (m[1] == "sha384" && len(b) != 48) || (m[1] == "sha512" && len(b) != 64) {
		return errors.New("SRI 長度或格式不對")
	}
	return nil
}

// tagAttrs 把 <script …> 的屬性解析成小寫鍵的 map
func tagAttrs(s string) map[string]string {
	out := map[string]string{}
	for _, m := range htmlAttr.FindAllStringSubmatch(s, -1) {
		out[strings.ToLower(m[1])] = m[2] + m[3] + m[4]
	}
	return out
}

func isExternal(u string) bool {
	l := strings.ToLower(u)
	return strings.HasPrefix(l, "http:") || strings.HasPrefix(l, "https:") || strings.HasPrefix(l, "//")
}

// checkCDN：playtogether.json 的 cdn 申報格式，以及上傳資料夾裡每個外部載入都要申報、寫 SRI。
func checkCDN(p *project, webDir string, add func(level, format string, a ...any)) {
	declared := map[string]string{}
	var list []cdnAsset
	if p != nil {
		list = p.CDN
	}
	if len(list) > maxCDNAssets {
		add("error", "playtogether.json 的 cdn 最多 %d 個檔案", maxCDNAssets)
	}
	for _, c := range list {
		if err := cdnURLError(c.URL); err != nil {
			add("error", "playtogether.json 的 cdn：%s：%v", c.URL, err)
			continue
		}
		if err := sriError(c.Integrity); err != nil {
			add("error", "playtogether.json 的 cdn：%s：%v（用 pt cdn add 產生）", c.URL, err)
			continue
		}
		if _, dup := declared[c.URL]; dup {
			add("error", "playtogether.json 的 cdn 重複：%s", c.URL)
		}
		declared[c.URL] = c.Integrity
	}
	used := map[string]bool{}
	_ = filepath.WalkDir(webDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		ext := strings.ToLower(filepath.Ext(d.Name()))
		if ext != ".html" && ext != ".js" && ext != ".mjs" && ext != ".css" {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(webDir, path)
		s := string(raw)
		inTags := map[string]bool{}
		if ext == ".html" {
			for _, m := range htmlTag.FindAllStringSubmatch(s, -1) {
				a := tagAttrs(m[2])
				src := a["src"]
				if strings.EqualFold(m[1], "link") {
					src = a["href"]
				}
				if !isExternal(src) {
					continue
				}
				inTags[src] = true
				want, ok := declared[src]
				switch {
				case strings.EqualFold(m[1], "link") && !strings.EqualFold(a["rel"], "stylesheet"):
					add("error", "%s 從外部載入 %s（rel=%s）：CDN 只能用在程式與樣式表，字型、圖片、預先載入都要打包進上傳的檔案", rel, src, a["rel"])
				case !ok:
					add("error", "%s 從外部載入 %s，但 playtogether.json 的 cdn 沒有申報（用 pt cdn add %s）", rel, src, src)
				case a["integrity"] != want:
					add("error", "%s 載入 %s 要加 integrity=\"%s\"（和申報的 SRI 相同）", rel, src, want)
				case a["crossorigin"] != "anonymous":
					add("error", "%s 載入 %s 要加 crossorigin=\"anonymous\"（沒有它 SRI 檢查會失敗）", rel, src)
				default:
					used[src] = true
				}
			}
			// import map：網址要申報，而且 integrity 要寫在 import map 裡（JS 用套件名稱 import，不寫網址）
			for _, m := range importMap.FindAllStringSubmatch(s, -1) {
				var im struct {
					Imports   map[string]string `json:"imports"`
					Integrity map[string]string `json:"integrity"`
				}
				if err := json.Unmarshal([]byte(m[1]), &im); err != nil {
					add("error", "%s 的 import map 不是合法的 JSON：%v", rel, err)
					continue
				}
				for _, u := range im.Imports {
					if !isExternal(u) {
						continue
					}
					inTags[u] = true
					switch want, ok := declared[u]; {
					case !ok:
						add("error", "%s 的 import map 從外部載入 %s，但 playtogether.json 的 cdn 沒有申報", rel, u)
					case im.Integrity[u] != want:
						add("error", "%s 的 import map 要在 \"integrity\" 寫上 %s 的 SRI：%s", rel, u, want)
					default:
						used[u] = true
					}
				}
			}
		}
		// 其他地方出現的 CDN 網址（JS 的 import、fetch、CSS 的 @import／url()）沒有 SRI 保護，平台也會擋
		for _, u := range cdnMention.FindAllString(s, -1) {
			if inTags[u] {
				continue
			}
			add("error", "%s 直接使用 CDN 網址 %s：只能在 HTML 的 <script>／<link> 或 import map 載入並寫 SRI（JS 裡用套件名稱 import）；字型、圖片、WASM 要打包進上傳的檔案", rel, u)
		}
		return nil
	})
	for u := range declared {
		if !used[u] {
			add("warn", "playtogether.json 的 cdn 申報了 %s，但頁面沒有用到（可以刪掉）", u)
		}
	}
	if len(declared) > 0 {
		add("info", "使用 %d 個 CDN 檔案：CDN 連不上時遊戲要能顯示錯誤畫面；核心程式建議打包進上傳的檔案", len(declared))
	}
}

// cmdCDN：pt cdn add <網址>：下載檔案、確認沒有重新導向與允許跨來源、算出 SRI，記到 playtogether.json，並印出要貼進 HTML 的標籤。
func cmdCDN(args []string) error {
	if len(args) < 1 || args[0] != "add" {
		return errors.New("用法：pt cdn add <網址>（例如 https://cdn.jsdelivr.net/npm/phaser@4.2.1/dist/phaser.min.js）")
	}
	fs := flag.NewFlagSet("cdn add", flag.ExitOnError)
	_ = fs.Parse(reorder(args[1:]))
	if fs.NArg() != 1 {
		return errors.New("用法：pt cdn add <網址>")
	}
	raw := fs.Arg(0)
	if err := cdnURLError(raw); err != nil {
		return fmt.Errorf("%s：%w", raw, err)
	}
	p, err := findProject()
	if err != nil {
		return err
	}
	if p == nil {
		return errors.New("找不到 playtogether.json：請在 pt new 建立的專案裡執行")
	}
	for _, c := range p.CDN {
		if c.URL == raw {
			return fmt.Errorf("已經申報過 %s", raw)
		}
	}
	if len(p.CDN) >= maxCDNAssets {
		return fmt.Errorf("最多 %d 個 CDN 檔案", maxCDNAssets)
	}
	sri, err := fetchSRI(raw)
	if err != nil {
		return err
	}
	p.CDN = append(p.CDN, cdnAsset{URL: raw, Integrity: sri})
	if err := p.save(); err != nil {
		return err
	}
	fmt.Printf("已加入 playtogether.json 的 cdn：%s\n\n貼進 HTML：\n", raw)
	if strings.HasSuffix(raw, ".css") {
		fmt.Printf("  <link rel=\"stylesheet\" href=\"%s\" integrity=\"%s\" crossorigin=\"anonymous\" referrerpolicy=\"no-referrer\">\n", raw, sri)
	} else if strings.HasSuffix(raw, ".mjs") || strings.Contains(raw, ".module.") {
		fmt.Printf("  <script type=\"importmap\">{\"imports\": {\"套件名稱\": \"%s\"}, \"integrity\": {\"%s\": \"%s\"}}</script>\n", raw, raw, sri)
		fmt.Println("  （JS 裡用 import … from \"套件名稱\"，不要直接寫網址）")
	} else {
		fmt.Printf("  <script src=\"%s\" integrity=\"%s\" crossorigin=\"anonymous\" referrerpolicy=\"no-referrer\"></script>\n", raw, sri)
	}
	return nil
}

// fetchSRI 下載檔案並計算 sha384 SRI；不跟隨重新導向、要允許跨來源讀取（和平台上傳時的檢查相同）
var fetchSRI = func(raw string) (string, error) {
	c := &http.Client{Timeout: time.Minute, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	req, _ := http.NewRequest(http.MethodGet, raw, nil)
	req.Header.Set("Origin", "https://pt-cdn-check.invalid")
	res, err := c.Do(req)
	if err != nil {
		return "", fmt.Errorf("下載 %s 失敗：%w", raw, err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%s 回應 %d（不接受重新導向：請用完整檔案網址）", raw, res.StatusCode)
	}
	if res.Header.Get("Access-Control-Allow-Origin") != "*" {
		return "", fmt.Errorf("%s 沒有允許跨來源讀取，瀏覽器的 SRI 檢查會失敗", raw)
	}
	h := sha512.New384()
	n, err := io.Copy(h, io.LimitReader(res.Body, 10<<20+1))
	if err != nil {
		return "", err
	}
	if n > 10<<20 {
		return "", errors.New("檔案超過 10 MB")
	}
	return "sha384-" + base64.StdEncoding.EncodeToString(h.Sum(nil)), nil
}
