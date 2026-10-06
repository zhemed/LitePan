package httpx

import (
	"net/http"
	"testing"
	"time"
)

// 上传客户端默认必须锁在 HTTP/1.1：关掉 h2 升级、清空 TLSNextProto、并把 ALPN 限制为 http/1.1。
// 只改其中一项时服务端仍可能协商到 h2（上游实测 h2 下这些网盘的大文件上传明显更慢）。
func TestNewUploadClientForcesHTTP11ByDefault(t *testing.T) {
	api := NewClient(ClientOptions{Timeout: 600 * time.Second})
	up := NewUploadClient(api, 60*time.Second, false)

	if up.Timeout != 0 {
		t.Fatalf("上传客户端不应有整段超时，实际 %v", up.Timeout)
	}
	tr, ok := up.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("transport 类型异常：%T", up.Transport)
	}
	if tr.ResponseHeaderTimeout != 60*time.Second {
		t.Fatalf("ResponseHeaderTimeout=%v，期望 60s（只对迟迟不返回响应头判死）", tr.ResponseHeaderTimeout)
	}
	if tr.ForceAttemptHTTP2 {
		t.Fatal("ForceAttemptHTTP2 必须为 false")
	}
	if len(tr.TLSNextProto) != 0 {
		t.Fatalf("TLSNextProto 必须清空，实际 %v", tr.TLSNextProto)
	}
	if tr.Protocols == nil || !tr.Protocols.HTTP1() || tr.Protocols.HTTP2() {
		t.Fatalf("传输协议必须只允许 HTTP/1.1，实际 %+v", tr.Protocols)
	}
	if tr.TLSClientConfig == nil {
		t.Fatal("TLSClientConfig 不应为 nil")
	}
	if len(tr.TLSClientConfig.NextProtos) != 1 || tr.TLSClientConfig.NextProtos[0] != "http/1.1" {
		t.Fatalf("ALPN 必须限制为 http/1.1，实际 %v", tr.TLSClientConfig.NextProtos)
	}
}

// 驱动显式声明需要 HTTP/2 时，必须原样返回流式客户端（保留 h2 能力）。
func TestNewUploadClientKeepsHTTP2WhenRequested(t *testing.T) {
	up := NewUploadClient(nil, 30*time.Second, true)
	if up.Timeout != 0 {
		t.Fatalf("上传客户端不应有整段超时，实际 %v", up.Timeout)
	}
	tr, ok := up.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("transport 类型异常：%T", up.Transport)
	}
	// h2 能力可以落在 ForceAttemptHTTP2 或 Protocols 上（Go 的默认 transport 用后者），
	// 只要"没有被人为锁死在 HTTP/1.1"即可。
	h2Available := tr.ForceAttemptHTTP2 || (tr.Protocols != nil && tr.Protocols.HTTP2())
	if !h2Available {
		t.Fatalf("显式声明 HTTP/2 时应保留 h2 能力：ForceAttemptHTTP2=%v Protocols=%+v",
			tr.ForceAttemptHTTP2, tr.Protocols)
	}
	if tr.TLSClientConfig != nil && len(tr.TLSClientConfig.NextProtos) == 1 && tr.TLSClientConfig.NextProtos[0] == "http/1.1" {
		t.Fatal("显式声明 HTTP/2 时不应把 ALPN 锁成 http/1.1")
	}
}

// 不修改传入的 API 客户端（transport 是 Clone 出来的，锁协议不影响普通 API 调用）。
func TestNewUploadClientDoesNotMutateBaseClient(t *testing.T) {
	api := NewClient(ClientOptions{Timeout: 30 * time.Second})
	baseTransport := api.Transport.(*http.Transport)
	before := baseTransport.ForceAttemptHTTP2

	_ = NewUploadClient(api, 60*time.Second, false)

	if baseTransport.ForceAttemptHTTP2 != before {
		t.Fatal("不应改动传入客户端的 transport")
	}
	if api.Timeout != 30*time.Second {
		t.Fatalf("不应改动传入客户端的总超时，实际 %v", api.Timeout)
	}
}
