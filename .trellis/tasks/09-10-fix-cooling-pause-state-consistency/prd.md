# 09-10-fix-cooling-pause-state-consistency（批次 1）

## Goal

修复**冷却重入队覆盖暂停**导致的状态分叉：账号冷却期间的重入队必须「暂停/取消优先」，且内存状态与持久化状态保持一致，**重启后用户已暂停的任务不得自行续传**。

## Background

- 取证（`09-10-investigate-11-cooldown-storm` §4）：19 条任务落库为 `status=pending` + `账号网络冷却中，30 秒后自动重试`，而内存/界面为 `paused` + `上传已暂停`（updated_at 相差 0.3ms）。`persist.go:80-91` 启动恢复会把 `pending` 行 `go m.runTask(id)` ⇒ **重启即自行续传**，与界面矛盾。
- 代码现状：
  - `worker.go:107-127` 先 `canCooldownWait(taskID)` 判断、再 `m.patch` 无条件写回 `pending` + 冷却文案 —— **检查与写入非原子**，中间落地的 `pause()` 会被覆盖；
  - `state.go:21-34` `patch()` 在解锁后 `persistTask(snap)` 且 `snap := st` 是**活体指针**，持久化读取与并发状态变更之间存在竞态（落库内容可能既非旧态也非新态）。

## Requirements

- **R1 原子守卫**：新增 `Manager.beginCooldownWait(taskID, seconds) bool`，在 `m.mu` 内一次性判定「任务存在、未停止、`cancelMode != "pause"`、状态为 `pending|running`」，满足才写回 `pending` + 冷却文案 + `resumePriority` 并返回 true；否则返回 false 且**不得修改任何字段**。`worker.go` 冷却分支改为：守卫失败立即 `return false`（保持暂停态），成功才进入等待。
- **R2 持久化快照**：`patch()` 持久化时使用**值拷贝**（`snap := *st`）而非活体指针，保证落库内容等于该次迁移的状态。
- **R3 等待结束仍以暂停优先**：等待结束后沿用 `canCooldownWait` 判定（false ⇒ 不重入队、保持暂停），行为不变。
- **R4 回归测试**：
  - 守卫拒绝：任务为 `paused` / `cancelMode="pause"` / 已取消 / 不存在 / 停止中 → 返回 false，且 `status`、`message`、`resumePriority` 均不变；
  - 竞态复现：处于冷却等待的任务被 `pause()` 后，等待结束不会重入队、不会覆盖为 pending；
  - 一致性：并发「冷却重入队 × 暂停」后，**内存状态 == 持久化行状态**（用 fake repo 断言），且持久化行不是 `pending`。
- **R5 范围**：仅 `internal/upload/{worker.go,state.go}`（+ 必要时 `types.go`/注释）与测试；不动并发/节流参数、不动前端、不动驱动。
- **R6 发布**：版本 `0.0.34`，质量门 + 镜像推送 + tag/release + 本地部署验证三连。

## Constraints

- 用户偏好保护：`upload_task_concurrency`、189 500ms 节流、冷却阈值（3 次/30s）均不得改动。
- 不引入配置开关；不改变 `resume_data`/进度语义。
- 生产机 `10.0.0.11` 不在范围内。

## Acceptance Criteria

- [ ] R1：`beginCooldownWait` 在 5 类不可等待场景下返回 false 且字段零变更（断言覆盖）
- [ ] R1：worker 冷却分支不再出现「检查后无条件写回 pending」
- [ ] R2：`patch()` 持久化使用值快照（`grep` 可验证，且测试覆盖）
- [ ] R3：等待结束后 `canCooldownWait == false` 时不重入队（测试覆盖）
- [ ] R4：新增测试在修复前失败、修复后通过（并发/竞态场景）
- [ ] R5：`git diff --name-only` 仅含 `internal/upload/**`、`.trellis/spec/**`、版本文件
- [ ] R6：`go vet`、`go test ./...`（`internal/upload` 加 `-race`）、`web type-check/build` 全绿；`0.0.34` 镜像推送 + tag/release + 本地部署三连
- [ ] spec 同步：`upload-task-api.md` §8 第 3 条更新为「暂停优先 + 原子守卫 + 值快照」的实现契约

## Notes

- 复杂任务：见本目录 `design.md`（边界/契约/取舍）与 `implement.md`（顺序/验证/回滚）。
