# 02-party-coop：多人合作派對遊戲

## 開發者的需求

> 我想做 4 到 8 人一起玩的合作派對遊戲：大家在同一個廚房裡分工切菜、煮湯、出餐，時間內完成越多訂單越好，用語音互相喊話。

（`request.txt`）

## 建立

```sh
pt new -y -game-server playtogether-managed -profile general df-party-coop
cd df-party-coop
```

```sh
pt content sync
pt check -local
```

## 關鍵宣告（`playtogether.json`）

| 欄位 | 值 |
| --- | --- |
| `contentProfile.profile` | `"general"` |
| `gameServer.mode` | `"playtogether-managed"` |
| `multiplayer.enabled` | `true` |
| `gameplay.tableGame.enabled` | `false` |

## 預期結果：`PASS`

多人遊戲用玩聚範本遊戲伺服器（`playtogether-managed`）判定分數；一般內容、沒有牌桌玩法、經濟全部 `false`。本機檢查時 `serverUrl` 還沒填只是說明（`-local`），上傳前要填。

必要的檢查結果：

- 沒有

完整的比對條件在 `expected.json`（`examples/dogfood/verify.sh` 會檢查）。
