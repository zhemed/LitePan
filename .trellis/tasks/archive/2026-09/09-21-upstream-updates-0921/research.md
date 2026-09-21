# 上游 LitePan 更新调查报告（2026-09-21）

> 任务：`09-21-upstream-updates-0921`｜口径：**只读**（副作用仅 `git fetch` 更新远程跟踪 ref `upstream/main`，未改工作区文件、未 merge、未 cherry-pick）
> 上游：`github.com/Ponphil/LitePan`｜本方：`zhemed/LitePan`（精简分支，仅 3 驱动）
> 上次核查：`2026-09-12`，基准 `46a0a89`（详见 `archive/2026-09/09-12-investigate-upstream-updates/research.md`）

---

## 0. 结论（TL;DR）

**上游 11 个新提交里，有 7 组改动是我们真正用得上的**（其中 4 项属缺陷/正确性修复），另有 **1 项是新能力需你拍板**；其余 4 个提交属已删功能的改进，**不适用**。

| 分级 | 项 | 内容 | 触达 |
|---|---|---|---|
| **P0 缺陷/正确性** | A1 | 115 全量清单**完整性加固**：空页重试 + 数量不足直接报错（不再静默返回残缺清单） | `drivers/115_Open/full_list.go` |
| **P0** | A2 | 115 `ResolveDirPath` 改为**相对账号挂载根**，不在根下时报错（不再拼出伪路径） | 同上 |
| **P0** | A3 | 115 目录名里的 `/`、`\` 替换为 `_`（**防伪造层级**） | 同上 |
| **P0** | A4 | 115 `pickBy` 明文缓存**加上限 10 万 + 删除成功后清理**（原为无上限增长） | `drivers/115_Open/ops.go` |
| **P1 性能/可观测** | B | `adminauth` **配置内存缓存**（RWMutex 一次性加载），消除每请求多次 SQLite 读 | `internal/adminauth/service.go` |
| **P1** | C | 慢接口日志**降为 Debug + 30 分钟抑制**（带 suppressed_count） | `internal/api/slow_dashboard_log.go`、`router.go` |
| **P1** | D | 服务器上传的符号链接解析**从"每文件"提到"每批次"** + 去掉一层间接调用 | `internal/api/local_upload.go` |
| **P1** | E | OAuth 转发全部重试失败时**记录真实原因**（原为 `_ = lastErr` 静默） | `internal/api/oauth.go` |
| **P2 清洁** | F | Dockerfile 构建时加 `npm run type-check`（类型不过就别出镜像） | `Dockerfile` |
| **P2** | G | 三处小清理：`accounts.go`/`files.go` 去掉零值分支、`commit_writer` 幂等下放、`main.go` 日志中文化、**删掉只写不读的 `admin_temp_password_last_reset_at`**（含僵尸配置键） | 4 文件 |
| **需你拍板** | H | 自动联动新增「**高级定时**」：按周几 / 每月几号 + 时分（我们现在只有 daily / interval 两种） | `internal/automation/*` + `AutomationPanel.vue` |

**不适用 4 个**：FUSE 读缓存优化（FUSE 已删）、飞牛影视扫库（fnosproxy 已删）、jellyfin 反代（embyproxy 已删）、目录整理/手动匹配/STRM 相关（均已删）。

**⚠️ 两条"别跟着上游做"**：① 上游这批里**删除了 `drivers/template/**` 整个模板驱动**（−435 行）——我们**故意保留**它（P3/P4 处置：新增驱动的骨架 + OAuth 守卫测试的唯一载体），**不要跟着删**；② 上游 `web/src/composables/useNotificationBadge.ts`、`internal/api/dashboard_overview.go` 等文件**我们没有**，不要为了"对齐上游"去新建。

---

## 1. 上游现状与基准

| 项 | 值 |
|---|---|
| 上游 `main` HEAD | `42a3ee9a84`（2026-09-20T03:39Z，`优化通知轮询`）|
| 上游最新 tag | **`v0.5.6-beta`** → `42a3ee9a84`（上次核查时为 `v0.5.5-beta` = `46a0a89`）|
| 上游 GitHub Release | **无**（`releases` 接口返回空数组 —— 上游只用 tag，不发 release）|
| 上次基准 `46a0a89` | **仍在上游历史中**（未被重写，`ahead_by=11 behind_by=0`）|
| 新增提交 | **11 个**（2026-09-13 → 09-20）|
| 改动文件 | 300 个，其中 **251 个是上游构建产物**（`internal/api/web/**`），代码/文档 **49 个** |
| 其中本方 tree 存在 | **32 个**（才可能适用）；不存在 17 个（多为已删功能）|

> 上游这次**没有**重写历史（上次核查时曾被迫 update）。这意味着：如果将来要跟上游做更深的合并，基准是 `46a0a89` 之后可以直接用 `46a0a89..42a3ee9a` 的区间。

---

## 2. 新增 11 提交逐条盘点

| # | sha | 日期 | 标题 | 改动在本方 tree | 适用性 |
|---|---|---|---|---|---|
| 1 | `924e4c75` | 09-13 | 大量收口与修复 | 26/42 | **部分适用**（见 A4/B*/D/E/F/G）|
| 2 | `511d7fe8` | 09-14 | 115STRM增强逻辑加固 | 1/3 | **适用**（A1/A2，`internal/strm` 部分不适用）|
| 3 | `60a17f69` | 09-14 | FUSE读缓存优化 | 0/4 | 不适用（FUSE 已删）|
| 4 | `a9f9603b29` | 09-14 | 自动联动添加高级定时 | 6/7 | **需拍板**（H）|
| 5 | `aaec3a29` | 09-14 | 支持飞牛影视扫库和刷新元数据 | 10/16 | 不适用（fnosproxy 已删）|
| 6 | `5e27f236` | 09-14 | 优化低性能设备接口响应慢问题 | **5/5** | **适用**（B + C）|
| 7 | `974ec4cf` | 09-14 | 修复手动匹配后残留空目录 | 0/2 | 不适用（媒体整理已删）|
| 8 | `a63a2c04` | 09-14 | 兼容jellyfin反代及联动 | 7/14 | 不适用（embyproxy 已删）|
| 9 | `d1ded1fd` | 09-14 | 目录整理合集规则优化 | 0/6 | 不适用（目录整理已删）|
| 10 | `6d392018` | 09-17 | 修复strm斜杠和误删 | 6/19 | **部分适用**（A3；`internal/strm` 与 strm 前端不适用）|
| 11 | `42a3ee9a` | 09-20 | 优化通知轮询 | 4/5 | 不适用（改的是我们没有的文件，见 §5）|

