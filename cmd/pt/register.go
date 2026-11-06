package main

// pt register：自助建立草稿遊戲並登記金鑰（團隊擁有者、管理者）。
//
//	pt login -register   瀏覽器同意「建立遊戲與管理金鑰」（7 天；和送審授權分開保存）
//	pt register          沒有私鑰就產生 → 平台建立草稿遊戲、登記公鑰 → 金鑰 ID 寫回 playtogether.json
//
// 私鑰只留在這台電腦；平台只收公鑰。逾時重試時用 GET /v1/dev/games/{id} 查回已建立的結果（公鑰相同才接手）。

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// registerKey 是 register 授權在 credentials.json 裡的鍵（和送審授權分開，過期不影響 pt upload 送審）。
func registerKey(base string) string { return strings.TrimRight(base, "/") + "#register" }

var registerErrors = map[string]string{
	"login_required":      "建立遊戲的授權已過期或被撤銷，請重新執行 pt login -register",
	"insufficient_scope":  "這個授權不能建立遊戲，請執行 pt login -register",
	"not_allowed":         "只有團隊擁有者與管理者可以建立遊戲；請團隊擁有者執行 pt register，或請他調整你的角色",
	"id_unavailable":      "這個遊戲代號不能使用（已經有人用了或是保留字）：改 playtogether.json 的 name 再試",
	"public_key_taken":    "這把公鑰已經登記在別的遊戲：刪掉 keys/ 裡的私鑰重新產生（pt keygen）再試",
	"draft_limit_reached": "團隊的草稿遊戲已經有 10 款：先把其中幾款送審上架，或聯絡平台",
	"hosting_disabled":    "這個平台還沒開放託管遊戲，請聯絡平台",
	"rate_limited":        "嘗試太多次，請過一陣子再試",
	"bad_id":              "遊戲代號格式不對（英文小寫、數字、連字號）",
	"bad_name":            "遊戲名稱要 1–40 字",
	"bad_key":             "金鑰格式不對：刪掉 keys/ 裡的私鑰重新產生再試",
}

// ensureKey 讀取私鑰；沒有就產生（權限 600）。檔案存在但格式不對時停下來，不覆寫。
func ensureKey(path string) (ed25519.PrivateKey, bool, error) {
	raw, err := os.ReadFile(path)
	if err == nil {
		seed, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(raw)))
		if err != nil || len(seed) != ed25519.SeedSize {
			return nil, false, fmt.Errorf("%s 不是 pt keygen 產生的私鑰；確認內容，或刪掉後重新執行", path)
		}
		return ed25519.NewKeyFromSeed(seed), false, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, false, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, false, err
	}
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, false, err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return nil, false, err
	}
	if _, err := f.WriteString(base64.StdEncoding.EncodeToString(priv.Seed()) + "\n"); err != nil {
		f.Close()
		return nil, false, err
	}
	return priv, true, f.Close()
}

func pubB64(k ed25519.PrivateKey) string {
	return base64.StdEncoding.EncodeToString(k.Public().(ed25519.PublicKey))
}

