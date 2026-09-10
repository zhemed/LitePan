# 上传任务 API 契约（窗口化列表 / 汇总 / SSE / 批量控制 / 冷却语义）

> 覆盖版本：0.0.18（批次化 + SSE）、0.0.23（删除受理即成功）、0.0.24（冷却等待）、0.0.25（批量恢复）、0.0.27（窗口化列表与汇总）。
> 本文档为**跨层契约**（前端 ↔ API ↔ upload.Manager ↔ 驱动执行器）的权威说明；改动任一段落必须同步前端与测试。

---

## 1. Scope / Trigger

适用触发条件（满足任一即需同步本文档）：

- 新增/变更 `/api/files/upload/tasks*` 任何端点的请求或响应字段
- 变更 SSE（`/api/files/upload/tasks/stream`）的载荷结构（`kind`/`tasks`/`counts`/`deleted_task_ids`）
- 变更任务窗口语义（哪些任务默认返回）、批量控制语义、账号冷却/重试语义

## 2. Signatures

| 端点 | 方法 | 说明 |
|---|---|---|
| `/api/files/upload/tasks` | GET | 默认返回**窗口**；带 `status`/`limit`/`offset` 时按过滤分页 |
| `/api/files/upload/tasks/summary` | GET | 任务级汇总：`{total, counts}`（与窗口无关的真实总数） |
| `/api/files/upload/tasks/stream` | GET(SSE) | `snapshot`（订阅即发，窗口化 + counts）/ `delta`（脏任务 + counts） |
| `/api/files/upload/tasks/{id}/pause` \| `/resume` | POST | 单任务控制 |
| `/api/files/upload/tasks/batch-pause` \| `/batch-resume` | POST | 批量控制（`{task_ids:[...]}`），**前端批量操作必须走这里** |
| `/api/files/upload/tasks/batch-delete` | POST | `{task_ids, delete_uploaded_file, delete_batch_roots}` |

Manager 侧签名（`internal/upload`）：

```go
const DefaultTaskWindow = 500
type ListFilter struct{ Statuses []string; Limit, Offset int }
type TaskSummary struct{ Total int; Counts map[string]int }

func (m *Manager) ListFiltered(ctx context.Context, accountID int64, f ListFilter) []Task
func (m *Manager) Summary(ctx context.Context, accountID int64) TaskSummary
func (m *Manager) WindowTasks(ctx context.Context, accountID int64, window int) ([]Task, TaskSummary)
func (m *Manager) BatchResume(ctx context.Context, taskIDs []string) BatchControlResult
```

## 3. Contracts

**窗口语义（0.0.27）**：默认列表/快照 = 全部非终态任务（`pending/running/paused/failed/canceled`）+ 最近 `DefaultTaskWindow`(500) 条终态任务（`success/skipped`），排序 **`CreatedAt DESC`**（新→旧）。

**请求参数**：

| 参数 | 类型 | 约束 |
|---|---|---|
| `account_id` | int64 | >0，缺省=全部账号 |
| `status` | string | 逗号分隔；缺省=窗口语义（等价于"非终态全量 + 最近 500 已完成"） |
| `limit` | int | ≥0；0=不限（显式传入即进入过滤分页模式） |
| `offset` | int | ≥0 |

**响应字段（列表项）**：`task_id/account_id/account_name/driver_type/file_name/status/progress/uploaded_bytes/total_bytes/message/error/result/created_at/updated_at…`
- `result` 仅含 `file_id/parent_id`（`file_name`/`size` 已按 0.0.26 去重剔除）
- **清理类字段（`cleanup_local_mode`/`cleanup_local_path`）不外发**（`json:"-"`，仅服务端与 DB 使用）

**SSE 载荷**：

```json
{"kind":"snapshot","tasks":[…窗口…],"counts":{"paused":1810,"success":6004},"total":7814}
{"kind":"delta","tasks":[…脏任务…],"deleted_task_ids":[…],"counts":{…},"total":7814}
```

**冷却契约（0.0.24）**：账号网络冷却错误 = `CodeDriverError` + `Details{"account_cooldown":true,"retry_after_seconds":N}`，文案仍为「该账号网络异常，约 N 秒后自动重试」；`domain.IsNetworkError` 对该错误返回 **false**（防自指反馈）。

**工作集保留契约（0.0.28）**：设置项 `upload_retention_days`（默认 30，1-3650）与 `upload_retention_max`（默认 0=不限，0-100000）。

