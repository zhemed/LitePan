# 上游 LitePan 更新调查报告（2026-09-12）

> 任务：`09-12-investigate-upstream-updates`｜口径：**只读**（唯一副作用：`git fetch origin` 更新远程跟踪 ref，未改工作区、未 merge、未 cherry-pick）
> 上游：`Ponphil/LitePan`（`git remote origin`）｜本方：`zhemed/LitePan`（精简分支，仅 3 驱动）
> 调查时间：2026-09-12 15:2x–16:0x（本地时区）

---

## 1. 结论摘要

1. **上游最新**：`main` = `46a0a89`（2026-09-12 11:23 `播放诊断及直读文案`），tag/release **`v0.5.5-beta`**（同 commit）；官方 changelog 已发布 `v0.5.5`（2026-09-12）。
2. **⚠️ 上游重写了历史**：本次 `git fetch` 报 `+ 374affd...46a0a89 main -> origin/main (forced update)`。上游当前 lineage 的根 `6313ef7 新版首次提交` 与本方根 `a69bb6e` **同日期、同标题、同 tree**（`b5bba895…`），但 SHA 全变；08-29 之前 133 个提交内容一一对应、SHA 全不同。**结论：上游在 2026-09-05 18:11（我方上次 fetch）之后 force-push 了重写后的历史，双方已无共同祖先**——未来同步**不能 merge/rebase**，只能做「内容级（tree diff）对照移植」。旧的 `origin/main`=`374affd` 与新 `209103c` tree 相同（`fad654ee…`），佐证是重写而非内容变更。
3. **自 fork 点（08-29 `f71a522`≡我方 `4c160d9`，tree `a5e94c7d`）以来，上游共 47 个提交**（08-30 → 09-12）。按「改动文件是否出现在本方 tree」评分：**11 个提交 0 个适用文件**（STRM/跨盘/Emby/AI/清理/多驱动），其余多数为前端样式与不适用模块。
4. **与本方 0.0.32~0.0.37 六项修复无正面冲突**：上游**没有**本方的冷却等待/熔断分级键/冷却日志抑制/终端桶展开机制（证据见 §5）；唯一同文件竞争是 `收口5-上传任务`（`5255775`），经逐行核对为**纯重构**（抽取 `isActiveUploadStatus`/`pausedMessage`/`taskPhase` 等 helper，无行为修复），移植价值低、冲突成本高。
5. **值得移植的候选（9 项，按优先级见 §6）**：**P1** `869974b 修复网盘上传超时`（`httpx.NewStreamingClient` + 189/115 上传专用客户端 + 115 OSS 分片重试，5 个适用文件，直击本方生产上传超时）；**P2** `d545e47` 中的 `internal/api/error_log.go` 客户端断连静默（1 文件，与本方日志卫生方向一致）；**P3** `99ea858 收口1-认证模块`（认证调度/收尾统一 + 日志漂移抑制，6 文件，~220 行差异）；**P4** `df86352` 账号级目录缓存失效；**P5** `b5c9308` 慢后台日志与 accountprofile；**P6** `20b66de` 目录整理 TMDB 误匹配（本方保留 `mediaorganize/rules`）；**P7** `ae75882` 上传取消语义/定时器子集；**P8** `adca0ee` 铃铛 SSE 推送；**P9** `46a0a89` 播放诊断。
6. **本轮零实施**：不 merge、不 cherry-pick、不改任何代码；如需移植，另建任务并按 0.0.x 递增发版。

---

## 2. 调查方法与口径

| 步骤 | 命令/方法 | 产出 |
|---|---|---|
| 基准获取 | `git fetch origin`（→ forced update）、GitHub API `commits/tags/releases` | 上游最新 sha/时间/tag |
| 历史关系判定 | `git merge-base --all main origin/main`、根提交 tree 对比、`git rev-parse <old>^{tree}` vs `<new>^{tree}` | 「unrelated histories」+ 重写证据 |
| fork 点定位 | 逐提交比对定位内容等价的双方提交（`4c160d9` ≡ `f71a522`，tree 相同） | fork 点 = 08-29 |
| **三树对照法** | `base=4c160d9（fork 点）`、`ours=main`、`theirs=origin/main`，按共享路径比对 blob | 262 差异文件中区分「上游单方改 97 / 本方单方改 84 / 双方都改 81 / 一致 384」 |
| 提交级适用性 | 每个上游提交列出改动文件，统计「出现在本方 tree 的文件数」 | 47 提交适用性表（§3.2） |
| 逐项精读 | `git show <sha> -- <path>`；对本方对应文件做 grep/并排比对 | 候选与冲突判定（§5/§6） |
| 外部交叉 | `tavily_search` ×8 路 + 官方 changelog `litepan.top/changelog.html` | 版本时间线交叉印证（§10） |

