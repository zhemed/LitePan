package file

import (
	"sync"
	"time"
)

// cooldownLogGate 是「账号网络冷却」日志的窗口抑制器（0.0.32）。
//
// 背景：账号级网络退避（driverexec：连续 3 次网络失败 → 30s）期间，每个被派发的
// 任务都会在入口短路返回冷却错误，于是「1 次冷却 × N 个任务 = N 条同构 INFO」——
// 生产实测一次 0.35 秒刷出 183 条（任务全部安全，只是噪音）。
//
// 策略：同一账号同一窗口内只保留 1 条 INFO（首条，含重试秒数），其余降为 Debug
// （默认 log_level=info 时不可见，调 debug 仍可取证）；窗口结束或该账号恢复成功时
// 补 1 条 INFO 汇总（暂缓任务数 + 窗口秒数），规模信息不丢、重复噪音消失。
type cooldownLogGate struct {
	mu      sync.Mutex
	windows map[int64]*cooldownLogWindow
}

// cooldownLogWindow 记录某账号当前冷却窗口内的命中统计。
type cooldownLogWindow struct {
	started  time.Time
	until    time.Time
	deferred int
}

// cooldownWindowSummary 供调用方输出一条汇总日志。
type cooldownWindowSummary struct {
	Deferred      int
	WindowSeconds int
}

func newCooldownLogGate() *cooldownLogGate {
	return &cooldownLogGate{windows: make(map[int64]*cooldownLogWindow)}
}

// noteCooldown 记录一次冷却命中：
//   - first=true 表示这是该窗口首条，应输出 INFO；
//   - seq 为窗口内命中序号（1 起），供 Debug 行定位重复次数；
//   - prev 非 nil 表示上一个窗口已过期，调用方应先输出它的汇总（避免无恢复成功时丢信息）。
func (g *cooldownLogGate) noteCooldown(accountID int64, retryAfter time.Duration, now time.Time) (first bool, seq int, prev *cooldownWindowSummary) {
	if g == nil {
		return true, 1, nil
	}
	if retryAfter <= 0 {
		retryAfter = time.Second
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.windows == nil {
		g.windows = make(map[int64]*cooldownLogWindow)
	}
	w := g.windows[accountID]
	if w == nil || now.After(w.until) {
		if w != nil {
			prev = w.summary(now)
		}
		w = &cooldownLogWindow{started: now}
		g.windows[accountID] = w
		first = true
	}
	w.until = now.Add(retryAfter)
	w.deferred++
	return first, w.deferred, prev
}

// noteRecovered 该账号一次调用已成功：关闭窗口并返回汇总；无窗口时返回 nil。
func (g *cooldownLogGate) noteRecovered(accountID int64, now time.Time) *cooldownWindowSummary {
	if g == nil {
		return nil
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	w := g.windows[accountID]
	if w == nil {
		return nil
	}
	delete(g.windows, accountID)
	return w.summary(now)
}

func (w *cooldownLogWindow) summary(now time.Time) *cooldownWindowSummary {
	seconds := int(now.Sub(w.started) / time.Second)
	if seconds < 1 {
		seconds = 1
	}
	return &cooldownWindowSummary{Deferred: w.deferred, WindowSeconds: seconds}
}
