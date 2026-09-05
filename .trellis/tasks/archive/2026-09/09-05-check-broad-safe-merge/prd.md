# 调查B-road任务树与增量SSE能否安全合入

## Goal

基于当前main重算de83b46剩余B-road部分（TaskPanel重写/SSE增量/manager-struct/BatchPause/router行等）的可移植性、原子组与成本，给出合入暂缓放弃结论

## Requirements

1. 以当前 `main`（含 0.0.14 认证、0.0.15/0.0.16 A-road）为基，对 B-road 文件逐个 `apply --check`：后端（`sse/manager-struct+BatchPause/worker/lifecycle-BatchPause/delete/progress/queue/state/router行/upload.go-handler/CreateServerLocalTasks透传`）、前端（`TaskPanel重写/uploadTaskTree/useUploadBatchActions/taskStream/panelActions/api-upload/useUploadTasks-toggle行/store-@69/manager_test/测试脚本`）。
2. 划分原子组（不可拆错的组合）并标出组间依赖；评估每组移植量（直接合/手工拆/重写）与回归面。
3. 评估收益：大批量任务渲染性能问题在当前本地是否存在（任务量/轮询频率实测或代码推导），给出 值得做 / 暂缓 / 放弃 的明确建议。
4. 只读调查，不合入、不改业务代码。

## Acceptance Criteria

- [ ] B-road 文件逐个可移植性结论（PASS/FAIL + 原因）
- [ ] 原子组划分与组内文件清单
- [ ] 收益评估与最终建议（合入/暂缓/放弃，分组给出）
- [ ] 全程只读，业务代码零改动

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.
