# Implementation Plan：安卓两跳握手

## Overview

实现 → 单测（含反向验证）→ 质量门 → 归档。**不发版、不部署**（另行授权）。

---

## Phase 1: 实现

- [x] 1.1 新增 `internal/playback/handoff.go`：
  - `isAndroidBrowserUA(ua string) bool`（KD1 第 2 条标记表）
  - `handoffTracker`（mutex + `seen map[string]time.Time` + `now func() time.Time` + `window`），
    方法 `first(key string) bool`；容量保护：`len(seen) > 512` 时清理过期项
  - `(s *Service) tracker()` 懒初始化访问器
  - `writeHandoffStub(w http.ResponseWriter, intent Intent)`：复用 `writeDynamicRedirectHeaders`，
    设 `Content-Disposition`（有 `intent.FileName` 则附 `filename*=UTF-8''`）、`Content-Type: application/octet-stream`、
    `Content-Length: 0`、状态 200、空体
- [x] 1.2 改 `internal/playback/service.go`：
  - `NewService` 初始化 tracker
  - `ServeHTTP` 中 `action == ActionRedirect` 分支改为：
    ```go
    if action == ActionRedirect {
        if s.handoffStub(w, r, req, intent) { return nil }
        writeRedirect(w, r, res, intent)
        return nil
    }
    ```
    其中 `handoffStub` 实现 KD1 的 5 个条件（`!intent.Inline`、GET、安卓浏览器 UA、窗口首次）
- [x] 1.3 `G1`：`go build ./...` + `gofmt -l internal/playback` 干净

## Phase 2: 单测

- [x] 2.1 新增 `internal/playback/handoff_test.go`，覆盖：
  1. 安卓浏览器 UA 首次 → `200`、`Content-Disposition` 含 attachment、`Content-Length: 0`、**无 `Location`**
  2. 同资源第二次（同 UA）→ `302` + `Location` 非空
  3. 同资源第二次（`AndroidDownloadManager/14 (Linux; U; Android 14)`）→ `302`
  4. 桌面 Chrome UA 首次 → `302`
  5. iOS Safari UA 首次 → `302`
  6. 安卓浏览器 UA + `HEAD` → `302`
  7. `intent.Inline = true`（播放）→ 直接 `302`（**不得**发空附件）
  8. 另一个 fileID 首次 → 仍是空附件（key 隔离）
  9. 注入时钟跨过 90s 窗口 → 再次握手（空附件）
  10. `intent.ForceProxy = true` → 走 stream（不经握手、不发空附件）
- [x] 2.2 **反向验证**：临时把 `handoffStub` 恒返回 false（或注释掉调用），确认用例 1/8/9 失败；
      恢复后再跑，确认全绿 —— 证明测试真在测该逻辑
- [x] 2.3 `G2`：`go test -count=1 ./internal/playback/` 全 ok

## Phase 3: 质量门与边界

- [x] 3.1 `make lint`（0 issues）、`go vet ./...`、`go test -count=1 ./...`（26 包）、`cd web && npm run type-check`
- [x] 3.2 **G3 边界**：`git status --short` 仅 `internal/playback/**` 与任务目录；
      `web/src`、`internal/api/web`、`drivers`、`internal/store`、`go.mod`、compose **零改动**
- [x] 3.3 基线：`deadcode ./cmd/litepan` ≤ 7、`golangci-lint --enable=unused` 0、`gofmt -l .` ≤ 15

## Phase 4: 归档

- [x] 4.1 回填执行记录 → `skill trellis-check` → `mark-check` → 勾选验收 → `pre-archive` → `archive`
- [x] 4.2 `add_session.py` + 提交（含任务锚点）；**不推送**（用户未授权本轮推送）

## Validation Commands

```bash
export PATH=/usr/local/go/bin:/root/go/bin:$PATH GOWORK=off GOPROXY=https://goproxy.cn,direct
cd /root/LitePan
go test -count=1 -run 'Handoff' -v ./internal/playback/ | tail -30
make lint && go vet ./... && go test -count=1 ./... && (cd web && npm run type-check)
git status --short
```

