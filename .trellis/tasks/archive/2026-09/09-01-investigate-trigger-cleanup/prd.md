# 调查触发条件是否清理干净

## Goal

以截图 `8c61b939 520x290`（`选择触发条件 每天定时/本次触发时间+间隔/第三方通知`）为靶，核查 `09-01-remove-webhook` 后 `第三方通知` 是否已彻底清理。

## Background

- `0.0.11` 已移除 `Webhook` 触发器 8+4 处，预期仅 `每天定时/按间隔` 2 项。
- 截图仍显示 3 项，疑 `GHCR 0.0.11` 未部署、`web` 缓存或 `local` 未 `build`。

## Requirements

- **代码**：`grep AutomationTriggerWebhook|webhook|第三方` 在 `main` 是否 0。
- **产物**：`internal/api/web` 是否含 `第三方` 文本，`docker images` `0.0.11` 与运行容器 `litepan` 镜像是否 `0.0.11/latest`。
- **部署**：`docker ps --format` `Image` 与 `GHCR latest digest` 是否一致。
- **产出**：`report.md` 含代码/产物/部署三表与是否清理结论。

## Constraints

- 只读，不改代码；写仅限 `report.md`。
- 遵循 `AGENTS.md`。

## Acceptance Criteria

- [ ] `report.md` 存在，含三表与结论（已/未清理）

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.
