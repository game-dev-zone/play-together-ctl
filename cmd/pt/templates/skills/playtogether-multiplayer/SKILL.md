---
name: playtogether-multiplayer
description: 把單機遊戲改成玩聚的多人聯機版本：先和開發者決定多人玩法，再用玩聚的範本遊戲伺服器（pt add server）把遊戲規則搬到伺服器、前端改成只送操作，最後用 pt dev 多開玩家視窗與 pt verify server 驗證。使用者說「改成多人」「多人連線」「聯機版」「讓朋友一起玩」「multiplayer」時使用。Use when turning a single-player game into a PlayTogether multiplayer game.
---

# 單機遊戲改成玩聚多人聯機

這個 skill 由 `pt` 提供。玩聚的平台接口（玩家身分、房間與座位、斷線重連、踢人、重新開始、回報一局、限流）都在 SDK 的遊戲伺服器套件裡，**你只需要寫遊戲規則**。照下面六個步驟做，不要跳步驟；每一步做完，用一兩句話告訴開發者做了什麼。

最重要的規則：**伺服器決定遊戲狀態**。分數、勝負、輪到誰、計時、隨機結果都在伺服器算；前端只送「操作」（按了什麼、選了什麼），畫面照伺服器回傳的狀態顯示。前端回報的分數或結果不能直接採用。

## 0. 保護原本的程式

1. `git rev-parse --is-inside-work-tree` 確認是不是 git 專案。
2. **不是 git 專案**：**問開發者**要不要先建立一個還原點。強烈建議，因為接下來會改到原本的程式。開發者同意時：
   - 先確認 `.gitignore` 排除了私鑰與密鑰（`keys/`、`*.key`、`.env*`）、`node_modules/` 與建置產物；
   - `git init`、`git add -A`、`git commit -m "改成多人前的原始版本"`；
   - commit 前用 `git status` 看一次要提交的檔案，有密鑰、密碼檔或很大的檔案就先排除，並告訴開發者。

   開發者不要時照樣繼續，但告訴他：之後的改動沒辦法一鍵還原。
3. **有還沒提交的改動**（`git status --porcelain` 不是空的）：**問開發者**先自己提交或暫存（stash）。**不要替他提交、stash 或丟棄**這些改動。
4. **開一個新分支**再改：`git switch -c playtogether-multiplayer`。名稱已經存在時加上 `-2`、`-3`。告訴開發者分支名稱，以及原本的分支（例如 `main`）完全沒動。已經在這個流程開的 `playtogether` 開頭分支上（例如上架時開的），繼續用它，不用再開。
5. 改的過程中，在這個分支上每完成一個階段就提交一次，訊息寫清楚做了什麼，例如「加上範本遊戲伺服器」「遊戲規則搬到伺服器」「前端改成只送操作」。
6. **不要** push、不要 force、不要改其他分支、不要改寫歷史（rebase、amend 開發者的 commit）。
7. 最後告訴開發者怎麼檢查與合併：`git diff <原分支>...playtogether-multiplayer` 看全部改動，滿意後由他自己合併。

## 1. 讀專案、判斷可不可行

- 讀 `playtogether.json`（`client`、`server`、`template`、`web`、`gameServer`）、`PLAYTOGETHER.md`（有的話）、`docs/content/GAMEPLAY_ECONOMY.md`。
- 讀遊戲程式：遊戲狀態放在哪裡、一局怎麼開始和結束、怎麼計分、有沒有計時、有沒有隨機。
- 判斷類型，**先告訴開發者結論**：

| 類型 | 適合度 | 說明 |
| --- | --- | --- |
| 回合制、問答、猜謎、文字、桌遊、派對小遊戲 | 適合 | 範本伺服器的輪詢（每 0.5 秒）就夠用 |
| 同時操作但不需要精準同步（比快、比多、限時作答） | 適合 | 伺服器用時間判定，前端只顯示 |
| 即時動作、物理碰撞、射擊、賽車 | 不適合直接改 | 需要每秒數十次同步與延遲補償，範本伺服器不是為這個設計。建議改成回合制或「各玩各的、伺服器比結果」的玩法，或請開發者自己準備即時伺服器（`gameServer.mode: external`） |

## 2. 和開發者決定多人玩法（一定要問）

多人怎麼玩是遊戲設計，**由開發者決定，不要自己假設**。依遊戲提出 2–3 個具體方案讓開發者選，例如：

- **同題競賽**：所有人玩同一關（伺服器發同一個題目或種子），比分數或速度。
- **輪流**：一人一回合，其他人看（或猜、投票）。
- **對戰**：兩人或多人互相影響（搶同一個資源、互相出題）。
- **合作**：一起完成目標，各自負責一部分。

要確認：人數上下限、一局多久、怎麼分勝負、中途加入或斷線的人怎麼處理、房主（通常是直播主）要不要有額外控制。玩聚的主要使用情境是直播主開房、觀眾用手機加入，所以**手機直接點就能玩、一局 3–5 分鐘內**的設計最適合。

