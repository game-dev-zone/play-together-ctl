package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// projectPolicy 讀出專案的宣告（測試用；有任何錯誤就失敗）。
func projectPolicy(t *testing.T, p project) policy {
	t.Helper()
	pl, _, ok := readPolicy(&p, func(level, format string, a ...any) {
		if isErrorLevel(level) {
			t.Fatalf("宣告有錯誤：%s", format)
		}
	})
	if !ok {
		t.Fatal("專案沒有宣告")
	}
	return pl
}

// projectFrom 把宣告寫成 playtogether.json 的五個區塊（不含 dir，不檢查文件）。
func projectFrom(pl policy) *project {
	raw := func(v any) json.RawMessage { b, _ := json.Marshal(v); return b }
	return &project{PlatformAudience: raw(pl.Audience), ContentProfile: raw(pl.Content), SensitiveContent: raw(pl.Sensitive), Gameplay: raw(pl.Gameplay), Economy: raw(pl.Economy)}
}

type levels map[string][]string

func runPolicy(p *project) levels {
	out := levels{}
	checkPolicy(p, func(level, format string, a ...any) { out[level] = append(out[level], format) })
	return out
}

// 結果等級：hard、schema、review 是否出現（warn、info 不影響）。
func (l levels) only(want ...string) bool {
	for _, lv := range []string{levelHard, levelSchema, levelBuild, levelReview} {
		if (len(l[lv]) > 0) != slices.Contains(want, lv) {
			return false
		}
	}
	return true
}

// 新專案：-profile 三種 preset、-rating 舊寫法、預設 general；產生的宣告與文件通過 pt check（PASS）。
func TestPolicyProfiles(t *testing.T) {
	requireEmbeddedSDK(t)
	dir := t.TempDir()
	t.Chdir(dir)
	cases := map[string][]string{
		"general": {"-y"}, "moderate": {"-y", "-profile", "moderate"}, "strong": {"-y", "-profile", "strong"},
	}
	for want, args := range cases {
		name := "p-" + want
		must(t, cmdNew(append(args, name)))
		p := readProject(t, filepath.Join(dir, name))
		p.dir = filepath.Join(dir, name)
		pl := projectPolicy(t, p)
		if pl.Content != presetProfile(want) || pl.Audience.MinimumUserAge != 18 || pl.Gameplay.TableGame.Enabled || pl.Economy != (economyPolicy{}) || pl.Sensitive != (sensitiveContent{}) {
			t.Fatalf("%s 的宣告：%+v", want, pl)
		}
		if l := runPolicy(&p); !l.only() || len(l[levelWarn]) != 0 {
			t.Fatalf("%s 的新專案應該 PASS：%v", want, l)
		}
		doc := readFile(t, dir, name, "docs/content/CONTENT_PROFILE.md")
		for _, s := range []string{"不是正式分級", "玩聚平台審核決定", "上限，不是要求", "CONTENT_PROFILE_CHANGE_REQUEST", "AI 不能自行提高內容邊界",
			"18+（玩聚的使用者資格，不是遊戲內容分級）", "描寫仇恨 ≠ 鼓吹仇恨", "不得**性化未成年人", profileLabel[want]} {
			if !strings.Contains(doc, s) {
				t.Errorf("%s 的 CONTENT_PROFILE.md 少了 %q", want, s)
			}
		}
		ge := readFile(t, dir, name, "docs/content/GAMEPLAY_ECONOMY.md")
		for _, s := range []string{"Table Game ≠ Real-Money Gambling", "`paidRandomReward`", "平台硬性規則", "技術上遊戲仍然不能自己接付款", "沒有需要平台特別審核"} {
			if !strings.Contains(ge, s) {
				t.Errorf("%s 的 GAMEPLAY_ECONOMY.md 少了 %q", want, s)
			}
		}
		if _, err := os.Stat(filepath.Join(dir, name, "docs/content/CONTENT_RATING.md")); err == nil {
			t.Errorf("%s 不該再產生 CONTENT_RATING.md", want)
		}
	}
	must(t, cmdNew([]string{"p-legacy", "-y", "-rating", "15plus"}))
	if pl := projectPolicy(t, readProject(t, filepath.Join(dir, "p-legacy"))); pl.Content.Profile != "strong" {
		t.Fatalf("-rating 15plus 要對應成 strong：%+v", pl.Content)
	}
	for _, bad := range [][]string{{"x1", "-y", "-profile", "custom"}, {"x2", "-y", "-profile", "adult"}, {"x3", "-y", "-profile", "general", "-rating", "general"},
		{"x4", "-y", "-table-game", "casino"}, {"x5", "-y", "-rating", "18plus"}} {
		if err := cmdNew(bad); err == nil {
			t.Errorf("%v 要回錯誤", bad)
		}
		if _, err := os.Stat(filepath.Join(dir, bad[0])); err == nil {
			t.Errorf("%v 失敗時不能留下資料夾", bad)
		}
	}
	pt := readFile(t, dir, "p-general", "PLAYTOGETHER.md")
	for _, s := range []string{"玩聚平台審核才是最終准駁者", "Table Game ≠ Real-Money Gambling", "描寫仇恨 ≠ 鼓吹仇恨", "需要平台審核 ≠ 已批准",
		"不得性化未成年人", "可辨識的真人", "不得鼓吹、煽動或合理化", "`-skip-check` 不能略過", "docs/content/CONTENT_PROFILE.md", "docs/content/GAMEPLAY_ECONOMY.md"} {
		if !strings.Contains(pt, s) {
			t.Errorf("PLAYTOGETHER.md 少了 %q", s)
		}
	}
	for _, banned := range []string{"文案不要出現賭博、博弈字眼", "不搭配籌碼", "模擬賭博", "CONTENT_RATING.md", "15+ 方向", "poker = ", "15plus"} {
		if strings.Contains(pt, banned) {
			t.Errorf("PLAYTOGETHER.md 不該再有 %q", banned)
		}
	}
	wf := readFile(t, dir, "p-general", "docs/ai/AI_WORKFLOW.md")
	for _, s := range []string{"Content Profile Authority", "CONTENT_PROFILE_CHANGE_REQUEST", "GAMEPLAY_POLICY_CHANGE_REQUEST", "不自動提高 profile",
		"不得靜默放寬", "不宣稱已批准", "不得為了通過檢查而修改宣告", "Table Game ≠ Real-Money Gambling", "描寫仇恨 ≠ 鼓吹仇恨", "不要因為涉及政治迫害或歧視就拒絕開發"} {
		if !strings.Contains(wf, s) {
			t.Errorf("AI_WORKFLOW.md 少了 %q", s)
		}
	}
	ctx := readFile(t, dir, "p-general", "docs/ai/PROJECT_CONTEXT.md")
	for _, s := range []string{"Platform Audience: 18+", "Content Profile: general", "Table Game: none", "pt Validation: PASS", "Final decision: PlayTogether Platform Review"} {
		if !strings.Contains(ctx, s) {
			t.Errorf("PROJECT_CONTEXT.md 少了 %q", s)
		}
	}
}

