# Implementation Plan: 移植上游三批修复

## Overview

三个 Phase 顺序执行，每 Phase 后跑门禁；全部完成后做一次端到端临时实例验证与全量质量门。

移植方式：**读上游 diff 作为规格，按本方代码 craft**（不 apply 补丁）。

---

## Phase 0: 基线快照

- [x] 0.1 `git status --short`（期望仅本任务目录）、`git log --oneline -1`
- [x] 0.2 `deadcode ./cmd/litepan` = 7、`golangci-lint --enable=unused` = 0
- [x] 0.3 `go test ./...` 包数与用例数基线（记录 `drivers/115_Open`、`drivers/189Cloud`、`internal/adminauth`、`internal/api` 四个包的测试数）
- [x] 0.4 上游 ref 可用性：`git rev-parse --short upstream/main`（= `42a3ee9a`）

**回滚点 R0**：以上即后续比对依据

---

## Phase A: 批次 1 —— 115 驱动（P0）

- [x] A0 先查调用点：`grep -rn "buildDirPath\|statLocalFile\|collectFullListPages" --include=*.go .`
- [x] A1 `full_list.go`：`fullListEmptyRetryWait` + `expectedCount` + 重试与终局校验（不满则 `CodeDriverError`）
- [x] A2 `full_list.go`：`buildDirPath(paths, selfName, rootID) (string, bool)`；`ResolveDirPath` 相对挂载根 + 越界报错 + 空入口 `CodeNotFound`
- [x] A3 `full_list.go`：新增 `pathSegmentName`，两处拼接改用它
- [x] A4 `ops.go`：`maxPickCodeCacheEntries` + 超限 `clear` + `DeleteFiles` 成功后清理缓存
- [x] A5 `drivers/189Cloud/ops.go`：正则提到包级 `insecureSchemeRe`
- [x] A6 测试：`full_list_test.go` 增加/移植用例（空页重试、Count 不足报错、越界报错、含 `/` 目录名消毒）
- [x] A7 **门禁 GA**：`go build ./...`、`go vet ./...`、`go test ./drivers/115_Open/ ./drivers/189Cloud/ ./...`
- **回滚点 RA**：`git checkout HEAD -- drivers/115_Open drivers/189Cloud`

---

## Phase B: 批次 2 —— 性能与可观测（P1）

- [x] B1 `internal/adminauth/service.go`：`configMu`/`configLoaded`/`configValues` + `configValue`/`setConfig`，替换全部 `configs.Get`/`configs.Set` 调用点；**注释写明"配置只经本方写入"的失效前提**
- [x] B1t `internal/adminauth/service_test.go`：缓存命中/失效（`setConfig` 后立即读到新值）、`All()` 失败不置 loaded
- [x] B2 `internal/api/slow_dashboard_log.go`：`slowRequestLogInterval` + `slowRequestLogs{shouldLog,recovered}` + 日志降 Debug + `suppressed_count`；`router.go` 加 `slowLogs` 字段
- [x] B2t `slow_dashboard_log_test.go`：抑制窗口内只记一次、恢复后重置
- [x] B3 `internal/api/local_upload.go`：批次级 `EvalSymlinks(m.Path)` + `resolveLocalUploadSourceUnderRoot` + 去掉 `statLocalFile`
- [x] B3t `local_upload_test.go`：按需取用上游越界/符号链接用例
- [x] B4 `internal/api/oauth.go`：`lastErr` 非空时 Warn
- [x] B5 **门禁 GB**：`go build`/`vet`/`go test ./...`
- **回滚点 RB**：`git checkout HEAD -- internal/adminauth internal/api`

---

## Phase C: 批次 3 —— 清洁（P2）

