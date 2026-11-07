package main

// 上架資料（store/）：名稱、簡介、說明（繁中＋英文必填）、標籤、人數、給審核者的說明，以及圖示、截圖、封面。
// pt check 在本機先檢查（格式錯誤是 ERROR_SCHEMA；還沒準備好只提醒），pt upload 上傳到平台，平台自動裁切、縮放、重新編碼。
// pt 不在本機改圖：上傳後平台回傳的就是最終樣子。

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

const storeDir = "store"

// 平台的遊戲標籤代碼（玩聚平台改標籤時要一起改）
var storeTags = []string{"party", "deduction", "turn-taking", "quiz", "drawing", "word", "cooperative", "team", "strategy", "reaction", "casual", "music", "icebreaker"}

type storeText struct {
	ZhTW string `json:"zh-TW"`
	En   string `json:"en"`
}

type storeListing struct {
	Name               storeText           `json:"name"`
	Summary            storeText           `json:"summary"`
	Description        storeText           `json:"description"`
	ScreenshotCaptions map[string][]string `json:"screenshotCaptions"`
	Tags               []string            `json:"tags"`
	Players            struct {
		Min int `json:"min"`
		Max int `json:"max"`
	} `json:"players"`
	SessionMinutes int    `json:"sessionMinutes"`
	ReviewNotes    string `json:"reviewNotes"`
}

// storeImage 是 store/ 裡的一張圖片。
type storeImage struct {
	Kind, Locale string
	Position     int
	Path         string
	data         []byte
}

// storeImageSpec 和平台的裁切規格相同（輸出尺寸、原圖短邊下限）
func storeImageSpec(kind, orientation string, w, h int) (outW, outH, minShort int) {
	switch kind {
	case "icon":
		return 1024, 1024, 512
	case "cover":
		return 1920, 1080, 720
	}
	if orientation == "portrait" || (orientation == "any" && h > w) {
		return 1080, 1920, 720
	}
	return 1920, 1080, 720
}

const (
	storeMaxImageBytes  = 10 << 20
	storeMaxImageSide   = 6000
	storeMaxImagePixels = 16_000_000
	storeMaxCropLoss    = 0.25
)

// storePlaceholder：範本的文字還沒改
func storePlaceholder(s string) bool {
	return strings.Contains(s, "（請填寫") || strings.Contains(s, "TODO")
}

