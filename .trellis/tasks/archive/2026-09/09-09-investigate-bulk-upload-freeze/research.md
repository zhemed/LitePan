# 调查报告：大批量上传"卡死"根因（2026-09-09）

## 1. 现象与取证

用户 21:07 通过 UI 上传 `xwechat_files` 文件夹（微信数据目录，深树）。实测数据（upload_tasks 表）：

| 指标 | 值 |
|---|---|
| 批次 | `folder-1788959341267-txgzch`，**791 个任务**（790 带 batch_id），总计仅 ~40MB |
| 状态分布 | 46 success + 745 paused（用户手动暂停） |
| 创建时间线 | created_at 从 **21:07:25 到 21:12:04**（前端 planner 边走边建，近 5 分钟） |
| 成功吞吐 | 每分钟：21:07→1，21:10→8，21:11→27，21:12→10（**暂停时仍在推进**） |
| 后端日志 | **零错误、零限流、零 panic**；只有登录记录 |
| 特征样本 | 1 个任务 progress=100/uploaded=total 但 paused（传完 100% 等待 finalize 时被暂停） |

## 2. 根因：吞吐塌陷（非死锁/非冻结）

### 主因：`upload_task_concurrency = 1`（configs 实测）

`internal/upload/types.go` 代码默认 `defaultLimit = 3`，但库里实际值是 **1**（设置页曾设为 1）。单并发 → 一次只传一个文件，其余 745 个排队。

### 放大器 A：189 账号级 500ms 操作间隔门

`drivers/189Cloud/transport.go: defaultOperationDelayMS = 500`；`DelayController.wait`（internal/driver/delay.go）**持账号锁串行**——同一账号所有云操作（List/mkdir/init/commit）任意两次间隔 ≥500ms，**全账号天花板 ≈ 2 请求/秒**，3 个并发 worker 也全部汇入同一把锁。数据面已豁免：分片 PUT 走 `d.uploadClient` 直连（upload.go:657，不过门）。

单文件最小门限成本 = init(0.5s) + commit(0.5s) ≈ 1s；实测 ~6s/文件（叠加单并发排队 + 目录解析 + 哈希计算）。

### 放大器 B：目录解析缓存 TTL 仅 30 秒

`target_dir.go: uploadTargetCacheTTL = 30s`。微信树的 rel_dir 形如 `msg/attach/<32位哈希>/2026-08/Img`——**数百个唯一哈希目录**，每个新前缀 = 1 次 List（或 +1 次 CreateFolder），全部过 500ms 门。更糟：TTL 只有 30s，20 分钟级别的长跑中**同一目录会被反复重新 List**。

### 量化模型（与观测吻合）

约 791 文件 × 2 门限操作 + 数百目录 × 1-2 门限操作 ≈ **2500-3000 个门限操作 ÷ 2/s ≈ 20-25 分钟起步**，叠加单并发与缓存过期重 List → 实际 1-1.5 小时量级。5 分钟只完成 6% 完全符合该模型 → 观感"卡死"。

## 3. 排除项（证据）

- **SSE 广播层不会死锁**：慢订阅者托底是"drop-1 再发"（sse.go:119-123），`ch` 缓冲 8，不会永久阻塞；上游 sse.go 同款且后续无修复提交（374affd/ce0992b 未触及）。791 任务的全量快照大（~数百 KB）但 120ms 合并窗口 + delta 事件已把量压住。
- 后端无 panic/无错误日志；管道暂停时仍在出成功（21:12 分钟 10 个）。
- 前端 planner 建任务 2.6/s（791 个 5 分钟建完）——不受门限拖累（建任务不做云操作），建完即止，非卡点。
- progress=100 未 finalize 的任务：commit 排在单并发队列里，被暂停时还没轮到——非 bug。

## 4. 次要发现（顺带）

1. **pause 不写 updated_at**：745 个 paused 行 updated_at=0（2000-01-01），批量暂停的持久化路径没带时间戳——外观 bug，影响排序/展示。
2. 上传并发设置（`upload_task_concurrency`）在设置页可调，但 UI 上没有针对"大批量场景"的引导说明。

## 5. 修复建议（按收益排序，待确认后另建任务实施）

1. **并发调 2-4**（configs 一行或设置页；若当初设 1 是有意防限流，可保持 2 折中）——直接 ~3x。
2. **目录缓存 TTL 30s → 10min**（一行改动；批次级运行期内目录基本不变）——消掉长跑重复 List，~1.5-2x。
3. **批次预解析**：上传启动前一次性收集批次全部唯一 rel_dir 预热缓存（仍在门限内但避免边传边解析的长尾）。
4. pause 持久化补 updated_at（外观修复）。
5. （可选）UI：>200 任务的树虚拟滚动/降频渲染，改善体感。

## 6. 验证记录

- `docker logs litepan --since 60m`：零上传错误。
- `upload_tasks` 表 791 行逐项核对（状态/时间线/大小/进度）。
- `configs.upload_task_concurrency = 1` 实测。
- `delay.go`/`transport.go`/`target_dir.go`/`sse.go`/`upload.go` 源码路径逐一核实；吞吐曲线与门限模型吻合。
- 全程只读，工作区干净。
