# c7a424c 认证修复安全合入评估报告

评估时间：2026-09-05（UTC+8）· 子任务1（父任务：调查剩余上游提交能否安全合并）
上游提交：`c7a424c`（09-04《修复一系列认证问题》，65 files，+2143/−236）

## 结论

**可以合入，但必须按排除清单逐文件挑拣，不可整提交 cherry-pick。** 风险等级：**中**。

- 相关源文件 26 个逐文件 `apply --check` **全 PASS**，组合子集检查亦 PASS。
- 无符号冲突：`domain.CodePermissionDenied/CodeRateLimited/Errf`、`driver.ClassifyOAuthRefreshError`、`LastFailureKind` 本地均已存在且语义兼容，补丁是扩展而非重声明。
- 接口变更 `RecoverAccount() → error`：本地仅 3 处触点（接口声明/调用者/实现），补丁全覆盖。
- 行为变化真实存在（认证核心重构），合入后必须跑测试 + 真机账号验证。

## 相关 / 忽略清单

可合入（26 源文件 + 6 测试文件）：
- `drivers/115_Open/auth.go、driver.go`、`drivers/189Cloud/auth.go、auth_response.go（新）、driver.go、transport.go`
- `internal/auth/control.go（新162行，账号锁+内联刷新）、gate.go、state_machine.go、refresh_runner.go、mutex.go、retry.go、schedule_calc.go、failure_kind.go、cooldown.go、service.go`
- `internal/driver/auth_control.go（新32行）、manager.go、refresh.go`、`internal/core/driverexec/exec.go（网络熔断）`
- `internal/domain/account.go（+AuthFailureUpstream）、conn_error.go`、`internal/account/service.go`、`internal/httpx/oauth.go`、`internal/taskauth/coordinator.go`
- 测试：`control_test.go、cloud189_integration_test.go、oauth_integration_test.go、recovery_test.go、exec_test.go、state_test.go、persistence_test.go`（全 PASS）

必须排除：
- 其他驱动（123/139/百度/光鸭/OneDrive/OpenList/Quark/template）：本地已删，13 文件
- `announcement×4`、`cacheretention×2`、`strm×3`：本地已删
- `internal/app/wire_http.go`：该 hunk 只改公告接线（`announcement.New` 减参），与认证无关
- `internal/httpx/user_agent.go`：版本号字符串（`v0.5.3` 发布附带），无关
- `internal/httpx/oauth_test.go`：修改了中间提交 `8e332f3` 新建的测试文件，本地无此基（FAIL），且只是测试
- `internal/buildinfo/version.go`、`web/src/version.ts`、`README.md`、`internal/api/web/index.html`：版本/构建物噪音
- `internal/settings/service.go`：`UpdateSilent` 唯一调用方是已删的 `api/announcement.go`，合入即死代码，建议排除（或接受一行死函数，自行决定）

## 依赖分析

- 中间提交 `8e332f3`（认证刷新优化）、`1c71fec`（连接检测）、`dd4c13d`（光鸭）中，只有 `oauth_test.go` 构成硬依赖（测试文件，可丢弃）；源文件无硬依赖。
- `control.go` 仅 import 标准库 + `domain` + `driver`，无已删模块引用；装配经 `Manager.SetAuthGuards`（nil-guarded，未装配时行为不变），调用方在 `internal/auth/service.go`（子集内）。

## 语义要点（为什么值得合）

1. 115/189 刷新改走统一节流守卫：误报 `AuthExpired`（如会话建立失败、网络/503/限流）不再触发"重新扫码"，`189 transport` 区分 401/403/429。
2. `189 doRefresh` 先落库新 token 再建会话，短暂不可用仍可恢复；`Init` 非认证错误直接上报不再多换一次 Token。
3. 账号级锁：冷启动多请求不再各建实例各刷一次；`driverexec` 加网络连续失败熔断（3次/30s）。

## 合入建议（给执行任务）

1. 用排除法套用子集（`git cherry-pick -n` 后 revert 排除文件，或 `git apply` 过滤后补丁）。
2. `go build + go vet + go test ./internal/auth/... ./internal/driver/... ./drivers/115_Open/... ./drivers/189Cloud/...` 全过。
3. 真机验证 115 + 天翼账号：手动触发刷新、断网恢复、观察无"重新扫码"误报。
4. 按基线 bump 发版（本任务不执行合入）。

## 只读确认

全程 `git show/diff/apply --check/grep`，未改业务代码，未合入。工作区除任务目录外干净。
