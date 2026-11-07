package main

import (
	"encoding/json"
	"testing"
)

// pt preview 挑的版本：跳過退件、撤回、檔案還沒檢查通過的；-version 指定時只找那一個。
func TestPreviewBuild(t *testing.T) {
	var st devStatus
	must(t, json.Unmarshal([]byte(`{"builds":[
		{"id":"b5","version":"1.4.0","status":"draft","artifactStatus":"processing"},
		{"id":"b4","version":"1.3.0","status":"rejected","artifactStatus":"ready"},
		{"id":"b3","version":"1.2.0","status":"withdrawn","artifactStatus":"ready"},
		{"id":"b2","version":"1.1.0","status":"in_review","artifactStatus":"ready"},
		{"id":"b1","version":"1.0.0","status":"released","artifactStatus":"ready"}]}`), &st))
	if id, ver := previewBuild(st, ""); id != "b2" || ver != "1.1.0" {
		t.Fatalf("預設是最新一個可以預覽的版本：%s %s", id, ver)
	}
	if id, _ := previewBuild(st, "1.0.0"); id != "b1" {
		t.Fatalf("指定版本：%s", id)
	}
	if id, _ := previewBuild(st, "1.3.0"); id != "" {
		t.Fatalf("退件的版本不能預覽：%s", id)
	}
	if id, _ := previewBuild(st, "9.9.9"); id != "" {
		t.Fatalf("不存在的版本：%s", id)
	}
}
