package api

import (
	"net/http"
	"sync"
	"time"
)

// slowDashboardRequestThreshold：后台概况类接口超过该时长记一条慢日志，便于定位"后台卡顿"。
const slowDashboardRequestThreshold = time.Second

// slowDashboardLogInterval：同一路径两条慢日志之间的最小间隔。
// 这类接口慢一点通常不影响使用，属于排查信息；后台概况接口一旦变慢往往持续变慢，
// 逐次刷屏只会淹没真正有用的日志，因此窗口内的慢请求只累计次数，
// 等窗口结束再带上 suppressed_count 汇总一条。
const slowDashboardLogInterval = 30 * time.Minute

// dashboardOverviewPaths 是需要观测响应时长的后台概况类 GET 接口。
// 注意：上游的路径表含本方精简版已删除的端点
// （cache-retention/*、media-organize/tasks、strm/tasks），此处按本方在线端点裁剪。
var dashboardOverviewPaths = map[string]struct{}{
	"/api/admin/accounts":                   {},
	"/api/admin/cache/stats":                {},
	"/api/admin/notifications":              {},
	"/api/admin/notifications/unread-count": {},
	"/api/logs/stats":                       {},
}

type slowRequestLogEntry struct {
	lastLogged time.Time
	suppressed int
}

// slowRequestLogs 记录各路径的慢日志抑制窗口。
// 零值可用：测试会直接构造 Handler，必须容忍尚未初始化的 map。
type slowRequestLogs struct {
	mu      sync.Mutex
	entries map[string]slowRequestLogEntry
}

// shouldLog 判断该路径现在是否可以写慢日志，并返回本次一并上报的压制次数。
// 处于抑制窗口内时只累计次数不写日志。
func (s *slowRequestLogs) shouldLog(path string, now time.Time) (int, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.entries == nil {
		s.entries = make(map[string]slowRequestLogEntry)
	}
	entry := s.entries[path]
	if !entry.lastLogged.IsZero() && now.Sub(entry.lastLogged) < slowDashboardLogInterval {
		entry.suppressed++
		s.entries[path] = entry
		return 0, false
	}
	s.entries[path] = slowRequestLogEntry{lastLogged: now}
	return entry.suppressed, true
}

// recovered 处理"本次已经变快"的请求：若此前有被压制的慢日志，返回次数并结束本轮窗口，
// 使下一次变慢可以立刻记录。
//
// 与上游不同之处：这里只在"确有压制记录"时才结束窗口。否则一次偶发的快请求
// （轮询、健康检查）会把 30 分钟窗口冲掉，间歇性变慢就退化成逐次刷屏。
func (s *slowRequestLogs) recovered(path string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.entries[path]
	if !ok || entry.suppressed == 0 {
		return 0
	}
	delete(s.entries, path)
	return entry.suppressed
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
			if suppressed := h.slowLogs.recovered(r.URL.Path); suppressed > 0 {
				requestLogger(r.Context()).Debug(
					"后台概况接口已恢复正常",
					"method", r.Method,
					"path", r.URL.Path,
					"suppressed_count", suppressed,
				)
			}
			return
		}
		// 降为 Debug：这是给排查用的诊断信息，不是用户该关心的状态。
		// 需要看的时候把日志级别调到 Debug，就能看到是哪个接口、慢了多少。
		suppressed, ok := h.slowLogs.shouldLog(r.URL.Path, time.Now())
		if !ok {
			return
		}
		requestLogger(r.Context()).Debug(
			"后台概况接口响应较慢",
			"method", r.Method,
			"path", r.URL.Path,
			"duration_ms", elapsed.Milliseconds(),
			"suppressed_count", suppressed,
		)
	})
}

func isDashboardOverviewPath(path string) bool {
	_, ok := dashboardOverviewPaths[path]
	return ok
}