// -table-game：建立與開發都支援；撲克、麻將、吃角子老虎等是 REVIEW_REQUIRED（不是錯誤，也不寫死分級）；卡牌、骰子、一般桌遊（計分 token）是 PASS。
func TestPolicyTableGameNew(t *testing.T) {
	requireEmbeddedSDK(t)
	dir := t.TempDir()
	t.Chdir(dir)
	for cat, review := range map[string]bool{"poker": true, "mahjong": true, "slot": true, "roulette": true, "baccarat": true, "other": true, "board": false, "card": false, "dice": false} {
		name := "t-" + cat
		must(t, cmdNew([]string{name, "-y", "-table-game", cat}))
		p := readProject(t, filepath.Join(dir, name))
		p.dir = filepath.Join(dir, name)
		l := runPolicy(&p)
		want := []string{}
		if review {
			want = []string{levelReview}
		}
		if !l.only(want...) || len(l[levelWarn]) != 0 {
			t.Errorf("%s：%v", cat, l)
		}
		ctx := readFile(t, dir, name, "docs/ai/PROJECT_CONTEXT.md")
		if !strings.Contains(ctx, "Table Game: "+cat) || strings.Contains(ctx, "15plus") {
			t.Errorf("%s 的 PROJECT_CONTEXT.md：%s", cat, ctx)
		}
		ge := readFile(t, dir, name, "docs/content/GAMEPLAY_ECONOMY.md")
		if review != strings.Contains(ge, "需要玩聚平台人工審核（可以送審，**不代表已批准**）") {
			t.Errorf("%s 的 GAMEPLAY_ECONOMY.md 檢查結果不對", cat)
		}
	}
}

