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

// recoverableDriver：前 remaining 次返回冷却，之后成功（模拟冷却窗口结束后的恢复）。
type recoverableDriver struct {
	cooldownLogDriver
	remaining int
}

func (d *recoverableDriver) UploadLocalFile(context.Context, driver.LocalUploadRequest) (*driver.LocalUploadResult, error) {
	if d.remaining > 0 {
		d.remaining--
		return nil, driverexec.CooldownError(30 * time.Second)
	}
	return &driver.LocalUploadResult{FileID: "f1", FileName: "cooldown_probe.bin", ParentID: "root"}, nil
}

// 0.0.32：同一账号同一冷却窗口内不得重复刷 INFO——
// 生产实测「点一次暂停」在 0.35 秒内刷出 183 条同构日志，必须收敛为 1 条。
func TestUploadLocalSuppressesCooldownLogBurst(t *testing.T) {
	const burst = 20
	svc, buf := newLogCaptureService(t, cooldownLogDriver{})
	for i := 0; i < burst; i++ {
		if err := uploadOnce(t, svc); err == nil {
			t.Fatal("期望冷却错误")
		}
	}
	out := buf.String()
	if got := strings.Count(out, "，稍后自动重试"); got != 1 {
		t.Fatalf("冷却窗口内 INFO 只应 1 条，实际 %d 条：%s", got, out)
	}
	if got := strings.Count(out, "同一窗口重复命中，已抑制"); got != burst-1 {
		t.Fatalf("其余 %d 条应降为 Debug，实际 %d 条", burst-1, got)
	}
	if strings.Contains(out, "level=WARN") {
		t.Fatalf("冷却不得产生 WARN：%s", out)
	}
}

// 冷却窗口结束后的首次成功必须补一条汇总，保留「影响多少任务」的运维信息。
func TestUploadLocalLogsCooldownRecoverySummary(t *testing.T) {
	const burst = 6
	drv := &recoverableDriver{remaining: burst}
	svc, buf := newLogCaptureService(t, drv)
	for i := 0; i < burst; i++ {
		_ = uploadOnce(t, svc)
	}
	if err := uploadOnce(t, svc); err != nil {
		t.Fatalf("冷却结束后应上传成功，实际 %v", err)
	}
	out := buf.String()
	if got := strings.Count(out, "，稍后自动重试"); got != 1 {
		t.Fatalf("窗口内 INFO 只应 1 条，实际 %d：%s", got, out)
	}
	if got := strings.Count(out, "账号网络冷却已恢复"); got != 1 {
		t.Fatalf("应恰好 1 条恢复汇总，实际 %d：%s", got, out)
	}
	if !strings.Contains(out, "deferred_tasks=6") {
		t.Fatalf("汇总应带本窗口暂缓任务数：%s", out)
	}
}
