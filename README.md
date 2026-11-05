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

`PLAYTOGETHER.md` 的「素材」一節建議用 Codex 生成圖片：整套素材共用風格卡 `art/STYLE.md`（範本附一套明亮派對風格：形狀、線條、平塗上色、調色盤色碼、光線、排除項目），先做一張樣板，其他素材都用 `-i` 參考樣板生成，降低「每張各自生成」的 AI 感；生成前同時讀 `docs/content/CONTENT_RATING.md`（內容分級）、提示詞寫明平台硬性規則、看過再用，每個素材記錄在 `art/SOURCES.md`。範本內建程式合成的預設音效（網頁版 `web/sfx.js`、Godot 版 `main.gd`，沒有音檔），並接上平台的「遊戲聲音」開關；要用音檔或音樂時照「音效與音樂」一節的來源順序（程式合成 → CC0 → AI 生成付費方案 → 確認沒有登記 Content ID 的免費音樂）。

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

### 內容分級（`-rating`）

開發者依遊戲內容選擇分級，AI 依分級工作。設定在 `playtogether.json` 的 `contentRating`，`pt new` 依它產生 `docs/content/CONTENT_RATING.md`（所有 AI 模式都有）。**這是開發期間的內容邊界／目標分級，不等於正式分級結果；最終分級依實際內容與玩聚審核決定。**

| `-rating` | 目標年齡 | 內容 |
| --- | --- | --- |
| `general` | 全年齡 | 全齡休閒、益智、家庭與派對 |
| `12plus` | 12+（輔12方向） | 輕度戰鬥、輕度恐怖、幻想暴力、少量粗話 |
| `15plus`（預設） | 15+（輔15方向） | 可依設定包含較強的恐怖、犯罪題材、酒類、中度暴力、少量血跡（玩聚目前接受的上限） |
| `custom` | 自選 | 在終端機逐項設定目標年齡與暴力、血腥、恐怖、性內容、粗俗語言、酒、菸、隨機機制（不能超過 15+ 方向）；非互動時回錯誤 |

- **分級是上限，不是要求**：AI 不會因為是 15+ 就主動加入暴力、血腥、酒類、性暗示、粗俗語言；要加入超過分級的內容時，AI 提出 `CONTENT_RATING_CHANGE_REQUEST`，由開發者決定。
- **玩聚平台硬性規則**（`PLAYTOGETHER.md`）任何分級都不能覆蓋：玩聚目前最高接受 15+ 方向（不接受限制級 18+）、真實金錢下注、下注押注、付費抽獎或轉蛋、模擬賭博（`gambling.realMoney`、`wagering`、`paidRandomReward` 一律 `false`，`simulated` 一律 `none`）、性化未成年人、未授權的真實品牌與可辨識的真人。骰子、桌遊牌具、隨機事件、隨機掉落等一般機制可以（`gambling.chanceMechanics`）；遊戲可以有未成年角色。
- `art/STYLE.md` 只管視覺風格，產生素材前同時讀 `PLAYTOGETHER.md` 與 `CONTENT_RATING.md`。
- `pt check` 只檢查設定的格式、平台上限與硬性規則，不判斷遊戲實際內容，也不推論法律分級。舊專案沒有 `contentRating` 時只提醒，不擋上傳。

| 選項 | 可用值 | 說明 |
| --- | --- | --- |
| `-client` | `web`（預設）、`godot` | 遊戲前端。`web` 產生 `web/`；`godot` 產生 Godot 專案 `game/`，匯出到 `build/` |
| `-server` | `go`（預設）、`node` | 遊戲伺服器（兌換啟動碼、判定勝負）|
| `-template` | `web-go`、`web-node`、`godot-go`、`godot-node` | 舊寫法，等於同時指定上面兩個 |
| `-game-server` | `playtogether-managed`（預設）、`external`、`none` | 遊戲伺服器架構（見上方）；同時決定 `multiplayer.enabled`（`none` 是單人） |
| `-rating` | `general`、`12plus`、`15plus`（預設）、`custom` | 內容分級（見上方）；`-y`、非互動與 CI 沒指定時也用 `15plus` |
| `-mode` | `standard`、`studio`、`lightweight`、`custom` | AI 開發模式（見上方）；`studio` 沒指定畫面時自動用 Godot |
| `-skills` | `full`、`core`、`none` | 自訂：遊戲開發 skill 的範圍 |
| `-ai` | `studios`、`none` | 自訂：加入 Claude Code Game Studios（只限 Godot；只指定它時自動用 Godot） |
| `-y` | | 不詢問，沒指定的項目用預設值 |

在終端機執行 `pt new` 時，沒用參數指定的項目（名稱、畫面、是否多人與遊戲伺服器、伺服器語言、方向、AI 開發模式）會逐步詢問，最後顯示對應的完整指令。CI（環境變數 `CI` 有值）或腳本裡（不是終端機）不會詢問；沒指定 AI 模式時用 `lightweight`，不需要網路。

既有專案想改用新的 AI 工作流程：用 `pt new` 以相同設定建一個暫存專案，把 `docs/ai/`、`CLAUDE.md`、`AGENTS.md` 複製過去（`CLAUDE.md`、`AGENTS.md` 改過的話手動合併），並在 `playtogether.json` 加上 `aiMode`、`multiplayer`、`gameServer`。

不需要金鑰、不需要連線到玩聚：直接開啟時 SDK 自動進入模擬模式。

