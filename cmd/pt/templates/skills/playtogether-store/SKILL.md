---
name: playtogether-store
description: 準備玩聚上架資料（store/）：依專案內容起草 store/listing.json（繁中＋英文的名稱、簡介、說明、截圖說明、標籤、人數、給審核者的說明），產生圖示與封面，用 pt store screenshot 擷取實際遊戲截圖，最後用 pt check 檢查到沒有錯誤。使用者說「準備上架資料」「寫簡介」「寫遊戲說明」「做圖示」「截圖」「store」「listing」時使用。要一路上架到送審時用 playtogether-publish（它會呼叫這個 skill）。Use when preparing the PlayTogether store listing, description, icon or screenshots.
---

# 玩聚上架資料（store/）

這個 skill 由 `pt` 提供（`pt new`、`pt init` 會放進專案；`pt init` 重跑會補上缺少的檔案）。上架資料放在專案的 `store/`，`pt upload` 會和遊戲版本一起上傳。**能否上架、正式分級、內容標示、警語、最終標籤由玩聚平台審核決定**，你只準備資料，不做這些判斷。

照下面五個步驟做，不要跳步驟。每一步做完，用一兩句話告訴開發者你做了什麼。

## 1. 讀專案

先讀，再寫：

- `playtogether.json`：`orientation`（`portrait` 直式、`landscape` 橫式、`any` 都支援）、`multiplayer.enabled`、`gameServer.mode`、`devAddr`、`client`（web 或 godot）。
- `PLAYTOGETHER.md`（有的話）、`docs/content/CONTENT_PROFILE.md`、`docs/content/GAMEPLAY_ECONOMY.md`：平台規則、內容邊界、玩法與經濟宣告。
- `docs/` 裡的設計文件（例如 `docs/ai/PROJECT_CONTEXT.md`、`design/`）。
- 遊戲程式的主要畫面與規則：怎麼開始一局、幾個人、一局多久、勝負怎麼判定、有哪些畫面（大廳、遊戲中、結算）。
- 既有的 `store/`：`store/README.md`（格式與規格）、`store/listing.json`、圖片。**開發者已經寫好的內容不要整段改掉**；只補缺的、修錯的，改動前先說明。
- `art/STYLE.md`（美術風格卡）與 `art/SOURCES.md`（素材來源）。

## 2. 起草 `store/listing.json`

格式照 `store/README.md`。範本裡「（請填寫…」和 `TODO` 是還沒填的意思，全部換掉。

| 欄位 | 內容 | 上限（`pt check` 會檢查） |
| --- | --- | --- |
| `name.zh-TW`／`name.en` | 遊戲名稱 | 各 30 字 |
| `summary.zh-TW`／`summary.en` | 列表上的一行簡介：一句話說出好玩的地方 | 繁中 60 字、英文 80 字 |
| `description.zh-TW`／`description.en` | 完整說明：怎麼玩、幾個人、一局多久、特色；可以換行分段 | 各 4000 字 |
| `screenshotCaptions.zh-TW`／`.en` | 每張截圖一句說明，順序和截圖檔名相同 | 每句 60 字、最多 10 句，不要比截圖多 |
| `tags` | 只能用這些代碼：`party`、`deduction`、`turn-taking`、`quiz`、`drawing`、`word`、`cooperative`、`team`、`strategy`、`reaction`、`casual`、`music`、`icebreaker` | 最多 5 個 |
| `players.min`／`players.max` | 實際能玩的人數（照遊戲程式與伺服器的設定，不要猜） | 1–50 |
| `sessionMinutes` | 一局大約幾分鐘 | 1–180 |
| `reviewNotes` | 給審核者看、不公開（見下面） | 4000 字 |

