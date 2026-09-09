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

// 0.0.20：HTTP 5xx/429 携带结构化状态码，retryableUploadURLFailure 放行重试。
func TestRawJSONStatusFeedsRetryClassification(t *testing.T) {
	cases := []struct {
		name       string
		status     int
		body       string
		wantCode   domain.ErrorCode
		wantRetry  bool
	}{
		{"511 网关超时可重试", 511, `{"code":"S3ClientException","msg":"Read timed out"}`, domain.CodeDriverError, true},
		{"502 可重试", 502, `{}`, domain.CodeDriverError, true},
		{"429 限流可重试", 429, `{}`, domain.CodeRateLimited, true},
		{"400 参数错误不重试", 400, `{"res_code":"InvalidParam"}`, domain.CodeDriverError, false},
		{"403 权限不重试", 403, `{}`, domain.CodePermissionDenied, false},
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
			err := d.rawJSON(context.Background(), http.MethodGet, srv.URL+"/op", nil, nil, nil, &map[string]any{})
			if got := retryableUploadURLFailure(context.Background(), err); got != tc.wantRetry {
				t.Fatalf("status=%d retryable=%v want=%v (err=%v)", tc.status, got, tc.wantRetry, err)
			}
			ae, _ := domain.AsAppError(err)
			if ae.Code != tc.wantCode {
				t.Fatalf("status=%d code=%s want=%s", tc.status, ae.Code, tc.wantCode)
			}
		})
	}
}

// 会话失效永远不可重试（避免换 Token 循环）。
func TestSessionExpiredNeverRetryable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		_, _ = w.Write([]byte(`{"res_code":"UserInvalidOpenToken","res_message":"session expired"}`))
	}))
	defer srv.Close()
	d := New().(*Driver)
	d.client = srv.Client()
	err := d.rawJSON(context.Background(), http.MethodGet, srv.URL+"/op", nil, nil, nil, &map[string]any{})
	if retryableUploadURLFailure(context.Background(), err) {
		t.Fatalf("auth-expired error must not be retryable: %v", err)
	}
}

// 0.0.22：HTTP 200 业务错 code/res_code = "-1"（服务暂时不可用）可重试；
// 其它业务码不可重试。
func TestBusinessCodeMinusOneRetryable(t *testing.T) {
	cases := []struct {
		name      string
		body      string
		wantRetry bool
	}{
		{"code=-1 服务暂时不可用", `{"code":"-1","message":"非常抱歉服务暂时不可用，如有问题请联系系统管理员"}`, true},
		{"res_code=-1", `{"res_code":"-1","res_message":"非常抱歉服务暂时不可用"}`, true},
		{"code=-2 其它业务错不可重试", `{"code":"-2","message":"其他错误"}`, false},
		{"res_code=FileNotFound", `{"res_code":"FileNotFound","res_message":"file not found"}`, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(tc.body))
			}))
			defer srv.Close()
			d := New().(*Driver)
			d.client = srv.Client()
			err := d.rawJSON(context.Background(), http.MethodGet, srv.URL+"/op", nil, nil, nil, &map[string]any{})
			if got := retryableUploadURLFailure(context.Background(), err); got != tc.wantRetry {
				t.Fatalf("retryable=%v want=%v (err=%v)", got, tc.wantRetry, err)
			}
		})
	}
}
