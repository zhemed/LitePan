# 报告：上传任务重试次数与重试规则

**任务**：`09-03-investigate-upload-retry`
**时间**：2026-09-03
**范围**：`/root/LitePan` 本地工作区（`internal/upload` + `drivers/*` + `internal/settings` + `web`）
**结论先行**：**任务级无自动重试次数/规则（失败即停，需手动点“重试”）；重试能力分散在三处：① 驱动分片级微重试（2~3次，写死不可配）② 认证过期被动刷新重试1次 ③ 前端手动“重试”= 后端 `Resume` 断点续传。无全局 `maxRetries` 配置项。**

---

## 一、任务级（`internal/upload`）：无自动重试

| 文件:行 | 证据 | 说明 |
|---|---|---|
| `internal/upload/worker.go:79-110` | `if err != nil { if ctx.Err()!=nil {pause/cancel} ; if shouldResetResumeState {StatusFailed} ; m.failTask() }` | 单次 `runLocalUpload` 失败即 `Failed`，**无 for 循环、无计数、无 sleep/退避** |
| `internal/upload/state.go:36-42` | `func (m *Manager) failTask { Status=Failed, Message="上传失败", Error=translateError }` | 终态落库，无自动重入队列 |
| `internal/upload/queue.go:117-184` | `acquireRunSlot` / `runTask` 仅 `executeUpload` 一次，`releaseSlot` 后结束 | 队列只做并发限流（`limit=defaultLimit=3`，见 `types.go:12` + `queue.go:19-32 RefreshConcurrencyLimit` 读 `settings.KeyUploadTaskConcurrency`），不做失败重调度 |
| `internal/upload/lifecycle.go:47-128` | `func Resume(ctx,taskID)` 仅允许 `Paused/Failed/Canceled` → `Pending` + `resumePriority=true` + `go runTask` | **手动重试唯一入口**，保留 `resumeData` 做断点续传；`os.Stat(localPath)` 缺失则 `markMissingLocalFileFailed` |
| `internal/upload/resume.go:9,51-65` | `resumePersistDebounce=2s`，`scheduleResumePersist` | 仅断点状态落库防抖，非重试 |
| `internal/api/upload.go:170-182` | `pauseUploadTask → uploads.Pause`，`resumeUploadTask → uploads.Resume` | 后端 API 仅暂停/继续，无自动重试参数 |
| `grep -R retry\|backoff internal/upload` | 仅 `manager_test.go:524 TestOfflineHandoffBatch...AfterRetry`（测试名含 Retry，逻辑为手动 `Resume`） | **无 `maxRetries/attempt` 字段**；`taskState`（`types.go:80-93`）无重试计数器 |

**判定**：任务级 **0 自动重试**。`translateError`（`progress.go:82-97`）仅做文案映射（含“请点击重试后从头上传”提示），不触发重试。

## 二、驱动分片级：有写死微重试（不可配）

| 位置 | 规则 | 可重试条件 |
|---|---|---|
| `drivers/189Cloud/upload.go:225-247 getMultiUploadURLs` | `for attempt 0..2` 共 **3次**，`attempt>0` 时 `delay=attempt*300ms`（300ms/600ms），`ctx.Done` 可中断 | 仅 `retryableUploadURLFailure`：`url.Error/net.Error/EOF/UnexpectedEOF`，且非 `sessionExpired`，见 `257-267` |
| `drivers/189Cloud/upload.go:280-293 retryDelay` | `delay=attempt*300ms` | 同上 |
| `drivers/189Cloud/upload.go:611-630 putUploadPart` | `maxAttempts=3`，失败且 `retryable==true` 才继续，最后 `annotateUploadPartRetry(lastErr,2)` 拼“（已自动重试2次）” | `putUploadPartOnce:659-676`：`url/net/EOF` 或 HTTP `408/429/5xx` 或 XML `InternalError/RequestTimeout/SlowDown` 才 `retryable=true`，否则直接返回 |
| `drivers/189Cloud/upload.go:719-728 ResolveTransferHash fetchFileInfo` | `for attempt 0..1` 共 **2次**，无 delay，失败 `continue` | 取 MD5 辅助流程 |
| `drivers/115_Open/upload.go:395-443 confirmUploadedFile` | `uploadConfirmAttempts=3`（`591`），间隔 `uploadConfirmDelay=250ms`（`592`） | 上传后确认文件存在（`GetFileInfo`+`ListFiles` 轮询），`ctx.Done` 中断；`pendingResult` 兜底仍算成功 |
| `drivers/115_Open/ops.go:159-187` | 回收站删除轮询 `for attempt 0..7` 共 **8次**，`attempt<4?300ms:800ms` | 非上传主路径（删除确认），`ctx.Done` 中断 |
| `drivers/*/upload.go` 其他 | `115_Open` 主分片上传无 `for attempt`（单次，靠 `189` 侧重试）；`LocalFs/template` 无重试 | — |

