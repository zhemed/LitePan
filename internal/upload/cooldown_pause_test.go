package upload

import (
	"context"
	"strings"
	"sync"
	"testing"

	"litepan/internal/domain"
)

// recordingUploadRepo 记录每次 Upsert 的落库内容，用于断言「内存态 == 持久化态」。
type recordingUploadRepo struct {
	mu    sync.Mutex
	rows  map[string]domain.UploadTaskRecord
	calls int
}

func newRecordingRepo() *recordingUploadRepo {
	return &recordingUploadRepo{rows: map[string]domain.UploadTaskRecord{}}
}

func (r *recordingUploadRepo) Upsert(_ context.Context, rec *domain.UploadTaskRecord) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls++
	r.rows[rec.TaskID] = *rec
	return nil
}

func (r *recordingUploadRepo) Delete(_ context.Context, taskID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.rows, taskID)
	return nil
}

func (r *recordingUploadRepo) List(context.Context) ([]*domain.UploadTaskRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]*domain.UploadTaskRecord, 0, len(r.rows))
	for _, row := range r.rows {
		copy := row
		out = append(out, &copy)
	}
	return out, nil
}

func (r *recordingUploadRepo) last(taskID string) (domain.UploadTaskRecord, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	row, ok := r.rows[taskID]
	return row, ok
}

// 0.0.34：冷却重入队必须「暂停/取消优先」——守卫拒绝时字段零变更。
func TestBeginCooldownWaitRejectsNonRunnableStates(t *testing.T) {
	cases := []struct {
		name string
		prep func(st *taskState)
	}{
		{"已暂停", func(st *taskState) { st.Status = StatusPaused; st.Message = "上传已暂停" }},
		{"已取消", func(st *taskState) { st.Status = StatusCanceled; st.Message = "上传任务已取消" }},
		{"暂停标记", func(st *taskState) { st.Status = StatusRunning; st.cancelMode = "pause" }},
		{"终态成功", func(st *taskState) { st.Status = StatusSuccess; st.Message = "上传成功" }},
	}
	for _, tc := range cases {
		m := NewManager(Options{})
		st := &taskState{Task: Task{TaskID: "t1", AccountID: 1, Status: StatusRunning, Message: "正在上传到网盘"}}
		tc.prep(st)
		beforeStatus, beforeMessage := st.Status, st.Message
		beforePriority := st.resumePriority
		m.tasks["t1"] = st

		if m.beginCooldownWait("t1", 30) {
			t.Fatalf("%s：守卫应拒绝冷却重入队", tc.name)
		}
		if st.Status != beforeStatus {
			t.Fatalf("%s：状态被改写（%s → %s）", tc.name, beforeStatus, st.Status)
		}
		if st.Message != beforeMessage {
			t.Fatalf("%s：消息被改写（%q → %q）", tc.name, beforeMessage, st.Message)
		}
		if st.resumePriority != beforePriority {
			t.Fatalf("%s：resumePriority 被改写", tc.name)
		}
		if strings.Contains(st.Message, "冷却") {
			t.Fatalf("%s：不得写入冷却文案", tc.name)
		}
	}
}

// 不存在 / 停止中同样拒绝，且不 panic。
func TestBeginCooldownWaitRejectsMissingAndStopping(t *testing.T) {
	m := NewManager(Options{})
	if m.beginCooldownWait("missing", 30) {
		t.Fatal("任务不存在应拒绝")
	}
	m.tasks["t1"] = &taskState{Task: Task{TaskID: "t1", Status: StatusPending}}
	m.mu.Lock()
	m.stopping = true
	m.mu.Unlock()
	if m.beginCooldownWait("t1", 30) {
		t.Fatal("停止中应拒绝")
	}
}

// 可重试状态必须放行，并写入冷却文案 + 置顶优先级。
func TestBeginCooldownWaitAllowsRunnableAndRecordsMessage(t *testing.T) {
	for _, status := range []string{StatusRunning, StatusPending} {
		m := NewManager(Options{})
		st := &taskState{Task: Task{TaskID: "t1", AccountID: 2, Status: status}}
		m.tasks["t1"] = st
		if !m.beginCooldownWait("t1", 30) {
			t.Fatalf("状态 %s 应放行冷却等待", status)
		}
		if st.Status != StatusPending {
			t.Fatalf("状态 %s 应写回 pending，实际 %s", status, st.Status)
		}
		if !strings.Contains(st.Message, "账号网络冷却中，30 秒后自动重试") {
			t.Fatalf("缺少冷却文案：%q", st.Message)
		}
		if !st.resumePriority {
			t.Fatal("冷却等待应置顶优先级（resumePriority）")
		}
	}
}

// 竞态核心断言：冷却等待期间暂停 → 不得被改回 pending；等待结束不得重入队。
func TestPauseDuringCooldownWaitWinsAndDoesNotRequeue(t *testing.T) {
	m := NewManager(Options{})
	st := &taskState{Task: Task{TaskID: "t1", AccountID: 2, Status: StatusRunning}}
	m.tasks["t1"] = st

	if !m.beginCooldownWait("t1", 30) {
		t.Fatal("首次应放行")
	}
	// 模拟用户点暂停（与 0.0.30 起的 pause() 语义一致）。
	m.mu.Lock()
	st.cancelMode = "pause"
	st.Status = StatusPaused
	st.Message = "上传已暂停"
	m.mu.Unlock()

	if m.beginCooldownWait("t1", 30) {
		t.Fatal("暂停后不得再次进入冷却等待")
	}
	if st.Status != StatusPaused || st.Message != "上传已暂停" {
		t.Fatalf("暂停状态被改写：status=%s message=%q", st.Status, st.Message)
	}
	if m.canCooldownWait("t1") {
		t.Fatal("等待结束判定必须为 false（不得重入队）")
	}
}

