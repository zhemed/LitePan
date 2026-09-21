# 移植上游三批修复（P0 115 驱动 / P1 性能可观测 / P2 清洁）

## Goal

按 `.trellis/tasks/archive/2026-09/09-21-upstream-updates-0921/research.md` §7 的三批清单，把上游 `Ponphil/LitePan`（`46a0a89..42a3ee9a`）中**本方适用**的修复移植进来，**按本方架构适配**（不直接 apply）。

用户已于本轮确认：**三批全做**；「高级定时」新能力**不做**（留档）。

## Background（证据来源）

- 上游 11 提交中 7 组适用（4 项 P0 正确性、4 项 P1、4 项 P2），详见调研报告 §3 的逐项证据（上游 sha、文件、diff 要点 + 本方现状行号）。
- **关键前提**：上游删除了 `drivers/template/**`（−435 行）与其它本方保留的东西 → **必须按本方代码 craft patch，不得 `git apply` 上游 patch**。
- 本方当前基线：`main = b89383be`、版本 `v0.0.47`（已发布部署），工作区干净。

## Requirements

### 批次 1（P0）：115 驱动正确性 + 189Cloud 微优化

来源 `511d7fe8`、`6d392018`、`924e4c75`；文件 `drivers/115_Open/{full_list.go,ops.go}`、`drivers/189Cloud/ops.go`。

