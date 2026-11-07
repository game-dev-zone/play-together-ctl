package main

// pt login：裝置授權。pt 向平台要一組短碼，開發者在瀏覽器用玩聚帳號確認短碼與團隊，pt 取得送審授權（綁定帳號與團隊，30 天）。
// 授權存在使用者設定資料夾（權限 600），依平台網址分開；之後 pt upload 會在檢查通過後直接送審。
// 上傳金鑰只能上傳，不能送審；送審授權只能送審，不能上傳或發佈。

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type credential struct {
	Token     string    `json:"token"`
	TeamID    string    `json:"teamId"`
	TeamName  string    `json:"teamName"`
	ExpiresAt time.Time `json:"expiresAt"`
}

func credentialsPath() (string, error) {
	if p := os.Getenv("PT_CREDENTIALS_FILE"); p != "" {
		return p, nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "playtogether", "credentials.json"), nil
}

func loadCredentials() map[string]credential {
	out := map[string]credential{}
	p, err := credentialsPath()
	if err != nil {
		return out
	}
	raw, err := os.ReadFile(p)
	if err == nil {
		_ = json.Unmarshal(raw, &out)
	}
	return out
}

func saveCredentials(c map[string]credential) error {
	p, err := credentialsPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return err
	}
	raw, _ := json.MarshalIndent(c, "", "  ")
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}

// credentialFor 回傳這個平台還有效的送審授權。
func credentialFor(base string) (credential, bool) {
	c, ok := loadCredentials()[strings.TrimRight(base, "/")]
	return c, ok && c.Token != "" && time.Now().Before(c.ExpiresAt)
}

// errNoPlatform：找不到平台網址時的說明。平台網址就是玩聚網站的網址，平台 API 在同一個網址的 /v1 底下。
var errNoPlatform = errors.New("需要平台網址：在 pt new 建立的專案裡執行（讀 playtogether.json 的 platform），或加上 -api <平台網址>，也可以設定環境變數 PT_API_URL。\n" +
	"平台網址就是玩聚網站的網址，例如 https://玩聚網域（平台 API 在同一個網址的 /v1 底下，不用另外加）；見開發者文件「快速開始」的「平台網址」")

// templatePlatform 是 pt new 寫進 playtogether.json 的預設值，代表還沒填。
const templatePlatform = "https://（平台網址）"

// validPlatform 檢查平台網址：不是範本預設值、格式正確、只有 scheme 與網域（不能帶 /v1、/api 等路徑）。
func validPlatform(base string) error {
	if base == "" {
		return errNoPlatform
	}
	if strings.Contains(base, "（") || strings.Contains(base, "平台網址") {
		return fmt.Errorf("平台網址還是範本的預設值 %s：請把 playtogether.json 的 platform 改成玩聚網站的網址，或加上 -api <平台網址>", templatePlatform)
	}
	u, err := url.Parse(base)
	if err != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") {
		return fmt.Errorf("平台網址格式錯誤：%s（要像 https://玩聚網域）", base)
	}
	if strings.Trim(u.Path, "/") != "" || u.RawQuery != "" || u.Fragment != "" {
		return fmt.Errorf("平台網址不要加路徑：%s。平台 API 固定在同一個網址的 /v1 底下，請改成 %s://%s", base, u.Scheme, u.Host)
	}
	return nil
}

// securePlatform：送審授權只能透過 https 傳送（本機開發的 localhost 例外）。
func securePlatform(base string) error {
	if err := validPlatform(base); err != nil {
		return err
	}
	u, _ := url.Parse(base)
	if u.Scheme == "http" && !isLocal(u.Hostname()) {
		return fmt.Errorf("平台網址要用 https://（送審授權不能用沒有加密的連線傳送）：%s", base)
	}
	return nil
}

func platformBase(p *project, flagAPI string) string {
	for _, v := range []string{flagAPI, os.Getenv("PT_API_URL")} {
		if v != "" {
			return strings.TrimRight(v, "/")
		}
	}
	if p != nil {
		return strings.TrimRight(p.Platform, "/")
	}
	return ""
}

func devPost(target string, body any, header map[string]string) (int, map[string]any, error) {
	raw, _ := json.Marshal(body)
	req, err := http.NewRequest(http.MethodPost, target, bytes.NewReader(raw))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range header {
		req.Header.Set(k, v)
	}
	res, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer res.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	var out map[string]any
	_ = json.Unmarshal(data, &out)
	return res.StatusCode, out, nil
}