- 后台每小时（启动即执行一次）清理**终态成功类**记录（`success/skipped/canceled`）：超期或超出条数上限者
- **绝不**清理未完成任务；**绝不**删除本地/网盘文件（复用 `Manager.Delete`，其本地清理对 `source_type=server_local` 直接跳过）
- 前端无感：清理通过 SSE `deleted` 通知收敛列表；`summary.counts` 随之下降
- 选择逻辑为纯函数 `selectRetentionVictims(tasks, cfg, now)`，改动必须同步单测（`retention_test.go`）
- **跳过 owned 批次根任务**（`result.batch_root_owned && batch_root_id`）：这些记录是"删除云端批次根目录"完整性判据的数据基础，自动清理会削弱该保护（0.0.29）
- **批次根删除完整性契约**：批次创建时写入 `result.batch_task_total`；`BatchDelete` 删除根目录前要求「选中数 == batch_task_total」（缺失→保守拒绝）。空目录自动清理必须同时满足「显式勾选删除批次根」+「父目录确为 owned 批次根」+「forceRefresh 实时空判定」

**批量控制契约（0.0.25）**：`batch-resume` 与 `batch-pause` 同形，返回 `{updated_task_ids, missing_task_ids}`；服务端逐个走单任务语义（内部仍受并发闸门与排队约束）。

## 4. Validation & Error Matrix

| 条件 | 结果 |
|---|---|
| `limit`/`offset` 非数字或负数 | `CodeValidation` → 400「非法 limit/offset」 |
| `account_id` 非法 | `CodeValidation` → 400「非法 account_id」 |
| 账号处于网络冷却 | 任务/请求返回冷却错误（**客户端应等待 `retry_after_seconds` 后重试**；worker 内部自动等待） |
| 会话/Token 失效 | `CodeAuthExpired`（不重试、走刷新链） |
| 189 大目录删除超确认窗口 | **视为已受理成功**（0.0.23，不再误报失败） |
| 删除显式失败（冲突/`failedCount>0`） | 原样报错（不吞） |

## 5. Good / Base / Bad

- **Good**：前端打开面板 → 并行取 `tasks`（窗口 ≈1.2MB@7814 任务）+ `summary`（计数）；徽标/导航用 `counts`；"已完成"截断时提示并可一键加载全部。
- **Base**：任务数少（<500 终态）时窗口=全量，行为与旧版一致。
- **Bad**：前端对每个任务逐个调用单任务端点（1620 次 HTTP + 响应式 patch）→ 主线程卡死（0.0.25 修复）；或前端自行遍历全量统计计数（窗口化后必然失真）。

## 6. Tests Required

| 断言点 | 位置 |
|---|---|
| 窗口保留**最新**终态任务、汇总与窗口无关 | `internal/upload/window_test.go` |
| `status/limit/offset` 过滤分页语义 | 同上 |
| 一任务下载（JSON 不含 `cleanup_local_*`、`result` 去重、必要字段保留） | `internal/upload/payload_test.go` |
| 冷却错误判定/网络判定去自指 | `internal/core/driverexec/cooldown_test.go`、`drivers/189Cloud/transport_test.go` |
| 批量恢复去重/缺失/短路 | `internal/upload/breaker_test.go` |

## 7. Wrong vs Correct

**Wrong**（窗口化后仍按全量假设写前端）：

```ts
const tasks = await uploadApi.listTasks();        // 只拿到窗口
const doneCount = tasks.filter(t => t.status === "success").length;  // ❌ 少算
```

**Correct**：

```ts
const [tasks, summary] = await Promise.all([uploadApi.listTasks(), uploadApi.tasksSummary()]);
// 徽标/计数用 summary.counts（真实总数），行数据用窗口
```

**Wrong**（批量操作逐任务请求）：

```ts
for (const task of paused) { await uploadApi.resumeTask(task.task_id); }  // ❌ 1620 次风暴
```

**Correct**：

```ts
await uploadApi.batchResume(paused.map(t => t.task_id));   // ✅ 一次请求
```

## 8. 账号级冷却的运维契约（0.0.31 生产取证，2026-09-10）

**来源**：`10.0.0.11` 真实批次（822 任务 / 115 网盘）出现「大面积上传暂缓：账号网络冷却，稍后自动重试」，取证见任务记录 `09-10-investigate-11-cooldown-storm`。

1. **放大律**：冷却来自 `internal/core/driverexec/exec.go` 的账号级网络退避（连续 `netFailThreshold=3` 次网络失败 → `netBackoff=30s`）。退避窗口内 `Run()` 在入口短路返回 `CooldownError(30s)`，**不访问上游**；于是「1 次冷却 × N 个被派发任务 = N 条 `上传暂缓` INFO 日志」。生产实测：**183 条 / 0.35 秒 / 183 个不同文件 / 同一 `account_id`**。→ 看到大量「暂缓」先判断是不是**单账号单事件放大**，不要当成批量真实失败。
   **0.0.32 起已在源头抑制**：`internal/file/cooldown_log.go` 的 `cooldownLogGate` 按账号维护冷却窗口——窗口内**只留首条 INFO**，其余降 Debug（`window_seq` 标注序号），窗口结束或该账号恢复成功时补 **1 条 INFO 汇总**（`deferred_tasks` + `window_seconds`）。因此正常运维下每个冷却事件最多 2 条 INFO；若又见「暂缓」刷屏，说明抑制逻辑被绕过（例如新日志路径未走 `Service.UploadLocal`），应作为回归处理。
