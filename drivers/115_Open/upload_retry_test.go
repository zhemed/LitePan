package pan115open

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"litepan/internal/driver"
	"litepan/internal/httpx"
)

type uploadRoundTripFunc func(*http.Request) (*http.Response, error)

func (f uploadRoundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type stubNetError struct{ msg string }

func (e stubNetError) Error() string   { return e.msg }
func (e stubNetError) Timeout() bool   { return false }
func (e stubNetError) Temporary() bool { return true }

// TestIsRetryableOSSUploadError 锁定"哪些错误值得重试"：瞬时网络/网关类可重试，
// 业务类（4xx/缺 ETag）不可重试。
func TestIsRetryableOSSUploadError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"net-error", stubNetError{msg: "dial tcp: i/o timeout"}, true},
		{"unexpected-eof", errors.New("unexpected EOF"), true},
		{"connection-reset", errors.New("read tcp: connection reset by peer"), true},
		{"broken-pipe", errors.New("write tcp: broken pipe"), true},
		{"http-429", errors.New("上传 115 OSS 分片失败(part 1): HTTP 429: slow down"), true},
		{"http-500", errors.New("上传 115 OSS 分片失败(part 1): HTTP 500: internal error"), true},
		{"http-502", errors.New("HTTP 502: bad gateway"), true},
		{"http-503", errors.New("HTTP 503: service unavailable"), true},
		{"http-504", errors.New("HTTP 504: gateway timeout"), true},
		{"http-403", errors.New("上传 115 OSS 分片失败(part 1): HTTP 403: AccessDenied"), false},
		{"http-404", errors.New("上传 115 OSS 分片失败(part 1): HTTP 404: NoSuchUpload"), false},
		{"missing-etag", errors.New("上传 115 OSS 分片失败(part 1)，未返回 ETag"), false},
	}
	for _, tc := range cases {
		if got := isRetryableOSSUploadError(tc.err); got != tc.want {
			t.Fatalf("%s: isRetryableOSSUploadError=%v，期望 %v", tc.name, got, tc.want)
		}
	}
}

// TestNewOSSUploadHTTPClientHasNoTotalTimeout 锁定数据面客户端语义：不设总时长上限。
func TestNewOSSUploadHTTPClientHasNoTotalTimeout(t *testing.T) {
	api := httpx.NewClient(httpx.ClientOptions{Timeout: 600 * time.Second})
	if api.Timeout != 600*time.Second {
		t.Fatalf("前置条件不成立：API 客户端总超时=%v", api.Timeout)
	}
	up := newOSSUploadHTTPClient(api)
	if up.Timeout != 0 {
		t.Fatalf("上传客户端不应有总超时，实际 %v", up.Timeout)
	}
	tr, ok := up.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("transport 类型异常：%T", up.Transport)
	}
	if tr.ResponseHeaderTimeout != 60*time.Second {
		t.Fatalf("ResponseHeaderTimeout=%v，期望 60s", tr.ResponseHeaderTimeout)
	}
}

// TestOSSUploadHTTPClientFallsBack 未初始化 uploadClient 时回退 API 客户端（防空指针）。
func TestOSSUploadHTTPClientFallsBack(t *testing.T) {
	api := httpx.NewClient(httpx.ClientOptions{Timeout: 30 * time.Second})
	d := &Driver{client: api}
	if d.ossUploadHTTPClient() != api {
		t.Fatal("uploadClient 为空时应回退到 client")
	}
	up := httpx.NewStreamingClient(api, 60*time.Second)
	d.uploadClient = up
	if d.ossUploadHTTPClient() != up {
		t.Fatal("已初始化时应使用 uploadClient")
	}
}