func devGet(target, token string) (int, map[string]any, error) {
	req, err := http.NewRequest(http.MethodGet, target, nil)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
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

// recoverRegistration：遊戲已經存在而且屬於這個團隊時（上次建立成功但沒收到回應），用公鑰比對找回金鑰 ID。
func recoverRegistration(base, token, id, uploadPub, serverPub string) (uploadID, serverID string, ok bool) {
	st, g, err := devGet(base+"/v1/dev/games/"+id, token)
	if err != nil || st != http.StatusOK {
		return "", "", false
	}
	keys, _ := g["keys"].([]any)
	for _, k := range keys {
		m, _ := k.(map[string]any)
		if m["status"] != "active" {
			continue
		}
		switch {
		case m["publicKey"] == uploadPub && m["purpose"] == "upload":
			uploadID, _ = m["id"].(string)
		case serverPub != "" && m["publicKey"] == serverPub && m["purpose"] == "server":
			serverID, _ = m["id"].(string)
		}
	}
	return uploadID, serverID, uploadID != "" && (serverPub == "" || serverID != "")
}

func cmdRegister(args []string) error {
	fs := flag.NewFlagSet("register", flag.ExitOnError)
	api := fs.String("api", "", "平台網址（或 PT_API_URL；預設讀 playtogether.json 的 platform）")
	title := fs.String("title", "", "遊戲名稱（預設用 store/listing.json 的繁中名稱，沒有時用代號）")
	_ = fs.Parse(reorder(args))
	p, err := findProject()
	if err != nil {
		return err
	}
	if p == nil {
		return errors.New("找不到 playtogether.json：請在用 pt new 建立的專案裡執行")
	}
	if p.UploadKeyID != "" {
		return fmt.Errorf("這個專案已經登記過了（uploadKeyId %s）。要換金鑰請到開發者頁的「金鑰」新增後再撤銷舊的", p.UploadKeyID)
	}
	base := platformBase(p, *api)
	if err := securePlatform(base); err != nil {
		return err
	}
	c, ok := loadCredentials()[registerKey(base)]
	if !ok || c.Token == "" || !time.Now().Before(c.ExpiresAt) {
		return errors.New("需要建立遊戲的授權：先執行 pt login -register（團隊擁有者或管理者，在瀏覽器同意）")
	}

	name := strings.TrimSpace(*title)
	if name == "" {
		var l storeListing
		if raw, err := os.ReadFile(p.path(filepath.Join("store", "listing.json"))); err == nil && json.Unmarshal(raw, &l) == nil && !storePlaceholder(l.Name.ZhTW) {
			name = strings.TrimSpace(l.Name.ZhTW)
		}
	}
	if name == "" {
		name = p.Name
	}

	upKey, upNew, err := ensureKey(p.path(p.UploadKeyFile))
	if err != nil {
		return err
	}
	if upNew {
		fmt.Println("已產生上傳金鑰：", p.UploadKeyFile, "（權限 600，不要提交到版本控制）")
	}
	uploadPub, serverPub := pubB64(upKey), ""
	if p.gameServerMode() != "none" {
		srvKey, srvNew, err := ensureKey(p.path(p.ServerKeyFile))
		if err != nil {
			return err
		}
		if srvNew {
			fmt.Println("已產生遊戲伺服器金鑰：", p.ServerKeyFile, "（只放在遊戲伺服器上）")
		}
		serverPub = pubB64(srvKey)
	}

	body := map[string]string{"id": p.Name, "name": name, "uploadPublicKey": uploadPub}
	if serverPub != "" {
		body["serverPublicKey"] = serverPub
	}
	st, out, err := devPost(base+"/v1/dev/games", body, map[string]string{"Authorization": "Bearer " + c.Token})
	uploadID, serverID := "", ""
	switch {
	case err != nil:
		// 網路中斷：平台可能已經建立了，查一次
		var ok bool
		if uploadID, serverID, ok = recoverRegistration(base, c.Token, p.Name, uploadPub, serverPub); !ok {
			return fmt.Errorf("連不到平台：%w（重新執行 pt register 即可，已建立的會自動接手）", err)
		}
	case st == http.StatusCreated:
		uploadID, _ = out["uploadKeyId"].(string)
		serverID, _ = out["serverKeyId"].(string)
		// 寫檔前確認回應完整：缺金鑰 ID 時不要寫進 playtogether.json（之後 pt register 會因為已登記而停下來）
		if out["gameId"] != p.Name || uploadID == "" || (serverPub != "" && serverID == "") {
			var ok bool
			if uploadID, serverID, ok = recoverRegistration(base, c.Token, p.Name, uploadPub, serverPub); !ok {
				return fmt.Errorf("平台的回應不完整（%v），沒有寫入 playtogether.json；請重新執行 pt register", out)
			}
		}
	case out["error"] == "id_unavailable":
		var ok bool
		if uploadID, serverID, ok = recoverRegistration(base, c.Token, p.Name, uploadPub, serverPub); !ok {
			return errors.New(registerErrors["id_unavailable"])
		}
		fmt.Println("這款遊戲之前已經建立（上次沒有收到回應），接手原本的金鑰。")
	default:
		code, _ := out["error"].(string)
		if msg, ok := registerErrors[code]; ok {
			return errors.New(msg)
		}
		return fmt.Errorf("平台回應 %d %v", st, out["error"])
	}

	p.UploadKeyID, p.ServerKeyID = uploadID, serverID
	if p.Platform == "" || strings.Contains(p.Platform, "（") {
		p.Platform = base
	}
	if err := p.save(); err != nil {
		return fmt.Errorf("寫回 playtogether.json 失敗：%w（重新執行 pt register 會查回金鑰 ID）", err)
	}
	fmt.Printf("已建立草稿遊戲 %s（%s），團隊 %s。\n", p.Name, name, c.TeamName)
	fmt.Printf("  上傳金鑰 ID：%s\n", uploadID)
	if serverID != "" {
		fmt.Printf("  遊戲伺服器金鑰 ID：%s（部署遊戲伺服器時設成 PT_KEY_ID，私鑰在 %s）\n", serverID, p.ServerKeyFile)
	}
	fmt.Println("已寫進 playtogether.json。下一步：pt upload -version 0.1.0（草稿遊戲不會出現在列表，上架要送審）")
	return nil
}
