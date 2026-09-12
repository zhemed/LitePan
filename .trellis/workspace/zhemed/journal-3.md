# Journal - zhemed (Part 3)

> Continuation from `journal-2.md` (archived at ~2000 lines)
> Started: 2026-09-11

---



## Session 121: 修复终态桶为空+清理任务待命,0.0.37
<!-- trellis-session: v=2 fp=e53f34bb1eb03527 -->

**Date**: 2026-09-11
**Task**: 修复终态桶为空+清理任务待命,0.0.37
**Package**: backend
**Branch**: `main`

### Summary

用户决策：方案A 开任务修「批次分组下终态桶为空」（0.0.37）+ 授权在批次结束后清理生产存量批次字段（已建任务待命）。0.0.37：buildUploadTaskLevel 新增 options.groupBatches（默认 true 兼容）；TaskPanel 仅「进行中」桶折叠，终态桶展开为逐文件行；断言 +6 条（MEMO-ALL-PASS）；spec §8.4 记录折叠规则。质量门 type-check/build/vet/test 全绿；镜像 e6c3a898 推送+tag+release+本地部署三连。待命任务 09-11-prod-cleanup-automation-batch-fields（planning，等生产批次结束；PRD 含备份/最小 UPDATE/前后对比/风险评估）。过程偏差如实记录：本任务代码改动早于 task.py start（PRD 仍先于 start 完成）

### Main Changes

- web/src/composables/upload/uploadTaskTree.ts, web/src/components/upload/TaskPanel.vue, web/scripts/check-upload-memo.mjs, spec §8.4, README/docker-compose v0.0.37

### Git Commits

| Hash | Message |
|------|---------|
| `65e91f0` | fix(web): expand terminal buckets so grouped batches stay browsable, bump to 0.0.37 |

### Testing

- [OK] npm run check:memo MEMO-ALL-PASS；type-check/build 通过；go vet/test 全绿；本地 0.0.37 health/登录/任务列表通过

### Status

[OK] **Completed**

### Next Steps

- 等用户告知生产批次传完→执行存量 batch 字段清理（需先确认无 pending/running、备份 DB、最小 UPDATE、前后对比）；生产升级 0.0.36/0.0.37 由用户操作


## Session 122: 生产存量批次字段清理：无需写入（目标已达成）
<!-- trellis-session: v=2 fp=8be8373921875398 -->

**Date**: 2026-09-11
**Task**: 生产存量批次字段清理：无需写入（目标已达成）
**Package**: backend
**Branch**: `main`

### Summary

用户升级后验证成功、传输完毕，启动待命任务执行清理。只读确认：生产机 v0.0.37(ImageID 36b11f22)，819 条任务全部 success、零 failed/paused/pending/running；关键发现 auto-% 行为 0、batch_id/batch_name 非空 = 0 ⇒ 无清理对象（用户升级到 ≥0.0.36 后的新运行不再写批次身份，0.0.35 存量批次已不在库）。按最小写入原则未执行备份与 UPDATE，全程零写入（仅 docker inspect + mode=ro SELECT）。任务以『目标已达成、无需写入』闭环，验收措辞校准与证据记入 research.md/PRD

### Main Changes

- .trellis/tasks/archive/2026-09/09-11-prod-cleanup-automation-batch-fields/{prd.md,research.md,task.json}

### Git Commits

| Hash | Message |
|------|---------|
| `5ffa648` | chore(task): archive 09-11-prod-cleanup-automation-batch-fields |

### Testing

- [OK] 只读复核：statuses=819 success；batch_id 非空 0；pending/running 0；未触碰容器/配置/队列

### Status

[OK] **Completed**

### Next Steps

- 无待办；生产机已 0.0.37 且面板无折叠，后续运行也不会再产生批次字段


## Session 123: 本地3条失败任务归档确认（只读复核）
<!-- trellis-session: v=2 fp=7e2f97b537e3d688 -->

**Date**: 2026-09-12
**Task**: 本地3条失败任务归档确认（只读复核）
**Package**: backend
**Branch**: `main`

### Summary

用户确认归档本地 3 条失败任务。只读复核：本地实例 v0.0.37(ImageID 36b11f22, Restarts=0)，任务 13 条全部 success、failed=0；对照 09-11 清单：bulk5k_1609.bin 与 bulk5k_1540.bin 已重传成功，bulk5k_1267.bin 记录已随列表清理。观察并如实标注：本地任务记录 15,098→13 为用户侧清理，非我方操作；云端文件与本地测试文件未受影响。本轮零写入

### Main Changes

- .trellis/tasks/archive/2026-09/09-12-record-local-failed-tasks-archived/{prd.md,research.md,task.json}

### Git Commits

| Hash | Message |
|------|---------|
| `214d17f` | chore(task): record local failed-tasks archived (read-only verified) |

### Testing

- [OK] 只读：SQLite mode=ro 查询 + docker inspect；无任何写操作

### Status

[OK] **Completed**

### Next Steps

- 无待办；若 189 HTTP 511/513 频繁出现再另开专项任务


## Session 124: 调查上游 LitePan 最新更新（v0.5.5-beta）：历史被重写，47 提交分类与 9 项移植候选
<!-- trellis-session: v=2 fp=66eb433874514a47 -->

**Date**: 2026-09-12
**Task**: 调查上游 LitePan 最新更新（v0.5.5-beta）：历史被重写，47 提交分类与 9 项移植候选
**Package**: backend
**Branch**: `main`

### Summary

只读调查 ponphil 上游自 fork 点(f71a522≡4c160d9, 08-29)以来 47 个提交：① 发现上游已 force-push 重写历史（根 tree 相同、SHA 全变、双方无共同祖先）→ 同步口径改为内容级 tree 对照，禁止 merge/rebase；② 逐提交适用性评分（11 个 0 适用文件；STRM/跨盘/Emby/AI/清理/多驱动均不适用）；③ 与本方 0.0.32~0.0.37 六项修复并排取证：上游无 breaker.go/无冷却等待/无 cooldown 日志抑制/无 groupBatches，唯一同文件竞争 5255775 收口5 为纯重构，不建议移植；④ 产出 P1~P9 候选（P1=869974b 上传超时修复最优先）；⑤ 官方 changelog v0.5.5/v0.5.4 与 git 提交交叉印证。本轮零代码改动（go vet 绿）

### Git Commits

(No commits - planning session)

### Testing

- [OK] 只读：git fetch/tree 三向对照/GitHub API/8 路 tavily+官方 changelog 交叉；零代码改动（git status 仅任务目录，git diff HEAD 为空）；go vet ./... 全绿

### Status

[OK] **Completed**

### Next Steps