// 顺序反转（先 pause 再冷却）与顺序正转（先冷却再 pause）都必须以暂停收尾。
func TestCooldownWaitAndPauseOrderingBothEndPaused(t *testing.T) {
	for _, pauseFirst := range []bool{true, false} {
		m := NewManager(Options{})
		st := &taskState{Task: Task{TaskID: "t1", AccountID: 2, Status: StatusRunning}}
		m.tasks["t1"] = st

		pause := func() {
			m.mu.Lock()
			st.cancelMode = "pause"
			st.Status = StatusPaused
			st.Message = "上传已暂停"
			m.mu.Unlock()
		}
		if pauseFirst {
			pause()
			m.beginCooldownWait("t1", 30)
		} else {
			m.beginCooldownWait("t1", 30)
			pause()
		}
		if st.Status != StatusPaused {
			t.Fatalf("pauseFirst=%v：最终状态应为 paused，实际 %s", pauseFirst, st.Status)
		}
		if strings.Contains(st.Message, "冷却") {
			t.Fatalf("pauseFirst=%v：暂停后不得保留冷却文案", pauseFirst)
		}
	}
}

// 回归证据：旧的两步式（先 canCooldownWait 判定、再 patch 无条件写回）在
// 「判定通过后、写入前」落入暂停时会把暂停改写回 pending——这正是生产上
// 出现「库内 pending / 内存 paused」分叉的来源。新守卫在同样交错下保持暂停。
func TestLegacyTwoStepOverwritesPauseButGuardDoesNot(t *testing.T) {
	m := NewManager(Options{})
	st := &taskState{Task: Task{TaskID: "t1", AccountID: 2, Status: StatusRunning}}
	m.tasks["t1"] = st

	if !m.canCooldownWait("t1") {
		t.Fatal("旧实现第一步判定应通过")
	}
	// 交错点：判定之后、写入之前，用户点了暂停。
	m.mu.Lock()
	st.cancelMode = "pause"
	st.Status = StatusPaused
	st.Message = "上传已暂停"
	m.mu.Unlock()
	// 旧实现第二步：无条件写回 pending（复现缺陷）。
	m.patch("t1", func(s *taskState) {
		s.Status = StatusPending
		s.Message = "账号网络冷却中，30 秒后自动重试"
		s.resumePriority = true
	})
	if st.Status != StatusPending {
		t.Fatal("旧两步式应复现「暂停被覆盖」的分叉")
	}

	// 新实现：同一交错下守卫拒绝，暂停状态保持不变。
	st.Status = StatusPaused
	st.Message = "上传已暂停"
	st.cancelMode = "pause"
	if m.beginCooldownWait("t1", 30) {
		t.Fatal("新守卫必须拒绝（暂停优先）")
	}
	if st.Status != StatusPaused || st.Message != "上传已暂停" {
		t.Fatalf("新守卫不得改写暂停：status=%s message=%q", st.Status, st.Message)
	}
}

// 持久化：patch / beginCooldownWait 落库内容必须等于该次迁移的状态（值快照）。
func TestCooldownWaitPersistsValueSnapshot(t *testing.T) {
	repo := newRecordingRepo()
	m := NewManager(Options{Repo: repo})
	m.tasks["t1"] = &taskState{Task: Task{TaskID: "t1", Status: StatusRunning, Message: "正在上传"}}

	if !m.beginCooldownWait("t1", 30) {
		t.Fatal("应放行")
	}
	row, ok := repo.last("t1")
	if !ok {
		t.Fatal("冷却等待应持久化一行")
	}
	if row.Status != StatusPending || !strings.Contains(row.Message, "账号网络冷却中") {
		t.Fatalf("落库应为冷却 pending，实际 status=%s message=%q", row.Status, row.Message)
	}

	m.patch("t1", func(st *taskState) {
		st.Status = StatusPaused
		st.Message = "上传已暂停"
	})
	row, _ = repo.last("t1")
	if row.Status != StatusPaused || row.Message != "上传已暂停" {
		t.Fatalf("暂停必须落库为最终态，实际 status=%s message=%q", row.Status, row.Message)
	}
}

// 并发烟雾：冷却守卫与暂停交错执行后，内存与落库必须一致（-race 下运行）。
func TestCooldownPauseConcurrentConsistency(t *testing.T) {
	repo := newRecordingRepo()
	m := NewManager(Options{Repo: repo})
	st := &taskState{Task: Task{TaskID: "t1", AccountID: 2, Status: StatusRunning}}
	m.tasks["t1"] = st

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < 200; i++ {
			m.beginCooldownWait("t1", 30)
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < 200; i++ {
			m.mu.Lock()
			st.cancelMode = "pause"
			st.Status = StatusPaused
			st.Message = "上传已暂停"
			m.mu.Unlock()
			m.patch("t1", func(s *taskState) {
				s.Status = StatusPaused
				s.Message = "上传已暂停"
			})
		}
	}()
	wg.Wait()

	m.mu.Lock()
	memStatus, memMessage := st.Status, st.Message
	m.mu.Unlock()
	row, ok := repo.last("t1")
	if !ok {
		t.Fatal("应有落库行")
	}
	if row.Status != memStatus || row.Message != memMessage {
		t.Fatalf("内存态与落库态不一致：mem=(%s,%q) db=(%s,%q)", memStatus, memMessage, row.Status, row.Message)
	}
	if memStatus != StatusPaused {
		t.Fatalf("暂停必须最终获胜，实际 %s", memStatus)
	}
}
