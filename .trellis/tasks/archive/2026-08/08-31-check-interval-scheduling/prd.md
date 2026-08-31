# 检查间隔执行时间逻辑

## Goal

排查截图 `36a465 704x149`（`00:05起每1小时`）与 `7aab37 274x154`（`下次执行 2026-09-01 00:05`）中，间隔轮询的“下次执行”为何落在次日而非当天，定位 `computeNextRun / advanceNextRun / normalizeInput` 逻辑缺陷并给出修复。

## Background

- 用户配置：`从指定时间开始按间隔轮询执行，00:05 起每 1 小时`，预期当天内每小时 00:05/01:05/... 触发，但 UI 显示 `下次执行 2026-09-01 00:05`（次日首档）。
- 当前时间推测为 `2026-08-31` 当天，`00:05` 已过，应为 `当天下一档`（如 `14:05`）而非次日 00:05。
- 需审计 `internal/automation/service_schedule.go` 与 `service.go normalizeInput` 的间隔计算，以及前端 `AutomationPanel` 的展示。

## Requirements

- **复现**：以 `now = 2026-08-31 14:00` 等为例，`computeNextRun("interval", {start_time:"00:05", interval_hours:1}, now)` 应返回当天 `14:05` 而非次日 `00:05`。
- **审计**：读取 `service_schedule.go` `advanceNextRun / computeNextRun / wallClockTimeIn` 实现，判断是否将 `当天已过锚点` 直接跳到次日锚点而非同日下一间隔。
- **产出**：`report.md` 含时间线、根因、修复方案（同日间隔递进 vs 次日重置）与验证用例。

## Constraints

- 只读审计优先；写仅限 `report.md`，修复待用户确认后另开任务。
- 遵循 `AGENTS.md`：`task.py start` 后执行。

## Acceptance Criteria

- [ ] `report.md` 存在，含复现用例与根因代码行
- [ ] 给出“当天/次日”两种预期的判定与建议修复

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.
