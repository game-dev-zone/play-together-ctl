---
name: playtogether-publish
description: 把遊戲上架到玩聚（PlayTogether）：從接上專案（pt init）、改到 pt check 通過、準備上架資料、登入授權、建立遊戲、上傳到送審，一路自動做完，只在需要開發者本人時才停下來。使用者說「幫我上架到玩聚」「上架到 pt」「送審」「publish to PlayTogether」「上架這個遊戲」時使用；只要準備上架資料（文字、圖示、截圖）時用 playtogether-store。
---

# 上架到玩聚

這個 skill 由 `pt`（玩聚開發者工具）提供：`pt skills install` 裝在使用者層級，`pt new`、`pt init` 也會放進專案。目標是開發者只說一句「幫我上架到玩聚」，其他步驟都由你完成。

**原則**

- 一路做下去，不要每一步都問。只在下面標明「問開發者」的地方停下來，而且一次把要問的問完。
- 每一步做完用一兩句話說你做了什麼；最後給一份總結。
- **能否上架、正式分級、內容標示、警語由玩聚平台審核決定**。你不判斷、也不對開發者保證會通過。
- **不要讀出、印出、複製或提交私鑰**（`keys/`、`*.key`），也不要讀 `pt` 的授權檔。`keys/` 要在 `.gitignore` 裡（`pt init` 會加）。
- 不要用 `-skip-check` 繞過檢查，不要執行 `pt logout`，不要改 `pt` 自己的檔案來讓檢查通過。
- 每個 `pt` 指令都可以用 `pt <指令> -h` 看參數；看不懂錯誤訊息時先讀它，不要猜。

## 0. 準備

1. `pt version`：確認有安裝。沒有時請開發者照文件站「安裝 pt」安裝，然後停下來。`pt` 印出有新版時，提醒開發者更新，但繼續做。
2. **平台網址**：專案的 `playtogether.json` 有 `platform`（不含「（」）就用它；開發者在這次對話裡給過也用它。都沒有時 **問開發者**：「要上架到哪個玩聚環境？請貼平台網址（文件站網址，例如 https://…）」。平台網址就是文件站所在的網址，結尾不加斜線、不加 `/api`。
3. 開發者要先是**玩聚開發者團隊的擁有者或管理者**（在平台網址的 `/developers` 申請，核准後才有團隊）。不確定時先繼續，到第 5 步登入時就會知道。

**保護原本的程式（改任何檔案之前）**：

1. `git rev-parse --is-inside-work-tree` 確認是不是 git 專案。
2. **不是 git 專案**：**問開發者**要不要先建立一個還原點。強烈建議，因為接下來會改到原本的程式。開發者同意時：
   - 先確認 `.gitignore` 排除了私鑰與密鑰（`keys/`、`*.key`、`.env*`）、`node_modules/` 與建置產物；
   - `git init`、`git add -A`、`git commit -m "接上玩聚前的原始版本"`；
   - commit 前用 `git status` 看一次要提交的檔案，有密鑰、密碼檔或很大的檔案就先排除，並告訴開發者。

   開發者不要時照樣繼續，但告訴他：之後的改動沒辦法一鍵還原。
3. **有還沒提交的改動**（`git status --porcelain` 不是空的）：**問開發者**先自己提交或暫存（stash）。**不要替他提交、stash 或丟棄**這些改動。
4. **開一個新分支**再改：`git switch -c playtogether`。名稱已經存在時加上 `-2`、`-3`。告訴開發者分支名稱，以及原本的分支（例如 `main`）完全沒動。已經在這個流程開的 `playtogether` 開頭分支上（例如上架時開的），繼續用它，不用再開。
5. 改的過程中，在這個分支上每完成一個階段就提交一次，訊息寫清楚做了什麼，例如「接上玩聚：pt init」「pt check 通過」。
6. **不要** push、不要 force、不要改其他分支、不要改寫歷史（rebase、amend 開發者的 commit）。
7. 最後告訴開發者怎麼檢查與合併：`git diff <原分支>...playtogether` 看全部改動，滿意後由他自己合併。

## 1. 接上專案（`pt init`）

已經有 `playtogether.json` 就跳過這一步（但仍然執行一次 `pt init`，它只會補上缺少的 skill 與說明檔，不改其他東西）。

沒有時，先**讀專案**決定參數，再執行一次 `pt init -y …`（`-y` 不詢問；參數都要明確給）：