## 依遊戲類型開始

不確定參數怎麼選時，可以先從遊戲類型決定。下面的指令都是起點，不需要先了解 Game Studios、skill 或其他 AI 內部工具；專案建立後，在資料夾裡啟動 Claude Code（`claude`）或 Codex，直接用自然語言描述要做的遊戲即可。

| 遊戲類型 | 建議畫面 | `gameServer.mode` | 建議 AI 模式 | 起始指令 |
| --- | --- | --- | --- | --- |
| 單人休閒／益智／節奏 | Web | `none` | `standard` | `pt new -client web -game-server none -mode standard my-game` |
| 敘事／恐怖解謎／RPG | Godot | `none` | `studio` | `pt new -client godot -game-server none -mode studio haunted-game` |
| 多人派對／合作 | Web 或 Godot | `playtogether-managed` | `standard` | `pt new -client web -game-server playtogether-managed -mode standard party-game` |
| 即時競技／動作對戰 | Godot | `playtogether-managed` | `standard` | `pt new -client godot -game-server playtogether-managed -mode standard arena-game` |
| 已有自己的 Game Server | 依既有遊戲 | `external` | `standard` | `pt new -client web -game-server external -mode standard existing-game` |
| 快速驗證玩法／Prototype | Web | `none` | `standard` | `pt new -client web -game-server none -mode standard prototype-game` |

### 單人休閒、益智、節奏遊戲

先用 Web 做最小可玩版本通常最快：

```sh
pt new -client web -game-server none -mode standard my-puzzle
cd my-puzzle
claude
```

接著直接告訴 AI：

> 幫我做一款手機直式的單人益智遊戲。先定義核心玩法循環，再做一個 5 分鐘內能玩完的 Prototype。

`gameServer.mode: none` 的正式遊戲不需要部署 Game Server，也不需要 `serverKeyId` 或 `serverUrl`；專案仍會附 `server/`，只供本機模擬、SDK 範例與 smoke test 使用。

### 敘事、恐怖解謎、RPG

這類遊戲通常需要較多世界觀、關卡、謎題與故事規劃，建議用 Godot + `studio`：

```sh
pt new -client godot -game-server none -mode studio haunted-school
cd haunted-school
claude
```

接著可以直接說：

> 我要做一款 20～30 分鐘的心理恐怖解謎遊戲，以台灣校園為背景。先規劃核心體驗、故事真相、第一組謎題與 playable vertical slice，再開始實作。

開發者不需要自己選 Game Studios 的 agent、skill 或 slash command；Claude 會依 `CLAUDE.md` 與 `docs/ai/AI_WORKFLOW.md` 自動判斷何時先規劃、何時進入實作。

目前 `studio` 模式只支援 Godot。

### 多人派對、合作遊戲

如果沒有既有後端，先用玩聚範本 Game Server：

```sh
pt new -client web -game-server playtogether-managed -mode standard party-game
cd party-game
claude
```

接著可以說：

> 幫我做一款 4 人合作派對遊戲，每局 3 分鐘。先做一個只有單一核心玩法的可玩版本，伺服器負責判定結果、斷線重連與一局生命週期。

多人專案的正式結果、分數與可信任狀態不要只放在畫面端；AI 會依專案宣告保留玩聚的連線與 server-authoritative 架構。

### 即時競技、動作對戰

需要較多角色控制、碰撞、動畫或即時操作時，可以從 Godot 開始：

```sh
pt new -client godot -game-server playtogether-managed -mode standard arena-game
cd arena-game
claude
```

接著可以說：

> 幫我做一個 2～4 人即時競技 Prototype。先定義 client prediction 可以做到哪裡、哪些結果一定由伺服器判定，再做最小可玩版本與延遲／斷線測試。

不要讓 AI 自行改成 Godot Multiplayer、ENet、P2P 或其他連線框架；連線架構以 `PROJECT_CONTEXT.md` 與 `playtogether.json` 的宣告為準。

### 已有自己的 Game Server

第三方遊戲已經有後端時，不需要遷移到玩聚範本伺服器：

```sh
pt new -client web -game-server external -mode standard existing-game
cd existing-game
claude
```

接著可以說：

> 這個專案已有正式 Game Server。先閱讀現有 protocol、登入、session 與測試方式，保留目前架構，只補上 PlayTogether 需要的啟動碼兌換、SDK 與平台生命週期整合。

`external` 模式下，`server/` 仍會產生，但只作為玩聚整合參考與本機模擬；正式環境使用開發者自己的 Game Server，`serverUrl` 填它的對外網址。

### 快速驗證新玩法

還不確定遊戲值不值得做完整版本時，先用 Web + `standard` 做最小 Prototype：

```sh
pt new -client web -game-server none -mode standard prototype-game
cd prototype-game
claude
```

例如：

> 我有一個玩家控制龍捲風吸收物件並破壞場景的點子。先不要做完整遊戲，幫我定義核心循環與成功標準，只做能驗證「控制龍捲風是否有趣」的第一個 Prototype。

預設流程是：

**概念 → 核心循環 → 最小可玩版本 → 實際遊玩 → 再決定是否擴充**

> **CI／自動化提醒**：`pt new -y`、非互動環境或 CI 在沒有明確指定 `-mode` 時，預設使用 `lightweight`，不下載任何第三方 AI 內容。需要 skill 或 studio 時請明確指定 `-mode standard` 或 `-mode studio`。

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