- 待用户决定是否建移植任务：P1 上传超时（httpx.NewStreamingClient+189/115 上传客户端+OSS 分片重试，bump 0.0.38）
- 可选：P2+P5 日志/可观测合并任务；P3 认证收口（需 -race 验证）
- 建议将「上游同步口径：内容对照移植」固化为 .trellis/spec 备忘（待用户确认）


## Session 125: P1 上传超时修复并发布 0.0.38：控制面/数据面客户端解耦
<!-- trellis-session: v=2 fp=104d6fc0e4bb00a1 -->

**Date**: 2026-09-12
**Task**: P1 上传超时修复并发布 0.0.38：控制面/数据面客户端解耦
**Package**: backend
**Branch**: `main`

### Summary

移植上游 869974b：httpx.NewStreamingClient（克隆 base transport 但不设 http.Client.Timeout，只设 ResponseHeaderTimeout=60s）；115 新增数据面 uploadClient（Init 建/Drop 关），单请求与分片 PUT 改用它，新增 ossUploadPartWithRetry（3 次、1s/2s 退避、可取消、重试前 Seek 复位）与 isRetryableOSSUploadError（net error/EOF/reset/broken pipe/429/5xx；凭证错误交给既有刷新链路避免 3× 放大）；189 uploadClient 由 300s+关 keep-alive 改为 NewStreamingClient(d.client,60s)。与上游同名函数逐字节一致（仅去掉多余 nil 防御）。定制未动：115 600s、512MB 分片、189 节流与分片重试。质量门 vet/test/build/web 三连全绿；golangci-lint 在本机 go1.27.0 下 staticcheck 于依赖包 poll panic（isInitialPkg=false），改用 go vet + 手工 depguard 等价核对并如实记录。发布 0.0.38：三 tag 同 digest 7fe5f5ea，tag v0.0.38 + release，本地容器重建三连（health/表单登录/任务汇总 total=13 success=13）。记录更正：09-12 调查报告 P1 的『30s』表述有误，我方 115 为 600s（4d8e868/0.0.3 起），问题实质＝API 与数据面共用客户端 + 缺分片重试

### Main Changes

- internal/httpx 新增 NewStreamingClient（数据面无总超时 + 60s 响应头兜底）
- drivers/115_Open 数据面专用客户端 + 分片 3 次瞬时故障重试（凭证错误不重试）
- drivers/189Cloud 上传客户端流式化（取消 300s 总超时、恢复连接复用）
- spec/backend/driver-development.md 增补『控制面 vs 数据面 HTTP 客户端』约定
- 版本 0.0.38（README + docker-compose）

### Git Commits

| Hash | Message |
|------|---------|
| `6ae4c11` | fix(driver): stream upload bodies and retry 115 parts, bump to 0.0.38 |

### Testing

- [OK] go vet ./... 全绿；go test ./... 全包 ok；go build ./... OK；web type-check+build+check:memo MEMO-ALL-PASS（构建产物零 churn）；新增 internal/httpx/client_test.go(5) 与 drivers/115_Open/upload_retry_test.go(9)；部署三连通过

### Status

[OK] **Completed**

### Next Steps

- ⚠️ 生产机 10.0.0.11 仍为 0.0.37（未授权未动）；如需享受本次上传超时修复，请授权后再升级
- 可选后续：P2+P5 日志/可观测任务、P3 认证收口任务
- 环境问题待办：golangci-lint 在本机 go1.27.0 崩溃（go.mod 声明 1.26.6）——需要升级 golangci-lint 或固定本地工具链


## Session 126: 驱动超时统一 30s 并发布 0.0.39（含两处流程偏差复盘）
<!-- trellis-session: v=2 fp=aae38cf2df5ea8a3 -->

**Date**: 2026-09-12
**Task**: 驱动超时统一 30s 并发布 0.0.39（含两处流程偏差复盘）
**Package**: backend
**Branch**: `main`

### Summary

按用户指示统一驱动侧超时为 30s：115 API 客户端 600s→30s（覆盖 rawRequest/postPassport/ossDo＝全部 API 与 OSS init/complete/凭证刷新）、115 上传响应头兜底 60s→30s、189 上传兜底 60s→30s（189 API 本就 30s）；上传传输仍无总超时，30s 只作用于『分片发完后上游不回应』与普通 API。测试：新增 TestInitConfiguresDriverTimeouts（离线 Init 锁定 API=30s/上传无总超时/上传兜底=30s），upload_retry_test.go 断言同步；go vet/test/build + web 三连全绿。发版 0.0.39：三 tag 同 digest 1f2b2bb4、本地容器重建三连通过。已知取舍：115 complete 类慢 API 30s 即失败（回退=只调大该值），慢速大分片传输不受影响。两处流程偏差如实记录：① gh release create 先于 git push main 执行，GitHub 自动打出的 v0.0.39 指向旧 main 头 86e52ca，发现后删远端 tag 重推至 7243a27（API 复核通过）——教训：必须 push main→tag→push tag→最后 release；② pre-archive 因仍有 1 项未勾选被拒（我漏勾了第一条验收项），但命令链仍执行了 archive，已按先例把任务目录移回、补勾证据、重跑 pre-archive 通过后重新归档，并在 PRD Notes 留痕

### Main Changes

- 115 API 客户端总超时 600s→30s
- 115 上传响应头兜底 60s→30s；189 上传兜底 60s→30s
- 版本 0.0.39 + 镜像三 tag + tag/release + 本地部署

### Git Commits

| Hash | Message |
|------|---------|
| `7243a27` | fix(driver): unify driver timeouts to 30s, bump to 0.0.39 |

### Testing

- [OK] go vet ./... 全绿；go test ./... 全包 ok；go build ./... OK；web type-check+build+check:memo MEMO-ALL-PASS；新增/更新 115 侧超时断言用例；本地容器重建三连（health/表单登录/任务汇总 total=13 success=13）

### Status

[OK] **Completed**

### Next Steps

- 观察：若 115 大文件在合并阶段报 Client.Timeout，只需把 115 API 客户端调大（最小回退）
- 可选后续：P2+P5 日志/可观测、P3 认证收口（上游 99ea858）
- 环境待办：本地 go1.27.0 与 go.mod 1.26.6 不一致导致 golangci-lint 崩溃，需固定工具链或升级 linter


## Session 127: 调查 P6 必要性：目录整理模块已是死代码，P6 作废
<!-- trellis-session: v=2 fp=a045a9e134251d98 -->

**Date**: 2026-09-12
**Task**: 调查 P6 必要性：目录整理模块已是死代码，P6 作废
**Package**: backend
**Branch**: `main`

### Summary

