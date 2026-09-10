# implement.md — 执行计划（0.0.33 前端暂停交付修复）

## 顺序清单（每步完成即验证）

1. **纯函数模块**（新增 `web/src/composables/upload/uploadPausePlan.ts`）
   - `TERMINAL_UPLOAD_STATUSES` / `isTerminalUploadStatus` / `isCooldownMessage`
   - `collectRemotePauseIds(tasks, isLocalTask)`（去重、跳本地/终态/空 id）
   - `resolveUploadDisplayStatus(status, { waitingResume, cooling })`
   - 验证：`npm run type-check`
2. **A 暂停必发 HTTP**（`useUploadBatchActions.pauseUploadTask`）
   - 删除 `isQueuedRemoteResumeTask(task)` 早退分支；远程任务统一「清集合 → 乐观 patch → `pauseTask` → 回刷」
   - 验证：静态检查（该函数内除本地分支外必须存在 `uploadApi.pauseTask(`）
3. **B 批量暂停收集改为纯函数**（`handleToggleUploadTasks` 非 resume 分支）
   - `collectRemotePauseIds(...)` 收集远程 id；本地任务仍逐个本地暂停；统一清 `pendingRemoteResumeTaskIds`；单次 `batchPause` + 回刷
   - 验证：`npm run type-check`
4. **C 展示不再掩码冷却**（`uploadTaskFormatters.ts`）
   - `getUploadTaskDisplayStatus` / `getUploadTaskPhaseLabel` 改用 `resolveUploadDisplayStatus`（`cooling = isCooldownMessage(task.message)`）
   - 验证：`npm run type-check`
5. **D 断言**（`web/scripts/check-upload-memo.mjs` 追加用例）
   - 用例：跳过本地/终态/空 id；**陈旧 `paused` 与冷却 `pending` 必须入选**；去重；`resolveUploadDisplayStatus` 在 cooling 下返回服务端状态、非 cooling 下维持掩码
   - 验证：`npm run check:memo` 全 PASS
6. **spec 同步**（`.trellis/spec/backend/backend/upload-task-api.md` §8）
   - 增补「前端暂停交付契约」：远程任务暂停必发请求；批量暂停只排除终态；冷却消息不被掩码；`isCooldownMessage` 与服务端文案耦合
7. **质量门**
   - `cd web && npm run type-check && npm run build`
   - `GOWORK=off GOTOOLCHAIN=local go vet ./...`、`go test ./...`
   - `git diff --name-only` 必须**不含** `internal/`、`drivers/`（PRD-E）
8. **版本与发布**
   - `README.md`、`docker-compose.yml` → `v0.0.33`
   - `docker build -t …:0.0.33 -t …:v0.0.33 -t …:latest .`；推送 ×3
   - `git tag v0.0.33` + 推送 + `gh release create v0.0.33 --repo zhemed/LitePan`
   - 本地容器重建（`docker rm -f litepan` + 原参数 `docker run`）→ 健康/登录/任务列表三连
9. **收尾**：`skill trellis-check` → `flow_gate mark-check` → `flow_gate pre-archive` → `task.py archive --skip-branch-validation` → `add_session.py` → `git push github main`

## 验证命令汇总

```bash
cd web && npm run type-check && npm run check:memo && npm run build
GOWORK=off GOTOOLCHAIN=local go vet ./... && GOWORK=off GOTOOLCHAIN=local go test ./...
git diff --name-only            # 期望仅 web/ + README + docker-compose + .trellis/spec
./.trellis/scripts/flow_gate.py pre-start|mark-check|pre-archive 09-10-fix-pause-not-delivered-frontend
```

## 人工演练（本地实例，可选但推荐）

1. 打开任务面板 → 找到一个 `pending` 且消息含「冷却」的行（可临时用脚本造数据或等真实冷却）；
2. 点该行主按钮（暂停）→ 期望：行立刻「已暂停」，且本次操作**确实**发出 `POST /api/files/upload/tasks/{id}/pause`（浏览器网络面板）；
3. 30 秒内不出现新的「上传暂缓/已恢复」日志对（任务没有再尝试上传）。

## 回滚点

- 步骤 2/3/4 各自独立可回退；整体回滚 = `git revert` 该次提交（无数据迁移、无配置变更）。

## 范围纪律

- **不做**：后端改动、并发/节流参数、冷却日志抑制逻辑（0.0.32 已完成）、删除/重传链路、`resumeMode` 语义。
- 任何超出上述文件范围的改动，先停下来向用户说明。
