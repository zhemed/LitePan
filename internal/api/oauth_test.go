package api

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// 重试全部失败时必须留下 Warn 日志（上游 b5c9308 之后的可观测性修复）：
// 只有响应体是“OAuth 服务暂时不可用”，运维侧无从判断失败原因。
func TestOAuthForwardLogsWarningWhenAllRetriesFail(t *testing.T) {
	dead := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	deadURL := dead.URL
	dead.Close()

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	h := &Handler{}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/oauth/start", nil)
	req = req.WithContext(context.WithValue(req.Context(), requestLoggerCtxKey{}, logger))

	h.oauthForward(rec, req, http.MethodPost, deadURL+"/api/oauth/start", []byte(`{}`), 0, time.Second)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("状态码 = %d，期望 %d", rec.Code, http.StatusBadGateway)
	}
	got := buf.String()
	if !strings.Contains(got, "OAuth 转发重试均失败") || !strings.Contains(got, "level=WARN") {
		t.Fatalf("全部重试失败应记录 WARN，实际：%q", got)
	}
	if !strings.Contains(got, "attempts=1") {
		t.Fatalf("日志应带重试次数，实际：%q", got)
	}
	if !strings.Contains(got, "err=") {
		t.Fatalf("日志应带最后一次错误，实际：%q", got)
	}
}

// 成功转发不应产生告警日志，也不得改写上游响应。
func TestOAuthForwardPassesThroughUpstreamResponse(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("User-Agent"); got == "" {
			t.Error("转发请求应带 User-Agent")
		}
		w.WriteHeader(http.StatusTeapot)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer upstream.Close()

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	h := &Handler{}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/oauth/status/1", nil)
	req = req.WithContext(context.WithValue(req.Context(), requestLoggerCtxKey{}, logger))

	h.oauthForward(rec, req, http.MethodGet, upstream.URL+"/api/oauth/status/1", nil, 0, time.Second)

	if rec.Code != http.StatusTeapot {
		t.Fatalf("状态码 = %d，期望透传 %d", rec.Code, http.StatusTeapot)
	}
	if body := rec.Body.String(); body != `{"ok":true}` {
		t.Fatalf("响应体 = %q，应原样透传", body)
	}
	if got := buf.String(); strings.Contains(got, "level=WARN") {
		t.Fatalf("成功转发不应有告警日志，实际：%q", got)
	}
}