## 3. 加上遊戲伺服器

- `pt init` 接入的既有專案（`template` 是 `existing`，`server` 是 `none`）：問開發者要 Go 還是 Node（預設 Go，可以交給玩聚代管），然後執行 `pt add server -server go`（或 `-server node`；要玩聚代管加 `-hosting playtogether`）。它會寫入 `server/`、`multiplayer/playtogether-net.js`（網頁前端），並更新 `playtogether.json`。
- `pt new` 建立的專案：已經有 `server/`，直接改它。
- `server/third_party/`、`server/playtogether-server.mjs` 是 SDK，**不要修改**（`pt sdk update` 會更新）。

## 4. 把遊戲規則搬到伺服器

範本 `server/main.go`（Node：`server/main.mjs`）是示範玩法「一起按」，照著它的結構換成你的遊戲：

| 範本 | 要做的事 |
| --- | --- |
| `game` 結構 | 一個房間的遊戲狀態（從前端的遊戲狀態搬過來） |
| `View(r, viewer)`（Node：`view`） | 給**這位玩家**看的狀態。只放他可以看的資料（別人的手牌、答案、隱藏角色不能放） |
| `onAction(r, seat, g, action, data)` | 處理操作：先檢查合不合法（輪到他了嗎、這一局在進行嗎、`data` 格式對嗎），再改狀態。不合法回 `&gameserver.ActionError{Code: "..."}`（Node：`throw new ActionError('...')`），前端收到 400 |
| `Tick(r)`（Node：`tick`） | 每 200ms 一次：計時、時間到結束一局 |
| `CanFree`、`Leave`、`Close` | 斷線太久能不能讓出座位、玩家離開時清狀態、房間結束時清理 |
| `r.StartRound(ids)`、`r.EndRound(...)` | 一局開始與結束時呼叫（平台的遊玩紀錄與成效靠它） |

限制：

- 這些函式在套件的鎖裡執行：不要自己加鎖，也**不能呼叫網路、讀寫檔案、開 goroutine 之後再改狀態**。
- 隨機用伺服器的亂數；玩家名稱用 `r.Seats()` 的 `Name`（平台暱稱），不要讓玩家自己填。
- 每個請求內容上限 1KB：`data` 只送操作需要的少量 JSON。
- 前端每 0.5 秒輪詢一次 `action: "state"`：不認得的 action 直接回 nil。

## 5. 前端改成只送操作

網頁（`client: web`）：

1. 把 `multiplayer/playtogether-net.js` 放到會被建置進 `playtogether.json` 的 `web` 資料夾的地方（沒有建置步驟的專案直接放進 `web` 資料夾；Vite 等工具放 `public/`），在 `index.html` 於玩聚 SDK 之後載入。用法寫在檔案開頭。
2. 遊戲邏輯（計分、判定、計時）從前端拿掉，改成 `net.act('操作', 資料)`；畫面在 `onState` 裡依伺服器的狀態重畫。
3. 動畫、音效、輸入手感可以留在前端，但結果以伺服器為準。
4. 玩家暱稱等文字一律 `textContent`，不要用 `innerHTML`。

Godot（`client: godot`）照下面的連線協定，用 `HTTPRequest` 實作（參考 `pt new -client godot` 範本的 `main.gd`）。

### 連線協定（範本伺服器）

- 遊戲伺服器網址：握手後的 `pt.launch.serverUrl`（Godot：`PlayTogether.server_url`）；空的時候用 `playtogether.json` 的 `serverUrl`，本機 `pt dev` 是同一個來源。不能用 `location.host`。
- `POST /join` `{"launchCode": "<pt.launch.code>"}` → `200 {"token","playerId","name","state"}`。啟動碼只用一次，不要存；`403` 無法加入、`409` 房主重新開始了、`503` 暫時不能加入，都請玩家重新整理。
- `POST /action` `{"token","action","data"?}` → `200` 是最新的 View；`400 {"error"}` 是規則拒絕；`401` token 失效（請玩家重新整理）；網路錯誤時繼續輪詢，恢復後自動接上。

## 6. 驗證

依序做，有錯就修，不要跳過：

1. 伺服器測試（Go：`cd server && go test ./...`；有寫的話）。
2. `pt dev -players 3`：開三個玩家視窗，實際玩完一局。測：一人斷線再回來、房主踢人、房主重新開始（控制面板）。
3. `pt verify server -start-action <開始一局的 action>`：驗證平台接口與「回報一局」。
4. `pt check -local`。
5. 更新 `docs/content/GAMEPLAY_ECONOMY.md` 與 `store/listing.json` 的人數、玩法說明（用 playtogether-store skill）。

完成後告訴開發者：改了哪些檔案、多人玩法怎麼運作、還沒處理的情況（例如中途加入）。要上架時用 playtogether-publish skill；自己部署的遊戲伺服器要把網址填進 `playtogether.json` 的 `serverUrl`。