// TestOSSUploadPartWithRetryRetriesTransientFailure 首次传输层失败后应重试并成功，
// 且两次尝试读到的都是同一段文件内容（Seek 复位，进度不重复累加）。
func TestOSSUploadPartWithRetryRetriesTransientFailure(t *testing.T) {
	const partSize = 1024
	payload := bytes.Repeat([]byte("LitePan"), partSize/7+1)[:partSize]
	f := writeTempUploadFile(t, payload)

	var bodies [][]byte
	attempts := 0
	progress := make([]int64, 0, 4)
	rt := uploadRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		attempts++
		body, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		bodies = append(bodies, body)
		if attempts == 1 {
			return nil, errors.New("unexpected EOF")
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			// 注意：http.Header 以"规范化键"为准，ETag 的规范形是 Etag；
			// 用 "ETag" 作为 map 字面量键会导致 Header.Get 查不到（测试本身踩过这个坑）。
			Header:  http.Header{"Etag": []string{`"etag-1"`}},
			Body:    io.NopCloser(strings.NewReader("")),
			Request: req,
		}, nil
	})
	d := &Driver{uploadClient: &http.Client{Transport: rt}}
	onProgress := func(sent, total int64, message string) { progress = append(progress, sent) }

	etag, err := d.ossUploadPartWithRetry(context.Background(), testOSSToken(), "bucket", "obj/key.bin",
		"upload-1", 1, f, partSize, 0, partSize, onProgress, 1)
	if err != nil {
		t.Fatalf("重试后应成功，实际 err=%v", err)
	}
	if etag != "etag-1" {
		t.Fatalf("etag=%q，期望 etag-1", etag)
	}
	if attempts != 2 {
		t.Fatalf("尝试次数=%d，期望 2", attempts)
	}
	for i, body := range bodies {
		if !bytes.Equal(body, payload) {
			t.Fatalf("第 %d 次尝试发送的字节与分片内容不一致（len=%d）", i+1, len(body))
		}
	}
	if len(progress) == 0 || progress[len(progress)-1] != partSize {
		t.Fatalf("进度应收敛到 %d，实际 %v", partSize, progress)
	}
}

// TestOSSUploadPartWithRetryHonorsOffset 带偏移的分片重试同样从 uploadedOffset 重新读取。
func TestOSSUploadPartWithRetryHonorsOffset(t *testing.T) {
	const partSize = 512
	payload := bytes.Repeat([]byte("x"), 2048)
	f := writeTempUploadFile(t, payload)

	var bodies [][]byte
	attempts := 0
	rt := uploadRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		attempts++
		body, _ := io.ReadAll(req.Body)
		bodies = append(bodies, body)
		if attempts == 1 {
			return nil, errors.New("connection reset by peer")
		}
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Etag": []string{`"e"`}}, Body: io.NopCloser(strings.NewReader("")), Request: req}, nil
	})
	d := &Driver{uploadClient: &http.Client{Transport: rt}}

	if _, err := d.ossUploadPartWithRetry(context.Background(), testOSSToken(), "b", "o", "u", 3, f, partSize, 1024, 2048, nil, 4); err != nil {
		t.Fatalf("重试后应成功：%v", err)
	}
	if attempts != 2 {
		t.Fatalf("尝试次数=%d，期望 2", attempts)
	}
	want := payload[1024 : 1024+partSize]
	for i, body := range bodies {
		if !bytes.Equal(body, want) {
			t.Fatalf("第 %d 次尝试的偏移内容不正确", i+1)
		}
	}
}

// TestOSSUploadPartWithRetryStopsAtLimit 连续瞬时失败最多尝试 ossUploadAttempts 次。
func TestOSSUploadPartWithRetryStopsAtLimit(t *testing.T) {
	f := writeTempUploadFile(t, bytes.Repeat([]byte("y"), 256))
	attempts := 0
	rt := uploadRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		attempts++
		return nil, errors.New("connection reset by peer")
	})
	d := &Driver{uploadClient: &http.Client{Transport: rt}}

	if _, err := d.ossUploadPartWithRetry(context.Background(), testOSSToken(), "b", "o", "u", 1, f, 256, 0, 256, nil, 1); err == nil {
		t.Fatal("持续失败应返回错误")
	}
	if attempts != ossUploadAttempts {
		t.Fatalf("尝试次数=%d，期望 %d", attempts, ossUploadAttempts)
	}
}

// TestOSSUploadPartWithRetryDoesNotRetryBusinessError 业务类错误（HTTP 403）立即返回。
func TestOSSUploadPartWithRetryDoesNotRetryBusinessError(t *testing.T) {
	f := writeTempUploadFile(t, bytes.Repeat([]byte("z"), 256))
	attempts := 0
	rt := uploadRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		attempts++
		return &http.Response{
			StatusCode: http.StatusForbidden,
			Header:     http.Header{},
			Body:       io.NopCloser(strings.NewReader("AccessDenied")),
			Request:    req,
		}, nil
	})
	d := &Driver{uploadClient: &http.Client{Transport: rt}}

	_, err := d.ossUploadPartWithRetry(context.Background(), testOSSToken(), "b", "o", "u", 1, f, 256, 0, 256, nil, 1)
	if err == nil {
		t.Fatal("HTTP 403 应返回错误")
	}
	if attempts != 1 {
		t.Fatalf("业务错误不应重试，实际尝试 %d 次", attempts)
	}
}

