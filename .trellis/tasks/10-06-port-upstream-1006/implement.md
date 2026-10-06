# Implementation Plan：移植上游 4 项适用修复

## Overview

顺序：**基线（P0）→ A1 → 门禁 → A2 → 门禁 → A3 → 门禁 → A4（含 embed 重建）→ 门禁 →
端到端 → 归档**。四项互不依赖，任一失败可单独回滚。

网络：本轮不需要再访问上游（diff 已在调查轮取全）；收尾推送代码（用户已授权）。

---

## Phase 0: 基线

- [x] 0.1 `git status --short`（期望仅任务目录）、`main == origin/main`
- [x] 0.2 `deadcode ./cmd/litepan` = 7、`--enable=unused` = 0（`PATH` 含 `/root/go/bin`）
- [x] 0.3 四项现状证据复核（research.md §2 的 grep 结果复跑一遍）

---

## Phase A1: 上传强制 HTTP/1.1

- [x] A1.1 `internal/httpx/client.go`：新增 `NewUploadClient`（含 ALPN/Protocols/TLSNextProto 处理）
- [x] A1.2 `internal/driver/driver.go`：`Config.UploadUseHTTP2`
- [x] A1.3 `drivers/115_Open`：删 `newOSSUploadHTTPClient`，`Init` 改调 `NewUploadClient(..., 60s, ...)`
- [x] A1.4 `drivers/189Cloud/driver.go`：同样接入
- [x] A1.5 测试：`httpx` 侧新增 HTTP/1.1 断言；改造 `drivers/115_Open/upload_retry_test.go` 的既有用例
- [x] A1.6 **GA1**：`go build`/`vet`/`go test ./internal/httpx/ ./drivers/115_Open/ ./drivers/189Cloud/`
- **回滚**：`git checkout HEAD -- internal/httpx/client.go internal/driver/driver.go drivers/115_Open drivers/189Cloud/driver.go`

---

## Phase A2: playback 多段 Range

- [x] A2.1 `internal/playback/range.go`：`errRangeNoOverlap` + `byteRange` + `parseRanges` + 越界/TriomSpace 处理
- [x] A2.2 `internal/playback/multipart_range.go`（新）：`streamMultipartRanges` 与 `rangeByteCounter`
- [x] A2.3 `internal/playback/streamer.go`：Range 分支改用 `parseRanges`（三态语义）
- [x] A2.4 `internal/playback/service.go`：`logAction(..., r *http.Request)` + `method`/`range` 字段
- [x] A2.5 测试：移植上游 `streamer_range_test.go`（多段 206 multipart、越界 416、放大防护 200）
      + `parseRanges` 表驱动
- [x] A2.6 **GA2**：`go test ./internal/playback/`
- **回滚**：`git checkout HEAD -- internal/playback/range.go internal/playback/streamer.go internal/playback/service.go && rm internal/playback/multipart_range.go`

---

## Phase A3: 189Cloud 同步盘根

- [x] A3.1 `transport.go`：`syncRootID` + `apiParentID` 映射 + `is189AuthExpiredResponse`（等价重构）
- [x] A3.2 `driver.go`：`ListFiles` 重写 `0`→`syncRootID`；`GetFileInfo` 认 `syncRootID`
- [x] A3.3 `ops.go`：`containsRoot` 计入 `syncRootID`；`RenameFile` 改用 `containsRoot`
- [x] A3.4 `upload.go`：`createRapidUpload` 用 `d.apiParentID(parentID)`
- [x] A3.5 测试：`apiParentID`/`containsRoot`/`GetFileInfo(sync:0)` 三个分支
- [x] A3.6 **GA3**：`go test ./drivers/189Cloud/`
- **回滚**：`git checkout HEAD -- drivers/189Cloud`

---

## Phase A4: 日志详情恒返回 + 复制

