# 產生 pt 內嵌的 awesome-gamedev-agent-skills 精選清單（cmd/pt/assets/ai/gdskills.json）。
# pt new -skills 下載這個 commit 的原始碼後，逐一比對清單裡的 SHA-256，不符就停止，不安裝清單外的檔案。
# 只收說明文件（SKILL.md、references/ 的 Markdown、create-game-assets 的範本），不收任何 scripts、agents、CI 或 starter。
# 升級版本：先逐一閱讀新版的精選 skill 是否和 PLAYTOGETHER.md 衝突（templates/ai/PLAYTOGETHER.md.tmpl 的「第三方遊戲開發 skill」衝突清單要同步），確認後：
#   git clone https://github.com/gamedev-skills/awesome-gamedev-agent-skills.git /tmp/gds && git -C /tmp/gds checkout <commit>
#   python3 -I scripts/gdskills-manifest.py /tmp/gds <日期版本，例如 2026-10-28> > cmd/pt/assets/ai/gdskills.json
import hashlib, json, os, subprocess, sys

root, version = sys.argv[1], sys.argv[2]
commit = subprocess.check_output(["git", "-C", root, "rev-parse", "HEAD"], text=True).strip()

# 精選：core 所有專案都裝；godot、web 依 pt new 的 -client 裝
SKILLS = {
    "core": [
        "disciplines/game-feel", "disciplines/game-ui-ux", "disciplines/input-systems", "disciplines/audio-design",
        "disciplines/create-game-assets", "disciplines/performance-optimization", "disciplines/level-design",
        "workflows/prototype-fast",
    ],
    "godot": [
        "godot/godot-gdscript", "godot/godot-nodes-scenes", "godot/godot-signals-groups", "godot/godot-ui-control",
        "godot/godot-audio", "godot/godot-animation", "godot/godot-2d-movement", "godot/godot-gdscript-headless-testing",
    ],
    "web": [
        "web-engines/phaser-core", "web-engines/phaser-arcade-physics", "web-engines/pixijs-rendering",
        "web-engines/threejs-scene-setup", "web-engines/threejs-materials-lighting", "web-engines/threejs-gltf-loading",
    ],
}
# create-game-assets 的美術方向範本與素材清單範本（純文字，不是程式）
EXTRA = {"disciplines/create-game-assets": ["assets/art-direction-brief.md", "assets/asset-manifest.json"]}

def entry(rel):
    full = os.path.join(root, rel)
    if os.path.islink(full):
        sys.exit(f"不支援符號連結：{rel}")
    data = open(full, "rb").read()
    return {"path": rel, "sha256": hashlib.sha256(data).hexdigest(), "size": len(data)}

files, skills = [entry("LICENSE"), entry("NOTICE")], []
for group, dirs in SKILLS.items():
    for d in dirs:
        base = os.path.join("skills", d)
        if not os.path.isfile(os.path.join(root, base, "SKILL.md")):
            sys.exit(f"找不到 {base}/SKILL.md")
        rels = [f"{base}/SKILL.md"]
        ref = os.path.join(root, base, "references")
        if os.path.isdir(ref):
            for dirpath, dirnames, filenames in os.walk(ref):
                dirnames.sort()
                for f in sorted(filenames):
                    if f.endswith(".md"):
                        rels.append(os.path.relpath(os.path.join(dirpath, f), root).replace(os.sep, "/"))
        rels += [f"{base}/{x}" for x in EXTRA.get(d, [])]
        files += [entry(r) for r in rels]
        skills.append({"name": os.path.basename(d), "dir": base, "group": group})
json.dump({"name": "awesome-gamedev-agent-skills", "repo": "gamedev-skills/awesome-gamedev-agent-skills", "version": version,
           "commit": commit, "license": "Apache-2.0", "skills": skills, "files": files}, sys.stdout, ensure_ascii=False, indent=0)
print()
