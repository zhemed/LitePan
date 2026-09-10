# 调查报告：UI 删除后文件夹需强制刷新才消失（2026-09-10 08:32 实证）

## 结论

**189 批量删除是异步任务；我们等它完成的轮询上限 30 秒，大文件夹（bulk-test-2000 = 2000 文件）删不完就超时报错——但 189 实际会继续删完。** UI 收到 502 报错 → 保留条目；稍后强制刷新时 189 已删完 → 文件夹消失。**删除从未失败，只是"确认等待"超时被当成了失败。**

## 实证链（08:32:31，正好对上用户操作）

1. `08:32:31 WARN 删除文件失败 count=1 err="DRIVER_ERROR: 等待批量任务完成超时"` + `ERROR API 请求失败 method=DELETE path=/api/files/delete status=502`（原始日志，时间吻合）
2. 驱动实现（drivers/189Cloud/ops.go:189/197）：`createBatchTask(DELETE)` 受理后 `waitBatchTask` 轮询（300ms 间隔，**30s 上限**）；DeleteMode=delete 还会追加清回收站批任务（**再等 40s**）→ 大删除总确认窗口 70s
3. `waitBatchTask` 超时 → `等待批量任务完成超时` → file service 视为失败返回（缓存失效+mutation 都跳过）→ 前端 `useFileActions` catch → toast 报错、行保留 ✓（前端逻辑本身正确：成功时会本地移除）
4. 189 侧异步批任务随后完成 → 云端实际已删（实测 root 列表无 bulk 残留 ✓）

## 次要因素

- 删除成功路径的缓存失效是有的（`DeleteFiles` 里 `InvalidateDirKeys`，parent_id 前端有传）——本例根因不在缓存层
- 189 索引 eventual consistency：超大目录删完后立即刷新仍可能短暂见到（189 侧 lag，非我方可解）

## 修复建议（未实施，待拍板）

**"受理即成功"语义**：`createBatchTask` 受理成功后，`waitBatchTask` 缩短为快速确认窗口（如 3-5s）；超时**不再报错**，返回成功 + Info 日志"删除已受理，云端异步完成"；显式失败（taskStatus=2 冲突 / failedCount>0）仍报错。UI 随即移除条目，体验与真实状态一致。
