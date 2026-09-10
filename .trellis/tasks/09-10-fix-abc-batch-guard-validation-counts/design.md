# 设计：三批修复

## A. 批次根删除保护（三层）

```
第 1 层（数据层）：保留策略 selectRetentionVictims 跳过 owned 批次根任务
     → 记录不再被自动清理，"整批可见"的不变量得以维持
第 2 层（判据层）：BatchDelete 根删除前要求 len(selectedBatchItems) == batch_task_total
     → batch_task_total 由批次创建时写入每任务 result（缺省 0 时直接跳过根删除）
第 3 层（既有）：内存完整性检查 + 云端 List 名称/父目录复核
```

数据流：`api/local_upload.go`（知道本批文件总数）→ `CreateParams.BatchTaskTotal` → `manager.createTask` 写入 `result.batch_task_total` → `retainBatchRootMetadata` 在成功后被驱动结果覆盖时保留该键 → `delete.go` 读取校验。

兼容性：历史批次无该字段 → 根删除被保守禁用（用户仍可逐文件删除），符合"宁可少删不可误删"。

## B. 空 file_id 校验

`internal/api/files_ops.go: deleteFiles` 前置过滤：
```go
ids := make([]string, 0, len(req.FileIDs))
for _, id := range req.FileIDs { if s := strings.TrimSpace(id); s != "" { ids = append(ids, s) } }
if len(ids) == 0 { 400 VALIDATION("请选择要删除的文件") }
```
（`BatchDelete`/`BatchPause`/`BatchResume` 已有空值跳过；此处补齐删除入口）

## C. 计数分桶

纯函数模块 `web/src/composables/upload/uploadTaskTotals.ts`：
```ts
export interface UploadTaskTotals { running: number; paused: number; failed: number; done: number; active: number }
export function computeUploadTaskTotals(total: number, counts: Record<string, number>): UploadTaskTotals
```
- `running = pending + running`；`paused = paused`；`failed = failed + canceled`；`done = success + skipped`
- `active = running + paused`（与列表"进行中"内容一致）
- store 用它替换现有 `active = total - success - skipped`；徽标优先级 running → paused → failed → done
- 校验脚本（`web/scripts/check-upload-memo.mjs` 扩展或新增）断言：paused=1810/success=6003 时徽标为"已暂停 1810"而非"上传中 1810"
