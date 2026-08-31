# 彻底移除跨盘下载

## Goal

按 `report.md` 8+3 清单彻底移除跨盘下载（`cross_transfer`）相关所有内容，并发布 `0.0.10`。

## Background

- 报告结论：与 `local_upload` 零耦合，可彻底移除。

## Requirements

- **后端**：`types.go SourceTypeCrossTransfer`、`worker.go executeCrossTransferDownload`、`manager.go 3分支`、`lifecycle/queue/persist` 跨盘判断、`manager_test` 4用例、`registry.go 文案`。
- **前端**：`useUploadBatchActions` `useUploadTaskStore` `TaskPanel` 跨盘分类/标签。
- **版本**：`README v0.0.9→v0.0.10`，`docker 0.0.10`。

## Constraints

- 所有写操作在 `task.py start` 后。
- `GOWORK=off go vet` `type-check` 必须 0。

## Acceptance Criteria

- [ ] `grep SourceTypeCrossTransfer --include=*.go | wc -l ==0`
- [ ] `grep 跨盘下载 --include=*.vue | wc -l ==0`
- [ ] `GOWORK=off go vet 0` `type-check 0` `docker build 105MB` `README v0.0.10`

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.
