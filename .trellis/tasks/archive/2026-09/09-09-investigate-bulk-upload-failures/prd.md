# 09-09-investigate-bulk-upload-failures

## Goal

调查 2000 文件批次（`/app/mounts/LitePan-123/bulk-test-2000`）的上传失败原因。只调查不改码。

## 取证结论（详见 research.md）

**天翼云盘上游 S3 网关瞬时超时（HTTP 511 S3ClientException "Read timed out"）**，窗口约 1 分钟（21:50:31 / 21:51:03 两例），共 2/66 失败（~3%）。失败发生在 init/commit 类步骤（错误无"上传分片"前缀）；重试分类器 `retryableUploadURLFailure` 只认传输层错误（与上游完全一致，非本方引入），HTTP 5xx 业务错误被判不可重试 → 一次 511 直接判死。批次整体健康：64 成功 / 5 进行 / 1929 排队继续跑；2 个失败任务可在 UI 重新上传。

修复建议（可选另建任务）：上传路径（init/commit/getMultiUploadURLs）对 HTTP 5xx/429 纳入重试分类。

## Acceptance Criteria

- [x] 失败子集取证（错误/阶段/时间窗/占比）
- [x] 重试链路核实（分片 PUT 3 次+状态码可重试 vs init/commit 无状态码重试）
- [x] 与上游实现对照（同一局限）
- [x] 结论 + 建议
- [ ] 归档 + journal + push
