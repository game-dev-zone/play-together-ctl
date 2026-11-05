# 03-board-score-tokens：一般桌遊（計分 token）

## 開發者的需求

> 做一款 2 到 6 人的骰子桌遊：輪流擲骰前進，走到格子上收集資源 token，最先湊齊三種資源的人獲勝。不用下注，token 只是計分用。

（`request.txt`）

## 建立

```sh
pt new -y -game-server playtogether-managed -profile general -table-game board df-board-score-tokens
cd df-board-score-tokens
```

```sh
pt content sync
pt check -local
```

## 關鍵宣告（`playtogether.json`）

| 欄位 | 值 |
| --- | --- |
| `gameplay.tableGame.enabled` | `true` |
| `gameplay.tableGame.category` | `"board"` |
| `gameplay.tableGame.scoreTokens` | `"allowed"` |
| `gameplay.tableGame.wageringTokens` | `"none"` |
| `gameplay.tableGame.virtualWagering` | `"none"` |
| `gameplay.tableGame.tokenPersistence` | `"none"` |

## 預期結果：`PASS`

一般桌遊的計分或資源 token（`scoreTokens: allowed`）本身**不需要審核**；沒有下注用的籌碼（`wageringTokens: none`）、沒有虛擬下注（`virtualWagering: none`），桌遊類型（`board`）也不在需要審核的類型裡，所以是 PASS。

必要的檢查結果：

- 沒有

完整的比對條件在 `expected.json`（`examples/dogfood/verify.sh` 會檢查）。