只读复核 P6（上游 20b66de 修复目录整理误匹配）在本方是否需要：结论=不需要。证据：① internal/mediaorganize/rules 零 import（全仓唯一提及是 name_align.go:394 的一句注释）；② 不在主程序依赖图（go list -deps ./cmd/litepan | grep -c mediaorganize = 0）；③ 三类入口全无（API 路由空、前端源码与构建产物空、自动化动作只剩 local_upload）；④ 1bcfac8（2026-08-30 删除目录整理）在同一提交把 name_align 的 mrules 依赖换成自带简化解析→该包自那时起无消费者；⑤ 规模 15 非测试文件/3796 行、无外部依赖。上游对照：origin/main 的 name_align 仍 import mrules，其 rules 是活跃代码，本方两者都不用。据此 P6 从候选清单作废（9→8），并更正三处过期记述：调查报告 §3.2/§6、journal-1.md:252（'为 name_align 保留'与实际不符）、spec directory-structure.md:44（仍写保留）。方法论教训：适用性判据需从『文件存在』升级为『文件存在且存在可达入口/消费者』（Go 可用 go list -deps 一票否决）。建议（未实施）：可选清理任务删除孤儿包并同步 spec；保留亦可但易再次误导。

### Git Commits

(No commits - planning session)

### Testing

- [OK] 只读：grep import 链/路由/前端/构建产物/自动化动作 + go list -deps 依赖图 + git show 历史取证；零代码改动

### Status

[OK] **Completed**

### Next Steps

- 待用户决定：是否建清理任务删除 internal/mediaorganize 孤儿包（19 文件/3796 行）并同步 spec 两处过期表述


## Session 128: 死代码全面排查：3 孤儿包 + 24 死函数 + 12 前端文件
<!-- trellis-session: v=2 fp=5a760f70eef09136 -->

**Date**: 2026-09-12
**Task**: 死代码全面排查：3 孤儿包 + 24 死函数 + 12 前端文件
**Package**: backend
**Branch**: `main`

### Summary

按用户要求全面排查残留死代码（只读，六类）：① Go 包级 45 个仓库内包中 4 个不在 cmd/litepan 依赖图——internal/proxybase(163行)、internal/taskauth(128+100)、pkg/strutil(13) 为真死，drivers/template(438行) 为测试专用骨架（保留）；② deadcode 生产视角 32 个不可达，与 -test 差集得 24 真死（backuprestore 孤儿清理三连、apikey Validate/ValidateTask、automation normalizePath/ternaryStatus、upload 离线 handoff/lifecycle progressForBytes/target_dir/worker taskLocalPath、uploadutil HashMD5/ReadProgress/UploadedBytesByPartKeys、file asAlignInt、settings stringSpec、api streamSSEMessages、189 signedForm、pkg 三处、notification DeleteByRef 等）+ 8 仅测试可达（httpx OAuth 一套 5 个 + domain 暂停原因 2 个 + upload OfflineHandoffClientID）；③ 前端 223 源文件 12 个零引用（2237 行：spaceCleanup/海报墙/封面/TMDB/FUSE/WebDAV/缓存面板等），30 依赖 0 未用；④ 路由/驱动注册表/自动化动作/设置键无残留，仅 1 个疑似未使用端点 /accounts/{id}/refresh-auth；⑤ 本地库 11 表全在用（已删功能表由迁移 0022 清除），upload_tasks 跨盘列仍被本机上传使用，迁移历史 9 文件建议保留；⑥ 构建产物 109 asset 全被引用。产出按优先级清理清单（P1 死包+死函数 / P2 前端分两步 / P3 template+OAuth 默认保留 / P4 预留端点）与误报排除记录（HashMD5 同名常量、面板未接线≠功能已删、测试接口桩不计）。方法沉淀：go list -deps 包级 + deadcode(prod/-test 差集) 符号级 + 前端引用计数 + 解压 .gz 互引 + 叶子路由匹配。零代码改动。

### Git Commits

(No commits - planning session)

### Testing

- [OK] 只读：go list/go list -deps/deadcode(prod & -test)/grep 引用计数/sqlite ro/py 脚本（前端引用、依赖、产物互引、路由匹配）；未改代码与 DB；git status 仅任务目录

### Status

[OK] **Completed**

### Next Steps

- 待用户决策是否执行清理：建议 P1（3 死包 + 24 死函数）合并为一个纯删除任务并发版
- P2 前端 12 文件建议分两步：先删已删功能相关的 4 个，再确认 FUSE/WebDAV/缓存三块未接线 UI 是否保留
- P3 drivers/template 与 httpx OAuth 链路：若不再新增驱动则整链清理（438+~150 行），否则保留


## Session 129: 死代码清理 P1 并发布 0.0.41（3 孤儿包 + 23 死函数）
<!-- trellis-session: v=2 fp=1f45fe4ce9bcd6d4 -->

**Date**: 2026-09-12
**Task**: 死代码清理 P1 并发布 0.0.41（3 孤儿包 + 23 死函数）
**Package**: backend
**Branch**: `main`

### Summary

执行排查报告 §9 P1：① 删除 3 个零消费者包 internal/proxybase、internal/taskauth、pkg/strutil（删前 grep 0 import + 不在 cmd/litepan 依赖图）；② 删除 23 个真死函数（backuprestore 孤儿清理三连、apikey Validate/ValidateTask、automation normalizePath/ternaryStatus、upload offlineHandoffGroupID/progressForBytes/joinUploadDisplayPath/taskLocalPath、uploadutil HashMD5/UploadedBytesByPartKeys、file asAlignInt、settings stringSpec、api streamSSEMessages、notification DeleteByRef、driver WithExtraAPIDelay/LocalUploadEpochMillis、189Cloud signedForm、pkg/security RequestBaseURL、pkg/speedsmoother NewDefault），并整文件删除 uploadutil/progress.go（类型未使用）与 backuprestore/maintenance.go（仅剩无用类型）；③ 排除 1 项：pkg/jsonvalue.FlexibleString.UnmarshalJSON —— 实现 json.Unmarshaler、由 encoding/json 反射调用，deadcode(RTA) 观察不到，删除会改变解码语义，保留并记录（判据修正：凡实现标准接口的方法先人工复核）。验证：deadcode prod 32→9（无新增），go vet/test/build + web 三连全绿，tidy -diff 空；共 24 文件 -781 行；二进制体积不变（20885666B，差分为链接布局）。发版 0.0.41：三 tag 同 digest 6617d652、tag=69a3177（API 复核）、release 已建（顺序 push→tag→push tag→release）、本地容器重建三连通过。

### Main Changes

- 删除 internal/proxybase、internal/taskauth、pkg/strutil 三个零消费者包
- 删除 23 个真死函数 + 2 个整文件（progress.go、maintenance.go）
- 保留 FlexibleString.UnmarshalJSON（反射调用语义），并沉淀「标准接口方法先复核」判据
- 版本 0.0.41 + 镜像三 tag + tag/release + 本地部署

