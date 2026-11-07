package main

// pt submit：把已經上傳、檢查通過的草稿版本（與草稿上架資料）送審，不重新上傳。
// pt withdraw：撤回審核中的版本（和 App Store 的「從審核中移除」相同；一起送審的上架資料回到草稿）。

import (
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
)

// devStatus 是 GET /v1/dev/status（用上傳金鑰簽章）的回應。
type devStatus struct {
	Game struct {
		Name, Status, Warning, LiveVersion string
		Rating                             *string
		ContentLabels                      []string
	} `json:"game"`
	Builds []struct {
		ID, Version, Status, ArtifactStatus, Reason string
		Codes                                       []string
		PolicyFindings                              *struct {
			Findings []struct{ Level, Reason string } `json:"findings"`
		} `json:"policyFindings"`
	} `json:"builds"`
	ListingRevisions []struct {
		ID, Status, Reason string
		BuildID            *string `json:"buildId"`
		Minor              bool
		Rating             *string
		Codes              []string
		PendingImages      int `json:"pendingImages"`
		FailedImages       int `json:"failedImages"`
	} `json:"listingRevisions"`
}

// projectClient 讀專案的平台網址與上傳金鑰（環境變數 > playtogether.json，和 pt upload 相同）。
func projectClient() (string, client, error) {
	p, err := findProject()
	if err != nil {
		return "", client{}, err
	}
	if p == nil {
		return "", client{}, errors.New("找不到 playtogether.json：請在 pt new 建立的專案裡執行")
	}
	first := func(vals ...string) string {
		for _, v := range vals {
			if v != "" {
				return v
			}
		}
		return ""
	}
	base := strings.TrimRight(first(os.Getenv("PT_API_URL"), p.Platform), "/")
	kid, kfile := first(os.Getenv("PT_UPLOAD_KEY_ID"), p.UploadKeyID), first(os.Getenv("PT_UPLOAD_KEY_FILE"), p.path(p.UploadKeyFile))
	if err := validPlatform(base); err != nil {
		return "", client{}, err
	}
	if kid == "" {
		return "", client{}, errors.New("需要上傳金鑰：填在 playtogether.json（uploadKeyId、uploadKeyFile）")
	}
	priv, err := readKey(kfile)
	if err != nil {
		return "", client{}, err
	}
	return base, client{base: base, keyID: kid, priv: priv}, nil
}

func fetchDevStatus(c client) (devStatus, error) {
	var st devStatus
	err := c.signed("GET", "/v1/dev/status", nil, &st)
	return st, err
}

// inReviewBuild 回傳查詢目前審核中版本的函式（送審遇到 build_in_review 時用）。
func inReviewBuild(c client) func() (string, string, error) {
	return func() (string, string, error) {
		st, err := fetchDevStatus(c)
		if err != nil {
			return "", "", err
		}
		for _, b := range st.Builds {
			if b.Status == "in_review" {
				return b.ID, b.Version, nil
			}
		}
		return "", "", nil
	}
}

func cmdSubmit(args []string) error {
	fs := flag.NewFlagSet("submit", flag.ExitOnError)
	version := fs.String("version", "", "要送審的版本（預設：最新一個檢查通過的草稿版本）")
	replace := fs.Bool("replace", false, "這款遊戲有其他版本在審核中時，撤回它、改送這個版本")
	noListing := fs.Bool("no-listing", false, "只送審版本，不帶草稿上架資料")
	_ = fs.Parse(reorder(args))
	base, c, err := projectClient()
	if err != nil {
		return err
	}
	st, err := fetchDevStatus(c)
	if err != nil {
		return err
	}
	buildID, buildVersion := "", ""
	for _, b := range st.Builds { // 平台依建立時間由新到舊排列
		if b.Status == "draft" && b.ArtifactStatus == "ready" && (*version == "" || b.Version == *version) {
			buildID, buildVersion = b.ID, b.Version
			break
		}
	}
	if *version != "" && buildID == "" {
		return fmt.Errorf("找不到可以送審的版本 %s（要是已經檢查通過、還沒送審的草稿；pt status 看狀態）", *version)
	}
	revID := ""
	if !*noListing {
		for _, r := range st.ListingRevisions {
			if r.Status == "draft" && r.PendingImages == 0 && r.FailedImages == 0 {
				revID = r.ID
				break
			}
		}
	}
	if buildID == "" && revID == "" {
		return errors.New("沒有可以送審的草稿版本或上架資料：先用 pt upload 上傳（pt status 看狀態）")
	}
	if buildID != "" {
		fmt.Printf("送審版本 %s", buildVersion)
		if revID != "" {
			fmt.Print("與上架資料")
		}
		fmt.Println()
	}
	logged, err := submitWithCredential(base, buildID, revID, *replace, inReviewBuild(c))
	if err != nil {
		return err
	}
	if !logged {
		return errors.New("還沒有送審授權：執行 pt login")
	}
	fmt.Println("審核進度：pt status（或開發者頁）")
	return nil
}

func cmdWithdraw(args []string) error {
	fs := flag.NewFlagSet("withdraw", flag.ExitOnError)
	_ = fs.Parse(reorder(args))
	base, c, err := projectClient()
	if err != nil {
		return err
	}
	st, err := fetchDevStatus(c)
	if err != nil {
		return err
	}
	body, what := map[string]string{}, ""
	for _, b := range st.Builds {
		if b.Status == "in_review" {
			body["buildId"], what = b.ID, "版本 "+b.Version
			break
		}
	}
	if what == "" { // 沒有審核中的版本：只送上架資料的審核中修訂
		for _, r := range st.ListingRevisions {
			if r.Status == "in_review" && r.BuildID == nil {
				body["revisionId"], what = r.ID, "上架資料"
				break
			}
		}
	}
	if what == "" {
		return errors.New("沒有審核中的版本或上架資料")
	}
	creds := submitCredentials(base)
	if len(creds) == 0 {
		return errors.New("還沒有送審授權：執行 pt login")
	}
	if err := securePlatform(base); err != nil {
		return err
	}
	code, out, err := devPostWithCredentials(base+"/v1/dev/submissions/withdraw", body, creds)
	if err != nil {
		return err
	}
	if code != http.StatusOK {
		if t, ok := submitErrors[fmt.Sprint(out["error"])]; ok {
			return errors.New("撤回失敗：" + t)
		}
		return fmt.Errorf("撤回失敗：平台回應 %d %v", code, out["error"])
	}
	fmt.Printf("已撤回%s的送審。", what)
	if body["buildId"] != "" {
		fmt.Print("一起送審的上架資料回到草稿；修正後 pt upload 上傳新版本（會一起送審）。")
	}
	fmt.Println()
	return nil
}