> 说明：因历史被重写，`git log main..origin/main` 这类提交差集不再可靠（会混入全部重写提交）；本报告一律以「内容对照 + 提交级适用文件统计」为准。

---

## 3. 上游现状

### 3.1 版本与基准

| 项 | 值 |
|---|---|
| 上游 `main` | `46a0a89f68895200dfa2caf2dbf8969c4626b2b2`（2026-09-12 11:23 +0800，`播放诊断及直读文案`） |
| 上游 tag（新/最新） | `v0.5.5-beta` → `46a0a89` |
| 上游历史根 | `6313ef7`（2026-07-27 19:31 `新版首次提交`），tree `b5bba895…` ≡ 我方 `a69bb6e` |
| 上游提交总数 | 180（本方 478 = 上游旧 133 + 本方可 345） |
| fork 点 | 上游 `f71a522` ≡ 我方 `4c160d9`（2026-08-29 14:26 `部分显示优化`），tree `a5e94c7d…` |
| 本次 fetch 变化 | `374affd...46a0a89 main (forced update)`；新增 tag `v0.5.5-beta`；**历史整体重写** |
| 我方当前 | `main` `beb07ff`，版本 `0.0.37`（镜像 `ghcr.io/zhemed/litepan:v0.0.37`） |

**历史重写证据链**：① `git merge-base --all main origin/main` 输出为空（无共同祖先）；② 双方根提交 tree 完全相同、SHA 不同；③ 旧 ref `374affd` 与新提交 `209103c`（同为 `秒传星图手机端优化`）tree 相同（`fad654ee…`）；④ fork 点两侧（`4c160d9`/`f71a522`）tree 相同。⇒ 内容连续、SHA 断开 = 上游 force-push 重写历史。**未能验证**：重写原因（未检索到公告/说明，属上游内部操作）。

### 3.2 自 fork 点以来上游 47 提交（全量）与适用性

「适用文件」= 该提交改动的一级路径中，出现在本方 tree 的文件数（已剔除 `internal/api/web/assets/*.gz`、`internal/api/web/index.html` 等上游构建产物）。