- **繁中與英文都必填**。英文要自然，不是逐字翻譯；名稱可以意譯。
- `reviewNotes` 寫：怎麼開始一局（按哪個按鈕、要等幾個人）、建議幾個人一起測、要特別操作才看得到的內容（隱藏關卡、結算畫面、特殊模式）、`pt check` 列出的每個 REVIEW_REQUIRED 項目的設計理由。
- **一律不寫**：真錢、獲利、賺錢、兌現、提領、出金、現金獎等宣稱或暗示；「已獲玩聚批准」「保證上架」；任何分級（例如「普遍級」「12 歲以上」）、警語、內容標示——這些由平台審核決定。
- 不寫做不到的功能，不誇大（例如沒有排行榜就不要寫排行榜）。

## 3. 圖示與封面

| 圖片 | 檔案 | 規格 |
| --- | --- | --- |
| 圖示（必填） | `store/icon.png`（或 `.jpg`） | 正方形，建議 1024×1024（至少 512×512）；平台輸出 1024×1024 |
| 封面（建議，預設要產生） | `store/cover.png`（或 `.jpg`） | 16:9，建議 1920×1080（短邊至少 720）。前台遊戲頁、房間頁、房間結束畫面的大圖；沒有時用平台的預設圖（`pt check` 會提醒） |

共同規則：PNG 或 JPEG、每張最多 10 MB、長邊最多 6000、最多 1600 萬像素；比例和規格差太多（任一邊要裁掉超過 25%）平台不能自動裁切。

- **圖示和封面都要產生**（封面除非開發者說不要）。封面是橫式的主視覺：和圖示同一個角色、物件與配色，但畫面更完整（場景、氣氛），主體放在中間偏左右的安全區，四周會被裁切或疊上文字的地方不要放重要內容。
- 用**開發者自己的圖片生成工具**產生（例如 Codex 的圖片生成；`PLAYTOGETHER.md` 有說明時照它）。**如果你沒有圖片生成工具，就直接告訴開發者，請他提供圖示（與封面），不要用截圖硬裁成圖示，也不要自己畫一個佔位圖交差。**
- 風格照 `art/STYLE.md`；沒有風格卡時，依遊戲實際畫面的風格與配色。
- **不要自己加圓角、陰影、外框**（前台會自己加圓角），**不要透明背景**（用滿版的不透明背景）。
- **不要文字**：不放遊戲名稱、字母、數字或很小的字（縮小後看不清楚，也不能翻譯）。
- **先讀 `PLAYTOGETHER.md` 的玩聚平台硬性規則（有的話）、`docs/content/CONTENT_PROFILE.md`（內容邊界）與 `docs/content/GAMEPLAY_ECONOMY.md`（玩法宣告）**，提示詞寫明下面的硬性規則：
  - **永遠不行**（平台硬性規則，任何宣告都不能覆蓋）：圖片裡的文字或 Logo、未經授權的真實品牌或商標、可辨識的真人、暗示真錢價值的畫面（真實鈔票或硬幣、兌現、提領）、限制級內容（露骨的性內容、裸露、大量寫實血腥）、任何涉及未成年人的性暗示。
  - **依專案宣告**：暴力、血腥、恐怖、酒類、菸、性暗示等，看內容邊界（`playtogether.json` 的 `contentProfile`）；牌、骰子、籌碼、牌桌看玩法宣告（`gameplay`），賭場風格的畫面要和 `presentation: casinoLike` 宣告一致。宣告允許、遊戲本身也有這些內容時，圖示可以呈現；但**上架圖片不能比遊戲實際內容更強烈**（例如不能用比遊戲更血腥的圖示吸引人）。
  - 遊戲裡虛構的金幣、寶石等一般遊戲道具可以畫。
  - 宣告沒寫清楚、拿不準時，先問開發者，不要自己決定。