### Git Commits

| Hash | Message |
|------|---------|
| `69a3177` | chore: remove dead code P1 (3 orphan packages + 23 dead funcs), bump to 0.0.41 |

### Testing

- [OK] 删除前后 deadcode 对比（32→9，差集=23 消除、0 新增）；go vet ./... 全绿；go test ./... 全包 ok；go build 通过；web type-check+build+check:memo MEMO-ALL-PASS；go mod tidy -diff 为空；本地容器 health/表单登录/任务汇总三连通过

### Status

[OK] **Completed**

### Next Steps

- 待用户决策 P2：前端 12 个零引用文件（建议先删已删功能相关的 4 个，FUSE/WebDAV/缓存 8 个待确认）
- P3：drivers/template + httpx OAuth 测试专用链路（默认保留，若不再新增驱动可整链清理 ~588 行）
- P4：/accounts/{id}/refresh-auth 预留端点，建议文档标注而非删除


## Session 130: 死代码清理 P2 并发布 0.0.42（前端 23 文件，迭代到不动点）
<!-- trellis-session: v=2 fp=a3fee0031ccf5136 -->

**Date**: 2026-09-12
**Task**: 死代码清理 P2 并发布 0.0.42（前端 23 文件，迭代到不动点）
**Package**: web
**Branch**: `main`

### Summary

执行排查报告 §9 P2：删除前端零引用文件并迭代到不动点——三层 12→7→4，共 23 文件/-3017 行。第一层（报告原列 12）：spaceCleanup/AdminStartupBanner/CacheRuntimeStats/CacheSettingsPanel/FuseManagement(1071)/WebDAVSettings/useConditionalPolling/useLiveElapsedClock/useStartupCountdown/useVirtualPosterWall/coverPoster/tmdbHit；第二层（7 传递孤儿）：AdminSettingsDrawer/AdminStatusPill/AdminTaskTabHeader/InputActionField/SettingsFormRow/useAccountPathLabel/useSettingsForm；第三层（4）：AdminStatsGrid/SettingsEntryCard/SettingsRowLabel/adminTaskTabHeader。删除前逐个复核可达性：FUSE 另有在线 UI（api/fuse.ts 被 Dashboard/SystemSettings/AuxTools 用）、缓存在 SystemSettings 被 filterOutCacheSettings 显式过滤、WebDAV 前端无入口、其余属已删功能（垃圾清理/海报墙/封面/TMDB）或通用工具；传递层用 git grep HEAD 复核引用者全部属于已删集合。保留 constants/cacheSettings.ts 与 api/fuse.ts。收敛后 web/src 零引用=0（223→204 文件）。验证：前端三连全绿且构建产物零 churn（反证不在 bundle）、go build/vet/test 全绿（42 包）。发版 0.0.42：digest 同 0.0.41（预期）、tag=75be52b（API 复核）、release 顺序正确、本地容器重建三连通过。方法论沉淀：死代码清理必须迭代到不动点。

### Main Changes

- 删除前端第一层 12 个零引用文件
- 迭代删除第二/三层共 11 个传递孤儿
- 复核并保留 cacheSettings.ts 与 api/fuse.ts（在线使用）
- 版本 0.0.42 + 镜像三 tag + tag/release + 本地部署

### Git Commits

| Hash | Message |
|------|---------|
| `75be52b` | chore(web): remove dead frontend files P2 (23 files, -3017 lines), bump to 0.0.42 |

### Testing

- [OK] 前端 type-check/build/check:memo 全绿（产物零 churn）；go build/vet/test 全绿；收敛后复扫 zero-reference=0；本地容器 health/登录/任务汇总三连通过

### Status

[OK] **Completed**

### Next Steps

- P3（可选）：drivers/template + httpx OAuth 测试专用链路，默认保留；若不再新增驱动可整链清理
- P4（可选）：/accounts/{id}/refresh-auth 预留端点，建议文档标注而非删除


## Session 131: P3/P4 裁定：均保留不修改，死代码清理收尾
<!-- trellis-session: v=2 fp=df78f1f1a2909fa3 -->

**Date**: 2026-09-12
**Task**: P3/P4 裁定：均保留不修改，死代码清理收尾
**Package**: backend
**Branch**: `main`

### Summary

裁定死代码排查报告遗留的 P3/P4，结论：两项均保留、不修改、零代码改动。P3 drivers/template（438 行）+ httpx OAuth 辅助：不是遗留死代码，而是① 统一认证守卫集成测试的驱动矩阵替身（oauth_integration_test.go:14 空导入，注释说明原 123_Open/Baidu_Open/OneDrive 已在本方移除），② spec driver-development.md:71 指定的新驱动骨架（cp -r drivers/template drivers/FooCloud）；保留成本≈0（生产不可达且链接器不进二进制，0.0.40 已实测删除死包后二进制逐字节相同），删除则削弱认证测试覆盖并移除脚手架。P4 /accounts/{id}/refresh-auth：全仓仅 router.go:180 一处引用、handler 30 行、前端只有 refresh-profile（api/accounts.ts:12）、无文档/脚本引用；属人工/脚本强制刷新账号认证的运维预留端点，系统本身有自动刷新不依赖它；删除＝改变已公开 HTTP 契约而收益仅 30 行，故保留。至此排查报告 §9 的 P1~P4 全部关闭：P1（0.0.41 三死包+23 死函数）、P2（0.0.42 前端 23 文件/-3017 行）已实施，P3/P4 裁定保留。

### Git Commits

(No commits - planning session)

### Testing

- [OK] 只读取证：grep/git grep 引用链 + spec 引用核对 + handler 阅读；零代码改动（git status 仅任务目录）

### Status

[OK] **Completed**

### Next Steps

- 如将来确定不再新增驱动或该端点长期无人使用，可另建任务做契约级清理（届时属有意变更，需同步 spec）


## Session 132: 移植上游 P2+P5（日志与可观测）并发布 0.0.43
<!-- trellis-session: v=2 fp=e8c1c3af667c8142 -->

**Date**: 2026-09-12
**Task**: 移植上游 P2+P5（日志与可观测）并发布 0.0.43
**Package**: backend
**Branch**: `main`

### Summary

移植候选清单 P2+P5：① P2（upstream d545e47）logAPIError 加取消/超时守卫——客户端提前断开不再记 ERROR，新增 error_log_cancel_test.go（取消/超时/正常三例）；② P5①（upstream b5c9308）新增 internal/api/slow_dashboard_log.go：后台概况类 GET 超 1s 记 INFO（method/path/duration_ms），router 于 attachRequestLogger 后接线，路径表按本方在线端点裁剪（上游含 cache-retention/media-organize/strm 等已删端点，代码注释说明），含白名单与阈值行为测试；③ P5②（upstream b5c9308）accountprofile 增加容量 1 的 refreshGate 串行闸门（父 ctx 取消即返回），移植上游串行性测试并补取消用例。质量门 vet/test/build/web 三连全绿。发版 0.0.43：三 tag 同 digest 6fd9d631、tag=ac6ec4a（API 复核）、release 顺序正确、本地容器重建三连通过。