// 驗證矩陣：硬性規則、宣告錯誤、需要審核、通過。
func TestCheckPolicyMatrix(t *testing.T) {
	base := func() policy { return defaultPolicy(presetProfile("general"), "none") }
	table := func(cat string) func(*policy) {
		return func(pl *policy) { pl.Gameplay.TableGame = tablePreset(cat) }
	}
	cases := []struct {
		name string
		mut  func(*policy)
		want []string
	}{
		{"一般益智遊戲", func(pl *policy) {}, nil},
		// 經濟硬性規則
		{"realMoneyWagering", func(pl *policy) { pl.Economy.RealMoneyWagering = true }, []string{levelHard}},
		{"purchasableStake", func(pl *policy) { table("poker")(pl); pl.Economy.PurchasableStake = true }, []string{levelHard, levelReview}},
		{"withdrawableValue", func(pl *policy) { pl.Economy.WithdrawableValue = true }, []string{levelHard}},
		{"redeemableValue", func(pl *policy) { pl.Economy.RedeemableValue = true }, []string{levelHard}},
		{"transferableRealWorldValue", func(pl *policy) { pl.Economy.TransferableRealWorldValue = true }, []string{levelHard}},
		{"paidRandomReward", func(pl *policy) { pl.Economy.PaidRandomReward = true }, []string{levelHard}},
		{"realMoney 但不是下注", func(pl *policy) { pl.Economy.RealMoney = true }, []string{levelReview}},
		// 經平台專用接口使用點數：由玩聚平台與律師決定（需要審核，不是硬性規則）
		{"platformPointsStake", func(pl *policy) { v := true; pl.Economy.PlatformPointsStake = &v }, []string{levelReview}},
		{"platformPointsRandom", func(pl *policy) { v := true; pl.Economy.PlatformPointsRandom = &v }, []string{levelReview}},
		{"platformPointsStake false", func(pl *policy) { v := false; pl.Economy.PlatformPointsStake = &v }, nil},
		// 內容
		{"custom＋露骨性內容", func(pl *policy) { pl.Content.Profile = "custom"; pl.Content.SexualContent = "explicit" }, []string{levelHard}},
		{"custom＋大量血腥", func(pl *policy) { pl.Content.Profile = "custom"; pl.Content.Blood = "strong" }, []string{levelHard}},
		{"general＋blood mild", func(pl *policy) { pl.Content.Blood = "mild" }, []string{levelSchema}},
		{"preset 被手改", func(pl *policy) { pl.Content = presetProfile("strong"); pl.Content.Tobacco = "none" }, []string{levelSchema}},
		{"custom＋合法的值", func(pl *policy) {
			pl.Content.Profile = "custom"
			pl.Content.Blood = "mild"
			pl.Content.Alcohol = "allowed"
		}, nil},
		{"不合法的 profile", func(pl *policy) { pl.Content.Profile = "15plus" }, []string{levelSchema}},
		{"不合法的維度值", func(pl *policy) { pl.Content.Profile = "custom"; pl.Content.Violence = "extreme" }, []string{levelSchema}},
		{"provisional false", func(pl *policy) { pl.Content.Provisional = false }, []string{levelSchema}},
		{"minimumUserAge 15", func(pl *policy) { pl.Audience.MinimumUserAge = 15 }, []string{levelSchema}},
		// 敏感題材
		{"歷史迫害", func(pl *policy) { pl.Sensitive.HistoricalPersecution = true }, []string{levelReview}},
		{"政治壓迫", func(pl *policy) { pl.Sensitive.PoliticalRepression = true }, []string{levelReview}},
		{"反派台詞", func(pl *policy) { pl.Sensitive.HateSpeechDepiction = true }, []string{levelReview}},
		{"批判性呈現歧視", func(pl *policy) { pl.Sensitive.DiscriminationDepiction = true }, []string{levelReview}},
		{"鼓吹仇恨", func(pl *policy) { pl.Sensitive.HateAdvocacy = true }, []string{levelHard}},
		{"鼓勵傷害受保護群體", func(pl *policy) { pl.Sensitive.EncourageProtectedClassViolence = true }, []string{levelHard}},
		{"返校類歷史恐怖", func(pl *policy) {
			pl.Content = presetProfile("strong")
			pl.Sensitive = sensitiveContent{HistoricalPersecution: true, PoliticalRepression: true, DiscriminationDepiction: true, HateSpeechDepiction: true}
		}, []string{levelReview}},
		{"歷史題材＋輕度呈現", func(pl *policy) { pl.Sensitive.HistoricalPersecution = true }, []string{levelReview}},
		// 牌桌
		{"純娛樂撲克", table("poker"), []string{levelReview}},
		{"麻將", table("mahjong"), []string{levelReview}},
		{"吃角子老虎", table("slot"), []string{levelReview}},
		{"輪盤", table("roulette"), []string{levelReview}},
		{"卡牌", table("card"), nil},
		{"骰子桌遊", table("dice"), nil},
		{"賭場風格", func(pl *policy) { table("card")(pl); pl.Gameplay.TableGame.Presentation = "casinoLike" }, []string{levelReview}},
		{"跨局免費籌碼", func(pl *policy) { table("poker")(pl); pl.Gameplay.TableGame.TokenPersistence = "persistent" }, []string{levelReview}},
		{"每日與看廣告送籌碼", func(pl *policy) {
			table("poker")(pl)
			pl.Gameplay.TableGame.FreeTokenSources = []string{"daily", "adReward"}
		}, []string{levelReview}},
		{"牌桌啟用但沒有類別", func(pl *policy) { pl.Gameplay.TableGame.Enabled = true }, []string{levelSchema}},
		{"沒有牌桌卻有計分 token", func(pl *policy) { pl.Gameplay.TableGame.ScoreTokens = "allowed" }, []string{levelSchema}},
		{"一般桌遊＋計分 token", table("board"), nil},
		{"桌遊計分 token 不需要保存範圍", func(pl *policy) { table("board")(pl); pl.Gameplay.TableGame.Cards = "allowed" }, nil},
		{"撲克＋虛擬下注＋下注籌碼", table("poker"), []string{levelReview}},
		{"只有下注籌碼沒有下注動作", func(pl *policy) {
			table("card")(pl)
			pl.Gameplay.TableGame.WageringTokens = "allowed"
			pl.Gameplay.TableGame.TokenPersistence = "round"
		}, []string{levelReview}},
		{"計分 token 不能拿來下注", func(pl *policy) {
			table("board")(pl)
			pl.Gameplay.TableGame.VirtualWagering = "allowed"
		}, []string{levelSchema, levelReview}},
		{"沒有下注籌碼卻有保存範圍", func(pl *policy) { table("board")(pl); pl.Gameplay.TableGame.TokenPersistence = "persistent" }, []string{levelSchema, levelReview}},
		{"虛擬下注沒有籌碼", func(pl *policy) { table("card")(pl); pl.Gameplay.TableGame.VirtualWagering = "allowed" }, []string{levelSchema, levelReview}},
		{"籌碼沒有保存範圍", func(pl *policy) { table("poker")(pl); pl.Gameplay.TableGame.TokenPersistence = "none" }, []string{levelSchema, levelReview}},
		{"不合法的類別", func(pl *policy) { table("card")(pl); pl.Gameplay.TableGame.Category = "casino" }, []string{levelSchema}},
		{"重複的籌碼來源", func(pl *policy) {
			table("poker")(pl)
			pl.Gameplay.TableGame.FreeTokenSources = []string{"achievement", "achievement"}
		}, []string{levelSchema, levelReview}},
	}
	for _, c := range cases {
		pl := base()
		c.mut(&pl)
		if l := runPolicy(projectFrom(pl)); !l.only(c.want...) {
			t.Errorf("%s：要 %v，結果 %v", c.name, c.want, l)
		}
	}
	// 類別和能力欄位不搭：只提醒
	pl := base()
	table("poker")(&pl)
	pl.Gameplay.TableGame.Cards = "none"
	if l := runPolicy(projectFrom(pl)); !l.only(levelReview) || len(l[levelWarn]) != 1 {
		t.Errorf("poker 卻沒有牌只提醒：%v", l)
	}
}

