package file

import (
	"testing"
	"time"
)

func TestCooldownLogGateSuppressesWithinWindow(t *testing.T) {
	g := newCooldownLogGate()
	base := time.Date(2026, 9, 10, 22, 30, 17, 0, time.Local)
	for i := 1; i <= 5; i++ {
		// 每次命中都落在 30s 窗口内（真实冷却期间任务被连续派发）。
		now := base.Add(time.Duration(i) * 50 * time.Millisecond)
		first, seq, prev := g.noteCooldown(2, 30*time.Second, now)
		if prev != nil {
			t.Fatalf("第 %d 次命中不应产生上一窗口汇总：%+v", i, prev)
		}
		if seq != i {
			t.Fatalf("窗口内序号应为 %d，实际 %d", i, seq)
		}
		if want := i == 1; first != want {
			t.Fatalf("第 %d 次命中 first 应为 %v，实际 %v", i, want, first)
		}
	}
}

func TestCooldownLogGateSummarizesPreviousWindow(t *testing.T) {
	g := newCooldownLogGate()
	base := time.Date(2026, 9, 10, 22, 30, 17, 0, time.Local)
	for i := 0; i < 3; i++ {
		g.noteCooldown(2, 30*time.Second, base.Add(time.Duration(i)*time.Millisecond))
	}
	// 窗口过期后再次命中：应输出上一窗口汇总，并作为新窗口首条。
	first, seq, prev := g.noteCooldown(2, 30*time.Second, base.Add(31*time.Second))
	if !first || seq != 1 {
		t.Fatalf("过期后应为新窗口首条，实际 first=%v seq=%d", first, seq)
	}
	if prev == nil || prev.Deferred != 3 {
		t.Fatalf("应汇总上一窗口 3 次暂缓，实际 %+v", prev)
	}
	if prev.WindowSeconds < 30 {
		t.Fatalf("窗口时长应≥30s，实际 %d", prev.WindowSeconds)
	}
}

func TestCooldownLogGateRecoveredClosesWindow(t *testing.T) {
	g := newCooldownLogGate()
	base := time.Date(2026, 9, 10, 22, 30, 17, 0, time.Local)
	for i := 0; i < 4; i++ {
		g.noteCooldown(2, 30*time.Second, base.Add(time.Duration(i)*time.Millisecond))
	}
	sum := g.noteRecovered(2, base.Add(12*time.Second))
	if sum == nil || sum.Deferred != 4 {
		t.Fatalf("恢复应汇总 4 次暂缓，实际 %+v", sum)
	}
	if sum.WindowSeconds != 12 {
		t.Fatalf("窗口时长应为 12s，实际 %d", sum.WindowSeconds)
	}
	if again := g.noteRecovered(2, base.Add(13*time.Second)); again != nil {
		t.Fatalf("窗口已关闭，再次恢复不应重复汇总：%+v", again)
	}
}

func TestCooldownLogGateAccountsIndependent(t *testing.T) {
	g := newCooldownLogGate()
	base := time.Date(2026, 9, 10, 22, 30, 17, 0, time.Local)
	if first, _, _ := g.noteCooldown(2, 30*time.Second, base); !first {
		t.Fatal("账号 2 首条应为 INFO")
	}
	if first, _, _ := g.noteCooldown(1, 30*time.Second, base); !first {
		t.Fatal("账号 1 独立窗口，首条应为 INFO")
	}
	if first, seq, _ := g.noteCooldown(2, 30*time.Second, base.Add(time.Millisecond)); first || seq != 2 {
		t.Fatalf("账号 2 第二次应被抑制，实际 first=%v seq=%d", first, seq)
	}
}

func TestCooldownLogGateNilSafe(t *testing.T) {
	var g *cooldownLogGate
	first, seq, prev := g.noteCooldown(2, 30*time.Second, time.Now())
	if !first || seq != 1 || prev != nil {
		t.Fatalf("nil gate 应回落为「直接记录首条」：first=%v seq=%d prev=%+v", first, seq, prev)
	}
	if sum := g.noteRecovered(2, time.Now()); sum != nil {
		t.Fatalf("nil gate 不应产生汇总：%+v", sum)
	}
}

// 零值 Service（未走构造器）也不得 panic：cooldownLog 为 nil 时回落直接记录。
func TestServiceZeroValueCooldownLogNilSafe(t *testing.T) {
	svc := &Service{}
	if first, seq, _ := svc.cooldownLog.noteCooldown(2, 30*time.Second, time.Now()); !first || seq != 1 {
		t.Fatalf("零值 Service 应回落为直接记录：first=%v seq=%d", first, seq)
	}
}