2. **触发源静默**：`recordNetFail` 进入/退出退避均无日志 → 生产日志里只有「后果」没有「原因」。排查顺序：`docker logs --since N h <容器> | grep -v 上传暂缓`；若除启动/认证/配置行外**无任何 WARN/ERROR**，即为冷却短路而非真实失败；此时触发源**不可归因**，应作为可观测性缺陷登记（见 §6B 建议）。
3. **状态一致性**：冷却分支（置 `pending` + 「N 秒后自动重试」）与 `pause()`（`lifecycle.go:41-76` 置 `paused` + 「上传已暂停」）并发时会写同一任务。**0.0.34 起改为原子守卫**：`Manager.beginCooldownWait(taskID, seconds)` 在 `m.mu` 内一次性判定（任务存在、未停止、`cancelMode != "pause"`、状态为 `pending|running`）并完成迁移，返回 false 时**不得改动任何字段**——此前「先 `canCooldownWait()` 判定、再 `patch()` 无条件写回」的窗口会让并发的 `pause()` 被覆盖，产生「内存 paused / 库内 pending」分叉（实测 19 条，相差 0.3ms），而 `persist.go:80-91` 启动恢复会把 `pending` 行 `go m.runTask` ⇒ 重启即自行续传。同时 `patch()` 持久化改用**值快照**（`snap := *st`），避免落库内容被并发迁移改写。改动冷却/暂停迁移时两条不变式必须保持：**暂停优先**、**内存态 == 最后落库态**。
4. **熔断分组（0.0.35 起）**：熔断安全网按 `batchKeyOf(st)` 分组——`batch_id` 非空用 `batch:<id>`；为空退化为 `acct:<account_id>|target:<target_path>`（目标目录也空则退化 `acct:<account_id>`）。计数与「挑选剩余 pending 暂停」必须使用**同一个键函数**。批次标识的产生：浏览器批量上传用 `client_task_id`（`api/local_upload.go`）；**自动化（`automation_rules` → `local_upload`）不写 `batch_id`**（0.0.35 曾写入运行级 id，因任务面板按批次折叠成一行、与用户预期不符，0.0.36 回退），其熔断保护完全依赖退化键 `acct:<id>|target:<path>`。
   **已知缺陷（登记）**：批次分组下**终态桶恒为空**——批次行状态取聚合值（`web/src/components/upload/TaskPanel.vue:563-567`：任一成员 active ⇒ active），桶过滤只看顶层行状态（同文件 `:629`），因此「已完成/失败」桶看不到批次内的终态文件（0.0.18 引入分组即存在，浏览器批量上传仍会触发；修复方向：桶内按状态下钻展开批次成员，或让批次行在各桶按成员计数出现）。
5. **验证口径**：判定「暂缓是否安全」的三条硬指标——`failed=0`、任务进度/`resume_data` 未丢、`pragma quick_check=ok`；三者通过即无需紧急处置，冷却 30 秒自愈。
6. **前端暂停交付契约（0.0.33）**：服务端对冷却等待中的任务**可即时暂停**——`pause()` 会 `cancel()` 该任务 ctx，`worker.go:107-127` 的等待 `select { case <-ctx.Done(): return false }` 立即返回。因此前端**禁止**出现「只改本地状态、不发 `pauseTask`」的暂停路径：远程任务点暂停必须调用 `POST /api/files/upload/tasks/{id}/pause` 或 `POST /api/files/upload/tasks/batch-pause`（历史缺陷：id 在 `pendingRemoteResumeTaskIds` 时静默早退 ⇒ 界面显示已暂停、服务端继续上传到冷却结束）。批量暂停的 id 收集只允许排除**终态（success/skipped）与浏览器内本地任务**，不得按本地乐观状态（`pending/running`）过滤；收集后须把这些 id 从待恢复集合中移除。展示层规则：任务消息含 `冷却` 时以**服务端状态**为准（返回 `resolveUploadDisplayStatus` 的原状态），不得被待恢复集合掩码成 `pending`。实现位于 `web/src/composables/upload/uploadPausePlan.ts`（纯函数，断言见 `npm run check:memo`）。
