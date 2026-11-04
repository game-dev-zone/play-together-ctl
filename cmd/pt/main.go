// pt 是玩聚的開發者指令列工具：把打包好的網頁遊戲上傳到玩聚託管。只用 Go 標準函式庫。
//
//	pt keygen -purpose upload -out upload.key   （遊戲伺服器金鑰：-purpose server -out server.key）
//	    產生金鑰：私鑰寫入檔案（權限 600），印出公鑰交給玩聚登記
//
//	pt new my-game；pt dev；pt check（見 project.go）
//
//	pt upload -version 1.2.0 -notes "修正…" ./dist
//	    打包 ./dist 的「內容」成 ZIP、上傳、等待平台檢查結果
//
// 設定（參數或環境變數）：-api／PT_API_URL、-key-id／PT_UPLOAD_KEY_ID、-key-file／PT_UPLOAD_KEY_FILE、
// -connect／PT_CONNECT_ORIGINS（遊戲會連線的外部來源，逗號分隔，例如 wss://game-server.example）。
//
// 上傳金鑰只能上傳；送審、審核、發佈都要另外由團隊成員與平台處理。
package main

import (
	"archive/zip"
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"runtime/debug"
	"slices"
	"sort"
	"strings"
	"time"

	playtogether "github.com/game-dev-zone/play-together-ctl/internal/playtogether"
)

// version 由發佈流程用 -ldflags "-X main.version=v1.2.3" 設定；用 go install 安裝時改讀模組版本
var version = "dev"

func init() {
	if info, ok := debug.ReadBuildInfo(); ok && version == "dev" && info.Main.Version != "" && info.Main.Version != "(devel)" {
		version = info.Main.Version
	}
}

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	var err error
	finish := func() {}
	if os.Args[1] != "version" && os.Args[1] != "-version" && os.Args[1] != "--version" {
		finish = startUpdateCheck()
	}
	switch os.Args[1] {
	case "new":
		err = cmdNew(os.Args[2:])
	case "dev":
		err = cmdDev(os.Args[2:])
	case "check":
		err = cmdCheck(os.Args[2:])
	case "build":
		err = cmdBuild(os.Args[2:])
	case "sdk":
		err = cmdSDK(os.Args[2:])
	case "keygen":
		err = keygen(os.Args[2:])
	case "upload":
		err = upload(os.Args[2:])
	case "cdn":
		err = cmdCDN(os.Args[2:])
	case "version", "-version", "--version":
		fmt.Print(versionInfo())
	default:
		usage()
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "錯誤：", err)
		finish()
		os.Exit(1)
	}
	finish()
}

func usage() {
	fmt.Fprint(os.Stderr, `pt：玩聚開發者工具

  pt new [名稱]                                 建立新專案（畫面＋遊戲伺服器，已接好 SDK）；在終端機會逐步詢問
         [-client web|godot] [-server go|node]  直接指定，不再詢問
         [-orientation portrait|landscape|any]  畫面方向：直式（預設）、橫式、兩者都支援
         [-ai studios|none] [-y]                Godot 專案加入 Claude Code Game Studios；-y 不詢問、用預設值
         [-skills none|core|full]               加入遊戲開發 skill（awesome-gamedev-agent-skills 精選）
  pt dev [-addr 127.0.0.1:8090]                本機模擬模式：啟動遊戲伺服器、打開兩個玩家分頁
         [-smoke]                               啟動驗證：啟動、確認首頁與 SDK 載入得到後結束（不開瀏覽器）
  pt build                                      Godot 專案：用 Godot 無頭模式匯出網頁版到 build/
  pt check [-local] [建置資料夾]                上傳前檢查（對照審核準則）；-local：platform、serverUrl 還沒填不算錯誤
  pt keygen -purpose upload|server [-out 檔案]   產生金鑰；在專案裡預設寫到 keys/<用途>.key
  pt upload -version <版本> [-notes …]          上傳版本；在專案裡其他設定（含畫面方向、CDN 申報）從 playtogether.json 讀
  pt cdn add <網址>                             從 jsDelivr／cdnjs 載入固定版本的程式或樣式：算出 SRI、記到 playtogether.json、印出要貼的標籤
  pt sdk status                                 檢查專案的 SDK、這個 pt 內嵌的 SDK、最新版 pt 的 SDK
  pt sdk update [-force]                        把專案裡的 SDK 更新成這個 pt 內嵌的版本（不會降版，除非 -force）
  pt sdk export <資料夾>                        把 SDK 檔案寫到資料夾（接到既有專案用）
  pt version                                    顯示版本並檢查有沒有新版

其他指令最多一天檢查一次新版（只在終端機；PT_NO_UPDATE_CHECK=1 關閉）。

文件：玩聚開發者文件站 /docs/pt　原始碼與下載：https://github.com/game-dev-zone/play-together-ctl
`)
	os.Exit(2)
}

