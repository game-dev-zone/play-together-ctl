# pt：玩聚開發者工具

`pt` 是玩聚 PlayTogether（台灣的瀏覽器多人派對遊戲平台）給第三方開發者的指令列工具：一個指令建立已接好玩聚 SDK 的遊戲專案、在本機用模擬模式開發、上傳前檢查、產生金鑰、上傳版本。完整文件在玩聚開發者文件站 `/docs/pt`。

*`pt` is the command-line tool for third-party developers building games for PlayTogether (玩聚), a browser party-game platform in Taiwan: scaffold a project wired to the PlayTogether SDK, develop locally in mock mode, check and upload builds.*

## 安裝

**下載執行檔**（建議）：到 [Releases](https://github.com/game-dev-zone/play-together-ctl/releases) 下載對應系統的檔案，改名成 `pt` 放進 PATH。

```sh
# macOS（Apple 晶片），其他系統換成 pt-darwin-amd64、pt-linux-amd64、pt-linux-arm64、pt-windows-amd64.exe
curl -fLo pt https://github.com/game-dev-zone/play-together-ctl/releases/latest/download/pt-darwin-arm64
chmod +x pt && sudo mv pt /usr/local/bin/
pt version
```

每個版本都附 `SHA256SUMS`，可以用 `shasum -a 256 -c SHA256SUMS --ignore-missing` 檢查。

**用 Go 安裝**（Go 1.24 以上）：

```sh
go install github.com/game-dev-zone/play-together-ctl/cmd/pt@latest
```

## 五分鐘開始

```sh
pt new                           # 逐步選擇名稱、畫面（網頁或 Godot）、多人與遊戲伺服器、方向、AI 開發模式
cd my-party-game
pt dev                           # 本機模擬模式：啟動參考伺服器並開啟玩家分頁（多人專案 2 個、單人專案 1 個）
make verify                      # 啟動驗證：伺服器測試、在模擬模式實際啟動一次、檢查要上傳的檔案
```

伺服器範本內建斷線重連骨架（`server/session.go`、`server/session.mjs`）：座位綁玩家 ID、新 token 取代舊 token、離線與讓座、定期向玩聚查房間（踢人、重新開始、關房）、查不到平台時暫停新加入；前端範本會自動重新連線。

`PLAYTOGETHER.md` 的「素材」一節建議用 Codex 生成圖片：整套素材共用風格卡 `art/STYLE.md`（範本附一套明亮派對風格：形狀、線條、平塗上色、調色盤色碼、光線、排除項目），先做一張樣板，其他素材都用 `-i` 參考樣板生成，降低「每張各自生成」的 AI 感；生成前同時讀 `docs/content/CONTENT_PROFILE.md`（內容邊界）與 `GAMEPLAY_ECONOMY.md`（玩法宣告）、提示詞寫明平台硬性規則、看過再用，每個素材記錄在 `art/SOURCES.md`。範本內建程式合成的預設音效（網頁版 `web/sfx.js`、Godot 版 `main.gd`，沒有音檔），並接上平台的「遊戲聲音」開關；要用音檔或音樂時照「音效與音樂」一節的來源順序（程式合成 → CC0 → AI 生成付費方案 → 確認沒有登記 Content ID 的免費音樂）。

新專案附有 `Makefile`（`make` 列出所有指令：`dev`、`server`、`verify`、`check`、`keys`、`upload`，Godot 另有 `build`、`clean`），每個指令都只是包裝 `pt`。

用 Godot 開發：

```sh
pt new -client godot -server go my-godot-game   # 遊戲伺服器也可以用 -server node
cd my-godot-game
pt dev                           # 先用 Godot 無頭模式匯出網頁版，再啟動模擬模式
```

Godot 專案需要 Godot 4.3 以上與網頁匯出範本（Godot 編輯器 → 編輯器 → 管理匯出範本）。`pt` 依序找環境變數 `GODOT`、PATH 裡的 `godot`、macOS 的 `/Applications/Godot.app`。範本已設定好網頁匯出（關閉執行緒支援、載入 SDK）、玩聚外掛，並附上 Noto Sans TC 常用字子集（SIL OFL 1.1），中文不會變成方框。

### 用 AI 開發

開發者不需要了解 AI 框架、skill 或指令：在專案資料夾啟動 Claude Code（`claude`）或 Codex，直接用一般的話描述想做的遊戲或功能（「幫我做一款 4 人合作遊戲」「幫我建立第一個可玩原型」「操作手感不好」）。`pt new` 產生的規則讓 AI 自己判斷要先做創意設計還是直接實作、新遊戲先做最小可玩原型、改玩法或故事時先提出設計變更請求，完成前跑驗證。

每個專案（所有 AI 模式）都有：

| 檔案 | 內容 |
| --- | --- |
| `PLAYTOGETHER.md` | 玩聚平台規則（架構原則、iframe 執行環境、審核），**最高優先** |
| `docs/ai/AI_WORKFLOW.md` | AI 工作流程：權威順序、創意與實作的分工、自動判斷需求、先做原型、遊戲伺服器架構規則、設計變更請求、完成前驗證 |
| `docs/ai/PROJECT_CONTEXT.md` | 專案背景：畫面、方向、AI 模式、**多人與遊戲伺服器架構**、已安裝的 AI 能力、重要路徑 |
| `CLAUDE.md`、`AGENTS.md` | Claude Code 與 Codex 的入口：依序讀上面三份（有 `docs/ai/CURRENT_TASK.md` 時也讀） |

### AI 開發模式（`-mode`）

| 模式 | 內容 | 適合 |
| --- | --- | --- |
| `standard`（精靈建議） | 平台規則＋[awesome-gamedev-agent-skills](https://github.com/gamedev-skills/awesome-gamedev-agent-skills)（Apache-2.0）精選：手感、手機介面、觸控輸入、音效、素材風格一致、效能、關卡節奏、快速原型，加上 Godot 4.7 或網頁引擎（Phaser、PixiJS、three.js）的 skill | 大部分遊戲；Claude Code（`.claude/skills/`）與 Codex（`.agents/skills/`）都能用 |
| `studio` | `standard`＋[Claude Code Game Studios](https://github.com/donchitos/claude-code-game-studios)（MIT）的規劃、故事、設計流程，以及 `docs/ai/CURRENT_TASK.md`（工作契約）、`docs/ai/DECISIONS.md`（設計變更紀錄）。**只限 Godot**；流程較重、token 用量大，Studios 流程只在 Claude Code 執行 | 故事、謎題、角色扮演或規模較大的遊戲 |
| `lightweight` | 只有平台規則與工作流程文件 | 不想加任何第三方內容；`-y`、非互動與 CI 沒指定模式時的預設（不下載任何東西） |
| `custom` | 用 `-skills none\|core\|full`、`-ai studios` 自己組合 | 進階；舊參數（只給 `-skills` 或 `-ai`）等於 `custom`，結果和以前相同 |

- 第三方內容一律固定版本，下載後逐一比對內嵌清單的 SHA-256，任何一個檔案不符就停止，不留下專案資料夾；`pt` 不安裝清單以外的檔案，也不執行下載的任何程式。授權與來源記錄在 `THIRD_PARTY/`。
- 遊戲開發 skill 只收說明文件（不收 router、scripts、agents），內容不修改；和玩聚規則衝突的（Godot 多人連線、Godot 匯出、單機存檔、itch／Steam 上架）不收。`pt` 自己決定收哪些、衝突時誰優先，不安裝上游的 router。
- Claude Code Game Studios（目前 1.2.0）：
  - 開發者不用學它的指令：`CLAUDE.md` 開頭的玩聚區塊讓 Claude 自己判斷何時使用它的流程（包括實作 story 與完成檢查）。
  - `pt` 在驗證完雜湊後只調整這幾處，都有「PlayTogether（pt new 加入）」標記，記錄在 `THIRD_PARTY/claude-code-game-studios/SOURCE.md`：`dev-story`、`story-done` 改成 AI 可以自動使用；`team-release`、`release-checklist` 改成只能手動（發佈由 `pt check`、`pt upload` 與玩聚審核負責）；`project.yaml` 預先寫入嚴謹度 `minimal`、自動化程度 `guided`，並關閉它連 GitHub 檢查更新的功能（升級一律透過新版 `pt`）。上游改版導致找不到要調整的地方時，安裝直接失敗。
  - 它的 hooks 會在你的電腦上由 Claude Code 執行。
  - Godot 專案改放在專案根目錄（範本的工具需要 `project.godot` 在根目錄）；`server/`、`design/`、`docs/` 等資料夾加上 `.gdignore`，不會被打包進遊戲。

### 遊戲伺服器架構（`-game-server`）

遊戲伺服器不是上架的必要條件。專案在 `playtogether.json` 宣告架構，AI 改連線前一定要先確認，不會自行改成引擎內建多人連線、P2P 或其他架構：

| `gameServer.mode` | 意思 | 範本 `server/` 的角色 |
| --- | --- | --- |
| `playtogether-managed`（預設） | 多人遊戲，用玩聚範本伺服器的架構 | 正式遊戲伺服器的起點，要部署並填 `serverUrl` |
| `external` | 多人遊戲，你已經有自己的遊戲伺服器 | 整合參考與本機模擬，**不需要部署**；`serverUrl` 填你的伺服器 |
| `none` | 單人或畫面端就能玩（解謎、敘事、冒險、節奏、休閒等） | 本機模擬與參考，**不需要部署**；`serverUrl` 留空，也不需要遊戲伺服器金鑰（只要上傳金鑰） |

`pt check` 依宣告的架構判斷：`none` 不要求 `serverUrl`；`multiplayer.enabled: true` 卻是 `none` 會被擋（多人遊戲需要經核准的連線架構）。舊專案沒有這兩個欄位時，當成多人＋`playtogether-managed`，結果和以前相同。

### 上架資料（`store/`）

名稱、簡介、說明（**繁中與英文必填**）、標籤、人數、一局時間、給審核者的說明放在 `store/listing.json`，圖示（`icon.png`）、截圖（`screenshots/01.png`…，1–10 張，實際遊戲畫面）、選填的封面（`cover.png`）放在同一個資料夾。規格見專案裡的 `store/README.md`。

- `pt upload` 會和版本一起上傳上架資料，**平台自動裁切、縮放、重新編碼**；內容沒變時沿用上一次的修訂。只想換截圖或修說明：`pt upload -listing-only`；只上傳版本：`-no-listing`。
- `pt check` 一併檢查 `store/`：範本文字還沒改或缺圖片時只說明（`pt upload` 會只上傳版本）；長度、標籤、圖片格式、尺寸與比例錯誤是 `ERROR_SCHEMA`（`-skip-check` 不能略過）。
- `pt upload` 也會送出 `playtogether.json` 的宣告，**平台自己判斷**硬性規則與需要審核的項目（不採信本機的 `pt check`）。
- `pt status`：版本與上架資料的審核狀態、退件條號與原因、正式分級（由平台審核決定）。
- `pt login`：在瀏覽器用玩聚帳號確認短碼與團隊，取得送審授權（30 天，可以在玩聚前台的開發者頁撤銷）。之後 `pt upload` 會在檢查通過後**直接送審**版本與上架資料（`-no-submit` 只上傳）；沒登入時只上傳，可以到開發者頁預覽、調整圖片裁切、送審。`pt logout` 刪除這台電腦上的授權。上傳金鑰只能上傳、不能送審。
- 託管遊戲第一次上架必須有核准的上架資料。只改簡介、說明裡的幾個字（例如錯字）免審、直接生效，平台事後抽查。

### 內容邊界與政策宣告（`-profile`、`-table-game`）

`playtogether.json` 記錄開發者的宣告，`pt new` 依它產生 `docs/content/CONTENT_PROFILE.md` 與 `docs/content/GAMEPLAY_ECONOMY.md`（所有 AI 模式都有），AI 依它工作：

```
Platform Audience（platformAudience）：玩聚使用者 18+，是平台的使用者資格，不是遊戲內容分級
Content Profile（contentProfile）：開發期間的內容邊界（呈現強度），不是正式分級
Sensitive Content（sensitiveContent）：歷史迫害、政治壓迫、歧視描寫等敏感題材
Gameplay Policy（gameplay）：牌、骰子、籌碼、虛擬下注、牌桌類型
Economy Policy（economy）：現實經濟價值
        ↓
pt check：PASS／REVIEW_REQUIRED／ERROR
        ↓
玩聚平台審核：唯一的最終准駁者（上架、正式分級、警語、標籤、玩法是否接受）
```

| `-profile` | 內容 |
| --- | --- |
| `general`（預設） | 休閒、益智、家庭與派對：輕度衝突、低恐怖、沒有血腥與成人題材 |
| `moderate` | 中度戰鬥、恐怖、犯罪故事、少量血跡、酒吧場景、輕度粗話 |
| `strong` | 較強恐怖與戰鬥、較明顯血跡、成人故事題材、較強語言、非露骨性暗示 |
| `custom` | 在終端機逐項設定暴力、血腥、恐怖、性內容、粗俗語言、酒、菸；非互動時回錯誤 |

- **內容邊界是上限，不是要求**：AI 不會主動加入這些內容。`general`、`moderate`、`strong` 是固定值，要調整個別項目就改成 `custom`。**AI 不能自行提高內容邊界**：要加入超出邊界的內容時先提出 `CONTENT_PROFILE_CHANGE_REQUEST`，開發者同意後才改。
- **牌桌不是博彩**（Table Game ≠ Real-Money Gambling）：牌、骰子、籌碼、Bet／Call／Raise／Fold／All-in 本身不違規。`-table-game card|poker|mahjong|dice|board|slot|roulette|baccarat|other` 建立牌桌遊戲（只填常用的起始值，不含分級或准駁）。
- **描寫仇恨 ≠ 鼓吹仇恨**：歷史、政治、批判性的描寫是需要平台審核的題材，鼓吹仇恨才是硬性規則。
- `-rating` 是 v0.14 的舊寫法：`general`、`12plus`（= `moderate`）、`15plus`（= `strong`）、`custom`。

`pt check` 的結果：

| 等級 | 內容 | `pt upload` |
| --- | --- | --- |
| `ERROR_HARD_RULE` | 平台硬性規則：真實金錢下注、可購買下注本金、提領、兌換、可轉讓現實價值、付費隨機獎勵、鼓吹仇恨、平台不收的內容；私鑰、廣告／追蹤 SDK、WebRTC、付款、麥克風、未申報的 CDN 等平台與安全架構規則 | 擋下，**`-skip-check` 不能略過** |
| `ERROR_SCHEMA` | 宣告錯誤或自相矛盾：缺欄位、拼錯的鍵、preset 被手改、牌桌欄位矛盾、`docs/content/` 的開發契約文件缺少或和宣告不一致、音效沒記錄來源 | 擋下，**`-skip-check` 不能略過** |
| `ERROR_BUILD` | 建置或本機設定：還沒匯出、沒有載入 SDK、`serverUrl`／`platform` 沒填或不一致 | 擋下，`-skip-check` 可以略過 |
| `REVIEW_REQUIRED` | 需要玩聚平台人工審核：撲克、麻將、吃角子老虎、輪盤、類百家樂、虛擬下注、下注用的籌碼（`wageringTokens`）、賭場風格、跨局或每日贈送的下注籌碼、商業模式涉及真實金錢、敏感題材；一般桌遊的計分 token（`scoreTokens`）不需要審核 | 可以送審，**不代表已批准** |
| `WARNING` | 舊格式、`policyVersion` 和這個 `pt` 的平台政策版本不同、建議事項 | 繼續 |

- `pt check` 只檢查設定檔，不判斷遊戲實際內容，不推論法律分級，也不做關鍵字偵測；`pt check -json` 輸出結果給 CI 與 AI 工具。
- 需要審核與硬性規則的清單內建在 `pt`（政策資料），玩聚平台政策改變時隨新版 `pt` 更新；`playtogether.json` 的 `policyVersion` 記錄專案同步的版本。
- 宣告經開發者確認修改後，執行 `pt content sync` 重新產生 `docs/content/` 的兩份文件並更新 `policyVersion`（宣告本身有錯誤時不產生）。
- v0.14 的 `contentRating` 仍可讀取：依 v0.14 的規則驗證，一般只有遷移提醒，但明確的真錢下注、付費抽獎仍是硬性規則，不合法的舊值是宣告錯誤；建議用 `pt new -profile …` 建暫存專案，把 `platformAudience`、`contentProfile`、`sensitiveContent`、`gameplay`、`economy` 與 `docs/content/` 複製過去。
- `art/STYLE.md` 只管視覺風格，產生素材前同時讀 `PLAYTOGETHER.md`、`CONTENT_PROFILE.md` 與 `GAMEPLAY_ECONOMY.md`。

| 選項 | 可用值 | 說明 |
| --- | --- | --- |
| `-client` | `web`（預設）、`godot` | 遊戲前端。`web` 產生 `web/`；`godot` 產生 Godot 專案 `game/`，匯出到 `build/` |
| `-server` | `go`（預設）、`node` | 遊戲伺服器（兌換啟動碼、判定勝負）|
| `-template` | `web-go`、`web-node`、`godot-go`、`godot-node` | 舊寫法，等於同時指定上面兩個 |
| `-game-server` | `playtogether-managed`（預設）、`external`、`none` | 遊戲伺服器架構（見上方）；同時決定 `multiplayer.enabled`（`none` 是單人） |
| `-profile` | `general`（預設）、`moderate`、`strong`、`custom` | 內容邊界（見上方）；`-y`、非互動與 CI 沒指定時也用 `general`（`-rating` 是舊寫法） |
| `-table-game` | `none`（預設）、`card`、`poker`、`mahjong`、`dice`、`board`、`slot`、`roulette`、`baccarat`、`other` | 牌桌玩法（見上方）；是否能上架由玩聚平台審核決定 |
| `-mode` | `standard`、`studio`、`lightweight`、`custom` | AI 開發模式（見上方）；`studio` 沒指定畫面時自動用 Godot |
| `-skills` | `full`、`core`、`none` | 自訂：遊戲開發 skill 的範圍 |
| `-ai` | `studios`、`none` | 自訂：加入 Claude Code Game Studios（只限 Godot；只指定它時自動用 Godot） |
| `-y` | | 不詢問，沒指定的項目用預設值 |

在終端機執行 `pt new` 時，沒用參數指定的項目（名稱、畫面、是否多人與遊戲伺服器、伺服器語言、方向、AI 開發模式）會逐步詢問，最後顯示對應的完整指令。CI（環境變數 `CI` 有值）或腳本裡（不是終端機）不會詢問；沒指定 AI 模式時用 `lightweight`，不需要網路。

既有專案想改用新的 AI 工作流程：用 `pt new` 以相同設定建一個暫存專案，把 `docs/ai/`、`CLAUDE.md`、`AGENTS.md` 複製過去（`CLAUDE.md`、`AGENTS.md` 改過的話手動合併），並在 `playtogether.json` 加上 `aiMode`、`multiplayer`、`gameServer`。

不需要金鑰、不需要連線到玩聚：直接開啟時 SDK 自動進入模擬模式。

## 依遊戲類型開始

七種常見情境（單人益智、多人合作、桌遊、撲克、麻將、歷史恐怖、類吃角子老虎）的建立指令、關鍵宣告與預期的 `pt check` 結果見 [`examples/dogfood/`](examples/dogfood/)。

不確定參數怎麼選時，可以先從遊戲類型決定。下面的指令都是起點，不需要先了解 Game Studios、skill 或其他 AI 內部工具；專案建立後，在資料夾裡啟動 Claude Code（`claude`）或 Codex，直接用自然語言描述要做的遊戲即可。

內容邊界（`-profile`）是開發期間的**內容上限，不是要求**，也不等於正式分級結果：選 `strong` 不代表 AI 會主動加入恐怖、暴力或酒類，只代表遊戲需要時可以用到這個程度。不確定時用預設的 `general`，之後要提高時由開發者修改 `playtogether.json`（AI 會先提出 `CONTENT_PROFILE_CHANGE_REQUEST`）。

| 遊戲類型 | 建議畫面 | `gameServer.mode` | 內容邊界 | 建議 AI 模式 | 起始指令 |
| --- | --- | --- | --- | --- | --- |
| 單人休閒／益智／節奏 | Web | `none` | `general` | `standard` | `pt new -client web -game-server none -profile general -mode standard my-game` |
| 敘事／恐怖解謎／RPG | Godot | `none` | `strong` | `studio` | `pt new -client godot -game-server none -profile strong -mode studio haunted-game` |
| 多人派對／合作 | Web 或 Godot | `playtogether-managed` | `general` | `standard` | `pt new -client web -game-server playtogether-managed -profile general -mode standard party-game` |
| 即時競技／動作對戰 | Godot | `playtogether-managed` | `moderate` | `standard` | `pt new -client godot -game-server playtogether-managed -profile moderate -mode standard arena-game` |
| 已有自己的 Game Server | 依既有遊戲 | `external` | 依既有內容 | `standard` | `pt new -client web -game-server external -profile strong -mode standard existing-game` |
| 快速驗證玩法／Prototype | Web | `none` | `general` | `standard` | `pt new -client web -game-server none -profile general -mode standard prototype-game` |
| 牌桌／卡牌（撲克、麻將） | Web | `playtogether-managed` | `general` | `standard` | `pt new -client web -game-server playtogether-managed -profile general -table-game poker -mode standard poker-night` |

### 單人休閒、益智、節奏遊戲

先用 Web 做最小可玩版本通常最快：

```sh
pt new -client web -game-server none -profile general -mode standard my-puzzle
cd my-puzzle
claude
```

接著直接告訴 AI：

> 幫我做一款手機直式的單人益智遊戲。先定義核心玩法循環，再做一個 5 分鐘內能玩完的 Prototype。

`gameServer.mode: none` 的正式遊戲不需要部署 Game Server，也不需要 `serverKeyId` 或 `serverUrl`；專案仍會附 `server/`，只供本機模擬、SDK 範例與 smoke test 使用。

### 敘事、恐怖解謎、RPG

這類遊戲通常需要較多世界觀、關卡、謎題與故事規劃，建議用 Godot + `studio`：

```sh
pt new -client godot -game-server none -profile strong -mode studio haunted-school
cd haunted-school
claude
```

接著可以直接說：

> 我要做一款 20～30 分鐘的心理恐怖解謎遊戲，以台灣校園為背景。先規劃核心體驗、故事真相、第一組謎題與 playable vertical slice，再開始實作。

恐怖、犯罪題材建議 `strong`；暴力、血腥的程度以 `docs/content/CONTENT_PROFILE.md` 為上限。描寫歷史或政治迫害時在 `sensitiveContent` 宣告（需要平台審核，不是錯誤）；玩聚不收限制級內容。

開發者不需要自己選 Game Studios 的 agent、skill 或 slash command；Claude 會依 `CLAUDE.md` 與 `docs/ai/AI_WORKFLOW.md` 自動判斷何時先規劃、何時進入實作。

目前 `studio` 模式只支援 Godot。

### 多人派對、合作遊戲

如果沒有既有後端，先用玩聚範本 Game Server：

```sh
pt new -client web -game-server playtogether-managed -profile general -mode standard party-game
cd party-game
claude
```

接著可以說：

> 幫我做一款 4 人合作派對遊戲，每局 3 分鐘。先做一個只有單一核心玩法的可玩版本，伺服器負責判定結果、斷線重連與一局生命週期。

多人專案的正式結果、分數與可信任狀態不要只放在畫面端；AI 會依專案宣告保留玩聚的連線與 server-authoritative 架構。

### 即時競技、動作對戰

需要較多角色控制、碰撞、動畫或即時操作時，可以從 Godot 開始：

```sh
pt new -client godot -game-server playtogether-managed -profile moderate -mode standard arena-game
cd arena-game
claude
```

接著可以說：

> 幫我做一個 2～4 人即時競技 Prototype。先定義 client prediction 可以做到哪裡、哪些結果一定由伺服器判定，再做最小可玩版本與延遲／斷線測試。

不要讓 AI 自行改成 Godot Multiplayer、ENet、P2P 或其他連線框架；連線架構以 `PROJECT_CONTEXT.md` 與 `playtogether.json` 的宣告為準。

### 已有自己的 Game Server

第三方遊戲已經有後端時，不需要遷移到玩聚範本伺服器：

```sh
pt new -client web -game-server external -profile strong -mode standard existing-game
cd existing-game
claude
```

接著可以說：

> 這個專案已有正式 Game Server。先閱讀現有 protocol、登入、session 與測試方式，保留目前架構，只補上 PlayTogether 需要的啟動碼兌換、SDK 與平台生命週期整合。

`external` 模式下，`server/` 仍會產生，但只作為玩聚整合參考與本機模擬；正式環境使用開發者自己的 Game Server，`serverUrl` 填它的對外網址。`-profile` 依既有遊戲的實際內容選（範例用 `strong`）。

### 快速驗證新玩法

還不確定遊戲值不值得做完整版本時，先用 Web + `standard` 做最小 Prototype：

```sh
pt new -client web -game-server none -profile general -mode standard prototype-game
cd prototype-game
claude
```

例如：

> 我有一個玩家控制龍捲風吸收物件並破壞場景的點子。先不要做完整遊戲，幫我定義核心循環與成功標準，只做能驗證「控制龍捲風是否有趣」的第一個 Prototype。

預設流程是：

**概念 → 核心循環 → 最小可玩版本 → 實際遊玩 → 再決定是否擴充**

> **CI／自動化提醒**：`pt new -y`、非互動環境或 CI 在沒有明確指定 `-mode` 時，預設使用 `lightweight`，不下載任何第三方 AI 內容；需要 skill 或 studio 時請明確指定 `-mode standard` 或 `-mode studio`。沒有指定 `-profile` 時預設 `general`（和精靈相同）；`-profile custom` 只能在終端機逐項設定。

## 指令

| 指令 | 說明 |
| --- | --- |
| `pt new [-client web\|godot] [-server go\|node] [-orientation portrait\|landscape\|any] [-game-server playtogether-managed\|external\|none] [-mode standard\|studio\|lightweight\|custom] [-skills none\|core\|full] [-ai studios] [-platform 平台網址] [-y] [名稱]` | 建立新專案（前端＋遊戲伺服器，已接好 SDK、模擬模式、跨來源設定、AI 工作流程）；`-orientation` 選直式、橫式或兩者都支援（預設直式） |
| `pt dev [-addr 127.0.0.1:8090] [-players N] [-smoke]` | 本機模擬模式（Godot 專案會先匯出）：啟動範本／參考伺服器並開啟玩家分頁，預設多人專案 2 個、單人專案（`multiplayer.enabled: false`）1 個，`-players` 可以明確指定；`-smoke` 啟動驗證：啟動、確認首頁與 SDK 載入得到後結束 |
| `pt build` | Godot 專案：用無頭模式匯出網頁版到 `build/` 並放上 SDK 轉接檔（網頁專案不需要） |
| `pt check [-local] [建置資料夾]` | 上傳前檢查：SDK、私鑰、廣告與追蹤工具、WebRTC、遊戲伺服器架構與網址設定、音檔都記錄在 `art/SOURCES.md`；`-local` 在部署前使用，`platform`、`serverUrl` 還沒填只提醒；`gameServer.mode: none` 不需要 `serverUrl` |
| `pt keygen -purpose upload\|server [-out 檔案]` | 產生 Ed25519 金鑰；公鑰交給玩聚登記，私鑰只在你手上 |
| `pt upload -version <版本> [-notes 說明] [-no-build] [-orientation portrait\|landscape\|any]` | 上傳版本（設定從 `playtogether.json` 讀，包括畫面方向；Godot 專案先匯出；上傳前自動檢查） |
| `pt sdk status` | 比較專案的 SDK、這個 `pt` 內嵌的 SDK、最新版 `pt` 內嵌的 SDK，並列出被修改過的 SDK 檔案 |
| `pt sdk update [-force]` | 把專案裡的 SDK 更新成這個 `pt` 內嵌的版本（專案的 SDK 比較新時不會降版，除非 `-force`） |
| `pt sdk export <資料夾>` | 把 SDK 檔案（瀏覽器、Godot、Go、Node.js）寫到資料夾，接到不是 `pt new` 建立的既有專案 |
| `pt version` | 顯示 `pt` 與內嵌 SDK 的版本，並檢查有沒有新版 |

## 這個 repo 裡有什麼

| 路徑 | 內容 |
| --- | --- |
| `cmd/pt/` | `pt` 的原始碼（只用 Go 標準函式庫） |
| `cmd/pt/templates/` | `pt new` 的專案範本 |
| `cmd/pt/assets/sdk/` | 內嵌的玩聚 SDK（瀏覽器、Node.js、Go），由玩聚從 SDK 原始碼同步，請不要直接修改 |
| `scripts/` | CI 用：只下載 Godot 網頁匯出範本、在瀏覽器測 Godot 範本握手；`ccgs-manifest.py` 產生 Claude Code Game Studios 的檔案清單 |
| `internal/playtogether/` | `pt` 自己用的 Go SDK（上傳時的請求簽章），同樣由玩聚同步 |

## 回報問題

### 新版提醒

在終端機執行 `pt` 指令時，最多一天檢查一次 GitHub 上最新的 `pt` 版本，有新版就在指令結束後提醒一行（附更新方式）。只讀 GitHub 的「最新版本」頁面轉址，不送任何專案資料，也不會自動下載或替換執行檔。CI（有 `CI` 環境變數）、輸出不是終端機、自己編譯的版本不檢查；`PT_NO_UPDATE_CHECK=1` 可以關閉。`pt version` 會立即檢查一次。更新 `pt` 後，在專案裡執行 `pt sdk update` 把 SDK 換成新版。

SDK 只隨 `pt` 發佈：`pt sdk status` 會查最新版 `pt` 內嵌的 SDK 版本。在專案裡執行 `pt dev`、`pt build` 時，專案的 SDK 比這個 `pt` 內嵌的舊（或新）會提醒一行；`pt check`、`pt upload` 列為警告，不會擋下上傳。

使用上的問題或建議請開 [Issue](https://github.com/game-dev-zone/play-together-ctl/issues)。**資安問題請不要公開開 Issue**，見 [SECURITY.md](SECURITY.md)。

## 授權

[MIT](LICENSE)
