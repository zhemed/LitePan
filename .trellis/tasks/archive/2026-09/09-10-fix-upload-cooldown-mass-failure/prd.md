# fix-upload-cooldown-mass-failure

## Goal

修复"账号网络冷却 × worker 无冷却感知"导致的大批量秒级判死（调查 09-10-investigate-pause-mass-failures，实测 451 失败/秒）。`0.0.24` 发布。

## 方案（四步）

1. **类型化冷却错误**（`internal/core/driverexec`）：冷却返回 `CodeDriverError` + 详情 `{"account_cooldown": true, "retry_after_seconds": N}`（文案不变），新增 `IsCooldownError(err) (int, bool)`
2. **`domain.IsNetworkError` 去自指**：带 `account_cooldown` 详情的错误直接判非网络错（消除冷却消息含"网络"被再次计数的反馈回路）
3. **worker 冷却感知**（`internal/upload/worker.go`）：错误路径先判冷却——命中则任务**退回 pending（置顶、保留 progress/resumeData、message"账号网络冷却中，N 秒后自动重试"）**，worker 原地等到冷却结束再继续（并发=1 时天然让队列等待）；不再 failTask
4. **批次熔断安全网**：同批次连续 5 个任务同签名（错误码+文案前缀）失败 → 自动暂停该批次剩余 pending 任务，message 注明"批次已自动暂停：连续同因失败"

## Constraints

- 不碰 500ms 门/并发=1；不改冷却阈值（3 连败/30s）；MOVE/COPY 等其它路径不动
- 冷却等待期间用户暂停要能立即中断（ctx 取消 → 保持 paused）

## Acceptance Criteria

- [ ] vet 全绿 + go test ./... 零失败（新增：冷却识别/网络判定去自指/熔断签名计数）
- [ ] 0.0.24 三 tag + release + 部署三连
- [ ] 归档 + journal + push
