# 调查报告：上游未移植项（守卫接线 / 上传批次化）

基线：本方 `main=9a5d83b`（0.0.16），上游 `origin/main=374affd`，merge-base `4c160d9`。全程只读。

## 项 1：internal/auth 守卫接线（c7a424c 的一部分）——结论 B：建议移植

### 机制拆解

上游新增 `internal/auth/control.go`（~130 行），在 `NewService` 里对 `*driver.Manager` 注入两个守卫（无需改 wire_http，自装配）：

1. **`initializeDriver`**：驱动实例创建进"每账号互斥锁"——冷启动时多个并发请求不再各建实例、各自刷新（防 init 风暴）；初始化失败写入状态机（配置错误 Validation/NotFound 例外，不污染认证态）。
2. **`refreshInline`**：内联续期统一收口：
   - 同一次请求 scope 内 `sync.Once` 去重——一次请求撞 10 个 401 只换一次 Token（本方现状：每个 401 各自触发一次 oauth 刷新）
   - `recentlyRefreshed` 复用窗口：刚刷过且未到期就不再刷（并校验到期时间防"假成功"掩盖真刷新）
   - 冷却/封锁态（authBlocked）先于刷新检查
   - 失败进状态机（`recordRefreshFailure`），认证态不再"无感知"——0.0.13 那个"态机以为健康、调度排到 5 天后"的问题在守卫体系下会更早暴露
3. 配套：`driverexec` 网络熔断（连续 3 次网络失败退避 30s，避免断网时空打上游——本方在 0.0.13 验收时亲历过连环失败）、`retry.go HandleRetryFailure`（重试失败上报状态机）、`taskauth` ctx 加固（**0.0.14 已移植 ✓**）、manager 守卫挂载点（**0.0.13 已移植 ✓**，现为惰性死代码）。

### 移植面（全部来自 c7a424c 单提交，本方 internal/auth 自 merge-base 零改动）

| 类别 | 文件 | 方式 |
|---|---|---|
| 新增 | `internal/auth/control.go` | 整取 |
| 覆盖 | `auth/{gate,state_machine,refresh_runner,mutex,failure_kind,schedule_calc,retry,service}.go` | 整取（ours-clean） |
| 覆盖 | `internal/core/driverexec/exec.go` | 整取（ours-clean） |
| 测试 | `state_test/exec_test/oauth_integration_test/persistence_test`（~370 行，随 c7a424c 而来） | 整取 |
| 手改 | `internal/account/service.go`：把 0.0.13 的 void 适配还原为上游 error 签名（`RecoverAccount(ctx,id) error` + 错误检查） | 小改 |
| 无需 | wire_http / settings（上游 settings 改动是公告 UpdateSilent，与本项无关）/ buildinfo | — |

依赖已就位：`AuthRefreshError`/`ClassifyOAuthRefreshError`（0.0.13）、`CallerPassive`、`domain` 错误码与 `conn_error` final（0.0.13）、manager final（0.0.13）。预埋的 `SetAuthGuards`/`authInitialize/authRefresh` 字段将首次被激活。

### 价值/风险评估

- 价值：**高**——并发刷新风暴消除、复用窗口、认证失败可观测、断网退避；与本方 0.0.13 亲历的两个痛点（刷新风暴风险、断网连环失败）直接对症；自带 ~370 行上游测试；移植后本方与上游 divergence 进一步收窄，未来 cherry-pick 更容易。
- 风险：**中**——auth 是核心链路；但改动单元完整、ours-clean、有测试；回归风险主要在"恢复 error 签名"这一处手改（编译器强制暴露）。
- 工作量：半天级（整取 + 1 处手改 + vet/test + 发布）。

## 项 2：上传批次化 de83b46（P3）——结论 C：暂不移植

### 内容

文件夹上传显示为单条批次任务 + 任务树分层查看 + 大批量渲染优化。真实源码面 **~30 文件**（155 个 assets 自动重建不计）：

- 后端：`manager.go`（97 行改动，**与本方已删减版本冲突**——本方删了 89 行 offline 挂钩）、`delete.go`+138（批次删除）、`sse.go`+113（批次事件流）、`types/persist/queue/state/progress` 小改、`upload_task_repo`+12、迁移 0022（`batch_id/batch_name` 列 + 索引，3 行）
- 前端：~1088 行——`TaskPanel.vue`+274、`useUploadFolderPlanner`+289、新 `uploadTaskTree.ts`+109、`useUploadTaskStore`+125、`useUploadBatchActions`+70 等；**与本方精简正面相撞**：本方在同批文件里删了 705 行（offline UI），且 `useUploadTaskStore/useUploadBatchActions` 双方都改过

### 价值/风险评估

- 价值：纯 UX/规模优化（大批量上传的渲染性能与任务树），**不含正确性修复**——目录错位 bug 是 353b830，0.0.13 已移植 ✓。
- 本方场景：个人单账号、批次规模中等，痛点低。
- 风险：**高**——前后端事件契约（SSE 批次事件）必须两侧同步移植，且与本方删减面冲突需大量手工调和；一旦夹带其它改动极易重演 0.0.13 式回归。
- 若将来要做：单独建任务、分三步走（迁移 0023 加列 → 后端批次分组+事件 → 前端任务树），不与其它移植混车。

## 总建议

1. **守卫接线：建议下一轮移植（B）**——收益明确、机械面干净、自带测试、收窄 divergence。
2. **上传批次化：挂起（C）**——等真实需求（大批量上传卡顿/想开批次 UX）出现再做，且必须独立成任务。

## 验证记录

- `git log 4c160d9..origin/main -- internal/auth/` 全部命中 c7a424c 单提交。
- `git diff 4c160d9..HEAD -- internal/auth/ internal/core/driverexec/` 为空（ours-clean）。
- `git grep SetAuthGuards origin/main`：唯一装配点在 `internal/auth/service.go:77`（NewService 内），wire_http 无需改动。
- `useUploadTaskStore/useUploadBatchActions` 双方均有改动（冲突实证），本方前端净删 705 行。
- 本次调查全程只读，工作区保持干净。
