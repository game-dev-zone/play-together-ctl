package main

// pt upgrade：把專案的範本檔案（平台規則、AI 契約、Makefile、pt 的 skill）更新成這個 pt 的版本。
// 先列出完整計畫再一次套用：沒改過的換新、改過的寫成 <檔名>.pt-new 讓你對照合併、新的範本檔補上；
// 新範本拿掉的檔案只列出（-delete-removed 才刪，而且只刪沒改過的）。遊戲本身的檔案（畫面、伺服器、上架資料、
// playtogether.json）一律不碰；SDK 用 pt sdk update。沒有 .pt/generated.json 的舊專案先用 pt upgrade -init。

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const upgradeUsage = "用法：pt upgrade [-dry-run] [-y] [-delete-removed]，或 pt upgrade -init（舊專案第一次使用）"

// 計畫的動作
const (
	actUpdate   = "update"   // 沒改過，換成新範本
	actCreate   = "create"   // 新的範本檔
	actConflict = "conflict" // 改過了：寫 .pt-new
	actRemoved  = "removed"  // 新範本拿掉了（-delete-removed 且沒改過才刪）
	actDeleted  = "deleted"  // 你刪掉的範本檔：不會補回
	actSame     = "same"     // 內容已經和新範本一樣（只更新基線）
)

type upgradeItem struct {
	Path, Action string
	content      []byte // 新範本的內容（update、create、conflict、same）
	modified     bool   // removed：檔案和基線不同
}

// inputsFromProject：用 playtogether.json 目前的設定重新產生範本（宣告改過就用改過的）。
func inputsFromProject(p *project) (genInputs, error) {
	var problems []string
	pl, legacy, ok := readPolicy(p, func(level, format string, a ...any) {
		if isErrorLevel(level) {
			problems = append(problems, fmt.Sprintf(format, a...))
		}
	})
	if !ok || legacy || len(problems) > 0 {
		msg := "playtogether.json 的內容與玩法宣告不完整或是舊格式，無法產生範本：先執行 pt check 修正"
		if len(problems) > 0 {
			msg += "（" + problems[0] + "）"
		}
		return genInputs{}, errors.New(msg)
	}
	ccgs, err := loadCCGS()
	if err != nil {
		return genInputs{}, err
	}
	gds, err := loadSkills()
	if err != nil {
		return genInputs{}, err
	}
	in := genInputs{Name: p.Name, Client: p.Client, Server: p.Server, Platform: p.Platform, Orientation: p.Orientation,
		AIMode: p.AIMode, Skills: p.Skills, Studios: p.AI == "studios", Policy: pl, CCGS: ccgs, GDS: gds, GameServer: "playtogether-managed"}
	if in.Orientation == "" {
		in.Orientation = "portrait"
	}
	if in.AIMode == "" {
		in.AIMode = "lightweight"
	}
	if p.GameServer != nil {
		in.GameServer, in.Hosting = p.GameServer.Mode, p.GameServer.Hosting
	}
	if in.Studios != (p.godotDir() == ".") && p.Client == "godot" {
		return genInputs{}, errors.New("playtogether.json 的 ai 與 godot 資料夾設定不一致，無法判斷範本位置")
	}
	return in, nil
}

// planUpgrade 比較專案、基線與新範本，回傳每個範本檔案要做的事（依路徑排序）。
func planUpgrade(p *project, m *genManifest, next map[string][]byte) ([]upgradeItem, error) {
	var items []upgradeItem
	for rel, content := range next {
		disk, exists, err := fileState(p, rel)
		if err != nil {
			return nil, err
		}
		base, tracked := m.get(rel)
		switch {
		case !exists && tracked:
			items = append(items, upgradeItem{Path: rel, Action: actDeleted})
		case !exists:
			items = append(items, upgradeItem{Path: rel, Action: actCreate, content: content})
		case bytes.Equal(disk, content):
			items = append(items, upgradeItem{Path: rel, Action: actSame, content: content})
		case tracked && base.BaselineSHA256 != "" && sha(disk) == base.BaselineSHA256:
			items = append(items, upgradeItem{Path: rel, Action: actUpdate, content: content})
		default:
			items = append(items, upgradeItem{Path: rel, Action: actConflict, content: content})
		}
	}
	for _, f := range m.Files {
		if f.Kind != kindTemplate {
			continue
		}
		if _, ok := next[f.Path]; ok {
			continue
		}
		disk, exists, err := fileState(p, f.Path)
		if err != nil {
			return nil, err
		}
		if !exists {
			m.remove(f.Path) // 範本和專案都沒有了
			continue
		}
		items = append(items, upgradeItem{Path: f.Path, Action: actRemoved, modified: f.BaselineSHA256 == "" || sha(disk) != f.BaselineSHA256})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Path < items[j].Path })
	return items, nil
}

