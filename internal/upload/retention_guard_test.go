package upload

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"litepan/internal/core/driverexec"
	"litepan/internal/domain"
	"litepan/internal/driver"
	"litepan/internal/file"
)

// 0.0.29 修复验证（原缺陷：0.0.28 保留策略清理记录后，仅选中剩余少数即可误删云端
// 批次根目录）。当前三重保护：①保留策略跳过 owned 批次根任务；②batch_task_total
// 完整性判据（缺失→保守拒绝）；③既有内存完整性检查 + 云端复核。
func TestRetentionPurgeWeakensBatchRootGuard(t *testing.T) {
	drv := &fakeDeleterDriver{listed: map[string][]domain.FileItem{
		"": {{ID: "batch-root-id", Name: "batch-root", IsDir: true}},
		// 目录内仍有文件（其任务记录可能已被清理）→ 绝不允许删目录
		"batch-root-id": {{ID: "leftover-file", Name: "leftover.bin"}},
	}}
	exec := driverexec.New(fakeProvider{drv: drv}, nil)
	files := file.NewService(exec, nil, nil, nil, nil, nil)
	m := NewManager(Options{Exec: exec, Files: files, DataDir: t.TempDir()})

	const batchID = "owned-batch"
	now := float64(time.Now().Unix())
	ids := []string{"t1", "t2", "t3"}
	m.mu.Lock()
	for i, id := range ids {
		m.tasks[id] = &taskState{Task: Task{
			TaskID:     id,
			BatchID:    batchID,
			BatchName:  "batch-root",
			AccountID:  7,
			Status:     StatusSuccess,
			TargetPath: "batch-root-id",
			CreatedAt:  now - float64(100-i), // t1 最老
			UpdatedAt:  now - float64(100-i),
			Result: map[string]any{
				"file_id":              id + "-file",
				"batch_root_id":        "batch-root-id",
				"batch_root_parent_id": "",
				"batch_root_owned":     true,
			},
		}}
	}
	m.mu.Unlock()

	// ① 修复后：owned 批次根任务不被保留策略清理
	if removed := m.pruneRetainedTasks(RetentionConfig{Max: 2}, time.Now()); removed != 0 {
		t.Fatalf("owned 批次根任务不应被清理，实际清理 %d 条", removed)
	}
	if _, exists := m.tasks["t1"]; !exists {
		t.Fatal("owned 批次根记录必须保留（否则完整性判据失去数据基础）")
	}

	// ② 用户只选中"看得见的 2 条"并勾选删除云端文件 + 批次根目录
	result := m.BatchDelete(context.Background(), []string{"t2", "t3"}, true, true)
	if len(result.FailedTaskIDs) != 0 {
		t.Fatalf("批量删除失败：%+v", result.FailedMessages)
	}

	// ③ 修复后：部分选择只删选中的文件，绝不删除云端批次根目录
	if deletedContains(drv.deleted, "batch-root-id") {
		t.Fatalf("部分选择不应删除云端批次根目录，实际删除调用：%#v", drv.deleted)
	}
	t.Logf("修复验证：仅选中 2/3 条记录 → 只删文件不删目录 ✓（删除调用 %#v）", drv.deleted)
}

// 正向用例：完整选择（选中数 == batch_task_total）时，根目录删除仍正常工作。
func TestBatchRootDeleteAllowedWhenCompleteSelection(t *testing.T) {
	drv := &fakeDeleterDriver{listed: map[string][]domain.FileItem{
		"": {{ID: "batch-root-id", Name: "batch-root", IsDir: true}},
	}}
	exec := driverexec.New(fakeProvider{drv: drv}, nil)
	files := file.NewService(exec, nil, nil, nil, nil, nil)
	m := NewManager(Options{Exec: exec, Files: files, DataDir: t.TempDir()})

	const batchID = "owned-batch"
	now := float64(time.Now().Unix())
	ids := []string{"t1", "t2", "t3"}
	m.mu.Lock()
	for i, id := range ids {
		m.tasks[id] = &taskState{Task: Task{
			TaskID: id, BatchID: batchID, BatchName: "batch-root", AccountID: 7,
			Status: StatusSuccess, TargetPath: "batch-root-id",
			CreatedAt: now - float64(100-i), UpdatedAt: now - float64(100-i),
			Result: map[string]any{
				"file_id":              id + "-file",
				"batch_root_id":        "batch-root-id",
				"batch_root_parent_id": "",
				"batch_root_owned":     true,
				"batch_task_total":     3,
			},
		}}
	}
	m.mu.Unlock()

	result := m.BatchDelete(context.Background(), ids, true, true)
	if len(result.FailedTaskIDs) != 0 {
		t.Fatalf("批量删除失败：%+v", result.FailedMessages)
	}
	if len(drv.deleted) != 1 || len(drv.deleted[0]) != 1 || drv.deleted[0][0] != "batch-root-id" {
		t.Fatalf("完整选择时应删除批次根目录，实际：%#v", drv.deleted)
	}
}

