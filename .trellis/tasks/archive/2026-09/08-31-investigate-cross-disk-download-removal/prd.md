# 调查跨盘下载是否可彻底移除

## Goal

以全量 `grep` 为准，审计 `跨盘下载`（含 `crosstransfer` 跨盘秒传、`upload cross_transfer`、前端 `跨盘下载` 面板）相关所有内容是否可安全彻底移除，并评估与 `local_upload` 的关联。

## Background

- 用户已移除 `offline_download` 与 `builtin_offline`，`跨盘下载` 为另一独立功能（`internal/crosstransfer` 已在 `08-31-remove-crosstransfer` 移除，但 `upload cross_transfer` 与前端 `跨盘下载` 面板可能仍存）。
- 需区分 `跨盘秒传`（已移除）与 `跨盘下载`（待查）是否为同一实现或不同。

## Requirements

- **全量扫描**：`grep -R 跨盘下载|cross.*transfer|crosstransfer|CrossTransfer|Cross.*Download` 在 `backend/frontend` 的文件清单。
- **依赖**：`upload SourceTypeCrossTransfer`、`api/cross_transfer`、`web 跨盘下载` 面板、`任务面板 跨盘下载` 分类、与 `local_upload` 是否耦合。
- **DB**：`upload_tasks source_type=cross_transfer` 是否有存量。
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
