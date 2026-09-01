# 调查第三方通知Webhook是否可移除

## Goal

以 `自动联动 → 触发条件 → 第三方通知`（`外部程序调用Webhook接口通知LitePan`）为靶，审计其相关所有内容是否可安全彻底移除，并评估与 `local_upload` 的关联。

## Background

- 触发条件现 4 项：`每天/按间隔/第三方通知/Webhook` 等，`第三方通知` 对应 `AutomationTriggerWebhook`，供外部程序 `POST /api/.../webhook` 触发。
- 用户已将动作精简至 `local_upload`，触发器中 `offline_download` 已移除，需判断 `webhook` 是否亦可移除。

## Requirements

- **全量扫描**：`grep -R webhook|Webhook|第三方通知` 在 `backend/frontend` 的文件清单。
- **依赖**：`domain.TriggerWebhook`、`api/webhook`、`web trigger_config`、`apikey` 鉴权、与 `local_upload` 是否耦合。
- **DB**：`automation_rules trigger_type=webhook` 存量。
- **产出**：`report.md` 含清单、依赖图、可否结论与待删列表。

## Constraints

- 只读，不改代码；写仅限 `report.md`。
- 遵循 `AGENTS.md`：`task.py start` 后执行。

## Acceptance Criteria

- [ ] `report.md` 存在，含清单、依赖、可否结论与待删文件表

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.
