# 04-poker-virtual-wagering：撲克（虛擬下注）

## 開發者的需求

> 做一款和朋友一起玩的德州撲克：每局每人發 1000 枚虛擬籌碼，可以 Check、Call、Raise、Fold、All-in；籌碼每局重置，不能購買、兌換或轉讓。

（`request.txt`）

## 建立

```sh
pt new -y -game-server playtogether-managed -profile general -table-game poker df-poker-virtual-wagering
cd df-poker-virtual-wagering
```

```sh
pt content sync
pt check -local
```

## 關鍵宣告（`playtogether.json`）

| 欄位 | 值 |
| --- | --- |
| `gameplay.tableGame.category` | `"poker"` |
| `gameplay.tableGame.wageringTokens` | `"allowed"` |
| `gameplay.tableGame.virtualWagering` | `"allowed"` |
| `gameplay.tableGame.tokenPersistence` | `"round"` |
| `economy.realMoneyWagering` | `false` |
| `economy.purchasableStake` | `false` |

## 預期結果：`REVIEW_REQUIRED`

牌、籌碼、Bet／Call／Raise 本身不違規（Table Game ≠ Real-Money Gambling），經濟全部 `false`，所以沒有硬性規則問題；但撲克類型、下注用的虛擬籌碼、虛擬下注都需要玩聚平台審核（是否接受、正式分級、警語、標籤、文案由平台決定），結果是 REVIEW_REQUIRED（可以開發與送審，不代表已批准）。

必要的檢查結果：

- REVIEW_REQUIRED：`gameplay.tableGame.category：poker`
- REVIEW_REQUIRED：`gameplay.tableGame.wageringTokens：allowed`
- REVIEW_REQUIRED：`gameplay.tableGame.virtualWagering：allowed`

完整的比對條件在 `expected.json`（`examples/dogfood/verify.sh` 會檢查）。
