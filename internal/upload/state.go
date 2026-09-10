package upload

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sort"
	"time"

	"litepan/internal/domain"
	"litepan/pkg/timeutil"
)

func markMissingLocalFileFailed(st *taskState) {
	st.Status = StatusFailed
	st.Message = "上传失败"
	st.Error = "本地临时文件不存在，无法继续上传"
	st.UpdatedAt = timeutil.UnixFloat(time.Now())
}

func (m *Manager) patch(taskID string, fn func(*taskState)) {
	m.mu.Lock()
	st, ok := m.tasks[taskID]
	if !ok {
		m.mu.Unlock()
		return
	}
	fn(st)
	st.UpdatedAt = timeutil.UnixFloat(time.Now())
	// 0.0.34：持久化改用值快照——此前传活体指针，落库内容会被并发迁移改写，
	// 产生「内存已暂停、库内仍 pending」的分叉（重启恢复会据此自行续传）。
	snap := *st
	m.mu.Unlock()
	m.persistStateSnapshot(taskID, &snap)
	m.broadcast(taskID)
}

// persistStateSnapshot 按新鲜度落库：串行化写入，并丢弃过期快照。
//
// 为什么需要：持久化在 m.mu 之外执行，两次并发迁移的写库顺序可能倒置，
// 过期快照（例如冷却写回 pending）会覆盖较新的状态（暂停），造成 DB 与内存分叉，
// 而启动恢复会把 DB 的 pending 行重新入队 ⇒ 已暂停任务在重启后自行续传。
func (m *Manager) persistStateSnapshot(taskID string, snap *taskState) {
	if m == nil || snap == nil || m.repo == nil {
		return
	}
	m.persistMu.Lock()
	defer m.persistMu.Unlock()
	m.mu.Lock()
	cur, ok := m.tasks[taskID]
	stale := ok && cur.UpdatedAt > snap.UpdatedAt
	m.mu.Unlock()
	if stale {
		return
	}
	_ = m.persistTask(snap)
}

// beginCooldownWait 原子进入「账号网络冷却等待」：在锁内一次性判定任务此刻是否
// 仍可重试，满足才写回 pending + 冷却文案 + resumePriority，并返回 true。
//
// 返回 false 表示任务已暂停/取消/停止/不存在——调用方必须保持其现有状态
// （暂停优先），不得把任务改回 pending。0.0.34 修复：先前 canCooldownWait()
// 与 patch() 之间存在窗口，期间落地的 pause() 会被随后的 patch 覆盖。
func (m *Manager) beginCooldownWait(taskID string, seconds int) bool {
	m.mu.Lock()
	st, ok := m.tasks[taskID]
	if !ok || m.stopping {
		m.mu.Unlock()
		return false
	}
	if st.cancelMode == "pause" || st.Status == StatusPaused || st.Status == StatusCanceled {
		m.mu.Unlock()
		return false
	}
	if st.Status != StatusRunning && st.Status != StatusPending {
		m.mu.Unlock()
		return false
	}
	st.Status = StatusPending
	st.SpeedBytesPerSecond = 0
	st.Message = fmt.Sprintf("账号网络冷却中，%d 秒后自动重试", seconds)
	st.Error = ""
	st.resumePriority = true
	st.UpdatedAt = timeutil.UnixFloat(time.Now())
	snap := *st
	m.runCond.Broadcast()
	m.mu.Unlock()
	m.persistStateSnapshot(taskID, &snap)
	m.broadcast(taskID)
	return true
}

func (m *Manager) failTask(taskID string, err error) {
	errMsg := ""
	if err != nil {
		errMsg = err.Error()
	}
	m.patch(taskID, func(st *taskState) {
		st.Status = StatusFailed
		st.SpeedBytesPerSecond = 0
		st.Message = "上传失败"
		st.Error = translateError(errMsg)
	})
	// 熔断安全网：同批次连续同因系统级失败 → 自动暂停批次剩余任务（0.0.24）。
	m.observeBatchFailure(taskID, err)
}

func (m *Manager) snapshot(st *taskState) *Task {
	t := st.Task
	t.Result = cloneMap(st.Result)
	// 瘦身：result 内 file_name/size 与顶层 file_name/total_bytes 重复，
	// 前端均有回退取值（result?.file_name || file_name），API/SSE 载荷不再重复传输。
	if t.Result != nil {
		delete(t.Result, "file_name")
		delete(t.Result, "size")
	}
	return &t
}

const deleteStopTimeout = 5 * time.Second

// 删除前等待任务退出，避免后台仍在传输。
func (m *Manager) stopTaskForDelete(ctx context.Context, taskID string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	m.mu.Lock()
	st, ok := m.tasks[taskID]
	if !ok {
		m.mu.Unlock()
		return nil
	}
	active := st.Status == StatusPending || st.Status == StatusRunning
	if !active && st.cancel == nil {
		m.mu.Unlock()
		return nil
	}
	done := st.runDone
	if done == nil {
		m.mu.Unlock()
		return domain.Errorf(domain.CodeInternal, "上传任务状态异常，请稍后再次删除")
	}
	select {
	case <-done:
		m.mu.Unlock()
		return nil
	default:
	}

	cancel := st.cancel
	if active {
		st.Status = StatusCanceled
		st.SpeedBytesPerSecond = 0
		st.Message = "正在停止上传任务"
		st.Error = ""
		st.UpdatedAt = timeutil.UnixFloat(time.Now())
	}
	st.cancelMode = "delete"
	snap := st
	m.mu.Unlock()

	if active {
		_ = m.persistTask(snap)
		m.broadcast(taskID)
	}
	if cancel != nil {
		cancel()
	}
	m.runCond.Broadcast()

	timer := time.NewTimer(deleteStopTimeout)
	defer timer.Stop()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return domain.Errorf(domain.CodeInternal, "任务正在停止，请稍后再次删除")
	case <-timer.C:
		return domain.Errorf(domain.CodeInternal, "任务正在停止，请稍后再次删除")
	}
}

func (m *Manager) popTask(taskID string) *taskState {
	m.mu.Lock()
	st := m.removeTaskLocked(taskID)
	if st == nil {
		m.mu.Unlock()
		return nil
	}
	m.mu.Unlock()
	m.deletePersisted(taskID)
	return st
}

func newTaskID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

func sortTasksDesc(tasks []Task) {
	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].CreatedAt > tasks[j].CreatedAt
	})
}
