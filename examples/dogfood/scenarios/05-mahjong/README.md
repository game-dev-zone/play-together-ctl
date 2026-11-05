# 05-mahjong：麻將（計分，沒有下注）

## 開發者的需求

> 做一款四人線上麻將：照台灣麻將規則胡牌，用分數記輸贏，一將打完看誰分數最高。不用任何籌碼下注。

（`request.txt`）

## 建立

```sh
pt new -y -game-server playtogether-managed -profile general -table-game mahjong df-mahjong
cd df-mahjong
```

```sh
pt content sync
pt check -local
```

## 關鍵宣告（`playtogether.json`）

| 欄位 | 值 |
| --- | --- |
| `gameplay.tableGame.category` | `"mahjong"` |
| `gameplay.tableGame.scoreTokens` | `"allowed"` |
| `gameplay.tableGame.wageringTokens` | `"none"` |
| `gameplay.tableGame.virtualWagering` | `"none"` |

## 預期結果：`REVIEW_REQUIRED`

麻將類型需要玩聚平台審核（REVIEW_REQUIRED）。用分數記輸贏是計分 token（`scoreTokens: allowed`），**本身不會造成額外的審核項目**；沒有下注用的籌碼與虛擬下注，所以只有一項需要審核。

必要的檢查結果：

- REVIEW_REQUIRED：`gameplay.tableGame.category：mahjong`

完整的比對條件在 `expected.json`（`examples/dogfood/verify.sh` 會檢查）。
