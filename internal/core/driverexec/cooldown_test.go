package driverexec

import (
	"testing"
	"time"

	"litepan/internal/domain"
)

func TestCooldownErrorRoundTrip(t *testing.T) {
	err := CooldownError(29 * time.Second)
	seconds, ok := IsCooldownError(err)
	if !ok || seconds != 29 {
		t.Fatalf("seconds=%d ok=%v", seconds, ok)
	}
	if err.Error() == "" {
		t.Fatal("文案不应为空")
	}
	// 网络判定必须排除冷却错误（否则冷却消息里的"网络"会自指反馈）
	if domain.IsNetworkError(err) {
		t.Fatal("cooldown error must not be classified as network error")
	}
}

func TestIsCooldownErrorNegative(t *testing.T) {
	if _, ok := IsCooldownError(domain.Errorf(domain.CodeDriverError, "普通错误")); ok {
		t.Fatal("普通错误不应判为冷却")
	}
	if _, ok := IsCooldownError(nil); ok {
		t.Fatal("nil 不应判为冷却")
	}
}
