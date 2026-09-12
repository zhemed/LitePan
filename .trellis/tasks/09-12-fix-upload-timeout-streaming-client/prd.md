# 09-12-fix-upload-timeout-streaming-client

## Goal

把「上传数据面」与「API 面」的 HTTP 客户端解耦：新增 `httpx.NewStreamingClient`（**不设整段传输总超时**，只设 `ResponseHeaderTimeout`），让 189 / 115 的上传 PUT 走专用客户端，并给 115 的分片 PUT 补上瞬时故障重试；发布 `0.0.38` 并完成本地部署验证。上游依据：`Ponphil/LitePan` `869974b`（v0.5.5 官方条目「修复上传超时问题」）。

## Background

- **现状（本方代码事实，2026-09-12 核对）**
  - 115：只有一个客户端 `drivers/115_Open/driver.go:74` = `httpx.NewClient({Timeout: 600 * time.Second})`，同时服务 API 调用与数据面 PUT（`drivers/115_Open/upload.go:1024` 单请求上传、`:1138` 分片上传）。`http.Client.Timeout` 语义是**整个请求的总时长**（建连 + 发完 body + 读完响应），不是空闲超时。
  - 115 分片策略为本方定制：`singlePartUploadLimit = 512MB`（`transport.go:46`）、`calculateOSSPartSize()` 固定返回 512MB（`upload.go:744`，用户 2026-08-26 定制）。⇒ 单个 512MB 分片必须在 600s 内传完（≥约 7.2 Mbps 持续上行），否则会被客户端在传输中途掐断；而当前 115 分片**没有网络/5xx 重试**，只有一次「凭证过期刷新重试」（`upload.go:1295-1303`）。
  - 189：`drivers/189Cloud/driver.go:82-88` 的 `uploadClient` = `{Timeout: 300s, DisableCompression: true, DisableKeepAlives: true}`，数据面 PUT 经 `httpx.Execute(d.uploadClient, ...)`（`upload.go:701`）；分片本身**已有重试**（`putUploadPartOnce` 返回 retryable，408/429/5xx/传输层错误）。
- **上游修复（`869974b`）**
  1. `internal/httpx/client.go` 新增 `NewStreamingClient(base *http.Client, responseHeaderTimeout time.Duration) *http.Client`：克隆 `base` 的 transport（沿用连接池/代理/压缩配置），**不设 `http.Client.Timeout`**，仅设 `ResponseHeaderTimeout`；
  2. 115 增加 `uploadClient`（`newOSSUploadHTTPClient`）+ `ossUploadHTTPClient()`，数据面 PUT 改用它；新增 `ossUploadPartWithRetry`（3 次，1s/2s 线性退避）与 `isRetryableOSSUploadError`（net error / `unexpected eof` / `connection reset` / `broken pipe` / HTTP 429/500/502/503/504；凭证错误不重试）；
  3. 189 的 `uploadClient` 改为 `httpx.NewStreamingClient(d.client, 60*time.Second)`。
- **记录更正（上一任务 `09-12-investigate-upstream-updates` 的表述错误）**：该报告 §6 P1 行写「顺带修正 115 分片误用 **30s** 总超时客户端」——**与我方代码不符**：30s 是上游（及我方 `0.0.3` 之前）的取值，我方自 `4d8e868`（2026-08-31，0.0.3）起 115 为 **600s**。本任务按实证口径执行与记录，并在 PRD/journal 留存更正说明。
- 该改动只影响**上传路径**；189 的 500ms 账号级节流、`upload_task_concurrency`、115 分片大小与 API 客户端超时均为既有定制，不在本任务范围内。

## Requirements

