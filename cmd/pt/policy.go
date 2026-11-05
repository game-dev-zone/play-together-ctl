package main

// 政策宣告與驗證（v0.15）：playtogether.json 記錄開發者的宣告，pt check 只做五件事：
// 記錄宣告、維持 AI 開發契約、驗證 schema 與宣告一致性、擋下明確的平台硬性規則、標示需要平台審核的項目。
// pt 不是內容審核器，也不是法律分級器：上架准駁、正式分級、警語、標籤、玩法是否接受，全部由玩聚平台審核決定。
//
//   Platform Audience（使用者資格 18+）→ Content Profile（呈現強度）＋ Sensitive Content（敏感題材）
//   → Gameplay Policy → Economy Policy → pt 驗證 → 玩聚平台審核（唯一的最終准駁者）

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"sort"
	"strings"
)

// 檢查結果的等級。
const (
	levelHard   = "hard"   // ERROR_HARD_RULE：平台硬性規則，-skip-check 不能略過
	levelSchema = "schema" // ERROR_SCHEMA：宣告錯誤或自相矛盾，-skip-check 不能略過
	levelBuild  = "build"  // ERROR_BUILD：建置或本機設定沒準備好，-skip-check 可以略過
	levelReview = "review" // REVIEW_REQUIRED：可以送審，需要玩聚平台人工審核（不代表已批准）
	levelWarn   = "warn"   // WARNING：遷移、文件不同步、建議事項
	levelInfo   = "info"   // 說明，不需要處理
)

func isErrorLevel(l string) bool { return l == levelHard || l == levelSchema || l == levelBuild }

type addFunc func(level, format string, a ...any)

type platformAudience struct {
	MinimumUserAge int `json:"minimumUserAge"` // 平台使用者資格（固定 18），不是遊戲內容分級
}

type contentProfile struct {
	Profile       string `json:"profile"`     // general、moderate、strong、custom
	Provisional   bool   `json:"provisional"` // 一律 true：開發期間的內容邊界，不是正式分級
	Violence      string `json:"violence"`
	Blood         string `json:"blood"`
	Horror        string `json:"horror"`
	SexualContent string `json:"sexualContent"`
	Language      string `json:"language"`
	Alcohol       string `json:"alcohol"`
	Tobacco       string `json:"tobacco"`
}

// sensitiveContent：描寫（歷史、政治、批判性敘事、反派台詞）需要平台審核；鼓吹仇恨是硬性規則。
type sensitiveContent struct {
	HistoricalPersecution           bool `json:"historicalPersecution"`
	PoliticalRepression             bool `json:"politicalRepression"`
	DiscriminationDepiction         bool `json:"discriminationDepiction"`
	HateSpeechDepiction             bool `json:"hateSpeechDepiction"`
	HateAdvocacy                    bool `json:"hateAdvocacy"`
	EncourageProtectedClassViolence bool `json:"encourageProtectedClassViolence"`
}

type tableGame struct {
	Enabled          bool     `json:"enabled"`
	Category         string   `json:"category"`
	Presentation     string   `json:"presentation"`
	Cards            string   `json:"cards"`
	Dice             string   `json:"dice"`
	ScoreTokens      string   `json:"scoreTokens"`      // 一般計分或資源 token（桌遊的分數、資源）：本身不需要審核
	WageringTokens   string   `json:"wageringTokens"`   // 下注用的籌碼（虛擬下注的本金）：需要平台審核
	VirtualWagering  string   `json:"virtualWagering"`  // Bet／Call／Raise／All-in 等下注型動作，要有下注用的籌碼
	TokenPersistence string   `json:"tokenPersistence"` // 下注籌碼的保存範圍
	FreeTokenSources []string `json:"freeTokenSources"` // 免費取得下注籌碼的方式
}

type gameplayPolicy struct {
	ChanceMechanics string    `json:"chanceMechanics"`
	TableGame       tableGame `json:"tableGame"`
}

type economyPolicy struct {
	RealMoney                  bool `json:"realMoney"`
	RealMoneyWagering          bool `json:"realMoneyWagering"`
	PurchasableStake           bool `json:"purchasableStake"`
	WithdrawableValue          bool `json:"withdrawableValue"`
	RedeemableValue            bool `json:"redeemableValue"`
	TransferableRealWorldValue bool `json:"transferableRealWorldValue"`
	PaidRandomReward           bool `json:"paidRandomReward"`
}

type policy struct {
	Audience  platformAudience
	Content   contentProfile
	Sensitive sensitiveContent
	Gameplay  gameplayPolicy
	Economy   economyPolicy
}

var (
	profiles       = []string{"general", "moderate", "strong", "custom"}
	defaultProfile = "general" // 使用者 18+ 是平台資格，不是內容強度；需要較強內容時明確選 moderate、strong 或 custom
	profileLabel   = map[string]string{"general": "General／輕量內容", "moderate": "Moderate／中度內容", "strong": "Strong／較強內容", "custom": "Custom／自訂"}
	// preset 是固定值：要調整任何一項就改成 custom
	profilePresets = map[string]contentProfile{
		"general":  {Violence: "mild", Blood: "none", Horror: "mild", SexualContent: "none", Language: "none", Alcohol: "none", Tobacco: "none"},
		"moderate": {Violence: "moderate", Blood: "mild", Horror: "moderate", SexualContent: "none", Language: "mild", Alcohol: "allowed", Tobacco: "none"},
		"strong":   {Violence: "strong", Blood: "moderate", Horror: "strong", SexualContent: "suggestive", Language: "strong", Alcohol: "allowed", Tobacco: "allowed"},
	}
	legacyProfileNames = map[string]string{"general": "general", "12plus": "moderate", "15plus": "strong", "custom": "custom"}

	tableCategories   = []string{"none", "card", "poker", "mahjong", "dice", "board", "slot", "roulette", "baccarat", "other"}
	tableLabel        = map[string]string{"none": "沒有牌桌玩法", "card": "卡牌", "poker": "撲克", "mahjong": "麻將", "dice": "骰子", "board": "桌遊", "slot": "吃角子老虎", "roulette": "輪盤", "baccarat": "類百家樂", "other": "其他牌桌玩法"}
	presentations     = []string{"standard", "casinoLike"}
	noneAllowed       = []string{"none", "allowed"}
	tokenPersistences = []string{"none", "round", "room", "persistent"}
	freeTokenSources  = []string{"daily", "adReward", "achievement"}
)

