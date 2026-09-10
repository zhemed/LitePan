package file

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	"litepan/internal/core/driverexec"
	"litepan/internal/domain"
	"litepan/internal/driver"
)

type cooldownLogDriver struct{}

func (cooldownLogDriver) Config() driver.Config      { return driver.Config{Name: "test"} }
func (cooldownLogDriver) GetAddition() any           { return struct{}{} }
func (cooldownLogDriver) Init(context.Context) error { return nil }
func (cooldownLogDriver) Drop(context.Context) error { return nil }
func (cooldownLogDriver) Ping(context.Context) error { return nil }
func (cooldownLogDriver) ListFiles(context.Context, string) ([]domain.FileItem, error) {
	return nil, nil
}
func (cooldownLogDriver) UploadLocalFile(context.Context, driver.LocalUploadRequest) (*driver.LocalUploadResult, error) {
	return nil, driverexec.CooldownError(30 * time.Second)
}

type denyLogDriver struct{ cooldownLogDriver }

func (denyLogDriver) UploadLocalFile(context.Context, driver.LocalUploadRequest) (*driver.LocalUploadResult, error) {
	return nil, domain.Errorf(domain.CodePermissionDenied, "权限不足")
}

type cancelLogDriver struct{ cooldownLogDriver }

func (cancelLogDriver) UploadLocalFile(ctx context.Context, _ driver.LocalUploadRequest) (*driver.LocalUploadResult, error) {
	return nil, context.Canceled
}

func newLogCaptureService(t *testing.T, drv driver.Driver) (*Service, *bytes.Buffer) {
	t.Helper()
	exec := driverexec.New(uploadRootProvider{drv: drv}, nil)
	svc := NewService(exec, nil, nil, nil, nil, nil)
	buf := &bytes.Buffer{}
	svc.SetLogger(slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	return svc, buf
}

func uploadOnce(t *testing.T, svc *Service) error {
	t.Helper()
	_, err := svc.UploadLocal(context.Background(), 7, driver.LocalUploadRequest{
		LocalPath:      t.TempDir() + "/x.bin",
		FileName:       "cooldown_probe.bin",
		ParentID:       "root",
		ConflictPolicy: "overwrite",
	})
	return err
}

// 0.0.31：账号冷却必须记为 Info「上传暂缓」而不是 Warn「上传文件失败」。
func TestUploadLocalLogsCooldownAsInfo(t *testing.T) {
	svc, buf := newLogCaptureService(t, cooldownLogDriver{})
	if err := uploadOnce(t, svc); err == nil {
		t.Fatal("期望冷却错误")
	}
	out := buf.String()
	if !strings.Contains(out, "上传暂缓") {
		t.Fatalf("冷却应记录「上传暂缓」：%s", out)
	}
	if strings.Contains(out, "上传文件失败") {
		t.Fatalf("冷却不得记成「上传文件失败」：%s", out)
	}
	if !strings.Contains(out, "level=INFO") {
		t.Fatalf("冷却日志级别应为 INFO：%s", out)
	}
	if !strings.Contains(out, "retry_after_seconds=30") {
		t.Fatalf("应附重试秒数：%s", out)
	}
}

// 真失败必须 Warn 且带错误码，便于分组排查。
func TestUploadLocalLogsFailureWithCode(t *testing.T) {
	svc, buf := newLogCaptureService(t, denyLogDriver{})
	if err := uploadOnce(t, svc); err == nil {
		t.Fatal("期望权限错误")
	}
	out := buf.String()
	if !strings.Contains(out, "上传文件失败") || !strings.Contains(out, "level=WARN") {
		t.Fatalf("真失败应为 WARN「上传文件失败」：%s", out)
	}
	if !strings.Contains(out, "code=PERMISSION_DENIED") {
		t.Fatalf("应附错误码：%s", out)
	}
}

// 取消保持 Debug（不污染告警）。
func TestUploadLocalLogsCancelAsDebug(t *testing.T) {
	svc, buf := newLogCaptureService(t, cancelLogDriver{})
	_ = uploadOnce(t, svc)
	out := buf.String()
	if !strings.Contains(out, "上传文件已取消") {
		t.Fatalf("取消应记录「上传文件已取消」：%s", out)
	}
	if strings.Contains(out, "level=WARN") || strings.Contains(out, "level=INFO") {
		t.Fatalf("取消不应产生 WARN/INFO：%s", out)
	}
}
