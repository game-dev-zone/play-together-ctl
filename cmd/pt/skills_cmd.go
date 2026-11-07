package main

// pt skills install：把 pt 的「上架到玩聚」skill 裝到使用者層級（Claude Code：~/.claude/skills；Codex：~/.agents/skills），
// 在任何遊戲資料夾打開 AI 助手說「幫我上架到玩聚」就會照它做（包含還沒 pt init 的專案）。
// 只裝 globalSkills 列出的 skill；其他 pt 的 skill（例如 playtogether-store）由 pt init／pt new 放進專案。
// 這些檔案由 pt 管理：重跑會更新成這個 pt 內嵌的版本；pt skills uninstall 移除。

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var globalSkills = []string{"playtogether-publish"}

// globalSkillFiles 回傳使用者層級要寫的檔案（家目錄內路徑 → 內容）。
func globalSkillFiles() (map[string][]byte, error) {
	files := map[string][]byte{}
	for _, name := range globalSkills {
		err := fs2Walk("templates/skills/"+name, func(rel string, content []byte) error {
			for _, root := range ptSkillRoots {
				files[filepath.Join(filepath.FromSlash(root), name, filepath.FromSlash(rel))] = content
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	if len(files) == 0 {
		return nil, errors.New("這個 pt 沒有內嵌上架 skill")
	}
	return files, nil
}

func cmdSkills(args []string) error {
	if len(args) != 1 || (args[0] != "install" && args[0] != "uninstall") {
		return errors.New("用法：pt skills install（裝到 ~/.claude/skills 與 ~/.agents/skills，之後在遊戲資料夾請 AI 助手「幫我上架到玩聚」）或 pt skills uninstall")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	if args[0] == "uninstall" {
		for _, root := range ptSkillRoots {
			for _, name := range globalSkills {
				dir := filepath.Join(home, filepath.FromSlash(root), name)
				if _, err := os.Stat(dir); err != nil {
					continue
				}
				if err := os.RemoveAll(dir); err != nil {
					return err
				}
				fmt.Println("  已移除", dir)
			}
		}
		return nil
	}
	files, err := globalSkillFiles()
	if err != nil {
		return err
	}
	changed := 0
	for rel, content := range files {
		dst := filepath.Join(home, rel)
		if old, err := os.ReadFile(dst); err == nil && bytes.Equal(old, content) {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(dst, content, 0o644); err != nil {
			return err
		}
		fmt.Println("  ✓", dst)
		changed++
	}
	if changed == 0 {
		fmt.Println("上架 skill 已經是這個 pt 的版本，沒有改任何檔案")
	} else {
		fmt.Println("已安裝上架 skill（" + strings.Join(globalSkills, "、") + "；pt 更新後重跑這個指令會更新它）")
	}
	fmt.Println("\n在遊戲的資料夾打開 Claude Code（或 Codex），說「幫我上架到玩聚」，它會從 pt init 一路做到送審。")
	return nil
}