// 嚴格解析：缺欄位、未知鍵、型別錯誤、缺整個區塊都是宣告錯誤。
func TestCheckPolicyStrict(t *testing.T) {
	good := projectFrom(defaultPolicy(presetProfile("general"), "none"))
	if l := runPolicy(good); !l.only() {
		t.Fatalf("合法的宣告：%v", l)
	}
	mutate := func(field string, f func(m map[string]any)) *project {
		p := *good
		var raw *json.RawMessage
		switch field {
		case "economy":
			raw = &p.Economy
		case "gameplay":
			raw = &p.Gameplay
		case "sensitiveContent":
			raw = &p.SensitiveContent
		}
		m := map[string]any{}
		_ = json.Unmarshal(*raw, &m)
		f(m)
		b, _ := json.Marshal(m)
		*raw = b
		return &p
	}
	cases := map[string]*project{
		"拼錯的鍵":         mutate("economy", func(m map[string]any) { delete(m, "paidRandomReward"); m["paidRandomRewards"] = true }),
		"缺欄位":          mutate("economy", func(m map[string]any) { delete(m, "realMoneyWagering") }),
		"型別錯誤":         mutate("economy", func(m map[string]any) { m["realMoney"] = "no" }),
		"敏感題材缺欄位":      mutate("sensitiveContent", func(m map[string]any) { delete(m, "hateAdvocacy") }),
		"tableGame 拼錯": mutate("gameplay", func(m map[string]any) { m["tableGame"].(map[string]any)["virtualWager"] = "none" }),
		"選填欄位是 null":   mutate("economy", func(m map[string]any) { m["platformPointsStake"] = nil }),
		"選填欄位拼錯":       mutate("economy", func(m map[string]any) { m["platformPointStake"] = true }),
	}
	for name, p := range cases {
		if l := runPolicy(p); len(l[levelSchema]) == 0 {
			t.Errorf("%s 要是宣告錯誤：%v", name, l)
		}
	}
	// 選填欄位：寫了 true 是需要審核，不是宣告錯誤
	if l := runPolicy(mutate("economy", func(m map[string]any) { m["platformPointsStake"] = true })); len(l[levelSchema]) != 0 || len(l[levelReview]) == 0 {
		t.Errorf("platformPointsStake 是選填、需要審核：%v", l)
	}
	p := *good
	p.Economy = nil
	if l := runPolicy(&p); len(l[levelSchema]) == 0 {
		t.Error("v0.15 宣告缺整個 economy 要是宣告錯誤")
	}
	// null 不能當成 false；重複的鍵不能讓後面的值蓋掉前面的
	for name, raw := range map[string]string{
		"null": `{"realMoney":false,"realMoneyWagering":false,"purchasableStake":false,"withdrawableValue":false,"redeemableValue":false,"transferableRealWorldValue":false,"paidRandomReward":null}`,
		"重複的鍵": `{"realMoney":false,"realMoneyWagering":true,"realMoneyWagering":false,"purchasableStake":false,"withdrawableValue":false,"redeemableValue":false,"transferableRealWorldValue":false,"paidRandomReward":false}`,
	} {
		p := *good
		p.Economy = json.RawMessage(raw)
		if l := runPolicy(&p); len(l[levelSchema]) == 0 {
			t.Errorf("%s 要是宣告錯誤：%v", name, l)
		}
	}
	p = *good
	p.Gameplay = json.RawMessage(`{"chanceMechanics":"allowed","tableGame":null}`)
	if l := runPolicy(&p); len(l[levelSchema]) == 0 {
		t.Errorf("tableGame 是 null 要是宣告錯誤：%v", l)
	}
}