---

## 3. 推荐移植清单（每项含：上游证据 / 本方现状 / 为什么需要 / 成本风险 / 分级）

### A 组：115 驱动（P0 —— 都是"数据正确性"类问题）

**A1 全量清单完整性加固**（`511d7fe8`｜`drivers/115_Open/full_list.go`）
- 上游做法：① 空页先 `time.Sleep(250ms)` 重试一次；② 全程记录 `expectedCount = max(page.Count)`；③ 重试后仍为空页且 `expectedCount > len(seenIDs)` → 返回 `CodeDriverError("115 全量清单不完整：应有 N 个文件，实际只获取 M 个，已停止扫描")`；④ 注释明确"分页只以连续空页为结束信号，Count 仅在结束时拦截不完整清单"。
- 本方现状：`drivers/115_Open/full_list.go` 的 `collectFullListPages` 与上游改前同款（空页即 break，不看 Count）。**且我方确实在用这条路径**：`internal/file/service.go:114-146`「清单模式：驱动支持 `FullListLister` 时一次拉取」。
- 为什么需要：115 侧 `Count` 短时偏小是已知现象；静默返回残缺清单会让我们**少列文件**（上游更惨：STRM 会把没扫到的目录当已删除清掉）。
- 成本/风险：单文件 ~40 行；需同步上游新增的 `full_list_test.go` 用例（+73 行）；无迁移、无跨层。
- **分级：P0**

