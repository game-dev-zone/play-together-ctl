<!-- 由 pt new 加入（PlayTogether）：玩聚平台規則與 AI 工作流程優先於下方 Claude Code Game Studios 的預設 -->
# 玩聚平台規則與 AI 工作流程（最優先）

這個專案是玩聚 PlayTogether 的遊戲，以 Claude Code Game Studios 作為創意與規劃流程。**下面匯入的玩聚文件優先於本檔案其他內容、各 agent 與 skill 的建議**。

**你負責自動路由**：開發者不需要了解 Studios、skill、agent 或 slash command，只會用一般的話描述想做的遊戲或功能。由你判斷要先做創意設計還是直接實作，並在適當時自己使用 Studios 的流程（包含實作 story 與完成檢查）；**不要要求開發者輸入指令**，也不要問嚴謹度或自動化程度（已設定在 `project.yaml`，規則見 `docs/ai/AI_WORKFLOW.md`）。

- 開始前依序讀：`PLAYTOGETHER.md`、`docs/ai/AI_WORKFLOW.md`、`docs/ai/PROJECT_CONTEXT.md`、`docs/content/CONTENT_PROFILE.md`、`docs/content/GAMEPLAY_ECONOMY.md`、`docs/ai/CURRENT_TASK.md`（都在下方匯入）。
- 引擎固定是 **Godot 4**（網頁匯出、Compatibility 算繪器），Godot 專案就在專案根目錄；設定引擎時選 Godot，不要重建或覆蓋 `project.godot`、`export_presets.cfg`、`addons/playtogether`。
- 改連線或遊戲伺服器相關程式前，先確認 `PROJECT_CONTEXT.md` 宣告的架構並保留它。
- 發佈、部署、上傳由玩聚流程負責（`pt check`、`pt upload`、審核），不使用 Studios 的發佈流程。
- 驗收：照 `docs/ai/AI_WORKFLOW.md`「完成前驗證」（`pt check -local` 通過，並用 `pt dev` 實際玩過）。

@PLAYTOGETHER.md
@docs/ai/AI_WORKFLOW.md
@docs/ai/PROJECT_CONTEXT.md
@docs/content/CONTENT_PROFILE.md
@docs/content/GAMEPLAY_ECONOMY.md
@docs/ai/CURRENT_TASK.md

<!-- PlayTogether 區塊結束；以下是 Claude Code Game Studios 的原始 CLAUDE.md -->

---

