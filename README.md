# pt：玩聚開發者工具

`pt` 是玩聚 PlayTogether（台灣的瀏覽器多人派對遊戲平台）給第三方開發者的指令列工具：一個指令建立已接好玩聚 SDK 的遊戲專案、在本機用模擬模式開發、上傳前檢查、產生金鑰、上傳版本。完整文件在玩聚開發者文件站 `/docs/pt`。

*`pt` is the command-line tool for third-party developers building games for PlayTogether (玩聚), a browser party-game platform in Taiwan: scaffold a project wired to the PlayTogether SDK, develop locally in mock mode, check and upload builds.*

## 安裝

**下載執行檔**（建議）：到 [Releases](https://github.com/Yanghsuanming/play-together-ctl/releases) 下載對應系統的檔案，改名成 `pt` 放進 PATH。

```sh
# macOS（Apple 晶片），其他系統換成 pt-darwin-amd64、pt-linux-amd64、pt-linux-arm64、pt-windows-amd64.exe
curl -fLo pt https://github.com/Yanghsuanming/play-together-ctl/releases/latest/download/pt-darwin-arm64
chmod +x pt && sudo mv pt /usr/local/bin/
pt version
```

每個版本都附 `SHA256SUMS`，可以用 `shasum -a 256 -c SHA256SUMS --ignore-missing` 檢查。

**用 Go 安裝**（Go 1.24 以上）：

```sh
go install github.com/Yanghsuanming/play-together-ctl/cmd/pt@latest
```

## 五分鐘開始

```sh
pt new my-party-game             # 預設：網頁（HTML／JS）前端＋Go 遊戲伺服器
cd my-party-game
pt dev                           # 本機模擬模式：啟動遊戲伺服器、打開兩個玩家分頁
```

用 Godot 開發：

```sh
pt new -client godot -server go my-godot-game   # 遊戲伺服器也可以用 -server node
cd my-godot-game
pt dev                           # 先用 Godot 無頭模式匯出網頁版，再啟動模擬模式
```

Godot 專案需要 Godot 4.3 以上與網頁匯出範本（Godot 編輯器 → 編輯器 → 管理匯出範本）。`pt` 依序找環境變數 `GODOT`、PATH 裡的 `godot`、macOS 的 `/Applications/Godot.app`。範本已設定好網頁匯出（關閉執行緒支援、載入 SDK）、玩聚外掛，並附上 Noto Sans TC 常用字子集（SIL OFL 1.1），中文不會變成方框。

### 用 AI 開發

每個新專案都附上 `PLAYTOGETHER.md`（玩聚的架構原則、iframe 執行環境限制、審核規則），以及讓 AI 助手先讀它的 `CLAUDE.md`（Claude Code）與 `AGENTS.md`（Codex 等）。

Godot 專案可以加入 [Claude Code Game Studios](https://github.com/donchitos/claude-code-game-studios)（MIT）：一套用 Claude Code 開發遊戲的 AI 角色與流程範本。

```sh
pt new -client godot -ai studios my-godot-game   # 沒加 -ai 時，Godot 專案會詢問；-ai none 不詢問
cd my-godot-game
claude                                            # 第一次執行 /start
```

- 版本固定（目前 1.2.0），下載後逐一比對內嵌清單的 SHA-256，任何一個檔案不符就停止，不留下專案資料夾。範本的 hooks 會在你的電腦上執行，所以 `pt` 不會安裝清單以外的檔案。
- Godot 專案改放在專案根目錄（範本的工具需要 `project.godot` 在根目錄）；`server/`、`design/`、`docs/` 等資料夾加上 `.gdignore`，不會被打包進遊戲。
- `CLAUDE.md` 開頭加上玩聚規則，優先於範本的預設（例如不得加入遊戲內分析、不得設計可兌換現金或財物的經濟系統）；另加 `.claude/rules/playtogether.md`。
- 授權與來源記錄在 `THIRD_PARTY/claude-code-game-studios/`。

| 選項 | 可用值 | 說明 |
| --- | --- | --- |
| `-client` | `web`（預設）、`godot` | 遊戲前端。`web` 產生 `web/`；`godot` 產生 Godot 專案 `game/`，匯出到 `build/` |
| `-server` | `go`（預設）、`node` | 遊戲伺服器（兌換啟動碼、判定勝負）|
| `-template` | `web-go`、`web-node`、`godot-go`、`godot-node` | 舊寫法，等於同時指定上面兩個 |
| `-ai` | `studios`、`none` | 加入 Claude Code Game Studios（只限 Godot）；Godot 專案沒指定時會詢問 |

不需要金鑰、不需要連線到玩聚：直接開啟時 SDK 自動進入模擬模式。

## 指令

| 指令 | 說明 |
| --- | --- |
| `pt new [-client web\|godot] [-server go\|node] [-ai studios\|none] [-platform 平台網址] <名稱>` | 建立新專案（前端＋遊戲伺服器，已接好 SDK、模擬模式、跨來源設定） |
| `pt dev [-addr 127.0.0.1:8090] [-players 2]` | 本機模擬模式（Godot 專案會先匯出） |
| `pt build` | Godot 專案：用無頭模式匯出網頁版到 `build/` 並放上 SDK 轉接檔（網頁專案不需要） |
| `pt check [建置資料夾]` | 上傳前檢查：SDK、私鑰、廣告與追蹤工具、WebRTC、遊戲伺服器網址設定 |
| `pt keygen -purpose upload\|server [-out 檔案]` | 產生 Ed25519 金鑰；公鑰交給玩聚登記，私鑰只在你手上 |
| `pt upload -version <版本> [-notes 說明] [-no-build]` | 上傳版本（設定從 `playtogether.json` 讀；Godot 專案先匯出；上傳前自動檢查） |
| `pt sdk update` | 把專案裡的 SDK 更新成這個 `pt` 內嵌的版本 |
| `pt sdk export <資料夾>` | 把 SDK 檔案（瀏覽器、Godot、Go、Node.js）寫到資料夾，接到不是 `pt new` 建立的既有專案 |
| `pt version` | 顯示 `pt` 與內嵌 SDK 的版本 |

## 這個 repo 裡有什麼

| 路徑 | 內容 |
| --- | --- |
| `cmd/pt/` | `pt` 的原始碼（只用 Go 標準函式庫） |
| `cmd/pt/templates/` | `pt new` 的專案範本 |
| `cmd/pt/assets/sdk/` | 內嵌的玩聚 SDK（瀏覽器、Node.js、Go），由玩聚從 SDK 原始碼同步，請不要直接修改 |
| `scripts/` | CI 用：只下載 Godot 網頁匯出範本、在瀏覽器測 Godot 範本握手；`ccgs-manifest.py` 產生 Claude Code Game Studios 的檔案清單 |
| `internal/playtogether/` | `pt` 自己用的 Go SDK（上傳時的請求簽章），同樣由玩聚同步 |

## 回報問題

使用上的問題或建議請開 [Issue](https://github.com/Yanghsuanming/play-together-ctl/issues)。**資安問題請不要公開開 Issue**，見 [SECURITY.md](SECURITY.md)。

## 授權

[MIT](LICENSE)
