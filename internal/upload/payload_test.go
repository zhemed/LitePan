package upload

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

// 0.0.26：API/SSE 载荷瘦身——清理字段不外发、result 去重复键。
func TestSnapshotSlimsPayload(t *testing.T) {
	m := NewManager(Options{})
	m.mu.Lock()
	m.tasks["t1"] = &taskState{Task: Task{
		TaskID:           "t1",
		AccountID:        1,
		Status:           StatusSuccess,
		FileName:         "a.bin",
		CleanupLocalMode: "keep",
		CleanupLocalPath: "/data/uploads/tmp/a.bin",
		Result: map[string]any{
			"file_id":   "cloud-file-1",
			"file_name": "a.bin", // 与顶层重复 → 应剔除
			"size":      float64(1048576),
			"parent_id": "cloud-parent-1",
		},
	}}
	m.mu.Unlock()

	task, ok := m.Get(context.Background(), "t1")
	if !ok || task == nil {
		t.Fatal("task missing")
	}
	// 内部字段仍保留在结构体上供持久化使用，但 JSON 载荷不含它们。
	raw, err := json.Marshal(task)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	payload := string(raw)
	if strings.Contains(payload, "cleanup_local") {
		t.Fatalf("载荷不应包含 cleanup_local_*：%s", payload)
	}
	if _, exists := task.Result["file_name"]; exists {
		t.Fatal("result.file_name 应被剔除（与顶层重复）")
	}
	if _, exists := task.Result["size"]; exists {
		t.Fatal("result.size 应被剔除（与 top total_bytes 重复）")
	}
	if task.Result["file_id"] != "cloud-file-1" || task.Result["parent_id"] != "cloud-parent-1" {
		t.Fatalf("必要字段丢失：%+v", task.Result)
	}
}
