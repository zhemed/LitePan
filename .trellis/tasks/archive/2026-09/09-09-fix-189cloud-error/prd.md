# 09-09-fix-189cloud-error

## Goal

0.0.13 验收不通过：天翼云盘(189Cloud)请求报 `DRIVER_ERROR: 天翼云盘 API HTTP 400: {"res_message":"error in login get unifyAccountInfo is null","res_code":"UserInvalidOpenToken"}`，且认证调度器按健康态排到 7440 分钟后。定位根因、修复、走 0.0.14 发布并实测验证恢复。

## 根因（已取证）

- 上一任务（09-09-adapt-upstream-fixes）照搬上游 c7a424c 的 `rawJSON/rawForm` 判定：`401 || (200 && is189AuthExpiredPayload)`，**收窄丢了 400 分支**。
- 189 现网对失效 open token/会话实际返回 **HTTP 400 + `UserInvalidOpenToken`/`unifyAccountInfo is null`**（`is189AuthExpiredPayload` 四个标记本来就能命中）。
- 0.0.12（merge-base）语义是"payload 命中即 `CodeAuthExpired`"→ `WithRetry`(IsAuthError) 被动刷新恢复；收窄后变成 `DRIVER_ERROR` → 不触发被动刷新、`Init` 的 `isSessionExpired` 不成立不重刷会话、认证态机无感知 → 调度器按健康态等待 7440 分钟。
- 上游自身还依赖 internal/auth 守卫接线兜底（本方未移植），故上游收窄在本方树中等价于砍掉恢复路径。

## Requirements

1. `drivers/189Cloud/transport.go` 的 `rawJSON`/`rawForm`：恢复"400 + `is189AuthExpiredPayload` 命中 → `CodeAuthExpired`"分支（payload 精确匹配 4 类失效标记，不影响其它 400 业务错）；保留上游新增的 403/429 分类。
2. `internal/taskauth/coordinator.go`：随取上游两处 `ctx.WithoutCancel + 10s 超时`加固（异步认证事件到达时请求 ctx 可能已取消）。
3. 质量门：`GOWORK=off go vet ./...` + `go test ./drivers/... ./internal/...`（存量 `internal/file` 失败除外）+ web 无需重建（无前端改动）。
4. 发布：bump `0.0.14`（README/compose×2）→ docker 三 tag 推送 → `git tag v0.0.14` + gh release → 重建容器 → 健康检查。
5. 实测验证：通过本地 API 用 admin 凭据登录后请求 `/api/files/list?account_id=1`，观察日志出现被动刷新（`HandlePassiveError`/`RefreshAuth`）后列表成功；若登录受阻则退化为容器日志证据 + 用户 UI 复测。

## Constraints

- 不改动 internal/auth 状态机与调度逻辑（与本修复正交）。
- 版本规则：`0.0.14` 递增，不跳 1.0.0。

## Acceptance Criteria

- [ ] 400+失效标记 → `CodeAuthExpired` 恢复，被动刷新后列表请求成功
- [ ] vet/test 全绿（存量失败已备案）
- [ ] `0.0.14` 三 tag 推送、tag+release、容器 `latest` 运行健康
- [ ] 日志或 UI 实测证据记录到任务
- [ ] 归档 + journal
