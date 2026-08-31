package automation

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"litepan/internal/apikey"
	"litepan/internal/domain"
)

func TestSubmitRunQueuesWhileStartupGateBlocked(t *testing.T) {
	service := New(Options{})
	service.SetStartupGate(make(chan struct{}))
	result := service.submitRun(7, "schedule", true)
	if !result.queued || service.runningRuleID != 0 || len(service.pendingRuns) != 1 {
		t.Fatalf("认证闸门放行前规则应只入队: result=%+v running=%d pending=%d",
			result, service.runningRuleID, len(service.pendingRuns))
	}
}

func TestTriggerWebhookQueuesEveryMatchedRule(t *testing.T) {
	t.Parallel()

	const rawKey = "lpk_api_webhook_test"
	rules := newAutomationRuleRepo(
		webhookRule(1, "规则一"),
		webhookRule(2, "规则二"),
	)
	runs := &automationRunRepo{}
	keys := apikey.New(apikey.Options{Repo: &apiKeyRepo{key: &domain.ApiKey{
		ID:      1,
		KeyHash: apikey.Hash(rawKey),
		KeyType: domain.ApiKeyTypeTask,
		Status:  domain.ApiKeyStatusActive,
	}}})
	service := New(Options{Rules: rules, Runs: runs, ApiKeys: keys})

	result, err := service.TriggerWebhook(
		context.Background(),
		"Bearer "+rawKey,
		WebhookEvent{Event: "library.updated", Source: "test", Path: "/media"},
	)
	if err != nil {
		t.Fatalf("触发 Webhook 失败: %v", err)
	}
	triggered, ok := result["triggered"].([]map[string]any)
	if !ok {
		t.Fatalf("triggered 类型异常: %#v", result["triggered"])
	}
	if len(triggered) != 2 {
		t.Fatalf("期望两条匹配规则都被接收，实际 %d 条: %#v", len(triggered), triggered)
	}

	deadline := time.Now().Add(4 * time.Second)
	for time.Now().Before(deadline) {
		if runs.count() == 2 {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("期望两条规则依次执行，实际创建 %d 条运行记录", runs.count())
}

func TestScheduleOnceQueuesDueRuleAndAdvancesNextRun(t *testing.T) {
	t.Parallel()

	now := time.Now()
	triggerConfig, _ := json.Marshal(map[string]any{"time": "00:00"})
	actions, _ := json.Marshal([]RuleAction{{
		ID:        "local-1",
		Type:      domain.AutomationActionLocalUpload,
		Condition: domain.AutomationConditionAlways,
		Params:    map[string]any{"account_id": 1, "mappings": []string{"test"}, "target_parent_id": "root"},
	}})
	rule := &domain.AutomationRule{
		ID:            10,
		Name:          "定时规则",
		TriggerType:   domain.AutomationTriggerDaily,
		TriggerConfig: triggerConfig,
		Actions:       actions,
		Status:        domain.AutomationStatusRunning,
		NextRunAt:     now.Add(-time.Minute),
	}
	rules := newAutomationRuleRepo(rule)
	service := New(Options{Rules: rules, Runs: &automationRunRepo{}})

	service.runningRuleID = 99
	service.scheduleOnce(context.Background())

	if got := len(service.pendingRuns); got != 1 {
		t.Fatalf("期望定时任务进入队列，实际队列长度 %d", got)
	}
	if service.pendingRuns[0].ruleID != rule.ID {
		t.Fatalf("队列中的规则 ID = %d, want %d", service.pendingRuns[0].ruleID, rule.ID)
	}
	stored, err := rules.Get(context.Background(), rule.ID)
	if err != nil {
		t.Fatalf("获取规则失败: %v", err)
	}
	if !stored.NextRunAt.After(now) {
		t.Fatalf("NextRunAt 未推进到未来时间: %v", stored.NextRunAt)
	}

	service.scheduleOnce(context.Background())
	if got := len(service.pendingRuns); got != 1 {
		t.Fatalf("期望同一条定时规则不重复排队，实际队列长度 %d", got)
	}
}

func TestComputeNextRunIntervalUsesNextDayAnchorWhenTodayStartPassed(t *testing.T) {
	t.Parallel()

	loc := time.FixedZone("UTC+8", 8*3600)
	base := time.Date(2026, 7, 31, 12, 0, 0, 0, loc)

	got := computeNextRun(domain.AutomationTriggerInterval, map[string]any{
		"start_time":     "01:00",
		"interval_hours": 1,
	}, base)
	want := time.Date(2026, 7, 31, 13, 0, 0, 0, loc)
	if !got.Equal(want) {
		t.Fatalf("NextRunAt = %v, want %v", got, want)
	}

	got = computeNextRun(domain.AutomationTriggerInterval, map[string]any{
		"start_time":     "13:00",
		"interval_hours": 1,
	}, base)
	want = time.Date(2026, 7, 31, 13, 0, 0, 0, loc)
	if !got.Equal(want) {
		t.Fatalf("当日未到起点时 NextRunAt = %v, want %v", got, want)
	}
}

func TestAdvanceNextRunIntervalKeepsSameDaySlotsThenResetsToNextAnchor(t *testing.T) {
	t.Parallel()

	loc := time.FixedZone("UTC+8", 8*3600)
	cfg := map[string]any{
		"start_time":     "13:00",
		"interval_hours": 5,
	}

	got := advanceNextRun(domain.AutomationTriggerInterval, cfg, time.Date(2026, 7, 31, 13, 0, 0, 0, loc))
	want := time.Date(2026, 7, 31, 18, 0, 0, 0, loc)
	if !got.Equal(want) {
		t.Fatalf("同日下一档 = %v, want %v", got, want)
	}

	got = advanceNextRun(domain.AutomationTriggerInterval, cfg, time.Date(2026, 7, 31, 23, 0, 0, 0, loc))
	want = time.Date(2026, 8, 1, 13, 0, 0, 0, loc)
	if !got.Equal(want) {
		t.Fatalf("跨天后应回到次日锚点，got %v want %v", got, want)
	}
}

func TestAdvanceNextRunIntervalUsesLocalDayBoundaryForPersistedUTCTime(t *testing.T) {
	loc := time.FixedZone("UTC+8", 8*3600)
	cfg := map[string]any{
		"start_time":     "12:42",
		"interval_hours": 1,
	}

	currentUTC := time.Date(2026, 7, 31, 23, 42, 0, 0, time.UTC)
	got := advanceIntervalRunAt(wallClockTimeIn(currentUTC, loc), cfg)
	want := time.Date(2026, 8, 1, 8, 42, 0, 0, loc)
	if !got.Equal(want) {
		t.Fatalf("持久化 UTC 时间恢复后下一档 = %v, want %v", got, want)
	}
}

func TestAdvanceNextRunDailyUsesLocalClockForPersistedUTCTime(t *testing.T) {
	loc := time.FixedZone("UTC+8", 8*3600)
	currentUTC := time.Date(2026, 7, 31, 17, 0, 0, 0, time.UTC)

	got := advanceDailyRunAt(wallClockTimeIn(currentUTC, loc), map[string]any{"time": "01:00"})
	want := time.Date(2026, 8, 2, 1, 0, 0, 0, loc)
	if !got.Equal(want) {
		t.Fatalf("persisted UTC daily next run = %v, want %v", got, want)
	}
}

func TestNormalizeDailyRunCorrectsShiftedPersistedTime(t *testing.T) {
	loc := time.FixedZone("UTC+8", 8*3600)
	shifted := time.Date(2026, 8, 2, 9, 0, 0, 0, loc)

	got := normalizeDailyRunAt(map[string]any{"time": "01:00"}, shifted)
	want := time.Date(2026, 8, 2, 1, 0, 0, 0, loc)
	if !got.Equal(want) {
		t.Fatalf("shifted daily next run = %v, want %v", got, want)
	}
}

func TestRunAsyncQueuesInsteadOfRejectingWhenBusy(t *testing.T) {
	t.Parallel()

	service := New(Options{Rules: newAutomationRuleRepo(), Runs: &automationRunRepo{}})
	service.runningRuleID = 99

	result, err := service.RunAsync(context.Background(), 42, "manual")
	if err != nil {
		t.Fatalf("RunAsync 不应在忙时直接报错: %v", err)
	}
	if result["submitted"] != true {
		t.Fatalf("submitted = %#v, want true", result["submitted"])
	}
	if result["queued"] != true {
		t.Fatalf("queued = %#v, want true", result["queued"])
	}
	if got := len(service.pendingRuns); got != 1 {
		t.Fatalf("期望手动执行进入队列，实际队列长度 %d", got)
	}
	if service.pendingRuns[0].ruleID != 42 {
		t.Fatalf("队列中的规则 ID = %d, want 42", service.pendingRuns[0].ruleID)
	}
}

func TestValidateRuleRequiresLibrarySelectionForEmbyLibraryMode(t *testing.T) {
	t.Parallel()

	service := New(Options{Rules: newAutomationRuleRepo(), Runs: &automationRunRepo{}})
	// 仅保留 local_upload，Emby 校验已移除，此测试改为校验 local_upload 缺少账号失败
	result, err := service.ValidateRule(context.Background(), []RuleAction{
		{ID: "local-1", Type: domain.AutomationActionLocalUpload, Params: map[string]any{}},
	})
	if err != nil {
		t.Fatalf("ValidateRule 返回错误: %v", err)
	}
	if result.OK {
		t.Fatalf("期望未选择账号时校验失败")
	}
	if len(result.Issues) == 0 || !strings.Contains(result.Issues[0].Message, "未选择") {
		t.Fatalf("issues=%#v", result.Issues)
	}
}

func webhookRule(id int64, name string) *domain.AutomationRule {
	triggerConfig, _ := json.Marshal(map[string]any{"event": "library.updated"})
	actions, _ := json.Marshal([]RuleAction{{
		ID:        "local-1",
		Type:      domain.AutomationActionLocalUpload,
		Condition: domain.AutomationConditionAlways,
		Params:    map[string]any{"account_id": 1, "mappings": []string{"test"}, "target_parent_id": "root"},
	}})
	return &domain.AutomationRule{
		ID:            id,
		Name:          name,
		TriggerType:   domain.AutomationTriggerWebhook,
		TriggerConfig: triggerConfig,
		Actions:       actions,
		Status:        domain.AutomationStatusRunning,
	}
}

type automationRuleRepo struct {
	mu        sync.Mutex
	rules     map[int64]*domain.AutomationRule
	nextID    int64
	createErr error
	updateErr error
}

func newAutomationRuleRepo(rules ...*domain.AutomationRule) *automationRuleRepo {
	repo := &automationRuleRepo{rules: make(map[int64]*domain.AutomationRule, len(rules))}
	for _, rule := range rules {
		repo.rules[rule.ID] = rule
		if rule.ID > repo.nextID {
			repo.nextID = rule.ID
		}
	}
	return repo
}

func (r *automationRuleRepo) Create(_ context.Context, rule *domain.AutomationRule) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.createErr != nil {
		return 0, r.createErr
	}
	r.nextID++
	copy := *rule
	copy.ID = r.nextID
	r.rules[copy.ID] = &copy
	return copy.ID, nil
}

func (r *automationRuleRepo) Update(_ context.Context, rule *domain.AutomationRule) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.updateErr != nil {
		return r.updateErr
	}
	copy := *rule
	r.rules[rule.ID] = &copy
	return nil
}

