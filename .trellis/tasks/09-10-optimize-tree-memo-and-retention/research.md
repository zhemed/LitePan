# 实施记录：批次树记忆化 + 上传记录保留策略（0.0.28）

## ① 前端记忆化

新增 `web/src/composables/upload/uploadRowMemo.ts`：
- `createRowMemo`：行按 `task_id + updated_at` 复用，FIFO 批量淘汰（默认上限 6000）
- `createNodeMemo` + `nodeSignature`：批次/文件夹节点按「条目数 | 最大 updated_at | 首任务标识 | 状态分布(active/failed/done) | 名称」签名复用（上限 2000）
- TaskPanel 接入：树节点与单任务行都走记忆化；`onUnmounted` 清理

**验证（`npm run check:memo`，新增可复跑脚本 `web/scripts/check-upload-memo.mjs`，用 TypeScript 编译器转译后 Node 断言）9/9 通过**：

| 断言 | 结果 |
|---|---|
| 5000 任务首轮全建 | builds=5000 |
| 第二轮未变化 | **零重建**（builds 仍 5000）+ 5000 命中 |
| 3 个任务变化 | 恰好 +3 次重建 |
| 上限保护（limit=10 插 20） | size ≤ 10 |
| 节点签名未变 | 复用（builds=1, hits=1） |
| 状态/时间变化 | 重建（builds=2） |
| 文件夹内任一任务状态变化 | 签名变化（会被重建） |

## ② 上传记录保留策略

- 设置项（设置页 schema 驱动，分类 system）：`upload_retention_days`（默认 30）与 `upload_retention_max`（默认 0=不限）
- `internal/upload/retention.go`：`RetentionConfig` 读取 → 纯函数 `selectRetentionVictims` → `pruneRetainedTasks`（复用 `Manager.Delete`，仅清记录）→ `retentionLoop`（启动即一次 + 每小时，随 runCtx 退出）
- 安全边界：仅 `success/skipped/canceled`；非终态永不清理；本地/网盘文件不动（server_local 源跳过本地清理）
- 单测 5 组：按天/按量/组合/禁用/边界 —— 全通过

## ③ 端到端实测（设置的动态生效）

| 步骤 | 结果 |
|---|---|
| 备份 DB | `data/backups/manual-pre-retention-test-19*.db` |
| 设 `upload_retention_max=6003`（当时 6004 条） | 设置 API 生效值确认 |
| 重启触发启动清扫 | 日志 `上传记录保留策略清理完成 removed=1 retention_days=30 retention_max=6003`；DB **6004 → 6003**（精确清理最旧 1 条） |
| 恢复默认 | `upload_retention_max=0`、`upload_retention_days=30` 已复原 |

## 部署与回归

- 0.0.28 三 tag（digest `2bdcb0a9`）+ release + 部署三连通过
- 设置 API 实测：`/api/admin/settings` 可见两个新键（默认 30 / 0，中文标签）
- 默认配置下启动清扫 **未误删**（6004 条完好）
- spec 同步：`upload-task-api.md` 增补「工作集保留契约」段
