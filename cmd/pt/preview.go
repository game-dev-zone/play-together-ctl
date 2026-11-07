package main

import (
	"errors"
	"flag"
	"fmt"
	"net/http"
	"slices"
)

// previewErrors：開預覽房間的錯誤碼（平台的 /v1/dev/builds/{id}/preview-room）。
var previewErrors = map[string]string{
	"not_found":                  "找不到這個版本（或不屬於登入時選的團隊）",
	"not_member":                 "只有團隊擁有者、管理員、管理者與開發者可以開預覽房間",
	"bad_state":                  "這個版本不能預覽（已撤回、退件，或檔案還沒檢查通過；pt status 看狀態）",
	"game_disabled":              "遊戲已停用",
	"onboarding_required":        "這個玩聚帳號還沒完成成人自我聲明：先在玩聚網站登入並完成首次設定",
	"too_many_previews":          "同時開著的預覽房間太多了（每人 3 間、每個團隊 5 間）；等舊的到期（1 小時）再開",
	"preview_server_unavailable": "這個版本有玩聚代管的遊戲伺服器，預覽要等平台啟動伺服器（送審後由審核者啟動，或聯絡平台）；可以先用 pt dev 在本機測試",
	"login_required":             "送審授權已失效：重新執行 pt login",
}

// cmdPreview：開一個預覽房間（綁定這個版本、1 小時、團隊成員才能進），印出網址。
func cmdPreview(args []string) error {
	fs := flag.NewFlagSet("preview", flag.ExitOnError)
	version := fs.String("version", "", "要預覽的版本（預設：最新一個檢查通過、沒有撤回或退件的版本）")
	_ = fs.Parse(reorder(args))
	base, c, err := projectClient()
	if err != nil {
		return err
	}
	st, err := fetchDevStatus(c)
	if err != nil {
		return err
	}
	buildID, buildVersion := previewBuild(st, *version)
	if buildID == "" {
		if *version != "" {
			return fmt.Errorf("找不到可以預覽的版本 %s（要檢查通過、沒有撤回或退件；pt status 看狀態）", *version)
		}
		return errors.New("沒有可以預覽的版本：先用 pt upload 上傳（pt status 看狀態）")
	}
	creds := submitCredentials(base)
	if len(creds) == 0 {
		return errors.New("還沒有送審授權：執行 pt login")
	}
	if err := securePlatform(base); err != nil {
		return err
	}
	code, out, err := devPostWithCredentials(base+"/v1/dev/builds/"+buildID+"/preview-room", map[string]string{}, creds)
	if err != nil {
		return err
	}
	if code != http.StatusOK {
		if t, ok := previewErrors[fmt.Sprint(out["error"])]; ok {
			return errors.New("開預覽房間失敗：" + t)
		}
		return fmt.Errorf("開預覽房間失敗：平台回應 %d %v", code, out["error"])
	}
	fmt.Printf("已開 %s 的預覽房間（1 小時內有效，只有團隊成員能進）：\n", buildVersion)
	fmt.Println(out["url"])
	fmt.Println("用手機打開這個網址，以玩聚帳號登入就能玩；多人測試請團隊其他成員用自己的帳號登入同一個網址。")
	return nil
}

// previewBuild：要預覽的版本。沒指定時是最新一個檢查通過、沒有撤回或退件的版本（平台依建立時間由新到舊排列）。
func previewBuild(st devStatus, version string) (id, ver string) {
	for _, b := range st.Builds {
		if slices.Contains([]string{"rejected", "withdrawn"}, b.Status) || (b.ArtifactStatus != "ready" && b.ArtifactStatus != "external") {
			continue
		}
		if version == "" || b.Version == version {
			return b.ID, b.Version
		}
	}
	return "", ""
}
