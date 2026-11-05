# 06-historical-horror：歷史心理恐怖（類《返校》）

## 開發者的需求

> 我要做一款類似《返校》的單人心理恐怖解謎遊戲：背景是戒嚴時期的校園，描寫白色恐怖、讀書會被告密、學生被審訊與失蹤，反派教官會說出歧視與威權的台詞，史料文件裡也有當時的仇恨用語。要有壓迫感，但不需要血腥畫面。

（`request.txt`）

## 建立

```sh
pt new -y -client web -game-server none -profile strong df-historical-horror
cd df-historical-horror
```

建立後依開發者的要求更新宣告（`declaration.patch.json`），再執行 `pt content sync`：

```json
{
  "sensitiveContent": {
    "historicalPersecution": true,
    "politicalRepression": true,
    "discriminationDepiction": true,
    "hateSpeechDepiction": true
  }
}
```

```sh
pt content sync
pt check -local
```

## 關鍵宣告（`playtogether.json`）

| 欄位 | 值 |
| --- | --- |
| `contentProfile.profile` | `"strong"` |
| `sensitiveContent.historicalPersecution` | `true` |
| `sensitiveContent.politicalRepression` | `true` |
| `sensitiveContent.hateAdvocacy` | `false` |
| `sensitiveContent.encourageProtectedClassViolence` | `false` |

## 預期結果：`REVIEW_REQUIRED`

開發者明確要求這些題材，**這個要求本身就是授權**：AI 直接把 `historicalPersecution`、`politicalRepression`、`discriminationDepiction`、`hateSpeechDepiction` 改成 `true`、執行 `pt content sync`，並告知會變成 REVIEW_REQUIRED；**不需要**再提出 `GAMEPLAY_POLICY_CHANGE_REQUEST`。

描寫仇恨 ≠ 鼓吹仇恨：這四個描寫欄位是 REVIEW_REQUIRED（需要平台審核，不是錯誤），不會變成 ERROR_HARD_RULE。`hateAdvocacy`、`encourageProtectedClassViolence` 維持 `false`。呈現強度依內容邊界（`strong`）。

實際製作這類敘事解謎遊戲時建議用 Godot（`-client godot`，見 README「依遊戲類型開始」）；這個範例用網頁版建立，讓驗證不需要安裝 Godot 與匯出。政策檢查的結果和畫面引擎無關。

必要的檢查結果：

- REVIEW_REQUIRED：`sensitiveContent.historicalPersecution：true`
- REVIEW_REQUIRED：`sensitiveContent.politicalRepression：true`
- REVIEW_REQUIRED：`sensitiveContent.discriminationDepiction：true`
- REVIEW_REQUIRED：`sensitiveContent.hateSpeechDepiction：true`

完整的比對條件在 `expected.json`（`examples/dogfood/verify.sh` 會檢查）。
