package main

// v0.14 的內容分級（contentRating）：只用來讀取舊專案（v0.15 改成 policy.go 的 platformAudience、contentProfile、
// sensitiveContent、gameplay、economy）。舊專案依 v0.14 的規則驗證，再由 legacyPolicy 對應到 v0.15；
// 這裡的 ratingDims 也是 v0.15 內容維度的說明來源。

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

type contentRating struct {
	Profile       string         `json:"profile"`     // general、12plus、15plus、custom
	TargetAge     string         `json:"targetAge"`   // all、6+、12+、15+（開發期間的目標年齡）
	Provisional   bool           `json:"provisional"` // 一律 true：開發期間的內容邊界，不是正式分級結果
	Violence      string         `json:"violence"`
	Blood         string         `json:"blood"`
	Horror        string         `json:"horror"`
	SexualContent string         `json:"sexualContent"`
	Language      string         `json:"language"`
	Alcohol       string         `json:"alcohol"`
	Tobacco       string         `json:"tobacco"`
	Gambling      gamblingRating `json:"gambling"`
}

type gamblingRating struct {
	RealMoney        bool   `json:"realMoney"`        // 審核準則 1.3：由玩聚平台與律師決定（需要審核）
	Wagering         bool   `json:"wagering"`         // 審核準則 1.3：由玩聚平台與律師決定（需要審核）
	PaidRandomReward bool   `json:"paidRandomReward"` // 付費抽獎、轉蛋：由玩聚平台與律師決定（需要審核）
	Simulated        string `json:"simulated"`        // 模擬賭博：none 以外需要玩聚平台審核（v0.15 改用 gameplay.tableGame 宣告）
	ChanceMechanics  string `json:"chanceMechanics"`  // 骰子、桌遊牌具、隨機事件、隨機掉落：none、allowed
}

var (
	ratingProfiles = []string{"general", "12plus", "15plus", "custom"}
	targetAges     = []string{"all", "6+", "12+", "15+"}
	ratingLabel    = map[string]string{"general": "General／普遍向", "12plus": "12+／輔12方向", "15plus": "15+／輔15方向", "custom": "Custom／自訂"}
	targetAgeLabel = map[string]string{"all": "全年齡（普遍級方向）", "6+": "6+（保護級方向）", "12+": "12+（輔12方向）", "15+": "15+（輔15方向）"}
)

// ratingDim 是一個內容維度：可以設定的值（由寬鬆到嚴格的反向，index 越大越強）與說明。
type ratingDim struct {
	key, title string
	levels     []string
	desc       map[string]string
	never      string // 任何分級都不行（玩聚平台政策）
	get        func(*contentRating) *string
	// beyond：玩聚不收的值（規則資料 policy/rules.json 回報為平台硬性規則）
	beyond []string
}

