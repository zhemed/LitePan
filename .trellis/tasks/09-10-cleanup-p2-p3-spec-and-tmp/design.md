# 设计：P2/P3 清理与契约沉淀

## 1. DB 备份归档

| 现状 | 目标 |
|---|---|
| `data/litepan.db.bak.1788077861`（229K，2026-08-30 16:17，疑似 0022 迁移前快照） | `data/backups/legacy-20260830-before-0022.db` |

原则：**归档不删除**（229K 成本极低，保留可追溯性）。命名规范：`<用途>-<日期>-<上下文>.db`，与本目录既有 `manual-pre-0022-20260909-202635.db`、`manual-pre-p1-20260910-192849.db` 一致。

## 2. 契约 spec（7 段式）

文件：`.trellis/spec/backend/backend/upload-task-api.md`
覆盖对象（本会话引入的跨层契约）：

| 契约 | 引入版本 | 关键点 |
|---|---|---|
| 任务列表窗口 | 0.0.27 | 默认返回「非终态全量 + 最近 500 条已完成」；`status/limit/offset` 显式控制 |
| 汇总端点 | 0.0.27 | `GET /files/upload/tasks/summary` → `{total,counts}` |
| SSE 载荷 | 0.0.18/0.0.27 | `snapshot`（窗口化+counts）/`delta`（脏任务+counts） |
| 批量控制 | 0.0.25 | `batch-resume` 与 `batch-pause` 同形；避免前端逐任务风暴 |
| 冷却等待语义 | 0.0.24 | 冷却错误带 `account_cooldown`/`retry_after_seconds`；worker 退回 pending 等待，不判死 |
| 受理即成功 | 0.0.23 | 189 删除确认窗口 5s，超时视为已受理 |

七段：Scope/Trigger、Signatures、Contracts、Validation & Error Matrix、Good/Base/Bad、Tests Required、Wrong vs Correct。

## 3. flow_gate 短名支持

`find_task_dir` 增加回退：`tasks_root.glob("*-<name>")` 唯一命中则采用；多个命中报歧义并要求全名。保持既有全名路径优先、archive 搜索不变。

## 4. /tmp 清理边界

仅删除本会话（2026-09-09 18:00 之后）由我创建的探针/中间文件，模式清单显式列出；
非本会话文件（`gh*.json`、`hf*.json`、`cfg*.json`、`i.md`、`idx.json`、`dsh-*`）一律保留，并在报告中说明未处置原因。
