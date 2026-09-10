package upload

import (
	"context"
	"fmt"
	"testing"
)

func seedWindowTasks(m *Manager, n int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := 0; i < n; i++ {
		st := &taskState{Task: Task{
			TaskID:    fmt.Sprintf("s-%02d", i),
			AccountID: 1,
			Status:    StatusSuccess,
			CreatedAt: float64(1000 + i), // 越大越新（列表按 CreatedAt DESC）
			UpdatedAt: float64(1000 + i),
		}}
		m.tasks[st.TaskID] = st
	}
	for i, status := range []string{StatusPaused, StatusFailed, StatusPending, StatusRunning} {
		st := &taskState{Task: Task{
			TaskID:    fmt.Sprintf("a-%d", i),
			AccountID: 1,
			Status:    status,
			CreatedAt: float64(500 + i),
			UpdatedAt: float64(500 + i),
		}}
		m.tasks[st.TaskID] = st
	}
}

// 0.0.27：默认窗口 = 全部非终态 + 最近 N 条已完成；汇总与窗口无关。
func TestWindowTasksKeepsActiveAndRecentTerminal(t *testing.T) {
	m := NewManager(Options{})
	seedWindowTasks(m, 20)
	tasks, summary := m.WindowTasks(context.Background(), 0, 5)
	if len(tasks) != 4+5 {
		t.Fatalf("窗口应含 4 非终态 + 5 已完成，实际 %d", len(tasks))
	}
	successes := 0
	newestSeen := ""
	for _, task := range tasks {
		if task.Status == StatusSuccess {
			successes++
			if newestSeen == "" {
				newestSeen = task.TaskID
			}
		}
	}
	if successes != 5 {
		t.Fatalf("窗口内已完成应 5 条，实际 %d", successes)
	}
	if newestSeen != "s-19" {
		t.Fatalf("窗口应保留最新成功记录，实际首条 %s", newestSeen)
	}
	if summary.Total != 24 || summary.Counts[StatusSuccess] != 20 || summary.Counts[StatusPaused] != 1 {
		t.Fatalf("汇总应与窗口无关：%+v", summary)
	}
}

func TestListFilteredStatusLimitOffset(t *testing.T) {
	m := NewManager(Options{})
	seedWindowTasks(m, 20)
	limit := m.ListFiltered(context.Background(), 0, ListFilter{Statuses: []string{StatusSuccess}, Limit: 3})
	if len(limit) != 3 || limit[0].TaskID != "s-19" {
		t.Fatalf("limit 应取最新 3 条：%d %v", len(limit), limit[0].TaskID)
	}
	offset := m.ListFiltered(context.Background(), 0, ListFilter{Statuses: []string{StatusSuccess}, Limit: 3, Offset: 3})
	if len(offset) != 3 || offset[0].TaskID != "s-16" {
		t.Fatalf("offset 语义错误：%v", offset[0].TaskID)
	}
	all := m.ListFiltered(context.Background(), 0, ListFilter{Statuses: []string{StatusSuccess}})
	if len(all) != 20 {
		t.Fatalf("无 limit 应返回全部成功记录，实际 %d", len(all))
	}
	active := m.ListFiltered(context.Background(), 0, ListFilter{Statuses: []string{StatusPaused, StatusFailed}})
	if len(active) != 2 {
		t.Fatalf("多状态过滤应返回 2 条，实际 %d", len(active))
	}
}
