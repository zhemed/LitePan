# fix-upload-retry-classification

## Goal

实施评估报告第 1 项：189 上传路径对 **HTTP 5xx/429** 纳入重试放行（init/commit/getMultiUploadURLs 等 rawJSON/rawForm 步骤的瞬时网关故障自动重试）。走 `0.0.20` 发布管线。

## 方案

1. `drivers/189Cloud/transport.go`：`rawJSON`/`rawForm` 非 200 分支与 429 分支的错误附加 `WithDetails{"http_status": code}`（结构化，不再只进文本）。
2. `retryableUploadURLFailure`：在传输层错误之外，识别 `Details["http_status"]`（兼容 int/float64）——`>=500 || ==429` → 可重试；400/403/会话失效仍不重试。既有 3 个调用点与重试上限、递增退避全部不动。
3. 分片 PUT 状态分支已自带 429/>=500 重试 ✓ 不动。
4. 单测（drivers/189Cloud/transport_test.go 扩展）：511 S3 体 → 可重试；429 → CodeRateLimited+可重试；400/403 → 不可重试；会话失效 → 不可重试。

## Constraints

- 不改重试次数与退避节奏；不改其它驱动；不碰 500ms 门与并发。

## Acceptance Criteria

- [ ] vet 全绿 + `go test ./...` 零失败（含新增用例）
- [ ] 0.0.20 三 tag 推送 + tag/release + 容器运行 + health/登录/列表三连
- [ ] 归档 + journal + push