- **A1 全量清单完整性**：`collectFullListPages` 空页先等 250ms 重试一次；全程记录 `expectedCount = max(page.Count)`；重试后仍空且 `expectedCount > len(seenIDs)` → 返回 `CodeDriverError`（"115 全量清单不完整：应有 N 个文件，实际只获取 M 个，已停止扫描"）。注释明确"只以连续空页作为结束信号，Count 仅用于终局拦截"。
- **A2 `ResolveDirPath` 相对挂载根**：`buildDirPath` 返回 `(string, bool)`；遇 `rootID` 段截断（`segs = segs[:0]`）；不在挂载根下 → `CodeDriverError`；`get_info` 无入口 → `CodeNotFound`。
- **A3 目录名消毒**：新增 `pathSegmentName()`，段内 `/`、`\` → `_`，用于 `buildDirPath` 两处拼接（防伪造层级）。
- **A4 `pickBy` 缓存治理**：`maxPickCodeCacheEntries = 100_000`，超限且键不存在时 `clear(d.pickBy)`；`DeleteFiles` 成功后从缓存删除已删 ID。
- **A5（顺手）**：`drivers/189Cloud/ops.go` 把 `regexp.MustCompile("(?i)^http://")` 提到包级 `insecureSchemeRe`。
- **测试**：按本方代码移植上游 `full_list_test.go` 的相关用例（完整性/越界/斜杠三类各至少 1 例），并确保 `drivers/115_Open` 与 `drivers/189Cloud` 包测试通过。

### 批次 2（P1）：性能与可观测

来源 `5e27f236`、`924e4c75`；文件 `internal/adminauth/service.go`、`internal/api/{slow_dashboard_log.go,router.go,local_upload.go,oauth.go}`。

- **B1 adminauth 配置内存缓存**：`Service` 增 `configMu` + `configLoaded bool` + `configValues map[string]string`；新增 `configValue()`（首次 `configs.All(ctx)` 全量加载后读内存）与 `setConfig()`（写库 + 回写内存）；`configString/configInt/configBool` 与所有 `s.configs.Set` 调用点改走二者。
  - **必须明确失效语义**：配置只经本方 UI/服务写 → 缓存安全；`setConfig` 回写保证同进程一致。
  - **落地偏差（实施期核查后收紧）**：只缓存 `serviceOwnedConfigKeys`（本服务独占键），其余键直读。理由：`internal/settings/service.go` 会写 `oauth_server_url`/`upload_task_concurrency`/`log_retention_days`/`auth_active_refresh_enabled`，而这四个键正好由 `adminauth.SystemConfig` 读取 → 整表缓存会显示陈旧值。`configMu` 用 `sync.Mutex`（单锁即可保证"写库成功后才更新快照"的顺序性；上游用 `RWMutex`）。
- **B2 慢接口日志降噪**：`slowDashboardLogInterval = 30 * time.Minute`（上游原名，取代本文档初稿的 `slowRequestLogInterval`）；`slowRequestLogs{shouldLog/recovered}` 抑制 + 恢复重置；日志**由 Info 降为 Debug** 并带 `suppressed_count`；`Handler` 增 `slowLogs` 字段。
  - **落地偏差**：`recovered` 只在"确有被压制的慢日志"时结束窗口（上游是任何快请求都 `delete` 条目）。否则轮询中的快请求会反复冲掉 30 分钟窗口，间歇性变慢退化为逐次刷屏，与降噪目标相悖。
- **B3 上传符号链接解析提级**：`createLocalUploadTasksSync` 批次开始处 `filepath.EvalSymlinks(m.Path)` 一次；循环内改用 `resolveLocalUploadSourceUnderRoot(s.abs, resolvedRoot)`；删 `statLocalFile` 间接层改直呼 `os.Stat`；根不可解析 → 整批快速失败并说明原因。
- **B4 OAuth 失败可观测**：`_ = lastErr` → `if lastErr != nil { requestLogger(...).Warn("OAuth 转发重试均失败", "url", url, "attempts", maxRetries+1, "err", lastErr) }`。
- **测试**：按需取用上游对应用例（adminauth 配置缓存、慢日志抑制），并对 B3 补一条"越界/符号链接"回归。

### 批次 3（P2）：清洁

- **C1** `Dockerfile`：`RUN npm run build` → `RUN npm run type-check && npm run build`。
- **C2** `internal/api/{accounts.go,files.go}`：去掉 `if !x.IsZero()` 分支直接赋值（**已核实** `FormatAPITime` 零值返回 `""`，行为等价）。
- **C3** `internal/api/commit_writer.go`：`Write` 内的幂等自增判断下放给 `WriteHeader`（行为等价）。
- **C4** `cmd/litepan/main.go`：三条错误日志改中文（`启动失败`/`关闭出错`/`运行出错`）。
- **C5** 删除**只写不读**的 `KeyAdminTempPasswordLastReset`：`internal/adminauth/service.go` 删常量 + `ResetPassword` 的写入 + `tempPasswordState.LastReset` 字段与赋值。
  - **迁移口径**：与上游保持一致 —— 先核对上游是否也清了 `internal/store/backup.go` 清洗名单与 `internal/backuprestore/service.go` 清空名单；若上游保留则本方保留（旧库残留键无害），**不新增迁移**（用户未要求，且属可选清理）。

## Constraints

- **严格按本方代码 craft**：不得 `git apply` 上游补丁；不得把上游已删功能的依赖带进来（`drivers/template`、`internal/strm`、`internal/embyproxy`、`internal/fnosproxy`、FUSE 等一律不碰）。
- **不越界**：不改 `.golangci.yml`、`go.mod`/`go.sum`（本批不需要新依赖）、`internal/buildinfo/version.go`、`web/src/**`；不动 `:5211` 容器与 `data/`。
- 每批完成后跑质量门（`make lint` / `GOWORK=off go vet ./...` / `go test ./...`；本批不改前端，故 **embed 不重建**）。
- 基线不得劣化：`deadcode` ≤ 7、`unused` = 0、`gofmt` 无新增脏文件。
- **不发版**：是否发 `v0.0.48` 由用户在本任务完成后另定。
- 本机 `danger-full-access`、审批关闭 —— 不请求 escalation；不改归档任务与历史 journal。

## Acceptance Criteria

- [x] **批次 1**：A1–A5 全部落地；`drivers/115_Open` 与 `drivers/189Cloud` 测试通过；新增/移植用例覆盖"空页重试与不完整清单报错""路径越界报错""目录名含斜杠消毒"三类
- [x] 批次 1 后 `grep` 证据：`collectFullListPages` 含 `fullListEmptyRetryWait` 与 `expectedCount`；`buildDirPath` 返回双值；存在 `pathSegmentName`；`pickBy` 有 `maxPickCodeCacheEntries`
- [x] **批次 2**：B1–B4 全部落地；`internal/adminauth` 与 `internal/api` 测试通过；B1 的缓存语义在代码注释中写明（配置只经本方写入）
- [x] 批次 2 后 `grep` 证据：`configMu`/`configLoaded`/`configValue` 存在；`slowDashboardLogInterval` 与 `shouldLog/recovered` 存在且日志为 `Debug`；`resolveLocalUploadSourceUnderRoot` 存在；`oauth.go` 无 `_ = lastErr`
- [x] **批次 3**：C1–C5 全部落地；`grep` 证据：Dockerfile 含 `npm run type-check`；`accounts.go`/`files.go` 无 `IsZero()` 分支；`commit_writer.Write` 直接调 `WriteHeader`；`main.go` 三条中文日志；全仓 `KeyAdminTempPasswordLastReset` 零命中
- [x] 全量质量门全绿：`make lint` 0 issues、`go vet` exit=0、`go test ./...` 全包 ok（包数与用法基线一致或增加）、`vue-tsc -b` exit=0
- [x] 基线不劣化：`deadcode` ≤ 7、`unused` = 0、`gofmt` 无新增；`git status` 无越界文件（`web/src/**`、`go.mod`、`.golangci.yml`、`version.go` 零改动）
- [x] **行为验证**：本机起临时实例（数据副本、非默认端口）验证 —— 健康 200、登录 200、账号列表 200、设置读写 200、`/api/files/upload/runtime` 200；干净启动阶段日志无 ERROR（后段 B3 快速失败为**故意注入**，仅产生 1 条对应 ERROR）
- [ ] 归档、journal、`main` 与 `origin/main` 同步；提交消息带 `[task:port-upstream-fixes-0921]`
- [x] 任务**不做**：`AutomationTriggerAdvanced`（用户已定留档）、任何已删功能的回填

## Notes

- Scope 标 `cross-layer`（驱动 + 鉴权 + API + 构建脚本，四层），按复杂任务补 `design.md` + `implement.md`（三批 = 三个 Phase，各自独立验证与回滚点）。
- 移植纪律（调研报告 §7 已写）：**craft patch + 本方回归测试 + 全量质量门**；上游测试按需取用，不整文件搬运。
- 上游依据：`511d7fe8`、`6d392018`、`924e4c75`、`5e27f236`（sha 与逐项证据见调研报告 §2/§3）。
