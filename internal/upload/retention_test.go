package upload

import (
	"testing"
	"time"
)

func seedRetentionTasks() []Task {
	now := time.Now()
	mk := func(id, status string, ageDays float64) Task {
		updated := now.Add(-time.Duration(ageDays * 24 * float64(time.Hour))).Unix()
		return Task{TaskID: id, Status: status, UpdatedAt: float64(updated), CreatedAt: float64(updated)}
	}
	return []Task{
		mk("s-new", StatusSuccess, 1),
		mk("s-mid", StatusSuccess, 10),
		mk("s-old", StatusSuccess, 40),
		mk("k-old", StatusSkipped, 50),
		mk("c-old", StatusCanceled, 60),
		mk("p-keep", StatusPaused, 90),
		mk("f-keep", StatusFailed, 90),
		mk("r-keep", StatusRunning, 90),
		mk("n-keep", StatusPending, 90),
	}
}

// 超期清理：仅终态成功类，非终态永不清理。
func TestSelectRetentionVictimsByDays(t *testing.T) {
	tasks := seedRetentionTasks()
	victims := selectRetentionVictims(tasks, RetentionConfig{Days: 30}, time.Now())
	want := map[string]bool{"s-old": true, "k-old": true, "c-old": true}
	if len(victims) != len(want) {
		t.Fatalf("victims=%v", victims)
	}
	for _, id := range victims {
		if !want[id] {
			t.Fatalf("不应清理 %s（非超期成功类）", id)
		}
	}
	for _, id := range []string{"p-keep", "f-keep", "r-keep", "n-keep", "s-new", "s-mid"} {
		for _, v := range victims {
			if v == id {
				t.Fatalf("非终态/未超期任务被清理：%s", id)
			}
		}
	}
}

// 超量清理：保留最新 Max 条，其余清理（含未超期者）。
func TestSelectRetentionVictimsByMax(t *testing.T) {
	tasks := seedRetentionTasks()
	victims := selectRetentionVictims(tasks, RetentionConfig{Max: 2}, time.Now())
	// 成功类候选按新→旧：s-new, s-mid, s-old, k-old, c-old → 保留前 2，清理后 3
	if len(victims) != 3 {
		t.Fatalf("victims=%v", victims)
	}
	got := map[string]bool{}
	for _, v := range victims {
		got[v] = true
	}
	for _, id := range []string{"s-old", "k-old", "c-old"} {
		if !got[id] {
			t.Fatalf("应清理 %s，实际 %v", id, victims)
		}
	}
	for _, id := range []string{"s-new", "s-mid"} {
		if got[id] {
			t.Fatalf("最新记录不应被清理：%s", id)
		}
	}
}

// 组合策略：Days 与 Max 同时生效时按更严格者（Max 之内仍看天数）。
func TestSelectRetentionVictimsCombined(t *testing.T) {
	tasks := seedRetentionTasks()
	victims := selectRetentionVictims(tasks, RetentionConfig{Days: 5, Max: 3}, time.Now())
	got := map[string]bool{}
	for _, v := range victims {
		got[v] = true
	}
	// Max=3 → 保留 s-new, s-mid, s-old；Days=5 → 其中 s-mid/s-old 仍超期 → 清理
	for _, id := range []string{"s-mid", "s-old", "k-old", "c-old"} {
		if !got[id] {
			t.Fatalf("组合策略应清理 %s，实际 %v", id, victims)
		}
	}
	if got["s-new"] {
		t.Fatalf("1 天内的最新成功记录不应被清理")
	}
}

// 禁用（Days<=0 且 Max<=0）不清理任何记录。
func TestSelectRetentionVictimsDisabled(t *testing.T) {
	tasks := seedRetentionTasks()
	if victims := selectRetentionVictims(tasks, RetentionConfig{}, time.Now()); len(victims) != 0 {
		t.Fatalf("禁用状态不应清理：%v", victims)
	}
}

// 边界：Max 大于候选数、Days 超大 → 不清理。
func TestSelectRetentionVictimsBoundaries(t *testing.T) {
	tasks := seedRetentionTasks()
	if victims := selectRetentionVictims(tasks, RetentionConfig{Max: 100}, time.Now()); len(victims) != 0 {
		t.Fatalf("Max 大于候选数不应清理：%v", victims)
	}
	if victims := selectRetentionVictims(tasks, RetentionConfig{Days: 3650}, time.Now()); len(victims) != 0 {
		t.Fatalf("超长保留期不应清理：%v", victims)
	}
}