| 參數 | 怎麼決定 |
| --- | --- |
| `-name` | 遊戲代號：英文小寫、數字、連字號，英文開頭，會出現在網址；用專案或遊戲的英文名稱 |
| `-client` | `web`（HTML5：PlayCanvas、Phaser、Cocos、Unity WebGL、純 JS…）或 `godot`（有 `project.godot`） |
| `-web` | web：要上傳的資料夾（根目錄有 `index.html`）。是建置產物（`dist/`、`build/`）時，**先照專案的建置指令建置**（例如 `npm run build`），確認 `index.html` 存在 |
| `-game-server` | `none`：單人、或多人只靠畫面端（同一台裝置輪流、沒有連線）。`external`：遊戲有自己的連線伺服器（WebSocket、HTTP API、Colyseus、Socket.IO、Firebase 等即時同步） |
| `-hosting` | 只有 `external` 才有。不填：遊戲伺服器由開發者自己部署。`playtogether`：交給玩聚代管，另用 `-server-dir` 指定伺服器的 Go 模組資料夾（預設 `.`）。見下面「遊戲伺服器由誰執行」 |
| `-orientation` | `portrait` 直式、`landscape` 橫式、`any` 都可以；看畫面尺寸、CSS、引擎設定 |
| `-profile` | 內容邊界，預設 `general`。遊戲本身有明顯暴力、血腥、恐怖、性暗示等內容時，**問開發者**要宣告 `moderate` 還是 `strong`（你不能自己提高內容邊界） |
| `-table-game` | 預設 `none`。有撲克、麻將、骰子、輪盤、百家樂、拉霸等玩法時填對應的值（這類玩法需要平台審核，要在上架資料的 `reviewNotes` 說明） |
| `-platform` | 第 0 步的平台網址 |

執行後讀 `pt init` 印出的「還要處理」，以及新產生的 `PLAYTOGETHER.md`（有的話）、`docs/content/`。

### 遊戲伺服器由誰執行（`external`）

遊戲伺服器有兩種跑法。**玩聚代管是建議的做法**（開發者不用自己找主機、不用另外付主機費用；目前要平台開通、只支援 Go，見文件站「遊戲伺服器」）。開發者問到時說明這個選項與限制，由開發者（或平台）決定能不能用。

| | 開發者自己部署（`-hosting` 不填） | 玩聚代管（`-hosting playtogether`） |
| --- | --- | --- |
| 誰部署 | 開發者自己的主機，填 `playtogether.json` 的 `serverUrl` | `pt upload` 在 `gameServer.dir` 執行 `GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build .`，和畫面一起上傳；發佈時平台啟動 |
| 語言 | 任何 | **只支援 Go**；`gameServer.dir` 要有 `go.mod`，在那裡 `go build .` 就要得到伺服器（main 在別的資料夾時，在那裡加一個呼叫同一份程式的 `main.go`） |
| 前端連哪裡 | 設定檔裡的完整網址；上傳加 `-connect` | 握手後的 `pt.launch.serverUrl`（每個房間由平台指定）；不用 `-connect`、不用 `serverUrl` |
| 金鑰 | 開發者把 `keys/server.key` 放到伺服器 | 平台每次部署自己產生並掛載（`PT_KEY_FILE`），不用處理 |

怎麼選：

- `playtogether.json` 已經有 `gameServer.hosting`，或開發者說要玩聚代管：用代管。
- 伺服器是 Go、開發者還沒說伺服器要放哪裡（專案裡沒有部署設定、也沒給網址）：**問開發者一次**要自己部署還是玩聚代管。
- 其他情況（不是 Go、已經部署在自己的網址）：自己部署。

代管的限制（伺服器程式要配合）：

- 啟動時讀環境變數：`ADDR`（`0.0.0.0:8080`；也有 `PORT=8080`）、`PT_API_URL`、`PT_KEY_ID`、`PT_KEY_FILE`、`ALLOW_ORIGINS`（遊戲畫面的網域，WebSocket Origin 與 CORS 照它檢查）、`TRUST_PROXY=1`。用伺服器 SDK 的 `gameserver.PlatformFromEnv()` 建立平台 client。
- 要有 `GET /healthz` 和 `GET /readyz`（都回 200 才算部署完成；遊戲被停用時 `/readyz` 回 503）。
- 容器不能連外：只能連 `PT_API_URL`（平台 API）。不能連自己的資料庫、Redis、第三方 API；要用 AI 走玩聚 AI 閘道（`/docs/ai-services`）。
- 單一執行個體、狀態放記憶體，重新部署時房間會結束；記憶體 256 MB、CPU 0.5、唯讀檔案系統（`/tmp` 16 MB）。要讀的資料檔用 `gameServer.data` 指定資料夾，執行時在 `/app/data`。
- WebSocket 可以用（走 `wss://`）。
- 要平台開通：沒開通的環境上傳不會擋，但審核通過後發佈會失敗（「沒有可用的遊戲伺服器主機」）。發生時告訴開發者聯絡平台，不要改宣告繞過。