// 保守用例：历史批次缺 batch_task_total → 不删除根目录（仍逐文件删除）。
func TestBatchRootDeleteSkippedWhenTotalMissing(t *testing.T) {
	drv := &fakeDeleterDriver{listed: map[string][]domain.FileItem{
		"":              {{ID: "batch-root-id", Name: "batch-root", IsDir: true}},
		"batch-root-id": {{ID: "leftover-file", Name: "leftover.bin"}},
	}}
	exec := driverexec.New(fakeProvider{drv: drv}, nil)
	files := file.NewService(exec, nil, nil, nil, nil, nil)
	m := NewManager(Options{Exec: exec, Files: files, DataDir: t.TempDir()})

	now := float64(time.Now().Unix())
	ids := []string{"l1", "l2"}
	m.mu.Lock()
	for i, id := range ids {
		m.tasks[id] = &taskState{Task: Task{
			TaskID: id, BatchID: "legacy-batch", BatchName: "batch-root", AccountID: 7,
			Status: StatusSuccess, TargetPath: "batch-root-id",
			CreatedAt: now - float64(10-i), UpdatedAt: now - float64(10-i),
			Result: map[string]any{
				"file_id":              id + "-file",
				"batch_root_id":        "batch-root-id",
				"batch_root_parent_id": "",
				"batch_root_owned":     true, // 无 batch_task_total（历史数据）
			},
		}}
	}
	m.mu.Unlock()

	result := m.BatchDelete(context.Background(), ids, true, true)
	if len(result.FailedTaskIDs) != 0 {
		t.Fatalf("批量删除失败：%+v", result.FailedMessages)
	}
	if deletedContains(drv.deleted, "batch-root-id") {
		t.Fatalf("缺 batch_task_total 时不得删除根目录，实际：%#v", drv.deleted)
	}
}

// owned 批次根任务不被保留策略清理。
func TestRetentionSkipsOwnedBatchRootTasks(t *testing.T) {
	old := float64(time.Now().Add(-120 * 24 * time.Hour).Unix())
	tasks := []Task{
		{TaskID: "owned", Status: StatusSuccess, UpdatedAt: old,
			Result: map[string]any{"batch_root_owned": true, "batch_root_id": "r1"}},
		{TaskID: "plain", Status: StatusSuccess, UpdatedAt: old},
	}
	victims := selectRetentionVictims(tasks, RetentionConfig{Days: 30}, time.Now())
	if len(victims) != 1 || victims[0] != "plain" {
		t.Fatalf("应仅清理普通记录，实际 %v", victims)
	}
}

// deletedContains 判断删除调用集合中是否包含指定目录/文件 ID。
func deletedContains(calls [][]string, target string) bool {
	for _, call := range calls {
		for _, id := range call {
			if id == target {
				return true
			}
		}
	}
	return false
}

// ---- 0.0.30：冷却等待不得产生孤儿任务，也不得覆盖暂停 ----

type cooldownThenSuccessDriver struct {
	calls int32
}

func (d *cooldownThenSuccessDriver) Config() driver.Config      { return driver.Config{Name: "mock"} }
func (d *cooldownThenSuccessDriver) GetAddition() any           { return &struct{}{} }
func (d *cooldownThenSuccessDriver) Init(context.Context) error { return nil }
func (d *cooldownThenSuccessDriver) Drop(context.Context) error { return nil }
func (d *cooldownThenSuccessDriver) Ping(context.Context) error { return nil }
func (d *cooldownThenSuccessDriver) ListFiles(context.Context, string) ([]domain.FileItem, error) {
	return nil, nil
}

