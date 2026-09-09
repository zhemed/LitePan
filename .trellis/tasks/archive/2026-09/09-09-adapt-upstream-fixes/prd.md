# 09-09-adapt-upstream-fixes

## Goal

按本方精简后代码（0.0.12）适配合并上游 `Ponphil/LitePan` 中与**保留功能**相关的修复（承接 `09-09-investigate-upstream-merge-safety` 结论 C：不整体 merge，按需移植）。移植后 bump `0.0.13` 走标准发布管线（vet/test/build/docker/tag/release/部署/健康检查）。

## Scope（移植清单）

按 merge-base→origin/main 累计 diff 取上游最终态，逐文件适配本方树：

| # | 来源提交 | 文件 | 内容 |
|---|---|---|---|
| 1 | 8e332f3 认证刷新优化 | `internal/httpx/oauth.go`(+19/-1)、`internal/api/oauth.go`(+3)、新增 `internal/httpx/oauth_test.go`(71行) | OAuth 代理状态码细分 401/400/429/403 + 显式 UA |
| 2 | 1c71fec 账号连接检测 | `internal/account/ping.go`、新增 `ping_test.go`、`drivers/115_Open/driver.go`、`web/src/composables/useOAuthAuth.ts` | 连接检测重构（剔除 Baidu_Open 部分） |
| 3 | c7a424c 认证修复子集 | `drivers/189Cloud/{auth.go,auth_response.go(新),driver.go,transport.go}`、`internal/account/service.go`、`drivers/115_Open/{auth.go,transport.go}` | 189Cloud 认证响应解析重构 |
| 4 | 353b830 上传目录错位 | `web/src/components/file/CloudLocalUploadPanel.vue` | 服务器上传目录错位修复 |

**明确排除**：`de83b46` 上传批次化（与本方精简 worker 语义相撞）、全部已删功能提交（STRM/跨盘/整理/emby/AI/光鸭/115离线）、`drivers/115_Open/offline_download.go`（本方已删）、版本发布提交 2eb66b4。

## Requirements

1. 逐文件核对上游最终态与本方现状；本方自 merge-base 未改的文件直接取上游最终版，本方改过的文件（`115_Open/driver.go`、`api/oauth.go`）手工合并。
2. 上游新代码引用的 domain 错误码（`CodeRateLimited`/`CodePermissionDenied` 等）在本方缺失时补齐。
3. 不引入对本方已删模块的任何 import/路由/UI。
4. 质量门：`GOWORK=off go vet ./...` 全绿 + `go test ./internal/httpx/ ./internal/account/ ./drivers/189Cloud/ ./drivers/115_Open/` 全绿 + `web` type-check+build 通过。
5. 版本 `0.0.12 → 0.0.13`：README.md、docker-compose.yml、docker-compose.fnos.yml、web/src/version.ts。
6. 发布管线：docker build/push 双 tag+latest → `git tag v0.0.13` + `gh release create` → 重建容器 `:latest` → `/api/health` 验证。

## Acceptance Criteria

- [ ] 上述 4 组修复全部落在工作区，无已删功能回归
- [ ] vet/test/type-check/build 全绿
- [ ] `0.0.13` 镜像推送（`0.0.13`/`v0.0.13`/`latest`），容器运行 `0.0.13` 健康检查 ok
- [ ] git tag `v0.0.13` + GitHub release
- [ ] 任务归档 + journal 记录
