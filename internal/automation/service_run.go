package automation

import (
	"context"
	"fmt"
	"strings"
	"time"

	"litepan/internal/domain"
	"litepan/internal/embyproxy"
)

func (s *Service) RunAsync(ctx context.Context, id int64, triggerSource string) (map[string]any, error) {
	res := s.submitRun(id, triggerSource, false)
	return map[string]any{
		"rule_id":        id,
		"submitted":      true,
		"trigger_source": triggerSource,
		"queued":         res.queued,
	}, nil
}

func (s *Service) runRule(id int64, triggerSource string) {
	defer s.endRun(id)
	parent := s.appCtx
	if parent == nil {
		parent = context.Background()
	}
	ctx, cancel := context.WithTimeout(parent, 6*time.Hour)
	defer cancel()

	rule, err := s.rules.Get(ctx, id)
	if err != nil {
		s.log.Warn("automation get rule failed", "rule_id", id, "err", err)
		return
	}
	actions := decodeActions(rule.Actions)
	run := &domain.AutomationRun{
		RuleID:        id,
		TriggerSource: triggerSource,
		Status:        domain.AutomationRunRunning,
		StartedAt:     time.Now(),
		Result:        mustJSON(map[string]any{"steps": []map[string]any{}}),
	}
	runID, err := s.runs.Create(ctx, run)
	if err != nil {
		s.log.Warn("automation create run failed", "rule_id", id, "err", err)
		return
	}
	run.ID = runID

	steps := make([]map[string]any, 0, len(actions))
	previousSuccess := true
	message := "执行完成"
	status := domain.AutomationRunSuccess
	for i, action := range actions {
		step := map[string]any{
			"index":     i,
			"type":      action.Type,
			"name":      actionDisplayName(action),
			"condition": normalizedCondition(action.Condition, i),
			"status":    "skipped",
			"success":   true,
			"message":   "条件未满足，已跳过",
		}
		if shouldRunAction(action.Condition, previousSuccess, i) {
			s.setRunningStep(id, i, actionDisplayName(action), action.Type)
			result := s.executeAction(ctx, action)
			for k, v := range result {
				step[k] = v
			}
			ok, _ := step["success"].(bool)
			previousSuccess = ok
			if step["status"] == "failed" {
				status = domain.AutomationRunFailed
				if msg := strings.TrimSpace(anyString(step["message"])); msg != "" {
					message = msg
				}
			}
		}
		steps = append(steps, step)
	}
	finishedAt := time.Now()
	run.Status = status
	run.Message = message
	run.Result = mustJSON(map[string]any{"steps": steps})
	run.FinishedAt = finishedAt
	_ = s.runs.Update(ctx, run)

	rule.LastRunAt = finishedAt
	rule.LastRunStatus = status
	rule.LastRunMessage = message
	if rule.Status == domain.AutomationStatusRunning {
		if rule.TriggerType == domain.AutomationTriggerWebhook {
			rule.NextRunAt = time.Time{}
		} else if triggerSource != "schedule" && (rule.NextRunAt.IsZero() || !rule.NextRunAt.After(finishedAt)) {
			rule.NextRunAt = computeNextRun(rule.TriggerType, decodeMap(rule.TriggerConfig), finishedAt)
		}
	}
	_ = s.rules.Update(ctx, rule)
}

func (s *Service) executeAction(ctx context.Context, action RuleAction) map[string]any {
	switch action.Type {
	case domain.AutomationActionDelay:
		return s.runDelay(ctx, action.Params)
	case domain.AutomationActionEmbyRefresh:
		return s.runEmbyRefresh(ctx, action.Params)
	default:
		return map[string]any{"status": "failed", "success": false, "message": "动作类型不支持"}
	}
}

func (s *Service) runDelay(ctx context.Context, params map[string]any) map[string]any {
	seconds := clampInt(anyInt(params["seconds"]), 1, 24*3600)
	timer := time.NewTimer(time.Duration(seconds) * time.Second)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return map[string]any{"status": "failed", "success": false, "message": "等待被取消"}
	case <-timer.C:
		return map[string]any{"status": "success", "success": true, "message": fmt.Sprintf("已等待 %d 秒", seconds), "data": map[string]any{"seconds": seconds}}
	}
}

