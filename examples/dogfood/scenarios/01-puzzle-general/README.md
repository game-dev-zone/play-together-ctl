# 01-puzzle-general：單人益智（一般內容）

## 開發者的需求

> 幫我做一款單人的益智遊戲：把顏色相同的方塊滑到一起消除，一關大約三分鐘，畫面明亮可愛，沒有暴力或恐怖元素。

（`request.txt`）

## 建立

```sh
pt new -y -game-server none -profile general df-puzzle-general
cd df-puzzle-general
```

```sh
pt content sync
pt check -local
```

## 關鍵宣告（`playtogether.json`）

| 欄位 | 值 |
| --- | --- |
| `contentProfile.profile` | `"general"` |
| `gameplay.tableGame.enabled` | `false` |
| `gameServer.mode` | `"none"` |

## 預期結果：`PASS`

一般內容（`general`）、沒有牌桌玩法、沒有敏感題材、經濟全部 `false`，沒有任何需要平台特別審核的項目。單人遊戲不需要遊戲伺服器（`gameServer.mode: none`）。

必要的檢查結果：

- 沒有

完整的比對條件在 `expected.json`（`examples/dogfood/verify.sh` 會檢查）。