| # | commit | 日期 | 标题 | 改动 | 适用 | 判定 |
|---|---|---|---|---|---|---|
| 1 | `acb19c2` | 08-30 | 上传任务批次化 | 40 | 38 | **已移植**（`0.0.18`，旧 sha `de83b46`） |
| 2 | `4a0b868` | 08-30 | 115open 离线支持 ed2k | 3 | 1 | 不适用（115 离线下载模块已删） |
| 3 | `4f703cc` | 08-30 | 修复手动匹配 | 3 | 0 | 不适用（目录整理手动匹配 `executor`/`service_binding` 本方未保留，仅留 `rules/`） |
| 4 | `f03744f` | 09-01 | 光鸭改为本地扫码获取认证 | 10 | 4 | 主体不适用（光鸭驱动已删）；**通用扫码链路**（`api/qr.go`/`driver/qrlogin.go`/`web/src/api/qr.ts`/`QrLoginModal.vue`）本方保留且 189 在用，可另议 |
| 5 | `d6a2e62` | 09-01 | 账号连接检测优化 | 5 | 4 | **已移植**（`0.0.13`，旧 sha `1c71fec`） |
| 6 | `20b66de` | 09-01 | 修复目录整理误匹配 | 4 | 3 | **候选 P6**（`mediaorganize/rules/tmdb.go` + 测试） |
| 7 | `cb332c9` | 09-02 | Releases v0.5.3 | 4 | 4 | 版本号/UA 类，随需 |
| 8 | `882122f` | 09-02 | 部分显示优化 | 15 | 5 | 低价值（注释/公告/分享样式） |
| 9 | `fc5ed4f` | 09-02 | fix115strm | 4 | 1 | 不适用（115 STRM 传输） |
| 10 | `cb25a9b` | 09-03 | 支持 emby 补全媒体信息 | 7 | 5 | 不适用（Emby/自动化动作已删） |
| 11 | `aa47ba2` | 09-03 | 修复 AI 初始化不渲染 | 1 | 0 | 不适用（AI 模块已删） |
| 12 | `a7d6124` | 09-03 | 认证刷新优化 | 3 | 3 | **已移植**（`0.0.13`，旧 sha `8e332f3`：OAuth UA + 状态码） |
| 13 | `4c387e0` | 09-03 | 整理分类支持未命中放指定文件夹 | 6 | 1 | 不适用（目录整理分类已删） |
| 14 | `df86352` | 09-03 | 联动刷新目录改为清理账号缓存 | 6 | 6 | **候选 P4**（`cache.InvalidateAccountDirKeys` + automation） |
| 15 | `afbcb50` | 09-03 | 修复从服务器上传目录错位 | 1 | 1 | **已移植**（`0.0.13`，旧 sha `353b830`） |
| 16 | `eae166d` | 09-04 | STRM 清理元数据 | 2 | 0 | 不适用 |
| 17 | `d545e47` | 09-04 | 跨盘传输功能扩展 | 12 | 3 | 主体不适用；**候选 P2**=`internal/api/error_log.go` 断连静默 |
| 18 | `294a708` | 09-04 | 修复一系列认证问题 | 64 | 39 | **已移植**（`0.0.13` 189 子集 + `0.0.17` auth/守卫/熔断；旧 sha `c7a424c`） |
| 19 | `b5c9308` | 09-05 | 优化后台响应 | 6 | 4 | **候选 P5**（新增 `api/slow_dashboard_log.go` + accountprofile） |
| 20 | `209103c` | 09-05 | 秒传星图手机端优化 | 2 | 0 | 不适用 |
| 21 | `47ad97f` | 09-06 | AI 辅助识别整理优化 | 10 | 0 | 不适用 |
| 22 | `1c5f2b0` | 09-06 | 整理异常调整及 strm 队列优化 | 12 | 4 | 低价值（admin 状态图标/运行状态） |
| 23 | `2e80827` | 09-06 | 修复 115 增强扫描误清理 | 2 | 0 | 不适用（115 STRM 增强） |
| 24 | `05a81ed` | 09-07 | 阶段收口代码瘦身 | 13 | 7 | 低价值（automation 参数传递签名重构） |
| 25 | `ae75882` | 09-07 | 修复部分错误及 Releases v0.5.5 | 61 | 33 | **候选 P7**（`upload/resume.go` 定时器、`api/upload.go` 取消判定、`local_upload` 批次名收敛） |
| 26 | `f133946` | 09-08 | 兼容 strm 路径形式 | 16 | 6 | 主体不适用（STRM/Emby 反代；`internal/file` 新增 helper 亦为 STRM 服务） |
| 27 | `869974b` | 09-08 | **修复网盘上传超时** | 16 | 5 | **候选 P1（最高优先）** |
| 28 | `167427c` | 09-09 | 垃圾清理废弃数据表 | 5 | 0 | 不适用（垃圾清理模块已删） |
| 29 | `99ea858` | 09-09 | 收口1-认证模块 | 6 | 6 | **候选 P3**（认证调度/收尾统一、日志漂移抑制） |
| 30 | `6a6219b` | 09-09 | 收口2-strm 模块 | 7 | 0 | 不适用 |
| 31 | `676ed7c` | 09-09 | 收口3-公告模块 | 1 | 0 | 不适用 |
| 32 | `277512e` | 09-09 | 跨盘提速修复 | 4 | 3 | 不适用主体（含 `upload/manager_test.go`） |
| 33 | `7115114` | 09-09 | 收口4-跨盘传输 | 3 | 0 | 不适用 |
| 34 | `5255775` | 09-09 | 收口5-上传任务 | 8 | 8 | **(d) 同文件竞争，纯重构，不建议移植**（见 §5.2） |
| 35 | `01c2583` | 09-10 | 跨盘传输多项优化 | 13 | 7 | 不适用主体（`internal/playback` 限流器可另议） |
| 36 | `0a0a670` | 09-10 | 后台页面响应优化 | 4 | 2 | 可选（新增聚合 `api/dashboard_overview.go`，需配套前端） |
| 37 | `66edc08` | 09-10 | 滚动时收藏夹始终显示 | 1 | 1 | 可选（纯前端小改） |
| 38 | `a52b9ad` | 09-11 | strm 可刮削更多信息 | 12 | 1 | 不适用 |
| 39 | `2889e47` | 09-11 | 联动 UI 改动 | 1 | 1 | 可选（联动=自动化 UI；本方保留 automation） |
| 40 | `354b9ff` | 09-11 | 收口6-前端样式 | 115 | 85 | **不建议移植**（全局样式重构，冲突面最大、收益低） |
| 41 | `393bdac` | 09-11 | 收口7-设置面板 | 7 | 4 | 可选（设置面板 UI） |
| 42 | `8bbcee7` | 09-11 | 仪表盘 UI 小改 | 5 | 4 | 可选（前端） |
| 43 | `adca0ee` | 09-11 | 铃铛消息推送优化 | 7 | 4 | **候选 P8**（通知改服务端 SSE 推送，含新测试） |
| 44 | `d7fb574` | 09-11 | 任务类面板统一信息 | 49 | 23 | 可选（含 `styles/upload-task-panel.css`，与本方上传面板相关但体量大） |
| 45 | `c009f63` | 09-12 | 优化重复刮削判断 | 6 | 0 | 不适用 |
| 46 | `15d7ac9` | 09-12 | 反代 WebSocket 修改 | 7 | 1 | 可选（`proxybase.go`，仅当用反代） |
| 47 | `46a0a89` | 09-12 | 播放诊断及直读文案 | 5 | 2 | **候选 P9**（`internal/playback/service.go` 诊断日志） |

