# 09-09-adapt-upload-batching

## Goal

移植上游 `de83b46` 上传批次化（文件夹显示为单条批次任务、任务树分层查看、大批量渲染优化）。按调查报告三步走、独立成车、不夹带其它移植。走 `0.0.18` 发布管线。

## 三步走

1. **迁移**：`internal/store/migrations/0023_upload_task_batches.sql`（同上游 0022 DDL：`batch_id`/`batch_name` 列 + 索引；上游编号与我们冲突，用本方下一个号）。
2. **后端**：`internal/upload/{types,manager,delete,lifecycle,sse,persist,progress,queue,state,worker}.go`、`internal/domain/upload_task.go`、`internal/store/upload_task_repo.go`、测试（manager_test/sse_test/store_test）。本方改过的 `manager.go`(-89)/`worker.go`(-423) 手工调和：去 offline 挂钩、保批次逻辑。
3. **前端**：`TaskPanel.vue`、`composables/upload/*`（含新 `uploadTaskTree.ts`）、`confirmUpload.ts`、`types/upload.ts`、`upload-task-panel.css`、`FileBrowser.vue`(+1)。本方同批文件删过 705 行（offline UI），冲突逐个调和。

## Constraints

- 只取 de83b46 单提交补丁（`git show de83b46 -- <path> | git apply -3`），不带入后续 ce0992b/374affd 等改动。
- 不恢复 offline_download 相关任何代码（保持精简）。
- 版本规则 0.0.17 → 0.0.18。

## Acceptance Criteria

- [ ] 迁移 0023 随启动执行，upload_tasks 新列就位
- [ ] 批次化单测（de83b46 自带 manager_test/sse_test/store_test）全绿
- [ ] vet 全绿 + web type-check/build 通过
- [ ] 0.0.18 三 tag 推送 + tag/release + 容器运行 + health/登录/列表三连
- [ ] 归档 + journal + push
