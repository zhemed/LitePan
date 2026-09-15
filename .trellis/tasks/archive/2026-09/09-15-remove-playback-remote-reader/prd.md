# 清理 FUSE 移除后的二级死代码：playback 的 RemoteReader 圈

## Goal

把 `internal/playback` 里**随 FUSE 删除而失去唯一入口**的三个文件删掉：

| 文件 | 行数 | 内容 |
|---|---|---|
| `internal/playback/remote_reader.go` | 116 | `RemoteReader` 类型 + `OpenRemoteReader` + `newRemoteReader` + `ReadAt/Size/Close` |
| `internal/playback/remote_window_reader.go` | 391 | 有界窗口合并 + 并发预取（只被 `RemoteReader` 使用）|
| `internal/playback/local_reader.go` | 57 | `openLocalFileReader` + `localFileReader`（只被 `OpenRemoteReader` 使用）|
| `internal/playback/remote_reader_test.go` | 413 | 上述代码的 10 个单测与测试辅助 |
| **合计** | **977** | |

## Background（可达性证据，2026-09-15 实测）

**唯一入口已随 FUSE 消失**

```
$ grep -rn "OpenRemoteReader" --include="*.go" .
internal/playback/remote_reader.go:23:func (s *Service) OpenRemoteReader(...)   ← 定义
（无任何调用方；FUSE 删除前唯一调用点是 internal/share/fuse/nodes.go:393）
```

**逐符号核查（删除前必须全绿）**

| 符号 | 引用点 | 判定 |
|---|---|---|
| `RemoteReader`（类型/方法）| 仅 `remote_reader.go` 自身 + `remote_reader_test.go` | 死 |
| `OpenRemoteReader` | 只有定义，**零调用** | 死 |
| `newRemoteReader` | 仅 `remote_reader_test.go:75` | 死 |
| `remoteWindowReader`（整文件）| 仅 `remote_reader.go` | 死 |
| `openLocalFileReader` / `localFileReader` | 仅 `remote_reader.go:41` | 死 |

**仍是活代码，不得误删**（同一包里名字相近）

- `internal/playback/local_file.go`（44 行）—— `serveLocalFile`，HTTP 播放的本地文件分支
- `internal/playback/{range_proxy,streamer,response,seeker,range,headers,pick,redirect,transport,service,cache,bench_env}.go` 及其测试（`range_proxy_parallel_test.go` 实测**不**引用被删符号）
- `internal/playback/account_range_limiter.go` —— `range_proxy.go` 在用

## Requirements

### D1 删除四个文件

- `git rm internal/playback/{remote_reader.go,remote_window_reader.go,local_reader.go,remote_reader_test.go}`

### D2 删除后可达性复核

- `grep -rn "RemoteReader\|remoteWindowReader\|localFileReader\|OpenRemoteReader\|newRemoteReader" --include="*.go" .` **零命中**
- `deadcode ./cmd/litepan` **不高于 7**（删除后不应新增不可达项）
- `golangci-lint --enable=unused` 仍为 **0 issues**

### D3 质量门

- `make lint` 0 issues、`GOWORK=off go vet ./...` exit=0、`GOWORK=off go test ./...` 全包 ok（**playback 包测试数按删除的测试等比减少，属预期**）、`cd web && npm run type-check` exit=0
- 不重建 embed（前端零改动）

### D4 spec 同步

- `.trellis/spec/` 中若有指向这三个文件/符号的引用，改为现状或删除；其余内容不动

## Constraints

- **只删这一圈**：不得顺手改播放链路（`range_proxy`/`streamer`/`service` 等一行都不动），不得动 `local_file.go`
- 不重建 embed、不发版、不动本机 `:5211` 实例、不改 `go.mod`（这三个文件不引入任何依赖）
- 本机 `danger-full-access`、审批关闭 —— 不请求 escalation
- 不改归档任务与历史 journal

## Acceptance Criteria

- [x] 四个文件已删除（977 行）；`internal/playback/` 下不再有 `remote_reader.go`/`remote_window_reader.go`/`local_reader.go`/`remote_reader_test.go`
- [x] 全仓 `grep` 五个符号（`RemoteReader`/`remoteWindowReader`/`localFileReader`/`OpenRemoteReader`/`newRemoteReader`）**零命中**
- [x] `local_file.go` 的 `serveLocalFile`、`range_proxy.go`、`streamer.go`、`service.go`、`account_range_limiter.go` **零改动**（`git status` 逐文件核对）
- [x] 质量门全绿：`make lint` 0 issues、`go vet` exit=0、`go test ./...` 全包 ok、`vue-tsc -b` exit=0
- [x] 基线：`deadcode ./cmd/litepan` ≤ 7、`unused` = 0、`gofmt` 无新增脏文件
- [x] `git diff --stat` 显示净删除 ≈ 977 行（不含 embed，且 embed 无改动）
- [x] spec 中不再有指向被删文件的引用（若有则已同步）
- [x] 本机 `:5211` 容器与 `data/` 未动；未发版、未 bump 版本
- [x] 任务归档、journal 记录、`main` 与 `origin/main` 同步

## Notes

- Scope `lightweight`（PRD-only），与先例 `09-12-remove-dead-code-p1`、`09-12-remove-mediaorganize-dead-package` 同规格。
- 本任务是 `09-15-remove-fuse` 的**计划内后续**（其 PRD 的 Out of Scope 与本 PRD 的 Goal 严格互补）。
- **方法学要点**：`deadcode` 与 `unused` 都**没有**报出这批代码（实测 deadcode 7 项里 FUSE/playback 相关 0；unused 0 issues）—— 因为它们是**导出的类型/方法**，RTA 与 U1000 对导出符号保守。所以本轮判据只能是**调用点计数**（全仓 grep 零调用），不能依赖工具。
