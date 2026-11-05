# 07-slot-like：類吃角子老虎（賭場風格）

## 開發者的需求

> 做一款派對用的吃角子老虎小遊戲：每人每局有 500 枚虛擬籌碼可以下注拉霸，畫面像賭場的拉霸機台，籌碼每局重置，最後籌碼最多的人獲勝，不能購買或兌換任何東西。

（`request.txt`）

## 建立

```sh
pt new -y -game-server playtogether-managed -profile general -table-game slot df-slot-like
cd df-slot-like
```

```sh
pt content sync
pt check -local
```

## 關鍵宣告（`playtogether.json`）

| 欄位 | 值 |
| --- | --- |
| `gameplay.tableGame.category` | `"slot"` |
| `gameplay.tableGame.presentation` | `"casinoLike"` |
| `gameplay.tableGame.wageringTokens` | `"allowed"` |
| `gameplay.tableGame.virtualWagering` | `"allowed"` |
| `economy.realMoneyWagering` | `false` |

## 預期結果：`REVIEW_REQUIRED`

經濟全部 `false`，沒有硬性規則問題；但吃角子老虎類型、賭場風格呈現（`casinoLike`）、下注用的虛擬籌碼、虛擬下注都需要玩聚平台審核。`pt` 不寫死「吃角子老虎一律禁止」，是否接受由平台審核決定。

必要的檢查結果：

- REVIEW_REQUIRED：`gameplay.tableGame.category：slot`
- REVIEW_REQUIRED：`gameplay.tableGame.presentation：casinoLike`
- REVIEW_REQUIRED：`gameplay.tableGame.wageringTokens：allowed`
- REVIEW_REQUIRED：`gameplay.tableGame.virtualWagering：allowed`

完整的比對條件在 `expected.json`（`examples/dogfood/verify.sh` 會檢查）。