## 2. 改到 `pt check -local` 通過

執行 `pt check -local`。訊息分級：`ERROR_HARD_RULE`／`ERROR_SCHEMA`／`ERROR_BUILD` 要修好；`REVIEW_REQUIRED` 不用改，但要在 `store/listing.json` 的 `reviewNotes` 寫設計理由；`WARNING` 能修就修。重複執行直到沒有錯誤。

常見要改的地方（細節以平台網址的文件站為準：`/docs/getting-started#existing`、`/docs/browser-sdk`、`/docs/game-server`、`/docs/godot`）：

1. **載入瀏覽器 SDK**（web）：`pt sdk export <空資料夾>`（例如暫存資料夾）→ 把 `browser/playtogether-sdk.iife.js` 複製到上傳資料夾 → 在 `index.html` 用 `<script>` 載入，遊戲開始前：

   ```js
   const pt = await PlayTogether.init({ platformOrigins: ['<平台網址>'] });
   // …載入完成、可以開始玩時
   pt.ready();
   ```

   直接在瀏覽器開（不在玩聚裡）時 SDK 會進入模擬模式，原本的開發方式不受影響。Godot 用外掛（`pt sdk export` 的 `godot/addons`，見 `/docs/godot`）。
2. **玩家身分改用啟動碼**（`external`）：不要再用遊戲自己的登入、訪客帳號或隨機 ID。前端把 `pt.launch.code` 交給遊戲伺服器，伺服器用伺服器 SDK（`pt sdk export` 的 `go/`、`node/`）向平台兌換，用兌換結果的玩家 ID 與暱稱。伺服器程式在這個專案裡就照 `/docs/game-server` 的「external 的最低要求」改；不在這個專案裡就把要改的地方列給開發者。玩聚代管時另外照上面「代管的限制」改（環境變數、`/healthz`、`/readyz`、不連外）。
3. **連遊戲伺服器用完整網址**（`external`）：上傳後畫面在 `https://<版本>.games.<平台網域>`，不能用 `location.host`、相對路徑（例如 `/ws`）。自己部署：改成設定檔裡的完整網址，伺服器要允許這個來源（CORS、WebSocket Origin），上傳時用 `-connect wss://…`（逗號分隔）申報。玩聚代管：握手後用 `pt.launch.serverUrl`（`https://` 換成 `wss://` 就是 WebSocket），`pt check` 會在上傳資料夾的 `.js` 檔找讀它的程式。
4. **外部載入的檔案**：CDN 上的程式與樣式表用 `pt cdn add <網址>` 申報；字型、圖片、音效改成放進上傳資料夾（例如 Google Fonts 改成自己的字型檔）。
5. **平台不允許的功能**：廣告、追蹤與分析工具、自己的付款、WebRTC、麥克風與相機。要拿掉時，**先列出你要拿掉的東西，問開發者一次**再動手（這會改變遊戲行為）。遊戲用麥克風是為了**唱歌、哼音高、用聲音控制**時，告訴開發者可以改用平台的音高功能（`PLAYTOGETHER.md`「用聲音玩：音高」）：遊戲拿到音高數字，要加遊戲伺服器的授權，玩法要按住說話。其他用途（錄音、語音辨識、音量偵測、和別人通話）目前沒有替代，只能拿掉。
6. **平台外殼樣式**：平台的選單、語音、按住說話按鈕會疊在遊戲上。看遊戲的主要畫面配色，在 `playtogether.json` 加上 `shell`（`PLAYTOGETHER.md`「外殼樣式」）：明亮畫面 `light`、深色畫面 `dark`、不想讓按鈕搶畫面 `minimal`，`accent` 用遊戲的主色（`pt check` 說對比不足就調整）。告訴開發者選了什麼；遊戲自己左上角、右上角、右下角有按鈕時，提醒會被平台按鈕蓋住，要移開（照 `pt.ui.on('insets')`）。
7. 改完後實際跑一次：web 用本機伺服器開啟上傳資料夾，或 `pt dev`；確認畫面能載入、沒有錯誤。有遊戲伺服器時用 `pt dev` 的本機假平台測加入、重新整理、踢人、重新開始、關房：玩聚代管的專案直接 `pt dev`（它會編譯並啟動伺服器，握手時給 `launch.serverUrl`）；自己部署的先啟動自己的伺服器，再 `pt dev -no-server -site <畫面網址>`（前端讀 `launch.serverUrl` 時加 `-server-url <伺服器網址>`）。

不確定怎麼改、或要改的範圍很大（例如整套登入系統）時，說明情況並**問開發者**，不要硬改。

## 3. 上架資料（`store/`）

