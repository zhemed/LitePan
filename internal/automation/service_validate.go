package automation

import (
	"context"
	"fmt"
	"strings"

	"litepan/internal/domain"
)

func (s *Service) ValidateRule(ctx context.Context, actions []RuleAction) (ValidationResult, error) {
	issues := make([]ValidationIssue, 0)
	for index, action := range actions {
		switch action.Type {
		case domain.AutomationActionEmbyRefresh:
			mode := strings.TrimSpace(anyString(action.Params["mode"]))
			if mode == "" {
				mode = "global"
			}
			if mode != "global" && mode != "library" {
				issues = append(issues, ValidationIssue{Level: "error", Message: "Emby 刷库模式无效", ActionIndex: index, ActionType: action.Type})
				continue
			}
			if mode == "library" && strings.TrimSpace(anyString(action.Params["library_id"])) == "" {
				issues = append(issues, ValidationIssue{Level: "error", Message: "请选择 Emby 媒体库", ActionIndex: index, ActionType: action.Type})
				continue
			}
			embyID := strings.TrimSpace(anyString(action.Params["emby_id"]))
			if s.emby == nil || !s.hasEmbyConfig(embyID) {
				issues = append(issues, ValidationIssue{Level: "error", Message: "所选 Emby 配置不存在", ActionIndex: index, ActionType: action.Type})
			}
		}
	}
	return ValidationResult{OK: len(issues) == 0, Issues: issues}, nil
}

func (s *Service) hasEmbyConfig(id string) bool {
	configs := s.emby.Snapshots(nil)
	if id == "" {
		return len(configs) > 0
	}
	for _, cfg := range configs {
		if cfg.ID == id {
			return true
		}
	}
	return false
}

func (s *Service) normalizeInput(ctx context.Context, in RuleInput) (RuleInput, error) {
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		return in, domain.Errorf(domain.CodeValidation, "请输入自动化名称")
	}
	if len([]rune(in.Name)) > 40 {
		return in, domain.Errorf(domain.CodeValidation, "自动化名称不能超过40个字符")
	}
	in.TriggerType = strings.TrimSpace(in.TriggerType)
	switch in.TriggerType {
	case domain.AutomationTriggerDaily, domain.AutomationTriggerInterval, domain.AutomationTriggerWebhook, domain.AutomationTriggerOfflineDownload:
	default:
		return in, domain.Errorf(domain.CodeValidation, "触发条件不支持")
	}
	if in.TriggerConfig == nil {
		in.TriggerConfig = map[string]any{}
	}
	switch in.TriggerType {
	case domain.AutomationTriggerDaily:
		if strings.TrimSpace(anyString(in.TriggerConfig["time"])) == "" {
			return in, domain.Errorf(domain.CodeValidation, "请选择每天触发时间")
		}
	case domain.AutomationTriggerInterval:
		if strings.TrimSpace(anyString(in.TriggerConfig["start_time"])) == "" {
			return in, domain.Errorf(domain.CodeValidation, "请选择首次触发时间")
		}
		if anyInt(in.TriggerConfig["interval_hours"]) <= 0 {
			return in, domain.Errorf(domain.CodeValidation, "间隔小时必须大于 0")
		}
	case domain.AutomationTriggerWebhook:
		if strings.TrimSpace(anyString(in.TriggerConfig["event"])) == "" {
			return in, domain.Errorf(domain.CodeValidation, "请输入 Webhook 事件名称")
		}
	case domain.AutomationTriggerOfflineDownload:
		if anyInt(in.TriggerConfig["account_id"]) <= 0 {
			return in, domain.Errorf(domain.CodeValidation, "请选择离线下载账号")
		}
		if strings.TrimSpace(anyString(in.TriggerConfig["path"])) == "" {
			return in, domain.Errorf(domain.CodeValidation, "请选择离线下载目录")
		}
	}
	if in.Status == "" {
		in.Status = domain.AutomationStatusRunning
	}
	if in.Status != domain.AutomationStatusRunning && in.Status != domain.AutomationStatusPaused {
		in.Status = domain.AutomationStatusRunning
	}
	if len(in.Actions) == 0 {
		return in, domain.Errorf(domain.CodeValidation, "至少需要添加一个执行动作")
	}
	if len(in.Actions) > 12 {
		return in, domain.Errorf(domain.CodeValidation, "当前最多支持 12 个动作")
	}
	for i := range in.Actions {
		if in.Actions[i].ID == "" {
			in.Actions[i].ID = fmt.Sprintf("act-%d", i+1)
		}
		in.Actions[i].Type = strings.TrimSpace(in.Actions[i].Type)
		switch in.Actions[i].Type {
		case domain.AutomationActionDelay, domain.AutomationActionEmbyRefresh:
		default:
			return in, domain.Errorf(domain.CodeValidation, "存在不支持的动作")
		}
		if in.Actions[i].Params == nil {
			in.Actions[i].Params = map[string]any{}
		}
		in.Actions[i].Condition = normalizedCondition(in.Actions[i].Condition, i)
	}
	validation, err := s.ValidateRule(ctx, in.Actions)
	if err != nil {
		return in, err
	}
	if !validation.OK {
		return in, domain.Errorf(domain.CodeValidation, "%s", validation.Issues[0].Message)
	}
	return in, nil
}