**判定**：分片/确认级有 **2~3次写死重试 + 300ms线性退避**，无指数退避、无抖动、无全局配置，错误分类明确（网络类可退，`sessionExpired/4xx业务错` 不退）。

## 三、认证/传输层：被动刷新重试1次

| 位置 | 规则 |
|---|---|
| `internal/auth/retry.go:13-24 WithRetry` | `err→ if !IsAuthError return err`，否则 `gate.HandlePassiveError` 后 **再执行 `fn()` 1次**；即认证过期最多 **1+1=2次调用** |
| `internal/core/driverexec/exec.go:29-43 Run` | 所有驱动调用经 `auth.WithRetry`，`IsNetworkError → resetTransport`（仅重置传输，不重试） |
| `drivers/template/transport.go:43-60 apiCall` | `401 CodeAuthExpired → doRefresh → rawRequest` **重试1次**（模板驱动示例，115/189各自实现类似） |
| `internal/driver/refresh.go:13 RefreshRetryable` | 仅认证刷新状态机用（网络超时可重试），与上传任务重试无关 |

## 四、配置项：无重试次数键

`grep retry|attempt internal/settings` → **0命中**。仅 `KeyUploadTaskConcurrency`（`queue.go:19-32`，默认 `defaultLimit=3`，见 `types.go:12`）控制并发槽位，与重试无关。

## 五、前端：手动“重试”按钮（无自动）

| 位置 | 证据 |
|---|---|
| `web/src/components/upload/TaskPanel.vue:396` | `retryable = status==="failed" \|\| status==="canceled"` |
| `web/src/components/upload/TaskPanel.vue:421` | `actionLabel: completed?"打开":retryable?"重试":undefined` |
| `web/src/composables/upload/useUploadBatchActions.ts:209-221 handleUploadTaskPrimaryAction` | `failed/paused/canceled → resumeUploadTask(task)`；`pending/running → pause` |
| `web/src/composables/upload/useUploadBatchActions.ts:125-150 resumeUploadTask` | 本地任务改 `pending` + 调度器；远端任务 `patch pending + startUploadTaskScheduler`（经 `stream.fetchUploadTasks` 刷新） |
| `web/src/api/upload.ts:38-45` | `pauseTask POST /:id/pause`，`resumeTask POST /:id/resume` 对应后端 `lifecycle.go` |
| `internal/upload/progress.go:91-93 translateError` | 文案含“请点击重试…/清理磁盘后重试”，印证手动语义 |

**判定**：前端无轮询自动重试，只有失败/取消态展示“重试”并调 `Resume`（断点续传：`resumedProgress` + `ResumeState`，见 `worker.go:19-21`）。

## 六、结论与建议

- **有无**：任务级自动重试：**无**；分片级自动微重试：**有（189:3次+300ms线性，天翼分片/取URL；115确认:3次+250ms；115删除轮询:8次）**；认证刷新重试：**有（1次）**；手动重试：**有（前后端 Resume + 断点续传）**。
- **次数/规则总表**：见二、三节；均写死在代码，无 `settings` 可配项，无指数退避。
- **建议（若需增强，需另开任务）**：① 在 `taskState` 加 `attemptCount/maxAttempts` + `failTask` 内按可重试错误（`IsNetworkError`）自动 `Pending` 重入，需评估与 `local_upload` 批量建任务的幂等性；② 或保持现状（当前语义清晰：小错驱动层消化，大错转人工点重试+断点续传）。

### 附：复现 grep

```bash
grep -rn "retry\|Retry\|backoff" internal/upload --include="*.go" | head
grep -rn "for attempt" drivers/189Cloud/upload.go drivers/115_Open/upload.go drivers/115_Open/ops.go
grep -rn "WithRetry\|RefreshRetryable" internal/auth internal/core/driverexec internal/driver
grep -rn "retryable\|Resume" web/src/components/upload/TaskPanel.vue | head
grep -rn "retry\|attempt" internal/settings --include="*.go"
```
