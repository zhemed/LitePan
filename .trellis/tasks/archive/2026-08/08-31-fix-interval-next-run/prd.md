# 修复间隔轮询首次次日问题

## Goal

修复 `service_schedule.go:154 computeIntervalStartRunAt` 的首次调度仅首档导致次日，改为按 `start_time + n*interval` 当天递进，并发布 `0.0.5`。

## Background

- 审计报告：`00:05 起每1h` 在 `now=14:00` 时返回 `次日00:05` 而非 `14:05/15:05` 当天，`advanceIntervalRunAt` 正确但 `compute` 错误。
- 影响创建后首次 `NextRunAt` 落次日，用户感知“为什么不是当天”。

## Requirements

- **修复**：`computeIntervalStartRunAt` 中 `anchor=当天00:05` 若 `After(base)` 返回，否则 `candidate=anchor; for candidate<=base {candidate+=interval; if !sameDay break}` 返回同日下一档，否则次日 `00:05`。
- **测试**：补 `service_test.go` 用例 `00:05 每1h at 14:00 → 14:05 或 15:05` 等，与现有 `13:00/01:00` 用例共存。
- **版本**：`README v0.0.4→v0.0.5`，`docker 0.0.5/v0.0.5/latest`，`git tag v0.0.5`。

## Constraints

- 仅改 `service_schedule.go` 与 `service_test.go`，`internal/cache` 等保留。
- 遵循 `AGENTS.md`：`task.py start` 后执行。

## Acceptance Criteria

- [ ] `computeIntervalStartRunAt` 在 `00:05 每1h at 12:00 → 13:05`，`at 00:00 → 00:05`，`at 23:10 → 次日00:05`
- [ ] `GOWORK=off go vet ./... 0` `go test -run TestComputeNextRunInterval -v`
- [ ] `docker build 118MB` `README v0.0.5` `health 200`

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.
