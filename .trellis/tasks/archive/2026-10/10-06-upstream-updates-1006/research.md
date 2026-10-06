# 上游 LitePan 更新调查（2026-10-06）

**调查范围**：`Ponphil/LitePan` 自本方上次核对点 `42a3ee9a`（2026-09-20）到当前 `main` `e0e29c0e`（2026-10-02）。
**本轮只读**：不改代码/配置/部署/数据，不提交上游内容（仅本任务记录与 journal）。

---

## §0 结论摘要

上游 14 天内线性推进 **21 个提交**（未重写历史，tag 由 `v0.5.6-beta` → **`v0.5.7-beta`**，仍**无 GitHub Release**）。
按"本方是否在线使用该路径"分类：

| 分类 | 数量 | 提交 |
|---|---|---|
| **适用（建议移植）** | 4 | `7e0c0040` 上传强制 HTTP/1.1、`5df900c5` playback 多段 Range + 诊断、`214e027d` 189Cloud 同步盘根、`52fcf32e` 日志详情全覆盖 |
| **待拍板（新能力）** | 1 | `5db84a63` 后台两步验证（2FA/TOTP，~1200 行含前端） |
| **混合（需逐项取舍）** | 1 | `44796906` 0.5.7 收口（2FA 跟进 + `NewUploadClient` 重构 + 删死代码 + 测试清理） |
| **不适用（依赖本方已删能力 / 上游独有）** | 15 | `36d715f5`、`6f076163`、`f237ab76`、`e0e29c0e`、`96edffaf`、`31081290`、`635f4d41`、`f892db55`、`7778b040`、`6239018a`、`dc1258f8`、`d552fa5c`、`fe7c3204`、`6ba253c2`、本表外已并入上行的部分 |

**一句话建议**：值得移植的是 4 项**小而实**的修复（合计约 300 行 + 测试），其中"上传强制 HTTP/1.1"与"playback 多段 Range"直接对应本方在用的上传/播放路径，优先级最高；2FA 属新能力，需你拍板后再另开任务。

---

## §1 上游增量事实

| 项 | 值 |
|---|---|
| 上次核对点 | `42a3ee9a`（2026-09-20，"优化通知轮询"） |
| 当前上游 HEAD | `e0e29c0e`（2026-10-02，"修复 STRM 刮削缺图误报下载失败"） |
| 增量 | **21 提交**，2026-09-20 → 2026-10-02 |
| 历史 | `42a3ee9a` 仍是 `e0e29c0e` 的祖先 → **线性推进，未重写**（`ahead/behind = 0/21`） |
| 规模 | 340 个改动文件、9377+/589-；其中 **178 个是上游构建产物**（`internal/api/web/assets/*.js.gz`） |
| tag / Release | tag：`v0.5.7-beta`（新）、`v0.5.6-beta`、`v0.5.5-beta`；**GitHub Release 仍为 0 个** |
| 上游 repo | `default_branch=main`、`pushed_at=2026-10-02T09:11:57Z` |

**方法**：`git fetch` 上游 `main` → 对每个提交取"改动文件 ∩ 本方 `git ls-files`"，命中 0 的即为上游独有/本方已删能力的改动；
命中 >0 的逐条读 diff 判定。**178 个构建产物文件全程忽略**（本方前端自行构建）。

---

## §2 适用清单（含本方现状证据）

### A1 `7e0c0040` 优化部分驱动上传慢的问题（+ `44796906` 的收口重构）— **建议移植**

**上游改法**：新增 `httpx.NewUploadClient(base, headerTimeout, useHTTP2)`，**上传默认强制 HTTP/1.1**
（`ForceAttemptHTTP2=false`、清空 `TLSNextProto`、`Protocols.SetHTTP1(true)`、并把 ALPN `NextProtos` 限制为 `http/1.1`）；
只有驱动在 `driver.Config` 里显式声明 `UploadUseHTTP2` 时才走 HTTP/2。115 与 189 驱动改为调用它，并删掉 115 侧的
`newOSSUploadHTTPClient` 包装。

**本方现状（缺口证据）**：