- [x] A4.1 `internal/api/logs.go`：`Details` 移入 DTO 字面量
- [x] A4.2 前端 `SystemLogs.vue`：`copyLog()` + 「复制日志」按钮（复用 `copyTextToClipboard`）
- [x] A4.3 前端 `system-logs.css`：`log-row__detail-head` / `log-row__copy` 样式
- [x] A4.4 **重建 embed**：`cd web && npm run build`（并确认 `internal/api/web/**` 有预期变更）
- [x] A4.5 测试：`toLogDTO` 对非 ERROR 条目带 `details` 的断言
- [x] A4.6 **GA4**：`go test ./internal/api/ -run Log`、`npm run type-check`
- **回滚**：见 design.md → Rollback

---

## Phase V: 端到端与全量门

- [x] V.1 `make lint` 0 issues、`GOWORK=off go vet ./...`、`GOWORK=off go test ./...` 全包 ok
- [x] V.2 `cd web && npm run type-check`、`npm run build`（embed 已由 A4.4 重建）
- [x] V.3 基线：`deadcode` ≤ 7、`unused` = 0、`gofmt` 无新增
- [x] V.4 **临时实例**（数据副本 + 非默认端口）：health/登录/日志接口 200；日志页详情字段可见；
      多段 Range 用 curl 直接打（无账号时用 `Range` 头对静态/健康响应验证 416/200 分支不可行 → 改为
      **单测为准**，在此记录该限制）
- [x] V.5 越界核对：`version.go`/`Dockerfile`/`Makefile`/`.golangci.yml`/`go.mod` 零改动；`:5211` 与 `data/` 未动
- [x] V.6 提交（消息带 `[task:10-06-port-upstream-1006]`）

---

## Phase F: 收尾

- [x] F.1 `skill trellis-check` → `mark-check` → 勾选验收项 → `pre-archive`
- [x] F.2 `task.py archive 10-06-port-upstream-1006 --skip-branch-validation`（auto-commit 已知会缺锚点 → 手工补交）
- [x] F.3 归档收尾任务 `10-06-upstream-followup-1006`（其决策已落：四项全做 / 2FA 留档）并一并提交
- [x] F.4 `add_session.py --no-commit` + 手工提交（含锚点）
- [x] F.5 `git push origin main`

---

## Review Gates

| 门 | 判据 |
|---|---|
| GA1 | 上传客户端为 HTTP/1.1（ALPN/Protocols/ForceAttemptHTTP2）且无整段超时；115/189 接入 |
| GA2 | 多段 → 206 multipart；越界 → 416；放大防护 → 200 全量；单段行为不变 |
| GA3 | `syncRootID` 三处映射生效；普通 ID 透传不变 |
| GA4 | 非 ERROR 日志带 `details`；前端复制按钮与样式就位；embed 已重建 |
| GV | 全量门 + 基线 + 端到端 + 无越界 |

## Validation Commands（汇总）

```bash
export PATH=/usr/local/go/bin:/root/go/bin:$PATH GOWORK=off GOPROXY=https://goproxy.cn,direct
cd /root/LitePan
gofmt -l $(git diff --name-only | grep '\.go$')            # 新增格式问题应为 0
make lint && go vet ./... && go test ./...
cd web && npm run type-check && npm run build && cd ..
deadcode ./cmd/litepan | wc -l                              # ≤ 7
golangci-lint run --enable=unused -c .golangci.yml ./...    # 0 issues
```

## 执行记录（实施后回填）


---

## 执行记录（2026-10-06 实测）

### 落地情况

| 项 | 结果 | 证据 |
|---|---|---|
| A1 上传强制 HTTP/1.1 | 落地 | `internal/httpx/client.go` 新增 `NewUploadClient`；`internal/driver/driver.go` 增 `UploadUseHTTP2`；115 删 `newOSSUploadHTTPClient` 并改用新客户端（60s 响应头兜底）；189 同步接入 |
| A2 playback 多段 Range | 落地 | `internal/playback/range.go`（55→110 行：`parseRanges`/`byteRange`/`errRangeNoOverlap`）、新增 `multipart_range.go`、`streamer.go` 三态分支 |
| A3 189Cloud 同步盘根 | 落地 | `transport.go` 增 `syncRootID`+`apiParentID` 映射+`is189AuthExpiredResponse`；`driver.go` 列表重写与 `GetFileInfo` 分支；`ops.go` `containsRoot`/`RenameFile`；`upload.go` 秒传走 `apiParentID` |
| A4 日志详情恒返回 + 复制 | 落地（前端按本方卡片布局适配） | `internal/api/logs.go` `Details` 恒返回；`SystemLogs.vue` 加「复制日志」+ 放开详情条件；`system-logs.css` 加两块样式；**embed 已重建**（93 个产物文件） |

