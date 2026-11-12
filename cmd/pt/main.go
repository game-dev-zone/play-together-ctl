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
	case "verify":
		err = cmdVerify(os.Args[2:])
	case "build":
		err = cmdBuild(os.Args[2:])
	case "sdk":
		err = cmdSDK(os.Args[2:])
	case "upgrade":
		err = cmdUpgrade(os.Args[2:])
	case "migrate":
		err = cmdMigrate(os.Args[2:])
	case "add":
		err = cmdAdd(os.Args[2:])
	case "keygen":
		err = keygen(os.Args[2:])
	case "upload":
		err = upload(os.Args[2:])
	case "cdn":
		err = cmdCDN(os.Args[2:])
	case "content":
		err = cmdContent(os.Args[2:])
	case "status":
		err = cmdStatus(os.Args[2:])
	case "submit":
		err = cmdSubmit(os.Args[2:])
	case "preview":
		err = cmdPreview(os.Args[2:])
	case "withdraw":
		err = cmdWithdraw(os.Args[2:])
	case "store":
		err = cmdStore(os.Args[2:])
	case "login":
		err = cmdLogin(os.Args[2:])
	case "register":
		err = cmdRegister(os.Args[2:])
	case "init":
		err = cmdInit(os.Args[2:])
	case "logout":
		err = cmdLogout(os.Args[2:])
	case "skills":
		err = cmdSkills(os.Args[2:])
	case "install":
		// 常見的打法：pt install skills
		if len(os.Args) == 3 && os.Args[2] == "skills" {
			err = cmdSkills([]string{"install"})
		} else {
			usage()
		}
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
	fmt.Fprint(os.Stderr, `pt：玩聚開發者工具（玩聚 Play Together：給直播主與觀眾的瀏覽器多人派對遊戲平台）

給 AI 助手（Claude Code、Codex）的總覽，先讀這段：
  1. 先執行 pt skills install：安裝玩聚的 skill（開新遊戲、上架到送審）。之後照 skill 的步驟做，不要自己猜流程
  2. 三種開始方式：
       空資料夾、還沒有遊戲   → pt new .（在目前資料夾建立；skill：playtogether-new）
       已經有自己的網頁遊戲   → pt init（不動原本的程式；skill：playtogether-publish）
       單機遊戲想改成多人     → pt init 之後 pt add server（專案裡的 skill：playtogether-multiplayer）
  3. 多人遊戲需要遊戲伺服器。建議交給「玩聚代管」（-hosting playtogether，Go）：開發者不用自己找主機、不用付主機費，
     上傳時伺服器和畫面一起交給玩聚；本機用 pt dev 測試（本機假平台，不用金鑰、不用連網）
  4. 畫面建議用 Godot 4（-client godot）：先確認電腦有 Godot 4.3 以上（godot --version），沒有時請開發者安裝；
     開發者不想安裝、或只做介面為主要很快載入的小遊戲時才用網頁（-client web）
  5. 平台規則（伺服器決定遊戲狀態、成人平台、內容邊界、不涉及現實經濟價值）寫在專案的 CLAUDE.md、PLAYTOGETHER.md
  6. 只有開發者本人能做：pt login 時在瀏覽器確認、提高內容邊界或改玩法的決定、手機實測；私鑰（keys/）不要貼進對話

  pt new [名稱|.]                               建立新專案（畫面＋遊戲伺服器，已接好 SDK）；在終端機會逐步詢問
         [-name 名稱]                           pt new . 在目前的空資料夾建立，名稱預設用資料夾名稱
         [-client godot|web] [-server go|node]  直接指定，不再詢問（畫面建議 godot）
         [-orientation portrait|landscape|any]  畫面方向：直式（預設）、橫式、兩者都支援
         [-game-server playtogether-managed|external|none]
                                                遊戲伺服器架構：玩聚範本伺服器（預設）、自己的遊戲伺服器、不需要
         [-hosting playtogether]                遊戲伺服器交給玩聚代管（只限 -server go，要平台開通；在終端機會詢問）
         [-profile general|moderate|strong|custom]
                                                內容邊界（開發期間的內容上限，不是正式分級）：預設 general（-rating 是舊寫法）
         [-table-game none|card|poker|mahjong|dice|board|slot|roulette|baccarat|other]
                                                牌桌玩法；是否能上架由玩聚平台審核決定
         [-mode standard|studio|lightweight|custom]
                                                AI 開發模式；-y、非互動與 CI 沒指定時用 lightweight（不下載）
         [-skills none|core|full] [-ai studios] 自訂（-mode custom）：遊戲開發 skill、Claude Code Game Studios
         [-y]                                   不詢問，沒指定的項目用預設值
  pt dev [-addr 127.0.0.1:8090]                本機開發：啟動遊戲伺服器與本機假平台（驗證簽章；控制面板可以踢人、重新開始、
                                                關房、模擬平台連不到與時鐘偏差、看呼叫紀錄），並開啟玩家分頁
         [-players N]                           玩家分頁數（預設：多人專案 2、單人專案 1）
         [-no-server] [-site 網址]              只啟動假平台與控制面板（接自己的遊戲伺服器）
         [-server-url http://127.0.0.1:埠]      搭配 -no-server：握手時當成 launch.serverUrl 給畫面（像玩聚代管）
                                                玩聚代管的專案（gameServer.hosting）不用參數：pt dev 自己編譯並啟動 gameServer.dir 的伺服器，
                                                畫面和伺服器在不同埠，launch.serverUrl 指向伺服器
         [-mock]                                舊的模擬模式（不經過假平台、不簽章）
         [-smoke]                               啟動驗證：啟動、確認首頁與 SDK 載入得到後結束（不開瀏覽器）
  pt verify server [-json]                      用本機假平台自動驗證範本伺服器和玩聚平台的接口：加入、重新加入、
                                                踢人、重新開始、關房、平台連不到、時鐘偏差、回報一局、跨來源、413、429、簽章
  pt build                                      Godot 專案：用 Godot 無頭模式匯出網頁版到 build/
  pt check [-local] [-json] [建置資料夾]        上傳前檢查：ERROR_HARD_RULE／ERROR_SCHEMA／ERROR_BUILD／REVIEW_REQUIRED／WARNING；
                                                -local：platform、serverUrl（需要時）還沒填不算錯誤；-json：JSON 輸出
  pt content sync                               依 playtogether.json 重新產生 docs/content/ 的開發契約文件並記錄平台政策版本
  pt keygen -purpose upload|server [-out 檔案]   產生金鑰；在專案裡預設寫到 keys/<用途>.key
  pt upload -version <版本> [-notes …]          上傳版本與上架資料（store/）；在專案裡其他設定（含畫面方向、CDN 申報、宣告）從 playtogether.json 讀
         [-no-listing] [-listing-only]          只上傳版本／只上傳上架資料
  pt status                                     版本與上架資料的審核狀態、退件原因、正式分級
  pt submit [-version 版本] [-replace]           送審已經上傳的草稿版本（與草稿上架資料），不重新上傳；-replace 撤回目前審核中的版本改送這個
  pt withdraw                                   撤回審核中的版本（一起送審的上架資料回到草稿），改好再送
  pt preview [-version 版本]                    開預覽房間（1 小時，團隊成員才能進），印出網址；用手機在玩聚外殼裡實測
  pt store screenshot                           用本機 Chrome／Chromium 無頭模式擷取實際遊戲畫面，存成上架截圖（store/screenshots/）：
                                                依畫面方向模擬手機（直式 390×694、橫式 694×390 CSS px），輸出剛好 1080×1920／1920×1080；
                                                檔名接在既有截圖後面（不覆寫）；全黑或全白的不存檔。先啟動遊戲（例如 pt dev -mock）
         [-url 網址] [-wait 5s]                 遊戲網址（預設 http://<devAddr>/；用 pt dev 時傳它印出的玩家分頁網址）、載入後等多久
         [-n 張數] [-interval 3s]               擷取多張：在同一個頁面（同一局），照真實時間每隔 interval 再擷取一張
         [-locale en] [-o 資料夾] [-replace]    英文介面專用（store/screenshots/en/）、輸出資料夾、從 01 開始覆寫
         [-orientation portrait|landscape] [-chrome 路徑]
                                                畫面方向（預設依 playtogether.json；any 時直式）、Chrome 執行檔（或 PT_CHROME）
  pt login [-api 平台網址] [-register]          在瀏覽器確認後取得送審授權（30 天）；之後 pt upload 會在檢查通過後直接送審（-no-submit 不送審）
                                                -register：另外取得建立遊戲與管理金鑰的權限（團隊擁有者、管理者；7 天）
  pt init [-name 代號] [-web 資料夾] [-game-server external|none] [-hosting playtogether] [-y]
                                                把不是 pt new 建立的既有專案接上玩聚（只新增 playtogether.json、宣告文件、store/、
                                                準備上架資料的 AI skill，不動原本的程式）；已經接上的專案重跑時只補上缺少的 skill
  pt register [-title 名稱]                     建立草稿遊戲並登記金鑰（沒有私鑰會自動產生），金鑰 ID 寫回 playtogether.json
  pt logout                                     刪除這台電腦上的送審授權
  pt skills install                             把玩聚的 skill（playtogether-new 開新遊戲、playtogether-publish 上架到送審）裝到
                                                ~/.claude/skills 與 ~/.agents/skills：之後在任何資料夾請 Claude Code／Codex
                                                「幫我做一款玩聚遊戲」或「幫我上架到玩聚」（pt install skills 也可以；uninstall 移除）
         [-skip-check]                          略過可略過的本機／建置檢查；平台硬性規則與專案宣告一致性仍會檢查
  pt cdn add <網址>                             從 jsDelivr／cdnjs 載入固定版本的程式或樣式：算出 SRI、記到 playtogether.json、印出要貼的標籤
  pt sdk status                                 檢查專案的 SDK、這個 pt 內嵌的 SDK、最新版 pt 的 SDK
  pt sdk update [-force] [-overwrite-modified]  把專案裡的 SDK 更新成這個 pt 內嵌的版本（不會降版，除非 -force；SDK 檔案被改過時停下來）
  pt upgrade [-dry-run] [-y] [-delete-removed] 把範本檔案（平台規則、AI 契約、Makefile、pt 的 skill）更新成這個 pt 的版本：
                                                沒改過的換新、改過的寫 .pt-new；遊戲本身的檔案不碰。舊專案先 pt upgrade -init
  pt migrate server-kit [-apply]                舊專案（server/session.go）改用 SDK 的遊戲伺服器套件：先產生對照與步驟，搬好後 -apply
  pt add server -server go|node [-hosting playtogether]
                                                pt init 接入的既有專案（例如單機遊戲）加上玩聚的範本遊戲伺服器，準備改成多人聯機
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
	skipCheck := fs.Bool("skip-check", false, "略過可略過的本機／建置檢查；平台硬性規則與專案宣告一致性仍會檢查")
	noBuild := fs.Bool("no-build", false, "Godot 專案：不重新匯出，直接上傳現有的 build/")
	orientation := fs.String("orientation", "", "畫面方向 portrait|landscape|any（在 pt new 建立的專案裡預設讀 playtogether.json）")
	noListing := fs.Bool("no-listing", false, "不上傳上架資料（store/），只上傳版本")
	listingOnly := fs.Bool("listing-only", false, "只上傳上架資料（換截圖、修說明），不打包遊戲")
	noSubmit := fs.Bool("no-submit", false, "只上傳，不送審（已經 pt login 時預設會在檢查通過後送審）")
	replace := fs.Bool("replace", false, "這款遊戲有其他版本在審核中時，撤回它、改送這個版本（沒有指定時在終端機詢問）")
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
	if *noListing && *listingOnly {
		return errors.New("-no-listing 和 -listing-only 只能指定一個")
	}
	if *listingOnly {
		return uploadListingOnly(p, *api, *keyID, *keyFile, *noSubmit)
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
	if err := validPlatform(base); err != nil {
		return err
	}
	if p != nil && p.Client == "godot" && !*noBuild && fs.Arg(0) == "" {
		if err := buildGodot(p); err != nil {
			return err
		}
	}
	fmt.Println("上傳前檢查：")
	res := printFindings(checkProject(p, dir))
	switch {
	case res.blocking():
		// 硬性規則與宣告錯誤：-skip-check 也不能略過，訊息不提示這個旗標
		return errors.New("有平台硬性規則或宣告錯誤，不能上傳：請修正後再上傳")
	case res.build && !*skipCheck:
		return errors.New("檢查沒有通過，請修正後再上傳（只剩建置／本機設定的錯誤；確定要略過時加 -skip-check）")
	case res.build:
		fmt.Println("已依 -skip-check 略過建置／本機設定的錯誤（平台硬性規則與宣告一致性已檢查）")
	}
	if res.review {
		fmt.Println("這個版本有需要玩聚平台人工審核的項目（見上方）：可以送審，但不代表已批准；上架、正式分級、警語與標籤由玩聚平台審核決定。")
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

	// 玩聚代管的遊戲伺服器：建置、打包，和畫面一起登記在同一個版本
	var serverZip []byte
	if p.hostedServer() {
		if serverZip, err = buildServerBundle(p); err != nil {
			return err
		}
	}

	c := client{base: base, keyID: kid, priv: priv}
	var created struct {
		BuildID          string `json:"buildId"`
		UploadPath       string `json:"uploadPath"`
		ServerUploadPath string `json:"serverUploadPath"`
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
	// 平台外殼樣式：沒設定時不送，平台當成預設的 dark
	if shell, ok := shellBody(p); ok {
		reg["shell"] = shell
	}
	// 遊戲伺服器架構：平台依此決定審核要不要測遊戲伺服器的部署（none 不需要）
	if p != nil {
		reg["gameServerMode"] = p.gameServerMode()
	}
	// 宣告快照：平台自己判斷硬性規則與需要審核的項目（不採信本機的 pt check）
	if decl, ok := declarationBody(p); ok {
		reg["declaration"], reg["policyVersion"] = decl, p.PolicyVersion
	}
	if serverZip != nil {
		ss := sha256.Sum256(serverZip)
		reg["server"] = map[string]any{"sha256": hex.EncodeToString(ss[:]), "size": len(serverZip)}
	}
	if err := c.signed("POST", "/v1/dev/builds", reg, &created); err != nil {
		return err
	}
	req, _ := http.NewRequest(http.MethodPut, base+created.UploadPath, bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/zip")
	if err := doJSON(req, nil); err != nil {
		return fmt.Errorf("上傳失敗：%w", err)
	}
	if serverZip != nil {
		if created.ServerUploadPath == "" {
			return errors.New("平台不支援代管遊戲伺服器（沒有回傳伺服器套件的上傳網址）")
		}
		req, _ := http.NewRequest(http.MethodPut, base+created.ServerUploadPath, bytes.NewReader(serverZip))
		req.Header.Set("Content-Type", "application/zip")
		if err := doJSON(req, nil); err != nil {
			return fmt.Errorf("上傳遊戲伺服器套件失敗：%w", err)
		}
	}
	fmt.Println("已上傳，版本 ID：", created.BuildID)
	revID := ""
	if !*noListing {
		var err error
		if revID, err = uploadStore(c, p); err != nil {
			return err
		}
	}
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
			ServerArtifactStatus string `json:"serverArtifactStatus"`
			ServerCheckReport    struct {
				Problems []string `json:"problems"`
			} `json:"serverCheckReport"`
		}
		if err := c.signed("GET", "/v1/dev/builds/"+created.BuildID, nil, &b); err != nil {
			return err
		}
		if serverZip != nil {
			switch b.ServerArtifactStatus {
			case "failed":
				fmt.Println("\n遊戲伺服器套件檢查沒有通過：")
				for _, p := range b.ServerCheckReport.Problems {
					fmt.Println("  ・", p)
				}
				return errors.New("請修正後用同一個版本號重新上傳")
			case "ready":
			default:
				fmt.Print(".")
				continue // 畫面與伺服器都通過才送審
			}
		}
		switch b.ArtifactStatus {
		case "ready":
			fmt.Printf("\n檢查通過（%d 個檔案）。版本網址：%s\n", b.CheckReport.FileCount, b.EntryURL)
			if !*noSubmit {
				logged, err := submitWithCredential(base, created.BuildID, revID, *replace, inReviewBuild(c))
				if err != nil {
					return err
				}
				if logged {
					fmt.Println("審核進度：pt status（或開發者頁）")
					return nil
				}
			}
			fmt.Println("下一步：執行 pt login 授權後，pt upload 會在檢查通過後直接送審；也可以到開發者頁送審。")
			if revID != "" {
				fmt.Printf("版本 ID %s、上架資料修訂 ID %s（兩者一起送審，發佈時一起生效）。\n", created.BuildID, revID)
			} else {
				fmt.Println("版本 ID：", created.BuildID)
			}
			fmt.Println("審核進度：pt status")
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
	"upload_quota":       "你的團隊 24 小時內上傳的上架圖片已達上限（200 張；沒有變更的圖片不算），請明天再試",
	"hosting_disabled":   "平台沒有開啟託管上架",
	"sha256_mismatch":    "上傳過程中檔案損毀，請重新上傳",
	"size_mismatch":      "上傳過程中檔案大小不符，請重新上傳",
	"bad_cdn_asset":      "playtogether.json 的 cdn 格式不對（只接受 jsDelivr、cdnjs 精確版本的 .js／.mjs／.css 完整網址加 SRI；用 pt cdn add 加入）",
	"cdn_check_failed":   "平台下載 CDN 檔案驗證失敗（重新導向、沒有允許跨來源，或內容和 SRI 不符）",
	"policy_hard_rule":   "playtogether.json 的宣告違反玩聚平台硬性規則，平台不收這個版本（執行 pt check 看原因）",
	"bad_declaration":    "playtogether.json 的宣告格式不對（執行 pt check 看原因）",
	"bad_name":           "store/listing.json 的 name：繁中與英文都必填，最多 30 字",
	"bad_summary":        "store/listing.json 的 summary：繁中最多 60 字、英文最多 80 字，都必填",
	"bad_description":    "store/listing.json 的 description：繁中與英文都必填，最多 4000 字",
	"bad_tags":           "store/listing.json 的 tags 有不認得的代碼",
	"bad_players":        "store/listing.json 的 players 不合法",
	"missing_images":     "store/ 要有圖示與至少一張截圖",
	"bad_images":         "store/ 的圖片數量或位置不對（截圖最多 10 張）",
	"too_many_drafts":    "還沒送審的上架資料草稿太多：請先送審，或請玩聚清除",
	"bad_image":          "圖片只接受 PNG 或 JPEG",
	"bad_image_size":     "圖片太大或太小",
	"bad_image_aspect":   "圖片比例和需要的差太多，平台不能自動裁切：請換一張",
	"busy":               "平台忙碌中，請重新執行 pt upload",
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