## Review Gates

| 门 | 判据 |
|---|---|
| G1 | 编译通过、gofmt 干净 |
| G2 | 10 条单测全绿 + 反向验证（临时关闭握手 → 用例失败） |
| G3 | 无越界改动；前端/embed/驱动/存储零改动 |
| G4 | 真机验证（**交付后在用户手机上做**，不由本轮完成）：手机点下载能拿到完整文件；电脑端行为不变 |

## Out of Scope

- 发版/推镜像/部署生产（另行授权）
- 为安卓系统组件补齐"无 cookie 下载"的短期令牌（若真机验证发现 401 再单开任务）
- 115 驱动取链逻辑、代理模式、播放路径的其它改动

## 执行记录（实施后回填）


---

## 执行记录（2026-10-16 实测）

| 门 | 结果 |
|---|---|
| G1 | `go build ./...` OK、`gofmt -l internal/playback` 干净 |
| G2 | **11 条单测全绿**（`TestHandoff*` / `TestIsAndroidBrowserUA` / `TestForceProxyStillStreams` / `TestHandoffTrackerPrunesExpired`） |
| G2 反向验证 | 临时把 `handoffStub` 置为恒 false → `TestHandoffFirstAndroidBrowserGetsEmptyAttachment`、`TestHandoffWindowIsPerFile`、`TestHandoffWindowExpiryAllowsNewStub` **失败**；恢复后全绿 → 测试确实在测握手逻辑 |
| G3 边界 | 改动仅 `internal/playback/handoff.go`(新)、`handoff_test.go`(新)、`service.go`(改)；`web/src`、`internal/api/web`、`drivers`、`internal/store`、`go.mod`、compose **零改动** |
| 质量门 | `make lint` 0 issues；`go vet ./...` exit=0；`go test -count=1 ./...` **26 包全 ok 0 FAIL**；`cd web && npm run type-check` exit=0 |
| 基线 | `deadcode ./cmd/litepan` = 7（同基线）、`--enable=unused` 0 issues、`gofmt -l .` = 15（同基线，均既有脏文件） |
| spec 同步 | 已把"115 直链签名绑定取链 UA + 安卓交接给系统下载器 + 两跳握手"写入 `.trellis/spec/backend/backend/driver-development.md`（新增「下载直链与 302：UA 绑定」一节） |

### 实现要点回填

1. 握手判定（`handoffStub`）：`GET` + **非 Inline**（播放/内联不吃空附件）+ 安卓浏览器 UA + 该资源窗口内首次；
   在 `ServeHTTP` 的 `ActionRedirect` 分支内调用，因此**代理模式（ActionStream）根本不经过该分支**。
2. 防循环：`handoffTracker` 按 `(accountID, fileID)` 记 90s 窗口；窗口内第二次及以后一律 302。
3. 已知代价（design KD3 已记录）：第一跳仍会解析一次（使用浏览器 UA，该链随后被丢弃），
   换来"代理模式/播放路径零影响"；解析缓存按完整 UA 分变体，故第二跳拿到的是**按组件 UA**新取的链。
4. 未做（Out of Scope）：安卓系统组件若不带 cookie 会 401 的问题——等真机验证结论再决定是否加短期令牌。

### 真机验证（未完成，交付后由用户执行）

手机上（同一网络或流量均可，签名不绑 IP）点下载 → 期望：通知栏出现下载、文件完整；
电脑端行为与改动前一致（仍是 302 直链）。**若安卓把空附件当成已完成下载而不发起第二跳，则需回退本提交。**

### 未做/限制

- **未发版、未部署**（生产 NAS 仍是 v0.0.49）——发版与部署需用户另行授权。
- 无数据库/配置/embed 变更，回滚 = 回退提交 + 重新部署上一版镜像。