// playtogether.json 最外層：拼錯的宣告鍵是宣告錯誤（不能讓整組宣告被當成不存在而只剩提醒）；重複的鍵是宣告錯誤；其他不認得的鍵只提醒。
func TestCheckTopLevelKeys(t *testing.T) {
	run := func(raw string) levels {
		var p project
		must(t, json.Unmarshal([]byte(raw), &p))
		p.raw = []byte(raw)
		return runPolicy(&p)
	}
	if l := run(`{"name":"x","econmy":{"realMoneyWagering":true}}`); len(l[levelSchema]) == 0 {
		t.Errorf("拼錯的宣告鍵要是宣告錯誤：%v", l)
	}
	if l := run(`{"name":"x","contentRating":{},"contentRating":{}}`); len(l[levelSchema]) == 0 {
		t.Errorf("重複的鍵要是宣告錯誤：%v", l)
	}
	if l := run(`{"name":"x","myNote":"hi","cdn":[],"orientation":"portrait"}`); !l.only() || len(l[levelWarn]) != 2 {
		t.Errorf("不認得的一般欄位只提醒（加上沒有宣告的提醒）：%v", l)
	}
}

// 舊格式（v0.14 的 contentRating）：預設的專案只有提醒；舊的 wagering=false 不解讀成禁止虛擬下注；明確的真錢下注、付費抽獎仍是硬性規則。
func TestCheckPolicyLegacy(t *testing.T) {
	legacy := func(f func(r *contentRating)) *project {
		r := ratingPreset("15plus")
		f(&r)
		return &project{ContentRating: &r}
	}
	for _, name := range []string{"general", "12plus", "15plus"} {
		r := ratingPreset(name)
		if l := runPolicy(&project{ContentRating: &r}); !l.only() || len(l[levelWarn]) == 0 {
			t.Errorf("v0.14 的 %s 應該只有提醒：%v", name, l)
		}
	}
	cases := []struct {
		name string
		p    *project
		want []string
	}{
		{"wagering=true", legacy(func(r *contentRating) { r.Gambling.Wagering = true }), []string{levelReview}},
		{"simulated=allowed", legacy(func(r *contentRating) { r.Gambling.Simulated = "allowed" }), []string{levelReview}},
		{"realMoney=true", legacy(func(r *contentRating) { r.Gambling.RealMoney = true }), []string{levelHard}},
		{"paidRandomReward=true", legacy(func(r *contentRating) { r.Gambling.PaidRandomReward = true }), []string{levelHard}},
		{"超過平台上限的血腥", legacy(func(r *contentRating) { r.Blood = "strong" }), []string{levelHard}},
		{"超過目標年齡", legacy(func(r *contentRating) { r.Profile, r.TargetAge = "custom", "12+" }), []string{levelSchema}},
		{"缺 simulated", legacy(func(r *contentRating) { r.Gambling.Simulated = "" }), nil},
	}
	for _, c := range cases {
		if l := runPolicy(c.p); !l.only(c.want...) {
			t.Errorf("%s：要 %v，結果 %v", c.name, c.want, l)
		}
	}
	// 新舊並存：以新格式為準並提醒
	p := projectFrom(defaultPolicy(presetProfile("general"), "none"))
	r := ratingPreset("15plus")
	r.Gambling.RealMoney = true // 舊欄位被忽略
	p.ContentRating = &r
	if l := runPolicy(p); !l.only() || len(l[levelWarn]) != 1 {
		t.Errorf("新舊並存要以新格式為準：%v", l)
	}
	// 沒有任何宣告（v0.14 之前）：只提醒
	if l := runPolicy(&project{}); !l.only() || len(l[levelWarn]) != 1 {
		t.Errorf("沒有宣告只提醒：%v", l)
	}
}

// 規則資料：欄位都存在、等級只有 hard／review、不寫分級或准駁；平台不收的內容值都有硬性規則。
func TestPolicyRulesData(t *testing.T) {
	pl := defaultPolicy(presetProfile("general"), "none")
	fields := pl.fields()
	rules := loadPolicyRules()
	if rules.Version == "" || len(rules.Rules) == 0 {
		t.Fatal("規則資料是空的")
	}
	ids := map[string]bool{}
	for _, r := range rules.Rules {
		if _, ok := fields[r.Field]; !ok {
			t.Errorf("%s 的欄位 %s 不存在", r.ID, r.Field)
		}
		if r.Level != levelHard && r.Level != levelReview {
			t.Errorf("%s 的等級只能是 hard 或 review：%s", r.ID, r.Level)
		}
		if ids[r.ID] {
			t.Errorf("規則 ID 重複：%s", r.ID)
		}
		ids[r.ID] = true
		for _, banned := range []string{"15+", "15plus", "輔15", "prohibited", "已批准"} {
			if strings.Contains(r.Reason, banned) {
				t.Errorf("%s 的原因不能寫分級或准駁：%q", r.ID, r.Reason)
			}
		}
	}
	for _, d := range profileDims {
		for _, v := range d.beyond {
			c := presetProfile("custom")
			*d.val(&c) = v
			if !slices.ContainsFunc(matchRules(defaultPolicy(c, "none")), func(r policyRule) bool { return r.Level == levelHard }) {
				t.Errorf("%s=%s 是平台不收的值，要有硬性規則", d.key, v)
			}
		}
	}
	for _, id := range []string{"real-money-wagering", "purchasable-stake", "withdrawable-value", "redeemable-value", "transferable-real-world-value", "paid-random-reward", "hate-advocacy"} {
		if !ids[id] {
			t.Errorf("少了硬性規則 %s", id)
		}
	}
}

