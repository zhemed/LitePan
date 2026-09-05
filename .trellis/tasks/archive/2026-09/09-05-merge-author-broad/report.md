# 合入作者B-road并发版0.0.17 · 执行报告

执行时间：2026-09-05 · commit `74230af` · tag `v0.0.17` · GHCR `sha256:90277c5`（105MB）· 万级实测按用户确认未做

## 结论

**B1+B2+B3 一次合入完成，`0.0.17` 在线健康，B2/B3 API 级冒烟全过。** UI 深测（千级真实渲染）留待真实业务。

## 合入方式（白名单+手工拆分）

- 直接合 14 文件：sse/lifecycle/delete/progress/queue/state/api-upload/sse_test/tree/batchActions/api-upload-ts/测试脚本 + manager mh_02 + worker wh_00。
- 手工：manager struct/init、router 单行、manager 单测（补 `file` import、去重、port 3 用例）、store `@69`、stream（含 relay 适配）、tasks toggle 单行、TaskPanel（10 hunk直合 + 11 hunk以本地681行为基移植，跳过中继/离线行）。
- 跳过：CreateServerLocal 透传（零调用方）、panelActions（中继引用，本地等价）、worker updateDownloadProgress（函数本地已删，void）。
- 移植中发现并修复：单测缺 `file` import、重复测试函数；`panelActions`/`taskStream` 悬空引用已排除在外。

## 验证

- `vue-tsc` 0 错误；`go build/vet` 干净；`upload/api/store` 及全量测试过（仅基线已知 file 失败）。
- 真传冒烟（3 文件 batch）：落库 batch 全对；`batch-pause` 对终态任务返回空更新（新语义正确）；SSE 首事件 `"kind":"snapshot"`（B1 协议在线）；`batch-delete` 3/3；云目录结构一致后删入回收站；配置恢复。
- 树算法：上游自带脚本实测 **10000 任务 8.3ms**（预算 1000ms）。
- 测试痕迹全清（任务 0 残留、配置恢复、容器/宿主目录与 cookie 删除）。

## 遗留（用户已确认）

- 万级阶梯实测未做，真实业务到量级后按评估报告阶梯方案执行。
- sessionStorage 5 行加固、虚拟列表未做（另立项）。
