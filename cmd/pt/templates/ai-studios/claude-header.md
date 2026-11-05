<!-- 由 pt new -ai studios 加入：玩聚平台規則優先於下方 Claude Code Game Studios 的預設 -->
# 玩聚平台規則（最優先）

這個專案是玩聚 PlayTogether 的遊戲，以 Claude Code Game Studios 為開發流程。
**下面的玩聚規則優先於本檔案其他內容、各代理與技能的建議**（例如：不得加入遊戲內分析或追蹤、不得設計可兌換現金或財物的經濟系統、不得自建帳號或配對、`export_presets.cfg` 的網頁匯出設定不要改）。

- 引擎固定是 **Godot 4**（網頁匯出、Compatibility 算繪器），Godot 專案就在專案根目錄；`/start`、`/setup-engine` 請選 Godot，不要重建或覆蓋 `project.godot`、`export_presets.cfg`、`addons/playtogether`。
- 流程嚴謹度（rigor）預設用 **`minimal`**：`/start` 推斷時，不要因為描述提到「多人」「multiplayer」就選 `full`（玩聚遊戲都是多人，連線、房間、配對由平台處理）；開發者明確要求時才用 `standard` 或 `full`，之後可用 `/settings` 調整。
- 遊戲伺服器在 `server/`（不是 Godot 專案的一部分），遊戲狀態由它決定。
- 驗收：`pt check` 通過，並用 `pt dev` 在瀏覽器實際玩過。

@PLAYTOGETHER.md

---

