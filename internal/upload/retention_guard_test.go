package upload

import (
	"context"
	"testing"
	"time"

	"litepan/internal/core/driverexec"
	"litepan/internal/domain"
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
