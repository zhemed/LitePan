# PRD：移植上游 4 项适用修复（2026-10-06）

## Goal

按用户拍板（**四项全做**），把调查报告 `archive/2026-10/10-06-upstream-updates-1006/research.md` §2 的
4 项"适用"改动按本方架构移植进来：

| 代号 | 上游提交 | 内容 |
|---|---|---|
| A1 | `7e0c0040` + `44796906`（收口） | 大文件上传默认强制 HTTP/1.1：新增 `httpx.NewUploadClient` + `driver.Config.UploadUseHTTP2`，115/189 驱动接入 |
| A2 | `5df900c5` | playback 多段 Range 支持（`parseRanges` + `multipart/byteranges` 响应）+ Range 诊断日志 |
| A3 | `214e027d` | 189Cloud 同步盘根独立标识 `syncRootID`、秒传父目录走 `apiParentID`、抽 `is189AuthExpiredResponse` |
| A4 | `52fcf32e` | 日志详情恒返回（后端）+ 日志详情复制（前端 2 文件） |

**用户已定**：2FA（`5db84a63`）本轮**不做**（留档）；不发版（版本号与部署由用户另定）。

## Background（本方现状，逐项证据见 research.md §2）

- A1：`internal/httpx/client.go` 只有 `NewStreamingClient`；`drivers/115_Open/upload.go:34` 的
  `newOSSUploadHTTPClient` 与 `drivers/189Cloud/driver.go:86` 都直接用它；`UploadUseHTTP2` 零命中。
- A2：`internal/playback/range.go`（55 行）与上游父提交**逐字节一致**（未移植），只有 `parseSingleRange`；
  `service.go` 的 `logAction` 只记 `user_agent`，无 `method`/`range`。
- A3：`grep syncRootID` 零命中；`ops.go:259` 的 `containsRoot` 只判 `root`/`"0"`/`"/"`；
  `createRapidUpload` 直接传 `parentID`。鉴权失效判定**本方已是等价条件**（`transport.go:213,253`）。
- A4：`internal/api/logs.go:166-168` 仍仅在 `Level >= Error` 时带 `Details`；
  前端 `SystemLogs.vue` 无"复制日志"按钮、`system-logs.css` 无对应样式；
  但依赖已具备：`web/src/composables/useToast.ts:38` 有 `copyTextToClipboard`。

## Requirements

### A1 上传强制 HTTP/1.1

- `internal/httpx/client.go` 新增 `NewUploadClient(base *http.Client, responseHeaderTimeout time.Duration, useHTTP2 bool) *http.Client`：
  以 `NewStreamingClient` 为基底；`useHTTP2` 为真时原样返回；否则把 transport 调整为
  `ForceAttemptHTTP2=false`、清空 `TLSNextProto`、`Protocols` 仅 HTTP/1.1、并限制 ALPN `NextProtos=["http/1.1"]`。
- `internal/driver/driver.go` 的 `Config` 新增 `UploadUseHTTP2 bool`（注释说明"默认 HTTP/1.1，驱动实测需要 HTTP/2 时才声明"）。
- 115：删除 `newOSSUploadHTTPClient`，`Init` 改调 `httpx.NewUploadClient(d.client, 60*time.Second, config.UploadUseHTTP2)`；
  189：同样改调（超时统一 60s）。
- 既有测试 `TestNewOSSUploadHTTPClientHasNoTotalTimeout` 随之改造（仍须断言"不限制整段传输时长"）。

### A2 playback 多段 Range

- `internal/playback/range.go`：新增 `errRangeNoOverlap`、`byteRange{start,end}`（`length()`、`contentRange(size)`）、
  `parseRanges(header, size) ([]byteRange, error)`；`parseSingleRange` 对分段字符串 TrimSpace，
  起点越界返回 `errRangeNoOverlap`。
  **语义**：分段数 ≥ 32 或累计长度 > 文件长度 → `(nil, nil)` 表示"忽略 Range 走全量"；全部越界 → `errRangeNoOverlap`。
- `internal/playback/multipart_range.go`（新文件）：`streamMultipartRanges(...)` 输出
  `multipart/byteranges`，Content-Length 按"仅 MIME 头 + 边界 + 各段长度"精确计算，正文仍走原有分片代理。
- `internal/playback/streamer.go`：Range 分支改用 `parseRanges`；`len>1` 走 multipart，`len==1` 走单段 206，
  `len==0 && err==nil` 落到全量路径；`size<=0` 有 Range 时仍 `passthrough`。