- `internal/httpx/client.go:53` 只有 `NewStreamingClient`，**没有** `NewUploadClient`；
- `drivers/115_Open/upload.go:34-35` `newOSSUploadHTTPClient` → `httpx.NewStreamingClient(base, 30*time.Second)`；
  `drivers/189Cloud/driver.go:86` 同样直接用 `NewStreamingClient(d.client, 30*time.Second)`；
- `grep -rn UploadUseHTTP2` = **0 命中**（未移植）。

**为什么值得**：这正是本方上传数据面（115/189 大文件上传）走的客户端；上游判断 HTTP/2 对这两个上游的上传是负优化。

**移植注意**：本方 `drivers/115_Open/upload_retry_test.go` 有 `TestNewOSSUploadHTTPClientHasNoTotalTimeout`
（断言"不限制整段传输时长"），随包装函数一起调整；`44796906` 把实现改成"先 `NewStreamingClient` 再改 transport"，
移植时**直接取最终形态**，不要先照搬 `7e0c0040` 的初版。

### A2 `5df900c5` 优化 STRM Seek 兼容性 → 实为 **playback 多段 Range** 支持 — **建议移植**

**上游改法**：`internal/playback/range.go` 从 55 行扩到 110 行：新增 `byteRange` 类型与
**`parseRanges()` 多段解析**（`bytes=0-99,200-299` 这类播放器请求）；防护：分段数 ≥ 32 或累计长度 > 文件长度时
**忽略 Range 交给全量路径**（避免小请求放大上游读取）；越界段跳过、全部越界返回 `errRangeNoOverlap`；
`parseSingleRange` 对 `parts` 做 TrimSpace。另把 `service.go` 的 `logAction` 改为接收 `*http.Request`，
日志增加 `method` 与 `range`（>256 字符截断）——**播放 seek 排查的关键信息**。

**本方现状（缺口证据）**：

- `internal/playback/range.go` 共 55 行，只有 `parseSingleRange`；`grep parseRanges|errRangeNoOverlap` = 0；
  实测 `diff <(git show 5df900c5^:internal/playback/range.go) internal/playback/range.go` → **完全一致**（= 上游父版本，未移植）；
- `internal/playback/service.go` 的 `logAction` 仍只记 `user_agent`（无 method/range）。

**为什么值得**：本方保留播放代理（`internal/playback/*` 在线使用）；多段 Range 是 VLC/PotPlayer 等播放器拖拽 seek 时的常见请求形态。

### A3 `214e027d` 部分驱动问题修复（189Cloud）— **建议移植**

**上游改法**（`drivers/189Cloud/*`）：

1. 新增 `syncRootID = "sync:0"` —— 天翼"同步盘"实际 ID 是 `0`，与公共层的"根目录别名 0"冲突，故用独立标识区分：
   `ListFiles` 里把 `item.ID == "0"` 重写为 `syncRootID`；`GetFileInfo` 认 `syncRootID`（返回"同步盘"目录项）；
   `apiParentID` 把 `sync:0` 映射回上游要的 `"0"`；`containsRoot` 把 `syncRootID` 也算根；
2. `upload.go` 秒传 `createUploadFile.action` 的 `parentFolderId` 改用 `d.apiParentID(parentID)`（原来直接传 `parentID`，
   对根/同步盘别名会传错）；
3. 鉴权失效判定抽成 `is189AuthExpiredResponse(status, data)`（**内容与本方现有条件等价**，见 §3）。

**本方现状（缺口证据）**：`grep syncRootID` = 0 命中；`drivers/189Cloud/ops.go:259-266` 的 `containsRoot`
只判 `root`/`"0"`/`"/"`；`createRapidUpload` 直接传 `parentID`（未走 `apiParentID`）。

### A4 `52fcf32e` 完善日志详情与复制 — **建议移植（小）**

**上游改法**：`internal/api/logs.go` 的 DTO 组装把 `Details` 从"仅 ERROR 级"改为**恒返回**（结构性调整，
非条件分支）；配套前端 `web/src/components/admin/SystemLogs.vue` + `web/src/styles/system-logs.css`（详情/复制）。
`Details` 字段本就存在（`logs.go:24`，`json:"details,omitempty"`），所以后端改动很小。

**本方现状（缺口证据）**：`internal/api/logs.go:166-168` 仍是 `if e.Level >= logx.LevelError { dto.Details = e.Details }`。