var ratingDims = []ratingDim{
	{key: "violence", title: "暴力", levels: []string{"none", "mild", "moderate", "strong"},
		desc: map[string]string{"none": "沒有暴力；衝突用比賽、解謎、搶答等方式呈現", "mild": "卡通或幻想風格的輕度打鬥，沒有受傷細節",
			"moderate": "非寫實或中度的戰鬥、角色之間的衝突；受傷不強調細節", "strong": "較強烈的戰鬥與衝突，但不過度強調殘忍或寫實的傷害細節"},
		never: "以殘忍、虐待為賣點的寫實暴力", get: func(r *contentRating) *string { return &r.Violence }},
	{key: "blood", title: "血腥", levels: []string{"none", "mild", "moderate"},
		desc:  map[string]string{"none": "不出現血", "mild": "少量血跡或非寫實的血", "moderate": "中度血跡（例如犯罪現場），不強調傷口細節"},
		never: "大量寫實血腥、肢解（玩聚不收限制級內容）", beyond: []string{"strong"}, get: func(r *contentRating) *string { return &r.Blood }},
	{key: "horror", title: "恐怖", levels: []string{"none", "mild", "moderate", "strong"},
		desc: map[string]string{"none": "沒有恐怖元素", "mild": "輕度驚嚇、可愛或幽默風格的鬼怪",
			"moderate": "緊張氣氛、怪物、輕度的突然驚嚇", "strong": "心理恐怖、突然驚嚇（jumpscare）、屍體、犯罪現場"},
		get: func(r *contentRating) *string { return &r.Horror }},
	{key: "sexualContent", title: "性內容", levels: []string{"none", "suggestive"},
		desc:  map[string]string{"none": "不加入性暗示或色情內容", "suggestive": "成人角色之間非露骨的性暗示（例如調情的台詞），沒有裸露"},
		never: "露骨的性內容、裸露（玩聚不收限制級內容）；任何涉及未成年人的性化內容", beyond: []string{"moderate", "explicit"},
		get: func(r *contentRating) *string { return &r.SexualContent }},
	{key: "language", title: "粗俗語言", levels: []string{"none", "mild", "strong"},
		desc:  map[string]string{"none": "不使用粗話", "mild": "少量輕度的粗話", "strong": "粗俗語言"},
		never: "仇恨、歧視、騷擾的字眼", get: func(r *contentRating) *string { return &r.Language }},
	{key: "alcohol", title: "酒類", levels: []string{"none", "allowed"},
		desc:  map[string]string{"none": "不出現酒類", "allowed": "可以作為成人角色的故事、場景或角色元素"},
		never: "真實的酒類品牌；鼓勵未成年人飲酒", get: func(r *contentRating) *string { return &r.Alcohol }},
	{key: "tobacco", title: "菸", levels: []string{"none", "allowed"},
		desc:  map[string]string{"none": "不出現菸", "allowed": "可以作為成人角色的故事、場景或角色元素"},
		never: "真實的菸品品牌；鼓勵未成年人吸菸", get: func(r *contentRating) *string { return &r.Tobacco }},
	{key: "gambling.chanceMechanics", title: "隨機機制", levels: []string{"none", "allowed"},
		desc: map[string]string{"none": "不使用骰子、抽牌、隨機事件", "allowed": "骰子、桌遊牌具、隨機事件、隨機掉落等一般遊戲機制（不涉及金錢或財物）"},
		get:  func(r *contentRating) *string { return &r.Gambling.ChanceMechanics }},
}

// ratingCaps 是各目標年齡的內容邊界上限（開發期間的草案，不是法律分級判定；最終分級依實際內容與玩聚審核）。
var ratingCaps = map[string]map[string]string{
	"all": {"violence": "none", "blood": "none", "horror": "none", "sexualContent": "none", "language": "none", "alcohol": "none", "tobacco": "none", "gambling.chanceMechanics": "allowed"},
	"6+":  {"violence": "mild", "blood": "none", "horror": "mild", "sexualContent": "none", "language": "none", "alcohol": "none", "tobacco": "none", "gambling.chanceMechanics": "allowed"},
	"12+": {"violence": "moderate", "blood": "mild", "horror": "moderate", "sexualContent": "none", "language": "mild", "alcohol": "none", "tobacco": "none", "gambling.chanceMechanics": "allowed"},
	"15+": {"violence": "strong", "blood": "moderate", "horror": "strong", "sexualContent": "suggestive", "language": "strong", "alcohol": "allowed", "tobacco": "allowed", "gambling.chanceMechanics": "allowed"},
}

