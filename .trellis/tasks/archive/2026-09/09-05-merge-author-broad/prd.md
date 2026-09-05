# 合入作者B-road代码

## Goal

一次合入上游 `de83b46` 剩余 B-road 部分（B1 后端增量 + B2 批量操作 + B3 任务树展示），发 `0.0.17`；万级阶梯实测暂不做（用户确认），sessionStorage 加固等优化另行立项

## Requirements

1. 后端直接合（整文件 PASS）：`sse/lifecycle/delete/progress/queue/state/api-upload(api/upload.go)/sse_test` + `manager` mh_02 + `worker` wh_00。
2. 后端手工：manager struct（mh_00）+ init（mh_01）、worker `broadcast(taskID)` 单行、`router` batch-pause 单行、`manager_test` 对齐；跳过 `CreateServerLocalTasks` 透传（本地零调用方）与 hydration 重复行（0.0.16 已等价合入）。
3. 前端直接合：`uploadTaskTree`（新）、`useUploadBatchActions`、`api/upload.ts`、测试脚本。
4. 前端手工：TaskPanel 以本地 681 行为基重写（保留跨盘/离线删除，叠加批量折叠渲染）、store `@69`（去排序+orderMap）、taskStream（含补 `hasActiveTransferTasks` 小函数）、`useUploadTasks` 加 toggle 导出单行；`panelActions` 整 hunk 排除（中继引用本地无）。
5. 验证：`go build/vet/test`、`vue-tsc`、vite 构建全过；B2 接口（batch-pause/delete）冒烟；UI 深测留待真实业务（用户确认不做万级实测）。
6. 版本 `0.0.16 → 0.0.17`，三标签推 GHCR，`github/main` 同步，本地部署健康。

## Acceptance Criteria

- [ ] B1：增量广播生效且旧无参调用兼容（`go test ./internal/upload/...` 过）
- [ ] B2：batch-pause/delete 接口可用（冒烟）
- [ ] B3：面板按批折叠显示正常（小批量冒烟：3 文件测试 batch）
- [ ] `v0.0.17` 三标签可拉取，`github/main` 同步，容器健康且为新镜像

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.