- [x] C0 核对上游口径：`git show 924e4c75 -- internal/store/backup.go internal/backuprestore/service.go`（判断 C5 是否连带清理名单）
- [x] C1 `Dockerfile`：加 `npm run type-check`
- [x] C2 `internal/api/{accounts.go,files.go}`：去 `IsZero()` 分支
- [x] C3 `internal/api/commit_writer.go`：`Write` 直接调 `WriteHeader`
- [x] C4 `cmd/litepan/main.go`：三条中文日志
- [x] C5 `internal/adminauth/service.go`：删 `KeyAdminTempPasswordLastReset` + 写入 + `tempPasswordState.LastReset`（连带清理与否按 C0 结论）
- [x] C6 **门禁 GC**：`go build`/`vet`/`go test ./...`
- **回滚点 RC**：按文件 `git checkout HEAD -- …`

---

## Phase D: 全量验证与收尾

- [x] D1 **全量质量门**：`make lint` 0 issues、`go vet` exit=0、`go test ./...` 全包 ok、`cd web && npm run type-check` exit=0
- [x] D2 **基线**：`deadcode` ≤ 7、`unused` = 0、`gofmt` 无新增；`git status` 无越界文件
- [x] D3 **端到端**：`go build -o /tmp/port-verify/litepan ./cmd/litepan` + 数据副本 → 临时实例（非默认端口）验证 health/登录/accounts/settings/upload runtime 全 200、日志无 ERROR
- [x] D4 提交（消息带 `[task:port-upstream-fixes-0921]`）→ `a54fd7ae`
- [ ] D5 `skill trellis-check` → `mark-check` → 勾选验收项 → `pre-archive` → `archive` → `add_session.py` → `git push`

---

## Review Gates

| 门 | 判据 |
|---|---|
| GA | 115/189 包测试通过 + 新增用例覆盖三类场景 |
| GB | 全仓 `go test ./...` 通过；B1 缓存语义有用例锁定 |
| GC | 全仓 `go test ./...` 通过；C 组 grep 证据齐全 |
| GD | 全量质量门 + 基线不劣化 + 端到端临时实例 200 四连 + 无越界 |

## Rollback

- 单提交落地 → `git revert`；若拆提交则逐批 revert（B1 优先关注）。
- 不涉及数据/迁移/部署，无 DROP、无 compose 变更。

## Out of Scope

- `AutomationTriggerAdvanced`（用户已定：留档）
- 已删功能的任何回填（STRM/媒体整理/缓存整理/FUSE/API 秘钥/跨盘/分享）
- 发版（`v0.0.48` 由用户另定）、前端改动与 embed 重建

---

## 实施记录（2026-09-21）

### 逐项结果