### 3.3 分类统计

| 类别 | 数量 | 明细 |
|---|---|---|
| (a) 本方已移植 | **5** | `acb19c2`、`d6a2e62`、`a7d6124`、`afbcb50`、`294a708`（对应本部 `0.0.13`/`0.0.17`/`0.0.18`） |
| (b) 与本方精简策略无关 | **20** | 改动文件 0 个适用（11：`4f703cc`/`aa47ba2`/`eae166d`/`209103c`/`47ad97f`/`2e80827`/`167427c`/`6a6219b`/`676ed7c`/`7115114`/`c009f63`）+ 主体不适用（9：`4a0b868`/`f03744f`/`fc5ed4f`/`cb25a9b`/`4c387e0`/`f133946`/`277512e`/`01c2583`/`a52b9ad`） |
| (c) 候选移植 | **9** | P1~P9，见 §6 |
| (d) 与本方 0.0.32~0.0.37 重叠/冲突 | **1** | `5255775`（同文件但纯重构，无修复语义） |
| 其余（低价值前端/小重构，可选） | **12** | `cb332c9`、`882122f`、`1c5f2b0`、`05a81ed`、`0a0a670`、`66edc08`、`2889e47`、`354b9ff`、`393bdac`、`8bbcee7`、`d7fb574`、`15d7ac9` |
| 合计 | **47** | 5+20+9+1+12 = 47 ✔ |

---

## 4. 上游 v0.5.5 / v0.5.4 官方条目 × 提交对照（交叉印证）

官方 changelog（`litepan.top/changelog.html`，2026-09-12 抓取）与 git 提交互相印证：

| 官方条目 | 对应上游提交 | 对本方适用性 |
|---|---|---|
| v0.5.5 修复：**上传超时问题** | `869974b`（09-08）+ v0.5.5 期间 `httpx`/驱动上传客户端 | ✅ **候选①**（本方可直接受益） |
| v0.5.5 修复：反代 WebSocket 握手 | `15d7ac9`（`internal/proxybase`） | 可选（仅用反代时需要） |
| v0.5.5 优化：铃铛通知服务端实时推送 | `adca0ee` | ✅ 候选⑦ |
| v0.5.5 优化：仪表盘 UI / 任务面板统一信息 | `8bbcee7`、`d7fb574` | 可选（前端） |
| v0.5.5 优化：自动联动界面及整理动作异常 | `1c5f2b0`、`05a81ed`、`2889e47` | 低价值 |
| v0.5.5 修复：115strm 扫描误清理 | `2e80827` | 不适用 |
| v0.5.4 修复：**账号认证刷新异常导致的日志剧增（重大 bug，后台卡死）** | 该窗口内的认证链修复：`294a708`（driverexec 网络熔断 + `SetAuthGuards` + 驱动实例重置）、`a7d6124`（OAuth UA） | **已移植**（`0.0.13`+`0.0.17`）；本方另有独立的冷却日志抑制（`0.0.32`） |