// readStore 讀出 store/：listing.json 與圖片。ready 是 false 表示還沒準備好（缺圖片或範本文字沒改）。
func readStore(p *project, add addFunc) (l storeListing, imgs []storeImage, present, ready bool) {
	if p == nil || p.dir == "" {
		return l, nil, false, false
	}
	dir := filepath.Join(p.dir, storeDir)
	if st, err := os.Stat(dir); err != nil || !st.IsDir() {
		if p.hasPolicy() {
			add(levelWarn, "沒有 store/（上架資料）：第一次上架需要名稱、簡介、說明、圖示與截圖；用 pt new 建暫存專案複製 store/ 範本")
		}
		return l, nil, false, false
	}
	raw, err := os.ReadFile(filepath.Join(dir, "listing.json"))
	if err != nil {
		add(levelSchema, "store/ 裡沒有 listing.json")
		return l, nil, true, false
	}
	if !strictDecode(raw, "store/listing.json", jsonKeys(storeListing{}), &l, func(level, format string, a ...any) {
		add(level, strings.ReplaceAll(format, "playtogether.json 的 ", ""), a...)
	}) {
		return l, nil, true, false
	}
	ready = true
	text := func(field string, s *string, max int, multiline bool) {
		*s = strings.TrimSpace(*s)
		if storePlaceholder(*s) {
			ready = false
			return
		}
		n := utf8.RuneCountInString(*s)
		if n == 0 {
			add(levelSchema, "store/listing.json 的 %s 必填（繁中與英文都要填）", field)
			return
		}
		if n > max {
			add(levelSchema, "store/listing.json 的 %s 太長（%d 字，上限 %d）", field, n, max)
		}
		for _, r := range *s {
			if unicode.IsControl(r) && !(multiline && r == '\n') {
				add(levelSchema, "store/listing.json 的 %s 有控制字元%s", field, map[bool]string{true: "", false: "或換行"}[multiline])
				break
			}
		}
	}
	text("name.zh-TW", &l.Name.ZhTW, 30, false)
	text("name.en", &l.Name.En, 30, false)
	text("summary.zh-TW", &l.Summary.ZhTW, 60, false)
	text("summary.en", &l.Summary.En, 80, false)
	text("description.zh-TW", &l.Description.ZhTW, 4000, true)
	text("description.en", &l.Description.En, 4000, true)
	if len(l.Tags) > 5 {
		add(levelSchema, "store/listing.json 的 tags 最多 5 個")
	}
	for _, t := range l.Tags {
		if !slices.Contains(storeTags, t) {
			add(levelSchema, "store/listing.json 的 tags 有不認得的代碼 %q（可用：%s）", t, strings.Join(storeTags, "、"))
		}
	}
	if l.Players.Min < 1 || l.Players.Max > 50 || l.Players.Min > l.Players.Max {
		add(levelSchema, "store/listing.json 的 players 不合法（1–50 人，min 不能大於 max）")
	}
	if l.SessionMinutes < 1 || l.SessionMinutes > 180 {
		add(levelSchema, "store/listing.json 的 sessionMinutes 要是 1–180（一局大約幾分鐘）")
	}
	if utf8.RuneCountInString(l.ReviewNotes) > 4000 {
		add(levelSchema, "store/listing.json 的 reviewNotes 太長（上限 4000 字）")
	}
	if storePlaceholder(l.ReviewNotes) {
		ready = false
	}
	for loc, cs := range l.ScreenshotCaptions {
		if loc != "zh-TW" && loc != "en" {
			add(levelSchema, "store/listing.json 的 screenshotCaptions 只能有 zh-TW、en（目前有 %q）", loc)
		}
		if len(cs) > 10 {
			add(levelSchema, "store/listing.json 的 screenshotCaptions.%s 最多 10 個", loc)
		}
		for _, c := range cs {
			if utf8.RuneCountInString(strings.TrimSpace(c)) > 60 {
				add(levelSchema, "store/listing.json 的截圖說明最多 60 字：%s", c)
			}
		}
	}
	for _, s := range []string{l.Name.ZhTW, l.Name.En, l.Summary.ZhTW, l.Summary.En, l.Description.ZhTW, l.Description.En} {
		if c := prohibitedClaim(s); c != "" {
			add(levelReview, "上架文案有「%s」：不得宣稱或暗示真錢獲利、兌現或提領，平台審核會確認", c)
			break
		}
	}

	// 圖片
	orientation := p.Orientation
	if orientation == "" {
		orientation = "portrait"
	}
	pick := func(base string) string {
		var found []string
		for _, ext := range []string{".png", ".jpg", ".jpeg"} {
			if _, err := os.Stat(filepath.Join(dir, base+ext)); err == nil {
				found = append(found, base+ext)
			}
		}
		if len(found) > 1 {
			add(levelSchema, "store/ 有多個 %s 檔（%s）：只留一個", base, strings.Join(found, "、"))
		}
		if len(found) == 0 {
			return ""
		}
		return found[0]
	}
	if f := pick("icon"); f != "" {
		imgs = append(imgs, storeImage{Kind: "icon", Path: f})
	} else {
		ready = false
	}
	if f := pick("cover"); f != "" {
		imgs = append(imgs, storeImage{Kind: "cover", Path: f})
	}
	for _, loc := range []string{"", "en"} {
		sub := filepath.Join(dir, "screenshots", loc)
		entries, _ := os.ReadDir(sub)
		var names []string
		for _, e := range entries {
			ext := strings.ToLower(filepath.Ext(e.Name()))
			if !e.IsDir() && (ext == ".png" || ext == ".jpg" || ext == ".jpeg") {
				names = append(names, e.Name())
			}
		}
		sort.Strings(names)
		if len(names) > 10 {
			add(levelSchema, "store/screenshots/%s 最多 10 張截圖（目前 %d 張）", loc, len(names))
			names = names[:10]
		}
		if loc == "" && len(names) == 0 {
			ready = false
		}
		for i, n := range names {
			imgs = append(imgs, storeImage{Kind: "screenshot", Locale: loc, Position: i, Path: filepath.ToSlash(filepath.Join("screenshots", loc, n))})
		}
	}
	for i := range imgs {
		im := &imgs[i]
		data, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(im.Path)))
		if err != nil {
			add(levelSchema, "store/%s 讀不到：%v", im.Path, err)
			continue
		}
		im.data = data
		if len(data) > storeMaxImageBytes {
			add(levelSchema, "store/%s 太大（上限 10 MB）", im.Path)
			continue
		}
		cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
		if err != nil || (format != "png" && format != "jpeg") {
			add(levelSchema, "store/%s 不是 PNG 或 JPEG", im.Path)
			continue
		}
		if cfg.Width > storeMaxImageSide || cfg.Height > storeMaxImageSide || cfg.Width*cfg.Height > storeMaxImagePixels {
			add(levelSchema, "store/%s 太大（%d×%d；長邊最多 %d、最多 1600 萬像素）", im.Path, cfg.Width, cfg.Height, storeMaxImageSide)
			continue
		}
		w, h, minShort := storeImageSpec(im.Kind, orientation, cfg.Width, cfg.Height)
		if min(cfg.Width, cfg.Height) < minShort {
			add(levelSchema, "store/%s 太小（%d×%d；短邊至少 %d）", im.Path, cfg.Width, cfg.Height, minShort)
			continue
		}
		target := float64(w) / float64(h)
		cw, ch := float64(cfg.Width), float64(cfg.Height)
		if cw/ch > target {
			cw = math.Round(ch * target)
		} else {
			ch = math.Round(cw / target)
		}
		if (float64(cfg.Width)-cw)/float64(cfg.Width) > storeMaxCropLoss || (float64(cfg.Height)-ch)/float64(cfg.Height) > storeMaxCropLoss {
			hint := "請換一張"
			if im.Kind == "screenshot" {
				hint = fmt.Sprintf("請用 pt store screenshot%s 重新擷取（依 playtogether.json 的畫面方向 %s）", map[bool]string{true: " -locale en", false: ""}[im.Locale == "en"], orientation)
			}
			add(levelSchema, "store/%s 的比例（%d×%d）和需要的 %d×%d 差太多，平台不能自動裁切（例如把橫式圖用在直式遊戲）：%s", im.Path, cfg.Width, cfg.Height, w, h, hint)
		}
	}
	shared := 0
	for _, im := range imgs {
		if im.Kind == "screenshot" && im.Locale == "" {
			shared++
		}
	}
	for loc, cs := range l.ScreenshotCaptions {
		if len(cs) > shared && shared > 0 {
			add(levelWarn, "store/listing.json 的 screenshotCaptions.%s 有 %d 個，但只有 %d 張截圖", loc, len(cs), shared)
		}
	}
	if !ready {
		add(levelInfo, "store/ 的上架資料還沒準備好（範本文字還沒改，或缺圖示、截圖）：pt upload 會只上傳版本，第一次上架前要補齊")
	}
	return l, imgs, true, ready
}