func (s *Service) runEmbyRefresh(ctx context.Context, params map[string]any) map[string]any {
	if s.emby == nil {
		return map[string]any{"status": "failed", "success": false, "message": "Emby 服务未就绪"}
	}
	req := embyproxy.RefreshRequest{
		ConfigID:  strings.TrimSpace(anyString(params["emby_id"])),
		Mode:      strings.TrimSpace(anyString(params["mode"])),
		LibraryID: strings.TrimSpace(anyString(params["library_id"])),
	}
	result, err := s.emby.RefreshLibrary(ctx, req)
	if err != nil {
		return map[string]any{"status": "failed", "success": false, "message": err.Error()}
	}
	message := "已通知 Emby 刷库"
	if result.Mode == "library" && result.LibraryName != "" {
		message = "已通知 Emby 扫描媒体库：" + result.LibraryName
	}
	return map[string]any{
		"status":  "success",
		"success": true,
		"message": message,
		"data": map[string]any{
			"emby_id":      result.ConfigID,
			"emby_name":    result.ConfigName,
			"mode":         result.Mode,
			"task_id":      result.TaskID,
			"library_id":   result.LibraryID,
			"library_name": result.LibraryName,
		},
	}
}

type submitRunResult struct {
	queued bool
}

func (s *Service) submitRun(ruleID int64, triggerSource string, dedupe bool) submitRunResult {
	s.mu.Lock()
	if dedupe && s.pendingCount[ruleID] > 0 {
		s.mu.Unlock()
		return submitRunResult{queued: true}
	}
	if s.startupGate != nil && !s.startupReady {
		s.pendingRuns = append(s.pendingRuns, queuedRun{ruleID: ruleID, triggerSource: triggerSource})
		s.pendingCount[ruleID]++
		s.mu.Unlock()
		return submitRunResult{queued: true}
	}
	if s.runningRuleID != 0 {
		s.pendingRuns = append(s.pendingRuns, queuedRun{ruleID: ruleID, triggerSource: triggerSource})
		s.pendingCount[ruleID]++
		s.mu.Unlock()
		return submitRunResult{queued: true}
	}
	s.runningRuleID = ruleID
	s.mu.Unlock()
	go s.runRule(ruleID, triggerSource)
	return submitRunResult{queued: false}
}

func (s *Service) releaseStartupQueue() {
	var next *queuedRun
	s.mu.Lock()
	s.startupReady = true
	if s.runningRuleID == 0 {
		next = s.takePendingRunLocked()
		if next != nil {
			s.runningRuleID = next.ruleID
		}
	}
	s.mu.Unlock()
	if next != nil {
		go s.runRule(next.ruleID, next.triggerSource)
	}
}

func (s *Service) takePendingRunLocked() *queuedRun {
	if len(s.pendingRuns) == 0 {
		return nil
	}
	queued := s.pendingRuns[0]
	s.pendingRuns = s.pendingRuns[1:]
	if s.pendingCount[queued.ruleID] > 1 {
		s.pendingCount[queued.ruleID]--
	} else {
		delete(s.pendingCount, queued.ruleID)
	}
	return &queued
}

func (s *Service) endRun(ruleID int64) {
	var next *queuedRun
	s.mu.Lock()
	if s.runningRuleID == ruleID {
		if queued := s.takePendingRunLocked(); queued != nil {
			s.runningRuleID = queued.ruleID
			next = queued
		} else {
			s.runningRuleID = 0
		}
	}
	delete(s.runningStep, ruleID)
	s.mu.Unlock()
	if next != nil {
		go s.runRule(next.ruleID, next.triggerSource)
	}
}

func (s *Service) setRunningStep(ruleID int64, index int, name, actionType string) {
	s.mu.Lock()
	s.runningStep[ruleID] = map[string]any{
		"index": index,
		"name":  name,
		"type":  actionType,
	}
	s.mu.Unlock()
}

func normalizedCondition(v string, index int) string {
	cond := strings.TrimSpace(v)
	if index == 0 {
		return domain.AutomationConditionAlways
	}
	switch cond {
	case domain.AutomationConditionAlways, domain.AutomationConditionPrevSuccess, domain.AutomationConditionPrevFailed:
		return cond
	default:
		return domain.AutomationConditionPrevSuccess
	}
}

func shouldRunAction(condition string, previousSuccess bool, index int) bool {
	switch normalizedCondition(condition, index) {
	case domain.AutomationConditionAlways:
		return true
	case domain.AutomationConditionPrevFailed:
		return !previousSuccess
	default:
		return previousSuccess
	}
}

func actionDisplayName(action RuleAction) string {
	if name := strings.TrimSpace(action.Name); name != "" {
		return name
	}
	switch action.Type {
	case domain.AutomationActionDelay:
		return "等待"
	case domain.AutomationActionEmbyRefresh:
		return "Emby 刷库"
	default:
		return action.Type
	}
}