func printPlan(items []upgradeItem, deleteRemoved bool) (changes int) {
	labels := []struct{ act, title string }{
		{actUpdate, "換成新版（你沒有改過）"},
		{actCreate, "新增"},
		{actConflict, "你改過了，新版寫成 <檔名>.pt-new，請自己對照合併（AI 助手可以幫忙）"},
		{actRemoved, "新版範本已經沒有這個檔案"},
		{actDeleted, "你刪掉了，不會補回"},
	}
	for _, l := range labels {
		var list []string
		for _, it := range items {
			if it.Action != l.act {
				continue
			}
			note := ""
			if it.Action == actRemoved {
				switch {
				case it.modified:
					note = "（你改過，保留）"
				case deleteRemoved:
					note = "（會刪除）"
				default:
					note = "（保留；加 -delete-removed 刪除）"
				}
			}
			list = append(list, "    "+it.Path+note)
		}
		if len(list) == 0 {
			continue
		}
		fmt.Printf("%s：\n%s\n", l.title, strings.Join(list, "\n"))
		if l.act != actDeleted {
			changes += len(list)
		}
	}
	return changes
}

func cmdUpgrade(args []string) error {
	fs := flag.NewFlagSet("upgrade", flag.ExitOnError)
	dry := fs.Bool("dry-run", false, "只列出計畫，不寫任何檔案")
	yes := fs.Bool("y", false, "不詢問，直接套用")
	init := fs.Bool("init", false, "舊專案（沒有 .pt/generated.json）第一次使用：記下目前的檔案，和這個 pt 的範本一樣的才當作沒改過")
	deleteRemoved := fs.Bool("delete-removed", false, "刪除新範本已經沒有、而且你沒改過的檔案")
	_ = fs.Parse(args)
	if fs.NArg() > 0 {
		return errors.New(upgradeUsage)
	}
	if err := requireAssets(); err != nil {
		return err
	}
	p, err := findProject()
	if err != nil {
		return err
	}
	if p == nil {
		return errors.New("找不到 playtogether.json：請在用 pt new 建立的專案裡執行")
	}
	if p.existing() {
		return errors.New("這個專案是用 pt init 接入的既有專案，範本檔案由你自己管理：pt init 重跑可以補上缺少的檔案")
	}
	in, err := inputsFromProject(p)
	if err != nil {
		return err
	}
	files, kinds, err := renderProject(in)
	if err != nil {
		return err
	}
	next := map[string][]byte{}
	for rel, k := range kinds {
		if k == kindTemplate {
			next[rel] = files[rel]
		}
	}
	m, err := loadManifest(p.dir)
	if err != nil {
		return err
	}
	if *init {
		return upgradeInit(p, m, next)
	}
	if m == nil {
		return errors.New("這個專案沒有 " + manifestPath + "（pt v0.24 之前建立）：先執行 pt upgrade -init，記下目前的檔案")
	}
	items, err := planUpgrade(p, m, next)
	if err != nil {
		return err
	}
	fmt.Printf("範本 %s → %s\n\n", m.TemplateVersion, version)
	changes := printPlan(items, *deleteRemoved)
	if c, ok := compareSDK(p.SDKVersion, sdkVersion()); ok && c < 0 {
		fmt.Printf("\nSDK 也有新版（%s → %s）：另外執行 pt sdk update\n", p.SDKVersion, sdkVersion())
	}
	if changes == 0 {
		fmt.Println("範本已經是最新版，不用更新。")
		if !*dry {
			return upgradeSave(p, m, items, false)
		}
		return nil
	}
	if *dry {
		fmt.Println("\n（-dry-run：沒有寫入任何檔案）")
		return nil
	}
	if !*yes {
		if !interactive() || ciEnv() {
			return errors.New("確認上面的計畫後，加 -y 套用")
		}
		ok, err := confirm("套用？", true)
		if err != nil {
			return err
		}
		if !ok {
			fmt.Println("沒有改任何檔案。")
			return nil
		}
	}
	if err := upgradeSave(p, m, items, *deleteRemoved); err != nil {
		return err
	}
	fmt.Println("\n已更新。用 git diff 檢查變更；有 .pt-new 的檔案合併後刪掉 .pt-new，下次 pt upgrade 會把合併後的內容當作沒改過。")
	return nil
}