### Main Changes

- P2：logAPIError 取消/超时守卫 + 回归测试
- P5①：慢后台接口日志（1s 阈值，路径表按本方端点裁剪）
- P5②：accountprofile refreshGate 后台刷新串行化
- 版本 0.0.43 + 镜像三 tag + tag/release + 本地部署

### Git Commits

| Hash | Message |
|------|---------|
| `ac6ec4a` | feat(api): port upstream P2+P5 log/observability fixes, bump to 0.0.43 |

### Testing

- [OK] go vet ./... 全绿；go test ./... 全绿（28 包 ok）；go build OK；web type-check+build+check:memo MEMO-ALL-PASS；新增 internal/api/error_log_cancel_test.go、slow_dashboard_log_test.go、internal/accountprofile/service_test.go（含上游串行性测试）；本地容器三连通过

### Status

[OK] **Completed**

### Next Steps

- 候选清单剩余：P7（上传取消语义/批次名）、P8（铃铛 SSE 推送）、P9（播放诊断）


## Session 133: 关闭候选 P3/P4：P3 归档不实施、P4 判定本方不适用
<!-- trellis-session: v=2 fp=b3386af9af7a7b8b -->

**Date**: 2026-09-12
**Task**: 关闭候选 P3/P4：P3 归档不实施、P4 判定本方不适用
**Package**: backend
**Branch**: `main`

### Summary

关闭上游候选清单 P3/P4（零代码改动）。P3（99ea858 认证收口）：按用户指示归档不实施——上游属重构+日志降噪非缺陷修复，本方 0.0.17 已同源移植（不含该批），认证链路无已知缺陷；重启条件=出现认证类问题（刷新风暴/调度抖动）。P4（df86352 账号级目录缓存失效）：查证判定本方不适用——① 上游所改的自动化动作在本方不存在（domain/automation.go 仅 local_upload，刷新目录/清缓存动作在 1bcfac8 精简时已删）；② 本方已有等价且更精确的能力：cache.InvalidateAccount（service.go:197）、写路径按目录失效（file/service.go:62/242/413）、事件驱动失效（cache/cleaner.go）、管理端 clear-cache→ClearAll（api/cache.go:55）；③ 强行移植只会新增无调用者 helper＝新死代码，与排查目标相反。候选收尾状态：P1✅(0.0.38/0.0.39)、P2✅(0.0.43)、P5✅(0.0.43)、P6❌作废、P3 归档、P4 不适用，剩 P7/P8/P9。方法论：候选移植必须查证目标载体是否仍存在（P4 与 P6 同教训）。

### Git Commits

(No commits - planning session)

### Testing

- [OK] 只读取证：grep 动作常量/缓存失效调用点/router 与函数阅读；零代码改动（git status 仅任务目录）

### Status

[OK] **Completed**

### Next Steps

- 待用户确认：P7+P9（顺带优化）是否立即开工；P8（铃铛改 SSE 推送）属功能项需单独确认


## Session 134: 候选清单收尾：P7/P8/P9 归档（含 P7 已等价与有意差异核实）
<!-- trellis-session: v=2 fp=d85b5beaa2b7c94d -->

**Date**: 2026-09-12
**Task**: 候选清单收尾：P7/P8/P9 归档（含 P7 已等价与有意差异核实）
**Package**: backend
**Branch**: `main`

### Summary

按用户指示归档候选 P7/P8/P9（零代码改动），并逐项核实后再记录：P7（ae75882 子集）——① 取消语义本方已内联等价（internal/api/upload.go:34/64/102 与上游 isClientGone/isCancelError 判断完全一致，上游只是抽 helper，移植零收益）；② local_upload 的 batchName 派生（local_upload.go:288-290）是面板显示文件夹批次所需的有意行为，照搬上游删除会造成显示回归；③ upload/resume.go 定时器简化属代码卫生无行为差异；重启条件=出现取消/中断处理类缺陷。P8（adca0ee 铃铛 SSE 推送）——功能增强非修复，本方轮询无缺陷，重启条件=需秒级通知或减少轮询。P9（46a0a89 播放诊断）——排障能力，本方 playback 无诊断且无故障驱动，重启条件=出现播放/直链故障需定位。候选清单最终状态：3 项已落地（P1 0.0.38+0.0.39、P2 0.0.43、P5 0.0.43）、2 项作废/不适用（P6 死代码作废、P4 目标动作已删）、4 项按用户决定归档（P3/P7/P8/P9）——清单全部关闭。

### Git Commits

(No commits - planning session)

### Testing

- [OK] 只读取证：grep 行号定位 + 文件阅读 + 候选对照；零代码改动（git status 仅任务目录）

### Status

[OK] **Completed**

### Next Steps

- 候选清单已全部关闭；后续如遇认证类问题、通知实时性需求或播放故障，可按记录的重启条件对应重启 P3/P8/P9


## Session 135: 安装本机完整开发环境：Docker 29.7.2 + Go 1.26.6 + make + gcc + golangci-lint
<!-- trellis-session: v=2 fp=722a5f3f36b2eb93 -->

**Date**: 2026-09-12
**Task**: 安装本机完整开发环境：Docker 29.7.2 + Go 1.26.6 + make + gcc + golangci-lint
**Package**: backend
**Branch**: `main`

### Summary

全新 Ubuntu 22.04 主机从零装齐五项工具链（零源码改动）。Docker 29.7.2 + Compose v5.4.0 复用仓库 install-docker.sh 并 apt-mark hold 锁版，daemon active、hello-world 通过；Go 1.26.6 官方 tarball sha256 校验后装 /usr/local/go 并加 /usr/local/bin 软链（DSH 非登录 shell 才可用）；make 4.3；golangci-lint v2.12.2 走 go install 命中 Makefile 回退路径；build-essential/gcc 11.4.0 为规划期补漏项（go test -race 依赖 cgo）。质量门全绿：make lint 0 issues、go vet exit=0、go test ./... 全包 ok、go test -race ./internal/store ok 3.518s、web vue-tsc -b exit=0。

### Main Changes

- 环境安装（宿主，非仓库）：Docker 29.7.2 + Compose v5.4.0 + containerd + buildx（apt-mark hold）；Go 1.26.6（sha256 708effb7…）；make 4.3；build-essential/gcc 11.4.0；golangci-lint v2.12.2
- 新增任务产物：.trellis/tasks/archive/2026-09/09-12-setup-dev-environment/{prd.md,design.md,implement.md,setup-env.sh,.check-passed}；scope=multi-deliverable
- 规划期回退修正：原路线 D 四项清单缺 gcc，而项目强制 go test -race 依赖 cgo —— 在零系统写入前补入 build-essential 并同步 design/implement

