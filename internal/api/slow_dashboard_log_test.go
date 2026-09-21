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
		"/api/admin/notifications":              true,
		"/api/admin/notifications/unread-count": true,
		"/api/logs/stats":                       true,
		// 未列入的路径不应被观测
		"/api/admin/accounts/1":      false,
		"/api/admin/automation/runs": false,
		"/api/files/list":            false,
		"/api/health":                false,
	}
	for path, want := range cases {
		if got := isDashboardOverviewPath(path); got != want {
			t.Fatalf("isDashboardOverviewPath(%q) = %v，期望 %v", path, got, want)
		}
	}
}

// TestSlowRequestLogsSuppressesWithinWindow 锁定抑制窗口：窗口内只记一次，窗口结束汇总压制次数。
func TestSlowRequestLogsSuppressesWithinWindow(t *testing.T) {
	var logs slowRequestLogs
	now := time.Now()

	if suppressed, ok := logs.shouldLog("/api/admin/accounts", now); !ok || suppressed != 0 {
		t.Fatalf("首次慢请求应记录：ok=%v suppressed=%d", ok, suppressed)
	}
	// 抑制窗口内的重复慢请求只累计次数
	for _, offset := range []time.Duration{time.Minute, 29 * time.Minute} {
		if _, ok := logs.shouldLog("/api/admin/accounts", now.Add(offset)); ok {
			t.Fatalf("窗口内（+%v）不得重复记录", offset)
		}
	}
	// 窗口结束后立刻记录，并带上被压制的次数
	suppressed, ok := logs.shouldLog("/api/admin/accounts", now.Add(slowDashboardLogInterval))
	if !ok {
		t.Fatal("窗口结束后应记录")
	}
	if suppressed != 2 {
		t.Fatalf("suppressed_count = %d，期望 2", suppressed)
	}
	// 计数已上报，不应重复累计
	if suppressed, ok := logs.shouldLog("/api/admin/accounts", now.Add(2*slowDashboardLogInterval)); !ok || suppressed != 0 {
		t.Fatalf("第二次汇总：ok=%v suppressed=%d，期望 true/0", ok, suppressed)
	}
}

// 各路径的抑制窗口互不影响。
func TestSlowRequestLogsKeepsPathsIndependent(t *testing.T) {
	var logs slowRequestLogs
	now := time.Now()
	if _, ok := logs.shouldLog("/api/admin/accounts", now); !ok {
		t.Fatal("首次应记录")
	}
	if _, ok := logs.shouldLog("/api/logs/stats", now); !ok {
		t.Fatal("另一条路径有自己的窗口，应记录")
	}
}

// 恢复正常时上报压制次数并结束窗口；下一次变慢可以立刻记录。
func TestSlowRequestLogsRecoveredResetsWindow(t *testing.T) {
	var logs slowRequestLogs
	now := time.Now()
	_, _ = logs.shouldLog("/api/admin/accounts", now)
	_, _ = logs.shouldLog("/api/admin/accounts", now.Add(time.Minute))
	_, _ = logs.shouldLog("/api/admin/accounts", now.Add(2*time.Minute))

	if suppressed := logs.recovered("/api/admin/accounts"); suppressed != 2 {
		t.Fatalf("恢复时应上报压制次数 2，实际 %d", suppressed)
	}
	if suppressed := logs.recovered("/api/admin/accounts"); suppressed != 0 {
		t.Fatalf("压制次数只能上报一次，实际重复上报 %d", suppressed)
	}
	if suppressed, ok := logs.shouldLog("/api/admin/accounts", now.Add(3*time.Minute)); !ok || suppressed != 0 {
		t.Fatalf("窗口已结束，变慢应立刻记录：ok=%v suppressed=%d", ok, suppressed)
	}
}

