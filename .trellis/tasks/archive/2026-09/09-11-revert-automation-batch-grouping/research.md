# research.md — 批次分组回退（0.0.36）

## 1. 现象与判定

用户 2026-09-11 提供两张任务面板截图（生产机 `10.0.0.11`，已升级 0.0.35）：

| 截图 | 现象 |
|---|---|
| 1 | 「进行中」只剩 **1 行**：`定时任务 定时全局备份`，来源「服务器上传」，状态 `传输中 13/819`，速度 4.3 MB/s |
| 2 | 「已完成 **13**」徽标有数字，但列表显示「暂无已完成上传任务」 |

**判定**：截图 1 = 0.0.35 为自动化运行写入批次身份后，面板按 `batch_id` 折叠（**预期外**的行为变化）；截图 2 = 由该折叠**暴露的既有缺陷**（非本次引入）。

## 2. 机理（代码定位）

| 环节 | 位置 | 说明 |
|---|---|---|
| 批次身份来源 | `internal/automation/service_run.go`（0.0.35 新增） | 写入 `auto-<ruleID>-<unix>` / `定时任务 <规则名>` |
| 折叠逻辑 | `web/src/composables/upload/uploadTaskTree.ts:54-63` | 同一 `batch_id` 的任务 → 一个文件夹节点 |
| 行状态聚合 | `web/src/components/upload/TaskPanel.vue:563-567` | 任一成员 active ⇒ 行状态 = active |
| 桶过滤 | 同文件 `:629` | `row.state === uploadStateFilter` ⇒ 批次行只进「进行中」，桶内终态文件在根层级不可见 |

⇒ 819 个任务折叠为 1 行、「已完成 13」桶为空；点进该行仍可看到逐条文件。

## 3. 用户决策

**方案 B（2026-09-11 明确选择）**：关掉自动化批次分组，恢复平铺列表。
未选择方案 A（保留分组 + 修终态桶）——该既有缺陷仍留在待办（spec §8.4 已登记，浏览器批量上传仍会触发）。

## 4. 回退实现与验证

| 项 | 证据 |
|---|---|
| 回退范围 | `internal/automation/service_run.go`：移除 `uploadBatchScope`、`executeAction/runLocalUpload` 额外参数、`CreateParams.BatchID/BatchName` |
| 形态等价 | `git diff v0.0.34 -- internal/automation/service_run.go` **无输出**（与 0.0.34 逐字一致） |
| 残留检查 | `grep -c "uploadBatchScope\|BatchID\|BatchName" internal/automation/service_run.go` = **0** |
| 熔断保留 | `internal/upload/breaker.go` 未改动；`breaker_key_test.go` 4 组用例（含空 `batch_id` 同目录触发熔断）全绿 |
| 前端零改动 | `git diff --name-only` 不含 `web/` |
| 质量门 | `go vet ./...`、`go test ./...`（30 包 ok）、`-race`（upload+automation）、`web type-check/build` 全绿 |
| 发布 | 0.0.36 镜像 `bae0eb7ec6e9…` 推送（3 tag）+ `git tag v0.0.36` + release + 本地部署三连通过 |

## 5. 存量数据（重要，未执行）

生产机当前那次运行创建的 **819 条任务仍带 `batch_id/batch_name`**，因此它们的折叠**不会**因升级 0.0.36 而消失（本版本只影响后续运行）。

恢复其平铺显示需要修改生产数据：

```sql
-- 需用户单独授权；先备份 DB，仅改这两列
UPDATE upload_tasks SET batch_id='', batch_name='' WHERE batch_id LIKE 'auto-%';
```

**风险评估**：这两列被三处逻辑读取——(a) 面板分组（目标即取消分组）；(b) 熔断分组键（清空后退化为 `acct|target`，仍受保护）；(c) 批次根目录删除/保留守卫（`internal/upload/delete.go` 的 owned-batch-root 判定，依赖 `result_json` 中的 `batch_root_owned` 元数据 + `batch_id` 一致性；清空 `batch_id` 会使该批次的「批次根」语义退化，删除守卫将按普通目录处理）。故**必须经用户确认**后再执行，且建议在该批任务全部完成后再做。

## 6. 附带观察（本地实例，非本次改动引起）

本地实例重启后队列继续运行（属设计的 `pending` 续传语义）：`success 8291 / pending 2700 / running 3 / paused 4101`，其中 **3 条失败**为上游错误——`天翼云盘 API HTTP 511 S3ClientException`、`HTTP 513 NoSuchUpload`（189 侧 S3 分片异常，与本次改动无关）。如需排查可另开任务。
