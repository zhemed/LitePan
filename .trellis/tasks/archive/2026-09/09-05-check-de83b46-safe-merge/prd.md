# 评估de83b46上传批次化能否安全合入

## Goal

比对上游de83b46上传任务批次化（migration 0022、SSE、任务树）与本地上传链路，判断与精简方向是否冲突，给出合入建议

## Requirements

1. 还原 `de83b46` 完整改动清单，分类为：DB 层（migration `0022_upload_task_batches.sql`、`upload_task_repo`）、后端上传链路（`manager/lifecycle/delete/persist/sse/queue/worker`、`api/local_upload.go`、`api/upload.go`）、前端任务树（`useUploadFolderPlanner`、`uploadTaskTree`、`TaskPanel` 等）、拖拽文件夹（`CloudLocalUploadPanel` 已在本地，`353b830` 已合入）。
2. 逐类比对本地现状：本地上传链路（`local_upload` 唯一入口、`UploadTaskSettingsPanel` 精简）是否与批次化语义冲突；migration `0022` 在本地是否已存在或冲突。
3. 评估合入成本：可直接 cherry-pick / 需大改 / 建议放弃，并说明 DB 迁移风险与前端冲突点。
4. 只读评估，不合入、不改业务代码、不跑 migration。

## Acceptance Criteria

- [ ] 已分类列出该提交改动清单
- [ ] migration `0022` 在本地的状态结论明确（已存在/缺失/冲突）
- [ ] 本地上传链路冲突点逐条列出
- [ ] 给出合入建议与风险等级，全程未改业务代码

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.
