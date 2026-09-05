# 合入c7a424c认证修复并发版0.0.14 · 执行报告

执行时间：2026-09-05 · commit `55cc27b` · tag `v0.0.14` · GHCR `sha256:d8c1f75`（105MB）

## 结论

**已合入并发布 `0.0.14`，容器运行新镜像且健康。** 待用户真机验证（115+天翼刷新/断网恢复无误报）。

## 合入方式

白名单补丁（32文件）`git apply --index`，执行中追加剔除 2 个测试（`oauth_integration_test.go` 引已删 123/百度/OneDrive 驱动，`persistence_test.go` 引已删 cacheretention+strm，vet 报错）→ **实合 30 文件**（23 改 + 7 新），与 PRD 排除清单一致，另含本次 vet 发现的 2 测试排除。

## 验证

- `go build ./...` 0 错误；`go vet`（9 包）干净（剔除后）。
- 定向 `go test`：auth/driver/115/189/driverexec/account/httpx/taskauth 全 ok。
- 全量 `go test ./...`：仅 `internal/file` 中文数字集号 2 用例失败，经 `git worktree + HEAD` 验证为**基线已存在失败**，与本次合入无关（未碰该路径）。
- 容器 `latest=3836d416`，`/api/health` ok，认证调度器启动正常（天翼账号有效，下次检查 09-07）。

## 发版

- README/compose `v0.0.13→v0.0.14`；`0.0.14/v0.0.14/latest` 已推 GHCR；`55cc27b + tag v0.0.14` 已推 `github/main`；远近同步。
- 前端零变更，`internal/api/web` 未重建。

## 待用户真机验证

1. 115 与天翼账号各手动触发一次认证刷新成功。
2. 短断网/限流后观察：任务重试走冷却、不弹"重新扫码"误报。
3. 次日认证调度器自动检查无异常日志。