- 生成後**親自看過每張圖**再放進 `store/`。原始檔放 `art/generated/`（或專案慣用的位置），`store/` 放最終版本。
- 每張圖在 `art/SOURCES.md` 記一筆：檔案、來源（生成工具與模型）、提示詞、日期、說明。沒有 `art/SOURCES.md` 時建立一個：

  ```markdown
  # 素材來源

  | 檔案 | 來源 | 授權（證明網址） | 日期 | 說明 |
  | --- | --- | --- | --- | --- |
  ```

## 4. 截圖

截圖放在 `store/screenshots/01.png`、`02.png`…（1–10 張，依檔名排序）；英文介面專用的放 `store/screenshots/en/`（選填）。

- **必須是實際遊戲畫面**。不要用生成圖、設計稿、拼貼或修過的圖冒充。
- 方向照 `playtogether.json` 的 `orientation`：直式 1080×1920、橫式 1920×1080；`any` 預設直式。
- 用 `pt store screenshot` 擷取（本機的 Chrome／Chromium 無頭模式，輸出尺寸剛好符合規格，檔名接在既有截圖後面、不覆寫）：
  1. 啟動遊戲。`pt new` 建立的專案：`pt dev -mock`（模擬模式，預設網址就能直接玩），或 `pt dev`（本機假平台）並把它印出的「玩家分頁」網址傳給 `-url`。其他專案用專案自己的啟動方式（例如 `make run`、`npm run dev`），再用 `-url` 指定網址。
  2. `pt store screenshot`（預設網址是 `http://<devAddr>/`、等 5 秒）。常用參數：`-url <網址>`、`-wait 8s`（遊戲載入較慢時加長）、`-n 5 -interval 4s`（在同一個頁面照真實時間每隔 4 秒再拍一張，拍到同一局的不同階段）、`-locale en`（存到 `store/screenshots/en/`）、`-replace`（從 01 開始覆寫）、`-chrome <路徑>`（找不到 Chrome 時）。
  3. 等待照**真實時間**計算，`-n` 的每一張都在同一個頁面擷取（不重新載入）：遊戲會自己往下進行（例如單機練習、示範模式、倒數自動開始）時，一次 `-n 5 -interval 4s` 就能拍到同一局的開始、進行中、結算。`pt store screenshot` 不會點擊畫面；需要操作才會前進的畫面，用遊戲已經有的網址參數或示範模式（例如 `-url "http://127.0.0.1:8090/?demo=practice"`）。遊戲沒有這種方式時，先問開發者能不能加一個只在本機使用的示範參數；不要擅自改遊戲規則。
- 每張截圖都**親自看過**：畫面正確、沒有錯誤訊息、沒有除錯資訊、沒有個資或真實暱稱。不好的就刪掉重拍。`pt store screenshot` 偵測到全黑或全白時不會存檔，通常是還沒載入完（加大 `-wait`）或 WebGL 沒有啟動。
- 本機沒有 Chrome／Chromium 時，告訴開發者，請他用手機或瀏覽器的手機模擬模式手動截圖（直式 9:16、橫式 16:9，短邊至少 720）。
- 截圖數量和 `screenshotCaptions` 對齊（說明可以比截圖少，不能比截圖多）。

## 5. 檢查與交付

1. 執行 `pt check`（還沒填 `platform`、`serverUrl` 時用 `pt check -local`），把 `store/` 相關的錯誤（ERROR_SCHEMA）全部修好；「上架資料還沒準備好」的提示也要消失。其他項目的問題告訴開發者，不要順手改宣告。
2. 列出這次新增或修改的檔案（`store/listing.json`、`store/icon.png`、`store/cover.png`、`store/screenshots/…`、`art/SOURCES.md`），請開發者**確認文字與圖片**。
3. 告訴開發者確認後執行 `pt upload -version <版本>`（只更新上架資料用 `pt upload -listing-only`），審核進度用 `pt status`。**不要自己執行 `pt upload`、送審或 `pt login`**，除非開發者明確要求。開發者要你一路上架到送審（`playtogether-publish`）就是明確要求：做完這裡的步驟後回到那個 skill 繼續。
