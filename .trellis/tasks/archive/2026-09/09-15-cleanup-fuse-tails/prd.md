# 清理 FUSE 移除后的三处小尾巴

## Goal

把 `09-15-remove-fuse` 之后残留的三处"孤儿"清干净，让 v0.0.46 的发布内容本身就是收尾态：

| # | 对象 | 性质 | 证据 |
|---|---|---|---|
| T1 | `Dockerfile` 的 `EXPOSE 5211 42069/tcp 42069/udp` | **僵尸声明**（08-31 移除内置离线下载/磁力时遗留）| 全仓 `grep 42069` 仅这一处命中（`*.go`/`*.yml`/`*.md`/`*.sh` 全零），`compose` 与 `install-docker.sh` 均不发布该端口 |
| T2 | `internal/upload/maintenance.go` 的 `ActiveTempPaths()` | **零调用者的导出方法** | `grep ActiveTempPaths` 除自身定义外 **0 处** |
| T3 | `internal/upload` 的 `TempRegistry` 链（`TempRegistry` 类型 / `NewTempRegistry` / `Track` / `Snapshot` / `Manager.tempRegistry` / `Manager.TempRegistry()`）| **随 FUSE 一起失去写入者**：`Track` 的**唯一调用者**是已删除的 `share/fuse/write_support.go:332` | `grep '\.Track(' internal/` = **0 处**；`Snapshot()` 只被 `activeTempPaths()` 读，而表恒空 → 该分支为死重 |

## Background

三者都是「功能删除后留下的空壳」，与 `09-12-remove-dead-code-p1`、`09-15-remove-playback-remote-reader` 同类；T1/T2 是本轮 FUSE 任务的调查结论里登记的待办，T3 是复核 T2 时顺带实证的同一类证据（`Track` 零调用者）。

**T3 的行为等价性论证**（删除前必须成立）：`TempRegistry.paths` 只由 `Track` 写入，`Track` 零调用 ⇒ 该 map **恒为空** ⇒ `activeTempPaths()` 里 `for p := range m.tempRegistry.Snapshot()` **一次都不执行**。因此删除该链**不改变** `CleanupOrphanTempFiles` 的任何行为（它只依赖 `m.tasks` 里的 `localPath`）。

**不动的部分（近名物）**：
- `internal/upload/temp.go` 的 `activeTempPaths()`（小写，仍被 `CleanupOrphanTempFiles` 使用）、`CleanupTempDir`、`CleanupOrphanTempFiles`、`StartTempCleanup` 与 `app.go:142` 的调用
- `TempMaxAge`、`TempDir()`（`api/upload.go:62` 在用）
- 上传/冷却/保留策略等一切业务逻辑

## Requirements

### D1 去掉僵尸端口声明

- `Dockerfile`：`EXPOSE 5211 42069/tcp 42069/udp` → `EXPOSE 5211`
- 复跑 `grep -rn 42069`（排除 `.trellis/`）**零命中**

### D2 删除 `ActiveTempPaths`

- 删 `internal/upload/maintenance.go`（16 行，文件内只有这一个方法）

### D3 删除 `TempRegistry` 链

- `internal/upload/temp.go`：删 `TempRegistry` 类型、`NewTempRegistry`、`Track`、`Snapshot`、`Manager.TempRegistry()`，以及 `activeTempPaths()` 里读取它的那段 `if m.tempRegistry != nil { … }`
- `internal/upload/manager.go`：删 `tempRegistry` 字段与 `m.tempRegistry = NewTempRegistry()`
- 复跑 `grep -rn "TempRegistry\|\.Track("`（排除 `.trellis/`）**零命中**

### D4 质量门

- `make lint` 0 issues、`GOWORK=off go vet ./...` exit=0、`GOWORK=off go test ./...` 全包 ok（**upload 包 70 个测试必须全过**，尤其 `TestUploadTaskRestoreAfterRestart` 与 retention/cooldown 系列）、`cd web && npm run type-check` exit=0
- 基线：`deadcode` ≤ 7、`unused` = 0、`gofmt` 无新增脏文件
- **行为回归自证**：临时用一个进程内小测试或直接调用证明 `CleanupOrphanTempFiles` 仍能清理过期临时文件（删除前后行为一致）

## Constraints

- **只做 T1/T2/T3**：不动上传业务逻辑、不动 `api/upload.go`、不重建 embed、不改 `go.mod`、不改 `.golangci.yml`
- 不发版、不动本机 `:5211` 容器与 `data/`（发版是紧随其后的独立任务）
- 本机 `danger-full-access`、审批关闭 —— 不请求 escalation
- 不改归档任务与历史 journal

## Acceptance Criteria

- [x] `Dockerfile` 只剩 `EXPOSE 5211`；全仓 `42069` 零命中（`.trellis/` 归档除外）
- [x] `internal/upload/maintenance.go` 已删除；`ActiveTempPaths` 全仓零命中
- [x] `TempRegistry` / `NewTempRegistry` / `Track` / `Snapshot` / `tempRegistry` / `Manager.TempRegistry()` **全仓零命中**；`activeTempPaths()` 中读取 registry 的分支已删
- [x] 保留项零改动：`activeTempPaths()`（小写）仍只依赖 `m.tasks`、`CleanupTempDir`/`CleanupOrphanTempFiles`/`StartTempCleanup` 与其在 `app.go` 的调用、`TempDir`、`TempMaxAge` 全部在位
- [x] 质量门全绿：`make lint` 0 issues、`go vet` exit=0、`go test ./...` 全包 ok（含 upload 70 测试）、`vue-tsc -b` exit=0
- [x] 行为等价性有实测证据：过期临时文件仍被清理（删除前跑一次、删除后再跑一次，结果一致）
- [x] 基线：`deadcode` ≤ 7、`unused` = 0、`gofmt` 零新增
- [x] 未发版、未 bump 版本、未动容器与 `data/`；embed 零改动
- [x] 任务归档、journal 记录、`main` 与 `origin/main` 同步

## Notes

- Scope `lightweight`（PRD-only），与 `09-12-remove-dead-code-p1`、`09-15-remove-playback-remote-reader` 同规格。
- 顺序说明：本任务**先于**发版执行，这样 v0.0.46 镜像里就不含这三处尾巴（避免发完再改一次）。
- T3 是用户清单之外、由 T2 复核顺带实证的同类项：若认为应严格只删 T2，删掉 D3 即可，T1/T2 不受影响。
