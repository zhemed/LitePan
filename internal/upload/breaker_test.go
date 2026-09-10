package upload

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"litepan/internal/core/driverexec"
	"litepan/internal/domain"
	"litepan/internal/driver"
	"litepan/internal/file"
)

func TestFailureSignatureGroupsSameCause(t *testing.T) {
	a := domain.Errorf(domain.CodeDriverError, "该账号网络异常，约 30 秒后自动重试")
	b := domain.Errorf(domain.CodeDriverError, "该账号网络异常，约 29 秒后自动重试")
	if failureSignature(a) == failureSignature(b) {
		t.Fatal("不同文案应产生不同签名（秒数差异）")
	}
	c := domain.Errorf(domain.CodeAuthExpired, "账号认证已失效")
	if failureSignature(c) == failureSignature(a) {
		t.Fatal("不同错误码签名必须不同")
	}
}

func TestSystemicFailureSelection(t *testing.T) {
	if !systemicFailure(domain.Errorf(domain.CodeDriverError, "网络异常")) {
		t.Fatal("driver error 应为系统级")
	}
	if !systemicFailure(domain.Errorf(domain.CodeAuthExpired, "失效")) {
		t.Fatal("auth expired 应为系统级")
	}
	if systemicFailure(domain.Errorf(domain.CodeValidation, "文件名为空")) {
		t.Fatal("参数错误不应触发熔断")
	}
	if systemicFailure(domain.Errorf(domain.CodeNotFound, "文件不存在")) {
		t.Fatal("文件不存在不应触发熔断")
	}
}

func TestSystemFailureLabel(t *testing.T) {
	if got := systemFailureLabel(domain.Errorf(domain.CodeAuthExpired, "x")); got != "账号认证已失效" {
		t.Fatalf("label=%q", got)
	}
	if got := systemFailureLabel(domain.Errorf(domain.CodeDriverError, "DRIVER_ERROR: 网络异常")); got == "" {
		t.Fatal("label 不应为空")
	}
}

type cooldownUploadDriver struct{ calls int32 }

func (d *cooldownUploadDriver) Config() driver.Config      { return driver.Config{Name: "mock"} }
func (d *cooldownUploadDriver) GetAddition() any           { return &struct{}{} }
func (d *cooldownUploadDriver) Init(context.Context) error { return nil }
func (d *cooldownUploadDriver) Drop(context.Context) error { return nil }
func (d *cooldownUploadDriver) Ping(context.Context) error { return nil }
func (d *cooldownUploadDriver) ListFiles(context.Context, string) ([]domain.FileItem, error) {
	return nil, nil
}

func (d *cooldownUploadDriver) UploadLocalFile(context.Context, driver.LocalUploadRequest) (*driver.LocalUploadResult, error) {
	atomic.AddInt32(&d.calls, 1)
	return nil, driverexec.CooldownError(0)
}