### Git Commits

| Hash | Message |
|------|---------|
| `0b7207c` | chore(task): archive 09-12-setup-dev-environment |

### Testing

- [OK] [OK] make lint → 0 issues（验证 Makefile 的 GOPATH/bin 回退路径与 depguard 链）
- [OK] [OK] GOWORK=off go vet ./... → exit=0；go test ./... → 全包 ok 无失败；go test -race ./internal/store → ok 3.518s
- [OK] [OK] cd web && npm ci（145 包）+ npm run type-check → exit=0；internal/api/web 受跟踪产物 0 diff
- [OK] [OK] docker run --rm hello-world 成功；docker info Server=29.7.2 Driver=overlayfs
- [OK] [OK] 边界未破坏：git diff --name-only HEAD = 0；3080/3081 仍由 DSH 监听；5211 仍空闲
- [OK] [注意] 刻意未跑 npm run build（会覆写 135 个受跟踪 embed 产物），属有意偏离并已在 prd.md 留痕

### Status

[OK] **Completed**

### Next Steps

- 部署 LitePan 到本机 :5211：需先确认路线（拉取现成镜像 ghcr.io/zhemed/litepan:v0.0.43 vs 本地 docker build）
- 候选：把 pipefail + head/grep -q 的 SIGPIPE 假失败陷阱写入 .trellis/spec/guides/（本任务因'零受跟踪文件改动'约束未执行，待用户决定）
- 部署任务需处理：本机为新库，管理员为首次启动默认值；上机备份 /tmp/litepan-backup-20260909-201827.db 不在本机


## Session 136: 补齐 golangci-lint 全局 PATH 软链
<!-- trellis-session: v=2 fp=d3b1b74f7ba392e7 -->

**Date**: 2026-09-12
**Task**: 补齐 golangci-lint 全局 PATH 软链
**Package**: backend
**Branch**: `main`

### Summary

承接 09-12-setup-dev-environment 的收尾遗留项：golangci-lint v2.12.2 原装于 /root/go/bin（不在 PATH），裸 shell 调用报 command not found。按用户指示建 /usr/local/bin/golangci-lint 软链（ln -sfn 幂等），与已有 go/gofmt 软链保持一致。验证：非登录非交互 shell 直出 2.12.2、command -v 命中 /usr/local/bin/golangci-lint、make lint 复跑 0 issues exit=0。零仓库改动（git diff --name-only HEAD = 0），3080/3081 DSH 未受影响、5211 仍空闲。

### Main Changes

- 宿主改动：/usr/local/bin/golangci-lint -> /root/go/bin/golangci-lint 软链（零仓库受跟踪文件改动）
- 新增任务产物：.trellis/tasks/archive/2026-09/09-12-golangci-lint-path-symlink/{prd.md,.check-passed}；scope=lightweight

### Git Commits

| Hash | Message |
|------|---------|
| `512d5df` | chore(task): archive 09-12-golangci-lint-path-symlink |

### Testing

- [OK] [OK] 非登录非交互 shell：golangci-lint --version → 2.12.2；command -v → /usr/local/bin/golangci-lint
- [OK] [OK] 回归：make lint → 0 issues. exit=0（软链未破坏 Makefile 的 GOPATH 回退路径）
- [OK] [OK] 边界：git diff --name-only HEAD = 0；3080/3081 仍由 DSH 监听；5211 仍空闲

### Status

[OK] **Completed**

### Next Steps

- 部署 LitePan 到本机 :5211（待用户确认路线：拉取 ghcr.io/zhemed/litepan:v0.0.43 vs 本地 docker build）
- 候选：将 pipefail + head/grep -q 的 SIGPIPE 假失败陷阱写入 .trellis/spec/guides/（连续两个任务因零改动约束未执行）


## Session 137: 项目与工作区维护：版本漂移修复 + 陈旧配置/文档清理 + Trellis 升级 0.6.17
<!-- trellis-session: v=2 fp=d849bc293f5e6c72 -->

**Date**: 2026-09-12
**Task**: 项目与工作区维护：版本漂移修复 + 陈旧配置/文档清理 + Trellis 升级 0.6.17
**Package**: backend
**Branch**: `main`

### Summary

承接用户「先把工作区和项目维护一下」的六项授权（D1-D6），全部基于只读巡查结论、零源码改动。D1 修 1.26.4→1.26.6 版本漂移三处（.golangci.yml run.go、Dockerfile 注释、spec quality-guidelines.md 内嵌示例）；D2 删除 .trellis/config.yaml 中与 AGENTS.md 冲突的过期管理员口令/hash，改为指向单一权威；D4 新增 spec/guides/shell-script-guide.md（130 行）并登记索引，沉淀上一任务两个真实假失败；D5 删 6 张孤儿文档图片（保留 banner/feature-browser）；D6 trellis update 0.6.16→0.6.17 走 skip-all 保护路径（config.yaml 逐字节未变）。另排除两项假警报：go:embed 前端产物经死代码引用反查确认无漂移；/tmp/gh-u.json 仅是 GitHub 公开资料无密钥。

### Main Changes

- D1+D2+D4：.golangci.yml、Dockerfile、spec/backend/backend/quality-guidelines.md、.trellis/config.yaml、spec/guides/index.md（+新增 shell-script-guide.md）
- D5：删除 6 张孤儿文档图片（feature-strm/strm-scrape/crosstransfer/organize/automation、wechat-tip）
- D6：trellis 0.6.16→0.6.17，4 个模板文件自动更新（active_task.py、task_store.py、2 个 session-insight skill 文档）
- D3（宿主）：清 /tmp 安装残留 5 个文件 + .trellis/.runtime 陈旧 marker，释放 66M（380M→314M）

### Git Commits

| Hash | Message |
|------|---------|
| `30a20a1` | chore(maintain): fix 1.26.4 version drift, prune stale config/docs, upgrade trellis to 0.6.17 |

### Testing

- [OK] [OK] 回归全绿：make lint 0 issues（关键：run.go 改 1.26.6 后仍解析）、go vet exit=0、go test ./... 全包 ok、vue-tsc -b exit=0
- [OK] [OK] 升级后脚本自检三连：get_context.py / task.py list（正确绑定 current 任务）/ flow_gate.py pre-start
- [OK] [OK] 边界：internal/drivers/pkg/cmd/web/go.mod/go.sum/Makefile/README 改动数均为 0；3080/3081 DSH 未受影响
- [OK] [注意] 未做版本 bump —— 0.0.43 只是镜像 tag 非源码常量，且本次零产品代码改动

