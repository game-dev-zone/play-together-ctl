package main

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// 玩聚代管的遊戲伺服器套件（server.zip）：manifest.json、server（linux/amd64 靜態執行檔）、選用的 data/。
// 平台會再檢查一次（只接受這三種內容、靜態連結的 ELF64 x86-64）。

const (
	serverBundleManifest = `{"runtime":"go","entrypoint":"server","port":8080,"arch":"linux/amd64"}` + "\n"
	maxServerBundleBytes = 50 << 20
)

// checkServerHosting 檢查 gameServer.hosting 的設定（pt check 與 pt upload 共用）。
func checkServerHosting(p *project, add func(level, format string, a ...any)) {
	if p == nil || p.GameServer == nil || p.GameServer.Hosting == "" {
		return
	}
	gs := p.GameServer
	if gs.Hosting != "playtogether" {
		add(levelHard, `playtogether.json 的 gameServer.hosting 只能是 "playtogether"（或不填）`)
		return
	}
	if p.gameServerMode() == "none" {
		add(levelHard, `gameServer.mode 是 none（沒有遊戲伺服器），不能設定 gameServer.hosting`)
	}
	dir := p.path(serverDir(gs))
	if _, err := os.Stat(filepath.Join(dir, "go.mod")); err != nil {
		add(levelBuild, "代管遊戲伺服器目前只支援 Go：%s 沒有 go.mod（gameServer.dir）", dir)
	}
	if gs.Data != "" {
		if st, err := os.Stat(p.path(gs.Data)); err != nil || !st.IsDir() {
			add(levelBuild, "gameServer.data 不是資料夾：%s", p.path(gs.Data))
		}
	}
}

func serverDir(gs *gameServerConfig) string {
	if gs.Dir == "" {
		return "."
	}
	return gs.Dir
}

// buildServerBundle 交叉編譯遊戲伺服器並打包。檔案順序與時間戳記固定，同樣的輸入得到同樣的 ZIP。
func buildServerBundle(p *project) ([]byte, error) {
	gs := p.GameServer
	dir := p.path(serverDir(gs))
	goBin, err := exec.LookPath("go")
	if err != nil {
		return nil, errors.New("代管遊戲伺服器要用 Go 建置：找不到 go 指令")
	}
	tmp, err := os.MkdirTemp("", "pt-server-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmp)
	out := filepath.Join(tmp, "server")
	fmt.Println("建置遊戲伺服器（linux/amd64、靜態連結）：", dir)
	cmd := exec.Command(goBin, "build", "-trimpath", "-ldflags=-s -w -buildid=", "-o", out, ".")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOOS=linux", "GOARCH=amd64", "CGO_ENABLED=0")
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("建置遊戲伺服器失敗：%w", err)
	}
	bin, err := os.ReadFile(out)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	stamp := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	add := func(name string, mode fs.FileMode, r io.Reader) error {
		h := &zip.FileHeader{Name: name, Method: zip.Deflate, Modified: stamp}
		h.SetMode(mode)
		w, err := zw.CreateHeader(h)
		if err != nil {
			return err
		}
		_, err = io.Copy(w, r)
		return err
	}
	if err := add("manifest.json", 0o644, strings.NewReader(serverBundleManifest)); err != nil {
		return nil, err
	}
	if err := add("server", 0o755, bytes.NewReader(bin)); err != nil {
		return nil, err
	}
	if gs.Data != "" {
		root := p.path(gs.Data)
		var paths []string
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if path != root && strings.HasPrefix(d.Name(), ".") {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if d.Type()&fs.ModeSymlink != 0 {
				return fmt.Errorf("%s 是符號連結，平台不接受；請改成實際檔案", path)
			}
			if d.Type().IsRegular() {
				paths = append(paths, path)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
		sort.Strings(paths)
		for _, path := range paths {
			rel, _ := filepath.Rel(root, path)
			f, err := os.Open(path)
			if err != nil {
				return nil, err
			}
			err = add("data/"+filepath.ToSlash(rel), 0o644, f)
			f.Close()
			if err != nil {
				return nil, err
			}
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	if buf.Len() > maxServerBundleBytes {
		return nil, fmt.Errorf("遊戲伺服器套件 %.1f MB，超過 50 MB 上限", float64(buf.Len())/(1<<20))
	}
	fmt.Printf("遊戲伺服器套件 %.1f MB（執行檔 %.1f MB）\n", float64(buf.Len())/(1<<20), float64(len(bin))/(1<<20))
	return buf.Bytes(), nil
}
