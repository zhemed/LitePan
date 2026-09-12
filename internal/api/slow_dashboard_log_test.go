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

// TestIsDashboardOverviewPath 锁定观测白名单（0.0.43；上游路径表已按本方在线端点裁剪）。
func TestIsDashboardOverviewPath(t *testing.T) {
	cases := map[string]bool{
		"/api/admin/accounts":                   true,
		"/api/admin/cache/stats":                true,
		"/api/admin/fuse/mounts":                true,
		"/api/admin/notifications":              true,
		"/api/admin/notifications/unread-count": true,
		"/api/logs/stats":                       true,
		// 未列入的路径不应被观测
		"/api/admin/accounts/1":     false,
		"/api/admin/automation/runs": false,
		"/api/files/list":           false,
		"/api/health":               false,
	}
	for path, want := range cases {
		if got := isDashboardOverviewPath(path); got != want {
			t.Fatalf("isDashboardOverviewPath(%q) = %v，期望 %v", path, got, want)
		}
	}
}

// TestLogSlowDashboardRequests 锁定行为：只有"白名单 GET 且超过阈值"才记录一条 INFO。
func TestLogSlowDashboardRequests(t *testing.T) {
	h := &Handler{}
	run := func(method, path string, sleep time.Duration) string {
		var buf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&buf, nil))
		handler := h.logSlowDashboardRequests(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			time.Sleep(sleep)
		}))
		req := httptest.NewRequest(method, path, nil)
		req = req.WithContext(context.WithValue(req.Context(), requestLoggerCtxKey{}, logger))
		handler.ServeHTTP(httptest.NewRecorder(), req)
		return buf.String()
	}

	if got := run(http.MethodGet, "/api/admin/accounts", 0); got != "" {
		t.Fatalf("快速请求不应记录，实际：%s", got)
	}
	if got := run(http.MethodGet, "/api/files/list", 0); got != "" {
		t.Fatalf("白名单外路径不应记录，实际：%s", got)
	}
	if got := run(http.MethodPost, "/api/admin/accounts", 0); got != "" {
		t.Fatalf("非 GET 不应记录，实际：%s", got)
	}

	// 超过阈值：手工把阈值内路径的耗时拉长（阈值 1s，用 1.05s 规避时钟抖动）
	got := run(http.MethodGet, "/api/admin/accounts", slowDashboardRequestThreshold+50*time.Millisecond)
	if !strings.Contains(got, "后台概况接口响应较慢") || !strings.Contains(got, "duration_ms") {
		t.Fatalf("超过阈值应记录慢日志，实际：%q", got)
	}
}