func keygen(args []string) error {
	fs := flag.NewFlagSet("keygen", flag.ExitOnError)
	out := fs.String("out", "", "私鑰檔案（必填；不要提交到版本控制）")
	purpose := fs.String("purpose", "", "金鑰用途：upload（上傳版本）或 server（遊戲伺服器）")
	_ = fs.Parse(args)
	if *out == "" && (*purpose == "upload" || *purpose == "server") {
		// 在 pt new 建立的專案裡：預設寫到 playtogether.json 指定的位置（keys/ 不進版控）
		if p, err := findProject(); err == nil && p != nil {
			*out = p.path(map[string]string{"upload": p.UploadKeyFile, "server": p.ServerKeyFile}[*purpose])
			_ = os.MkdirAll(filepath.Dir(*out), 0o700)
		}
	}
	if *out == "" {
		return errors.New("請用 -out 指定私鑰檔案")
	}
	labels := map[string]string{"upload": "上傳版本（pt upload）", "server": "遊戲伺服器（兌換啟動碼、發言規則、回報一局）"}
	label, ok := labels[*purpose]
	if !ok {
		return errors.New("請用 -purpose 指定用途：upload 或 server（兩種用途要用不同的金鑰）")
	}
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(*out, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return fmt.Errorf("無法建立私鑰檔案（檔案已存在？）：%w", err)
	}
	if _, err := f.WriteString(base64.StdEncoding.EncodeToString(priv.Seed()) + "\n"); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	fmt.Println("私鑰已寫入", *out, "（權限 600，請勿分享或提交到版本控制）")
	fmt.Printf("請把下面的公鑰交給玩聚登記（用途：%s）：\n", label)
	fmt.Println(base64.StdEncoding.EncodeToString(pub))
	return nil
}

func env(flagVal, name string) string {
	if flagVal != "" {
		return flagVal
	}
	return os.Getenv(name)
}

