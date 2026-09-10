# fix-abc-batch-guard-validation-counts

## Goal

分批修复事故调查（`09-10-investigate-security-incident`）确认的三个缺陷，每批独立提交、独立验证：

- **A（高危·数据销毁）**：保留策略削弱"云端批次根目录删除"的完整性保护
- **B（中危）**：删除接口接受空 file_id 并报成功
- **C（中危·显示）**：徽标/导航把暂停/失败计入"上传中"

发布 `0.0.29`（三批可独立回滚）。

## Requirements

### A（三重保护）
1. **保留策略跳过 owned 批次根任务**（`result.batch_root_owned==true` 且 `batch_root_id!=""`）——保护不变量的数据基础
2. **持久化完整性判据**：批次创建时记录 `batch_task_total`（每任务 result），`BatchDelete` 删除根目录前要求"选中数 == 该批历史总数"；**缺失该字段（历史批次）时保守拒绝根删除**
3. 既有内存完整性检查与云端 List 复核保留（三层独立生效）

### B
- `/api/files/delete` 过滤空白 file_id；全部为空 → 400 VALIDATION（不得再出现"删除 0 项却报成功"）

### C
- 计数分桶：`running = pending+running`、`paused`、`failed = failed+canceled`、`done = success+skipped`；徽标按 running→paused→failed→done 优先级显示；导航"进行中"= running+paused（与列表内容一致）
- 计数逻辑抽为**纯函数**，纳入前端可复跑校验脚本

## Constraints

- 不改上传吞吐/并发/间隔门；不动已发布的窗口化 API 契约（仅新增字段）
- 每批修复：独立提交 + 对应测试；A 批必须让复现测试转为"不触发根删除"

## Acceptance Criteria

- [x] A：三层保护落地（含空目录清理路径加固）；复现测试转修复断言 + 正向/保守/跳过 4 类用例全过
- [x] B：过滤+400 落地，单测 3 组 + 部署后实测（[""]、["   "] 均 400）
- [x] C：纯函数 + store/面板接入；校验脚本 7 条断言全过；type-check/build 通过
- [x] 全量门禁全绿；0.0.29 三 tag digest 8e8e5d60 + release + 部署三连
- [x] 实测：空 ID 400 ✓、summary 分桶 ✓、存量记录无 batch_task_total → 删根保守禁用 ✓
- [x] 门禁 + 归档 + journal + push
