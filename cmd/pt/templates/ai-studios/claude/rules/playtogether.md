---
paths:
  - "**/*.gd"
  - "**/*.tscn"
  - "project.godot"
  - "export_presets.cfg"
  - "server/**"
---

# 玩聚平台（由 pt new -ai studios 加入）

完整規則在專案根目錄的 `PLAYTOGETHER.md`，以下是寫程式時最常踩到的：

- 實作技巧（GDScript、場景、UI、動畫、音效、手感、效能、測試）先查 `.claude/skills/` 的遊戲開發 skill；Studios 決定「要做什麼、為什麼」，skill 決定「怎麼做」。
- 改連線或遊戲伺服器相關程式前，先確認 `docs/ai/PROJECT_CONTEXT.md` 的多人與遊戲伺服器模式，保留宣告的架構。
- 多人遊戲的分數、勝負、回合流程由遊戲伺服器判定；GDScript 只送玩家操作，不要在畫面端決定結果。
- 玩家身分：用 `PlayTogether` autoload 取得啟動碼，交給遊戲伺服器兌換；不要自建登入、帳號、配對。
- 不要用 `JavaScriptBridge.eval`；`HTTPRequest` 用 `config.gd` 的完整網址。
- `export_presets.cfg`：`variant/thread_support=false`、行動裝置 VRAM 壓縮關閉、Head Include 載入 `playtogether-sdk-godot.iife.js`，不要改。
- 中文用 `fonts/` 的字型。
- 不要加入任何分析、追蹤、廣告 SDK；遵守 `PLAYTOGETHER.md` 的玩聚平台硬性規則（現實經濟價值的玩法：真實金錢下注、點數下注、提領、兌換、可轉讓現實價值、付費隨機獎勵由玩聚平台與律師決定，照實宣告；遊戲不能自己接付款或改點數，平台開放前不實作）。牌、骰子、籌碼、Bet／Call／Raise 本身不違規（Table Game ≠ Real-Money Gambling）。
- 內容不超過 `docs/content/CONTENT_PROFILE.md` 的內容邊界（上限，不是要求）；超過時提出 `CONTENT_PROFILE_CHANGE_REQUEST`，不自行提高。玩法與經濟宣告在 `docs/content/GAMEPLAY_ECONOMY.md`。能否上架由玩聚平台審核決定。
- 改完跑 `pt check`，再用 `pt dev` 實際玩一次。
