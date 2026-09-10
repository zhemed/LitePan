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