func upload(args []string) error {
	fs := flag.NewFlagSet("upload", flag.ExitOnError)
	api := fs.String("api", "", "平台 API 網址（或 PT_API_URL）")
	keyID := fs.String("key-id", "", "上傳金鑰 ID（或 PT_UPLOAD_KEY_ID）")
	keyFile := fs.String("key-file", "", "上傳私鑰檔案（或 PT_UPLOAD_KEY_FILE）")
	version := fs.String("version", "", "版本號，例如 1.2.0（必填，不能重複）")
	notes := fs.String("notes", "", "版本說明（給審核者看）")
	connect := fs.String("connect", "", "遊戲會連線的外部來源，逗號分隔（或 PT_CONNECT_ORIGINS）")
	wait := fs.Duration("wait", 5*time.Minute, "等待檢查結果的時間（0 表示不等）")
	skipCheck := fs.Bool("skip-check", false, "不執行上傳前檢查（不建議）")
	noBuild := fs.Bool("no-build", false, "Godot 專案：不重新匯出，直接上傳現有的 build/")
	orientation := fs.String("orientation", "", "畫面方向 portrait|landscape|any（在 pt new 建立的專案裡預設讀 playtogether.json）")
	_ = fs.Parse(reorder(args))
	p, err := findProject()
	if err != nil {
		return err
	}
	dir := fs.Arg(0)
	// 在 pt new 建立的專案裡：參數沒給的從 playtogether.json 讀（參數 > 環境變數 > playtogether.json）
	or := func(v, fallback string) string {
		if v != "" {
			return v
		}
		return fallback
	}
	if p != nil {
		dir = or(dir, p.path(p.Web))
		*api = or(*api, os.Getenv("PT_API_URL"))
		*api = or(*api, p.Platform)
		*keyID = or(*keyID, os.Getenv("PT_UPLOAD_KEY_ID"))
		*keyID = or(*keyID, p.UploadKeyID)
		*keyFile = or(*keyFile, os.Getenv("PT_UPLOAD_KEY_FILE"))
		*keyFile = or(*keyFile, p.path(p.UploadKeyFile))
		*connect = or(*connect, os.Getenv("PT_CONNECT_ORIGINS"))
		*connect = or(*connect, p.ServerURL)
		*orientation = or(*orientation, p.Orientation)
	}
	// 畫面方向是版本的一部分：平台依此提示玩家旋轉手機（沒有給時平台當成 portrait）
	if *orientation != "" && !slices.Contains(orientations, *orientation) {
		return fmt.Errorf("-orientation 必須是 %s", strings.Join(orientations, "、"))
	}
	if dir == "" || *version == "" || fs.NArg() > 1 {
		return errors.New("用法：pt upload -version <版本> [-notes …] [建置資料夾]（在 pt new 建立的專案裡可以省略資料夾）")
	}
	base := strings.TrimRight(env(*api, "PT_API_URL"), "/")
	kid := env(*keyID, "PT_UPLOAD_KEY_ID")
	kfile := env(*keyFile, "PT_UPLOAD_KEY_FILE")
	if base == "" || kid == "" || kfile == "" {
		return errors.New("需要平台網址、上傳金鑰 ID、上傳私鑰檔案：用 -api、-key-id、-key-file，或填在 playtogether.json（platform、uploadKeyId、uploadKeyFile）")
	}
	if p != nil && p.Client == "godot" && !*noBuild && fs.Arg(0) == "" {
		if err := buildGodot(p); err != nil {
			return err
		}
	}
	if !*skipCheck {
		fmt.Println("上傳前檢查：")
		if printFindings(checkProject(p, dir)) {
			return errors.New("檢查沒有通過，請修正後再上傳（確定要略過時加 -skip-check）")
		}
	}
	priv, err := readKey(kfile)
	if err != nil {
		return err
	}
	origins := []string{}
	for _, o := range strings.Split(env(*connect, "PT_CONNECT_ORIGINS"), ",") {
		if o = strings.TrimSpace(o); o != "" {
			origins = append(origins, o)
		}
	}

	data, count, err := zipDir(dir)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(data)
	fmt.Printf("已打包 %d 個檔案（%.1f MB）\n", count, float64(len(data))/(1<<20))

	c := client{base: base, keyID: kid, priv: priv}
	var created struct {
		BuildID    string `json:"buildId"`
		UploadPath string `json:"uploadPath"`
	}
	reg := map[string]any{
		"version": *version, "sha256": hex.EncodeToString(sum[:]), "size": len(data), "notes": *notes, "connectOrigins": origins,
	}
	if *orientation != "" {
		reg["orientation"] = *orientation
	}
	if p != nil && len(p.CDN) > 0 {
		reg["cdnAssets"] = p.CDN
	}
	if err := c.signed("POST", "/v1/dev/builds", reg, &created); err != nil {
		return err
	}
	req, _ := http.NewRequest(http.MethodPut, base+created.UploadPath, bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/zip")
	if err := doJSON(req, nil); err != nil {
		return fmt.Errorf("上傳失敗：%w", err)
	}
	fmt.Println("已上傳，版本 ID：", created.BuildID)
	if *wait == 0 {
		return nil
	}

	fmt.Print("平台檢查中")
	deadline := time.Now().Add(*wait)
	for time.Now().Before(deadline) {
		time.Sleep(2 * time.Second)
		var b struct {
			ArtifactStatus string `json:"artifactStatus"`
			EntryURL       string `json:"entryUrl"`
			CheckReport    struct {
				Problems  []string `json:"problems"`
				FileCount int      `json:"fileCount"`
			} `json:"checkReport"`
		}
		if err := c.signed("GET", "/v1/dev/builds/"+created.BuildID, nil, &b); err != nil {
			return err
		}
		switch b.ArtifactStatus {
		case "ready":
			fmt.Printf("\n檢查通過（%d 個檔案）。版本網址：%s\n", b.CheckReport.FileCount, b.EntryURL)
			fmt.Println("下一步：請玩聚為這個版本開預覽房間，用手機實際玩一局；沒問題後請玩聚送審（開發者主控台開放前由平台協助）。")
			fmt.Println("版本 ID 請一併提供給玩聚。")
			return nil
		case "failed":
			fmt.Println("\n檢查沒有通過：")
			for _, p := range b.CheckReport.Problems {
				fmt.Println("  ・", p)
			}
			return errors.New("請修正後用同一個版本號重新上傳")
		}
		fmt.Print(".")
	}
	fmt.Println()
	return errors.New("等待逾時；檢查仍在進行，稍後可再查詢")
}

func readKey(path string) (ed25519.PrivateKey, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("無法讀取私鑰檔案：%w", err)
	}
	seed, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(raw)))
	if err != nil || len(seed) != ed25519.SeedSize {
		return nil, errors.New("私鑰檔案格式錯誤（應為 pt keygen 產生的檔案）")
	}
	return ed25519.NewKeyFromSeed(seed), nil
}

