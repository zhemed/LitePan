# 彻底移除第三方通知Webhook

## Goal

按 `report.md` 8+4 清单彻底移除第三方通知Webhook相关所有内容，并发布 `0.0.11`。

## Background

- 报告结论：与 `local_upload` 零耦合，可彻底移除。

## Requirements

- **后端**：`domain TriggerWebhook`、`service_webhook.go`、`validate` 白名单、`service.go WebhookEvent`、`service_run` 分支、`api automationWebhook`+`router`、`service_test` 用例。
- **前端**：`api/automation.ts webhook` 类型、`AutomationPanel` 9 处第三方卡、`ApiKeySettings` 文案。
- **版本**：`README v0.0.10→v0.0.11`，`docker 0.0.11`。

## Constraints

- 所有写操作在 `task.py start` 后。
- `GOWORK=off go vet` `type-check` 必须 0。

## Acceptance Criteria

- [ ] `grep -r Webhook --include=*.go --exclude-dir=_extracted | wc -l ==0`（除 `delayTime` 兼容外）
- [ ] `grep webhook --include=*.vue | wc -l ==0`
- [ ] `GOWORK=off go vet 0` `type-check 0` `docker 105MB` `README v0.0.11`

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.
