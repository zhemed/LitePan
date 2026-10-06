package api

import (
	"testing"

	"litepan/internal/logx"
)

// 日志详情不再只给 ERROR 级别：非错误级别也必须带上 details（前端"查看详细信息/复制日志"依赖它）。
func TestToLogDTOCarriesDetailsForNonErrorLevels(t *testing.T) {
	entry := logx.Entry{
		Level:   logx.LevelInfo,
		Module:  "api",
		Message: "系统设置已更新",
		Details: map[string]any{"keys": "log_retention_days", "count": 1},
	}
	dto := toLogDTO(entry, 1)
	if dto.Details == nil {
		t.Fatal("INFO 级日志的 details 不应被丢弃")
	}
	if dto.Details["keys"] != "log_retention_days" {
		t.Fatalf("details 内容异常：%v", dto.Details)
	}

	warn := toLogDTO(logx.Entry{Level: logx.LevelWarn, Module: "api", Message: "x", Details: map[string]any{"a": 1}}, 2)
	if warn.Details == nil {
		t.Fatal("WARN 级日志的 details 不应被丢弃")
	}

	none := toLogDTO(logx.Entry{Level: logx.LevelInfo, Module: "api", Message: "无详情"}, 3)
	if none.Details != nil {
		t.Fatalf("无详情时应保持 nil（omitempty 语义），实际 %v", none.Details)
	}
}