| 项 | 结果 | 证据 |
|---|---|---|
| A1 完整性 | 落地 | `full_list.go:14-17,57-86`（空页 250ms 重试 + 终局 `expectedCount` 校验） |
| A2 相对挂载根 | 落地 | `full_list.go:121-170`（`buildDirPath` 双值 + `CodeNotFound`/`CodeDriverError`） |
| A3 段名消毒 | 落地 | `full_list.go:177-184`（`pathSegmentName`，`/`、`\` → `_`） |
| A4 缓存治理 | 落地 | `drivers/115_Open/ops.go:16`、`rememberPickCode` 超限 clear、`DeleteFiles` 成功后清理 |
| A5 正则提级 | 落地 | `drivers/189Cloud/ops.go:24-25`；与上游 `924e4c75` 该 hunk **逐字一致** |
| B1 配置缓存 | 落地（有偏差） | `internal/adminauth/service.go:101-127,614-686`；新增 `internal/adminauth/config_cache_test.go`（7 例） |
| B2 慢日志降噪 | 落地（有偏差） | `internal/api/slow_dashboard_log.go`、`router.go:90`；测试 7 例（含上游签名对齐） |
| B3 批次级解析 | 落地 | `internal/api/local_upload.go:319-323,394`、`resolveLocalUploadSourceUnderRoot:512-523`；`statLocalFile` 已删 |
| B4 OAuth 告警 | 落地 | `internal/api/oauth.go:74-83`；新增 `internal/api/oauth_test.go`（2 例） |
| C1 | 落地 | `Dockerfile:12`，与上游 `924e4c75` 一致 |
| C2 | 落地 | `accounts.go:54-55`、`files.go:20`，与上游 `924e4c75` 一致 |
| C3 | 落地 | `commit_writer.go:23-27`，与上游 `internal/api/commit_writer.go` 一致 |
| C4 | 落地 | `cmd/litepan/main.go:48,59,63`，与上游 `924e4c75` 一致 |
| C5 | 落地 | 删常量/写入/`LastReset`；上游 `924e4c75` **保留**两处清洗名单 → 本方保留（`internal/store/backup.go:97`、`internal/backuprestore/service.go:34,451`） |

### 与上游的刻意偏差（及理由）

1. **B1 只缓存本服务独占键**（上游缓存整表）：上游的 `configValue` 把 `All()` 全量塞进 `configValues`。
   本方核查发现 `internal/settings/service.go:196` 会写 `oauth_server_url`、`upload_task_concurrency`、
   `log_retention_days`、`auth_active_refresh_enabled`，而 `adminauth.SystemConfig`（`service.go:404-407`）
   正好要读这四个键 → 整表缓存会让管理面板在设置页保存后一直显示旧值。
   故新增 `serviceOwnedConfigKeys` 白名单（默认不缓存，新增键漏登记只会退化为直读）。
   独占键的写入路径已全量核查：仅 `adminauth.setConfig`；备份恢复写的是 **staging 库**，
   且 `internal/app/app.go:71` 在 Store 打开前 `ApplyPending` → 恢复必然重启进程。
2. **B2 `recovered` 仅在“确有压制记录”时结束窗口**（上游：任何快请求都 `delete`）：
   上游写法下，间歇性变慢会被轮询里的快请求反复冲掉窗口，退化成逐次刷屏（与“降噪”目标相悖）。
   另有偏差：常量名/签名采用上游的 `slowDashboardLogInterval` 与 `shouldLog(path, now) (int, bool)`，
   便于后续与上游对照；消息文案保留本方原文（结构化字段已有 `path`）。
3. **B3 把根解析提前到函数最前**（上游在 `LookupUploadAccount` 之后），失败文案带映射路径与底层原因
   （上游为通用文案）。行为目标一致：批次级快速失败、逐文件不再重复 `EvalSymlinks`。

### 基线复核（D2）

- `gofmt -l`：15 个文件，**逐一比对 HEAD 全部早已脏**（零新增；此前记录的 16 为口径差异）。
- `deadcode ./cmd/litepan` = 7，条目与基线完全一致；`golangci-lint --enable=unused` = 0 issues。
- `git status`：改动仅 PRD File Map 列出的 17 个文件 + 2 个新测试文件；`web/src/**`、`internal/api/web/**`
  （`npm run build` 后仍零 diff）、`go.mod`、`.golangci.yml`、`version.go` 全部零改动。

### 端到端（D3）

`/tmp/port-verify/litepan` + 数据副本、`LITEPAN_LISTEN=127.0.0.1:35221`：

- 200 六连：`/api/health`、`/api/files/upload/runtime`、`/api/admin/tools/local-upload/config`、
  `/api/admin/accounts`、`/api/admin/settings`、`/api/admin/system-config`；登录 200。
- **B1 写透**：`POST /api/admin/update-credentials`（admin_username + session_timeout=3）后回读即 3；
  **非独占键不陈旧**：`PUT /api/admin/settings` 写 `log_retention_days=45` 后回读即 45；
  再登录仍 200（凭据走缓存路径）。
- **B3 快速失败**：删除映射根后提交上传 → `created=0`，日志
  `VALIDATION: 映射目录 /tmp/port-verify/media 无法访问：lstat …: no such file or directory`；
  `rel_path=../etc/passwd` → 400「未找到可上传的文件」。
- 日志：除上述**故意注入**的一条例外，ERROR = 0。
- 现场清理：临时实例已停、`/tmp/port-verify` 已删；`:5211` 容器 `v0.0.47` 未动，
  工作区 `data/litepan.db` mtime 仍为 Sep 15 13:41。