// profileDim 是 Content Profile 的一個維度（說明沿用 v0.14 的 ratingDims）。
type profileDim struct {
	ratingDim
	val func(*contentProfile) *string
}

var profileDims = func() []profileDim {
	getters := map[string]func(*contentProfile) *string{
		"violence": func(c *contentProfile) *string { return &c.Violence }, "blood": func(c *contentProfile) *string { return &c.Blood },
		"horror": func(c *contentProfile) *string { return &c.Horror }, "sexualContent": func(c *contentProfile) *string { return &c.SexualContent },
		"language": func(c *contentProfile) *string { return &c.Language }, "alcohol": func(c *contentProfile) *string { return &c.Alcohol },
		"tobacco": func(c *contentProfile) *string { return &c.Tobacco },
	}
	var out []profileDim
	for _, d := range ratingDims {
		if g, ok := getters[d.key]; ok {
			out = append(out, profileDim{d, g})
		}
	}
	return out
}()

// presetProfile 回傳 profile 的固定值（custom 從 general 開始逐項調整）。
func presetProfile(profile string) contentProfile {
	base := profile
	if base == "custom" {
		base = "general"
	}
	c := profilePresets[base]
	c.Profile, c.Provisional = profile, true
	return c
}

// tablePreset 是 -table-game 各類型的起始值：只填常用的能力欄位，不含任何准駁或分級。
func tablePreset(category string) tableGame {
	t := tableGame{Category: "none", Presentation: "standard", Cards: "none", Dice: "none", ScoreTokens: "none", WageringTokens: "none", VirtualWagering: "none", TokenPersistence: "none", FreeTokenSources: []string{}}
	if category == "" || category == "none" {
		return t
	}
	t.Enabled, t.Category = true, category
	switch category {
	case "card":
		t.Cards = "allowed"
	case "poker", "baccarat":
		t.Cards, t.WageringTokens, t.VirtualWagering = "allowed", "allowed", "allowed"
	case "mahjong":
		t.Cards, t.ScoreTokens = "allowed", "allowed"
	case "dice":
		t.Dice = "allowed"
	case "board":
		t.Dice, t.ScoreTokens = "allowed", "allowed"
	case "slot", "roulette":
		t.WageringTokens, t.VirtualWagering = "allowed", "allowed"
	}
	if category == "slot" || category == "roulette" || category == "baccarat" {
		t.Presentation = "casinoLike"
	}
	if t.WageringTokens == "allowed" {
		t.TokenPersistence = "round"
	}
	return t
}

func defaultPolicy(c contentProfile, table string) policy {
	return policy{Audience: platformAudience{18}, Content: c, Gameplay: gameplayPolicy{ChanceMechanics: "allowed", TableGame: tablePreset(table)}}
}

// policyJSON 是寫進 playtogether.json 的五個區塊（縮排配合範本）。
func policyJSON(pl policy) string {
	blocks := []struct {
		key string
		v   any
	}{{"platformAudience", pl.Audience}, {"contentProfile", pl.Content}, {"sensitiveContent", pl.Sensitive}, {"gameplay", pl.Gameplay}, {"economy", pl.Economy}}
	parts := make([]string, 0, len(blocks))
	for _, b := range blocks {
		raw, _ := json.MarshalIndent(b.v, "  ", "  ")
		parts = append(parts, fmt.Sprintf("%q: %s", b.key, raw))
	}
	return strings.Join(parts, ",\n  ")
}

// ---------- 規則資料（policy/rules.json） ----------

// policyRule：條件（欄位等於其中一個值）→ 硬性規則或需要平台審核。清單由平台產品／審核政策負責人決定，
// pt 維護者同步；規則只寫「需要審核」與原因，不寫分級、不寫准駁。
type policyRule struct {
	ID       string   `json:"id"`
	Field    string   `json:"field"`
	Values   []string `json:"values"`
	Level    string   `json:"level"` // hard、review
	Category string   `json:"category"`
	Reason   string   `json:"reason"`
}

type policyRules struct {
	Version string       `json:"version"`
	Rules   []policyRule `json:"rules"`
}

func loadPolicyRules() policyRules {
	raw, err := embedded.ReadFile("policy/rules.json")
	if err != nil {
		panic("內嵌的 policy/rules.json 不見了：" + err.Error())
	}
	var r policyRules
	if err := json.Unmarshal(raw, &r); err != nil {
		panic("內嵌的 policy/rules.json 格式錯誤：" + err.Error())
	}
	return r
}

// fields 把宣告攤平成「欄位路徑 → 值」（布林是 true／false，陣列是每個元素）。
func (pl *policy) fields() map[string][]string {
	b := func(v bool) []string { return []string{fmt.Sprint(v)} }
	s := func(v string) []string { return []string{v} }
	t, c, e, x := pl.Gameplay.TableGame, pl.Content, pl.Economy, pl.Sensitive
	return map[string][]string{
		"platformAudience.minimumUserAge": s(fmt.Sprint(pl.Audience.MinimumUserAge)),
		"contentProfile.profile":          s(c.Profile), "contentProfile.violence": s(c.Violence), "contentProfile.blood": s(c.Blood),
		"contentProfile.horror": s(c.Horror), "contentProfile.sexualContent": s(c.SexualContent), "contentProfile.language": s(c.Language),
		"contentProfile.alcohol": s(c.Alcohol), "contentProfile.tobacco": s(c.Tobacco),
		"sensitiveContent.historicalPersecution": b(x.HistoricalPersecution), "sensitiveContent.politicalRepression": b(x.PoliticalRepression),
		"sensitiveContent.discriminationDepiction": b(x.DiscriminationDepiction), "sensitiveContent.hateSpeechDepiction": b(x.HateSpeechDepiction),
		"sensitiveContent.hateAdvocacy": b(x.HateAdvocacy), "sensitiveContent.encourageProtectedClassViolence": b(x.EncourageProtectedClassViolence),
		"gameplay.chanceMechanics": s(pl.Gameplay.ChanceMechanics), "gameplay.tableGame.enabled": b(t.Enabled),
		"gameplay.tableGame.category": s(t.Category), "gameplay.tableGame.presentation": s(t.Presentation),
		"gameplay.tableGame.cards": s(t.Cards), "gameplay.tableGame.dice": s(t.Dice), "gameplay.tableGame.scoreTokens": s(t.ScoreTokens),
		"gameplay.tableGame.wageringTokens":  s(t.WageringTokens),
		"gameplay.tableGame.virtualWagering": s(t.VirtualWagering), "gameplay.tableGame.tokenPersistence": s(t.TokenPersistence),
		"gameplay.tableGame.freeTokenSources": t.FreeTokenSources,
		"economy.realMoney":                   b(e.RealMoney), "economy.realMoneyWagering": b(e.RealMoneyWagering),
		"economy.purchasableStake": b(e.PurchasableStake), "economy.withdrawableValue": b(e.WithdrawableValue),
		"economy.redeemableValue": b(e.RedeemableValue), "economy.transferableRealWorldValue": b(e.TransferableRealWorldValue),
		"economy.paidRandomReward": b(e.PaidRandomReward),
	}
}