照 `.claude/skills/playtogether-store/SKILL.md`（Codex：`.agents/skills/playtogether-store/SKILL.md`；`pt init` 會放進專案）的步驟做：起草繁中＋英文的 `store/listing.json`、圖示與封面、用 `pt store screenshot` 擷取截圖。開發者已經寫好的內容不要整段改掉。圖示與封面用 Codex 生成（`codex exec`，以遊戲的實際截圖當參考，步驟在 playtogether-store）；Codex 沒有安裝或沒有登入時，**問開發者**要安裝登入，還是由他提供圖示（與封面）。

playtogether-store 最後說「不要自己執行 `pt upload`」：開發者說「幫我上架」就是明確要求，做完上架資料後繼續第 4 步，不用再問。

做完再執行一次 `pt check -local`，到沒有錯誤為止。

## 4. 授權（`pt login -register`）

只有這一步一定要開發者本人。授權存在這台電腦，已經登入過（30 天內）的 `pt register`、`pt upload` 會直接成功，可以先試第 5 步，出現要登入的錯誤再回來。

1. **在背景執行** `pt login -register`（它會等開發者在瀏覽器確認，最多 10 分鐘；Claude Code 用背景執行或把逾時設成 10 分鐘，不要用預設的 2 分鐘）。
2. 立刻把 `pt` 印出的**網址與短碼**告訴開發者：「請打開這個網址，用玩聚帳號登入，確認短碼是 XXXX，選擇團隊後按同意。」
3. 等指令結束。成功會印出團隊名稱。失敗時：
   - 沒有團隊或不是擁有者／管理者：請開發者到平台網址的 `/developers` 申請，或請團隊擁有者執行；停下來。
   - 逾時：再執行一次。

## 5. 建立遊戲（`pt register`）

`playtogether.json` 已經有 `uploadKeyId` 就跳過。否則執行 `pt register`：建立草稿遊戲、產生金鑰（私鑰寫在 `keys/`，只送公鑰），金鑰 ID 自動寫回 `playtogether.json`。

- 代號已被使用或是保留字：換一個 `name`（改 `playtogether.json`）再試，並告訴開發者新代號。
- 草稿已達上限：請開發者到開發者頁放棄不用的草稿。

## 6. 上傳並送審（`pt upload`）

1. 決定版本號：`pt status` 看已經上傳過的版本；第一次用 `1.0.0`，之後用比最新版本大的號碼（修正用第三碼、加功能用第二碼）。版本號不能重複。
2. 執行 `pt upload -version <版本> -notes "<這版改了什麼，給審核者看>"`，`external` 自己部署時加上 `-connect <遊戲伺服器的 wss:// 或 https:// 來源>`；玩聚代管不用（`pt upload` 會先在本機建置伺服器，要有 `go` 指令）。`external` 的版本說明要寫出限流方式（審核會照它實測）。它會檢查、打包、上傳版本與 `store/`，檢查通過後直接送審。
3. 出現「這款遊戲有其他版本在審核中」：**問開發者**要不要撤回舊的、改送這個版本（同意後加 `-replace` 重跑）。
4. 平台檢查失敗：讀錯誤訊息、修正、換一個新的版本號再上傳。
5. 檢查通過後執行 `pt preview`（版本檔案還在處理時等幾秒再試），把印出的預覽房間網址給開發者：用手機打開、以玩聚帳號登入就能在平台外殼裡實測（1 小時，只有團隊成員能進；多人測試請團隊其他成員用自己的帳號一起進）。玩聚代管的版本回「預覽要等平台啟動伺服器」時，告訴開發者先用 `pt dev` 在本機測試，不要改宣告繞過。

## 7. 回報

執行 `pt status`，然後告訴開發者：

- 送審了哪個版本、上架資料有沒有一起送審。
- 預覽房間的網址（`pt preview`），請開發者用 iPhone 與 Android 實際玩完一局。
- 開發者頁：`<平台網址>/dev/<遊戲代號>`：看審核進度、調整上架資料與截圖裁切、審核通過後決定發佈時間。
- 審核通過前遊戲不會出現在列表；審核結果會用站內通知告訴團隊。
- `external` 自己部署：遊戲伺服器要部署在公開網址，並用 `pt register` 產生的伺服器金鑰（`keys/` 裡，設定方式見 `/docs/game-server`）。
- `external` 玩聚代管：不用部署。審核通過、發佈時平台才啟動伺服器，部署完成才上線；發佈失敗（例如環境沒開通代管）會在開發者頁顯示原因，可以重試。
- 這次改了哪些專案檔案（建議開發者檢查後提交；不要提交 `keys/`）。
- 還需要開發者做的事（如果有）。
