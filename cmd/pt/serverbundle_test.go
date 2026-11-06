package main

import (
	"archive/zip"
	"bytes"
	"debug/elf"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestBuildServerBundle(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("沒有 go 指令")
	}
	root := t.TempDir()
	write := func(rel, data string) {
		t.Helper()
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(data), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("go.mod", "module tiny\n\ngo 1.24\n")
	write("main.go", "package main\n\nfunc main() {}\n")
	write("words/a.txt", "大風吹")
	write("words/.DS_Store", "x")
	t.Setenv("GOWORK", "off")
	t.Setenv("GOFLAGS", "")
	p := &project{dir: root, GameServer: &gameServerConfig{Mode: "external", Hosting: "playtogether", Data: "words"}}

	var problems []finding
	checkServerHosting(p, func(level, format string, a ...any) { problems = append(problems, finding{level, format}) })
	if len(problems) != 0 {
		t.Fatalf("設定正確不該有問題：%v", problems)
	}
	data, err := buildServerBundle(p)
	if err != nil {
		t.Fatal(err)
	}
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, f := range zr.File {
		names = append(names, f.Name)
		if f.Name == "server" {
			if f.Mode().Perm() != 0o755 {
				t.Fatalf("server 要可執行：%v", f.Mode())
			}
			rc, _ := f.Open()
			bin, _ := io.ReadAll(rc)
			rc.Close()
			e, err := elf.NewFile(bytes.NewReader(bin))
			if err != nil || e.Machine != elf.EM_X86_64 || e.Class != elf.ELFCLASS64 {
				t.Fatalf("server 要是 linux/amd64 ELF：%v", err)
			}
			for _, prog := range e.Progs {
				if prog.Type == elf.PT_INTERP {
					t.Fatal("server 要靜態連結")
				}
			}
		}
	}
	if want := []string{"manifest.json", "server", "data/a.txt"}; len(names) != 3 || names[0] != want[0] || names[1] != want[1] || names[2] != want[2] {
		t.Fatalf("套件內容：%v", names)
	}

	for _, bad := range []*gameServerConfig{
		{Mode: "external", Hosting: "aws"},
		{Mode: "none", Hosting: "playtogether"},
		{Mode: "external", Hosting: "playtogether", Dir: "missing"},
		{Mode: "external", Hosting: "playtogether", Data: "nope"},
	} {
		var got []finding
		checkServerHosting(&project{dir: root, GameServer: bad}, func(level, format string, a ...any) { got = append(got, finding{level, format}) })
		if len(got) == 0 {
			t.Errorf("%+v 應有問題", bad)
		}
	}
}