**A2 `ResolveDirPath` 相对挂载根 + 越界报错**（`511d7fe8`）
- 上游做法：`buildDirPath` 由 `string` 改为 `(string, bool)`；遇到 `rootID` 段**截断**（`segs = segs[:0]`）使结果相对账号挂载根；目录不在根下 → `CodeDriverError`；`get_info` 空入口 → `CodeNotFound`。
- 本方现状：`buildDirPath(paths, selfName) string`（无 rootID 概念、恒返回路径）；调用方 `internal/file/service.go:148-159`（"清单模式 pid→路径 补漏"）。
- 为什么需要：不相对挂载根时，多根账号下拼出的路径会把"挂载根之上"也带进来；越界时我们得到的是**看似合法却错误**的路径。
- 成本/风险：与 A1 同文件；需同步改我们的调用点与测试。
- **分级：P0**

**A3 目录名分隔符消毒**（`6d392018`｜`pathSegmentName`）
- 上游做法：新增 `pathSegmentName()`，把段内 `/`、`\` 替换为 `_`；用于 `buildDirPath` 的两处拼接。
- 本方现状：直接 `strings.TrimSpace(name)` 后拼 `/`。
- 为什么需要：网盘允许目录名带 `/`；拼进以 `/` 分隔的路径后，上层再也分不清"一个带斜杠的名字"和"多层目录" —— 上游因此把 STRM 建到错误目录并误删正确文件；我们虽无 STRM，但 pid→路径补漏同样会被伪造层级骗到。
- 成本/风险：极小（新增一个 8 行函数 + 2 处调用）。
- **分级：P0**

**A4 `pickBy` 缓存上限与失效清理**（`924e4c75`｜`drivers/115_Open/ops.go`）
- 上游做法：`const maxPickCodeCacheEntries = 100_000`；`rememberPickCode` 在**键不存在且已达上限**时 `clear(d.pickBy)`；`DeleteFiles` 成功后把已删 ID 从缓存里删掉。
- 本方现状（实测）：`d.pickBy` 是**无上限**的 `map[string]string`（`ops.go:114-119`），且删除文件后**不清理**。
- 为什么需要：① 内存无上限增长（大库长期运行）；② 删除后残留明文 pickcode。
- 成本/风险：单文件 ~15 行。
- **分级：P0**

### B 组：adminauth 配置内存缓存（P1 —— 直击"低配设备接口慢"）

- 上游证据（`5e27f236`）：`Service` 增 `configMu sync.RWMutex` + `configLoaded bool` + `configValues map[string]string`；新增 `configValue()`（首次 `configs.All(ctx)` 全量加载后走内存）与 `setConfig()`（写库 + 回写内存）；所有 `s.configs.Set/Get` 调用点改走这两个方法。
- 本方现状：`internal/adminauth/service.go` 的 `configString/configInt/configBool` **每次都 `s.configs.Get`**（管理员鉴权链路上每请求数次 SQLite 读）。
- 为什么需要：管理员页面每个请求都要过 `ReadSession` + `EnsureAdminAccess`，其中多次读配置；机械盘/NAS 上这是可感知的延迟（上游就是为"低性能设备接口响应慢"改的）。
- 成本/风险：单文件 ~50 行 + 上游 66 行测试；**需注意多实例/外部改库时缓存会陈旧**（上游接受该折衷，因为配置只经本方 UI 写）。
- **分级：P1**

### C 组：慢接口日志降级与抑制（P1）

- 上游证据（`5e27f236`）：`slow_dashboard_log.go` 增 `slowRequestLogInterval = 30 * time.Minute` 与 `slowRequestLogs{entries map[string]slowRequestLogEntry}`（`shouldLog` 抑制 + `recovered` 重置），日志**由 Info 降为 Debug**，附 `suppressed_count`；`Handler` 增 `slowLogs` 字段。
- 本方现状：`internal/api/slow_dashboard_log.go` 只有"超过 1 秒记一条 Info"——**无抑制、无恢复、级别是 Info**（即后台概况接口一慢就刷常规日志）。该文件正是我们在 `0.0.43` 从上游 `b5c9308` 移植的 P5①，本提交是它的后续改进。
- 为什么需要：避免把"诊断信息"当用户可见日志刷屏（我方日志保留 30 天，噪声会挤占真实错误信息）。
- 成本/风险：单文件 + 上游 51 行测试；`router.go` 加 1 行字段。
- **分级：P1**

### D 组：服务器上传的符号链接解析提级（P1）

- 上游证据（`924e4c75`｜`internal/api/local_upload.go`）：在批次开始时 `resolvedRoot, err := filepath.EvalSymlinks(m.Path)`，循环内改用 `resolveLocalUploadSourceUnderRoot(s.abs, resolvedRoot)`；删除 `statLocalFile` 包装改直呼 `os.Stat`；映射目录不可解析时**整批快速失败**并给出明确错误。
- 本方现状（实测）：`resolveLocalUploadSource(abs, root)` **在每个文件里** `EvalSymlinks(root)`（`local_upload.go:511-516`），并有 `statLocalFile` 间接层（`:506`）。
- 为什么需要：N 个文件 → N 次 `EvalSymlinks`；我们历史上有 2000 文件批量上传的场景，属可测量开销。
- 成本/风险：单文件改写 3 处；**同时要保留我们已删/新增的本地化差异**（我们的 `isWithinRoot` 校验与上游一致，需按本方代码 craft patch，不能直接 apply）。
- **分级：P1**

### E 组：OAuth 转发失败可观测（P1）

- 上游证据（`924e4c75`｜`internal/api/oauth.go`）：`_ = lastErr` → `if lastErr != nil { requestLogger(...).Warn("OAuth 转发重试均失败", "url", url, "attempts", maxRetries+1, "err", lastErr) }`；顺带 import 排序。
- 本方现状：同款 `_ = lastErr`（我们已删过死代码但保留了这行占位）。
- 为什么需要：现在失败只返回一句通用文案，排查时**看不到真实原因**。
- 成本/风险：3 行。
- **分级：P1**

### F/G 组：清洁项（P2）

- **F**：`Dockerfile` 的 `RUN npm run build` → `RUN npm run type-check && npm run build`（上游）。我方 Dockerfile 目前只 build；加上后**类型不过就构建失败**，与我们"质量门全绿才发版"的纪律一致。1 行。
- **G1**：`internal/api/accounts.go`、`files.go` 去掉 `if !x.IsZero()` 分支 → 直接赋值。**已核实行为等价**：我方 `FormatAPITime` 对零值返回 `""`（`internal/api/format.go:7-12`）。纯简化。
- **G2**：`internal/api/commit_writer.go`：`Write` 里的自增幂等判断下放给 `WriteHeader`（上游注释："幂等由 WriteHeader 内部保证"）。行为等价。
- **G3**：`cmd/litepan/main.go` 三条启动/关闭/运行错误日志改中文（`启动失败`/`关闭出错`/`运行出错`），与项目其余日志一致。
- **G4（值得一提）**：`internal/adminauth/service.go` 删除 `KeyAdminTempPasswordLastReset`。**本方实测：`tempPasswordState.LastReset` 只被赋值、全仓无读者**（`grep` 仅命中定义与赋值两行），且该键还出现在 `internal/store/backup.go:97` 的清洗名单与 `internal/backuprestore/service.go:34,451` 的清空名单里 —— 是一条**只写不读的僵尸配置键**（与本项目此前清理过的 `strm_base_url` 等同类）。删除需连带上迁移清键（可并入下次迁移）。
- **分级：F = P2（1 行、收益明确）；G = P2（清洁，其中 G4 建议随下次迁移一起做）**

---

## 4. 需你拍板：自动联动「高级定时」（H）

- 上游证据（`a9f9603b29`）：新增 `domain.AutomationTriggerAdvanced`；`internal/automation/service_schedule.go` 增 `nextAdvancedRun()`（按 `schedule_mode = weekly|monthly`，`weekdays`/`month_days` + `time` 计算下次运行，最多向前找 366 天）；`service_validate.go` 增校验；前端 `AutomationPanel.vue` 大改（+189/-…，加周几/几号选择）。
- 本方现状：触发器只有 `AutomationTriggerDaily`（`:17`）与 `AutomationTriggerInterval`（`:18`）；`internal/automation/service_schedule.go` 已存在（可自然扩展）。
- 性质：**这是新能力，不是缺陷修复**（我们现在用"每 N 小时"绕开，无法表达"每周一 03:00"这类需求）。
- 成本/风险：后端约 70 行 + 前端约 190 行 + 测试；需与我们的 `service_validate.go` 差异对齐（我们没有上游的 offline_download 触发器）。
- **建议**：如果你有"只在特定星期/日期跑"的需求就做（P1）；否则不必跟进（上游喜欢堆触发器，我们已明确只保留 3 驱动与必要功能）。
- **状态：等你一句话**。

---

## 5. 明确不适用 / 需要小心的项

| 项 | 为什么不做 |
|---|---|
| `60a17f69` FUSE 读缓存优化 | FUSE 已在 `09-15-remove-fuse` 整体删除（0/4 文件在本方 tree）|
| `aaec3a29` 飞牛影视扫库、`a63a2c04` jellyfin 反代 | 依赖 `internal/fnosproxy/**`、`internal/embyproxy/**`（已删）；其新增的 automation 动作（`FnosScan` 等）也就无从挂载 |
| `974ec4cf` 手动匹配残留空目录、`d1ded1fd` 目录整理合集 | 媒体整理/目录整理已删（0 文件命中）|
| `6d392018` 的 `internal/strm/**` 与 strm 前端部分（约 13 个文件、含 500+ 行测试）| STRM 已删；**只取其中的 115 `pathSegmentName`（A3）** |
| `42a3ee9a` 通知轮询 | 改的是 `web/src/composables/useNotificationBadge.ts`，**本方没有该文件**（我们是 `AdminNotificationBell.vue`，30 秒 `setInterval` 轮询、无 SSE）。文件级不可移植；**其思路**（长连接建立后停掉兜底轮询、坏帧立即刷新）可作为将来优化我方轮询的参考，但不在本轮建议内 |
| 上游删除 `drivers/template/**`（−435 行）与 `drivers/all.go` 的驱动清单调整 | **不要跟随**：模板驱动是我们**故意保留**的（`09-12-close-deadcode-p3-p4` 已定案：新增驱动骨架 + `TestOAuthDriversUseUnifiedGuard` 的唯一载体）|
| 上游新增测试文件（`api/router_test.go` +176、`api/local_upload_test.go` +85/−2 等）| 多数对应其新端点/新功能；移植 A–G 时**按需取用相关用例**，不整文件搬运 |
| 251 个 `internal/api/web/assets/*.gz` 变更 | 上游构建产物，一律忽略（我们有自己的 `vite build`）|

**已收敛项（本轮无需动作，记录备查）**：`internal/api/sse.go` 删 `streamSSEMessages` 与 `drivers/189Cloud/transport.go` 删 `signedForm` —— 这两处在**我方 `0.0.41` 的死代码清理里已经删掉了**，说明两边判断一致。`189Cloud/ops.go` 把 `regexp.MustCompile` 提到包级（避免每次下载解析重编译）是**我们可顺手采纳的 3 行优化**（可并入 A 组批次，非独立项）。

---

## 6. 官方 release / changelog 交叉印证与发布面影响

- 上游**没有 GitHub Release**（`releases` 返回空），只有 tag `v0.5.6-beta` → `42a3ee9a`；该 tag 指向的提交**主体是版本号 bump + 重建产物**，真正的功能改动是 `web/src/composables/useNotificationBadge.ts`（我方不适用）。
- 也就是说：**tag 覆盖面 ≠ 功能覆盖面** —— 本轮真正的价值集中在 09-13～09-17 那批修复里，而不是最新 tag 上。这再次印证"必须看 diff，不能只看版本号/标题"。
- 对本方发布计划的影响：**无需因上游更新而调整**。我方 `v0.0.47` 已发布部署；若决定移植 A–G，按项目惯例走一个独立任务 + 质量门，再决定是否发 `v0.0.48`。

---

## 7. 建议的移植批次（供后续任务引用）

| 批次 | 内容 | 预估改动 | 风险 |
|---|---|---|---|
| **批次 1（P0，建议尽快）** | A1 + A2 + A3 + A4（115 驱动四项）+ 189Cloud 正则提级 | `drivers/115_Open/{full_list.go,ops.go}` + 同步上游 2 个测试文件，`drivers/189Cloud/ops.go`，约 150–200 行 | 低（单层、有上游测试可参考；需按本方差异 craft patch）|
| 批次 2（P1，性能/可观测） | B + C + D + E | `adminauth/service.go`、`api/slow_dashboard_log.go`、`api/router.go`、`api/local_upload.go`、`api/oauth.go`，约 150 行 + 测试 | 中（B 引入缓存需明确失效语义；D 需按本方代码 craft）|
| 批次 3（P2，清洁） | F + G1 + G2 + G3 +（G4 随下次迁移） | 5 文件、约 30 行 | 低 |
| 待定 | H 高级定时（新能力） | 后端 ~70 + 前端 ~190 行 | 中 |

**每批必须**：按本方架构适配而非直接 apply（上游删了很多我们留的东西，直接 apply 会连带删除）；改动后跑项目质量门（`make lint` / `go vet` / `go test` / `vue-tsc` / `build`）；新增/移植回归测试；不引入上游的已删功能依赖。

---

## 8. 可复核命令（本报告每条结论的复跑入口）

```bash
cd /root/LitePan
# 上游基准与新增提交（网络：仅上游仓库）
gh api repos/Ponphil/LitePan/commits/main --jq '{sha:.sha,date:.commit.committer.date,title:(.commit.message|split("\n")[0])}'
gh api repos/Ponphil/LitePan/tags --jq '.[0:3][] | "\(.name) \(.commit.sha)"'
gh api "repos/Ponphil/LitePan/compare/46a0a89...42a3ee9a84" --jq '{ahead:.ahead_by,commits:[.commits[].sha[0:10]]}'
git fetch --no-tags https://github.com/Ponphil/LitePan.git main:refs/remotes/upstream/main
git log --oneline 46a0a89..upstream/main

# 适用性过滤（改动文件是否在本方 tree）
for sha in 924e4c75 511d7fe8 60a17f69 a9f9603b29 aaec3a29 5e27f236 974ec4cf a63a2c04 d1ded1fd 6d392018 42a3ee9a; do
  echo "== $sha"; git show --name-only --format="" $sha | grep -v '^$' | grep -v 'internal/api/web/' | while read f; do git ls-files --error-unmatch "$f" >/dev/null 2>&1 && echo "   ✓ $f"; done
done

# 本方现状核对（A/B/C/D/G4）
grep -n "pickBy\|pickMu" drivers/115_Open/ops.go
grep -n "configs.Get\|configs.All" internal/adminauth/service.go | head
grep -n "shouldLog\|suppressed" internal/api/slow_dashboard_log.go
grep -n "EvalSymlinks" internal/api/local_upload.go
grep -rn "LastReset" --include="*.go" internal/ | grep -v KeyAdminTempPasswordLastReset
grep -n "func FormatAPITime" -A 6 internal/api/format.go

# 不适用与"别跟着删"的证据
git show --stat 924e4c75 -- drivers/template | tail -3
ls web/src/composables/useNotificationBadge.ts 2>/dev/null || echo "本方无此文件"
```
