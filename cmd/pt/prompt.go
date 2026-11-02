package main

// pt new 的互動式詢問：在終端機執行且沒有用參數指定時，逐步讓開發者選擇。
// 不是終端機（CI、腳本）時不詢問，沒指定的項目用預設值。測試時換掉 interactive 與 promptIn。

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

var (
	interactive = func() bool {
		st, err := os.Stdin.Stat()
		return err == nil && st.Mode()&os.ModeCharDevice != 0
	}
	promptIn = bufio.NewReader(os.Stdin)
)

var errCanceled = errors.New("已取消")

type option struct{ value, label, detail string }

// readLine 讀一行；輸入結束（Ctrl-D）視為取消
func readLine() (string, error) {
	line, err := promptIn.ReadString('\n')
	if err != nil && (line == "" || !errors.Is(err, io.EOF)) {
		return "", errCanceled
	}
	return strings.TrimSpace(line), nil
}

// askText 問一行文字；check 回傳錯誤時顯示原因並重問
func askText(question string, check func(string) error) (string, error) {
	for {
		fmt.Print(question + "：")
		v, err := readLine()
		if err != nil {
			return "", err
		}
		if err := check(v); err != nil {
			fmt.Println("  " + err.Error())
			continue
		}
		return v, nil
	}
}

// choose 列出選項讓開發者輸入編號；def 是直接按 Enter 的值，空字串表示一定要選。
func choose(title string, opts []option, def string) (string, error) {
	fmt.Println(title)
	defNum := ""
	for i, o := range opts {
		mark := ""
		if o.value == def {
			defNum, mark = strconv.Itoa(i+1), "（預設）"
		}
		fmt.Printf("  %d) %s%s\n", i+1, o.label, mark)
		if o.detail != "" {
			fmt.Printf("     %s\n", o.detail)
		}
	}
	prompt := fmt.Sprintf("請輸入 1–%d", len(opts))
	if defNum != "" {
		prompt += "，直接按 Enter 選 " + defNum
	}
	for {
		fmt.Print(prompt + "：")
		v, err := readLine()
		if err != nil {
			return "", err
		}
		if v == "" && def != "" {
			fmt.Println()
			return def, nil
		}
		if n, err := strconv.Atoi(v); err == nil && n >= 1 && n <= len(opts) {
			fmt.Println()
			return opts[n-1].value, nil
		}
		for _, o := range opts { // 也接受直接輸入值，例如 godot
			if strings.EqualFold(v, o.value) {
				fmt.Println()
				return o.value, nil
			}
		}
	}
}

// confirm 問是或否，直接按 Enter 是 def
func confirm(question string, def bool) (bool, error) {
	hint := "[y/N]"
	if def {
		hint = "[Y/n]"
	}
	for {
		fmt.Print(question + " " + hint + " ")
		v, err := readLine()
		if err != nil {
			return false, err
		}
		switch strings.ToLower(v) {
		case "":
			return def, nil
		case "y", "yes", "是", "好":
			return true, nil
		case "n", "no", "否", "不要":
			return false, nil
		}
	}
}
