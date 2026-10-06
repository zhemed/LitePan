# Design：移植上游 4 项适用修复

## 目标与不变量

**目标**：把 research.md §2 的 4 项改动按本方架构重做，**行为对上上游、实现随本方**。

**不变量**

1. **不引入上游已删依赖**（`internal/strm*`、`mediaorganize`、`cloudshare`、`offlinedownload`、
   `spacecleanup`、`drivers/template` 一律不碰）；上游删掉的本方文件（如 `drivers/115_Open/full_list_test.go`）
   一律保留。
2. 每项都能独立验证、独立回滚；四项之间无耦合（可单独 revert）。
3. **不改版本号、不发版、不动部署与数据**（本轮用户授权范围止于代码）。
4. 前端改动必须重建 embed 并随提交带上（否则线上页面与源码不一致）。

## 关键决策

### KD1 A1：超时从 30s 调到 60s 的取舍

上游在同一提交把上传客户端的"响应头超时"从 30s 提到 60s。本方原值是 30s（`newOSSUploadHTTPClient`
与 189 都用 30s）。**采用上游 60s**：国产网盘的上传会话建立（`createUploadFile` → 分片 OSS 端点）
偶发慢启动，30s 会在弱网/大文件首片时误判；60s 仍由 `ResponseHeaderTimeout` 兜底，不会无限等待。
该值只影响"等响应头"的时长，不影响整段传输（保持不设总超时）。

### KD2 A2：多段 Range 的三条语义必须在代码里写死

`parseRanges` 的返回值承担三种含义，任何一处理解错都会让播放器行为异常：

| 返回 | 含义 | streamer 的处理 |
|---|---|---|
| `(ranges, nil)`，`len>=1` | 正常段 | `len>1` → multipart/byteranges 206；`len==1` → 普通 206 |
| `(nil, nil)` | **忽略 Range**（分段 ≥32 或累计长度 > 文件长度，防小请求放大上游读取） | 落到全量路径（200 + 整文件） |
| `(nil, errRangeNoOverlap)` 或其它 error | 全部越界 / 语法错误 | 416 + `Content-Range: bytes */size` |

另：`size<=0`（未知大小）时仍 `passthrough`，不做 Range 处理。

### KD3 A3：`syncRootID` 是"别名防冲突"，不是新能力

天翼云盘"同步盘"的真实 ID 是 `0`，与公共层把 `0` 当"根目录别名"的约定冲突（本方的
`normalizeParent` / `isRootAlias` 都把 `0` 视作根）。上游用 `syncRootID = "sync:0"` 在**本方内部**
区分二者，出网前再经 `apiParentID` 还原为 `"0"`：

- 入方向（列表/详情）：`ListFiles` 把 `item.ID == "0"` 重写为 `syncRootID`；`GetFileInfo` 认 `syncRootID`；
- 出方向（请求参数）：`apiParentID(sync:0) = "0"`；
- 根判定：`containsRoot` 把 `syncRootID` 也算根（不允许重命名/删除同步盘本身）。

**验证边界**：无真实 189 账号 → 只能单测 `apiParentID` / `containsRoot` / `GetFileInfo` 的映射分支，
真实列表行为留档（写进执行记录，不声称已验证）。

### KD4 A4：前端只做"详情 + 复制"，不顺手改样式体系

上游前端改动 48 行（Vue 23 + CSS 27），本方 `SystemLogs.vue` 已具备 `formatDetails`、
`useToast.ts` 已导出 `copyTextToClipboard`，所以按上游形态移植即可；**不**重构日志页其它部分。
后端一行为等价改动（`Details` 从条件赋值改为字面量字段），注意 `omitempty` 语义不变。

### KD5 测试策略：逐项给"旧实现必失败"的证据

- A1：断言 `NewUploadClient(..., false)` 返回的 transport `ForceAttemptHTTP2 == false`、
  `TLSClientConfig.NextProtos == ["http/1.1"]`、`Protocols` 仅 HTTP/1.1、`ResponseHeaderTimeout == 期望值`、
  且 `Timeout == 0`（不设整段超时）；`useHTTP2=true` 时保持 h2 尝试。
- A2：移植上游 `streamer_range_test.go`（多段请求 → 206 + `multipart/byteranges` + 各段 body、
  越界 → 416、放大防护 → 200 全量），并补 `parseRanges` 纯函数表驱动用例。
- A3：`apiParentID`（普通/根/`sync:0`）、`containsRoot`（含 `sync:0`）、`GetFileInfo(sync:0)` 三分支。
- A4：后端断言非 ERROR 级条目也带 `details`（用 `toLogDTO` 直接单测，避免起 HTTP 服务）。

## 风险与处置

| 风险 | 处置 |
|---|---|
| multipart Content-Length 算错导致播放器卡住 | 按上游"只统计 MIME 头 + 边界 + 段长"的算法，并用上游同款测试断言 `Content-Length` 与实际写出字节数一致 |
| 放大防护把正常单段请求也忽略 | 语义表驱动用例：单段、两段、32 段、累计超长四种输入 |
| A3 改了出网参数导致 189 上传 404 | 只改"秒传"这一处父目录参数，且在 `apiParentID` 里对普通 ID 保持原样（透传） |
| 前端改动未重建 embed | 质量门里加 `git status --short internal/api/web` 必须**有**变更（本批与 v0.0.48 相反） |
| 顺手改了不该改的 | 提交前 `git status` 逐文件核对，`version.go`/`Dockerfile`/`go.mod` 必须零改动 |

## Rollback

四项在一个提交里（如需可分文件 revert）：

- A1：`git checkout HEAD -- internal/httpx/client.go internal/driver/driver.go drivers/115_Open/driver.go drivers/115_Open/upload.go drivers/189Cloud/driver.go`
- A2：`git checkout HEAD -- internal/playback/{range,streamer,service}.go && rm internal/playback/multipart_range.go`
- A3：`git checkout HEAD -- drivers/189Cloud/{driver,ops,transport,upload}.go`
- A4：`git checkout HEAD -- internal/api/logs.go web/src/components/admin/SystemLogs.vue web/src/styles/system-logs.css` + 重建 embed

## File Map

| 路径 | 动作 |
|---|---|
| `internal/httpx/client.go` | A1 新增 `NewUploadClient` |
| `internal/driver/driver.go` | A1 `Config.UploadUseHTTP2` |
| `drivers/115_Open/{driver.go,upload.go}`（+ `upload_retry_test.go`） | A1 接入与测试改造 |
| `drivers/189Cloud/driver.go` | A1 接入 |
| `internal/playback/range.go`、`multipart_range.go`(新)、`streamer.go`、`service.go` | A2 |
| `internal/playback/streamer_range_test.go`(新)、`range_test.go`(如需) | A2 测试 |
| `drivers/189Cloud/{driver,ops,transport,upload}.go` + 测试 | A3 |
| `internal/api/logs.go` + `internal/api/logs_test.go`(如需) | A4 后端 |
| `web/src/components/admin/SystemLogs.vue`、`web/src/styles/system-logs.css` | A4 前端 |
| `internal/api/web/**` | embed 重建产物（前端改动后必须更新） |

**零改动（越界即失败）**：`internal/buildinfo/version.go`、`Dockerfile`、`Makefile`、`.golangci.yml`、
`go.mod`/`go.sum`、`drivers/template/**`、`data/**`。
