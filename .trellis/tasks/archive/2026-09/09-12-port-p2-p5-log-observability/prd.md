# 09-12-port-p2-p5-log-observability

## Goal

移植上游候选清单的 **P2 + P5**（两件都属"日志/可观测"）：
- **P2**（上游 `d545e47`）：客户端提前断开（`context.Canceled`/`DeadlineExceeded`）不再记 ERROR —— `internal/api/error_log.go` 前置守卫 + 回归测试；
- **P5**（上游 `b5c9308`）：① **慢后台接口日志**（新增 `internal/api/slow_dashboard_log.go`，路径表按本方**在线端点**裁剪 + 在 `router.go` 接线 + 测试）；② **accountprofile 后台刷新串行闸门**（`refreshGate`，避免并发刷新打爆上游）+ 移植上游串行性测试。

随后 bump `0.0.43`、发版、本地部署验证。

## Background

- 候选来源：`.trellis/tasks/archive/2026-09/09-12-investigate-upstream-updates/research.md` §6（P2 / P5）。
- **P2 现状**：`grep "Context().Err()" internal/api/error_log.go` → 空（未移植）；上游改动是在 `logAPIError` 入口加：
  ```go
  if errors.Is(r.Context().Err(), context.Canceled) || errors.Is(r.Context().Err(), context.DeadlineExceeded) { return }
  ```
  动机：上传/下载中断、页面关闭属客户端行为，不该计为服务端 ERROR（与本方 0.0.32 冷却日志抑制同属"日志噪声治理"）。
- **P5① 现状**：无 `internal/api/slow_dashboard_log.go`；上游实现＝中间件，仅对若干"后台概况类" GET 路径计时，超过 **1s** 记一条 INFO（`后台概况接口响应较慢` + method/path/duration_ms）。**适配点**：上游路径表含本方已删端点（`cache-retention/*`、`media-organize/tasks`、`strm/tasks`），须裁剪为本方在线端点。
- **P5② 现状**：`internal/accountprofile/service.go:55 refresh()` 无并发闸门；上游加了容量 1 的 `refreshGate chan struct{}`，`New()` 初始化，`refresh()` 先取闸（`parent.Done()` 时直接返回）再执行 45s 超时刷新 ⇒ 后台资料刷新**串行**，避免多账号同时打上游。

## Requirements

- **R1 P2 守卫**：`internal/api/error_log.go` 在 `logAPIError` 最前加取消/超时守卫（`errors.Is(r.Context().Err(), context.Canceled)` 或 `DeadlineExceeded` → 直接 return），import 增 `errors`；**新增回归测试**（同包 `error_log_test.go`）：取消上下文的请求不产生日志、正常上下文仍产生日志。
- **R2 P5① 慢后台日志**：新增 `internal/api/slow_dashboard_log.go`，逻辑与上游一致（阈值 `slowDashboardRequestThreshold = 1s`、仅 `GET`、命中路径表才计时、超阈值打 INFO）；**路径表按本方端点裁剪**为：`/api/admin/accounts`、`/api/admin/cache/stats`、`/api/admin/fuse/mounts`、`/api/admin/notifications`、`/api/admin/notifications/unread-count`、`/api/logs/stats`（并在代码注释说明"上游含已删端点，已裁剪"）；`router.go` 在 `attachRequestLogger` 之后接线；新增测试覆盖路径判定与"慢才记录"。
- **R3 P5② 刷新闸门**：`accountprofile.Service` 增 `refreshGate chan struct{}`（`New` 中 `make(chan struct{}, 1)`），`refresh()` 取闸（`parent.Done()` 时返回）后执行；移植上游 `service_test.go` 的串行性测试（`TestBackgroundProfileRefreshRunsSerially`）。
- **R4 质量门**：`go vet ./...`、`go test ./...`、`go build ./...`、`cd web && npm run type-check && npm run build && npm run check:memo` 全绿。
- **R5 发版**：bump `0.0.43`；镜像三 tag；顺序 **push main → tag → push tag → 最后 release**；本地容器重建 + health/登录/任务汇总三连。
- **R6 记录**：PRD 勾选附证据；journal 记录移植清单、适配差异（路径表裁剪）与验证结果。

## Constraints

- 只做 P2/P5；P3（认证收口）按用户指示**归档不实施**（另建记录任务），P4/P7/P8/P9 后续单独处理。
- 不触碰生产机；不改前端源码。
- 版本规则 `0.0.x` 递增 → `0.0.43`；门禁三关全过（pre-archive 被拒不得 archive）。

## Acceptance Criteria

- [x] P2 守卫已加且有回归测试（取消上下文不记日志 / 正常上下文仍记录）
      → `internal/api/error_log.go:46-50` 守卫（`errors.Is(ctx.Err(), Canceled/DeadlineExceeded)`）；新增 `internal/api/error_log_cancel_test.go`（3 例，`go test ./internal/api/` 通过）
- [x] P5① 慢后台日志已加：路径表为本方在线端点（含裁剪说明注释）、`router.go` 已接线、测试覆盖路径判定与阈值行为
      → 新增 `internal/api/slow_dashboard_log.go`（阈值 1s、仅 GET、6 条在线端点，注释注明上游已删端点被裁剪）；`router.go:132` 接线；新增 `slow_dashboard_log_test.go`（白名单判定 + 快速/非白名单/非 GET 不记 + 超阈值记录）
- [x] P5② refreshGate 已加，且移植的串行性测试通过（两个账号刷新不并发）
      → `internal/accountprofile/service.go` 新增 `refreshGate chan struct{}`（`New` 初始化、`refresh` 取闸、父 ctx 取消即返回）；`service_test.go` 含上游串行性测试 + 新增"取消的父上下文直接返回"用例；`go test ./internal/accountprofile/` 通过
- [x] `go vet ./...`、`go test ./...`、`go build ./...`、web 三连全绿
      → 全绿（`go test ./...` exit 0、28 个包 ok；web `MEMO-ALL-PASS`）
- [x] 版本号 `0.0.43` 已更新（`README.md` + `docker-compose.yml`）
      → README 2 处 + docker-compose 1 处
- [x] 镜像 `ghcr.io/zhemed/litepan:0.0.43` 三 tag 已推送；tag 指向提交（API 复核）；release 已创建（顺序正确）
      → 三 tag 同 digest `sha256:6fd9d63191b95aad927fe67630b1f3cc68b6a63ec638454c100a6c40b7c3d8f7`（ImageID `d64aabd93966`）；tag `v0.0.43` = `ac6ec4a`（API 复核一致）；release https://github.com/zhemed/LitePan/releases/tag/v0.0.43；顺序 push → tag → push tag → release ✔
- [x] 本地容器已重建到 0.0.43，health / form 登录 / 任务汇总三连通过
      → ImageID `d64aabd93966`、`Restarts=0`；health ok；登录 ok；任务汇总 `total=13 success=13`
- [x] 移植清单与适配差异已写入 journal
      → Session 131

## Notes

- `scope=lightweight`（两处小改动 + 测试 + 发版），无接口/DTO/DB 变更。
- 上游 `slow_dashboard_log.go` 与 `accountprofile` 改动**原样移植**，仅路径表裁剪属有意适配（已注明）。
- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
