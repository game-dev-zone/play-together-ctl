---
name: playtogether-new
description: 在空資料夾開一款新的玩聚（PlayTogether）遊戲：先和開發者確認玩法與幾個設定（多人遊戲的伺服器建議交給玩聚代管），用 pt new . 在目前資料夾建立專案，讀專案裡的平台規則，然後做出第一個可以在本機試玩的原型。使用者說「幫我做一款玩聚遊戲」「開新遊戲」「pt new」「我想做一個…遊戲」而且目前資料夾還沒有遊戲時使用；已經有遊戲要上架用 playtogether-publish。Use when starting a new PlayTogether game in an empty folder.
---

# 開一款新的玩聚遊戲

這個 skill 由 `pt`（玩聚開發者工具）提供，用 `pt skills install` 安裝。目標：開發者在空資料夾打開你，說想做什麼遊戲，你建立專案、做出可以試玩的原型。

**原則**

- 遊戲設計（玩法、人數、故事、內容邊界）由開發者決定。不確定的地方**一次問完**，不要每一步都問。
- 每一步做完用一兩句話說你做了什麼。
- 不要讀出、印出、複製或提交私鑰（`keys/`、`*.key`）。
- 每個 `pt` 指令都可以用 `pt <指令> -h` 看參數。看不懂錯誤訊息時先讀它，不要猜。

## 0. 準備

1. `pt version`：確認有安裝。沒有時請開發者照文件站「安裝 pt」安裝，然後停下來。
2. **確認目前資料夾**：`playtogether.json` 已經存在，表示專案已經建好，跳到第 3 步。資料夾裡已經有遊戲程式（不是只有 `.git` 等隱藏檔）時，不要用這個 skill：要上架就改用 playtogether-publish（`pt init`）；要另外開新遊戲，請開發者換一個空資料夾。

## 1. 問開發者（一次問完）

先根據開發者已經說的內容整理，只問還不知道的。給建議值，開發者說「都可以」就用建議值：

| 項目 | 建議值 | 對應的參數 |
| --- | --- | --- |
| 玩法一句話：幾個人、怎麼贏、一局多久 | 玩聚以直播主開房、觀眾用手機加入為主：**手機直接點就能玩、一局 3–5 分鐘**最適合 | （寫進專案說明） |
| 單人還是多人 | 多人 | 多人 `-game-server playtogether-managed`；單人 `-game-server none` |
| 多人遊戲的伺服器放在哪裡 | **交給玩聚代管**：開發者不用自己找主機、不用付主機費，上傳時伺服器和畫面一起交給玩聚（目前只支援 Go、要平台開通） | 代管 `-server go -hosting playtogether`；自己部署 `-server go` 或 `-server node`，不加 `-hosting` |
| 畫面用什麼做 | 網頁（HTML／JavaScript）；要用 Godot 編輯器時選 Godot | `-client web` 或 `-client godot` |
| 手機直式還是橫式 | 直式 | `-orientation portrait`、`landscape`、`any` |
| 內容邊界 | `general`（普遍級）。有明顯暴力、恐怖等內容時由開發者決定 `moderate` 或 `strong`，你不能自己提高 | `-profile` |
| 有沒有牌桌玩法（撲克、麻將、骰子等） | 沒有 | `-table-game`，有的話照實填（需要平台審核） |
| 平台網址 | 開發者給過就用；沒有時問「要用哪個玩聚環境？請貼文件站的網址」 | `-platform`（結尾不加斜線） |

不適合直接做的類型（即時動作、物理碰撞、射擊、賽車）：先說明玩聚的遊戲伺服器是每 0.5 秒同步的回合／派對型，建議改成回合制或「各玩各的、伺服器比結果」，由開發者決定。

## 2. 建立專案（`pt new .`）

在目前的資料夾建立，**不要**另外建子資料夾：

```
pt new . -y -platform <平台網址> -client web -game-server playtogether-managed -server go -hosting playtogether -orientation portrait -profile general -table-game none
```

- 專案名稱預設用資料夾名稱。不符合格式（小寫英文、數字、連字號，英文開頭）時加 `-name <名稱>`。
- `-y` 不詢問，所以每個參數都要明確給。
- `pt new .` 只能在空資料夾用（`.git`、`.claude` 等隱藏檔可以有）。

讀 `pt new` 的輸出。有「需要玩聚平台人工審核」的項目時告訴開發者。

## 3. 讀專案的規則（一定要做）

`pt new` 寫入的規則檔，這次對話一開始還不存在，所以**現在逐一讀完並照著做**：

- `CLAUDE.md`（Codex：`AGENTS.md`）、`PLAYTOGETHER.md`：玩聚平台規則與硬性規則。
- `docs/ai/AI_WORKFLOW.md`、`docs/ai/PROJECT_CONTEXT.md`：工作流程與專案說明。
- `docs/content/CONTENT_PROFILE.md`、`docs/content/GAMEPLAY_ECONOMY.md`：內容邊界、玩法與經濟宣告。
- 範本程式：`server/main.go`（只有遊戲規則，平台接口在 SDK 套件裡）與畫面（`web/` 或 `game/`）。

告訴開發者：之後重新打開 Claude Code 時，這些規則會自動載入。

## 4. 做出第一個可以試玩的原型

照 `docs/ai/AI_WORKFLOW.md` 做。重點：

- **伺服器決定遊戲狀態**：分數、勝負、輪到誰、計時都在 `server/`，畫面只送操作、照伺服器的狀態顯示。
- 先做最小可玩的版本（一局能從開始玩到結束），不要一次做完所有功能。
- 做完執行 `make verify`（伺服器測試、`pt dev -smoke`、`pt verify server`、`pt check -local`），有錯就修到通過。
- 請開發者自己在終端機執行 `pt dev` 試玩：它會打開玩家分頁與控制面板，可以模擬踢人、重新開始、斷線。你在對話裡執行會一直佔著。

## 5. 接下來

告訴開發者：

- 試玩後說哪裡要改，一次改一件事。
- 想讓更多人一起測：上傳後用 `pt preview` 開預覽房間，用手機實測。
- 準備好了就說「幫我上架到玩聚」（playtogether-publish）。
