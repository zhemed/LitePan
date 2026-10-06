package httpx

import (
	"crypto/tls"
	"net/http"
	"net/url"
	"time"
)

const (
	DefaultTimeout         = 30 * time.Second
	defaultIdleConnTimeout = 90 * time.Second
)

type ClientOptions struct {
	Timeout            time.Duration
	IdleConnTimeout    time.Duration
	DisableCompression bool
	DisableKeepAlives  bool
	Proxy              func(*http.Request) (*url.URL, error)
}

func NewClient(opts ClientOptions) *http.Client {
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	tr := http.DefaultTransport.(*http.Transport).Clone()
	idle := opts.IdleConnTimeout
	if idle <= 0 && !opts.DisableKeepAlives {
		idle = defaultIdleConnTimeout
	}
	if idle > 0 {
		tr.IdleConnTimeout = idle
	}
	if opts.DisableCompression {
		tr.DisableCompression = true
	}
	if opts.DisableKeepAlives {
		tr.DisableKeepAlives = true
		tr.MaxIdleConnsPerHost = 0
	}
	if opts.Proxy != nil {
		tr.Proxy = opts.Proxy
	}
	return &http.Client{Timeout: timeout, Transport: tr}
}

// NewStreamingClient 复用普通客户端的连接配置，但不限制整段文件传输时长。
//
// 与 NewClient 的差别只在于：不设置 http.Client.Timeout（该字段是"整个请求"的总时长上限，
// 会把大分片/慢链路的正常传输掐断），改为设置 ResponseHeaderTimeout —— 只对"连上了却
// 迟迟不返回响应头"判死。用于上传等数据面请求；普通 API 调用仍应使用 NewClient。
func NewStreamingClient(base *http.Client, responseHeaderTimeout time.Duration) *http.Client {
	tr := http.DefaultTransport.(*http.Transport).Clone()
	if base != nil {
		if baseTransport, ok := base.Transport.(*http.Transport); ok {
			tr = baseTransport.Clone()
		}
	}
	if responseHeaderTimeout > 0 {
		tr.ResponseHeaderTimeout = responseHeaderTimeout
	}
	return &http.Client{Transport: tr}
}

// NewUploadClient 创建文件上传客户端：上传默认使用 HTTP/1.1，只有驱动明确声明时才使用 HTTP/2。
//
// 115、189 这类上游的 OSS 数据面在 HTTP/2 下大文件上传明显变慢（上游实测，见 0.5.7 的
// 「优化部分驱动上传慢的问题」），因此这里默认把协议锁到 HTTP/1.1：不仅关掉 h2 升级，
// 还要清掉 TLSNextProto 并把 ALPN 限制为 http/1.1 —— 只改前者时，服务端仍可能通过 ALPN 协商到 h2。
func NewUploadClient(base *http.Client, responseHeaderTimeout time.Duration, useHTTP2 bool) *http.Client {
	client := NewStreamingClient(base, responseHeaderTimeout)
	if useHTTP2 {
		return client
	}
	tr, ok := client.Transport.(*http.Transport)
	if !ok {
		return client
	}
	tr.ForceAttemptHTTP2 = false
	tr.TLSNextProto = map[string]func(string, *tls.Conn) http.RoundTripper{}
	tr.Protocols = new(http.Protocols)
	tr.Protocols.SetHTTP1(true)
	// Clone 可能已经把 h2 写入 ALPN，必须同时限制 TLS 协商协议。
	if tr.TLSClientConfig == nil {
		tr.TLSClientConfig = &tls.Config{}
	}
	tr.TLSClientConfig.NextProtos = []string{"http/1.1"}
	return client
}

func CloseClient(c *http.Client) {
	if c == nil {
		return
	}
	tr, ok := c.Transport.(*http.Transport)
	if !ok {
		return
	}
	tr.CloseIdleConnections()
}
