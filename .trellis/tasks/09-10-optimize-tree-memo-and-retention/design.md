# 设计

## ① 前端记忆化

**行缓存**（TaskPanel 内）：
```ts
const rowCache = new Map<string, { updatedAt: number; row: PanelRow }>();
const ROW_CACHE_LIMIT = 6000;
function buildUploadRowCached(task: UploadTask): PanelRow
```
- key：`task_id`；命中条件：缓存 `updatedAt === task.updated_at`
- 超限时按插入序批量淘汰（简单 FIFO，避免引入 LRU 复杂度）
- `watch(uploadTasks, …)` 全量替换时清空（`replaceRemoteUploadTasks` 语义）

**批次节点缓存**：
```ts
const batchNodeCache = new Map<string, { sig: string; node: UploadTaskTreeNode }>();
```
- 每批签名：`${count}|${maxUpdatedAt}`（一次 O(n) 轻量扫描，不建对象）
- 命中则复用节点，未命中重建该批次节点

**验证**：导出 `__treeMemoStats`（build 计数）供测试断言；Node 侧用 esbuild 编译纯逻辑 + 造 5000 任务跑两轮，断言第二轮无重建。

## ② 保留策略

**设置项**（`internal/settings/registry.go`）：
| key | 默认 | 范围 | 说明 |
|---|---|---|---|
| `upload_success_retention_days` | 30 | 1-3650 | 成功类记录保留天数 |
| `upload_success_retention_max` | 0 | 0-100000 | 成功类记录最大保留条数（0=不限） |

**Manager**：
```go
type RetentionConfig struct{ Days, Max int }
// Options 增加 Retention func() RetentionConfig（默认零值=不清理）

func (m *Manager) pruneRetainedTasks(cfg RetentionConfig) int   // 纯执行
func selectRetentionVictims(tasks []Task, cfg RetentionConfig, now time.Time) []string // 纯函数，便于单测
```
选择规则（顺序）：
1. 候选 = `status ∈ {success, skipped, canceled}`（非终态一律排除）
2. 若 `Max > 0` 且候选数 > Max：按 `UpdatedAt` 降序保留最新 Max 条，其余标记删除
3. 若 `Days > 0`：`UpdatedAt < now - Days*24h` 且不在保留集 → 删除
4. `Days<=0 && Max<=0` → 不清理

**删除动作**：内存移除 + DB 行删除（store repo）+ `broadcastDeleted`（SSE 通知前端移除）。

**循环**：`go m.retentionLoop(runCtx)`：启动后立即执行一次，之后每小时；`ctx.Done()` 退出。

**配置注入**：app 装配处用 settings 服务构造闭包（`func() RetentionConfig`）。