// ratingPreset 回傳各 profile 的預設（custom 從 15plus 開始逐項調整）。預設值保守：上限內的較低值，開發者要明確打開。
func ratingPreset(profile string) contentRating {
	r := contentRating{Profile: profile, Provisional: true, Violence: "none", Blood: "none", Horror: "none", SexualContent: "none",
		Language: "none", Alcohol: "none", Tobacco: "none",
		Gambling: gamblingRating{Simulated: "none", ChanceMechanics: "allowed"}}
	switch profile {
	case "general":
		r.TargetAge = "all"
	case "12plus":
		r.TargetAge, r.Violence, r.Horror, r.Language = "12+", "mild", "mild", "mild"
	default: // 15plus、custom
		r.TargetAge, r.Violence, r.Blood, r.Horror, r.Language, r.Alcohol = "15+", "moderate", "mild", "strong", "mild", "allowed"
	}
	return r
}

func levelIndex(d ratingDim, v string) int { return slices.Index(d.levels, v) }

// checkContentRating 依 v0.14 的規則驗證舊格式（值、目標年齡的內容邊界、profile 與 targetAge 一致）。
// 平台不收的內容值、真錢下注、付費抽獎由規則資料回報（legacyPolicy 對應後），這裡不重複。
func checkContentRating(r *contentRating, add addFunc) {
	if !slices.Contains(ratingProfiles, r.Profile) {
		add(levelSchema, "contentRating.profile 只能是 general、12plus、15plus 或 custom（目前是 %q）", r.Profile)
	}
	capOK := slices.Contains(targetAges, r.TargetAge)
	if !capOK {
		add(levelSchema, "contentRating.targetAge 只能是 all、6+、12+ 或 15+（目前是 %q）", r.TargetAge)
	}
	for _, d := range ratingDims {
		v := *d.get(r)
		if slices.Contains(d.beyond, v) {
			continue
		}
		if levelIndex(d, v) < 0 {
			add(levelSchema, "contentRating.%s 只能是 %s（目前是 %q）", d.key, strings.Join(d.levels, "、"), v)
			continue
		}
		if capOK {
			if limit := ratingCaps[r.TargetAge][d.key]; levelIndex(d, v) > levelIndex(d, limit) {
				add(levelSchema, "contentRating.%s 是 %q，超過目標年齡 %s 的內容邊界（最多 %q）：調高 targetAge 或降低這一項（或改用 v0.15 的 contentProfile）", d.key, v, r.TargetAge, limit)
			}
		}
	}
	if !r.Provisional {
		add(levelSchema, "contentRating.provisional 必須是 true：這是開發期間的內容邊界，不是正式分級結果")
	}
	if r.Profile != "custom" && slices.Contains(ratingProfiles, r.Profile) {
		p := ratingPreset(r.Profile)
		if capOK && r.TargetAge != p.TargetAge {
			add(levelSchema, "contentRating.targetAge 是 %q，和 profile %s 的 %q 不符：改用 profile custom，或把 targetAge 改回 %q", r.TargetAge, r.Profile, p.TargetAge, p.TargetAge)
		} else if !sameRating(p, *r) {
			add(levelWarn, "contentRating 和 %s 的預設不同（已手動修改）：確認 profile 與 targetAge，或把 profile 改成 custom", r.Profile)
		}
	}
}

func sameRating(a, b contentRating) bool {
	a.Profile, b.Profile, a.Provisional, b.Provisional = "", "", false, false
	return a == b
}

// checkContentRatingDoc：docs/content/CONTENT_RATING.md 要和 playtogether.json 的 contentRating 一致（AI 讀的是文件）。
func checkContentRatingDoc(p *project, add addFunc) {
	if p.ContentRating == nil || p.dir == "" {
		return
	}
	got, err := os.ReadFile(filepath.Join(p.dir, "docs", "content", "CONTENT_RATING.md"))
	if err != nil {
		add(levelWarn, "沒有 docs/content/CONTENT_RATING.md：AI 需要它才知道內容邊界（用 pt new 以相同設定建暫存專案複製過來）")
		return
	}
	if string(got) != contentRatingDoc(p.Name, *p.ContentRating) {
		add(levelWarn, "docs/content/CONTENT_RATING.md 和 playtogether.json 的 contentRating 不一致：改了分級後要一起更新文件（AI 讀的是文件）")
	}
}

