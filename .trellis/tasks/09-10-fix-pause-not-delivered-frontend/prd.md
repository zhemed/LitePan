# 09-10-fix-pause-not-delivered-frontend

## Goal

修复前端暂停链路：**账号冷却等待中的任务点暂停不生效**（用户在生产机实测「有一个文件停留在 30 秒等待无法暂停，等冷却恢复才能完全暂停」）。让暂停请求**必定发往服务端**，并让界面展示服务端真实状态。

## Background

取证见已归档任务 `09-10-investigate-cooldown-wait-pause-delay`：

- 服务端**有**「立即暂停」能力（`worker.go:107-127` 冷却等待 `<-ctx.Done()`；`lifecycle.go:58-70` `pause()` 会 `cancel()`）；实测该任务完整跑完 30 秒等待后上传成功 ⇒ **暂停从未送达服务端**（强证据）。
- 前端两条静默失效路径（`web/src/composables/upload/useUploadBatchActions.ts`）：
  - **108-113 行**：任务 id ∈ `pendingRemoteResumeTaskIds` 时，`pauseUploadTask` **只改本地状态、不发 HTTP**；
  - **180-196 行**：批量暂停按**本地乐观状态**过滤（`!["pending","running"].includes(task.status)` 即跳过），本地状态过期就整条漏掉。
- 展示层 `uploadTaskFormatters.ts:32-34` 会把「在待恢复集合 + 已暂停」渲染成 `pending`，掩盖服务端真实状态。

## Requirements

- **A 暂停必发 HTTP**：`pauseUploadTask` 对**所有远程任务**统一走「清待恢复集合 → 乐观置暂停 → 调 `pauseTask` → 刷新列表」，删除「只改本地不发请求」的捷径；失败时保留 `silent` 语义（批量循环不弹 N 个 toast），但**列表必须回刷**以暴露真实状态。
- **B 批量暂停不依赖本地乐观状态**：批量暂停收集 id 时**只排除终态**（`success`/`skipped`）与本地（浏览器内）任务，不做 `pending/running` 过滤；收集后统一清出 `pendingRemoteResumeTaskIds`（停止客户端待恢复重试）。
- **C 展示不掩盖服务端状态**：任务的最后一条消息是冷却语义时（`冷却`），展示状态与阶段标签以**服务端状态**为准，不再显示成「等待继续」。
- **D 可断言**：把 A/B/C 的判定抽成**无依赖纯函数模块**，并在 `npm run check:memo` 断言脚本中新增用例（本地任务/终态排除、去重、陈旧 paused 与冷却任务必须入选、冷却消息不被掩码）。
- **E 不改后端**：服务端语义保持不变（`pause()` 对 `pending/running` 生效、冷却等待期间 `cancel()` 中断等待）；不改 `upload_task_concurrency`、189 节流、冷却参数。
- **F 交付**：版本递增 `0.0.33`（`fix` 级），质量门（type-check/build/go vet/go test）全绿，构建推送镜像 + tag/release + 本地部署验证三连；**不部署生产机**。

## Constraints

- 版本规则 `0.0.33`（不跳 `1.0.0`）。
- 不引入新依赖、不加配置开关；纯函数模块不得 import 运行时代码（便于断言脚本转译）。
- 前端行为变更必须同时覆盖**行内主按钮**（`handleUploadTaskPrimaryAction`）与**批量/选择**（`handleToggleUploadTasks`）两条入口。

## Acceptance Criteria

- [ ] A：远程任务暂停**总是**调用 `pauseTask`（含 id 在待恢复集合的高危场景）
- [ ] B：批量暂停按「非终态 + 远程」收集，本地乐观状态不再导致漏发；收集后清空待恢复集合
- [ ] C：冷却消息存在时展示状态/阶段标签取服务端状态（不再显示「等待继续」）
- [ ] D：`npm run check:memo` 新增断言全过（纯函数行为覆盖）
- [ ] E：后端零改动（`git diff --name-only` 不含 `internal/`、`drivers/`）
- [ ] F：`npm run type-check`、`npm run build`、`go vet ./...`、`go test ./...` 全绿
- [ ] F：`0.0.33` 镜像构建推送、`git tag v0.0.33` + release、本地容器部署验证（健康/登录/任务列表）通过
- [ ] spec 同步：`.trellis/spec/backend/backend/upload-task-api.md` §8 增补「前端暂停交付契约」

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- 本任务为**复杂任务**：另有 `design.md`（改动边界/契约/取舍）与 `implement.md`（实施顺序/验证命令/回归点）。
