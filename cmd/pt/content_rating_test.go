package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// 內容分級：三種 profile 的設定與文件、非互動 custom 回錯誤、STYLE.md 不再寫死內容限制、
// PLAYTOGETHER.md 保留平台硬性規則、所有 AI 模式都有 CONTENT_RATING.md、PROJECT_CONTEXT 與 AI_WORKFLOW 的內容。
func TestContentRatingProfiles(t *testing.T) {
	requireEmbeddedSDK(t)
	fakeAI(t)
	dir := t.TempDir()
	t.Chdir(dir)

	want := map[string]struct{ age, violence, alcohol string }{
		"general": {"all", "none", "none"},
		"12plus":  {"12+", "mild", "none"},
		"15plus":  {"15+", "moderate", "allowed"},
	}
	for profile, w := range want {
		name := "r-" + profile
		must(t, cmdNew([]string{name, "-y", "-rating", profile}))
		r := readProject(t, filepath.Join(dir, name)).ContentRating
		if r == nil || r.Profile != profile || r.TargetAge != w.age || r.Violence != w.violence || r.Alcohol != w.alcohol || !r.Provisional ||
			r.Gambling.RealMoney || r.Gambling.Wagering || r.Gambling.PaidRandomReward || r.Gambling.Simulated != "none" || r.Gambling.ChanceMechanics != "allowed" {
			t.Fatalf("%s 的 contentRating：%+v", profile, r)
		}
		doc := readFile(t, dir, name, "docs/content/CONTENT_RATING.md")
		for _, s := range []string{"不等於正式分級結果", "最終分級依實際內容與平台審核決定", "上限，不是要求", "CONTENT_RATING_CHANGE_REQUEST",
			ratingLabel[profile], targetAgeLabel[w.age], "`paidRandomReward`", "玩聚平台硬性規則", "可以有未成年角色", "不得**性化未成年人"} {
			if !strings.Contains(doc, s) {
				t.Errorf("%s 的 CONTENT_RATING.md 少了 %q", profile, s)
			}
		}
		var fs []finding
		checkContentRating(r, func(level, format string, a ...any) { fs = append(fs, finding{level, format}) })
		if len(fs) != 0 {
			t.Errorf("%s 的預設應通過 pt check：%v", profile, fs)
		}
	}
	if err := cmdNew([]string{"r-custom", "-y", "-rating", "custom"}); err == nil || !strings.Contains(err.Error(), "逐項") {
		t.Fatalf("非互動的 custom 要回清楚的錯誤：%v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "r-custom")); err == nil {
		t.Fatal("custom 失敗時不能留下資料夾")
	}
	if err := cmdNew([]string{"r-18", "-y", "-rating", "18plus"}); err == nil || !strings.Contains(err.Error(), "15+") {
		t.Fatalf("不接受 15+ 以上的分級：%v", err)
	}

	style := readFile(t, dir, "r-15plus", "art/STYLE.md")
	for _, banned := range []string{"人物一律是成人", "血腥暴力", "酒類", "骰子"} {
		if strings.Contains(style, banned) {
			t.Errorf("STYLE.md 不該再寫死內容限制：%q", banned)
		}
	}
	if !strings.Contains(style, "CONTENT_RATING.md") || !strings.Contains(style, "PLAYTOGETHER.md") {
		t.Error("STYLE.md 要指向 PLAYTOGETHER.md 與 CONTENT_RATING.md")
	}
	pt := readFile(t, dir, "r-15plus", "PLAYTOGETHER.md")
	for _, s := range []string{"玩聚平台硬性規則", "不接受限制級（18+）", "真實金錢下注", "付費抽獎或轉蛋", "模擬賭博", "不得性化未成年人", "可辨識的真人",
		"15+ 方向 ≠ 可以真實金錢下注或模擬賭博", "docs/content/CONTENT_RATING.md"} {
		if !strings.Contains(pt, s) {
			t.Errorf("PLAYTOGETHER.md 少了平台硬性規則 %q", s)
		}
	}
	if strings.Contains(pt, "人物一律是成人") || strings.Contains(pt, "不可以有色情、血腥暴力、酒類") {
		t.Error("PLAYTOGETHER.md 的素材範例不該再寫死分級內容")
	}
}

func TestContentRatingAllModes(t *testing.T) {
	requireEmbeddedSDK(t)
	fakeAI(t)
	dir := t.TempDir()
	t.Chdir(dir)
	must(t, cmdNew([]string{"c-light", "-mode", "lightweight"}))
	must(t, cmdNew([]string{"c-std", "-mode", "standard"}))
	must(t, cmdNew([]string{"c-studio", "-mode", "studio"}))
	for _, p := range []string{"c-light", "c-std", "c-studio"} {
		readFile(t, dir, p, "docs/content/CONTENT_RATING.md")
		ctx := readFile(t, dir, p, "docs/ai/PROJECT_CONTEXT.md")
		if !strings.Contains(ctx, "Content Rating Profile") || !strings.Contains(ctx, "目標年齡（Target Age）") || !strings.Contains(ctx, "docs/content/CONTENT_RATING.md") {
			t.Errorf("%s 的 PROJECT_CONTEXT.md 要有內容分級與目標年齡", p)
		}
		wf := readFile(t, dir, p, "docs/ai/AI_WORKFLOW.md")
		if !strings.Contains(wf, "Content Rating Authority") || !strings.Contains(wf, "project-authoritative") || !strings.Contains(wf, "CONTENT_RATING_CHANGE_REQUEST") ||
			!strings.Contains(wf, "上限，不是要求") {
			t.Errorf("%s 的 AI_WORKFLOW.md 要有內容分級規則", p)
		}
	}
	if rules := readFile(t, dir, "c-studio", ".claude/rules/playtogether.md"); !strings.Contains(rules, "CONTENT_RATING.md") {
		t.Error("Studios 的規則也要指向 CONTENT_RATING.md")
	}
}

// pt check 只做結構驗證：合法的設定通過；不合法的值、平台硬性規則、超過目標年齡的內容邊界是錯誤；舊專案沒有分級只警告。
func TestCheckContentRating(t *testing.T) {
	run := func(r *contentRating) (errs, warns []string) {
		checkContentRating(r, func(level, format string, a ...any) {
			if level == "error" {
				errs = append(errs, format)
			} else {
				warns = append(warns, format)
			}
		})
		return
	}
	ok := ratingPreset("15plus")
	if e, w := run(&ok); len(e)+len(w) != 0 {
		t.Fatalf("合法的 15plus：%v %v", e, w)
	}
	if e, w := run(nil); len(e) != 0 || len(w) != 1 {
		t.Fatalf("舊專案沒有分級只警告：%v %v", e, w)
	}
	cases := map[string]func(r *contentRating){
		"不合法的 profile":     func(r *contentRating) { r.Profile = "mature" },
		"18+":              func(r *contentRating) { r.TargetAge = "18+" },
		"不合法的值":            func(r *contentRating) { r.Violence = "extreme" },
		"超過玩聚上限的血腥":        func(r *contentRating) { r.Blood = "strong" },
		"超過玩聚上限的性內容":       func(r *contentRating) { r.SexualContent = "moderate" },
		"realMoney":        func(r *contentRating) { r.Gambling.RealMoney = true },
		"wagering":         func(r *contentRating) { r.Gambling.Wagering = true },
		"paidRandomReward": func(r *contentRating) { r.Gambling.PaidRandomReward = true },
		"模擬賭博":             func(r *contentRating) { r.Gambling.Simulated = "allowed" },
		"超過目標年齡的內容邊界（12+ 的酒類）":   func(r *contentRating) { r.Profile, r.TargetAge = "custom", "12+" },
		"provisional 缺少或是 false": func(r *contentRating) { r.Provisional = false },
		"profile 和 targetAge 不符": func(r *contentRating) { r.Profile = "general" },
	}
	for name, mut := range cases {
		r := ratingPreset("15plus")
		mut(&r)
		if e, _ := run(&r); len(e) == 0 {
			t.Errorf("%s 要回錯誤", name)
		}
	}
	// 手動改過的預設 profile（targetAge 不變）：警告（不是錯誤）
	r := ratingPreset("12plus")
	r.Violence = "moderate"
	if e, w := run(&r); len(e) != 0 || len(w) != 1 {
		t.Fatalf("手動修改預設只警告：%v %v", e, w)
	}
}

// CONTENT_RATING.md 要和 playtogether.json 一致：剛建立時一致；只改 JSON 沒更新文件時提醒。
func TestContentRatingDocInSync(t *testing.T) {
	requireEmbeddedSDK(t)
	dir := t.TempDir()
	t.Chdir(dir)
	must(t, cmdNew([]string{"sync-game", "-y", "-rating", "12plus"}))
	p := readProject(t, filepath.Join(dir, "sync-game"))
	p.dir = filepath.Join(dir, "sync-game")
	var warns []string
	add := func(level, format string, a ...any) { warns = append(warns, format) }
	checkContentRatingDoc(&p, add)
	if len(warns) != 0 {
		t.Fatalf("剛建立時應一致：%v", warns)
	}
	p.ContentRating.Violence = "moderate"
	checkContentRatingDoc(&p, add)
	if len(warns) != 1 || !strings.Contains(warns[0], "不一致") {
		t.Fatalf("改了分級沒更新文件要提醒：%v", warns)
	}
}