func cmdLogin(args []string) error {
	fs := flag.NewFlagSet("login", flag.ExitOnError)
	api := fs.String("api", "", "平台網址（或 PT_API_URL；在專案裡預設讀 playtogether.json 的 platform）")
	noBrowser := fs.Bool("no-browser", false, "不要自動打開瀏覽器")
	register := fs.Bool("register", false, "另外要求建立遊戲與管理金鑰的權限（pt register；團隊擁有者、管理者；7 天）")
	_ = fs.Parse(reorder(args))
	p, err := findProject()
	if err != nil {
		return err
	}
	base := platformBase(p, *api)
	if err := securePlatform(base); err != nil {
		return err
	}
	label, _ := os.Hostname()
	if len([]rune(label)) > 60 {
		label = string([]rune(label)[:60])
	}
	req := map[string]any{"label": label}
	if *register {
		req["scopes"] = []string{"submit", "register"}
	}
	st, dc, err := devPost(base+"/v1/dev/device-codes", req, nil)
	if err != nil {
		return err
	}
	if st != http.StatusCreated {
		return fmt.Errorf("平台回應 %d %v", st, dc["error"])
	}
	device, _ := dc["deviceCode"].(string)
	user, _ := dc["userCode"].(string)
	verify, _ := dc["verificationUrl"].(string)
	interval := 5 * time.Second
	if n, ok := dc["interval"].(float64); ok && n > 0 {
		interval = time.Duration(n) * time.Second
	}
	expires := time.Now().Add(10 * time.Minute)
	if n, ok := dc["expiresIn"].(float64); ok {
		expires = time.Now().Add(time.Duration(n) * time.Second)
	}
	fmt.Printf("在瀏覽器打開下面的網址，用玩聚帳號登入，確認短碼是 %s，再選擇團隊：\n  %s\n", user, verify)
	if !*noBrowser && interactive() && !ciEnv() {
		openBrowser(verify)
	}
	fmt.Print("等待確認")
	for time.Now().Before(expires) {
		time.Sleep(interval)
		st, out, err := devPost(base+"/v1/dev/device-codes/token", map[string]string{"deviceCode": device}, nil)
		if err != nil {
			return err
		}
		switch {
		case st == http.StatusOK:
			c := credential{TeamID: fmt.Sprint(out["teamId"]), TeamName: fmt.Sprint(out["teamName"])}
			c.Token, _ = out["token"].(string)
			c.ExpiresAt, _ = time.Parse(time.RFC3339Nano, fmt.Sprint(out["expiresAt"]))
			all := loadCredentials()
			if *register {
				all[registerKey(base)] = c // 和送審授權分開保存：建立遊戲的授權比較短，過期不影響送審
			} else {
				all[base] = c
			}
			if err := saveCredentials(all); err != nil {
				return fmt.Errorf("保存授權失敗：%w", err)
			}
			if *register {
				fmt.Printf("\n已授權建立遊戲與管理金鑰：團隊 %s（到 %s）。下一步：pt register；撤銷請到開發者頁。\n", c.TeamName, c.ExpiresAt.Local().Format("2006-01-02"))
				return nil
			}
			fmt.Printf("\n已授權：團隊 %s（到 %s）。之後 pt upload 會在檢查通過後直接送審；撤銷請到開發者頁。\n", c.TeamName, c.ExpiresAt.Local().Format("2006-01-02"))
			return nil
		case out["error"] == "authorization_pending":
			fmt.Print(".")
		case out["error"] == "slow_down":
			interval += 5 * time.Second
		case out["error"] == "access_denied":
			fmt.Println()
			return errors.New("已在瀏覽器拒絕授權")
		case out["error"] == "expired_token":
			fmt.Println()
			return errors.New("短碼已經過期，請重新執行 pt login")
		default:
			fmt.Println()
			return fmt.Errorf("平台回應 %d %v", st, out["error"])
		}
	}
	fmt.Println()
	return errors.New("等待逾時，請重新執行 pt login")
}

func cmdLogout(args []string) error {
	fs := flag.NewFlagSet("logout", flag.ExitOnError)
	api := fs.String("api", "", "平台網址（或 PT_API_URL）")
	_ = fs.Parse(reorder(args))
	p, err := findProject()
	if err != nil {
		return err
	}
	base := platformBase(p, *api)
	all := loadCredentials()
	_, a1 := all[base]
	_, a2 := all[registerKey(base)]
	if !a1 && !a2 {
		fmt.Println("這個平台沒有保存的授權")
		return nil
	}
	delete(all, base)
	delete(all, registerKey(base))
	if err := saveCredentials(all); err != nil {
		return err
	}
	fmt.Println("已刪除這台電腦上的授權（平台上的授權請到開發者頁撤銷）")
	return nil
}