func (d *cooldownThenSuccessDriver) UploadLocalFile(context.Context, driver.LocalUploadRequest) (*driver.LocalUploadResult, error) {
	if atomic.AddInt32(&d.calls, 1) == 1 {
		return nil, driverexec.CooldownError(0) // 首次返回冷却（1 秒等待）
	}
	return &driver.LocalUploadResult{FileID: "fid", FileName: "a.bin", Size: 4, Message: "上传成功"}, nil
}

func newCooldownTestManager(t *testing.T, drv driver.Driver) (*Manager, string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "a.bin")
	if err := os.WriteFile(path, []byte("data"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	exec := driverexec.New(fakeProvider{drv: drv}, nil)
	files := file.NewService(exec, nil, nil, nil, nil, nil)
	m := NewManager(Options{Exec: exec, Files: files, DataDir: t.TempDir()})
	m.mu.Lock()
	state := &taskState{Task: Task{
		TaskID: "c1", AccountID: 1, Status: StatusPending, TotalBytes: 4,
		TargetPath: "root", FileName: "a.bin",
	}}
	state.localPath = path
	m.tasks["c1"] = state
	m.mu.Unlock()
	return m, path
}

// 冷却等待结束后必须返回 requeue=true（否则 runTask 协程退出 → 孤儿任务）。
func TestCooldownReturnsRequeueTrue(t *testing.T) {
	drv := &cooldownThenSuccessDriver{}
	m, _ := newCooldownTestManager(t, drv)

	requeue := m.executeUpload(context.Background(), "c1")
	if !requeue {
		t.Fatal("冷却等待结束后应返回 requeue=true（0.0.30 修复点）")
	}
	m.mu.Lock()
	status, msg, priority := m.tasks["c1"].Status, m.tasks["c1"].Message, m.tasks["c1"].resumePriority
	m.mu.Unlock()
	if status != StatusPending {
		t.Fatalf("冷却后应回到 pending，实际 %s", status)
	}
	if !strings.Contains(msg, "冷却") {
		t.Fatalf("提示应说明冷却等待，实际 %q", msg)
	}
	if !priority {
		t.Fatal("冷却任务应置顶（resumePriority）")
	}
}

// 任务处于暂停态时，冷却分支不得覆盖其状态，且不重入队列。
func TestCooldownDoesNotOverridePause(t *testing.T) {
	drv := &cooldownThenSuccessDriver{}
	m, _ := newCooldownTestManager(t, drv)
	m.mu.Lock()
	m.tasks["c1"].Status = StatusPaused
	m.tasks["c1"].cancelMode = "pause"
	m.tasks["c1"].Message = "上传已暂停"
	m.mu.Unlock()

	if m.canCooldownWait("c1") {
		t.Fatal("暂停态不应允许进入冷却等待")
	}
	requeue := m.executeUpload(context.Background(), "c1")
	if requeue {
		t.Fatal("暂停态不得重入队列")
	}
	m.mu.Lock()
	status, msg := m.tasks["c1"].Status, m.tasks["c1"].Message
	m.mu.Unlock()
	if status != StatusPaused || msg != "上传已暂停" {
		t.Fatalf("暂停态被冷却分支覆盖：status=%s msg=%q", status, msg)
	}
}

// 取消/停止态同样不可进入冷却等待。
func TestCanCooldownWaitRejectsCanceledAndStopped(t *testing.T) {
	m, _ := newCooldownTestManager(t, &cooldownThenSuccessDriver{})
	m.mu.Lock()
	m.tasks["c1"].Status = StatusCanceled
	m.mu.Unlock()
	if m.canCooldownWait("c1") {
		t.Fatal("取消态不应允许冷却等待")
	}
	m.mu.Lock()
	m.tasks["c1"].Status = StatusPending
	m.tasks["c1"].cancelMode = ""
	m.stopping = true
	m.mu.Unlock()
	if m.canCooldownWait("c1") {
		t.Fatal("停止中不应允许冷却等待")
	}
}