// pt 不做關鍵字偵測：原始碼裡的「消滅」「叛徒」「下注」等字詞不產生任何結果。
func TestNoKeywordDetection(t *testing.T) {
	dir := t.TempDir()
	must(t, os.WriteFile(filepath.Join(dir, "index.html"), []byte(`<script src="playtogether-sdk.iife.js"></script>`), 0o644))
	must(t, os.WriteFile(filepath.Join(dir, "story.js"), []byte(`const lines = ["史料記載：他們被稱為叛徒，政權要消滅敵人", "下注", "All-in", "Raise"];`), 0o644))
	for _, f := range checkProject(nil, dir) {
		if f.level != levelInfo {
			t.Errorf("不應產生結果：%+v", f)
		}
	}
}

// 平台與安全架構硬性規則：私鑰字串、付款、麥克風。
func TestSecurityHardRules(t *testing.T) {
	for name, src := range map[string]string{
		"私鑰":   "const k = `-----BEGIN PRIVATE KEY-----\nMC4C\n-----END PRIVATE KEY-----`;",
		"付款":   "new PaymentRequest([], {});",
		"金流":   `import("https://js.stripe.com/v3/")`,
		"麥克風":  "navigator.mediaDevices.getUserMedia({audio: true});",
		"語音辨識": "const rec = new (window.SpeechRecognition || window.webkitSpeechRecognition)();",
	} {
		dir := t.TempDir()
		must(t, os.WriteFile(filepath.Join(dir, "index.html"), []byte(`<script src="playtogether-sdk.iife.js"></script>`), 0o644))
		must(t, os.WriteFile(filepath.Join(dir, "game.js"), []byte(src), 0o644))
		if !hasFinding(checkProject(nil, dir), levelHard, "game.js") {
			t.Errorf("%s 要是硬性規則", name)
		}
	}
}

// pt upload：硬性規則與宣告錯誤加 -skip-check 也擋下（訊息不提這個旗標）；只剩建置錯誤時 -skip-check 可以繼續；需要審核的項目可以送審。
func TestUploadSkipCheckSemantics(t *testing.T) {
	requireEmbeddedSDK(t)
	uploads := 0
	platform := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "POST" && r.URL.Path == "/v1/dev/builds":
			uploads++
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"buildId":"b1","uploadPath":"/v1/dev/uploads/tok"}`))
		case r.Method == "PUT":
			_, _ = w.Write([]byte(`{"artifactStatus":"processing"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(platform.Close)
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	keyFile := filepath.Join(t.TempDir(), "upload.key")
	must(t, os.WriteFile(keyFile, []byte(base64.StdEncoding.EncodeToString(priv.Seed())+"\n"), 0o600))
	dir := t.TempDir()
	t.Chdir(dir)
	must(t, cmdNew([]string{"skipper", "-y", "-platform", platform.URL}))
	root := filepath.Join(dir, "skipper")
	t.Chdir(root)
	orig, err := os.ReadFile(filepath.Join(root, projectFile))
	must(t, err)
	setJSON := func(f func(m map[string]any)) {
		m := map[string]any{}
		must(t, json.Unmarshal(orig, &m))
		f(m)
		b, _ := json.MarshalIndent(m, "", "  ")
		must(t, os.WriteFile(filepath.Join(root, projectFile), b, 0o644))
	}
	up := func(extra ...string) error {
		return upload(append([]string{"-version", "1.0.0", "-api", platform.URL, "-key-id", "k1", "-key-file", keyFile, "-wait", "0"}, extra...))
	}
	// 新專案沒有填 serverUrl：只有建置錯誤
	if err := up(); err == nil || !strings.Contains(err.Error(), "-skip-check") {
		t.Fatalf("只有建置錯誤時要提示 -skip-check：%v", err)
	}
	if err := up("-skip-check"); err != nil || uploads != 1 {
		t.Fatalf("只有建置錯誤時 -skip-check 可以繼續：%v", err)
	}
	// 硬性規則：-skip-check 也擋下
	setJSON(func(m map[string]any) { m["economy"].(map[string]any)["realMoneyWagering"] = true })
	if err := up("-skip-check"); err == nil || strings.Contains(err.Error(), "-skip-check") || uploads != 1 {
		t.Fatalf("硬性規則要擋下、訊息不提 -skip-check：%v", err)
	}
	// 宣告錯誤：-skip-check 也擋下
	setJSON(func(m map[string]any) { m["contentProfile"].(map[string]any)["blood"] = "mild" })
	if err := up("-skip-check"); err == nil || strings.Contains(err.Error(), "-skip-check") || uploads != 1 {
		t.Fatalf("宣告錯誤要擋下：%v", err)
	}
	// 改了宣告沒同步文件：宣告錯誤，-skip-check 也擋下
	setJSON(func(m map[string]any) {
		g := m["gameplay"].(map[string]any)
		raw, _ := json.Marshal(tablePreset("poker"))
		var tg map[string]any
		_ = json.Unmarshal(raw, &tg)
		g["tableGame"] = tg
	})
	if err := up("-skip-check"); err == nil || strings.Contains(err.Error(), "-skip-check") || uploads != 1 {
		t.Fatalf("文件和宣告不一致要擋下：%v", err)
	}
	// pt content sync 之後：需要審核的項目可以送審
	must(t, cmdContent([]string{"sync"}))
	if err := up("-skip-check"); err != nil || uploads != 2 {
		t.Fatalf("需要審核的項目可以送審：%v", err)
	}
}

