package api

import (
	"bytes"
	"context"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"

	"litepan/internal/domain"
)

// captureAPIErrorLog 用注入到请求上下文的 logger 捕获 logAPIError 的输出。
func captureAPIErrorLog(ctx context.Context) string {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	req := httptest.NewRequest("GET", "/api/admin/accounts", nil)
	req = req.WithContext(context.WithValue(ctx, requestLoggerCtxKey{}, logger))
	logAPIError(req, domain.Errorf(domain.CodeDriverError, "网盘服务异常"))
	return buf.String()
}

// TestLogAPIErrorSkipsCanceledContext 锁定 0.0.43 行为：客户端提前断开/主动放弃
// （上下文取消或超时）不算服务端错误，不得写 ERROR 日志（upstream d545e47）。
func TestLogAPIErrorSkipsCanceledContext(t *testing.T) {
	t.Run("客户端取消不记录", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if got := captureAPIErrorLog(ctx); got != "" {
			t.Fatalf("取消上下文不应记录日志，实际输出：%s", got)
		}
	})

	t.Run("请求超时不记录", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 0)
		defer cancel()
		if got := captureAPIErrorLog(ctx); got != "" {
			t.Fatalf("超时上下文不应记录日志，实际输出：%s", got)
		}
	})

	t.Run("正常上下文仍记录", func(t *testing.T) {
		got := captureAPIErrorLog(context.Background())
		if !strings.Contains(got, "API 请求失败") || !strings.Contains(got, "网盘服务异常") {
			t.Fatalf("正常上下文应记录 ERROR，实际输出：%q", got)
		}
	})
}
