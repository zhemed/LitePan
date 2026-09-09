# 09-09-adapt-auth-guard-wiring

## Goal

按调查报告 `09-09-investigate-upstream-unported-items` 结论 B，移植上游 c7a424c 的认证守卫接线：内联刷新去重/复用窗口/冷却、初始化风暴防护、失败入状态机、网络熔断退避。走 `0.0.17` 发布管线并实测。

## Scope

**整取（ours-clean，全部来自 c7a424c 单提交）**：
- `internal/auth/`：`control.go`(新)、`gate.go`、`state_machine.go`、`refresh_runner.go`、`mutex.go`、`failure_kind.go`、`schedule_calc.go`、`retry.go`、`service.go`
- `internal/core/driverexec/exec.go`（含网络熔断 netFailState）
- 测试：`state_test.go`、`exec_test.go`(新)、`oauth_integration_test.go`(新)、`taskauth/persistence_test.go`(新)

**手改**：
- `internal/account/service.go`：`AuthCoordinator.RecoverAccount` 恢复上游 error 签名；Update 调用点改回 `if err := s.auth.RecoverAccount(ctx, id); err != nil { return View{}, err }`（撤销 0.0.13 的 void 适配，编译器强制核对）

**明确排除**：settings `UpdateSilent`（公告专用）、buildinfo 版本号、wire_http（守卫在 NewService 自装配，无需改）。

**已就位依赖**：`driver.AuthRefreshError`/`ClassifyOAuthRefreshError`、manager 预埋挂载点、`CallerPassive`、domain 错误码、`conn_error` final（均 0.0.13 已移植）。

## Acceptance Criteria

- [ ] 守卫接线激活：`NewService` 调用 `SetAuthGuards`，驱动内联刷新走统一状态机
- [ ] `go vet ./...` 全绿；`go test ./...` 全模块零失败（含上游 ~370 行 auth 测试）
- [ ] `0.0.17` 三 tag 镜像推送 + tag/release + 容器运行 + health/登录/列表三连通过
- [ ] 启动日志无迁移/装配错误；认证调度行为正常（首次健康检查、下次检查时间计算）
- [ ] 归档 + journal + push
