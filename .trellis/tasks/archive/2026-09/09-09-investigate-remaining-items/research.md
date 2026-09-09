# 调查报告：暂缓项深入调查（2026-09-09 夜）

## ① 认证取消 WARN 噪音（10 条 @21:37:32）

**溯源**：`internal/driver/manager.go:177`（`injectAuth`）——10 个并发驱动实例构建时，各自的请求 ctx 同时被取消（页面关闭/中止面板预检）→ `authStates.Get(ctx)` 返回 context canceled → 逐实例告警。**单时刻、自限性、零业务影响**。

**降噪补丁可行性**：1 行——`Warn` 前加 `ctx.Err() != nil` 短路（取消属预期行为）。建议：随下次发布的顺带项，不值得单开任务。

## ② 大批量渲染天花板——**评估报告该条作废：虚拟滚动已存在**

0.0.18 移植的批次化自带窗口化渲染（de83b46 原生）：
- `TaskPanel.vue:601` `renderedRows = visibleRows.slice(virtualStart, virtualStart+virtualCount)`
- 滚动驱动 `virtualStart`，`virtualRowHeight=53px` + overscan，行外用 spacer 占位
- 实测 2000 节点全程流畅与之吻合

结论：5000-10000 规模的 DOM 压力已被窗口化吸收，**无需任何动作**。上轮评估把该项列为"不修"实为"已具备"。

## ③ 测试数据清理清单（对账完成）

| 对象 | 位置 | 量 | 清理方式 | 风险 |
|---|---|---|---|---|
| 云端测试文件 | 天翼 `/bulk-test-2000/bulk-test-2000/` | **1986 个 .bin**（实测 List 对账 = 成功数 1:1） | UI 手删 或 API 批删（破坏性，需用户确认） | 误删风险低（独立目录） |
| 本地测试文件 | `/root/LitePan/mounts/LitePan-123/bulk-test-2000` | 2.0G / 2000 文件 | 宿主 rm -rf（可随时重建） | 无 |
| 临时探针 | `/tmp/bulk_all_md5.txt`、`md5check.txt`、`lp_cookie`（会话 cookie） | KB 级 | rm（cookie 建议清） | 无 |
| 失败任务记录 | DB upload_tasks 14 行 failed | 14 行 | UI 重传（补传 1986 之外的 14 个）或删除记录 | 留作账目亦可 |

## 结论

| 项 | 处置 |
|---|---|
| ① 噪音 | 1 行降噪，随下次发布顺带 |
| ② 虚拟滚动 | 已具备，无动作（评估条目作废） |
| ③ 清理 | 四类清单就绪，等用户逐项拍板（云删需明确确认） |
