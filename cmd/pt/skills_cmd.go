package main

// pt skills install：把 pt 的 skill 裝到使用者層級（Claude Code：~/.claude/skills；Codex：~/.agents/skills）：
// playtogether-new（在空資料夾開一款新遊戲）、playtogether-publish（上架到送審，包含還沒 pt init 的專案）。
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

var globalSkills = []string{"playtogether-new", "playtogether-publish"}

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
		fmt.Println("玩聚的 skill 已經是這個 pt 的版本，沒有改任何檔案")
	} else {
		fmt.Println("已安裝玩聚的 skill（" + strings.Join(globalSkills, "、") + "；pt 更新後重跑這個指令會更新它）")
	}
	fmt.Println("\n接下來在 Claude Code（或 Codex）說：")
	fmt.Println("  空資料夾：「幫我做一款玩聚遊戲：<你想做的玩法>」→ 照 playtogether-new 建立專案（多人遊戲建議交給玩聚代管伺服器）")
	fmt.Println("  已經有遊戲：「幫我上架到玩聚」→ 照 playtogether-publish 從 pt init 一路做到送審")
	fmt.Println("Claude Code 已經開著時，沒看到新的 skill 就重新開一次對話。")
	return nil
}
