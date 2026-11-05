#!/usr/bin/env bash
# 重新生成 dogfood 範例：用目前 repository 建置的 pt，在暫存目錄執行實際的 pt new，
# 套用情境需要的最小宣告修改（declaration.patch.json），再執行 pt content sync。
# 不改動 repository 內的檔案，不需要網路或平台帳號。
#
# 用法：examples/dogfood/generate.sh [輸出資料夾] [pt 執行檔]
#   沒指定輸出資料夾時建立暫存資料夾；沒指定 pt 時用 go build 建置。最後一行印出輸出資料夾。
set -euo pipefail
here="$(cd "$(dirname "$0")" && pwd)"
repo="$(cd "$here/../.." && pwd)"
out="${1:-$(mktemp -d)}"
mkdir -p "$out"
out="$(cd "$out" && pwd)"
pt="${2:-}"
if [ -z "$pt" ]; then
  pt="$out/.bin/pt"
  mkdir -p "$out/.bin"
  (cd "$repo" && go build -o "$pt" ./cmd/pt)
fi
# 不連網、不讀寫開發者自己的送審授權
export PT_NO_UPDATE_CHECK=1 PT_CREDENTIALS_FILE="$out/.credentials.json"

for dir in "$here"/scenarios/*/; do
  id="$(basename "$dir")"
  name="df-${id#*-}"                       # 01-puzzle-general → df-puzzle-general
  read -r -a args <"$dir/new.args"
  rm -rf "${out:?}/$id"
  mkdir -p "$out/$id"
  (cd "$out/$id" && "$pt" new -y "${args[@]}" "$name" >/dev/null)
  project="$out/$id/$name"
  if [ -f "$dir/declaration.patch.json" ]; then
    # 只改情境需要的欄位（深層合併到 playtogether.json），再重新產生契約文件
    python3 -I - "$project/playtogether.json" "$dir/declaration.patch.json" <<'PY'
import json, sys
path, patch = sys.argv[1], sys.argv[2]
def merge(dst, src):
    for k, v in src.items():
        if isinstance(v, dict) and isinstance(dst.get(k), dict):
            merge(dst[k], v)
        else:
            dst[k] = v
with open(path, encoding="utf-8") as f:
    doc = json.load(f)
with open(patch, encoding="utf-8") as f:
    merge(doc, json.load(f))
with open(path, "w", encoding="utf-8") as f:
    json.dump(doc, f, ensure_ascii=False, indent=2)
    f.write("\n")
PY
  fi
  (cd "$project" && "$pt" content sync >/dev/null)
  echo "已生成 ${id}（${project}）" >&2
done
echo "$out"
