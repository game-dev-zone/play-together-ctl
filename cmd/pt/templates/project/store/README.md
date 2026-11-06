# 上架資料（store/）

`pt upload` 會把這個資料夾的內容和遊戲版本一起上傳；只想換截圖或修說明時用 `pt upload -listing-only`。
平台會**自動裁切、縮放、重新編碼**圖片，上傳後看到的就是最終樣子。能否上架、正式分級、警語、標籤由玩聚平台審核決定。

| 檔案 | 內容 |
| --- | --- |
| `listing.json` | 名稱、簡介（列表上的一行）、完整說明、截圖說明、標籤、人數、一局大約幾分鐘、給審核者的說明（不公開）。**繁中與英文都必填** |
| `icon.png`（或 `.jpg`） | 圖示：正方形或接近正方形，至少 512×512（平台輸出 1024×1024，前台自己加圓角；不要自己加圓角、陰影或透明背景） |
| `screenshots/01.png` … | 截圖 1–10 張（依檔名排序）：**必須是實際遊戲畫面**；依 `playtogether.json` 的畫面方向，直式 9:16（1080×1920）、橫式 16:9（1920×1080），短邊至少 720。用 `pt store screenshot` 擷取 |
| `screenshots/en/01.png` … | 選填：英文介面專用的截圖；沒有時英文頁面用上面的截圖 |
| `cover.png`（或 `.jpg`） | 選填：16:9 封面（精選等大版面用），短邊至少 720 |

- 標籤代碼：`party`、`deduction`、`turn-taking`、`quiz`、`drawing`、`word`、`cooperative`、`team`、`strategy`、`reaction`、`casual`、`music`、`icebreaker`（最多 5 個）。
- 圖片只收 PNG、JPEG，每張最多 10 MB；比例差太多（例如把橫式圖用在直式遊戲）平台不能自動裁切，要換一張。
- 文案不得宣稱或暗示真錢獲利、兌現、提領；圖片不可有真實品牌、可辨識的真人、真實貨幣（`PLAYTOGETHER.md`）。AI 生成的圖示、封面記錄在 `art/SOURCES.md`。
- 只改簡介、說明、截圖說明裡的幾個字（例如錯字）時免審、直接生效，平台事後抽查；名稱、圖片、標籤、人數的修改要審核。
- `pt check` 會檢查這個資料夾；範本文字還沒改或缺圖片時只提醒，`pt upload` 會只上傳版本。
- 截圖：先啟動遊戲（例如 `pt dev -mock`），再執行 `pt store screenshot`（本機 Chrome／Chromium 無頭模式，尺寸剛好符合規格，檔名接在既有截圖後面；`-url`、`-wait`、`-n`、`-locale en` 見 `pt help`）。
- 請 AI 助手（Claude Code、Codex）「幫我準備上架資料」時，它會照 `.claude/skills/playtogether-store/SKILL.md`（Codex：`.agents/skills/`）的步驟做，完成後請你確認再 `pt upload`。
