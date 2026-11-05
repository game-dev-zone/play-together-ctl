#!/usr/bin/env bash
# 驗證 dogfood 範例：重新生成所有情境，執行 pt check -local -json，比對結果與必要的 findings，
# 並確認契約文件和 playtogether.json 一致（再執行一次 pt content sync 不能有任何變更）。
# REVIEW_REQUIRED 是預期結果；任何非預期的 ERROR_SCHEMA、ERROR_HARD_RULE、ERROR_BUILD 都讓這個腳本失敗。
#
# 用法：examples/dogfood/verify.sh [pt 執行檔]
set -euo pipefail
here="$(cd "$(dirname "$0")" && pwd)"
out="$(mktemp -d)"
trap 'rm -rf "$out"' EXIT
"$here/generate.sh" "$out" "${1:-}" >/dev/null
pt="${1:-$out/.bin/pt}"
export PT_NO_UPDATE_CHECK=1 PT_CREDENTIALS_FILE="$out/.credentials.json"

fail=0
for dir in "$here"/scenarios/*/; do
  id="$(basename "$dir")"
  project="$out/$id/df-${id#*-}"
  # pt check 在有錯誤時以非 0 結束；結果仍然要比對（錯誤是否符合預期由 expected.json 決定）
  (cd "$project" && "$pt" check -local -json >"$out/$id.check.json") || true
  sync="$(cd "$project" && "$pt" content sync)"
  if grep -q "已更新\|policyVersion" <<<"$sync"; then
    echo "✗ ${id}：契約文件和 playtogether.json 不一致（pt content sync 又改了檔案）"
    fail=1
  fi
  if ! python3 -I - "$dir/expected.json" "$out/$id.check.json" "$project" "$id" <<'PY'
import json, sys, os
expected_path, actual_path, project, sid = sys.argv[1:5]
with open(expected_path, encoding="utf-8") as f:
    exp = json.load(f)
with open(actual_path, encoding="utf-8") as f:
    act = json.load(f)
with open(os.path.join(project, "playtogether.json"), encoding="utf-8") as f:
    decl = json.load(f)
problems = []
findings = act.get("findings", [])
if act.get("result") != exp["result"]:
    problems.append(f"結果是 {act.get('result')}，預期 {exp['result']}")
# 只允許預期的等級出現（INFO 是說明，一律允許）
allowed = set(exp.get("allowedSeverities", [])) | {"INFO"}
for fd in findings:
    if fd["severity"] not in allowed:
        problems.append(f"非預期的 {fd['severity']}：{fd['message']}")
for req in exp.get("requiredFindings", []):
    if not any(fd["severity"] == req["severity"] and req["contains"] in fd["message"] for fd in findings):
        problems.append(f"少了 {req['severity']}（{req['contains']}）")
for bad in exp.get("forbiddenFindings", []):
    for fd in findings:
        if fd["severity"] == bad["severity"] and bad["contains"] in fd["message"]:
            problems.append(f"不該有 {bad['severity']}：{fd['message']}")
if "reviewCount" in exp:
    n = sum(1 for fd in findings if fd["severity"] == "REVIEW_REQUIRED")
    if n != exp["reviewCount"]:
        problems.append(f"REVIEW_REQUIRED 有 {n} 項，預期 {exp['reviewCount']} 項")
# 關鍵宣告
for path, want in exp.get("declaration", {}).items():
    cur = decl
    for part in path.split("."):
        cur = cur.get(part) if isinstance(cur, dict) else None
    if cur != want:
        problems.append(f"playtogether.json 的 {path} 是 {cur!r}，預期 {want!r}")
# 契約文件內容
for rel, texts in exp.get("docContains", {}).items():
    with open(os.path.join(project, rel), encoding="utf-8") as f:
        doc = f.read()
    for t in texts:
        if t not in doc:
            problems.append(f"{rel} 少了「{t}」")
for p in problems:
    print(f"✗ {sid}：{p}")
if not problems:
    print(f"✓ {sid}：{act.get('result')}（{sum(1 for fd in findings if fd['severity'] == 'REVIEW_REQUIRED')} 項需要平台審核）")
sys.exit(1 if problems else 0)
PY
  then
    fail=1
  fi
done
exit "$fail"
