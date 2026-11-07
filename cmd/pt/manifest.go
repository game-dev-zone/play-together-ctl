package main

// .pt/generated.json：pt 產生的檔案與當時的內容雜湊（基線）。pt upgrade、pt sdk update 用它判斷檔案有沒有被改過：
// 沒改過的才換新，改過的不覆蓋。這個檔案要放進版本控制（團隊成員的 pt 才看得到同一份基線）。

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

const (
	manifestPath    = ".pt/generated.json"
	manifestVersion = 1
)

type manifestFile struct {
	Path string `json:"path"`
	Kind string `json:"kind"` // sdk、template
	// 產生時的內容 SHA-256；空字串表示不知道（pt upgrade -init 時和這個 pt 的範本不同，視為改過）
	BaselineSHA256 string `json:"baselineSha256"`
	GeneratedBy    string `json:"generatedBy"` // 產生這個基線的 pt 版本
}

type genManifest struct {
	ManifestVersion int            `json:"manifestVersion"`
	TemplateVersion string         `json:"templateVersion"` // 範本與升級規則的版本（目前等於 pt 版本）
	SDKVersion      string         `json:"sdkVersion"`
	Files           []manifestFile `json:"files"`
}

func sha(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

// newManifest 只記 sdk 與 template（starter 建立後就是開發者的，第三方內容不由 pt 管理）。
func newManifest(files map[string][]byte, kinds map[string]string) *genManifest {
	m := &genManifest{ManifestVersion: manifestVersion, TemplateVersion: version, SDKVersion: sdkVersion()}
	for rel, k := range kinds {
		if k == kindSDK || k == kindTemplate {
			m.set(rel, k, sha(files[rel]))
		}
	}
	return m
}

func (m *genManifest) get(rel string) (manifestFile, bool) {
	for _, f := range m.Files {
		if f.Path == rel {
			return f, true
		}
	}
	return manifestFile{}, false
}

func (m *genManifest) set(rel, kind, baseline string) {
	for i := range m.Files {
		if m.Files[i].Path == rel {
			m.Files[i] = manifestFile{Path: rel, Kind: kind, BaselineSHA256: baseline, GeneratedBy: version}
			return
		}
	}
	m.Files = append(m.Files, manifestFile{Path: rel, Kind: kind, BaselineSHA256: baseline, GeneratedBy: version})
}

func (m *genManifest) remove(rel string) {
	out := m.Files[:0]
	for _, f := range m.Files {
		if f.Path != rel {
			out = append(out, f)
		}
	}
	m.Files = out
}

func (m *genManifest) save(dir string) error {
	sort.Slice(m.Files, func(i, j int) bool { return m.Files[i].Path < m.Files[j].Path })
	raw, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	f := filepath.Join(dir, filepath.FromSlash(manifestPath))
	if err := os.MkdirAll(filepath.Dir(f), 0o755); err != nil {
		return err
	}
	return os.WriteFile(f, append(raw, '\n'), 0o644)
}

// loadManifest：沒有這個檔案（v0.24 之前建立的專案）回傳 nil, nil。
func loadManifest(dir string) (*genManifest, error) {
	raw, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(manifestPath)))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var m genManifest
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, errors.New(manifestPath + " 格式錯誤：" + err.Error())
	}
	if m.ManifestVersion > manifestVersion {
		return nil, errors.New(manifestPath + " 是較新的 pt 產生的：請先更新 pt")
	}
	return &m, nil
}

// fileState 讀專案裡的檔案：不存在時 exists 是 false。
func fileState(p *project, rel string) (content []byte, exists bool, err error) {
	content, err = os.ReadFile(p.path(filepath.FromSlash(rel)))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, false, nil
	}
	return content, err == nil, err
}