// matchRules 回傳宣告符合的規則（依規則資料的順序）。
func matchRules(pl policy) []policyRule {
	fields := pl.fields()
	var out []policyRule
	for _, r := range loadPolicyRules().Rules {
		for _, v := range fields[r.Field] {
			if slices.Contains(r.Values, v) {
				out = append(out, r)
				break
			}
		}
	}
	return out
}

func applyRules(pl policy, add addFunc) {
	for _, r := range matchRules(pl) {
		v := strings.Join(pl.fields()[r.Field], ",")
		if r.Level == levelHard {
			add(levelHard, "%s（%s：%s）：玩聚平台硬性規則，任何情況都不能上架（%s）", r.Reason, r.Field, v, r.Category)
		} else {
			add(levelReview, "%s（%s：%s）", r.Reason, r.Field, v)
		}
	}
}

// reviewReasons 是需要平台審核的項目（給 pt new 的訊息與文件）。
func reviewReasons(pl policy) (hard, review []string) {
	for _, r := range matchRules(pl) {
		if r.Level == levelHard {
			hard = append(hard, r.Reason)
		} else {
			review = append(review, r.Reason)
		}
	}
	return
}

// ---------- 讀取與驗證 ----------

// strictDecode：raw 必須是 JSON 物件、有全部的 keys、沒有未知的鍵、型別正確（json.Unmarshal 會忽略未知鍵，也分不出缺欄位與 false）。
func strictDecode(raw json.RawMessage, name string, keys []string, dst any, add addFunc) bool {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil || m == nil {
		add(levelSchema, "playtogether.json 的 %s 要是 JSON 物件", name)
		return false
	}
	ok := true
	for _, k := range duplicateKeys(raw) {
		add(levelSchema, "playtogether.json 的 %s 有重複的欄位 %q：每個欄位只能寫一次", name, k)
		ok = false
	}
	for k, v := range m {
		if strings.TrimSpace(string(v)) == "null" && slices.Contains(keys, k) {
			add(levelSchema, "playtogether.json 的 %s.%s 不能是 null：要寫明確的值", name, k)
			ok = false
		}
	}
	extra := []string{}
	for k := range m {
		if !slices.Contains(keys, k) {
			extra = append(extra, k)
		}
	}
	sort.Strings(extra)
	for _, k := range extra {
		add(levelSchema, "playtogether.json 的 %s 有未知的欄位 %q（拼錯了嗎？可用的欄位：%s）", name, k, strings.Join(keys, "、"))
		ok = false
	}
	for _, k := range keys {
		if _, has := m[k]; !has {
			add(levelSchema, "playtogether.json 的 %s 缺少 %s：宣告要完整（用 pt new 以相同設定建暫存專案，對照複製）", name, k)
			ok = false
		}
	}
	if err := json.Unmarshal(raw, dst); err != nil {
		add(levelSchema, "playtogether.json 的 %s 欄位型別錯誤：%v", name, err)
		return false
	}
	return ok
}

// duplicateKeys 回傳 JSON 物件最外層重複出現的鍵（json.Unmarshal 會默默用後面的值）。
func duplicateKeys(raw []byte) []string {
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	if t, err := dec.Token(); err != nil || t != json.Delim('{') {
		return nil
	}
	seen := map[string]bool{}
	var dup []string
	for dec.More() {
		t, err := dec.Token()
		if err != nil {
			return dup
		}
		k, _ := t.(string)
		if seen[k] && !slices.Contains(dup, k) {
			dup = append(dup, k)
		}
		seen[k] = true
		var skip json.RawMessage
		if err := dec.Decode(&skip); err != nil {
			return dup
		}
	}
	return dup
}

// checkTopLevel：playtogether.json 最外層的鍵不能重複；看起來像拼錯的宣告鍵是宣告錯誤（否則整組宣告會被當成不存在）。
func checkTopLevel(raw []byte, add addFunc) {
	if raw == nil {
		return
	}
	for _, k := range duplicateKeys(raw) {
		add(levelSchema, "playtogether.json 有重複的欄位 %q：每個欄位只能寫一次", k)
	}
	var m map[string]json.RawMessage
	if json.Unmarshal(raw, &m) != nil {
		return
	}
	known := projectKeys()
	policyKeys := []string{"platformAudience", "contentProfile", "sensitiveContent", "gameplay", "economy", "contentRating"}
	known = append(known, policyKeys...)
	extra := []string{}
	for k := range m {
		if !slices.Contains(known, k) {
			extra = append(extra, k)
		}
	}
	sort.Strings(extra)
	for _, k := range extra {
		lk := strings.ToLower(k)
		suspicious := false
		for _, w := range []string{"audience", "profile", "sensitive", "gameplay", "economy", "rating", "content", "policy"} {
			if strings.Contains(lk, w) {
				suspicious = true
			}
		}
		for _, pk := range policyKeys {
			if editDistance(lk, strings.ToLower(pk)) <= 3 {
				suspicious = true
			}
		}
		if suspicious {
			add(levelSchema, "playtogether.json 的 %q 不是 pt 認得的欄位：宣告的欄位是 %s（拼錯了嗎？）", k, strings.Join(policyKeys[:5], "、"))
		} else {
			add(levelWarn, "playtogether.json 的 %q 不是 pt 認得的欄位（pt 不會讀它）", k)
		}
	}
}

