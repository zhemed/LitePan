# 上游合入完整性审计报告

审计时间：2026-09-05 · 先 `fetch origin` 确认作者无新增（仍 20 提交，HEAD `374affd`）· 只读

## 结论

**所需项 100% 合入（12 项实证全过）；发现 1 个测试遗漏（25 行，可干净移植）；其余全部分类为有意跳过或可选。**

## 所需项（12/12 实证存在）

- `353b830` → `v0.0.13`：watch 越界保护原文在位。
- `c7a424c` 子集 → `v0.0.14`：`AuthRefreshControl`（115/189 驱动）、`SetAuthGuards`（auth 服务）、`parseRefreshResponse`（189 新文件）。
- A-road → `v0.0.15` + hydration `v0.0.16`：`0022` 迁移文件、`trimUploadBatchRoot`、`enqueueUploadFolderFiles`。
- B-road → `v0.0.17`：`broadcastDeletedTaskIDs`（sse）、`BatchPause`（lifecycle+路由）、`buildUploadTaskLevel`（TaskPanel 接线）、SSE `kind:delta`（stream）。

## 文件级交叉核对

- `c7a424c` 65 文件：30 已合，余 34 文件逐个命中跳过类（他驱动 16、已删模块 9、接线/版本噪音 4、删模块绑定的测试 2、中间提交依赖的测试 1、死代码 settings 1、userAgent 1）。
- `de83b46` 40 源码文件：39 已合（3 版_commit_并集覆盖，工具重算两次——第一次路径写法错误得空集，已纠正重算）；余 3 文件：`123_Open/upload.go`（驱动已删）、`store_test.go`（**遗漏**，见下）、`panelActions`（中继引用， intentionskip）。

## 遗漏 ×1（测试，无生产影响）

- `internal/store/store_test.go` 之 `TestUploadTaskBatchFieldsPersist`（25 行）：测 batch 列 Upsert/List 回环，正是我们合入的 0022+repo。`apply --check` 通过，`newTestStore` 本地存在。建议：单独立个小任务合入（+跑 `go test ./internal/store/`），不值得单独发版，可搭下次顺风车。

## 可选微补丁（待定夺，默认不做）

- `b2ecd58` 115 transport `430004→NotFound`（2 行，可干净合入；无 115 账号实测）。
- `8e332f3` OAuth 请求 UA 头（装饰性；已验证 OAuth 流程正常）。
- `ce0992b` dashboard 防抖刷新门（1 账号下无体感）。
- `1c71fec` 连接检测优化（被已删百度驱动卡住，拆分成本>收益；调度器日志证明现行检测正常）。

## 跳过确认（抽查复核，有意）

`dd4c13d` 光鸭、`9aa08fe` 跨盘、`906b7cb` STRM、`d08cd04` emby、`6b3b379/9e75c6b/a1162dc` 整理、`2eb66b4` 版本号、`a9458b2` 显示、`374affd` 秒传星图、`f90d890`（aiorganize 本地已无此目录）——均系已删模块或分叉版本线。
