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

- 分數、勝負、回合流程由 `server/` 判定；GDScript 只送玩家操作，不要在畫面端決定結果。
- 玩家身分：用 `PlayTogether` autoload 取得啟動碼，交給遊戲伺服器兌換；不要自建登入、帳號、配對。
- 不要用 `JavaScriptBridge.eval`；`HTTPRequest` 用 `config.gd` 的完整網址。
- `export_presets.cfg`：`variant/thread_support=false`、行動裝置 VRAM 壓縮關閉、Head Include 載入 `playtogether-sdk-godot.iife.js`，不要改。
- 中文用 `fonts/` 的字型。
- 不要加入任何分析、追蹤、廣告 SDK；不要有賭博、博弈或可兌換現金或財物的機制。
- 改完跑 `pt check`，再用 `pt dev` 實際玩一次。
