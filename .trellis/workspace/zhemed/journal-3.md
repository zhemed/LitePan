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