var submitErrors = map[string]string{
	"login_required":    "送審授權已過期或被撤銷，請重新執行 pt login",
	"not_member":        "你不是這款遊戲所屬團隊中可以送審的成員",
	"not_found":         "找不到這個版本或上架資料（或它不屬於你授權的團隊）",
	"bad_build":         "這個版本不能送審（檔案還沒就緒，或已經審核過）",
	"build_in_review":   "這款遊戲已經有其他版本在審核中",
	"listing_in_review": "這款遊戲已經有上架資料在審核中",
	"images_pending":    "還有上架圖片沒有上傳或處理失敗，請重新執行 pt upload",
	"not_draft":         "這份上架資料已經送審過",
	"replace_mismatch":  "要取代的版本已經不在審核中，請重新執行",
	"not_in_review":     "這個版本已經不在審核中",
}

// submitWithCredential 用送審授權送審版本和／或上架資料修訂。logged 是 false 表示沒有登入（不是錯誤）。
// replace：這款遊戲有其他版本在審核中時，直接撤回它、改送這個版本（-replace）；沒有指定時在終端機詢問。
// inReview 查目前審核中的版本（用上傳金鑰；nil 時不處理取代）。
func submitWithCredential(base, buildID, revisionID string, replace bool, inReview func() (id, version string, err error)) (logged bool, err error) {
	creds := submitCredentials(base)
	if len(creds) == 0 {
		return false, nil
	}
	if err := securePlatform(base); err != nil {
		return true, err
	}
	body := map[string]string{"buildId": buildID, "revisionId": revisionID}
	st, out, err := devPostWithCredentials(base+"/v1/dev/submissions", body, creds)
	if err != nil {
		return true, err
	}
	// 其他版本審核中：撤回它、改送這個版本（App Store 的「從審核中移除」）
	if st == http.StatusConflict && out["error"] == "build_in_review" && buildID != "" && inReview != nil {
		id, version, ierr := inReview()
		if ierr == nil && id != "" && id != buildID {
			ok := replace
			if !ok && interactive() && !ciEnv() {
				ok, _ = confirm(fmt.Sprintf("%s 正在審核中。要撤回它、改送這個版本嗎？（撤回後一起送審的上架資料會回到草稿，和這個版本一起重新送審）", version), false)
			}
			if !ok {
				return true, fmt.Errorf("送審失敗：%s 正在審核中。要改送這個版本：pt submit -replace（或 pt withdraw 撤回後再 pt submit）", version)
			}
			body["replaceBuildId"] = id
			if st, out, err = devPostWithCredentials(base+"/v1/dev/submissions", body, creds); err != nil {
				return true, err
			}
			if st == http.StatusOK {
				fmt.Printf("已撤回 %s 的送審。\n", version)
			}
		}
	}
	if st != http.StatusOK {
		if t, ok := submitErrors[fmt.Sprint(out["error"])]; ok {
			return true, errors.New("送審失敗：" + t)
		}
		return true, fmt.Errorf("送審失敗：平台回應 %d %v", st, out["error"])
	}
	switch {
	case out["minor"] == true:
		fmt.Println("已送審：上架資料是小幅修改，已直接生效（平台會事後抽查）。")
	case out["listingStatus"] == "unchanged":
		fmt.Println("已送審版本（上架資料沒有變更，沿用目前生效的）。")
	case buildID != "" && revisionID != "":
		fmt.Println("已送審：版本和上架資料一起審核，發佈版本時上架資料一起生效。")
	default:
		fmt.Println("已送審，等待玩聚平台審核。")
	}
	return true, nil
}

// submitCredentials 是可以送審的授權：pt login 的送審授權，再來是 pt login -register 的（範圍也含送審）。
func submitCredentials(base string) []credential {
	var out []credential
	if c, ok := credentialFor(base); ok {
		out = append(out, c)
	}
	if c, ok := credentialFor(registerKey(base)); ok {
		out = append(out, c)
	}
	return out
}

// devPostWithCredentials 依序用每一份授權送出；授權失效（login_required）時換下一份。
func devPostWithCredentials(target string, body any, creds []credential) (int, map[string]any, error) {
	var st int
	var out map[string]any
	var err error
	for _, c := range creds {
		st, out, err = devPost(target, body, map[string]string{"Authorization": "Bearer " + c.Token})
		if err != nil || !(st == http.StatusUnauthorized && out["error"] == "login_required") {
			return st, out, err
		}
	}
	return st, out, err
}