// zipDir 打包資料夾的內容（不含資料夾本身）：檔案依路徑排序、固定時間戳記；
// 隱藏檔（.DS_Store 等）略過；遇到符號連結直接停止（平台不接受）。
func zipDir(dir string) ([]byte, int, error) {
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return nil, 0, fmt.Errorf("%s 不是資料夾", dir)
	}
	var paths []string
	err = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if p != dir && strings.HasPrefix(d.Name(), ".") {
			fmt.Println("略過隱藏檔：", p)
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Type()&fs.ModeSymlink != 0 {
			return fmt.Errorf("%s 是符號連結，平台不接受；請改成實際檔案", p)
		}
		if d.Type().IsRegular() {
			paths = append(paths, p)
		}
		return nil
	})
	if err != nil {
		return nil, 0, err
	}
	sort.Strings(paths)
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	stamp := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for _, p := range paths {
		rel, _ := filepath.Rel(dir, p)
		w, err := zw.CreateHeader(&zip.FileHeader{Name: filepath.ToSlash(rel), Method: zip.Deflate, Modified: stamp})
		if err != nil {
			return nil, 0, err
		}
		f, err := os.Open(p)
		if err != nil {
			return nil, 0, err
		}
		_, err = io.Copy(w, f)
		f.Close()
		if err != nil {
			return nil, 0, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, 0, err
	}
	return buf.Bytes(), len(paths), nil
}

type client struct {
	base  string
	keyID string
	priv  ed25519.PrivateKey
}

func (c client) signed(method, path string, body, out any) error {
	var raw []byte
	if body != nil {
		raw, _ = json.Marshal(body)
	}
	req, _ := http.NewRequest(method, c.base+path, bytes.NewReader(raw))
	for k, v := range playtogether.SignRequest(playtogether.SignableRequest{Method: method, Path: path, Body: raw},
		c.keyID, c.priv, time.Now().UnixMilli(), playtogether.NewNonce()) {
		req.Header[k] = v
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return doJSON(req, out)
}

var errorText = map[string]string{
	"unknown_key":        "金鑰無效（金鑰 ID 錯誤、已撤銷、不是上傳金鑰，或遊戲已停用）",
	"bad_signature":      "簽章錯誤（私鑰和登記的公鑰不符）",
	"clock_skew":         "電腦時間和伺服器差太多，請校正時間",
	"bad_version":        "版本號格式必須是 主.次.修訂，例如 1.2.0",
	"version_exists":     "這個版本號已經用過，請換一個",
	"bad_size":           "檔案太大（上限 200MB）",
	"bad_connect_origin": "外部連線必須是 https:// 或 wss:// 加網域，不含路徑",
	"rate_limited":       "上傳太頻繁，請稍後再試",
	"hosting_disabled":   "平台沒有開啟託管上架",
	"sha256_mismatch":    "上傳過程中檔案損毀，請重新上傳",
	"size_mismatch":      "上傳過程中檔案大小不符，請重新上傳",
	"bad_cdn_asset":      "playtogether.json 的 cdn 格式不對（只接受 jsDelivr、cdnjs 精確版本的 .js／.mjs／.css 完整網址加 SRI；用 pt cdn add 加入）",
	"cdn_check_failed":   "平台下載 CDN 檔案驗證失敗（重新導向、沒有允許跨來源，或內容和 SRI 不符）",
}

func doJSON(req *http.Request, out any) error {
	res, err := (&http.Client{Timeout: 10 * time.Minute}).Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if res.StatusCode >= 300 {
		var e struct {
			Error  string `json:"error"`
			URL    string `json:"url"`
			Detail string `json:"detail"`
		}
		_ = json.Unmarshal(body, &e)
		if t, ok := errorText[e.Error]; ok {
			if e.URL != "" {
				t += "：" + e.URL + "（" + e.Detail + "）"
			}
			return errors.New(t)
		}
		return fmt.Errorf("平台回應 %d %s", res.StatusCode, e.Error)
	}
	if out != nil {
		return json.Unmarshal(body, out)
	}
	return nil
}