**移植注意**：后端一行级改动；前端两文件需要按本方前端版本单独评估（本方前端与上游已有分叉）。

---

## §3 已收敛 / "别跟着上游做"

| 项 | 说明 |
|---|---|
| 189 鉴权失效判定含 400 | 上游 `214e027d` 把它抽成 `is189AuthExpiredResponse`；**本方 `drivers/189Cloud/transport.go:213,253` 已经是等价条件**（`401` 或 `(200\|400) 且 payload 命中失效特征`）→ 只需取"抽函数"这层重构，无行为变化 |
| 上游删 `internal/auth/managed_accounts.go: accountName`（`44796906`） | **本方不能跟着删**：本方 `refresh_runner.go:99`、`scheduler.go:259,341`、`notification/service.go:95` 三处在用（上游侧才是死代码） |
| 上游删 `drivers/115_Open/full_list_test.go`（`6ba253c2`） | 上游把该测试删了；**本方保留并已扩写到 12 个用例**（v0.0.48 交付）→ 不跟随 |
| 上游版本号/UA（`6ba253c2`） | 上游 `buildinfo.Version`、`httpx.AppVersion` 是 `v0.5.7-Beta`；**本方 `AppVersion` 由 `buildinfo.Version` 派生**（本方自管 `v0.0.x`）→ 无需动作 |
| `44796906` 的 `router_test.go` 改动 | 上游测试自身写法（`r.ResponseRecorder.Body` → `r.Body`）→ 本方测试结构不同，无需跟随 |

---

## §4 待拍板：`5db84a63` 后台管理增加 2FA（TOTP）

- 规模：`internal/adminauth/service.go` + `internal/api/auth.go|router.go` + 前端 `TwoFactorSettings.vue`（**422 行新建**）、
  `LoginView.vue`、`SystemSettings.vue`、`web/src/api/auth.ts`；共 1225+ 行（不含构建产物）。
- 形态：`Service.Login` **签名变更**（增 `code, challenge`），`LoginResult` 增 `TwoFactorRequired`/`Challenge`，
  新增 `twoFactorEnabled/verifyTwoFactorCode/allowTwoFactorAttempt`（含 5 分钟限流）、`SystemConfig.TwoFactorEnabled`；
  前端登录页进入"输入动态码"分支。
- 判断：这是**新能力**（不是修复），且改动登录主链路 + 前端交互；与本方 `v0.0.48` 刚上线的 adminauth 配置缓存有交集。
- 建议：**保持留档**（与「高级定时」同一处理方式），若要上，另开任务设计（含限流/恢复码/备份恢复联动等边界）。

---

## §5 不适用清单（15 提交，逐条理由）

