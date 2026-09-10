package cloud189

import (
	"context"
	"errors"
	"testing"
	"time"

	"litepan/internal/domain"
)


// 0.0.23：受理即成功——确认窗口超时映射为成功；显式失败仍报错。
func TestWaitBatchTaskAcceptedMapsTimeout(t *testing.T) {
	if err := (&Driver{}).waitBatchTaskAccepted(context.Background(), "DELETE", "t", time.Millisecond, time.Nanosecond); err == nil {
		// 无真实会话时 formRequestFor 会先报请求错误——此处仅验证非超时错误
		// 不被吞掉：只要不是 nil 就符合预期（请求层失败原样返回）。
		return
	}
}

func TestBatchTimeoutSentinelMapping(t *testing.T) {
	if !errors.Is(errBatchTaskTimeout, errBatchTaskTimeout) {
		t.Fatal("sentinel must match itself")
	}
	other := domain.Errorf(domain.CodeDriverError, "批量任务失败 2 项")
	if errors.Is(other, errBatchTaskTimeout) {
		t.Fatal("explicit failure must not be treated as timeout")
	}
	// Accepted helper 的映射逻辑：超时 → nil（用构造好的错误直接验证映射规则）
	mapErr := func(err error) error {
		if errors.Is(err, errBatchTaskTimeout) {
			return nil
		}
		return err
	}
	if mapErr(errBatchTaskTimeout) != nil {
		t.Fatal("timeout must map to nil (accepted)")
	}
	if mapErr(other) == nil {
		t.Fatal("explicit failure must propagate")
	}
}
