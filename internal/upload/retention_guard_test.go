package upload

import (
	"context"
	"testing"
	"time"

	"litepan/internal/core/driverexec"
	"litepan/internal/domain"
	"litepan/internal/file"
)

// 缺陷复现（0.0.28 引入）：保留策略清理历史成功记录后，会削弱「删除云端批次根目录」
// 的完整性保护——BatchDelete 的 completeBatch 判定只看**当前内存中该批次的任务**
// 是否都被选中，记录被清理后"部分选择"会被误判为"整批选择"，从而删除云端批次根
// 目录（连同目录内所有文件）。
//
// 说明：本测试固化缺陷现状（characterization test）。修复后应改为断言"不触发根删除"。
func TestRetentionPurgeWeakensBatchRootGuard(t *testing.T) {
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
			TaskID:     id,
			BatchID:    batchID,
			BatchName:  "batch-root",
			AccountID:  7,
			Status:     StatusSuccess,
			TargetPath: "batch-root-id",
			CreatedAt:  now - float64(100-i), // t1 最老
			UpdatedAt:  now - float64(100-i),
			Result: map[string]any{
				"file_id":             id + "-file",
				"batch_root_id":       "batch-root-id",
				"batch_root_parent_id": "",
				"batch_root_owned":    true,
			},
		}}
	}
	m.mu.Unlock()

	// ① 保留策略清理最旧的 1 条（模拟 0.0.28 的自动清理）
	if removed := m.pruneRetainedTasks(RetentionConfig{Max: 2}, time.Now()); removed != 1 {
		t.Fatalf("保留策略应清理 1 条，实际 %d", removed)
	}
	if _, exists := m.tasks["t1"]; exists {
		t.Fatal("最旧记录应已被清理")
	}

	// ② 用户只选中"看得见的 2 条"并勾选删除云端文件 + 批次根目录
	result := m.BatchDelete(context.Background(), []string{"t2", "t3"}, true, true)
	if len(result.FailedTaskIDs) != 0 {
		t.Fatalf("批量删除失败：%+v", result.FailedMessages)
	}

	// ③ 现状：完整性判定被绕过 → 云端批次根目录被删除
	if len(drv.deleted) == 0 || len(drv.deleted[0]) != 1 || drv.deleted[0][0] != "batch-root-id" {
		t.Fatalf("预期复现缺陷（删除批次根目录），实际删除调用：%#v", drv.deleted)
	}
	t.Logf("缺陷复现：仅选中剩余 2/3 条记录即删除云端批次根目录 %v", drv.deleted)
}