// prohibitedClaim 是上架文案不能有的宣稱（和平台相同，只用來提醒；最終由審核判斷）。
func prohibitedClaim(s string) string {
	l := strings.ToLower(s)
	for _, w := range []string{"可兌現", "兌換現金", "換現金", "提領", "出金", "穩賺", "賺錢", "真錢", "現金獎", "贏錢",
		"cash out", "cashout", "withdraw", "real money", "win money", "earn money"} {
		if strings.Contains(l, w) {
			return w
		}
	}
	return ""
}

// checkStore 是 pt check 的上架資料部分。
func checkStore(p *project, add addFunc) { readStore(p, add) }

// declarationBody 是 pt upload 送給平台的宣告快照（v0.15 的五個區塊；舊格式不送）。
func declarationBody(p *project) (json.RawMessage, bool) {
	if p == nil || !p.hasPolicy() {
		return nil, false
	}
	body := map[string]any{"platformAudience": p.PlatformAudience, "contentProfile": p.ContentProfile,
		"sensitiveContent": p.SensitiveContent, "gameplay": p.Gameplay, "economy": p.Economy}
	// 選填的 gameServer 區塊：有使用 AI 服務才送（平台判斷為需要審核）
	if ai := p.aiProviders(); len(ai) > 0 {
		body["gameServer"] = map[string][]string{"ai": ai}
	}
	raw, _ := json.Marshal(body)
	return raw, true
}