> 说明：官方未逐条标注 commit，「日志剧增」一条为**基于提交影响的推断**（该窗口内仅认证链路改动具备此效果），标注为**未获上游书面确认**。

---

## 5. 与本方 0.0.32~0.0.37 修复的重叠与冲突（并排核对）

### 5.1 我方六项修复在上游的状态（证据）

| 我方修复 | 涉及路径 | 上游状态 | 证据（只读命令） |
|---|---|---|---|
| `0.0.32` 冷却日志放大抑制（1 INFO/窗口 + 摘要） | `internal/file/cooldown_log.go`、`service.go` | **上游无对应实现** | `git show origin/main:internal/file/service.go \| grep -n "冷却\|cooldown"` → 空；上游该文件仅把日志字段 `err`→`error`（`869974b`） |
| `0.0.33` 暂停投递前端修复 | `web/src/composables/upload/uploadPausePlan.ts`、`useUploadBatchActions.ts` | **上游无** | 上述文件上游自批次化 `acb19c2` 后**再无提交**（文件级 last-touch = `acb19c2`） |
| `0.0.34` 冷却等待原子化（暂停优先/值快照落库） | `internal/upload/state.go`、`worker.go`、`persist.go` | **上游无冷却等待概念** | `git grep -n "Cooldown\|cooldown\|冷却" origin/main -- internal/upload internal/file` → 无命中；上游 `exec.go` 仅返回普通错误文案 |
| `0.0.35` 熔断分级键（批次键/回退键） | `internal/upload/breaker.go` | **上游无该文件** | `git ls-tree -r --name-only origin/main -- internal/upload` → 无 `breaker.go` |
| `0.0.36` 自动化不写批次身份 | `internal/automation/service_run.go` | 上游同文件仅有参数签名重构（`05a81ed`） | `git show 05a81ed -- internal/automation/service_run.go` |
| `0.0.37` 终端桶展开（分组仅进行中） | `web/src/composables/upload/uploadTaskTree.ts`、`TaskPanel.vue` | **上游无 `groupBatches`** | `git show origin/main:web/src/composables/upload/uploadTaskTree.ts \| grep groupBatches` → 空 |

⇒ **本方的上传链路修复整体领先于上游**，不存在「上游已修同名问题、我方重复修」的情况；未来若从上游取用 `internal/upload/*`、`internal/file/*`、`web/src/composables/upload/*` 代码，必须保留本方的冷却等待、熔断分级键与终端桶展开扩展。

### 5.2 `5255775 收口5-上传任务`（唯一同文件竞争）逐行结论

改动 `internal/upload/{lifecycle,manager,persist,queue,state,types,worker}.go`（7 文件）+ 1 前端，**全部为等价重构**：

- 新增判定 helper：`isActiveUploadStatus`、`isResumableUploadStatus`、`isCompletedUploadStatus`、`isCrossTransferDownload`、`pausedMessage`、`taskPhase`、`taskCleanupMode`、`taskSourceType`、`requiredLocalFileMissing`；
- 替换内联判断为 helper（`pause`/`Resume`/`restoreTasks`/`taskSlotKindLocked`/`pendingMessage`/`executeUpload` 等）；
- 删除重复 helper（`snapshotCopy`、`progressForBytes`）。
- **未涉及**：冷却等待、熔断键、暂停/落库竞态、批次折叠、终端桶 —— 与 §5.1 我方修复无重叠语义。

⇒ 结论：**不建议移植**（收益≈0；与本方 0.0.32~0.0.37 大量局部改动冲突，需逐个文件手工调和）。若未来移植上游其它 `internal/upload` 改动，可顺带吸收其中我方尚未使用的 helper。

---

## 6. 候选移植清单（建议排序）

