# 合入A路文件夹上传链并发版0.0.15

## Goal

合入de83b46中文件夹上传链（面板拖拽+planner/dispatcher/API），手工拆store新增hunk，前后端验证，bump到0.0.15发版推送部署

## Requirements

1. 基线：子任务1已合入的 main（含 `0.0.14`）。合入集：`CloudLocalUploadPanel` 拖拽文件夹 hunk、`useUploadFolderPlanner`、`useLocalUploadDispatcher`、`useUploadFileInput`、`confirmUpload`、`FileBrowser` 1行、`types/upload.ts`、`uploadTaskTypes`、`api/upload.ts`、`local_upload.go` API hunk、`uploadTaskFormatters`、`useUploadPanelActions`、`useUploadTaskStream`、`useUploadTasks`，以及 `useUploadTaskStore` 中的纯新增 hunk（`addLocalUploadTasks` 等，手工拆）。
2. 不合入：B路（`TaskPanel` 重写、`sse/manager struct/BatchPause`、`worker`、`router batch-pause`、`migration 0022` 暂缓——若 A路 API hunk 不依赖则不合；执行中若发现依赖则单独评估后决定）。
3. 先做依赖判定：`local_upload.go` 70行 hunk 是否引用 B路符号；`folderPlanner` 引用的 store 函数是否都有定义；有缺失则补最小 hunk。
4. 验证：`vue-tsc` 0 错误、vite 构建、`go build/vet/test` 全过；UI 手工验证由用户确认，任务内列出验证清单。
5. 版本 `0.0.14 → 0.0.15`，发版推送部署流程同子任务1。

## Acceptance Criteria

- [ ] 依赖判定结论书面记录，合入集完整且可编译
- [ ] `vue-tsc/go build/vet/test` 全过
- [ ] `v0.0.15` 三标签可拉取，`github/main` 同步，容器健康且为新镜像
- [ ] 待用户 UI 验证项已书面列出

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.
