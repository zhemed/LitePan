# 评估万级文件上传链路瓶颈并给出方案

## Goal

针对单次1-2万文件上传，量化当前+A-road链路的前后端瓶颈，确定B-road是否够用，列出额外加固项与配置建议

## Requirements

1. 以单次 10000〜20000 文件（多层目录）为负载，沿链路逐段量化：浏览器（File 对象内存、dispatcher payload、store 列表、TaskPanel 渲染、400ms 轮询）、传输（local_upload API 分批、batchSize、并发）、服务端（任务创建、DB 写、persist 频率、SSE 广播量）、上游 189（建目录/上传 API 次数、限流、并发上限）。
2. 明确 B-road（B1/B2/B3）能解决哪些段、解决到什么量级；找出 B-road 之外的残留瓶颈并分级（致命/严重/优化）。
3. 给出可执行建议：必须做的合入、配置调优（并发/轮询/分批）、需要新写的加固（虚拟列表？服务端分页？上传前校验？）、以及验证方法（先用 1000/5000/10000/20000 阶梯实测）。
4. 只读调查，可读代码、跑本地无损压测推导；不动业务代码，不动线上数据，不实际上传万级文件到云盘。

## Acceptance Criteria

- [ ] 链路分段瓶颈清单（含数量级估算与代码依据）
- [ ] B-road 覆盖度结论（够/不够，差在哪）
- [ ] 分级建议（必做/调优/暂缓）与阶梯验证方案
- [ ] 全程只读，业务代码与线上数据零改动

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.