// editDistance 是兩個字串的編輯距離（判斷拼錯的宣告鍵）。
func editDistance(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	prev := make([]int, len(rb)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(ra); i++ {
		cur := make([]int, len(rb)+1)
		cur[0] = i
		for j := 1; j <= len(rb); j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev = cur
	}
	return prev[len(rb)]
}

// projectKeys 是 project 結構所有的 json 欄位名稱（含 omitempty 的欄位）。
func projectKeys() []string {
	t := reflect.TypeOf(project{})
	var keys []string
	for i := 0; i < t.NumField(); i++ {
		if name, _, _ := strings.Cut(t.Field(i).Tag.Get("json"), ","); name != "" && name != "-" {
			keys = append(keys, name)
		}
	}
	return keys
}

func jsonKeys(v any) []string {
	raw, _ := json.Marshal(v)
	var m map[string]json.RawMessage
	_ = json.Unmarshal(raw, &m)
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func (p *project) hasPolicy() bool {
	return p.PlatformAudience != nil || p.ContentProfile != nil || p.SensitiveContent != nil || p.Gameplay != nil || p.Economy != nil
}

// readPolicy 讀出宣告並做 schema 驗證；舊格式（v0.14 的 contentRating）依舊規則驗證後對應過來。
// ok 是 false 表示沒有任何宣告。
func readPolicy(p *project, add addFunc) (pl policy, legacy, ok bool) {
	if !p.hasPolicy() {
		if p.ContentRating == nil {
			add(levelWarn, "playtogether.json 沒有內容與玩法宣告（platformAudience、contentProfile、sensitiveContent、gameplay、economy）：請補上（用新版 pt new 以相同設定建暫存專案複製，連同 docs/content/），讓 AI 知道開發契約")
			return pl, false, false
		}
		return legacyPolicy(p.ContentRating, add), true, true
	}
	if p.ContentRating != nil {
		add(levelWarn, "playtogether.json 同時有 v0.14 的 contentRating 與 v0.15 的宣告：以 v0.15 為準，請刪掉 contentRating")
	}
	pl = defaultPolicy(presetProfile(defaultProfile), "none")
	blocks := []struct {
		name string
		raw  json.RawMessage
		dst  any
	}{{"platformAudience", p.PlatformAudience, &pl.Audience}, {"contentProfile", p.ContentProfile, &pl.Content},
		{"sensitiveContent", p.SensitiveContent, &pl.Sensitive}, {"economy", p.Economy, &pl.Economy}}
	for _, b := range blocks {
		if b.raw == nil {
			add(levelSchema, "playtogether.json 缺少 %s：v0.15 的宣告要完整（用 pt new 以相同設定建暫存專案，對照複製）", b.name)
			continue
		}
		strictDecode(b.raw, b.name, jsonKeys(b.dst), b.dst, add)
	}
	if p.Gameplay == nil {
		add(levelSchema, "playtogether.json 缺少 gameplay：v0.15 的宣告要完整")
	} else {
		var g struct {
			ChanceMechanics string          `json:"chanceMechanics"`
			TableGame       json.RawMessage `json:"tableGame"`
		}
		strictDecode(p.Gameplay, "gameplay", []string{"chanceMechanics", "tableGame"}, &g, add)
		pl.Gameplay.ChanceMechanics = g.ChanceMechanics
		if g.TableGame != nil {
			pl.Gameplay.TableGame = tableGame{}
			strictDecode(g.TableGame, "gameplay.tableGame", jsonKeys(tablePreset("none")), &pl.Gameplay.TableGame, add)
		}
	}
	return pl, false, true
}

// checkPolicy 是 pt check 的政策部分。
func checkPolicy(p *project, add addFunc) {
	checkTopLevel(p.raw, add)
	pl, legacy, ok := readPolicy(p, add)
	if !ok {
		return
	}
	if !legacy {
		validatePolicy(pl, add)
	}
	applyRules(pl, add)
	checkPolicyDocs(p, pl, legacy, add)
}

// validatePolicy：值是否合法、preset 是否被手改、牌桌宣告是否一致（硬性規則與審核項目由規則資料處理）。
func validatePolicy(pl policy, add addFunc) {
	if pl.Audience.MinimumUserAge != 18 {
		add(levelSchema, "platformAudience.minimumUserAge 必須是 18（目前是 %d）：這是玩聚的平台使用者資格，開發者不能修改；它不是遊戲內容分級", pl.Audience.MinimumUserAge)
	}
	c := pl.Content
	if !slices.Contains(profiles, c.Profile) {
		hint := ""
		if n, legacyName := legacyProfileNames[c.Profile]; legacyName && n != c.Profile {
			hint = fmt.Sprintf("；v0.14 的 %s 請改成 %s", c.Profile, n)
		}
		add(levelSchema, "contentProfile.profile 只能是 general、moderate、strong 或 custom（目前是 %q%s）", c.Profile, hint)
	}
	if !c.Provisional {
		add(levelSchema, "contentProfile.provisional 必須是 true：這是開發期間的內容邊界，不是正式分級")
	}
	valid := true
	for _, d := range profileDims {
		v := *d.val(&c)
		if slices.Contains(d.beyond, v) {
			continue // 平台不收的值：規則資料回報硬性規則
		}
		if levelIndex(d.ratingDim, v) < 0 {
			add(levelSchema, "contentProfile.%s 只能是 %s（目前是 %q）", d.key, strings.Join(d.levels, "、"), v)
			valid = false
		}
	}
	if preset, isPreset := profilePresets[c.Profile]; isPreset && valid {
		var diff []string
		for _, d := range profileDims {
			if got, want := *d.val(&c), *d.val(&preset); got != want {
				diff = append(diff, fmt.Sprintf("%s 是 %q（預設 %q）", d.key, got, want))
			}
		}
		if len(diff) > 0 {
			add(levelSchema, "contentProfile.profile 是 %s，但 %s：general、moderate、strong 是固定值，要調整個別項目請把 profile 改成 custom（AI 不能自行提高內容邊界，要先提出 CONTENT_PROFILE_CHANGE_REQUEST）", c.Profile, strings.Join(diff, "、"))
		}
	}

	g := pl.Gameplay
	in := func(name, v string, list []string) {
		if !slices.Contains(list, v) {
			add(levelSchema, "%s 只能是 %s（目前是 %q）", name, strings.Join(list, "、"), v)
		}
	}
	in("gameplay.chanceMechanics", g.ChanceMechanics, noneAllowed)
	t := g.TableGame
	in("gameplay.tableGame.category", t.Category, tableCategories)
	in("gameplay.tableGame.presentation", t.Presentation, presentations)
	in("gameplay.tableGame.cards", t.Cards, noneAllowed)
	in("gameplay.tableGame.dice", t.Dice, noneAllowed)
	in("gameplay.tableGame.scoreTokens", t.ScoreTokens, noneAllowed)
	in("gameplay.tableGame.wageringTokens", t.WageringTokens, noneAllowed)
	in("gameplay.tableGame.virtualWagering", t.VirtualWagering, noneAllowed)
	in("gameplay.tableGame.tokenPersistence", t.TokenPersistence, tokenPersistences)
	seen := map[string]bool{}
	for _, s := range t.FreeTokenSources {
		in("gameplay.tableGame.freeTokenSources 的項目", s, freeTokenSources)
		if seen[s] {
			add(levelSchema, "gameplay.tableGame.freeTokenSources 重複：%s", s)
		}
		seen[s] = true
	}
	if !t.Enabled {
		if t.Category != "none" || t.Presentation != "standard" || t.Cards != "none" || t.Dice != "none" || t.ScoreTokens != "none" || t.WageringTokens != "none" ||
			t.VirtualWagering != "none" || t.TokenPersistence != "none" || len(t.FreeTokenSources) > 0 {
			add(levelSchema, "gameplay.tableGame.enabled 是 false，但其他牌桌欄位有設定：沒有牌桌玩法時 category、cards、dice、scoreTokens、wageringTokens、virtualWagering、tokenPersistence 要是 none、presentation 是 standard、freeTokenSources 是空的；有牌桌玩法時把 enabled 改成 true")
		}
		return
	}
	if t.Category == "none" {
		add(levelSchema, "gameplay.tableGame.enabled 是 true，但 category 是 none：請宣告牌桌類型（%s）", strings.Join(tableCategories[1:], "、"))
	}
	if t.VirtualWagering == "allowed" && t.WageringTokens != "allowed" {
		add(levelSchema, "gameplay.tableGame.virtualWagering 是 allowed，但 wageringTokens 是 none：虛擬下注要宣告下注用的籌碼（一般計分 token 是 scoreTokens，不能拿來下注）")
	}
	if t.WageringTokens == "allowed" && t.TokenPersistence == "none" {
		add(levelSchema, "gameplay.tableGame.wageringTokens 是 allowed：請宣告下注籌碼的保存範圍 tokenPersistence（round、room 或 persistent）")
	}
	if t.WageringTokens != "allowed" && (t.TokenPersistence != "none" || len(t.FreeTokenSources) > 0) {
		add(levelSchema, "gameplay.tableGame.wageringTokens 是 none，但 tokenPersistence 或 freeTokenSources 有設定：這兩項是下注籌碼的設定，沒有下注籌碼時要是 none 與空的")
	}
	if slices.Contains([]string{"card", "poker", "mahjong", "baccarat"}, t.Category) && t.Cards != "allowed" {
		add(levelWarn, "gameplay.tableGame.category 是 %s，但 cards 是 none：確認宣告是否正確", t.Category)
	}
	if t.Category == "dice" && t.Dice != "allowed" {
		add(levelWarn, "gameplay.tableGame.category 是 dice，但 dice 是 none：確認宣告是否正確")
	}
}

// policyDocs 是依宣告產生的開發契約文件（相對路徑 → 內容）。
func policyDocs(name string, pl policy) map[string]string {
	return map[string]string{
		"docs/content/CONTENT_PROFILE.md":  contentProfileDoc(name, pl),
		"docs/content/GAMEPLAY_ECONOMY.md": gameplayEconomyDoc(name, pl),
	}
}

// cmdContent：pt content sync 依 playtogether.json 重新產生 docs/content/ 的開發契約文件，並記錄目前的平台政策版本。
// 宣告本身有錯誤（schema）時不產生，避免把錯的宣告寫成 AI 的契約；硬性規則與需要審核的項目照實寫進文件。
func cmdContent(args []string) error {
	if len(args) != 1 || args[0] != "sync" {
		return errors.New("用法：pt content sync（依 playtogether.json 重新產生 docs/content/CONTENT_PROFILE.md 與 GAMEPLAY_ECONOMY.md）")
	}
	p, err := findProject()
	if err != nil {
		return err
	}
	if p == nil {
		return errors.New("找不到 playtogether.json：請在 pt new 建立的專案裡執行")
	}
	if !p.hasPolicy() {
		return errors.New("playtogether.json 沒有 v0.15 的宣告（platformAudience、contentProfile、sensitiveContent、gameplay、economy）：v0.14 的專案請先用 pt new -profile … 建暫存專案，複製這五個區塊")
	}
	var bad []string
	add := func(level, format string, a ...any) {
		if level == levelSchema {
			bad = append(bad, fmt.Sprintf(format, a...))
		}
	}
	checkTopLevel(p.raw, add)
	pl, _, _ := readPolicy(p, add)
	validatePolicy(pl, add)
	if len(bad) > 0 {
		for _, m := range bad {
			fmt.Println("  ✗", m)
		}
		return errors.New("宣告有錯誤，沒有產生文件：請先修正 playtogether.json")
	}
	for file, content := range policyDocs(p.Name, pl) {
		dst := filepath.Join(p.dir, filepath.FromSlash(file))
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		old, _ := os.ReadFile(dst)
		if string(old) == content {
			fmt.Println("  ・", file, "（沒有變更）")
			continue
		}
		if err := os.WriteFile(dst, []byte(content), 0o644); err != nil {
			return err
		}
		fmt.Println("  ✓", file, "已更新")
	}
	if v := loadPolicyRules().Version; p.PolicyVersion != v {
		fmt.Printf("  ✓ policyVersion：%s → %s\n", map[bool]string{true: "（沒有）", false: p.PolicyVersion}[p.PolicyVersion == ""], v)
		p.PolicyVersion = v
		if err := p.save(); err != nil {
			return err
		}
	}
	fmt.Println("完成：再執行 pt check 確認結果（需要平台審核的項目仍要送審，不代表已批准）")
	return nil
}

// checkPolicyDocs：docs/content/ 的文件要和宣告一致（AI 讀的是文件）。
func checkPolicyDocs(p *project, pl policy, legacy bool, add addFunc) {
	if p.dir == "" {
		return
	}
	if legacy {
		checkContentRatingDoc(p, add)
		return
	}
	// AI 讀的是這兩份文件（開發契約）：v0.15 的文件缺少或和宣告不一致是宣告錯誤（-skip-check 不能略過）
	for file, want := range policyDocs(p.Name, pl) {
		got, err := os.ReadFile(filepath.Join(p.dir, filepath.FromSlash(file)))
		if err != nil {
			add(levelSchema, "沒有 %s：AI 依這份開發契約工作，執行 pt content sync 依 playtogether.json 產生", file)
		} else if string(got) != want {
			add(levelSchema, "%s 和 playtogether.json 的宣告不一致：AI 讀的是文件，確認宣告正確後執行 pt content sync 重新產生（不要為了通過檢查而修改宣告）", file)
		}
	}
	switch {
	case p.PolicyVersion == "":
		add(levelWarn, "playtogether.json 沒有 policyVersion：執行 pt content sync 記錄目前的平台政策版本（%s）", loadPolicyRules().Version)
	case p.PolicyVersion != loadPolicyRules().Version:
		add(levelWarn, "玩聚平台政策已更新（專案記錄的是 %s，這個 pt 是 %s）：確認上面的檢查結果後，執行 pt content sync 同步文件與版本", p.PolicyVersion, loadPolicyRules().Version)
	}
	if _, err := os.Stat(filepath.Join(p.dir, "docs", "content", "CONTENT_RATING.md")); err == nil {
		add(levelWarn, "docs/content/CONTENT_RATING.md 是 v0.14 的舊文件，已經沒有作用：請刪除（AI 改讀 CONTENT_PROFILE.md 與 GAMEPLAY_ECONOMY.md）")
	}
}

// ---------- 舊格式（v0.14 的 contentRating） ----------

// legacyPolicy：依 v0.14 的規則驗證（v0.14 的 preset 和 v0.15 不同，不直接對應成新 preset，否則舊專案會變成宣告錯誤），
// 再把內容對應成 custom 邊界。gambling.wagering／simulated 的舊語意沒分真錢與虛擬，不對應成 virtualWagering。
func legacyPolicy(r *contentRating, add addFunc) policy {
	checkContentRating(r, add)
	newName := legacyProfileNames[r.Profile]
	if newName == "" {
		newName = "custom"
	}
	add(levelWarn, "playtogether.json 使用 v0.14 的 contentRating（%s → v0.15 的 %s）：舊格式仍可使用，建議改成 v0.15 的 platformAudience、contentProfile、sensitiveContent、gameplay、economy（用新版 pt new -profile %s 建暫存專案複製，連同 docs/content/）；targetAge 已移除：玩聚使用者資格固定 18+，內容只看 profile。注意：舊的 docs/content/CONTENT_RATING.md 寫著下注、模擬賭博、籌碼與博弈字眼一律禁止，v0.15 已改成牌、骰子、籌碼、Bet／Call／Raise 本身不違規、牌桌玩法由平台審核，遷移後 AI 改讀 CONTENT_PROFILE.md 與 GAMEPLAY_ECONOMY.md", r.Profile, newName, newName)
	c := contentProfile{Profile: "custom", Provisional: true, Violence: r.Violence, Blood: r.Blood, Horror: r.Horror,
		SexualContent: r.SexualContent, Language: r.Language, Alcohol: r.Alcohol, Tobacco: r.Tobacco}
	pl := defaultPolicy(c, "none")
	pl.Gameplay.ChanceMechanics = r.Gambling.ChanceMechanics
	pl.Economy.RealMoneyWagering = r.Gambling.RealMoney
	pl.Economy.PaidRandomReward = r.Gambling.PaidRandomReward
	if r.Gambling.Wagering {
		add(levelReview, "contentRating.gambling.wagering 是 true（v0.14 的舊欄位沒分真錢與虛擬下注）：需要玩聚平台人工審核；請改用 v0.15 的 gameplay.tableGame.virtualWagering（虛擬下注）或 economy.realMoneyWagering（真錢下注）說明是哪一種")
	}
	if r.Gambling.Simulated == "" {
		add(levelWarn, "contentRating.gambling.simulated 沒有填（v0.14 的舊欄位）：v0.15 不再使用，請改用 gameplay.tableGame 宣告牌桌玩法")
	} else if r.Gambling.Simulated != "none" {
		add(levelReview, "contentRating.gambling.simulated 是 %q（v0.14 的舊欄位）：需要玩聚平台人工審核；請改用 v0.15 的 gameplay.tableGame 宣告牌桌玩法", r.Gambling.Simulated)
	}
	return pl
}

// ---------- 文件 ----------

func onOff(v string) string {
	if v == "allowed" {
		return "使用"
	}
	return "沒有使用"
}

func boolMark(v bool) string {
	if v {
		return "`true`"
	}
	return "`false`"
}

// contentProfileDoc 產生 docs/content/CONTENT_PROFILE.md。
func contentProfileDoc(name string, pl policy) string {
	var b strings.Builder
	w := func(format string, a ...any) { fmt.Fprintf(&b, format, a...) }
	c := pl.Content
	w("# 內容邊界（%s）\n\n", name)
	w("> **Development Content Profile：開發期間的內容邊界，不是正式分級。** 正式分級、警語、標籤與能否上架由玩聚平台審核決定。\n")
	w("> 設定在 `playtogether.json` 的 `contentProfile` 與 `sensitiveContent`，這份文件依它產生。\n\n")
	w("| 項目 | 內容 |\n| --- | --- |\n")
	w("| Platform Audience | 18+（玩聚的使用者資格，不是遊戲內容分級） |\n")
	w("| Content Profile | %s（`%s`） |\n\n", profileLabel[c.Profile], c.Profile)
	w("## 怎麼使用\n\n")
	w("- 這是**上限，不是要求**：AI 不會因為上限較寬就主動加入暴力、血腥、恐怖、性暗示、粗俗語言、酒或菸；只在遊戲設計需要、開發者要求時使用，而且不超過下面的等級。\n")
	w("- **AI 不能自行提高內容邊界**：要加入超過目前等級的內容時，先提出 `CONTENT_PROFILE_CHANGE_REQUEST`（格式在 `docs/ai/AI_WORKFLOW.md`），開發者同意後修改 `playtogether.json`、執行 `pt content sync` 重新產生這份文件，再實作。不要先做內容、再為了通過 `pt check` 改宣告。\n")
	w("- 發現既有內容已經超出邊界時：降低內容以符合邊界，或提出 `CONTENT_PROFILE_CHANGE_REQUEST`；不要靜默放寬。\n")
	w("- `general`、`moderate`、`strong` 是固定值；要調整個別項目，profile 改成 `custom`（`pt check` 會檢查）。\n")
	w("- 牌、骰子、籌碼、虛擬下注不屬於內容邊界，見 `GAMEPLAY_ECONOMY.md`。\n\n")
	for _, d := range profileDims {
		v := *d.val(&c)
		w("## %s\n\n等級：`%s`\n\n", d.title, v)
		i := levelIndex(d.ratingDim, v)
		if i == 0 {
			w("目前：%s。\n", d.desc[v])
		} else if i > 0 {
			w("允許：%s。\n", d.desc[v])
		}
		if i >= 0 && i+1 < len(d.levels) {
			parts := []string{}
			for _, a := range d.levels[i+1:] {
				parts = append(parts, d.desc[a])
			}
			w("\n不加入（超過目前等級）：%s。\n", strings.Join(parts, "；"))
		}
		if d.never != "" {
			w("\n任何等級都不行（玩聚平台硬性規則）：%s。\n", d.never)
		}
		w("\n")
	}
	x := pl.Sensitive
	w("## 敏感題材\n\n")
	w("描寫仇恨 ≠ 鼓吹仇恨。歷史、教育、紀實、敘事、批判性內容可以描寫仇恨、迫害、歧視、政治暴力與威權統治；呈現強度依上面的內容邊界。\n\n")
	w("| 項目 | 宣告 | pt 的結果 |\n| --- | --- | --- |\n")
	w("| 歷史迫害、戰爭、殖民、屠殺、人權侵害（`historicalPersecution`） | %s | 需要平台審核 |\n", boolMark(x.HistoricalPersecution))
	w("| 威權統治、白色恐怖、監控、審訊、政治監禁（`politicalRepression`） | %s | 需要平台審核 |\n", boolMark(x.PoliticalRepression))
	w("| 劇情或批判性敘事中呈現的歧視（`discriminationDepiction`） | %s | 需要平台審核 |\n", boolMark(x.DiscriminationDepiction))
	w("| 角色台詞、史料中的仇恨或歧視言論（`hateSpeechDepiction`） | %s | 需要平台審核 |\n", boolMark(x.HateSpeechDepiction))
	w("| 鼓吹對受保護群體的仇恨、歧視、騷擾（`hateAdvocacy`） | %s | 平台硬性規則 |\n", boolMark(x.HateAdvocacy))
	w("| 把傷害真實受保護群體當成目標或獎勵、煽動暴力（`encourageProtectedClassViolence`） | %s | 平台硬性規則 |\n\n", boolMark(x.EncourageProtectedClassViolence))
	w("不得讓真實受保護群體成為應被仇恨、迫害或消滅的對象。\n\n")
	w("- 開發者明確要求上表的描寫題材時，這個要求本身就是授權：AI 直接把對應欄位改成 `true`、執行 `pt content sync`，並告知會變成需要平台審核。\n")
	w("- AI 自己發現既有內容可能涉及沒有宣告的題材時，先請開發者確認再更新。\n")
	w("- 不得為了讓 `pt check` 通過而刪除、淡化合法題材，或把應該是 `true` 的欄位改回 `false`；需要平台審核不是錯誤。\n\n")
	w("## 未成年角色\n\n")
	w("可以有未成年角色（例如校園、成長故事、家庭劇情、青少年運動）。任何內容邊界都**不得**性化未成年人，未成年人也不得出現在任何性內容（玩聚平台硬性規則）。\n")
	return b.String()
}

// gameplayEconomyDoc 產生 docs/content/GAMEPLAY_ECONOMY.md。
func gameplayEconomyDoc(name string, pl policy) string {
	var b strings.Builder
	w := func(format string, a ...any) { fmt.Fprintf(&b, format, a...) }
	t, e := pl.Gameplay.TableGame, pl.Economy
	w("# 玩法與經濟宣告（%s）\n\n", name)
	w("> 設定在 `playtogether.json` 的 `gameplay` 與 `economy`，這份文件依它產生。**能否上架、正式分級、警語、標籤與玩法是否接受，由玩聚平台審核決定**；`pt` 只擋明確的平台硬性規則、標示需要審核的項目。\n\n")
	w("## 牌桌玩法\n\n")
	w("牌桌、牌、骰子、籌碼、Bet／Call／Raise／Fold／All-in／Check（下注、跟注、加注、棄牌、全押、過牌）是玩法元素，**本身不違規**（Table Game ≠ Real-Money Gambling）。\n\n")
	w("| 項目 | 宣告 |\n| --- | --- |\n")
	w("| 一般隨機機制：骰子、抽牌、隨機事件、隨機掉落（`chanceMechanics`） | %s |\n", onOff(pl.Gameplay.ChanceMechanics))
	if !t.Enabled {
		w("| 牌桌玩法（`tableGame.enabled`） | 沒有 |\n\n")
	} else {
		w("| 牌桌類型（`tableGame.category`） | %s（`%s`） |\n", tableLabel[t.Category], t.Category)
		w("| 呈現（`presentation`） | `%s` |\n", t.Presentation)
		w("| 牌（`cards`） | %s |\n| 骰子（`dice`） | %s |\n| 一般計分或資源 token（`scoreTokens`） | %s |\n", onOff(t.Cards), onOff(t.Dice), onOff(t.ScoreTokens))
		w("| 下注用的籌碼（`wageringTokens`） | %s |\n", onOff(t.WageringTokens))
		w("| 虛擬下注（`virtualWagering`） | %s |\n", onOff(t.VirtualWagering))
		w("| 下注籌碼的保存範圍（`tokenPersistence`） | `%s` |\n", t.TokenPersistence)
		src := "沒有"
		if len(t.FreeTokenSources) > 0 {
			src = "`" + strings.Join(t.FreeTokenSources, "`、`") + "`"
		}
		w("| 免費取得下注籌碼（`freeTokenSources`） | %s |\n\n", src)
	}
	w("## 經濟（現實價值）\n\n")
	w("| 項目 | 宣告 | pt 的結果 |\n| --- | --- | --- |\n")
	w("| 真實金錢下注（`realMoneyWagering`） | %s | 平台硬性規則 |\n", boolMark(e.RealMoneyWagering))
	w("| 可購買的下注本金或籌碼，含用平台點數換得（`purchasableStake`） | %s | 平台硬性規則 |\n", boolMark(e.PurchasableStake))
	w("| 可提領（`withdrawableValue`） | %s | 平台硬性規則 |\n", boolMark(e.WithdrawableValue))
	w("| 可兌換現金、商品或現實財物（`redeemableValue`） | %s | 平台硬性規則 |\n", boolMark(e.RedeemableValue))
	w("| 玩家之間可轉讓具現實價值的資產（`transferableRealWorldValue`） | %s | 平台硬性規則 |\n", boolMark(e.TransferableRealWorldValue))
	w("| 付費隨機獎勵：付費抽獎、點數抽獎、付費轉蛋（`paidRandomReward`） | %s | 平台硬性規則 |\n", boolMark(e.PaidRandomReward))
	w("| 商業模式涉及真實金錢，但不是下注（`realMoney`） | %s | 需要平台審核 |\n\n", boolMark(e.RealMoney))
	w("- 不得宣稱或暗示可以用真實金錢獲利、兌換現金、提領或取得現實財物；不得導流到外部博彩服務。\n")
	w("- `realMoney` 需要審核只是商業模式的判斷；**技術上遊戲仍然不能自己接付款**（iframe 沒有付款能力，不得加入金流 SDK）。\n")
	w("- 看廣告取得籌碼是需要審核的玩法；**技術上遊戲仍然不能自己加入廣告 SDK**。\n\n")
	hard, review := reviewReasons(pl)
	w("## 目前的檢查結果\n\n")
	switch {
	case len(hard) > 0:
		w("有平台硬性規則問題（`pt upload` 會擋下）：%s。\n", strings.Join(hard, "；"))
	case len(review) > 0:
		w("需要玩聚平台人工審核（可以送審，**不代表已批准**）：%s。\n", strings.Join(review, "；"))
	default:
		w("沒有需要平台特別審核的玩法或經濟項目（上架仍要經過一般審核）。\n")
	}
	w("\n要改玩法或經濟宣告時，AI 先提出 `GAMEPLAY_POLICY_CHANGE_REQUEST`（格式在 `docs/ai/AI_WORKFLOW.md`），開發者同意後修改 `playtogether.json`，再執行 `pt content sync` 重新產生這份文件（文件和宣告不一致時 `pt check` 是宣告錯誤）。\n")
	return b.String()
}

// policySummary 是 PROJECT_CONTEXT.md 的摘要列。
func policySummary(pl policy) map[string]string {
	t := pl.Gameplay.TableGame
	table := "none"
	if t.Enabled {
		table = fmt.Sprintf("%s（presentation: %s）", t.Category, t.Presentation)
	}
	src := "none"
	if len(t.FreeTokenSources) > 0 {
		src = strings.Join(t.FreeTokenSources, ", ")
	}
	sens := []string{}
	x := pl.Sensitive
	for _, s := range []struct {
		on   bool
		name string
	}{{x.HistoricalPersecution, "historicalPersecution"}, {x.PoliticalRepression, "politicalRepression"}, {x.DiscriminationDepiction, "discriminationDepiction"}, {x.HateSpeechDepiction, "hateSpeechDepiction"}} {
		if s.on {
			sens = append(sens, s.name)
		}
	}
	sensitive := "none"
	if len(sens) > 0 {
		sensitive = strings.Join(sens, ", ")
	}
	hard, review := reviewReasons(pl)
	result := "PASS"
	if len(hard) > 0 {
		result = "ERROR_HARD_RULE"
	} else if len(review) > 0 {
		result = "REVIEW_REQUIRED（" + strings.Join(review, "；") + "）"
	}
	return map[string]string{
		"ProfileName": pl.Content.Profile, "ProfileLabel": profileLabel[pl.Content.Profile], "SensitiveSummary": sensitive,
		"TableGameSummary": table, "TableAssets": fmt.Sprintf("%s / %s / %s / %s", t.Cards, t.Dice, t.ScoreTokens, t.WageringTokens), "VirtualWagering": t.VirtualWagering,
		"TokenSummary": fmt.Sprintf("%s；Free Token Sources: %s", t.TokenPersistence, src), "ValidationSummary": result,
	}
}

// askContentProfile 是精靈的內容邊界題目；custom 時逐項詢問（平台不收的值不列出）。
func askContentProfile(profile *string) (*contentProfile, error) {
	var err error
	if *profile == "" {
		if *profile, err = choose("內容邊界（開發期間的內容上限，不是正式分級；AI 不會因此主動加入這些內容）：", []option{
			{"general", profileLabel["general"], "休閒、益智、家庭與派對：輕度衝突、低恐怖、沒有血腥與成人題材"},
			{"moderate", profileLabel["moderate"], "中度戰鬥、恐怖、犯罪故事、少量血跡、酒吧場景、輕度粗話"},
			{"strong", profileLabel["strong"], "較強恐怖與戰鬥、較明顯血跡、成人故事題材、較強語言、非露骨性暗示"},
			{"custom", profileLabel["custom"], "自己逐項設定"},
		}, defaultProfile); err != nil {
			return nil, err
		}
	}
	c := presetProfile(*profile)
	if *profile != "custom" {
		return &c, nil
	}
	for _, d := range profileDims {
		lv := make([]option, 0, len(d.levels))
		for _, v := range d.levels {
			lv = append(lv, option{v, v, d.desc[v]})
		}
		if *d.val(&c), err = choose(d.title+"：", lv, *d.val(&c)); err != nil {
			return nil, err
		}
	}
	return &c, nil
}
