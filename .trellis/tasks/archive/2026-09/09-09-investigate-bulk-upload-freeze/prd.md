# 09-09-investigate-bulk-upload-freeze

## Goal

调查 0.0.18 大批量上传"卡死"原因：取证（容器日志、upload_tasks 表、广播/SSE 路径、间隔门语义、目录解析缓存），定位根因并给出修复建议。本任务只调查不改码。

## 取证结论（详见 research.md）

**不是卡死，是吞吐塌陷**。791 文件批次（xwechat_files，仅 40MB）在 5 分钟只完成 46 个（~6%），用户观感"卡死"后手动暂停（745 paused）。后端零错误日志、管道在暂停时仍在推进（21:12 那分钟还有 10 个成功）。

根因分层：
1. **主因：`upload_task_concurrency = 1`**（configs 实测值）——单并发串行。
2. **放大器：189 账号级 500ms 操作间隔门**（`defaultOperationDelayMS=500`，`DelayController.wait` 持锁串行）——账号级天花板 2 请求/秒，所有云操作共享。
3. **放大器：目录解析缓存 TTL 仅 30s**（`uploadTargetCacheTTL`）——长跑中重复 List 数百个微信哈希目录。
4. 数据面已豁免（分片 PUT 走 uploadClient 不过门 ✓）。
5. 次要发现：pause 路径不写 updated_at（外观 bug）；progress=100 未 finalize 的任务是队列等待被暂停，非 bug。

修复建议（待用户确认后另建任务）：并发调 2-4、目录缓存 TTL 延长、批次预解析目录。

## Acceptance Criteria

- [x] 取证：日志/表/广播/间隔门/缓存 全链路证据
- [x] 根因结论 + 量化模型（吞吐曲线与门限吻合）
- [x] 修复建议分级
- [ ] 归档 + journal + push