| 优先级 | 项 | 上游依据 | 受益 | 风险 | 工作量 | 最小验证方案 |
|---|---|---|---|---|---|---|
| **P1** | 上传超时修复：`httpx.NewStreamingClient` + 189/115 上传专用客户端 + 115 OSS 分片重试 | `869974b`（v0.5.5 官方条目「修复上传超时问题」） | 直击我方生产上传超时（189 `HTTP 511 S3ClientException` 类瞬时故障）；修正 115 OSS 分片误用 30s 总超时客户端 | 中：涉及驱动网络行为，需确认不改变 189 既有 500ms 节流（本次改动不触碰 gate） | 中（4~6 文件 + 单测） | 新增 `httpx.NewStreamingClient` 单测（transport 复用/ResponseHeaderTimeout/无总超时）；115 `isRetryableOSSUploadError` 表驱动单测；`go test ./drivers/... ./internal/httpx/` 全绿 |
| **P2** | 客户端断连不计 ERROR：`internal/api/error_log.go` 前置 `context.Canceled/DeadlineExceeded` 静默 | `d545e47` | 消除上传/下载中断造成的 ERROR 噪声（与本方日志卫生方向一致） | 低（1 文件 + 测试） | 小 | 单测：cancel 上下文 + 错误 → 不写日志；正常错误 → 仍写 |
| **P3** | 认证收口：`completeRefresh` 统一收尾 + 调度计算复用 + 调度日志漂移抑制 | `99ea858`（6 文件，~220 行差异） | 认证刷新路径单一收尾，减少重复失败记录与调度日志噪声；本方 115/189 认证是长期痛点 | 中（`control.go`/`refresh_runner.go`/`scheduler.go` 与 0.0.17 移植版同源，需保持 `RecoverAccount` 等本方适配） | 中 | `go test ./internal/auth/` 全绿；`TestScheduleLogIgnoresSmallRecalculationDrift`、`TestConcurrentActivePassiveAndInlineShareRefresh` 生效 |
| **P4** | 账号级目录缓存失效 `cache.InvalidateAccountDirKeys`（联动「刷新目录」改为账号级清理） | `df86352`（6 文件全部适用） | 联动/自动化改目录后按账号整片失效目录缓存，减少用户触发人工刷新与重复 List | 低-中（`internal/file`/`internal/cache` 接口语义需对齐本方缓存键约定） | 小-中 | `go test ./internal/cache/... ./internal/automation/`；核对 `prefixDir+accountID` 键空间 |
| **P5** | 慢后台可观测：`api/slow_dashboard_log.go` + `accountprofile` 优化 | `b5c9308` | 后台卡顿时可定位慢请求（与 v0.5.4「后台卡死」同类问题的运维手段） | 低 | 小 | `go test ./internal/accountprofile/`；中间件在慢请求时打点 |
| **P6** | 目录整理误匹配修复（TMDB 单汉字/唯一候选降级收紧） | `20b66de` | 本方保留 `mediaorganize/rules`（19 文件），可减少整理误匹配 | 低（规则模块自包含） | 小 | `go test ./internal/mediaorganize/...`；新增用例覆盖单汉字查询 |
| **P7** | 上传取消语义与批次名收敛：`api/upload.go` `isClientGone/isCancelError`、`upload/resume.go` 定时器简化、`local_upload` 批次名移除 | `ae75882`（适用 33 文件中挑取） | 取消场景不再记 ERROR / 不再误判；批次名逻辑与 0.0.36 决策趋同 | 低-中（需确认与本方批次显示规则一致） | 小-中 | `go test ./internal/upload/ ./internal/api/`；对照本方 0.0.36 决策记录 |
| **P8** | 铃铛通知服务端推送（SSE） | `adca0ee` | 去掉定时轮询，通知实时；含新测试 | 中（新增流式端点 + 前端改造） | 中 | 新端点 `notifications_stream_test.go` 同款用例；前端联调 |
| **P9** | 播放诊断：`internal/playback/service.go` 诊断日志/直读文案 | `46a0a89` | 播放问题可诊断（本方保留 playback） | 低 | 小 | `go test ./internal/playback/` |

> 可选项（本例仅列示，不建议本轮启动）：`0a0a670` 聚合仪表盘端点（需前端）、`354b9ff` 全局样式收口（85 文件，冲突面最大）、`d7fb574` 任务面板统一信息（49 文件）、`15d7ac9` 反代 WebSocket、`f03744f` 通用扫码链路（需先确认本方 189 扫码是否受益）。

---

## 7. 明确不实施 / 不适配