// upgradeSave 寫入檔案並更新基線。
func upgradeSave(p *project, m *genManifest, items []upgradeItem, deleteRemoved bool) error {
	write := func(rel string, content []byte) error {
		dst := p.path(filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		return os.WriteFile(dst, content, 0o644)
	}
	for _, it := range items {
		switch it.Action {
		case actUpdate, actCreate, actSame:
			if it.Action != actSame {
				if err := write(it.Path, it.content); err != nil {
					return err
				}
			}
			m.set(it.Path, kindTemplate, sha(it.content))
		case actConflict:
			if err := write(it.Path+".pt-new", it.content); err != nil {
				return err
			}
		case actRemoved:
			if deleteRemoved && !it.modified {
				if err := os.Remove(p.path(filepath.FromSlash(it.Path))); err != nil {
					return err
				}
				m.remove(it.Path)
			}
		}
	}
	m.TemplateVersion = version
	return m.save(p.dir)
}

// upgradeInit：舊專案建立基線。內容和這個 pt 的範本一樣的才記成「沒改過」；不一樣的基線留空，之後一律當作改過（不會被覆蓋）。
func upgradeInit(p *project, m *genManifest, next map[string][]byte) error {
	if m != nil {
		return errors.New(manifestPath + " 已經存在：直接執行 pt upgrade")
	}
	m = &genManifest{ManifestVersion: manifestVersion, TemplateVersion: version, SDKVersion: p.SDKVersion}
	var same, differ []string
	for rel, content := range next {
		disk, exists, err := fileState(p, rel)
		if err != nil {
			return err
		}
		if !exists {
			continue // 之後 pt upgrade 會新增
		}
		if bytes.Equal(disk, content) {
			m.set(rel, kindTemplate, sha(content))
			same = append(same, rel)
		} else {
			m.set(rel, kindTemplate, "")
			differ = append(differ, rel)
		}
	}
	// SDK：和這個 pt 內嵌的一樣（版本相同而且沒改過）才有基線
	if sdk, err := sdkFiles(p.Client, p.Server, p.godotDir()); err == nil {
		for rel, content := range sdk {
			if disk, exists, err := fileState(p, rel); err == nil && exists {
				m.set(rel, kindSDK, map[bool]string{true: sha(content), false: ""}[bytes.Equal(disk, content)])
			}
		}
	}
	if err := m.save(p.dir); err != nil {
		return err
	}
	sort.Strings(differ)
	fmt.Printf("已建立 %s：%d 個範本檔案和這個 pt 的範本一樣。\n", manifestPath, len(same))
	if len(differ) > 0 {
		fmt.Printf("這些檔案和這個 pt 的範本不同（可能是你改過，或是舊版範本），之後 pt upgrade 不會覆蓋，新版會寫成 .pt-new：\n    %s\n", strings.Join(differ, "\n    "))
	}
	fmt.Println("接著執行 pt upgrade -dry-run 看會更新什麼。把 .pt/generated.json 加進版本控制。")
	return nil
}