// v0.15 的開發契約文件：缺少或和宣告不一致是宣告錯誤；pt content sync 重新產生並記錄平台政策版本；
// 宣告有錯誤時不產生文件；policyVersion 缺少或不同只提醒。v0.14 的舊文件不一致仍只提醒。
func TestPolicyDocsAndSync(t *testing.T) {
	requireEmbeddedSDK(t)
	dir := t.TempDir()
	t.Chdir(dir)
	must(t, cmdNew([]string{"docs-game", "-y"}))
	root := filepath.Join(dir, "docs-game")
	t.Chdir(root)
	check := func() levels {
		t.Helper()
		p, err := findProject()
		must(t, err)
		return runPolicy(p)
	}
	if l := check(); !l.only() || len(l[levelWarn]) != 0 {
		t.Fatalf("新專案要 PASS：%v", l)
	}
	if p, _ := findProject(); p.PolicyVersion != loadPolicyRules().Version {
		t.Fatalf("新專案要記錄平台政策版本：%q", p.PolicyVersion)
	}
	profile := filepath.Join(root, "docs/content/CONTENT_PROFILE.md")
	must(t, os.Remove(profile))
	if l := check(); len(l[levelSchema]) != 1 {
		t.Fatalf("文件缺少要是宣告錯誤：%v", l)
	}
	// JSON 改成 strong，文件仍是 general：宣告錯誤
	must(t, cmdContent([]string{"sync"}))
	raw, _ := os.ReadFile(filepath.Join(root, projectFile))
	m := map[string]any{}
	must(t, json.Unmarshal(raw, &m))
	cpRaw, _ := json.Marshal(presetProfile("strong"))
	var cp map[string]any
	_ = json.Unmarshal(cpRaw, &cp)
	m["contentProfile"] = cp
	write := func() {
		b, _ := json.MarshalIndent(m, "", "  ")
		must(t, os.WriteFile(filepath.Join(root, projectFile), b, 0o644))
	}
	write()
	if l := check(); len(l[levelSchema]) != 1 || !strings.Contains(l[levelSchema][0], "不一致") {
		t.Fatalf("文件和宣告不一致要是宣告錯誤：%v", l)
	}
	must(t, cmdContent([]string{"sync"}))
	if l := check(); !l.only() {
		t.Fatalf("sync 之後要 PASS：%v", l)
	}
	if doc, _ := os.ReadFile(profile); !strings.Contains(string(doc), "Strong") {
		t.Fatal("sync 要依新宣告重新產生文件")
	}
	// 宣告有錯誤時不產生文件
	m["contentProfile"].(map[string]any)["blood"] = "mild"
	write()
	before, _ := os.ReadFile(profile)
	if err := cmdContent([]string{"sync"}); err == nil {
		t.Fatal("宣告有錯誤時 sync 要失敗")
	}
	if after, _ := os.ReadFile(profile); string(after) != string(before) {
		t.Fatal("宣告有錯誤時不能改寫文件")
	}
	// policyVersion：缺少或和這個 pt 不同只提醒
	m["contentProfile"].(map[string]any)["blood"] = "moderate"
	for _, v := range []any{nil, "2000-01-01"} {
		if v == nil {
			delete(m, "policyVersion")
		} else {
			m["policyVersion"] = v
		}
		write()
		if l := check(); !l.only() || len(l[levelWarn]) != 1 || !strings.Contains(l[levelWarn][0], "pt content sync") {
			t.Fatalf("policyVersion %v 只提醒：%v", v, l)
		}
	}
	must(t, cmdContent([]string{"sync"}))
	if p, _ := findProject(); p.PolicyVersion != loadPolicyRules().Version {
		t.Fatalf("sync 要更新 policyVersion：%q", p.PolicyVersion)
	}
	if l := check(); !l.only() || len(l[levelWarn]) != 0 {
		t.Fatalf("sync 之後沒有提醒：%v", l)
	}
	// v0.14 的舊文件不一致：只提醒
	r := ratingPreset("15plus")
	legacy := &project{Name: "old", ContentRating: &r, dir: t.TempDir()}
	must(t, os.MkdirAll(filepath.Join(legacy.dir, "docs/content"), 0o755))
	must(t, os.WriteFile(filepath.Join(legacy.dir, "docs/content/CONTENT_RATING.md"), []byte("舊的"), 0o644))
	if l := runPolicy(legacy); !l.only() {
		t.Fatalf("v0.14 的舊文件不一致只提醒：%v", l)
	}
	if err := func() error { t.Chdir(t.TempDir()); return cmdContent([]string{"sync"}) }(); err == nil {
		t.Fatal("不在專案裡 sync 要失敗")
	}
}

