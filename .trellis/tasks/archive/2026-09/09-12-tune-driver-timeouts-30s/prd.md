# 09-12-tune-driver-timeouts-30s

## Goal

按用户要求**统一把驱动侧超时收敛到 30 秒**：115 API 总超时 `600s → 30s`、115 上传"发完后等响应头"兜底 `60s → 30s`、189 上传同类兜底 `60s → 30s`（189 API 本就是 30s，不变）；发布 `0.0.39` 并完成本地部署验证。

## Background

- 上一版 `0.0.38` 把上传数据面改成"无总超时 + 60s 响应头兜底"，残留三处时间参数：115 API `600s`（`drivers/115_Open/driver.go:75`）、115 上传兜底 `60s`（`drivers/115_Open/upload.go` `newOSSUploadHTTPClient`）、189 上传兜底 `60s`（`drivers/189Cloud/driver.go:86`）。
- 用户 2026-09-12 明确指示：**都改成 30 秒**（"都改成30秒"）。
- 115 API 客户端覆盖面（`drivers/115_Open/`）：`rawRequest`（列表/创建/删除等全部 API）、`postPassport`（token 刷新）、`ossDo`（OSS init/complete/list/上传凭证刷新）。把 600s 降到 30s 后，**115 服务端处理较慢的调用（典型：大文件 `completeMultipartUpload` 合并分片）可能由"能等到"变成"30s 超时失败"**——这是本次显式接受的取舍，已在下方记录症状与回退方式。

## Requirements

- **R1** 115 API 客户端总超时：`600 * time.Second` → `30 * time.Second`（`drivers/115_Open/driver.go`）。
- **R2** 115 上传数据面响应头兜底：`60s` → `30s`（`drivers/115_Open/upload.go: newOSSUploadHTTPClient`）。
- **R3** 189 上传数据面响应头兜底：`60s` → `30s`（`drivers/189Cloud/driver.go`）。
- **R4** 189 API 客户端保持 `30s`（已是目标值，不得改动）。
- **R5** 不改动与本项无关的定制：115 分片大小仍固定 512MB、`singlePartUploadLimit` 仍 512MB、189 分片大小与分片重试语义、账号 500ms 节流、`upload_task_concurrency`、冷却等待/熔断逻辑。
- **R6** 测试同步：更新 `drivers/115_Open/upload_retry_test.go` 中 60s 断言为 30s；新增 115 侧离线用例锁定三个值（`Init` 后：API 客户端 `Timeout == 30s`、上传客户端 `Timeout == 0`、上传客户端 `ResponseHeaderTimeout == 30s`）。189 侧仅常量调整，机制由 `internal/httpx/client_test.go` 覆盖，不新增测试（在 check 记录中说明）。
- **R7** 质量门与发版：`go vet ./...`、`go test ./...`、`go build ./...`、`cd web && npm run type-check && npm run build && npm run check:memo` 全绿；版本 `0.0.39`；镜像三 tag 同 digest；tag + release；本地容器重建 + health/登录/任务汇总三连。

## Constraints

- 用户对本机（本地容器）授权发版部署；**生产机不涉及、不触碰**。
- 只改时间参数与对应测试/注释；不做任何结构重构、不引入新的可配置项（"界面可调"若要另议）。
- 已知取舍（显式接受）：115 `completeMultipartUpload` 等慢调用在 30s 内未返回即失败；失败表现为任务在"上传完成/合并"阶段报 `context deadline exceeded (Client.Timeout exceeded ...)`。若实际频繁出现，最小回退 = 仅把 115 API 客户端调回较大值（如 120s/300s）。
- 版本规则：`0.0.1` 基线、`0.0.x` 递增、不跳 `1.0.0`。

## Acceptance Criteria

- [ ] 115 API 总超时已是 30s（`drivers/115_Open/driver.go`），且 `git diff` 未见其它定制被改动
- [x] 115 上传响应头兜底已是 30s；189 上传响应头兜底已是 30s；189 API 仍 30s
      → `drivers/115_Open/upload.go:35`、`drivers/189Cloud/driver.go:86` 均 `30*time.Second`；`189Cloud/driver.go:80` API 仍 30s（未改）
- [x] 115 侧新增/更新测试锁定三个值；`go test ./drivers/115_Open/ ./internal/httpx/` 通过
      → 新增 `TestInitConfiguresDriverTimeouts`（离线 Init 后断言 API=30s / 上传无总超时 / 上传响应头=30s）；`upload_retry_test.go` 断言同步改 30s；两包测试全绿
- [x] `go vet ./...`、`go test ./...`、`go build ./...` 全绿；web 三连（type-check/build/check:memo）全绿
      → 全部通过（`go test ./...` 含 `drivers/115_Open 5.06s`、`internal/httpx 0.21s`；web `MEMO-ALL-PASS`）
- [x] 版本号 `0.0.39` 已更新（`README.md` + `docker-compose.yml`）
      → README 2 处 + docker-compose 1 处
- [x] 镜像 `ghcr.io/zhemed/litepan:0.0.39`（含 `:v0.0.39`、`:latest`）三 tag 同 digest 已推送
      → 三 tag 同 digest `sha256:1f2b2bb4e24af552f68eef5c3c17054643d3d2a3cd9b005d80c4a3c0fa3c4a1d`（ImageID `d7039ebbde26`）
- [x] `git tag v0.0.39` 已推送，GitHub release `v0.0.39` 已创建
      → commit `7243a27`；tag 最终指向 `7243a27`（GitHub API 复核 `.object.sha = 7243a2799a3f40dd…`）；release https://github.com/zhemed/LitePan/releases/tag/v0.0.39
      → ⚠️ **过程中出过一次顺序错误并已修正**：本次把 `gh release create` 放在了 `git push main` **之前**，GitHub 自动打出的 `v0.0.39` 指向旧的 main 头 `86e52ca`；发现后删除远端 tag 并用本地正确 tag 重推，现已指向修复提交（见 journal 复盘）
- [x] 本地容器已重建到 0.0.39，health / form 表单登录 / 上传任务汇总三连通过
      → ImageID `d7039ebbde26`、`Restarts=0`；health ok；登录 ok；任务汇总 `total=13 success=13`
- [x] 已知取舍（115 慢 API 30s 失败的症状与回退方式）已记录在 PRD/Journal
      → 见本 PRD「Constraints」与 journal（Session 126）

## Notes

- 本任务为参数收敛（`scope=lightweight`），无新增接口/契约。
- **发版顺序教训（本任务实证）**：必须 `git push main` → `git tag` → `git push <tag>` → **最后** `gh release create`。把 release 放前面会让 GitHub 用当时的 main 头自动创建 tag，导致 tag 指向错误提交（本次已发生并修正）。
- **189 侧未新增测试的说明**：`drivers/189Cloud` 的超时值是 `Init` 内联常量，且其 `Init` 会发起会话请求（无法离线构造），故 189 侧只做常量调整，机制由 `internal/httpx/client_test.go`（流式客户端语义）与 115 侧同构用例覆盖。
- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
