# 09-09-fix-bulk-upload-throughput

## Goal

按调查报告 `09-09-investigate-bulk-upload-freeze` 修复大批量上传吞吐塌陷。**明确不动用户设置的 `upload_task_concurrency=1`（用户确认有意为之）**。走 `0.0.19` 发布管线。

## Scope

1. **目录缓存 TTL 30s → 10min**（`target_dir.go: uploadTargetCacheTTL`）：消掉长跑中同一目录的反复重 List。
2. **批次目录预解析**：批次任务创建后，后台按去重+字典序预解析全部唯一 rel_dir 前缀（走既有 `ensureUploadTargetDir` 缓存与账号间隔门），使上传 worker 不再边传边解析；同批次防重复预热。
3. **批量暂停补 updated_at**：暂停路径持久化时写当前时间戳（修复 745 行零值外观 bug）。
4. 单测：TTL 语义、预解析预热后 List 零新增、暂停持久化带时间戳。

## Constraints

- 不改 `upload_task_concurrency`（configs 与默认值都不动）。
- 不改 500ms 间隔门（防限流设计保留）。
- 可选项 #5（虚拟滚动）不在本任务。

## Acceptance Criteria

- [ ] vet 全绿 + `go test ./...` 零失败（含新增单测）
- [ ] 0.0.19 三 tag 推送 + tag/release + 容器运行 + health/登录/列表三连
- [ ] 归档 + journal + push
