package accountprofile

import (
	"context"
	"testing"
	"time"

	"litepan/internal/driver"
)

// blockingProfileExecutor 阻塞在 Run 中，用于观测后台刷新的并发性。
type blockingProfileExecutor struct {
	started chan int64
	release chan struct{}
}

func (e *blockingProfileExecutor) Run(
	ctx context.Context,
	accountID int64,
	_ func(driver.Driver) error,
) error {
	select {
	case e.started <- accountID:
	case <-ctx.Done():
		return ctx.Err()
	}
	select {
	case <-e.release:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// TestBackgroundProfileRefreshRunsSerially 锁定 refreshGate 语义（upstream b5c9308）：
// 多个账号的后台资料刷新必须串行，不得并发打上游。
func TestBackgroundProfileRefreshRunsSerially(t *testing.T) {
	exec := &blockingProfileExecutor{
		started: make(chan int64, 2),
		release: make(chan struct{}, 2),
	}
	svc := New(exec)
	done := make(chan struct{}, 2)
	start := make(chan struct{})

	for _, accountID := range []int64{1, 2} {
		go func(id int64) {
			<-start
			svc.refresh(context.Background(), id)
			done <- struct{}{}
		}(accountID)
	}
	close(start)

	select {
	case <-exec.started:
	case <-time.After(time.Second):
		t.Fatal("首个账号资料刷新未启动")
	}
	select {
	case id := <-exec.started:
		t.Fatalf("后台账号资料刷新发生并发，账号 %d 提前启动", id)
	case <-time.After(80 * time.Millisecond):
	}

	exec.release <- struct{}{}
	select {
	case <-exec.started:
	case <-time.After(time.Second):
		t.Fatal("首个刷新结束后，第二个账号资料刷新未启动")
	}
	exec.release <- struct{}{}

	for range 2 {
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Fatal("后台账号资料刷新未结束")
		}
	}
}

// TestBackgroundProfileRefreshAbortsOnCanceledParent 父上下文已取消时不应等待闸门。
func TestBackgroundProfileRefreshAbortsOnCanceledParent(t *testing.T) {
	exec := &blockingProfileExecutor{
		started: make(chan int64, 1),
		release: make(chan struct{}, 1),
	}
	svc := New(exec)
	// 先占住闸门
	blocked := make(chan struct{})
	go func() {
		svc.refreshGate <- struct{}{}
		close(blocked)
		time.Sleep(200 * time.Millisecond)
		<-svc.refreshGate
	}()
	<-blocked

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	started := time.Now()
	svc.refresh(ctx, 99)
	if elapsed := time.Since(started); elapsed > 100*time.Millisecond {
		t.Fatalf("父上下文已取消时应立即返回，实际耗时 %v", elapsed)
	}
	select {
	case id := <-exec.started:
		t.Fatalf("取消的父上下文不应触发刷新，实际启动了账号 %d", id)
	default:
	}
}