// pushListing 上傳上架資料：建立修訂（內容相同時平台沿用既有的）並上傳每張圖片。回傳修訂 ID。
func pushListing(c client, p *project, l storeListing, imgs []storeImage) (string, bool, error) {
	orientation := p.Orientation
	if orientation == "" {
		orientation = "portrait"
	}
	type imgReq struct {
		Kind     string `json:"kind"`
		Locale   string `json:"locale"`
		Position int    `json:"position"`
		SHA256   string `json:"sha256"`
		Size     int    `json:"size"`
	}
	reqImgs := []imgReq{}
	byKey := map[string][]byte{}
	for _, im := range imgs {
		sum := sha256.Sum256(im.data)
		reqImgs = append(reqImgs, imgReq{im.Kind, im.Locale, im.Position, hex.EncodeToString(sum[:]), len(im.data)})
		byKey[fmt.Sprintf("%s/%s/%d", im.Kind, im.Locale, im.Position)] = im.data
	}
	captions := l.ScreenshotCaptions
	if captions == nil {
		captions = map[string][]string{}
	}
	body := map[string]any{"orientation": orientation, "name": l.Name, "summary": l.Summary, "description": l.Description,
		"screenshotCaptions": captions, "tags": l.Tags, "players": map[string]int{"min": l.Players.Min, "max": l.Players.Max},
		"sessionMinutes": l.SessionMinutes, "reviewNotes": l.ReviewNotes, "images": reqImgs}
	var out struct {
		RevisionID string `json:"revisionId"`
		Reused     bool   `json:"reused"`
		Uploads    []struct {
			Kind       string `json:"kind"`
			Locale     string `json:"locale"`
			Position   int    `json:"position"`
			UploadPath string `json:"uploadPath"`
		} `json:"uploads"`
	}
	if err := c.signed("POST", "/v1/dev/listing-revisions", body, &out); err != nil {
		return "", false, fmt.Errorf("上傳上架資料失敗：%w", err)
	}
	for _, u := range out.Uploads {
		data := byKey[fmt.Sprintf("%s/%s/%d", u.Kind, u.Locale, u.Position)]
		req, _ := http.NewRequest(http.MethodPut, c.base+u.UploadPath, bytes.NewReader(data))
		if err := doJSON(req, nil); err != nil {
			name := u.Kind
			if u.Kind == "screenshot" {
				name = fmt.Sprintf("截圖 %s%d", map[bool]string{true: "en/", false: ""}[u.Locale == "en"], u.Position+1)
			}
			return out.RevisionID, false, fmt.Errorf("上傳%s失敗：%w", name, err)
		}
	}
	return out.RevisionID, out.Reused, nil
}

// cmdStatus：pt status 查詢版本與上架資料的審核狀態、退件原因、正式分級。
func cmdStatus(args []string) error {
	_, c, err := projectClient()
	if err != nil {
		return err
	}
	st, err := fetchDevStatus(c)
	if err != nil {
		return err
	}
	fmt.Printf("%s：%s", st.Game.Name, map[string]string{"draft": "尚未上架", "live": "上架中", "unlisted": "已下架", "disabled": "停用"}[st.Game.Status])
	if st.Game.LiveVersion != "" {
		fmt.Printf("（目前版本 %s）", st.Game.LiveVersion)
	}
	if st.Game.Rating == nil {
		fmt.Println("\n正式分級：尚未審核（由玩聚平台審核決定）")
	} else {
		fmt.Printf("\n正式分級：%s（由玩聚平台審核決定）\n", *st.Game.Rating)
	}
	if len(st.Game.ContentLabels) > 0 || st.Game.Warning != "" {
		fmt.Printf("內容標示：%s　警語：%s\n", strings.Join(st.Game.ContentLabels, "、"), st.Game.Warning)
	}
	status := map[string]string{"draft": "草稿（還沒送審）", "in_review": "審核中", "approved": "已核准（等待發佈）", "rejected": "退件",
		"released": "已發佈", "withdrawn": "已撤回", "replaced": "已被取代"}
	fmt.Println("\n版本：")
	for _, b := range st.Builds {
		line := fmt.Sprintf("  %s  %s", b.Version, status[b.Status])
		if b.ArtifactStatus != "ready" {
			line += "（檔案：" + b.ArtifactStatus + "）"
		}
		fmt.Println(line)
		if b.Status == "rejected" {
			fmt.Printf("      退件：%s %s\n", strings.Join(b.Codes, "、"), b.Reason)
		}
		if b.PolicyFindings != nil {
			for _, f := range b.PolicyFindings.Findings {
				if f.Level == "review" {
					fmt.Printf("      需要平台審核：%s\n", f.Reason)
				}
			}
		}
	}
	fmt.Println("\n上架資料：")
	for _, r := range st.ListingRevisions {
		line := fmt.Sprintf("  %s  %s", r.ID[:8], status[r.Status])
		if r.Minor {
			line += "（小幅修改，免審已生效）"
		}
		if r.PendingImages > 0 || r.FailedImages > 0 {
			line += fmt.Sprintf("（圖片：%d 張未上傳、%d 張失敗）", r.PendingImages, r.FailedImages)
		}
		fmt.Println(line)
		if r.Status == "rejected" {
			fmt.Printf("      退件：%s %s\n", strings.Join(r.Codes, "、"), r.Reason)
		}
	}
	if len(st.ListingRevisions) == 0 {
		fmt.Println("  （還沒有上傳過；準備好 store/ 後執行 pt upload）")
	}
	return nil
}