// contentRatingDoc 產生 docs/content/CONTENT_RATING.md（依 playtogether.json 的 contentRating）。
func contentRatingDoc(name string, r contentRating) string {
	var b strings.Builder
	w := func(format string, a ...any) { fmt.Fprintf(&b, format, a...) }
	w("# 內容分級（%s）\n\n", name)
	w("> **這是開發期間的內容邊界／目標分級，不等於正式分級結果；最終分級依實際內容與平台審核決定。**\n")
	w("> 設定在 `playtogether.json` 的 `contentRating`，這份文件依它產生。\n\n")
	w("| 項目 | 內容 |\n| --- | --- |\n")
	w("| Profile | %s（`%s`） |\n", ratingLabel[r.Profile], r.Profile)
	w("| 目標年齡 | %s |\n\n", targetAgeLabel[r.TargetAge])
	w("## 怎麼使用\n\n")
	w("- 這是**上限，不是要求**：AI 不會因為上限較寬就主動加入暴力、血腥、恐怖、酒類、菸、性暗示、粗俗語言或隨機機制；只在遊戲設計需要、開發者要求時才使用，而且不超過下面的等級。\n")
	w("- 要加入超過目前等級的內容時，AI 不直接改分級，提出 `CONTENT_RATING_CHANGE_REQUEST`（格式在 `docs/ai/AI_WORKFLOW.md`），開發者同意後由開發者修改 `playtogether.json` 並更新這份文件。\n")
	w("- **玩聚平台硬性規則**（`PLAYTOGETHER.md`）永遠優先，任何分級都不能覆蓋。\n")
	w("- 上架時，玩聚登記的遊戲分級要和實際內容相符；`pt check` 只檢查這裡的設定格式與平台上限，不判斷遊戲實際內容。\n\n")
	for _, d := range ratingDims {
		v := *d.get(&r)
		w("## %s\n\n等級：`%s`\n\n", d.title, v)
		if levelIndex(d, v) == 0 {
			w("目前：%s。\n", d.desc[v])
		} else {
			w("允許：%s。\n", d.desc[v])
		}
		if i := levelIndex(d, v); i >= 0 && i+1 < len(d.levels) {
			above := d.levels[i+1:]
			parts := make([]string, 0, len(above))
			for _, a := range above {
				parts = append(parts, d.desc[a])
			}
			w("\n不加入（超過目前等級）：%s。\n", strings.Join(parts, "；"))
		}
		if d.never != "" {
			w("\n任何等級都不行：%s。\n", d.never)
		}
		w("\n")
	}
	w("## 賭博與付費\n\n")
	w("| 項目 | 設定 |\n| --- | --- |\n")
	w("| 真實金錢下注（`realMoney`） | `false`（由玩聚平台與律師決定；寫 true 是需要審核） |\n")
	w("| 下注、押注（`wagering`） | `false`（由玩聚平台與律師決定；寫 true 是需要審核） |\n")
	w("| 付費抽獎、轉蛋（`paidRandomReward`） | `false`（由玩聚平台與律師決定；寫 true 是需要審核） |\n")
	w("| 模擬賭博：模擬賭場、下注畫面、籌碼（`simulated`） | `none`（其他值需要玩聚平台審核） |\n")
	w("| 一般隨機機制：骰子、桌遊牌具、隨機事件、隨機掉落（`chanceMechanics`） | `%s` |\n\n", r.Gambling.ChanceMechanics)
	w("文案不要出現賭博、博弈字眼。牌具只作為遊戲本身的玩法，不搭配籌碼、金錢。\n\n")
	w("## 未成年角色\n\n")
	w("可以有未成年角色（例如校園、成長故事、家庭劇情、青少年運動）。任何分級都**不得**性化未成年人，未成年人也不得出現在任何性內容，或其他玩聚禁止的未成年人內容。\n")
	return b.String()
}
