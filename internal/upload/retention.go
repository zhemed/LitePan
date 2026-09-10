package upload

import (
	"context"
	"sort"
	"strings"
	"time"

	"litepan/internal/settings"
)

// 工作集保留策略（0.0.28）：历史成功记录不再永久堆积。
//
// 语义：只清理**终态成功类**任务记录（success/skipped/canceled）——超期（早于
// Days 天）或超出条数上限（多于 Max 条，保留最新的）者；**绝不**触碰未完成任务、
// 本地文件或网盘文件（删除动作复用 Manager.Delete，其本地清理对 server_local
// 源直接跳过）。
type RetentionConfig struct {
	Days int
	Max  int
}

const (
	// 默认保留天数：registry 未配置时的兜底（设置项默认值同为 30）。
	defaultRetentionDays = 30
	retentionInterval    = time.Hour
)

// RetentionConfig 读取运行时配置；Settings 未注入时使用默认值。
func (m *Manager) RetentionConfig() RetentionConfig {
	if m == nil || m.settings == nil {
		return RetentionConfig{Days: defaultRetentionDays}
	}
	return RetentionConfig{
		Days: m.settings.Int(settings.KeyUploadRetentionDays),
		Max:  m.settings.Int(settings.KeyUploadRetentionMax),
	}
}

// selectRetentionVictims 纯函数：返回应清理的任务 ID。
// 规则（Days 与 Max 同时生效时按更严格者）：
//   - 仅 success/skipped/canceled 参与（非终态永不清理）
//   - Max>0：按 UpdatedAt 降序保留最新 Max 条，其余直接清理
//   - Days>0：其余记录中 UpdatedAt 早于 now-Days 的清理
//   - Days<=0 且 Max<=0：不清理
func selectRetentionVictims(tasks []Task, cfg RetentionConfig, now time.Time) []string {
	if cfg.Days <= 0 && cfg.Max <= 0 {
		return nil
	}
	candidates := make([]Task, 0, len(tasks))
	for _, task := range tasks {
		if ownsBatchRoot(task) {
			// 第一层保护（0.0.29）：owned 批次根任务的记录是"删除云端批次根目录"
			// 完整性判据的数据基础，自动清理会削弱该保护 → 一律保留。
			continue
		}
		switch task.Status {
		case StatusSuccess, StatusSkipped, StatusCanceled:
			candidates = append(candidates, task)
		}
	}
	if len(candidates) == 0 {
		return nil
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].UpdatedAt == candidates[j].UpdatedAt {
			return candidates[i].CreatedAt > candidates[j].CreatedAt
		}
		return candidates[i].UpdatedAt > candidates[j].UpdatedAt
	})
	var cutoff time.Time
	if cfg.Days > 0 {
		cutoff = now.Add(-time.Duration(cfg.Days) * 24 * time.Hour)
	}
	victims := make([]string, 0, len(candidates))
	for index, task := range candidates {
		if cfg.Max > 0 && index >= cfg.Max {
			victims = append(victims, task.TaskID)
			continue
		}
		if !cutoff.IsZero() && time.Unix(int64(task.UpdatedAt), 0).Before(cutoff) {
			victims = append(victims, task.TaskID)
		}
	}
	return victims
}

// pruneRetainedTasks 执行一次清理，返回实际清理条数。
func (m *Manager) pruneRetainedTasks(cfg RetentionConfig, now time.Time) int {
	if m == nil || (cfg.Days <= 0 && cfg.Max <= 0) {
		return 0
	}
	victims := selectRetentionVictims(m.List(context.Background(), 0), cfg, now)
	if len(victims) == 0 {
		return 0
	}
	removed := 0
	ctx := context.Background()
	for _, taskID := range victims {
		ok, err := m.Delete(ctx, taskID, false)
		if err != nil || !ok {
			continue
		}
		removed++
	}
	if removed > 0 && m.log != nil {
		m.log.Info("上传记录保留策略清理完成",
			"removed", removed,
			"retention_days", cfg.Days,
			"retention_max", cfg.Max)
	}
	return removed
}

// retentionLoop 启动后立即清理一次，之后每小时一次；随 runCtx 优雅退出。
func (m *Manager) retentionLoop(ctx context.Context) {
	if m == nil || ctx == nil {
		return
	}
	run := func() {
		defer func() {
			if r := recover(); r != nil && m.log != nil {
				m.log.Warn("上传记录保留策略清理异常", "panic", r)
			}
		}()
		m.pruneRetainedTasks(m.RetentionConfig(), time.Now())
	}
	run()
	ticker := time.NewTicker(retentionInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			run()
		}
	}
}

// ownsBatchRoot 判断任务是否携带 owned 批次根元数据（其记录必须保留）。
func ownsBatchRoot(task Task) bool {
	if task.Result == nil {
		return false
	}
	owned, _ := task.Result["batch_root_owned"].(bool)
	if !owned {
		return false
	}
	rootID, _ := task.Result["batch_root_id"].(string)
	return strings.TrimSpace(rootID) != ""
}