func (r *automationRuleRepo) Delete(context.Context, int64) error { return nil }

func (r *automationRuleRepo) Get(_ context.Context, id int64) (*domain.AutomationRule, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	rule := *r.rules[id]
	return &rule, nil
}

func (r *automationRuleRepo) List(context.Context, bool) ([]*domain.AutomationRule, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	ids := make([]int64, 0, len(r.rules))
	for id := range r.rules {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	out := make([]*domain.AutomationRule, 0, len(ids))
	for _, id := range ids {
		rule := *r.rules[id]
		out = append(out, &rule)
	}
	return out, nil
}

type automationRunRepo struct {
	mu   sync.Mutex
	next int64
	runs map[int64]*domain.AutomationRun
}

func (r *automationRunRepo) Create(_ context.Context, run *domain.AutomationRun) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.runs == nil {
		r.runs = make(map[int64]*domain.AutomationRun)
	}
	r.next++
	copy := *run
	copy.ID = r.next
	r.runs[r.next] = &copy
	return r.next, nil
}

func (r *automationRunRepo) Update(_ context.Context, run *domain.AutomationRun) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	copy := *run
	r.runs[run.ID] = &copy
	return nil
}

func (r *automationRunRepo) List(context.Context, int64, int) ([]*domain.AutomationRun, error) {
	return nil, nil
}

func (r *automationRunRepo) Clear(context.Context) (int, error) { return 0, nil }

func (r *automationRunRepo) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.runs)
}

type apiKeyRepo struct {
	key *domain.ApiKey
}

func (r *apiKeyRepo) List(context.Context) ([]*domain.ApiKey, error) {
	return []*domain.ApiKey{r.key}, nil
}
func (r *apiKeyRepo) Get(context.Context, int64) (*domain.ApiKey, error)        { return r.key, nil }
func (r *apiKeyRepo) GetByHash(context.Context, string) (*domain.ApiKey, error) { return r.key, nil }
func (r *apiKeyRepo) Count(context.Context) (int, error)                        { return 1, nil }
func (r *apiKeyRepo) Create(context.Context, *domain.ApiKey) (int64, error)     { return 1, nil }
func (r *apiKeyRepo) Update(context.Context, *domain.ApiKey) error              { return nil }
func (r *apiKeyRepo) Delete(context.Context, int64) error                       { return nil }
func (r *apiKeyRepo) TouchLastUsed(context.Context, int64, time.Time) error     { return nil }
