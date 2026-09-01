# 间隔小时改间隔分钟

## Goal

将 `选择触发条件 → 本次触发时间 + 间隔` 的 `间隔小时 72` 改为 `间隔分钟`，完成前后端、校验、调度与文案统一，构建并部署验证。

## Background

- 截图 `ff2f5b 512x284`（`首次触发时间 请选择时间` `间隔小时 72`）需改为分钟 granularity。
- 当前 `interval_hours 1-365*24` 对 `local_upload` 72 小时（3 天）等场景不够精细，分钟更灵活。

## Requirements

- **后端**：`service_validate.go` `interval_hours` → `interval_minutes`（或兼容），`service_schedule.go` `interval*60`→`interval_minutes`，`api` 不变仅字段名。
- **前端**：`AutomationPanel.vue` `首次触发时间` 不变，`间隔小时` 文案/字段 → `间隔分钟`，`trigger_config interval_hours → interval_minutes`，校验 `1-525600`（365*1440）。
- **迁移**：旧 `interval_hours` 若存在自动 `*60` 兼容或 `normalizeInput` 转换。
- **版本**：`0.0.11→0.0.12`，`docker 0.0.12`。

## Constraints

- 仅改间隔单位，不改 `daily/webhook` 逻辑；`task.py start` 后执行。

## Acceptance Criteria

- [ ] `grep interval_hours` 仅兼容代码或 0，`interval_minutes` 生效
- [ ] 前端显示 `间隔分钟`，`72` 小时对应 `4320` 分钟可输
- [ ] `GOWORK=off go vet 0` `type-check 0` `docker 105MB` `health 200`

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.