- **R1 流式上传客户端**：在 `internal/httpx` 新增 `NewStreamingClient(base *http.Client, responseHeaderTimeout time.Duration) *http.Client`——克隆 base 的 `*http.Transport`（base 为 nil 或非 `*http.Transport` 时回退 `http.DefaultTransport` 克隆），`responseHeaderTimeout > 0` 时设置 `ResponseHeaderTimeout`，**不设置 `Timeout`**。
- **R2 115 数据面专用客户端**：115 驱动新增 `uploadClient` 字段，`Init` 中按需构造（`newOSSUploadHTTPClient(d.client)` → `httpx.NewStreamingClient(base, 60s)`），`Drop` 中关闭（`httpx.CloseClient`）；`ossSinglePartUpload` 与 `ossUploadPart` 的 PUT 改用该客户端（保留 `d.client` 作为 API 客户端，**其 600s 超时不改**）。
- **R3 115 分片瞬时故障重试**：新增 `ossUploadPartWithRetry`（上限 3 次尝试）与 `isRetryableOSSUploadError`；重试前 `Seek(uploadedOffset)` 复位；退避 1s/2s 且可被 `ctx` 取消；**凭证类错误不在此重试**（交由既有 `isOSSCredentialError` 刷新路径）；重试不得造成上传进度重复累加。
- **R4 189 上传客户端流式化**：`uploadClient` 改为 `httpx.NewStreamingClient(d.client, 60*time.Second)`（去掉 300s 总超时与 `DisableKeepAlives`，启用连接复用）。不改 189 分片大小、重试语义与账号节流。
- **R5 保留本方定制（不得改动）**：115 `singlePartUploadLimit` / 固定 512MB 分片；115 API 客户端 600s；189 `retryableUploadURLFailure` 分类与 `maxAttempts`；`upload_task_concurrency`；任何 gate/节流参数。
- **R6 测试**：
  - `internal/httpx/client_test.go`（新）：`NewStreamingClient` → 无总超时、继承 base transport 配置（如 `Proxy`/`DisableCompression`）、`ResponseHeaderTimeout` 生效、nil base 回退默认 transport、`CloseClient` 可用。
  - `drivers/115_Open/upload_retry_test.go`（新）：`isRetryableOSSUploadError` 表驱动；`ossUploadPartWithRetry` 行为（假 transport：前 N 次可重试失败 → 最终成功；达到上限返回最后错误；不可重试错误立即返回；成功时不重试）。
- **R7 记录更正**：本任务 PRD/design/journal 明确记录「09-12 调查报告 P1 的 30s 表述有误，实为 600s；问题实质＝API/数据面共用客户端 + 缺分片重试」，避免错误结论留在归档记录里无人纠正。
- **R8 质量门与发版**：`go vet ./...`、`go test ./...`、`go build ./...`、`cd web && npm run type-check && npm run build && npm run check:memo` 全绿；版本 bump `0.0.38`（`README.md`、`docker-compose.yml`）；构建并推送镜像三 tag（同 digest）；`git tag v0.0.38` + GitHub release；本地容器重建并完成 health / 登录 / 任务汇总三连验证。

## Constraints

- **不触碰生产机 `10.0.0.11`**（本轮无生产授权；生产升级另行确认）。
- **不扩张范围**：不做 115 分片大小/并发策略调整，不改 API 客户端超时，不在前端做任何源码改动；若发现必须跨出边界的问题，先报告再定。
- 移植以上游 `869974b` 为准；与本方定制冲突处（600s、512MB 分片）**保留本方定制**，差异写入 `design.md`「适配差异」表。
- 失败不降级：测试或验证失败即停下报告，不通过放宽断言/关掉测试绕过。
- 版本规则：`0.0.1` 基线、`0.0.x` 递增、**不跳 `1.0.0`**；本任务 bump 到 `0.0.38`。

## Acceptance Criteria

- [ ] `internal/httpx.NewStreamingClient` 已实现且 `internal/httpx` 单测覆盖（无总超时 / transport 继承 / 响应头超时 / nil base），`go test ./internal/httpx/` 通过
- [ ] 189 上传客户端已改为流式（无总超时 + 60s 响应头超时），`go test ./drivers/189Cloud/...` 通过
- [ ] 115 数据面（单请求 + 分片）已使用 `uploadClient`，分片具备 3 次瞬时故障重试（含分类表测与假 transport 行为测试），`go test ./drivers/115_Open/...` 通过
- [ ] 未改动 R5 所列本方定制（`git diff` 逐行核对：115 超时仍 600s、分片仍 512MB、189 节流/重试未动）
- [ ] `go vet ./...`、`go test ./...`、`go build ./...` 全绿；`cd web && npm run type-check && npm run build && npm run check:memo` 全绿
- [ ] 版本号 `0.0.38` 已更新（`README.md` + `docker-compose.yml`）
- [ ] 镜像 `ghcr.io/zhemed/litepan:0.0.38`（含 `:v0.0.38`、`:latest`）三 tag 同 digest 已推送
- [ ] `git tag v0.0.38` 已推送，GitHub release `v0.0.38` 已创建
- [ ] 本地容器已重建到 0.0.38，health / form 表单登录 / 上传任务汇总三连通过
- [ ] 记录更正（30s → 600s，问题实质）已写入本任务文档与 journal
- [ ] 上游 `869974b` 的**适配差异表**（保留项/未移植项/理由）已写入 `design.md`

## Notes

- 上游 `869974b` 同时改了 139/123/百度/光鸭/OneDrive/Quark 六个驱动的上传客户端与 `InternalExperimental` 标记 —— 本方精简分支**只保留 189/115 两个驱动**，相关改动不适用（差异表已列）。
- 数据面无总超时后的残留风险（服务端长期不返回响应体/不读 body）与缓解手段（`ResponseHeaderTimeout=60s`、任务暂停/取消携带 ctx、TCP keepalive、驱动 `Drop`）已在 `design.md` 讨论；如认为不可接受，可在实施前提出收紧方案。
- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