- `internal/playback/service.go`：`logAction` 改收 `*http.Request`，Debug 日志增 `method` 与 `range`
  （>256 字符截断）。

### A3 189Cloud 同步盘根

- `transport.go`：新增 `syncRootID = "sync:0"`；`apiParentID` 把 `sync:0` 映射为 `"0"`；把鉴权失效判定抽成
  `is189AuthExpiredResponse(status, data)`（**行为等价**，仅去重）。
- `driver.go`：`ListFiles` 中 `!isFamily() && item.ID == "0"` 的目录项 ID 重写为 `syncRootID`；
  `GetFileInfo` 认 `syncRootID`（返回 `"同步盘"` 目录项，`IDStable`）。
- `ops.go`：`containsRoot` 计入 `syncRootID`；`RenameFile` 改用 `containsRoot`。
- `upload.go`：`createRapidUpload` 的 `parentFolderId` 改用 `d.apiParentID(parentID)`。

### A4 日志详情恒返回 + 复制

- `internal/api/logs.go`：`Details` 移入 DTO 字面量（不再限 `Level >= Error`）。
- 前端：`web/src/components/admin/SystemLogs.vue` 增加"复制日志"按钮与 `copyLog()`（复用
  `copyTextToClipboard`）；`web/src/styles/system-logs.css` 增加 `log-row__detail-head` / `log-row__copy` 样式。
- **前端改动 → 必须重建 embed**（`cd web && npm run build`）并提交 `internal/api/web/**` 变更。

### 质量门与验证

- `make lint` 0 issues、`GOWORK=off go vet ./...` exit=0、`GOWORK=off go test ./...` 全包 ok、
  `cd web && npm run type-check` exit=0。
- 新增单测：A1（上传客户端 HTTP/1.1 断言：`ForceAttemptHTTP2=false`、ALPN、无整段超时）、
  A2（`parseRanges` 多段/越界/放大防护 + multipart 响应头与正文，按上游 `streamer_range_test.go` 移植）、
  A3（`apiParentID`/`containsRoot` 对 `syncRootID` 的行为）。
- 端到端：本机临时实例（数据副本、非默认端口）验证日志页详情可见（含非 ERROR 条目）与基础 200 面。
- 基线不劣化：`deadcode` ≤ 7、`unused` = 0、`gofmt` 无新增脏文件。

## Constraints

- **craft，不 apply**：不得 `git apply` 上游补丁（上游删了 `drivers/template` 等本方保留的东西）。
- **不越界**：`internal/buildinfo/version.go`、`Dockerfile`、`Makefile`、`.golangci.yml`、
  `go.mod`/`go.sum`（本批不需要新依赖）零改动；不动 `:5211` 容器与 `data/`。
- 不发版、不推送镜像；仅最后推送代码提交（用户已授权的 push 范围）。
- 网络动作仅 `github.com`（读取上游 diff 已在本轮完成；收尾推送代码）。

## Acceptance Criteria

- [x] A1–A4 四项全部落地，`grep` 证据齐全（`NewUploadClient`/`UploadUseHTTP2`/`parseRanges`/
      `streamMultipartRanges`/`syncRootID`/`apiParentID` 接入/`Details` 恒返回/前端复制按钮）
- [x] 新增单测覆盖 A1/A2/A3 的关键语义，且**在旧实现下会失败**（逐项反向验证或说明原因）
- [x] 质量门全绿（lint/vet/test/type-check），embed 已按前端改动重建
- [x] 基线不劣化：`deadcode` ≤ 7、`unused` = 0、`gofmt` 无新增
- [x] 端到端：临时实例（数据副本、非默认端口）日志页与基础面 200，日志无 ERROR（除故意注入）
- [x] 未越界：`version.go`/`Dockerfile`/`Makefile`/`.golangci.yml`/`go.mod` 零改动；`:5211` 与 `data/` 未动
- [x] 任务归档 + journal + 推送

## Notes

- Scope `cross-layer`（httpx/driver/playback/api/web 五处），按复杂任务补 `design.md` + `implement.md`。
- 本轮**不做**：2FA、上游已删能力的任何回填、A2 的"忽略 Range 全量"之外的策略调整。
- A3 无真实 189 账号 → 行为验证只能到单测层，端到端留档（在 implement 执行记录里写明）。