// 敏感題材：開發者明確要求時，AI 直接更新宣告並 pt content sync（不需要變更申請），結果是 REVIEW_REQUIRED（不是錯誤）；
// AI 自己發現時要請開發者確認；不得為了通過檢查把 true 靜默改回 false（沒有重新產生文件時是宣告錯誤）。
func TestSensitiveContentWorkflow(t *testing.T) {
	requireEmbeddedSDK(t)
	dir := t.TempDir()
	t.Chdir(dir)
	must(t, cmdNew([]string{"fanxiao", "-y", "-profile", "strong"}))
	root := filepath.Join(dir, "fanxiao")
	t.Chdir(root)

	// AI 契約：明確要求即授權、AI 自己發現要確認、不得弱化題材；敏感題材不再要求 GAMEPLAY_POLICY_CHANGE_REQUEST
	wf := readFile(t, dir, "fanxiao", "docs/ai/AI_WORKFLOW.md")
	for _, s := range []string{"開發者明確要求敏感題材", "這個要求本身就是授權", "不需要再提出 `GAMEPLAY_POLICY_CHANGE_REQUEST`",
		"你自己發現", "請開發者確認", "你不是 `sensitiveContent` 的審核者", "不把應該是 `true` 的 `sensitiveContent` 改回 `false`", "REVIEW_REQUIRED 不是錯誤"} {
		if !strings.Contains(wf, s) {
			t.Errorf("AI_WORKFLOW.md 少了 %q", s)
		}
	}
	for _, f := range []string{"docs/ai/AI_WORKFLOW.md", "docs/ai/PROJECT_CONTEXT.md", "docs/content/CONTENT_PROFILE.md"} {
		doc := readFile(t, dir, "fanxiao", f)
		for _, banned := range []string{"敏感題材走 `GAMEPLAY_POLICY_CHANGE_REQUEST`", "玩法、經濟或敏感題材宣告要改", "要加入上表的題材時，先提出 `GAMEPLAY_POLICY_CHANGE_REQUEST`"} {
			if strings.Contains(doc, banned) {
				t.Errorf("%s 還要求敏感題材先提出變更申請：%q", f, banned)
			}
		}
	}
	prof := readFile(t, dir, "fanxiao", "docs/content/CONTENT_PROFILE.md")
	if !strings.Contains(prof, "這個要求本身就是授權") || !strings.Contains(prof, "先請開發者確認") || !strings.Contains(prof, "改回 `false`") {
		t.Error("CONTENT_PROFILE.md 要寫明敏感題材的流程")
	}

	// 開發者要求類《返校》的歷史政治恐怖題材：AI 直接改四個描寫欄位並 sync
	setSensitive := func(vals map[string]bool) {
		raw, _ := os.ReadFile(filepath.Join(root, projectFile))
		m := map[string]any{}
		must(t, json.Unmarshal(raw, &m))
		sc := m["sensitiveContent"].(map[string]any)
		for k, v := range vals {
			sc[k] = v
		}
		b, _ := json.MarshalIndent(m, "", "  ")
		must(t, os.WriteFile(filepath.Join(root, projectFile), b, 0o644))
	}
	check := func() levels {
		p, err := findProject()
		must(t, err)
		return runPolicy(p)
	}
	depictions := map[string]bool{"historicalPersecution": true, "politicalRepression": true, "discriminationDepiction": true, "hateSpeechDepiction": true}
	setSensitive(depictions)
	must(t, cmdContent([]string{"sync"}))
	l := check()
	if !l.only(levelReview) || len(l[levelReview]) != 4 {
		t.Fatalf("四個描寫欄位都是 REVIEW_REQUIRED、沒有錯誤：%v", l)
	}
	if !strings.Contains(readFile(t, dir, "fanxiao", "docs/content/CONTENT_PROFILE.md"), "| 威權統治、白色恐怖、監控、審訊、政治監禁（`politicalRepression`） | `true` |") {
		t.Error("sync 之後文件要反映宣告")
	}

	// 為了讓 pt check 通過把 true 靜默改回 false（沒有重新產生文件）：文件和宣告不一致，是宣告錯誤
	setSensitive(map[string]bool{"politicalRepression": false})
	if l := check(); len(l[levelSchema]) == 0 {
		t.Fatalf("靜默把 true 改回 false 要被抓到（文件不一致）：%v", l)
	}
	setSensitive(depictions)

	// 兩個鼓吹／煽動欄位仍是硬性規則
	for _, k := range []string{"hateAdvocacy", "encourageProtectedClassViolence"} {
		setSensitive(map[string]bool{k: true})
		must(t, cmdContent([]string{"sync"})) // 宣告格式正確就能 sync（硬性規則照實寫進文件）
		if l := check(); len(l[levelHard]) != 1 {
			t.Errorf("%s 要是 ERROR_HARD_RULE：%v", k, l)
		}
		setSensitive(map[string]bool{k: false})
		must(t, cmdContent([]string{"sync"}))
	}
}
