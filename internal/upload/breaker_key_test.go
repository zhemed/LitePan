package upload

import (
	"fmt"
	"strings"
	"testing"

	"litepan/internal/domain"
)

// 0.0.35：熔断分组键三态——有批次 id / 仅账号+目标目录 / 仅账号。
func TestBatchKeyOfFallbacks(t *testing.T) {
	withBatch := &taskState{Task: Task{TaskID: "a", AccountID: 2, BatchID: "auto-2-1", TargetPath: "/root/x"}}
	if got := batchKeyOf(withBatch); got != "batch:auto-2-1" {
		t.Fatalf("有批次 id 应优先用批次键，实际 %q", got)
	}
	noBatch := &taskState{Task: Task{TaskID: "b", AccountID: 2, TargetPath: "/root/x"}}
	if got := batchKeyOf(noBatch); got != "acct:2|target:/root/x" {
		t.Fatalf("无批次 id 应退化为账号+目标目录，实际 %q", got)
	}
	bare := &taskState{Task: Task{TaskID: "c", AccountID: 2}}
	if got := batchKeyOf(bare); got != "acct:2" {
		t.Fatalf("既无批次也无目标应退化为账号级，实际 %q", got)
	}
	if batchKeyOf(nil) != "" {
		t.Fatal("nil 应返回空键")
	}
}

// 无 batch_id 的自动化批次（生产现状）也必须受熔断保护：
// 修复前 batch_id 为空直接 return，剩余 pending 不会被暂停。
func TestBreakerTripsForBatchlessTasksWithSameTarget(t *testing.T) {
	m := NewManager(Options{})
	const target = "/备份/pve_backup"
	m.mu.Lock()
	for i := 0; i < 8; i++ {
		id := fmt.Sprintf("task-%d", i)
		m.tasks[id] = &taskState{Task: Task{
			TaskID: id, AccountID: 2, Status: StatusPending, TargetPath: target,
		}}
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
		t.Fatalf("无 batch_id 批次剩余 3 个 pending 应被暂停，实际 %d", paused)
	}
	if len(m.batchFailures) != 0 {
		t.Fatalf("熔断后计数应清零，实际 %+v", m.batchFailures)
	}
}

// 退化分组不得跨目标目录误伤：不同目录的无批次任务互不牵连。
func TestBatchlessBreakerDoesNotCrossTargets(t *testing.T) {
	m := NewManager(Options{})
	m.mu.Lock()
	for i := 0; i < 6; i++ {
		id := fmt.Sprintf("a-%d", i)
		m.tasks[id] = &taskState{Task: Task{TaskID: id, AccountID: 2, Status: StatusPending, TargetPath: "/dirA"}}
	}
	for i := 0; i < 3; i++ {
		id := fmt.Sprintf("b-%d", i)
		m.tasks[id] = &taskState{Task: Task{TaskID: id, AccountID: 2, Status: StatusPending, TargetPath: "/dirB"}}
	}
	m.mu.Unlock()

	err := domain.Errorf(domain.CodeAuthExpired, "账号认证已失效")
	for i := 0; i < batchBreakerThreshold; i++ {
		m.failTask(fmt.Sprintf("a-%d", i), err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	for i := 0; i < 3; i++ {
		st := m.tasks[fmt.Sprintf("b-%d", i)]
		if st.Status != StatusPending {
			t.Fatalf("/dirB 的任务不应被 /dirA 的熔断牵连：task b-%d = %s", i, st.Status)
		}
	}
	if m.tasks["a-5"].Status != StatusPaused {
		t.Fatalf("/dirA 剩余任务应被暂停，实际 %s", m.tasks["a-5"].Status)
	}
}

// 有 batch_id 时仍按批次分组：同账号不同批次不得互相牵连（回归既有语义）。
func TestBatchBreakerStaysScopedToBatchID(t *testing.T) {
	m := NewManager(Options{})
	m.mu.Lock()
	for i := 0; i < 6; i++ {
		id := fmt.Sprintf("x-%d", i)
		m.tasks[id] = &taskState{Task: Task{TaskID: id, AccountID: 2, Status: StatusPending, BatchID: "batch-x", TargetPath: "/dirA"}}
	}
	m.tasks["y-0"] = &taskState{Task: Task{TaskID: "y-0", AccountID: 2, Status: StatusPending, BatchID: "batch-y", TargetPath: "/dirA"}}
	m.mu.Unlock()

	err := domain.Errorf(domain.CodeAuthExpired, "账号认证已失效")
	for i := 0; i < batchBreakerThreshold; i++ {
		m.failTask(fmt.Sprintf("x-%d", i), err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if m.tasks["y-0"].Status != StatusPending {
		t.Fatalf("其它批次不得被牵连，实际 %s", m.tasks["y-0"].Status)
	}
	if m.tasks["x-5"].Status != StatusPaused {
		t.Fatalf("同批次剩余任务应被暂停，实际 %s", m.tasks["x-5"].Status)
	}
}