// TestOSSUploadPartWithRetryDoesNotRetryCredentialError 凭证类错误交由调用方刷新链路处理，
// 不在这里形成 3× 刷新放大（HTTP 500 + SecurityTokenExpired 同时命中两条判定）。
func TestOSSUploadPartWithRetryDoesNotRetryCredentialError(t *testing.T) {
	f := writeTempUploadFile(t, bytes.Repeat([]byte("w"), 256))
	attempts := 0
	rt := uploadRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		attempts++
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
			Header:     http.Header{},
			Body:       io.NopCloser(strings.NewReader(`{"Code":"SecurityTokenExpired"}`)),
			Request:    req,
		}, nil
	})
	d := &Driver{uploadClient: &http.Client{Transport: rt}}

	_, err := d.ossUploadPartWithRetry(context.Background(), testOSSToken(), "b", "o", "u", 1, f, 256, 0, 256, nil, 1)
	if err == nil {
		t.Fatal("凭证错误应返回错误")
	}
	if !isOSSCredentialError(err) {
		t.Fatalf("测试前提不成立：错误未被判为凭证类：%v", err)
	}
	if !isRetryableOSSUploadError(err) {
		t.Fatalf("测试前提不成立：该错误本应落入瞬时重试分类：%v", err)
	}
	if attempts != 1 {
		t.Fatalf("凭证错误不应在此重试，实际尝试 %d 次", attempts)
	}
}

// TestOSSUploadPartWithRetryStopsOnCanceledContext 任务取消（暂停/删除）后不再发起新的尝试。
//
// 说明：这里用"在首次尝试中取消"来验证本函数的语义——ctx 在尝试之间被检查，取消后立即返回，
// 不做第二次尝试。至于"取消后连第一次请求都不发出"由 net/http 的 Transport 负责（真实
// transport 会在拨号前检查 ctx），假 transport 不承担该职责，故不在此断言 attempts==0。
func TestOSSUploadPartWithRetryStopsOnCanceledContext(t *testing.T) {
	f := writeTempUploadFile(t, bytes.Repeat([]byte("c"), 256))
	attempts := 0
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	rt := uploadRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		attempts++
		cancel() // 第一次尝试途中用户取消（暂停/删除任务）
		return nil, errors.New("unexpected EOF")
	})
	d := &Driver{uploadClient: &http.Client{Transport: rt}}

	_, err := d.ossUploadPartWithRetry(ctx, testOSSToken(), "b", "o", "u", 1, f, 256, 0, 256, nil, 1)
	if err == nil {
		t.Fatal("取消后的失败应返回错误")
	}
	if attempts != 1 {
		t.Fatalf("取消后不应再尝试，实际尝试 %d 次", attempts)
	}
}

// TestOSSUploadPartWithRetryAbortsBackoffOnCancel 退避等待期间取消应立即返回，不空等完 1s/2s。
func TestOSSUploadPartWithRetryAbortsBackoffOnCancel(t *testing.T) {
	f := writeTempUploadFile(t, bytes.Repeat([]byte("d"), 256))
	attempts := 0
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	rt := uploadRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		attempts++
		go func() {
			time.Sleep(50 * time.Millisecond)
			cancel()
		}()
		return nil, errors.New("connection reset by peer")
	})
	d := &Driver{uploadClient: &http.Client{Transport: rt}}

	start := time.Now()
	_, err := d.ossUploadPartWithRetry(ctx, testOSSToken(), "b", "o", "u", 1, f, 256, 0, 256, nil, 1)
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("取消应返回错误")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("取消应返回 context.Canceled，实际 %v", err)
	}
	if attempts != 1 {
		t.Fatalf("退避期间取消不应再次尝试，实际 %d 次", attempts)
	}
	if elapsed >= time.Second {
		t.Fatalf("退避应被取消打断，实际耗时 %v", elapsed)
	}
}

func testOSSToken() ossTokenData {
	return ossTokenData{
		AccessKeyID:     "ak-test",
		AccessKeySecret: "sk-test",
		SecurityToken:   "st-test",
		Endpoint:        "oss-cn-test.aliyuncs.com",
	}
}

func writeTempUploadFile(t *testing.T, data []byte) *os.File {
	t.Helper()
	path := filepath.Join(t.TempDir(), "part.bin")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = f.Close() })
	return f
}

var _ driver.UploadProgress = func(int64, int64, string) {}
