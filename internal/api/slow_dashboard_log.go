package api

import (
	"net/http"
	"time"
)

// slowDashboardRequestThreshold：后台概况类接口超过该时长记一条 INFO，便于定位"后台卡顿"。
const slowDashboardRequestThreshold = time.Second

// dashboardOverviewPaths 是需要观测响应时长的后台概况类 GET 接口。
// 注意：上游（b5c9308）的路径表含本方精简版已删除的端点
// （cache-retention/*、media-organize/tasks、strm/tasks），此处按本方在线端点裁剪。
var dashboardOverviewPaths = map[string]struct{}{
	"/api/admin/accounts":                   {},
	"/api/admin/cache/stats":                {},
	"/api/admin/fuse/mounts":                {},
	"/api/admin/notifications":              {},
	"/api/admin/notifications/unread-count": {},
	"/api/logs/stats":                       {},
}

func (h *Handler) logSlowDashboardRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || !isDashboardOverviewPath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		startedAt := time.Now()
		next.ServeHTTP(w, r)
		elapsed := time.Since(startedAt)
		if elapsed < slowDashboardRequestThreshold {
			return
		}
		requestLogger(r.Context()).Info(
			"后台概况接口响应较慢",
			"method", r.Method,
			"path", r.URL.Path,
			"duration_ms", elapsed.Milliseconds(),
		)
	})
}

func isDashboardOverviewPath(path string) bool {
	_, ok := dashboardOverviewPaths[path]
	return ok
}