### Status

[OK] **Completed**

### Next Steps

- 部署 LitePan 到 :5211（用户已明确先搁置）
- 候选发现：internal/buildinfo/version.go 默认值为上游 v0.5.2-Beta，而 Dockerfile 构建未传 -ldflags -X ...Version= 覆盖 → 容器内版本自报可能与镜像 tag v0.0.43 不一致，待核实是否有意为之


## Session 138: 版本号收敛为单一来源（后端运行期提供）并发布 v0.0.44
<!-- trellis-session: v=2 fp=5393c4e84df17e92 -->

**Date**: 2026-09-12
**Task**: 版本号收敛为单一来源（后端运行期提供）并发布 v0.0.44
**Package**: backend
**Branch**: `main`

### Summary

修复项目诞生起就存在的问题：版本号从上游继承后从未更新 —— internal/buildinfo、web/src/version.ts、internal/httpx/user_agent.go 三处各自持有 v0.5.2-Beta 副本，Dockerfile 也未注入 -ldflags，导致界面『当前版本/关于』与发给上游的 User-Agent 长期报错版本。按用户选定的方案C 收敛为唯一真值 buildinfo.Version：Deps/Handler 注入 + /api/public/system-config 暴露 version 字段 + 前端新增 appInfo store 运行期读取（inflight 去重，无 fallback 字面量）+ version.ts 删两个字面量 + GITHUB_URL 改指 zhemed/LitePan。httpx 的 AppVersion/DefaultUserAgent 改为由该值派生（连带 115 驱动 ossUserAgent 由 const 改 var）。重建 109 个 embed 产物。完整发版：镜像 v0.0.44/0.0.44/latest 三 tag 同 digest sha256:a864057d46ac，git tag 远端==本地 593d125（未重演 0.0.39 顺序问题），release 已建，并删本地镜像从 GHCR 真拉回重验三项全过。

### Main Changes

- 后端：api.Deps/Handler 新增 Version（app 层注入 buildinfo.Version）；publicSystemConfig 新增 version 字段（纯新增，既有 3 字段不变）；新增 internal/api/public_version_test.go
- 第三处硬编码（实施期由验收项抓出）：internal/httpx/user_agent.go 改由 buildinfo.Version 运行期派生；drivers/115_Open/upload.go 的 ossUserAgent 由 const 改 var（仅关键字，值与 3 处用法不变）
- 前端：新增 stores/appInfo.ts（inflight 去重）；AppFooter.vue/AdminAccountChip.vue 改经 store 取值；version.ts 删 APP_VERSION 与 APP_VERSION_BADGE、GITHUB_URL 改指 zhemed/LitePan
- 版本与产物：buildinfo v0.0.44；README×2 + docker-compose×1 镜像 tag 同步；重建 internal/api/web（54 新/54 删/1 改）
- spec 同步：api-layering.md 新增『版本号单一来源』小节，含历史教训与回归 grep，防止后人再写死版本号

### Git Commits

| Hash | Message |
|------|---------|
| `593d125` | feat(version): backend becomes the single source of truth for the version, bump to 0.0.44 |
| `7f5e0f2` | docs(spec): 记录版本号单一来源约定（v0.0.44 起） |

### Testing

- [OK] [OK] 质量门：make lint 0 issues；go vet exit=0；go test 28 包全 ok（含新测试 2 子测试）；vue-tsc -b exit=0
- [OK] [OK] 单一真值达成：grep v0.0.44 在全部代码中只命中 version.go 一处；grep v0.5.2 零命中
- [OK] [OK] 产物复核：解压 102 个 embed 产物，0 处含上游版本号、含 public/system-config（证明改运行期取）
- [OK] [OK] 发版：GHCR API 显示单条 version(id 1240621421) 挂载 v0.0.44/0.0.44/latest 三 tag 同 digest；远端 tag 与本地 593d125 一致；release 已建
- [OK] [OK] 拉回重验：删本地镜像 → docker pull → digest 一致 → 容器三项（health 200 / version=v0.0.44 / 表单登录 200）全过
- [OK] [OK] 边界：5211 仍空闲（部署仍搁置）；5212 已释放；DSH 3080/3081 未受影响；仓库无 data//mounts/；容器已清理

### Status

[OK] **Completed**

### Next Steps

- 部署 LitePan 到 :5211（用户已明确搁置，随时可开）
- 未决观察：README 首屏声称 118M，实测 v0.0.44 压缩后 41.8MiB / 未压缩 161MB，三者口径不一致（118M 是 v0.0.1 时代数字），是否更新待用户决定
- 本机新库默认口令为 admin/admin（非 AGENTS.md 记录的 123456，后者是上一台机器重置后的值）；正式部署时需决定是否改密


## Session 139: 部署 LitePan v0.0.44 到本机 :5211（含数据持久化验证）
<!-- trellis-session: v=2 fp=ec3b654641c022d6 -->

**Date**: 2026-09-12
**Task**: 部署 LitePan v0.0.44 到本机 :5211（含数据持久化验证）
**Package**: backend
**Branch**: `main`

### Summary

按用户指示把已发布的 v0.0.44 部署到本机 :5211，用仓库 tracked docker-compose.yml（未修改），零仓库受跟踪改动。起始前提：Docker 29.7.2+Compose v5.4.0 active、镜像已在本地、5211/42069 空闲、data/ 不存在（首次启动只新建不覆盖）。部署结果：容器 litepan Up、5211 双栈发布（0.0.0.0+[::]）、约 4s 就绪；health 200、SPA 首页含 LitePan、表单编码登录 200（is_admin+must_change_password）、/api/public/system-config 返回 version=v0.0.44（顺带在部署形态下复核了上一任务成果）、日志含『HTTP 服务已监听』、RestartPolicy=unless-stopped。持久化用四条与文件大小无关的证据证明：同一 inode 13533519 全程未变、只读读库得 11 张表与真实 pbkdf2 哈希、secret.key inode/mtime 未变、重启前的 session cookie 在整容器重启后仍通过鉴权。边界：DSH 3080/3081 的 node PID 与基线完全一致、git 零受跟踪改动、data/ 与 mounts/ 由 gitignore 覆盖。

### Main Changes

- 宿主/容器（非仓库）：新增容器 litepan（ghcr.io/zhemed/litepan:v0.0.44）、发布端口 5211、创建 data/ 与 mounts/（均 gitignored）
- 任务产物：.trellis/tasks/archive/2026-09/09-12-deploy-litepan-local-5211/{prd.md,design.md,implement.md,.check-passed}；scope=infra
- 选型依据：本机用根 docker-compose.yml（相对路径+ports 收敛），不用 README/fnos 变体（NAS 绝对路径 + host 网络）

