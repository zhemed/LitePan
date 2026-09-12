package httpx

import (
	"net/http"
	"net/url"
	"testing"
	"time"
)

// TestNewStreamingClientHasNoTotalTimeout 保证数据面客户端不受"整请求总时长"限制。
func TestNewStreamingClientHasNoTotalTimeout(t *testing.T) {
	base := NewClient(ClientOptions{Timeout: 600 * time.Second})
	c := NewStreamingClient(base, 60*time.Second)
	if c.Timeout != 0 {
		t.Fatalf("流式客户端不应设置总超时，实际 %v", c.Timeout)
	}
}

// TestNewStreamingClientSetsResponseHeaderTimeout 校验响应头超时（唯一的兜底判死条件）。
func TestNewStreamingClientSetsResponseHeaderTimeout(t *testing.T) {
	c := NewStreamingClient(nil, 60*time.Second)
	tr, ok := c.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("transport 类型异常：%T", c.Transport)
	}
	if tr.ResponseHeaderTimeout != 60*time.Second {
		t.Fatalf("ResponseHeaderTimeout=%v，期望 60s", tr.ResponseHeaderTimeout)
	}

	// <=0 时不设置，避免误把 0 当成"立即超时"。
	c2 := NewStreamingClient(nil, 0)
	tr2 := c2.Transport.(*http.Transport)
	if tr2.ResponseHeaderTimeout != 0 {
		t.Fatalf("未指定响应头超时时不应设置，实际 %v", tr2.ResponseHeaderTimeout)
	}
}

// TestNewStreamingClientInheritsBaseTransport 校验继承 base 的连接配置（代理/压缩/空闲连接）。
func TestNewStreamingClientInheritsBaseTransport(t *testing.T) {
	proxy := func(*http.Request) (*url.URL, error) { return url.Parse("http://proxy.invalid:8080") }
	base := NewClient(ClientOptions{Timeout: 30 * time.Second, DisableCompression: true, Proxy: proxy})
	baseTr := base.Transport.(*http.Transport)
	if !baseTr.DisableCompression {
		t.Fatal("前置条件不成立：base 应禁用压缩")
	}

	c := NewStreamingClient(base, 60*time.Second)
	tr := c.Transport.(*http.Transport)
	if !tr.DisableCompression {
		t.Fatal("应继承 base 的 DisableCompression")
	}
	if tr.Proxy == nil {
		t.Fatal("应继承 base 的 Proxy")
	}
	if tr.IdleConnTimeout != defaultIdleConnTimeout {
		t.Fatalf("应继承 base 的 IdleConnTimeout=%v，实际 %v", defaultIdleConnTimeout, tr.IdleConnTimeout)
	}
	// base 自身不得被修改（共享 transport 会互相影响）。
	if baseTr.ResponseHeaderTimeout != 0 {
		t.Fatalf("base 的 ResponseHeaderTimeout 被污染：%v", baseTr.ResponseHeaderTimeout)
	}
}

// TestNewStreamingClientFallsBackToDefaultTransport 校验 base 为 nil 或非 *http.Transport 时可用。
func TestNewStreamingClientFallsBackToDefaultTransport(t *testing.T) {
	for name, base := range map[string]*http.Client{
		"nil":              nil,
		"nil-transport":    {Transport: nil},
		"custom-transport": {Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) { return nil, nil })},
	} {
		c := NewStreamingClient(base, 60*time.Second)
		if c.Timeout != 0 {
			t.Fatalf("%s: 总超时应为 0，实际 %v", name, c.Timeout)
		}
		tr, ok := c.Transport.(*http.Transport)
		if !ok {
			t.Fatalf("%s: 期望回退为 *http.Transport，实际 %T", name, c.Transport)
		}
		if tr.ResponseHeaderTimeout != 60*time.Second {
			t.Fatalf("%s: ResponseHeaderTimeout=%v", name, tr.ResponseHeaderTimeout)
		}
	}
}

// TestCloseClientAcceptsStreamingClient 校验关闭路径对两类客户端都安全。
func TestCloseClientAcceptsStreamingClient(t *testing.T) {
	CloseClient(nil)
	CloseClient(NewStreamingClient(NewClient(ClientOptions{}), 60*time.Second))
	CloseClient(&http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) { return nil, nil })})
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
