package playback

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"litepan/internal/core/driverexec"
	"litepan/internal/domain"
	"litepan/internal/driver"
)

const (
	uaAndroidChrome = "Mozilla/5.0 (Linux; Android 14; SM-S9110) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Mobile Safari/537.36"
	uaAndroidDLM    = "AndroidDownloadManager/14 (Linux; U; Android 14)"
	uaDesktopChrome = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Safari/537.36"
	uaIOSSafari     = "Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.0 Mobile/15E148 Safari/604.1"
)

// —— 测试替身：驱动只用于满足 Executor，直链由 resolveHook 提供 ——

type stubDriver struct{}

func (stubDriver) Config() driver.Config                                        { return driver.Config{Name: "stub"} }
func (stubDriver) GetAddition() any                                             { return &struct{}{} }
func (stubDriver) Init(context.Context) error                                   { return nil }
func (stubDriver) Drop(context.Context) error                                   { return nil }
func (stubDriver) Ping(context.Context) error                                   { return nil }
func (stubDriver) ListFiles(context.Context, string) ([]domain.FileItem, error) { return nil, nil }

type stubProvider struct{}

func (stubProvider) Get(context.Context, int64) (driver.Driver, error) { return stubDriver{}, nil }

// newHandoffTestService 返回一个 302 模式（DownloadRedirect）的服务，并记录每次解析用的 UA。
func newHandoffTestService(t *testing.T) (*Service, *[]string) {
	t.Helper()
	s := NewService(driverexec.New(stubProvider{}, nil), nil)
	calls := &[]string{}
	s.SetDownloadResolverHook(func(_ context.Context, _ int64, _, fileID, ua string, _ bool) (*domain.DownloadInfo, bool, error) {
		*calls = append(*calls, ua)
		return &domain.DownloadInfo{
			URL:      "https://cdn.invalid/f/" + fileID + "?ua=" + url.QueryEscape(ua),
			Mode:     domain.DownloadRedirect,
			Size:     1024,
			FileName: "测试.zip",
		}, true, nil
	})
	return s, calls
}

func doRequest(t *testing.T, s *Service, method, ua, fileID string, intent Intent) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(method, "/api/files/download?account_id=1&file_id="+fileID, nil)
	r.Header.Set("User-Agent", ua)
	rec := httptest.NewRecorder()
	if err := s.ServeHTTP(rec, r, Request{AccountID: 1, FileID: fileID}, intent); err != nil {
		t.Fatalf("ServeHTTP 返回错误：%v", err)
	}
	return rec
}