// 没有压制记录的快请求不得冲掉抑制窗口。
func TestSlowRequestLogsRecoveredIgnoresQuietPath(t *testing.T) {
	var logs slowRequestLogs
	now := time.Now()
	if _, ok := logs.shouldLog("/api/admin/accounts", now); !ok {
		t.Fatal("首次应记录")
	}
	if suppressed := logs.recovered("/api/admin/accounts"); suppressed != 0 {
		t.Fatalf("无压制记录时应返回 0，实际 %d", suppressed)
	}
	if _, ok := logs.shouldLog("/api/admin/accounts", now.Add(time.Minute)); ok {
		t.Fatal("快请求不得结束抑制窗口，窗口内仍应抑制")
	}
}

// TestLogSlowDashboardRequests 锁定行为：只有"白名单 GET 且超过阈值"才记一条慢日志，
// 且日志级别为 Debug（诊断信息不刷用户日志）。
func TestLogSlowDashboardRequests(t *testing.T) {
	// 每个用例用独立的 Handler：抑制窗口按路径共享，复用会让后面的用例被前一次压住，
	// 从而把"级别降噪"误判成"被抑制"。抑制语义本身由上面几组 shouldLog/recovered 用例覆盖。
	run := func(level slog.Level, method, path string, sleep time.Duration) string {
		h := &Handler{}
		var buf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: level}))
		handler := h.logSlowDashboardRequests(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			time.Sleep(sleep)
		}))
		req := httptest.NewRequest(method, path, nil)
		req = req.WithContext(context.WithValue(req.Context(), requestLoggerCtxKey{}, logger))
		handler.ServeHTTP(httptest.NewRecorder(), req)
		return buf.String()
	}

	if got := run(slog.LevelDebug, http.MethodGet, "/api/admin/accounts", 0); got != "" {
		t.Fatalf("快速请求不应记录，实际：%s", got)
	}
	if got := run(slog.LevelDebug, http.MethodGet, "/api/files/list", 0); got != "" {
		t.Fatalf("白名单外路径不应记录，实际：%s", got)
	}
	if got := run(slog.LevelDebug, http.MethodPost, "/api/admin/accounts", 0); got != "" {
		t.Fatalf("非 GET 不应记录，实际：%s", got)
	}

	// 超过阈值：手工把阈值内路径的耗时拉长（阈值 1s，用 1.05s 规避时钟抖动）
	got := run(slog.LevelDebug, http.MethodGet, "/api/admin/accounts", slowDashboardRequestThreshold+50*time.Millisecond)
	if !strings.Contains(got, "后台概况接口响应较慢") || !strings.Contains(got, "duration_ms") {
		t.Fatalf("超过阈值应记录慢日志，实际：%q", got)
	}
	if !strings.Contains(got, "suppressed_count=0") {
		t.Fatalf("慢日志应带 suppressed_count，实际：%q", got)
	}
	if !strings.Contains(got, "level=DEBUG") {
		t.Fatalf("慢日志应为 DEBUG 级别，实际：%q", got)
	}

	// 默认 Info 级别下不可见：这是降噪的目的，需回归锁定
	if got := run(slog.LevelInfo, http.MethodGet, "/api/admin/accounts", slowDashboardRequestThreshold+50*time.Millisecond); got != "" {
		t.Fatalf("Info 级别不应输出慢日志，实际：%q", got)
	}
}

// TestDashboardOverviewPathsExcludeRemovedEndpoints 锁定上游 6d392018 路径表里的端点不在本方白名单：
// 本方已删除 cache-retention / media-organize / strm 相关接口，跟随上游时必须按在线端点裁剪。
func TestDashboardOverviewPathsExcludeRemovedEndpoints(t *testing.T) {
	for _, path := range []string{
		"/api/admin/cache-retention/overview",
		"/api/admin/media-organize/tasks",
		"/api/admin/strm/tasks",
	} {
		if isDashboardOverviewPath(path) {
			t.Fatalf("本方已删除的端点不应被观测：%s", path)
		}
	}
}