- **不适配（模块已删或本分支不使用）**：STRM（`6a6219b`/`eae166d`/`2e80827`/`a52b9ad`/`c009f63`）、跨盘传输（`7115114`/`277512e`/`01c2583` 主体）、Emby 与反代（`cb25a9b`/`f133946` 主体）、AI 辅助（`47ad97f`/`aa47ba2`）、垃圾清理（`167427c`）、公告（`676ed7c`）、非本方 5 个驱动（`869974b` 中的 123/139/Baidu/Guangya/OneDrive/Quark）。
- **不建议移植**：`5255775` 收口5（纯重构，与 0.0.32~0.0.37 冲突）；`354b9ff` 收口6（85 文件全局样式）。
- **本轮零实施**：未执行任何 merge / cherry-pick / 代码或构建改动；`git fetch` 是唯一仓库级操作（只更新远程跟踪 ref）。

---

## 8. 未能验证 / 限制

1. **历史重写原因**：未检索到上游公告或 Issue；仅能从 tree/SHA 事实判定「已重写、无共同祖先」。
2. **v0.5.4「日志剧增」对应提交**：官方 changelog 未标注 commit，本报告按改动内容推断为 `294a708` 等认证链修复（**推断，非上游确认**）。
3. **上游各版本社区反馈**：`tavily_search` 8 路检索仅得到项目介绍类内容（官网/文档站/视频/博客），**未找到**针对 09-05~09-12 这批提交的讨论、Issue 或回归报告。
4. **运行时行为**：未运行上游代码/镜像，候选移植的收益评估基于源码对照与官方条目，非实测。
5. **未连接生产机**：本轮不涉及 `10.0.0.11`，无生产侧验证。

---

## 9. 后续任务建议

1. **如果想要 P1（上传超时修复）**：建独立任务，范围＝`internal/httpx`（新增 `NewStreamingClient` + 单测）+ `drivers/189Cloud/driver.go` + `drivers/115_Open/{driver,upload}.go`（OSS 分片重试），bump `0.0.38`，走完整九步流程 + 发版部署。
2. **如果想要 P2+P5（日志/可观测）**：可合并为一个 `fix` 任务（都是「减少噪声/增加定位手段」），bump `0.0.39`。
3. **如果想要 P3（认证收口）**：单独任务，先在本地跑 `internal/auth` 全量测试与 `-race`，再评估是否发版。
4. **如果想要 P4/P6（缓存与整理规则）**：可各自独立小任务（`fix`），互不影响。
5. **历史重写的长期影响**：① 上游任何后续同步都改走「内容对照移植」；② 建议在 `.trellis/spec` 记录一条「上游同步口径」备忘（本次已在本报告 §2 定义方法，可按需固化为 spec）。
6. **不建议**把本分支整体对齐上游（历史已分叉且本方为精简版）。

---

## 10. 信息来源

- 主来源（一手）：本地 git —— `git fetch origin`、`git log/show/ls-tree`、三树 blob 对照；GitHub REST API（`repos/Ponphil/LitePan/commits|tags|releases`）。
- 交叉来源（二手）：官方文档站更新日志 <https://www.litepan.top/changelog.html>（2026-09-12 抓取，含 v0.5.5/v0.5.4 条目）；上游仓库页 <https://github.com/Ponphil/LitePan>（README 部署说明、驱动列表）；`tavily_search` 8 路（项目介绍/版本/changelog/驱动/上传超时/目录整理/精简分支）结果以第三方教程与视频为主，未见针对性变更讨论。
- 本方历史口径：`.trellis/workspace/zhemed/journal-2.md` Session 67/68/74/75/76（不整体 merge、按需 cherry-pick 的既有决策）。

---

### 附：本次执行的关键命令（可复现）

```bash
git fetch origin                                  # → + 374affd...46a0a89 main (forced update)
git merge-base --all main origin/main             # → 空（无共同祖先）
git log -1 --format='%h %ci %s' $(git rev-list --max-parents=0 main) $(git rev-list --max-parents=0 origin/main)
git rev-parse <root>^{tree}                       # 双方根 tree 相同 b5bba895…
git log --reverse --pretty='%h|%ad|%s' --date=format:'%m-%d %H:%M' f71a522..origin/main   # 47 提交
# 三树对照（base=4c160d9 / ours=main / theirs=origin/main）逐路径 blob 比对（脚本见会话记录）
git show 869974b -- drivers/115_Open/upload.go internal/httpx/client.go
git show 5255775 -- internal/upload/
git show 99ea858 -- internal/auth/
git ls-tree -r --name-only origin/main -- internal/upload   # 无 breaker.go
```