// uploadStore 在 pt upload 時上傳上架資料：store/ 還沒準備好或格式錯誤時不上傳（格式錯誤在上傳前檢查就已經擋下）。
func uploadStore(c client, p *project) (string, error) {
	var bad bool
	l, imgs, present, ready := readStore(p, func(level, format string, a ...any) {
		if isErrorLevel(level) {
			bad = true
		}
	})
	switch {
	case !present:
		return "", nil
	case bad || !ready:
		fmt.Println("上架資料（store/）還沒準備好，這次只上傳版本（pt check 會列出缺什麼）")
		return "", nil
	}
	id, reused, err := pushListing(c, p, l, imgs)
	if err != nil && strings.Contains(err.Error(), "平台回應 404") && id == "" {
		// 平台還沒開放上架資料（較舊的平台版本，或沒有開啟媒體目錄）：版本照常上傳
		fmt.Println("提醒：這個玩聚平台還不接受上架資料，這次只上傳版本")
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if reused {
		fmt.Println("上架資料沒有變更，沿用修訂 ID：", id)
	} else {
		fmt.Printf("已上傳上架資料（%d 張圖片，平台已自動裁切），修訂 ID：%s\n", len(imgs), id)
	}
	return id, nil
}

// uploadListingOnly：pt upload -listing-only，只上傳上架資料。
func uploadListingOnly(p *project, api, keyID, keyFile string, noSubmit bool) error {
	if p == nil {
		return errors.New("-listing-only 要在 pt new 建立的專案裡執行")
	}
	first := func(vals ...string) string {
		for _, v := range vals {
			if v != "" {
				return v
			}
		}
		return ""
	}
	base := strings.TrimRight(first(api, os.Getenv("PT_API_URL"), p.Platform), "/")
	kid := first(keyID, os.Getenv("PT_UPLOAD_KEY_ID"), p.UploadKeyID)
	kfile := first(keyFile, os.Getenv("PT_UPLOAD_KEY_FILE"), p.path(p.UploadKeyFile))
	if err := validPlatform(base); err != nil {
		return err
	}
	if kid == "" {
		return errors.New("需要上傳金鑰：填在 playtogether.json（uploadKeyId、uploadKeyFile）")
	}
	fmt.Println("上架資料檢查：")
	var fs []finding
	readStore(p, func(level, format string, a ...any) { fs = append(fs, finding{level, fmt.Sprintf(format, a...)}) })
	if res := printFindings(fs); res.blocking() || res.build {
		return errors.New("上架資料有錯誤，請修正後再上傳")
	}
	priv, err := readKey(kfile)
	if err != nil {
		return err
	}
	id, err := uploadStore(client{base: base, keyID: kid, priv: priv}, p)
	if err != nil {
		return err
	}
	if id == "" {
		return errors.New("上架資料還沒準備好")
	}
	if !noSubmit {
		logged, err := submitWithCredential(base, "", id, false, nil)
		if err != nil {
			return err
		}
		if logged {
			return nil
		}
	}
	fmt.Printf("下一步：執行 pt login 授權後會直接送審，或到開發者頁送審上架資料修訂 %s（只改簡介、說明的小幅修改會直接生效）。審核進度：pt status\n", id)
	return nil
}