func assertStub(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	if rec.Code != http.StatusOK {
		t.Fatalf("空附件应为 200，实际 %d", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "" {
		t.Fatalf("空附件不应带 Location，实际 %q", loc)
	}
	if cd := rec.Header().Get("Content-Disposition"); !strings.Contains(cd, "attachment") {
		t.Fatalf("空附件应带 attachment，实际 %q", cd)
	}
	if cl := rec.Header().Get("Content-Length"); cl != "0" {
		t.Fatalf("空附件应为 0 字节，实际 %q", cl)
	}
	if rec.Body.Len() != 0 {
		t.Fatalf("空附件不应有响应体，实际 %d 字节", rec.Body.Len())
	}
}

func assertRedirect(t *testing.T, rec *httptest.ResponseRecorder, wantUA string) {
	t.Helper()
	if rec.Code != http.StatusFound {
		t.Fatalf("应 302，实际 %d", rec.Code)
	}
	loc := rec.Header().Get("Location")
	if !strings.HasPrefix(loc, "https://cdn.invalid/f/") {
		t.Fatalf("Location 异常：%q", loc)
	}
	if wantUA != "" && !strings.Contains(loc, url.QueryEscape(wantUA)) {
		t.Fatalf("Location 未按本次请求 UA 取链：%q 应含 %q", loc, url.QueryEscape(wantUA))
	}
}

// 1. 安卓浏览器首次 → 空附件（把手边的地址交给系统下载组件）
func TestHandoffFirstAndroidBrowserGetsEmptyAttachment(t *testing.T) {
	s, calls := newHandoffTestService(t)
	rec := doRequest(t, s, http.MethodGet, uaAndroidChrome, "f1", Intent{FileName: "测试.zip"})
	assertStub(t, rec)
	if len(*calls) != 1 || (*calls)[0] != uaAndroidChrome {
		t.Fatalf("首次解析应使用浏览器 UA，实际 %v", *calls)
	}
}

// 2. 同资源第二次（同 UA）→ 302（浏览器自己回来取的情形）
func TestHandoffSecondRequestSameUARedirects(t *testing.T) {
	s, _ := newHandoffTestService(t)
	doRequest(t, s, http.MethodGet, uaAndroidChrome, "f2", Intent{})
	rec := doRequest(t, s, http.MethodGet, uaAndroidChrome, "f2", Intent{})
	assertRedirect(t, rec, uaAndroidChrome)
}

// 3. 同资源第二次（安卓系统下载组件 UA）→ 302，且**按组件 UA**取链（本修复的核心）
func TestHandoffSecondRequestDownloaderUARedirectsWithOwnUA(t *testing.T) {
	s, calls := newHandoffTestService(t)
	doRequest(t, s, http.MethodGet, uaAndroidChrome, "f3", Intent{})
	rec := doRequest(t, s, http.MethodGet, uaAndroidDLM, "f3", Intent{})
	assertRedirect(t, rec, uaAndroidDLM)
	if last := (*calls)[len(*calls)-1]; last != uaAndroidDLM {
		t.Fatalf("第二跳应按下载组件 UA 取链，实际 %q", last)
	}
}

// 4/5. 桌面与 iOS 首次即 302，行为与改动前一致
func TestHandoffDesktopAndIOSRedirectImmediately(t *testing.T) {
	for name, ua := range map[string]string{"desktop": uaDesktopChrome, "ios": uaIOSSafari} {
		s, _ := newHandoffTestService(t)
		rec := doRequest(t, s, http.MethodGet, ua, "f-"+name, Intent{})
		assertRedirect(t, rec, ua)
	}
}

// 6. HEAD 走原 302（元数据探测不受影响）
func TestHandoffHeadRedirects(t *testing.T) {
	s, _ := newHandoffTestService(t)
	rec := doRequest(t, s, http.MethodHead, uaAndroidChrome, "f4", Intent{})
	assertRedirect(t, rec, uaAndroidChrome)
}

// 7. 播放/内联不吃空附件
func TestHandoffInlineIntentRedirects(t *testing.T) {
	s, _ := newHandoffTestService(t)
	rec := doRequest(t, s, http.MethodGet, uaAndroidChrome, "f5", Intent{Inline: true})
	assertRedirect(t, rec, uaAndroidChrome)
}

// 8. 不同文件互不影响（key 隔离）
func TestHandoffWindowIsPerFile(t *testing.T) {
	s, _ := newHandoffTestService(t)
	assertStub(t, doRequest(t, s, http.MethodGet, uaAndroidChrome, "f6a", Intent{}))
	assertStub(t, doRequest(t, s, http.MethodGet, uaAndroidChrome, "f6b", Intent{}))
}

// 9. 握手窗口过期后可再次握手（用户重试体验）
func TestHandoffWindowExpiryAllowsNewStub(t *testing.T) {
	s, _ := newHandoffTestService(t)
	assertStub(t, doRequest(t, s, http.MethodGet, uaAndroidChrome, "f7", Intent{}))
	assertRedirect(t, doRequest(t, s, http.MethodGet, uaAndroidChrome, "f7", Intent{}), uaAndroidChrome)
	s.tracker().window = 0 // 模拟窗口过期（等价于注入时钟跨过窗口）
	assertStub(t, doRequest(t, s, http.MethodGet, uaAndroidChrome, "f7", Intent{}))
}

// 10. 代理模式不受握手影响（ForceProxy 仍走流式）
func TestForceProxyStillStreams(t *testing.T) {
	if got := PickAction(domain.DownloadRedirect, domain.DownloadInfo{ForceProxy: true}, Intent{}); got != ActionStream {
		t.Fatalf("ForceProxy 应走 ActionStream，实际 %v", got)
	}
	if got := PickAction(domain.DownloadProxy, domain.DownloadInfo{}, Intent{}); got != ActionStream {
		t.Fatalf("代理模式应走 ActionStream，实际 %v", got)
	}
}

// UA 判定表：只有安卓浏览器/WebView 命中，系统下载组件不命中。
func TestIsAndroidBrowserUA(t *testing.T) {
	cases := map[string]bool{
		uaAndroidChrome: true,
		uaAndroidDLM:    false,
		uaDesktopChrome: false,
		uaIOSSafari:     false,
		"":              false,
		"Mozilla/5.0 (Linux; Android 13; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/141.0.0.0 Mobile Safari/537.36": true,
		"Mozilla/5.0 (Linux; Android 13) AppleWebKit/537.36 (KHTML, like Gecko) SamsungBrowser/23.0 Chrome/115 Mobile Safari/537.36":   true,
		"curl/8.4.0": false,
	}
	for ua, want := range cases {
		if got := isAndroidBrowserUA(ua); got != want {
			t.Fatalf("isAndroidBrowserUA(%q) = %v，期望 %v", ua, got, want)
		}
	}
}

// 状态表容量保护：清理过期项后仍能正常判定。
func TestHandoffTrackerPrunesExpired(t *testing.T) {
	tr := &handoffTracker{seen: make(map[string]time.Time), now: time.Now, window: 0}
	for i := 0; i < handoffStateMax; i++ {
		if !tr.first("k" + string(rune('a'+i%26)) + string(rune('a'+i/26))) {
			t.Fatalf("第 %d 次插入应视为首次", i)
		}
	}
	if !tr.first("brand-new") {
		t.Fatal("容量满后仍应能判定新键为首次")
	}
}