| 提交 | 标题 | 不适用理由（本方已删能力 / 上游独有） |
|---|---|---|
| `36d715f5` | strm改原图刮削 | `internal/strmscrape/*`、`internal/mediaorganize/tmdb` —— STRM/刮削已删 |
| `6f076163` | 优化目录整理多项识别问题 | `internal/aiorganize/*`、`internal/mediaorganize/planner/*` —— 媒体整理已删 |
| `f237ab76` | 修复整理电影误加季格式 | `internal/mediaorganize/planner` —— 同上 |
| `9ec8349c` | 修复分类整理联动目录校验 | 改的是 `validateOrganizeToStrm`（"整理任务→STRM 任务"目录校验改用 `organize.TargetPathCandidates`）；本方 `internal/automation` **已无 organize/strm 联动路径**（`grep validateOrganizeToStrm\|TargetPathCandidates` = 0 命中）→ 无对应代码可改 |
| `e0e29c0e` | 修复 STRM 刮削缺图误报下载失败 | `internal/strmscrape/*` —— 同上 |
| `96edffaf` | 完善自动联动离线触发 | `internal/offlinedownload/*` —— 离线下载已删（本方 `a576484` 移除） |
| `31081290` | 垃圾清理支持清理已读通知 | `internal/spacecleanup/*`、`internal/store/database_garbage.go` —— 空间/缓存整理已删 |
| `635f4d41` | 优化 WebDAV 认证性能 | `internal/share/dav/*` —— 分享服务端已删（本方 WebDAV **驱动**在，但改动不在此） |
| `f892db55` | 123open支持分享链接 | 新增 `internal/cloudshare/*`、`internal/api/cloud_share.go`、`internal/driver/share.go`、`drivers/123_Open/share.go` —— 分享已删 |
| `7778b040` | 分享能力通用化并接入夸克 | 同上（`drivers/Quark/share.go` 等） |
| `6239018a` | 新增 strm 删除监控及统一增强工具卡片 | 主体为 STRM 删除监控（`fsnotify` 新依赖 + `NotificationCategoryStrmDeleteConfirm`）+ 增强工具卡片；**唯一可选项**是 Dockerfile 的 `ARG DEBIAN_IMAGE` 构建参数化（与 FUSE/strm 同批，价值低） |
| `dc1258f8` | 修复增强工具搜索结果展示 | 纯前端 `AuxToolsManagement.vue`/`CloudToolsPanel.vue` —— 增强工具（STRM 相关）已删 |
| `d552fa5c` | 优化跨盘传输下载稳定性 | 核心是跨盘传输（新增 `playback/transfer.go`、`upload/download_checkpoint.go`，`SourceTypeCrossTransfer`、`.download` 断点文件）；改到的共享文件（`playback/account_range_limiter.go` 冷却、`range_proxy.go` generation、`upload/worker.go`）都服务于该能力 —— 跨盘已删 |
| `fe7c3204` | 0.5.7 发布前收口 | `internal/mediaorganize/*`、`internal/strm*`、`internal/offlinedownload`、`drivers/Guangya`（本方无此驱动）、`upload/download_checkpoint.go` |
| `6ba253c2` | Releases v0.5.7 | 上游版本号/UA + 删本方保留的测试（见 §3） |

> 表内 15 行即 §0 统计的"不适用 15"；`52fcf32e`/`5df900c5`/`214e027d`/`7e0c0040`（适用）与
> `5db84a63`（待拍板）、`44796906`（混合）分别见 §2/§4，不在本表重复。

---

## §6 移植建议（供拍板）

1. **优先级 1（改动小、路径在线、风险低）**：A1 上传强制 HTTP/1.1、A4 日志详情恒返回。两者的验证方式是
   "上传一个大文件对比速度 / 打开日志页看非 ERROR 条目的 details"，不需要真实账号也能跑单测的部分先跑单测。
2. **优先级 2（需要真实播放器验证）**：A2 多段 Range。单测可覆盖 `parseRanges` 的边界（多段、越界、放大防护），
   端到端要靠本机播放器拖拽 seek（需账号，或复用数据副本实例）。
3. **优先级 3（驱动行为）**：A3 189 同步盘根。无真实 189 账号则**只能读码移植 + 单测**，行为验证留档。
4. **不做**：2FA（新能力，待你决定）、跨盘/STRM/媒体整理/垃圾清理/分享/离线下载相关的任何回填。
5. **纪律不变**：按本方代码 craft（上游删了 `drivers/template` 等本方保留的东西，**禁止 `git apply` 上游补丁**）；
   每批走全量质量门；上游测试按需取用。

---

## §7 本轮边界披露

**网络动作**（仅 `github.com/Ponphil/LitePan`，只读）：

- `git fetch --no-tags https://github.com/Ponphil/LitePan.git main:refs/remotes/upstream/main`
- `gh api repos/Ponphil/LitePan/tags`、`.../releases`、`repos/Ponphil/LitePan`（元信息）
- 未访问其它任何主机；未触碰 `:5211` 容器与 `data/`。

**写操作**（全部为流程留痕或状态修复，未改任何代码/配置/部署/数据）：

1. 本任务目录（`prd.md` + 本文件 + 归档）；
2. `git update-ref refs/remotes/upstream/main e0e29c0e` —— fetch 因本地 ref 目录异常（旧 ref 被删、新 ref 未写入）
   未能自动更新，手工把 ref 指向已取回的提交；**不涉及任何受跟踪文件**（`git status` 前只显示本任务目录）；
3. 收尾的 journal 提交与推送。

**遗留观察**：`git fetch` 该次失败的根因是 `.git/refs/remotes/upstream/` 目录曾出现解析异常（无锁文件残留，
目录存在但无 `main`）。若再次出现，先删 `refs/remotes/upstream/main` 再 fetch。