// 0.0.24：冷却错误不得把任务判死——退回 pending（置顶）并等待冷却结束。
func TestCooldownErrorReturnsTaskToPending(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.bin")
	if err := os.WriteFile(path, []byte("data"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	drv := &cooldownUploadDriver{}
	exec := driverexec.New(fakeProvider{drv: drv}, nil)
	files := file.NewService(exec, nil, nil, nil, nil, nil)
	m := NewManager(Options{Exec: exec, Files: files, DataDir: t.TempDir()})

	m.mu.Lock()
	state := &taskState{
		Task: Task{
			TaskID:        "t1",
			BatchID:       "batch-1",
			AccountID:     1,
			Status:        StatusPending,
			TotalBytes:    4,
			TargetPath:    "root",
			FileName:      "a.bin",
			UploadedBytes: 1,
		},
	}
	state.localPath = path
	state.resumeData = map[string]any{"upload_id": "keep-me"}
	m.tasks["t1"] = state
	m.mu.Unlock()

	m.executeUpload(context.Background(), "t1")

	m.mu.Lock()
	st := m.tasks["t1"]
	status, msg, errText := st.Status, st.Message, st.Error
	resumeKept := st.resumeData != nil
	uploaded := st.UploadedBytes
	m.mu.Unlock()

	if status != StatusPending {
		t.Fatalf("冷却后任务应退回 pending，实际 %s（msg=%q err=%q）", status, msg, errText)
	}
	if errText != "" {
		t.Fatalf("冷却不应留下失败原因：%q", errText)
	}
	if !strings.Contains(msg, "冷却") {
		t.Fatalf("提示应说明冷却等待，实际 %q", msg)
	}
	if !resumeKept || uploaded == 0 {
		t.Fatalf("冷却等待必须保留进度/resumeData（resumeKept=%v uploaded=%d）", resumeKept, uploaded)
	}
	if drv.calls != 1 {
		t.Fatalf("驱动应被调用一次，实际 %d", drv.calls)
	}
}

// 熔断安全网：同批次连续同因系统级失败达阈值 → 剩余 pending 自动暂停。
func TestBatchBreakerPausesRemainingPending(t *testing.T) {
	m := NewManager(Options{})
	const batchID = "batch-x"
	m.mu.Lock()
	for i := 0; i < 8; i++ {
		id := fmt.Sprintf("task-%d", i)
		m.tasks[id] = &taskState{Task: Task{TaskID: id, BatchID: batchID, AccountID: 1, Status: StatusPending}}
	}
	m.mu.Unlock()

	err := domain.Errorf(domain.CodeAuthExpired, "账号认证已失效")
	for i := 0; i < batchBreakerThreshold; i++ {
		m.failTask(fmt.Sprintf("task-%d", i), err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	paused := 0
	for _, st := range m.tasks {
		if st.Status == StatusPaused {
			paused++
			if !strings.Contains(st.Message, "批次已自动暂停") {
				t.Fatalf("暂停任务缺少熔断提示：%q", st.Message)
			}
		}
	}
	if paused != 3 {
		t.Fatalf("剩余 3 个 pending 应被暂停，实际 %d", paused)
	}
	if len(m.batchFailures) != 0 {
		t.Fatalf("熔断后计数应清零，实际 %+v", m.batchFailures)
	}
}

// 文件级错误不触发熔断。
func TestBatchBreakerIgnoresFileLevelErrors(t *testing.T) {
	m := NewManager(Options{})
	m.mu.Lock()
	for i := 0; i < 6; i++ {
		id := fmt.Sprintf("f-%d", i)
		m.tasks[id] = &taskState{Task: Task{TaskID: id, BatchID: "batch-y", AccountID: 1, Status: StatusPending}}
	}
	m.mu.Unlock()
	err := domain.Errorf(domain.CodeValidation, "文件名为空")
	for i := 0; i < 6; i++ {
		m.failTask(fmt.Sprintf("f-%d", i), err)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, st := range m.tasks {
		if st.Status == StatusPaused {
			t.Fatal("文件级错误不应触发熔断")
		}
	}
}

// 0.0.25：批量恢复端点——去重、缺失登记、本地文件缺失时的确定性短路。
func TestBatchResumeHandlesBatch(t *testing.T) {
	m := NewManager(Options{})
	ids := []string{"r-0", "r-1", "r-2"}
	m.mu.Lock()
	for _, id := range ids {
		state := &taskState{Task: Task{
			TaskID:     id,
			AccountID:  1,
			Status:     StatusPaused,
			SourceType: SourceTypeServerLocal,
		}}
		state.localPath = "" // 服务器本地文件缺失 → Resume 走确定性失败短路
		m.tasks[id] = state
	}
	m.mu.Unlock()

	result := m.BatchResume(context.Background(), []string{"r-0", "r-1", "r-1", "r-2", "", "nope"})
	if len(result.UpdatedTaskIDs) != 3 {
		t.Fatalf("updated=%v", result.UpdatedTaskIDs)
	}
	if len(result.MissingTaskIDs) != 1 || result.MissingTaskIDs[0] != "nope" {
		t.Fatalf("missing=%v", result.MissingTaskIDs)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, id := range ids {
		if m.tasks[id].Status == StatusPaused {
			t.Fatalf("任务 %s 仍处于 paused（批量恢复未生效）", id)
		}
	}
}
