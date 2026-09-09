# fix-shared-dir-cache-and-189-res-code-retry

## Goal

实施终局报告遗留优化 2+3，随 `0.0.22` 发布：

1. **批次目录解析接入共享缓存**：`Handler.uploads` 即 `*upload.Manager` 本体（零接线成本）——Manager 新增导出方法 `ResolveUploadTargetDir`，handler 的 `ensureLocalUploadTargetDir` 改为走它（manager.targetDirCache，TTL 10min，跨请求命中，驱动无关 115 同益）；resolver 返回本调用新建的前缀集合，保持 BatchRootOwned 语义；预解析（warmTargetDirs）由此真正生效
2. **189 业务错 -1 纳入重试**：HTTP 200 + `code:"-"1`/"res_code:-1"（"服务暂时不可用"）附加 `189_business_code` 详情，`retryableUploadURLFailure` 识别 `"-1"` → 可重试（既有 3 次上限+退避不动）

## Constraints

- 不改 500ms 门/并发；不实施防重复触发（已另任务归档为决策记录）
- BatchRootOwned 语义保持：仅"本次调用新建的前缀"计入

## Acceptance Criteria

- [ ] vet 全绿 + go test ./... 零失败（含新增：缓存跨请求命中零 List、createdPrefixes 正确、-1 可重试/其它 code 不可重试）
- [ ] 0.0.22 三 tag 推送 + release + 部署三连
- [ ] 归档 + journal + push
