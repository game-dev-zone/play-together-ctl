package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// 送審：送審授權失效時改用 -register 的授權；其他版本審核中時，-replace 指定取代它（不指定時不撤回）。
func TestSubmitFallbackAndReplace(t *testing.T) {
	t.Setenv("PT_CREDENTIALS_FILE", filepath.Join(t.TempDir(), "credentials.json"))
	var got []map[string]any
	var auths []string
	platform := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var in map[string]any
		_ = json.Unmarshal(body, &in)
		auths = append(auths, r.Header.Get("Authorization"))
		switch {
		case r.Header.Get("Authorization") == "Bearer ptd_old":
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"login_required"}`))
		case r.URL.Path == "/v1/dev/submissions" && in["replaceBuildId"] == nil:
			got = append(got, in)
			w.WriteHeader(http.StatusConflict)
			_, _ = w.Write([]byte(`{"error":"build_in_review"}`))
		case r.URL.Path == "/v1/dev/submissions":
			got = append(got, in)
			_, _ = w.Write([]byte(`{"buildStatus":"in_review","listingStatus":"in_review"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer platform.Close()
	future := time.Now().Add(time.Hour)
	must(t, saveCredentials(map[string]credential{
		platform.URL:              {Token: "ptd_old", ExpiresAt: future},
		registerKey(platform.URL): {Token: "ptd_reg", ExpiresAt: future},
	}))
	inReview := func() (string, string, error) { return "b-old", "0.1.0", nil }

	// 沒有 -replace（測試不是互動終端機）：不撤回，告訴開發者怎麼做
	_, err := submitWithCredential(platform.URL, "b-new", "rev-1", false, inReview)
	if err == nil || !strings.Contains(err.Error(), "pt submit -replace") {
		t.Fatalf("沒有指定取代時要停下並說明：%v", err)
	}
	if len(got) != 1 || got[0]["replaceBuildId"] != nil {
		t.Fatalf("不能自己取代：%v", got)
	}
	if auths[0] != "Bearer ptd_old" || auths[1] != "Bearer ptd_reg" {
		t.Fatalf("送審授權失效要改用 -register 的授權：%v", auths)
	}

	got = nil
	logged, err := submitWithCredential(platform.URL, "b-new", "rev-1", true, inReview)
	if err != nil || !logged {
		t.Fatalf("-replace：%v %v", logged, err)
	}
	if len(got) != 2 || got[1]["replaceBuildId"] != "b-old" || got[1]["buildId"] != "b-new" {
		t.Fatalf("取代要指定目前審核中的版本：%v", got)
	}
}
