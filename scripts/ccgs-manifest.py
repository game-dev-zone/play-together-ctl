# 產生 pt 內嵌的 Claude Code Game Studios 檔案清單（cmd/pt/assets/ai/ccgs.json）。
# pt new -ai studios 下載這個 commit 的原始碼後，逐一比對清單裡的 SHA-256，不符就停止，不安裝清單外的檔案。
# 升級版本：先閱讀新版的 hooks 與 scripts（會在開發者電腦執行），確認沒問題後：
#   git clone https://github.com/donchitos/claude-code-game-studios.git /tmp/ccgs && git -C /tmp/ccgs checkout <tag>
#   python3 -I scripts/ccgs-manifest.py /tmp/ccgs <版本> > cmd/pt/assets/ai/ccgs.json
import hashlib, json, os, subprocess, sys

root, version = sys.argv[1], sys.argv[2]
commit = subprocess.check_output(["git", "-C", root, "rev-parse", "HEAD"], text=True).strip()
# 不裝進遊戲專案的：範本自己的 CI、範本本身的測試、說明與貢獻文件
skip_dirs = {".git", ".github", "studio-tests"}
skip_files = {"README.md", "CONTRIBUTING.md", "SECURITY.md", "CHANGELOG.md", ".gitattributes"}
files = []
for dirpath, dirnames, filenames in os.walk(root):
    rel_dir = os.path.relpath(dirpath, root)
    dirnames[:] = sorted(d for d in dirnames if not (rel_dir == "." and d in skip_dirs))
    for f in sorted(filenames):
        rel = os.path.normpath(os.path.join(rel_dir, f)).replace(os.sep, "/")
        if rel in skip_files:
            continue
        full = os.path.join(dirpath, f)
        if os.path.islink(full):
            sys.exit(f"不支援符號連結：{rel}")
        data = open(full, "rb").read()
        files.append({"path": rel, "sha256": hashlib.sha256(data).hexdigest(), "size": len(data)})
json.dump({"name": "Claude Code Game Studios", "repo": "donchitos/claude-code-game-studios", "version": version, "commit": commit,
           "license": "MIT", "files": files}, sys.stdout, ensure_ascii=False, indent=0)
print()
