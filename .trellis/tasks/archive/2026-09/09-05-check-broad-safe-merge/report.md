# B-road（任务树+增量SSE）安全合入调查报告

调查时间：2026-09-05 · 基线：当前 main（含 0.0.14 认证、0.0.15/0.0.16 A-road）· 只读，未改业务代码

## 结论

**值得做，但拆 B1→B2/B3 两步走；B1 可独立先合（风险低-中），B2+B3 绑一起后合（需 UI 实测）。**

- 后端 10 文件：`sse/lifecycle/delete/progress/queue/state/api-upload` 整文件 PASS；`manager` 拆 6 hunk（struct+init 需手工、createBatch-ids 可直接合、hydration 两 hunk 已在 0.0.16 等价合入、CreatServerLocal透传可合但无调用方）；`worker` 拆 2 hunk（batch-meta 可直接合、`broadcast(taskID)` 1 行手工）；`router` 1 行手工；`manager_test` 需手工对齐。
- 前端：`uploadTaskTree`（新）、`useUploadBatchActions`（70行）符号齐备可合；`taskStream` 缺 `hasActiveTransferTasks` 不可直接合；`panelActions` 引用已删中继不可合；`TaskPanel` 274 行重写需以本地 681 行精简版为基手工移植；`store` 仅剩 `@69` 排序 hunk（B 展示配套）；`useUploadTasks` 剩 toggle 单行（随 B2）。
- PASS≠可编译已逐项排除：`activeRelayCount`（panelActions）、`hasActiveTransferTasks`（taskStream）本地不存在。

## 原子组

- **B1·后端增量组（先合）**：`sse.go` + manager struct/init/createBatch-ids + worker 两处 + progress/queue/state 单行 + `sse_test` + `manager_test` 手工对齐。关键性质：`broadcast(taskIDs...)` 为 variadic，旧无参调用全兼容；无 IDs 时仍全量快照——**合入后旧前端行为零变化**，纯粹铺好增量管道。
- **B2·批量操作组（随 B3）**：lifecycle `BatchPause` + `upload.go` handler + router 单行 + `delete.go` 删整批 + `api/upload.ts` + `useUploadBatchActions` + toggle 导出行。后端单独合无害，但按钮在 TaskPanel 里，须等 B3 才有意义。
- **B3·展示组（随 B2）**：TaskPanel 手工重写 + `uploadTaskTree` + store `@69` + taskStream（需补 `hasActiveTransferTasks`，逻辑本地已有等价内联，可抽小函数）。
- **跳过**：`CreateServerLocalTasks` 透传（本地零调用方）、`panelActions`（本地等价）、测试脚本可选。

## 收益评估（为什么值得）

- 现状实测推导：面板打开时每 400ms 全量 `listTasks`（全字段含 result_json）+ 全量替换 + 全行重渲染（无虚拟化，`visibleRows` 只是搜索过滤）；后端每次进度 tick 全量 SSE 快照。
- A-road 刚启用的文件夹上传一次可产生数百上千任务——正是上游做批次化的原始动机（"万级文件夹上传"）。B1 把高频路径从"全量"降为"增量"，B3 把列表从"全行"折叠为"按批"，对症。
- 反之：若用户日常任务量 <100，体感差异小，可暂缓 B2/B3，只合 B1（便宜且无感）。

## 建议执行顺序

1. B1（1 个任务：移植+`go test`+发版，无需 UI 测试）。
2. B2+B3（1 个任务：移植+`vue-tsc`+ 真机 1000+ 任务 UI 实测 + 发版）。
