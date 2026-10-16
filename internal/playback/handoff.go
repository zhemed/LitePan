package playback

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// 安卓两跳握手：见 .trellis/tasks/10-16-fix-android-302-handoff/design.md
//
// 背景：115 CDN 直链的签名只绑定"取链时的 User-Agent"（实测：同 UA 跨 IP 可下，换 UA 一律
// 403 invalid signature）。安卓浏览器不自己下载附件，而是把**最终 URL** 交给系统下载组件，
// 组件用的是另一个 UA → 必然 403；而该 403 的响应头是 application/zip + attachment，
// 安卓侧就表现为 ERR_INVALID_RESPONSE。
//
// 处置：对安卓浏览器先回一个"空附件"（不 302），让浏览器把**本服务自己的下载地址**交给下载组件；
// 组件带着自己的 UA 回来请求时，再按该 UA 取链 302 —— 组件随后用同一个 UA 直连 CDN，签名匹配。

const (
	// handoffWindow：同一资源在此时长内只发一次空附件（防循环 + 覆盖"浏览器自己回来取"）。
	handoffWindow = 90 * time.Second
	// handoffStateMax：状态表容量上限，超过即清理过期项。
	handoffStateMax = 512
)

// androidBrowserMarkers：安卓浏览器 / WebView 的标识；安卓系统下载组件（如
// "AndroidDownloadManager/14 (Linux; U; Android 14)"）不含这些标识。
var androidBrowserMarkers = []string{
	"Chrome/", "Safari/", "Firefox/", "WebView", "SamsungBrowser", "MiuiBrowser",
	"UCBrowser", "Quark", "HuaweiBrowser", "HeyTapBrowser", "EdgA/", "VivoBrowser", "OPR/",
}

// isAndroidBrowserUA 判定"会交接给系统下载器的安卓浏览器"。
func isAndroidBrowserUA(ua string) bool {
	if !strings.Contains(ua, "Android") {
		return false
	}
	for _, m := range androidBrowserMarkers {
		if strings.Contains(ua, m) {
			return true
		}
	}
	return false
}

type handoffTracker struct {
	mu     sync.Mutex
	seen   map[string]time.Time
	now    func() time.Time
	window time.Duration
}

func newHandoffTracker() *handoffTracker {
	return &handoffTracker{seen: make(map[string]time.Time), now: time.Now, window: handoffWindow}
}

// first 报告 key 是否是窗口内的首次出现；是则记录当前时间。
func (t *handoffTracker) first(key string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	now := t.now()
	if t.seen == nil {
		t.seen = make(map[string]time.Time)
	}
	if at, ok := t.seen[key]; ok && now.Sub(at) < t.window {
		return false
	}
	if len(t.seen) >= handoffStateMax {
		for k, at := range t.seen {
			if now.Sub(at) >= t.window {
				delete(t.seen, k)
			}
		}
	}
	t.seen[key] = now
	return true
}

func handoffKey(accountID int64, fileID string) string {
	return fmt.Sprintf("%d\x00%s", accountID, fileID)
}

func (s *Service) tracker() *handoffTracker {
	s.handoffOnce.Do(func() {
		if s.handoff == nil {
			s.handoff = newHandoffTracker()
		}
	})
	return s.handoff
}

// handoffStub 在满足握手条件时写出空附件并返回 true。
// 条件（全部满足）：GET、非 Inline（播放不看不动）、安卓浏览器 UA、该资源窗口内首次。
func (s *Service) handoffStub(w http.ResponseWriter, r *http.Request, req Request, intent Intent) bool {
	if r.Method != http.MethodGet || intent.Inline || !isAndroidBrowserUA(r.UserAgent()) {
		return false
	}
	if !s.tracker().first(handoffKey(req.AccountID, req.FileID)) {
		return false
	}
	writeHandoffStub(w, intent)
	return true
}

// writeHandoffStub 写出"空附件"：只是把手边的下载地址交给系统下载组件，本身不含数据。
func writeHandoffStub(w http.ResponseWriter, intent Intent) {
	writeDynamicRedirectHeaders(w.Header())
	disposition := "attachment"
	if name := strings.TrimSpace(intent.FileName); name != "" {
		disposition += fmt.Sprintf("; filename*=UTF-8''%s", url.PathEscape(name))
	}
	h := w.Header()
	h.Set("Content-Disposition", disposition)
	h.Set("Content-Type", "application/octet-stream")
	h.Set("Content-Length", "0")
	w.WriteHeader(http.StatusOK)
}