### Git Commits

| Hash | Message |
|------|---------|
| `ebf95cd` | chore(task): archive 09-12-deploy-litepan-local-5211 |

### Testing

- [OK] [OK] 可用性：health 200 / SPA 200 / 表单登录 200 / version=v0.0.44 / 日志含监听行
- [OK] [OK] 持久化：inode 同一 + 只读读库 11 表与真实哈希 + secret.key 未重建 + 重启前 cookie 仍有效
- [OK] [OK] 边界：DSH PID 与基线一致；git status 仅任务目录；git check-ignore 证实 data//mounts/ 被忽略
- [OK] [教训] 持久化验证中我先后用『文件清单』与『db+wal+shm 总字节数』两个无效判据产生假阴性 —— WAL 模式下 checkpoint 使总字节数下降属正常；已改用与大小无关的判据

### Status

[OK] **Completed**

### Next Steps

- 用户在浏览器改默认口令后，建议同步更新 AGENTS.md 的口令记录（属规则文件，需另建任务）
- 待决：docker-compose.fnos.yml 的 image tag 仍为 v0.0.31（落后 13 版），bump 会把多版本变更推给 NAS 用户且本机无法验证 fnOS
- 待决：README 首屏声称 118M，实测 v0.0.44 压缩后 41.8MiB / 未压缩 161MB，口径不一致


## Session 140: 修正 NAS compose 过期 tag 与目录分歧、清除三处不可核实的体积数字
<!-- trellis-session: v=2 fp=2e8b4952ac369486 -->

**Date**: 2026-09-12
**Task**: 修正 NAS compose 过期 tag 与目录分歧、清除三处不可核实的体积数字
**Package**: backend
**Branch**: `main`

### Summary

收尾前两轮暴露的两项遗留（零代码改动，4 文件 6 insertions/6 deletions）。① docker-compose.fnos.yml 双重过期：image v0.0.31→v0.0.44（落后 13 版），数据目录 litepango→litepan 与 README 快速开始统一（此前照 README 抄会落到不同目录）；config --quiet 通过，其余字段未动。② 全仓库三处 118M（README:9/AGENTS.md:62/.trellis/config.yaml:9）一并清除、按用户决策不填新值；取证方式为从 GHCR 分页查出仍在的 v0.0.1 并 docker pull 实测——未压缩 178MB、压缩层 46.0MiB，与 118M 均不符，且 v0.0.44 更小（161MB/41.8MiB，因功能精简），故该数字是『从未准确』而非『过时』。保留仍准确的『3驱动』等声明；config.yaml 非注释部分与备份零差异。回归：运行中容器未受影响（Up + health 200）。

### Main Changes

- docker-compose.fnos.yml：image 升至 v0.0.44；两处卷路径 /vol1/1000/docker/litepango/ → /vol1/1000/docker/litepan/
- README.md:9：去掉 118M → **115 · 天翼 · 本机 · 一个界面**（diff 仅此一行）
- AGENTS.md:62 与 .trellis/config.yaml:9：版本基线句去掉 118M，保留 0.0.1 基线/已推镜像/tag/递增规则/不跳 1.0.0 与『3驱动』

### Git Commits

| Hash | Message |
|------|---------|
| `106bfc0` | fix(docs): 修正 NAS compose 的过期 tag 与目录分歧、清除不可核实的体积数字 |
| `343b6d8` | chore(task): archive 09-12-fix-nas-compose-and-size-claim |

### Testing

- [OK] [OK] grep 118M 与 litepango 均零命中；docker compose -f docker-compose.fnos.yml config --quiet 通过
- [OK] [OK] 未误改核验：仅 3 驱动 / 115_open / 189_cloud / localfs / Vue 3.5.41 / v0.0.44 / go 1.26.6 全部在位
- [OK] [OK] config.yaml 非注释部分与上轮备份 diff 零差异（只动注释）
- [OK] [OK] 边界：运行中的 litepan 容器未受影响（Up + /api/health 200）

### Status

[OK] **Completed**

### Next Steps

- 已知影响待你判断：NAS 用户从 v0.0.31 直接跳到 v0.0.44 会跨 13 个版本（含 STRM/跨盘秒传/多驱动删除），如需分步升级请改中间版本自行验证


## Session 141: 同步管理员口令记录到本机实例（用户 2026-09-12 改密）
<!-- trellis-session: v=2 fp=4637916f7e116f57 -->

**Date**: 2026-09-12
**Task**: 同步管理员口令记录到本机实例（用户 2026-09-12 改密）
**Package**: backend
**Branch**: `main`

### Summary

用户在本机 :5211 实例界面完成改密后，同步 AGENTS.md 的权威记录（单文件 2 insertions/1 deletion，零代码改动）。原记录停留在上一台机器：哈希 aa9764 与其备份路径 /tmp/litepan-backup-20260909-201827.db 均不属于本机（后者早前已实测不存在）。主记录更新为本机现状（2026-09-12 改密、新盐哈希 af0dac85f9dad16、库位于 /root/LitePan/data/litepan.db、must_change_password 已解除），并新增「口令变更史」标注每一条属于哪台机器以防混淆；流程约束原文保留。核实：新口令 admin/123456 登录 200、旧口令 admin/admin 401；DB mtime 与任务开始前逐位一致，证明未触碰运行数据；Trellis 托管块零改动。风险事实（仓库 public + 实例监听 0.0.0.0 ⇒ 该记录等同公开凭据）已在提问中明示并给出三个替代方案，用户答复『这个是给你维护用的，你记一下就行了』后按原做法执行，事实已留档。

### Main Changes

- AGENTS.md：口令主记录更新为本机现状（af0dac85f9dad16 / 2026-09-12 / /root/LitePan/data/litepan.db / must_change_password 已解除）
- AGENTS.md：新增「口令变更史」，标注 aa9764 与 /tmp 备份属上一台机器、该备份不在本机

### Git Commits

| Hash | Message |
|------|---------|
| `4236da0` | chore(docs): 同步管理员口令记录到本机实例（2026-09-12 改密） |
| `c95dd14` | chore(task): archive 09-12-update-admin-credential-record |

### Testing

- [OK] [OK] 新口令登录 200 且 must_change_password:false；旧口令 401（改密确实生效）
- [OK] [OK] DB mtime 与任务开始前逐位一致（12:22:03.940111862），未触碰运行数据
- [OK] [OK] 实例未受影响：容器 Up、/api/health 200
- [OK] [OK] Trellis 托管块零改动（diff 中标记出现 0 次），不会被 trellis update 覆盖

### Status

[OK] **Completed**

### Next Steps

- 若日后要收紧凭据暴露：有效手段是『不记录 + 改强口令』，而非改记哈希（123456 的 pbkdf2 可秒破）
