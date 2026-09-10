# implement-task-list-pagination-summary

## Goal

实施优化报告③（用户拍板"继续修复"）：任务列表/快照窗口化 + 服务端任务级计数 + 过滤分页。`0.0.27` 发布。

## 实测收益（7814 任务）

| 指标 | 0.0.26（瘦身后） | 0.0.27（窗口化） | 变化 |
|---|---|---|---|
| 默认列表载荷 | 4,556,545 B | **1,210,584 B** | **-73.4%** |
| SSE 订阅首帧 | 4,556,497 B | **1,210,589 B** | **-73.4%** |
| 窗口内容 | — | 非终态全量(1814) + 最近 500 条已完成 | 与快照一致 |
| 计数 | 前端全量统计 | 服务端 `{total:7814, counts:{...}}` | 窗口外历史也计入 ✓ |

（相对优化前 5.41MB 累计 **-77.6%**）

## 改动清单

**后端**
- `ListFiltered(status/limit/offset)`、`Summary(total/counts)`、`WindowTasks(非终态全量 + 最近 DefaultTaskWindow=500 条已完成)`
- `GET /files/upload/tasks` 默认返回窗口；带参数时按过滤分页
- 新 `GET /files/upload/tasks/summary`
- SSE 快照窗口化并带 counts/total；delta 亦带 counts（徽标实时真实）

**前端**
- `fetchUploadTasks` 并行取窗口 + 汇总；徽标/导航计数改用服务端真实计数
- 已完成列表截断提示 + 一键"加载全部"（`loadAllCompletedTasks`）

**测试**
- 窗口选取（保留最新成功记录、汇总与窗口无关）、过滤/分页语义；全模块零失败

## Acceptance Criteria

- [x] vet/全模块测试/web 类型检查与构建全绿
- [x] 0.0.27 三 tag + release + 部署（digest e9be6322）
- [x] 实测载荷 -73.4%（累计 -77.6%）
- [ ] 归档 + journal + push
