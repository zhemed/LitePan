# optimize-payload-and-batch-actions

## Goal

实施优化报告 ①前半 + ②（用户拍板）：载荷瘦身 + 前端批量操作去 N-patch + 徽标单遍计数。`0.0.26` 发布。

## 实测结果

| 指标 | 改前 | 改后 | 变化 |
|---|---|---|---|
| 列表 API 载荷 | 5,408,334 B | **4,556,545 B** | **-852KB / -15.8%** |
| SSE 订阅首帧 | 5,408,286 B | **4,556,497 B** | -852KB |
| `cleanup_local` 字段 | 外发 ~490KB | 不外发 | 前端零引用，安全 |
| `result` 键 | file_id/file_name/size/parent_id | **仅 file_id/parent_id** | 去重复键 |

## 改动清单

1. `Task.CleanupLocalMode/CleanupLocalPath` → `json:"-"`（DB 列与内部逻辑不受影响）
2. `snapshot()` 投影剔除 result 内 `file_name`/`size`（持久化保留原始 result）
3. 前端徽标：最多 4 次全量 filter（7814 条）→ 单遍状态计数
4. 前端批量暂停：逐个响应式 patch（1620 规模卡主线程）→ 单次 batchPause + 一次刷新

## Acceptance Criteria

- [x] vet/全模块测试/类型检查/构建全绿（含载荷瘦身断言测试）
- [x] 0.0.26 三 tag + release + 部署（digest 1476ff2d）
- [x] 实测载荷下降 15.8%
- [ ] 归档 + journal + push
