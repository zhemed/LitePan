# 三批修复执行记录（0.0.29）

## 批 A（高危·数据销毁）— 三重保护

| 层 | 改动 | 文件 |
|---|---|---|
| ①数据层 | 保留策略跳过 owned 批次根任务（`batch_root_owned && batch_root_id`） | `internal/upload/retention.go` |
| ②判据层 | 批次创建写入 `result.batch_task_total`；`BatchDelete` 删根前要求"选中数 == 历史总数"，缺失→保守拒绝 | `types.go`/`manager.go`/`api/local_upload.go`/`delete.go` |
| ③清理路径加固 | 空目录自动清理必须同时满足「显式勾选删批次根」+「父目录确为 owned 批次根」+「**forceRefresh 实时空判定**」（原实现：不受开关/归属限制、用可能过期的缓存列表） | `internal/upload/delete.go` |

**关键发现**：真正的风险不止显式的删根路径，还有一条**未加保护的"空父目录自动清理"**——它会因为缓存"假空"或非 owned 目录而删掉文件夹。已一并加固。

**测试（8 项全绿）**：部分选择只删文件不删目录（原复现测试转为修复断言）、完整选择仍可删根（正向）、缺 `batch_task_total` 保守拒绝、保留策略跳过 owned、既有 4 项批次删除测试补字段后全过。

## 批 B（中危·校验）

`DELETE /api/files/delete` 过滤空白 file_id；全空 → 400「请选择要删除的文件」。
**实测**：`[""]` 与 `["   "]` 均返回 400 ✓。单测 3 组通过。

## 批 C（中危·显示）

新增纯函数 `web/src/composables/upload/uploadTaskTotals.ts`：
`running=pending+running`、`paused`、`failed=failed+canceled`、`done=success+skipped`、`active=running+paused`；
徽标优先级 上传中→已暂停→失败→上传完成；store/TaskPanel 接入。
**校验脚本 7 条新断言全过**（paused 1810 场景 → 徽标"已暂停 1810"；分桶与 active 组合正确）。

## 发布与实测

- 0.0.29 三 tag（digest `8e8e5d60`）+ release + 部署三连 ✓
- B 实测：空白 ID → 400 ✓
- C 实测：summary `{failed:1, success:6005}` 分桶正确 ✓
- A 实测：存量记录 `batch_task_total` 计数 = **0**（历史批次）→ 根删除保守禁用 ✓
- 我的测试残留（`nested_probe.bin`×2、`e2e_probe.bin`）已从任务记录清理（6006 → 6003）✓
- spec 同步：`upload-task-api.md` 补充"跳过 owned 批次根"与"批次根删除完整性契约"