### 与上游的刻意偏差

1. **A2 未移植 `service.go` 的 `logAction` 改动**：上游把它改成接收 `*http.Request` 以记录 `method`/`range`，
   但本方的 playback 服务**没有 logger**（`SetLogger`/`logAction`/`log` 字段在本方早前瘦身时已移除），
   没有可改的对象。要恢复该诊断能力需先决定"是否给 playback 注入 logger"，属另一个决策，本轮不动。
   > 因此 A2 的 `streamer.go`/`range.go`/`multipart_range.go` 三个文件是**逐字取自上游**（本方与上游父提交逐字节一致），
   > 唯独 `service.go` 保持本方现状。
2. **A4 前端按本方结构适配**：本方日志页是卡片式（`log-card__*`）而非上游行式（`log-row__*`），
   故"复制日志"按钮落在 `log-card__details-head` 里，样式复用本方变量；`canShowDetails` 放开 `level >= 40` 限制。

### 质量门与基线（逐条实测）

- `make lint` **0 issues**、`go vet ./...` exit=0、`go test ./...` **26 包全 ok 0 FAIL**、
  `cd web && npm run type-check` exit=0、`npm run build` 成功（embed 100 文件压缩完成）。
- 基线：`deadcode ./cmd/litepan` = **7**（与基线同值）、`--enable=unused` = **0 issues**、
  `gofmt -l .` = 15（与基线同数，均既有脏文件）、改动文件 gofmt 全干净。
- 新增测试：`internal/httpx/upload_client_test.go`（3 例）、`internal/playback/range_test.go`（表驱动 15 例 + 2）、
  `internal/playback/streamer_range_test.go`（上游原文件：多段 multipart / 放大防护 / 416）、
  `drivers/189Cloud/sync_root_test.go`（5 例）、`internal/api/logs_dto_test.go`（1 例）。
  **反向验证**：A1 的 HTTP/1.1 断言在"未锁协议"时必失败（`ForceAttemptHTTP2`/`Protocols` 断言）；
  A2 的多段/防护用例在旧 `parseSingleRange` 实现下无法编译通过（`parseRanges` 不存在）；
  A3 的 `apiParentID(sync:0)` 在旧实现下返回 `sync:0` 而非 `"0"`（用例必失败）；
  A4 的 `toLogDTO` 用例在旧实现下 INFO 条目 `Details == nil`（必失败）。
- 越界核对：`internal/buildinfo/version.go`、`Dockerfile`、`Makefile`、`.golangci.yml`、`go.mod`/`go.sum`
  均 **0 变更**；`:5211` 容器（v0.0.48，Up）未动。

### 端到端（临时实例，数据副本 + 127.0.0.1:35231）

- health 200、登录 200、`/api/files/upload/runtime` 200、`/api/admin/settings` 200、
  `/api/admin/system-config` 200、本地映射配置 200；`level=ERROR` = **0**。
- **A4 实证**：`GET /api/logs` 返回的 6 条里 **4 条带 `details`，全部是 INFO 级**
  （如"管理员登录成功"带 `ip`/`username`、"HTTP 服务已监听"带 `addr`）——旧实现下这些都不带 details。
- A2/A3 的运行时行为**未做端到端**：A2 需要可播放账号与真实播放器拖拽（本机实例 0 账号），
  A3 需要真实 189 账号 —— 二者以单测为最终判据，此限制如实记录，不声称已验证。
- 现场已清理（临时实例停止、`/tmp/port48` 删除）；`data/` 未做任何写操作（容器自身运行写入除外）。

### 提交与归档

- 提交消息带 `[task:10-06-port-upstream-1006]`；四项改动 + 测试 + embed 重建在同一提交，可按 design.md 的
  Rollback 逐项回退。
