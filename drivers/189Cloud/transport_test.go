package cloud189

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"litepan/internal/domain"
)

// TestRawJSONClassifies400AuthPayloadAsAuthExpired 回归测试：
// 0.0.13 曾将 HTTP 400 + UserInvalidOpenToken/unifyAccountInfo is null 判为 DRIVER_ERROR，
// 导致 WithRetry 被动刷新、Init.isSessionExpired、认证调度恢复全部失效。
// 修复后 400 + 失效 payload 必须返回 CodeAuthExpired（与 0.0.12 语义一致）。
func TestRawJSONClassifies400AuthPayloadAsAuthExpired(t *testing.T) {
	cases := []struct {
		name   string
		status int
		body   string
		want   domain.ErrorCode
	}{
		{"400 无效open token", 400, `{"res_code":"UserInvalidOpenToken","res_message":"error in login get unifyAccountInfo is null"}`, domain.CodeAuthExpired},
		{"400 会话失效", 400, `{"res_code":"InvalidSessionKey","res_message":"session expired"}`, domain.CodeAuthExpired},
		{"400 普通参数错误不误判", 400, `{"res_code":"InvalidParam","res_message":"bad parameter"}`, domain.CodeDriverError},
		{"401 仍为认证失效", 401, `{}`, domain.CodeAuthExpired},
		{"200+失效payload 仍为认证失效", 200, `{"res_code":"UserInvalidOpenToken"}`, domain.CodeAuthExpired},
		{"403 权限", 403, `{}`, domain.CodePermissionDenied},
		{"429 限流", 429, `{}`, domain.CodeRateLimited},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer srv.Close()

			d := New().(*Driver)
			d.client = srv.Client()
			err := d.rawJSON(context.Background(), http.MethodGet, srv.URL+"/list", nil, nil, nil, &map[string]any{})
			ae, ok := domain.AsAppError(err)
			if !ok {
				t.Fatalf("expected AppError, got %T: %v", err, err)
			}
			if ae.Code != tc.want {
				t.Fatalf("status=%d code=%s want=%s (msg=%q)", tc.status, ae.Code, tc.want, ae.Message)
			}
		})
	}
}
