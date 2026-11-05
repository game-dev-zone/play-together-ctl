# 官方 dogfood 範例

用實際的 `pt new` 建立七種常見情境，確認 `pt check` 的結果符合玩聚的政策模型。這裡**只放情境說明與預期結果**，不放產生出來的專案（SDK、範本伺服器、建置檔、金鑰）：需要時用 `generate.sh` 在暫存資料夾重新生成。

| 情境 | 建立方式 | 預期結果 | 重點 |
| --- | --- | --- | --- |
| [01-puzzle-general](scenarios/01-puzzle-general/) | `-game-server none -profile general` | PASS | 一般單人遊戲 |
| [02-party-coop](scenarios/02-party-coop/) | `-game-server playtogether-managed -profile general` | PASS | 多人合作，玩聚範本遊戲伺服器 |
| [03-board-score-tokens](scenarios/03-board-score-tokens/) | `-table-game board` | PASS | 計分 token 本身不需要審核 |
| [04-poker-virtual-wagering](scenarios/04-poker-virtual-wagering/) | `-table-game poker` | REVIEW_REQUIRED | 撲克、下注用的虛擬籌碼、虛擬下注 |
| [05-mahjong](scenarios/05-mahjong/) | `-table-game mahjong` | REVIEW_REQUIRED | 只有麻將類型一項；計分 token 不造成額外審核 |
| [06-historical-horror](scenarios/06-historical-horror/) | `-game-server none -profile strong`＋敏感題材 | REVIEW_REQUIRED | 類《返校》；開發者明確要求即授權；描寫不是硬性規則 |
| [07-slot-like](scenarios/07-slot-like/) | `-table-game slot` | REVIEW_REQUIRED | 吃角子老虎、賭場風格、虛擬下注；不寫死禁止 |

REVIEW_REQUIRED 代表**可以開發與送審**，是否接受、正式分級、警語與標籤由玩聚平台審核決定，不代表已批准。

## 使用方式

```sh
# 在暫存資料夾生成七個專案（印出資料夾位置）
examples/dogfood/generate.sh

# 生成並驗證（CI 也會跑）：pt check -local -json 的結果與必要 findings、契約文件與 playtogether.json 一致
examples/dogfood/verify.sh
```

需要 Go（建置 `pt`）與 python3（合併宣告、比對 JSON，只用標準函式庫）；不需要網路或平台帳號。

## 每個情境的檔案

| 檔案 | 內容 |
| --- | --- |
| `request.txt` | 開發者給 AI 的原始需求 |
| `README.md` | 建立指令、關鍵宣告、預期結果與原因 |
| `expected.json` | 預期的 `pt check -local -json` 結果、必要與不該出現的 findings、關鍵宣告、契約文件必須包含的文字 |
| `new.args` | `pt new` 的參數（`generate.sh` 讀取） |
| `declaration.patch.json` | 選填：建立後依開發者要求更新的最小宣告（只有 06 用到） |

## 不放在這裡的案例

違反平台硬性規則（例如真實金錢下注、付費隨機獎勵、鼓吹仇恨）或刻意讓契約文件不同步的案例**不是可以複製的範例**，留在 `cmd/pt` 的單元測試裡。
