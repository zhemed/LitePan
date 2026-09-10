# 09-10-fix-cooldown-log-amplification

## Goal

消除「账号网络冷却」日志的重复放大：生产实测**一次冷却事件在 0.35 秒内刷出 183 条完全同构的 INFO**（用户只点了一次暂停），需要在保持可观测性的前提下把重复日志收敛为「窗口首条 + 恢复汇总」。

## Background

- 现象取证见已归档任务 `09-10-investigate-11-cooldown-storm`（`research.md` §2/§3/§6A）：`driverexec` 账号级网络退避（3 次网络失败 → 30s）期间，每个被派发的任务都会在入口短路并记录一条 `上传暂缓：账号网络冷却，稍后自动重试`（`internal/file/service.go:372-378`），于是「1 次冷却 × N 任务 = N 条日志」。
- 用户诉求（2026-09-10）：「日志太多了……都是重复日志，你看看这个能不能改一下」。

## Requirements

- **R1 抑制重复**：同一账号、同一冷却窗口内，「上传暂缓」只输出 **1 条 INFO**（保留 `account_id` / `name` / `retry_after_seconds`），窗口内其余命中降为 **Debug**（默认 `log_level=info` 时不可见，调 debug 仍可取证）。
- **R2 保留规模信息**：窗口结束（超时后再次命中，或该账号恢复成功）时输出 **1 条 INFO 汇总**，含本窗口暂缓次数与持续秒数——不能因为抑制而丢失「影响多少任务」这一运维信息。
- **R3 按账号隔离**：多账号（如 `1=天翼云盘` / `2=115网盘`）各自独立窗口，互不抑制。
- **R4 安全**：并发安全（HTTP + worker 并发调用）；`nil` gate 或未初始化的 Service 不得 panic；不得改变任何任务状态、错误返回或重试语义（纯日志行为变更）。
- **R5 回归测试**：新增 gate 单测 + service 级日志断言（捕获 slog 输出），断言 N 次冷却只产生 1 条 INFO「上传暂缓」+ 1 条汇总，其余为 Debug。
- **R6 范围纪律**：**只改日志行为**；不修「冷却 patch 与 pause 竞态导致的 DB/内存分叉」（research.md §6C，用户当前选择观察）、不动并发闸门与 189 节流参数、不改前端。
- **R7 文档与发布**：spec `upload-task-api.md` §8 增补抑制规则；版本递增 `0.0.32`（`fix` 级别），构建镜像、打 tag、发布、本地容器部署并做「健康 + 登录 + 上传任务列表」三连验证。

## Constraints

- 版本规则：`0.0.32`（不跳 `1.0.0`）。
- 生产机 `10.0.0.11` **不在本任务范围内**（不部署、不修改）；仅本地实例验证。
- 不引入配置项/开关（避免为不存在的场景加扩展点）；抑制策略用固定常量（与 `netBackoff` 语义一致）。

## Acceptance Criteria

- [x] 同一账号同一窗口内 N 次冷却：INFO「上传暂缓」恰好 1 条，其余为 Debug，且 Debug 携带窗口内序号
- [x] 窗口结束/恢复成功：恰好 1 条 INFO 汇总，含 `deferred_tasks` 与窗口秒数
- [x] 多账号互不影响；`nil` gate / `nil` Service 不 panic
- [x] 新增回归测试（gate 单测 + service 级日志断言）全过
- [x] 任务状态、错误返回、重试语义与改动前一致（无行为回归）
- [x] `go vet` / `go test` / `web type-check` / `web build` 全绿
- [x] spec `upload-task-api.md` §8 已同步抑制规则
- [x] `0.0.32` 镜像构建并推送、`git tag v0.0.32` 与 release、本地容器部署验证三连通过

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.
