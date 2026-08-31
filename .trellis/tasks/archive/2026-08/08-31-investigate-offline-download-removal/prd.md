# 调查离线下载是否可彻底移除

## Goal

以全量 `grep` 为准，审计 `offline_download` 触发器、任务、API、前端、驱动等相关所有内容是否可安全彻底移除，并评估与 `local_upload` 的关联。

## Background

- 当前 `LitePan v0.0.5` 的 `automation` 仍保留 `offline_download` 触发器（`domain.AutomationTriggerOfflineDownload`），且 `internal/offlinedownload`、`internal/domain/offline_download`、`web` 离线下载入口可能仍存。
- 用户已将自动化动作精简至 `local_upload`，需判断离线下载是否亦可移除。

## Requirements

- **全量扫描**：`grep -R offline_download|OfflineDownload` 在 `backend/frontend` 的文件清单、行数、归属层。
- **依赖**：`domain` 触发器、`service` 调度/事件、`api`、前端 `trigger_config`、驱动 `offline_download.go`、与 `local_upload` 是否耦合。
- **DB**：`configs/offline_download_tasks` 表、`automation_rules trigger_type=offline_download` 是否有存量规则。
- **产出**：`report.md` 含清单、依赖图、可否彻底移除结论与待删列表。

## Constraints

- 只读，不改代码；写仅限 `report.md`。
- 遵循 `AGENTS.md`：`task.py start` 后执行。

## Acceptance Criteria

- [ ] `report.md` 存在，含清单、依赖、可否结论与待删文件表

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.
