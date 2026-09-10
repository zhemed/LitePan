package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// 0.0.29：空/空白 file_id 必须被显式拒绝（此前会"删除成功 1 个项目"却什么都没删）。
func TestDeleteFilesRejectsBlankFileIDs(t *testing.T) {
	cases := []struct {
		name string
		body string
		want int
	}{
		{"全为空字符串", `{"account_id":1,"file_ids":[""],"parent_id":""}`, http.StatusBadRequest},
		{"全为空白", `{"account_id":1,"file_ids":["   ","\t"],"parent_id":""}`, http.StatusBadRequest},
		{"空数组", `{"account_id":1,"file_ids":[],"parent_id":""}`, http.StatusBadRequest},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := &Handler{}
			req := httptest.NewRequest(http.MethodDelete, "/api/files/delete", strings.NewReader(tc.body))
			rec := httptest.NewRecorder()
			h.deleteFiles(rec, req)
			if rec.Code != tc.want {
				t.Fatalf("status=%d want=%d body=%s", rec.Code, tc.want, rec.Body.String())
			}
			var payload map[string]any
			_ = json.Unmarshal(rec.Body.Bytes(), &payload)
			if payload["success"] == true {
				t.Fatalf("空白 file_id 不应返回成功：%s", rec.Body.String())
			}
		})
	}
}
