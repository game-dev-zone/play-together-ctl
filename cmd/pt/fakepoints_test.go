package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/game-dev-zone/play-together-ctl/internal/playtogether"
)

// 本機假平台的點數下注：宣告、平台核准、三種流程（含機台型）、重送、上限、牌桌守恆、玩家與房間狀態。
func TestFakePoints(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	fp := newFakePlatform(pub)
	srv := httptest.NewServer(fp.handler(nil))
	t.Cleanup(srv.Close)
	c := &playtogether.Client{BaseURL: srv.URL, KeyID: fakeKeyID, PrivateKey: priv}
	ctx := context.Background()
	apiErr := func(err error) (int, string) {
		var e *playtogether.APIError
		if errors.As(err, &e) {
			return e.Status, e.Code
		}
		if err != nil {
			return 0, err.Error()
		}
		return 200, ""
	}
	expect := func(what string, err error, status int, code string) {
		t.Helper()
		if s, c := apiErr(err); s != status || c != code {
			t.Fatalf("%s：要 %d %s，結果 %d %s", what, status, code, s, c)
		}
	}
	join := func(code string) *fakePlayer {
		lc, p, _, err := fp.issue(code)
		must(t, err)
		_, err = c.ExchangeLaunchCode(ctx, lc)
		must(t, err)
		return p
	}
	p1, p2, p3 := join("p1"), join("p2"), join("p3")
	roomID := fp.currentRoom()
	bet := func(p *fakePlayer, id string, stake, payout int64) playtogether.Bet {
		return playtogether.Bet{RoomID: roomID, PlayerID: p.ID, BetID: id, RoundID: "round-1", Stake: stake, Payout: payout}
	}

	// 沒有宣告 economy.platformPointsStake：一律 403 declaration_required
	_, err := c.GetPointsBalance(ctx, roomID, p1.ID)
	expect("沒有宣告", err, 403, "declaration_required")
	_, err = c.SettleBet(ctx, bet(p1, "b0", 10, 0))
	expect("沒有宣告", err, 403, "declaration_required")
	fp.pointsStake = true

	b, err := c.GetPointsBalance(ctx, roomID, p1.ID)
	must(t, err)
	if b.Balance != fakePointsStart || !b.Enabled || b.StakeLeftToday != fakeDailyStakeCap || b.Epoch != 0 || b.MaxStake != 100 || b.MaxPayout != 1000 || b.MaxBuyIn != 300 || !b.TestPoints {
		t.Fatalf("餘額：%+v", b)
	}

	// 一次結算＋重送＋內容不同
	r, err := c.SettleBet(ctx, bet(p1, "b1", 10, 30))
	must(t, err)
	if r.Status != "settled" || r.BeforeBalance != 1000 || r.Balance != 1020 || r.Payout == nil || *r.Payout != 30 || r.Replayed || !r.TestPoints {
		t.Fatalf("一次結算：%+v", r)
	}
	r, err = c.SettleBet(ctx, bet(p1, "b1", 10, 30))
	must(t, err)
	if !r.Replayed || r.Balance != 1020 || r.BeforeBalance != 1000 {
		t.Fatalf("重送要回傳原結果：%+v", r)
	}
	_, err = c.SettleBet(ctx, bet(p1, "b1", 10, 40))
	expect("同一張注單不同內容", err, 409, "idempotency_conflict")
	_, err = c.SettleBet(ctx, bet(p1, "b-big", 101, 0))
	expect("單筆下注上限", err, 400, "bad_amount")
	_, err = c.SettleBet(ctx, bet(p1, "b-big", 1, 1001))
	expect("單筆派彩上限", err, 400, "bad_amount")
	old := bet(p1, "b-old", 1, 0)
	old.Epoch = 1
	_, err = c.SettleBet(ctx, old)
	expect("局數不對", err, 409, "epoch_mismatch")

	// 先扣後派
	r, err = c.StakeBet(ctx, bet(p1, "b2", 20, 0))
	must(t, err)
	if r.Status != "staked" || r.BeforeBalance != 1020 || r.Balance != 1000 || r.Payout != nil {
		t.Fatalf("先扣：%+v", r)
	}
	r, err = c.PayoffBet(ctx, "b2", 50, "win")
	must(t, err)
	if r.Status != "settled" || r.BeforeBalance != 1000 || r.Balance != 1050 {
		t.Fatalf("派彩：%+v", r)
	}
	r, err = c.PayoffBet(ctx, "b2", 50, "win")
	must(t, err)
	if !r.Replayed || r.Balance != 1050 {
		t.Fatalf("派彩重送：%+v", r)
	}
	r, err = c.StakeBet(ctx, bet(p1, "b2", 20, 0))
	must(t, err)
	if !r.Replayed || r.Status != "staked" || r.Balance != 1000 {
		t.Fatalf("派彩後重送下注，要回傳下注當時的回執：%+v", r)
	}
	_, err = c.PayoffBet(ctx, "b2", 60, "")
	expect("已經派彩", err, 409, "bet_settled")
	_, err = c.CancelBet(ctx, "b2")
	expect("已經派彩不能取消", err, 409, "bet_settled")
	_, err = c.PayoffBet(ctx, "b1", 0, "")
	expect("一次結算的注單不能派彩", err, 409, "bet_settled")

	_, err = c.StakeBet(ctx, bet(p1, "b3", 20, 0))
	must(t, err)
	r, err = c.CancelBet(ctx, "b3")
	must(t, err)
	if r.Status != "cancelled" || r.BeforeBalance != 1030 || r.Balance != 1050 {
		t.Fatalf("取消：%+v", r)
	}
	r, err = c.CancelBet(ctx, "b3")
	must(t, err)
	if !r.Replayed {
		t.Fatalf("取消重送：%+v", r)
	}
	_, err = c.PayoffBet(ctx, "b3", 0, "")
	expect("已經取消", err, 409, "bet_cancelled")
	_, err = c.CancelBet(ctx, "nope")
	expect("沒有這張注單", err, 404, "bet_not_found")

	// 牌桌（玩家之間）：守恆、每日上限（p1 今天 10+20=30，取消的不算）
	tr, err := c.TableBuyIn(ctx, playtogether.TableBuyIn{RoomID: roomID, PlayerID: p1.ID, TableID: "T1", SessionID: "s1", Amount: 200})
	must(t, err)
	if tr.Status != "seated" || tr.BuyIn != 200 || tr.BeforeBalance != 1050 || tr.Balance != 850 || tr.House {
		t.Fatalf("入座：%+v", tr)
	}
	_, err = c.TableBuyIn(ctx, playtogether.TableBuyIn{RoomID: roomID, PlayerID: p1.ID, TableID: "T1", SessionID: "s1", Amount: 100})
	expect("同一次入座不同內容", err, 409, "idempotency_conflict")
	_, err = c.TableBuyIn(ctx, playtogether.TableBuyIn{RoomID: roomID, PlayerID: p1.ID, TableID: "T9", SessionID: "s-limit", Amount: 71})
	expect("每日上限（30＋200＋71）", err, 409, "daily_limit")
	_, err = c.TableBuyIn(ctx, playtogether.TableBuyIn{RoomID: roomID, PlayerID: p1.ID, TableID: "T9", SessionID: "s-big", Amount: 301})
	expect("單次入座上限", err, 400, "bad_amount")
	_, err = c.TableBuyIn(ctx, playtogether.TableBuyIn{RoomID: roomID, PlayerID: p2.ID, TableID: "T1", SessionID: "s2", Amount: 100})
	must(t, err)
	_, err = c.TableBuyIn(ctx, playtogether.TableBuyIn{RoomID: roomID, PlayerID: p2.ID, TableID: "T1", SessionID: "s-mix", Amount: 10, House: true})
	expect("同一張桌混用機台型", err, 409, "table_mode_mismatch")
	tr, err = c.TableCashOut(ctx, "s1", 250)
	must(t, err)
	if tr.Status != "left" || tr.CashOut == nil || *tr.CashOut != 250 || tr.Balance != 1100 {
		t.Fatalf("轉出：%+v", tr)
	}
	_, err = c.TableCashOut(ctx, "s2", 51)
	expect("轉出超過桌上剩下的", err, 409, "table_overdraw")
	_, err = c.TableCashOut(ctx, "s2", 50)
	must(t, err)
	tr, err = c.TableCashOut(ctx, "s2", 50)
	must(t, err)
	if !tr.Replayed {
		t.Fatalf("轉出重送：%+v", tr)
	}
	_, err = c.TableCashOut(ctx, "s2", 0)
	expect("已經轉出", err, 409, "session_closed")
	_, err = c.TableCashOut(ctx, "nope", 0)
	expect("沒有這次入座", err, 404, "session_not_found")

	// 機台型：轉出可以多於轉入（一次最多淨贏 1,000），淨贏算進每日派彩上限（p2 今天 100）
	tr, err = c.TableBuyIn(ctx, playtogether.TableBuyIn{RoomID: roomID, PlayerID: p2.ID, TableID: "fish", SessionID: "h1", Amount: 100, House: true})
	must(t, err)
	if !tr.House || tr.Balance != 850 {
		t.Fatalf("機台轉入：%+v", tr)
	}
	_, err = c.TableCashOut(ctx, "h1", 1101)
	expect("機台淨贏超過 1,000", err, 400, "bad_amount")
	tr, err = c.TableCashOut(ctx, "h1", 1100)
	must(t, err)
	if !tr.House || tr.Balance != 1950 {
		t.Fatalf("機台轉出：%+v", tr)
	}
	for i := range 4 { // 派彩 1000 × 4 ＋ 機台淨贏 1000 ＝ 5000
		_, err = c.SettleBet(ctx, bet(p2, fmt.Sprintf("big-%d", i), 1, 1000))
		must(t, err)
	}
	_, err = c.SettleBet(ctx, bet(p2, "big-x", 1, 1))
	expect("每日派彩上限", err, 409, "game_payout_limit")
	_, err = c.SettleBet(ctx, bet(p2, "big-0", 1, 0))
	expect("上限擋下的不影響重送比對", err, 409, "idempotency_conflict")

	// 點數不夠、沒有開啟點數玩法
	fp.mu.Lock()
	fp.pointsLocked().balance["p3"] = 5
	fp.mu.Unlock()
	_, err = c.SettleBet(ctx, bet(p3, "p3-1", 10, 0))
	expect("點數不夠", err, 402, "insufficient_test_points")
	must(t, fp.pointsControl("points-grant", "p3", ""))
	_, err = c.SettleBet(ctx, bet(p3, "p3-1", 10, 0))
	must(t, err)
	must(t, fp.pointsControl("points-toggle", "p3", ""))
	_, err = c.StakeBet(ctx, bet(p3, "p3-2", 10, 0))
	expect("沒有開啟點數玩法", err, 409, "points_not_enabled")
	b, err = c.GetPointsBalance(ctx, roomID, p3.ID)
	must(t, err)
	if b.Enabled || b.StakeLeftToday != 0 || b.Balance != 495 {
		t.Fatalf("沒有開啟時的餘額：%+v", b)
	}

	// 平台核准：未核准、停權只擋新的操作；已經成立的注單與入座照樣結束，重送照樣回傳
	_, err = c.StakeBet(ctx, bet(p1, "b4", 10, 0))
	must(t, err)
	_, err = c.TableBuyIn(ctx, playtogether.TableBuyIn{RoomID: roomID, PlayerID: p1.ID, TableID: "T2", SessionID: "s5", Amount: 20})
	must(t, err)
	must(t, fp.pointsControl("points-approval", "", "pending"))
	_, err = c.GetPointsBalance(ctx, roomID, p1.ID)
	expect("未核准查餘額", err, 403, "points_not_approved")
	_, err = c.SettleBet(ctx, bet(p1, "b5", 10, 0))
	expect("未核准結算", err, 403, "points_not_approved")
	_, err = c.SettleBet(ctx, bet(p1, "b1", 10, 30))
	must(t, err)
	must(t, fp.pointsControl("points-approval", "", "suspended"))
	_, err = c.StakeBet(ctx, bet(p1, "b5", 10, 0))
	expect("停權先扣", err, 403, "points_suspended")
	_, err = c.TableBuyIn(ctx, playtogether.TableBuyIn{RoomID: roomID, PlayerID: p1.ID, TableID: "T2", SessionID: "s6", Amount: 20})
	expect("停權入座", err, 403, "points_suspended")
	_, err = c.PayoffBet(ctx, "b4", 0, "")
	must(t, err)
	_, err = c.TableCashOut(ctx, "s5", 20)
	must(t, err)
	if fp.pointsControl("points-approval", "", "maybe") == nil {
		t.Fatal("不認得的核准狀態要拒絕")
	}
	must(t, fp.pointsControl("points-approval", "", "approved"))

	// 被踢、關房：不能下注；已經成立的注單仍然可以派彩
	_, err = c.StakeBet(ctx, bet(p1, "b6", 10, 0))
	must(t, err)
	fp.kick("p2")
	_, err = c.GetPointsBalance(ctx, roomID, p2.ID)
	expect("被踢的人", err, 409, "player_not_in_room")
	fp.closeRoom()
	_, err = c.SettleBet(ctx, bet(p1, "b7", 1, 0))
	expect("房間關閉", err, 409, "room_closed")
	_, err = c.PayoffBet(ctx, "b6", 5, "")
	must(t, err)
	_, err = c.GetPointsBalance(ctx, "r_"+"AAAAAAAAAAAAAAAAAAAAAA", p1.ID)
	expect("沒有這個房間", err, 404, "room_not_found")

	// 控制面板的狀態
	st := fp.state(0).Points
	if !st.Declared || st.Approval != "approved" || len(st.Players) != 3 || len(st.Txs) == 0 || len(st.Open) != 0 {
		t.Fatalf("面板狀態：%+v", st)
	}
	if st.Players[0].Code != "p1" || st.Players[0].StakeToday != 30+200+10+20+10 {
		t.Fatalf("面板的玩家：%+v", st.Players[0])
	}
}

func TestDevPointsStake(t *testing.T) {
	for raw, want := range map[string]bool{"": false, `{"platformPointsStake": false}`: false, `{"realMoney": false}`: false, `{"platformPointsStake": true}`: true} {
		p := &project{}
		if raw != "" {
			p.Economy = []byte(raw)
		}
		if got := devPointsStake(p); got != want {
			t.Errorf("%s：%v", raw, got)
		}
	}
	if devPointsStake(nil) {
		t.Error("nil 專案")
	}
}
