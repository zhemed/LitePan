# 09-10-fix-persist-stale-overwrite（批次 1 补充）

## Goal

关闭批次 1（`09-10-fix-cooling-pause-state-consistency`）留下的**残余竞态**：状态持久化在 `m.mu` 之外执行，两次并发迁移的写库顺序可能倒置，**过期快照会覆盖较新状态**（内存 `paused` / 库内 `pending`），进而让重启恢复把「用户已暂停」的任务重新入队。

## Background

- 批次 1 已把「判定 + 迁移」合并为原子守卫（`beginCooldownWait`）并把持久化改为**值快照**，但写入仍在锁外：并发下可能出现
  `A(冷却, UpdatedAt=T1)` → `B(暂停, UpdatedAt=T2>T1)` → 写库顺序却是 `B` 先、`A` 后 ⇒ 库里留下过期的 `pending`。
- 该缺口在批次 2 的验证阶段由 `go test -race -count=1 ./internal/upload/` **重复运行复现**（`TestCooldownPauseConcurrentConsistency` 间歇失败；同时暴露测试自身未加锁写 `m.tasks` 的干扰）。
- 对应不变式（spec §8.3 第 2 条）：**任意时刻内存态 == 最后落库态**。

## Requirements

- **R1 新鲜度守卫**：新增 `Manager.persistStateSnapshot(taskID string, snap *taskState)`：
  1) 用独立 `persistMu` 串行化写入，保证「更晚的迁移」后写；
  2) 写前在 `m.mu` 下比较当前 `UpdatedAt` 与快照 `UpdatedAt`，**过期快照直接丢弃**（不写库）。
- **R2 统一入口**：`patch()`、`beginCooldownWait()`、`pause()` 三处状态迁移统一改走该入口（其余路径保持原样，避免扩大改动面）。
- **R3 测试可信**：修正本包新增测试中**未加锁**写 `m.tasks` 的写法（并发保留协程读取时会被 race 检测捕获），确保 race 结果反映产品代码而非测试自身。
- **R4 证据**：`CGO_ENABLED=1 go test -race -count=1 ./internal/upload/` **连续 12 次全绿**（修复前间歇失败），并保留失败样本说明。
- **R5 范围**：仅 `internal/upload/{manager.go,state.go,lifecycle.go,cooldown_pause_test.go}`；不改熔断/批处理（批次 2）、不改前端/驱动。

## Constraints

- 不改变 `UpdatedAt` 语义与既有可见行为；不引入配置开关；不为其它迁移路径做「顺手重构」。
- 与本项同批发布的 0.0.35 一并交付（版本由仓库级 bump 承担，记录中注明）。

## Acceptance Criteria

- [x] R1：`persistStateSnapshot` 实现串行化 + 过期丢弃，并被 `patch/beginCooldownWait/pause` 使用
- [x] R2：`grep` 可见三处均不再直接调用 `m.persistTask(...)` 落状态快照
- [x] R3：测试内 `m.tasks` 写入全部在 `m.mu` 保护下
- [x] R4：`-race -count=1` 连续 12 次通过（记录失败样本：修复前 `TestCooldownPauseConcurrentConsistency` 间歇 FAIL，伴随 retention 协程读 map 的 DATA RACE 报告）
- [x] R5：`git diff --name-only` 仅含上述 4 个文件（+ 任务记录）
- [x] 质量门（`go vet ./...`、`go test ./...`、`-race`）与发布（随 0.0.35）通过

## Notes

- Lightweight 任务（PRD-only）：守卫逻辑简单，改动集中在 3 个调用点 + 1 个测试修正。
