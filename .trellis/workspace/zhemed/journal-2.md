# Journal - zhemed (Part 2)

> Continuation from `journal-1.md` (archived at ~2000 lines)
> Started: 2026-08-31

---



## Session 56: Adjust task concurrency width
<!-- trellis-session: v=2 fp=bab91bbefdfb1382 -->

**Date**: 2026-08-31
**Task**: Adjust task concurrency width
**Package**: web
**Branch**: `main`

### Summary

满宽 0.0.9

### Main Changes

- 420px -> 100% 满宽
- 红框窄留白

### Git Commits

| Hash | Message |
|------|---------|
| `912b97d` | style: expand task concurrency to full width and bump to 0.0.9 |

### Testing

- [OK] type-check 0 build 102

### Status

[OK] **Completed**


## Session 57: Investigate cross disk download
<!-- trellis-session: v=2 fp=f60578c37057425d -->

**Date**: 2026-09-01
**Task**: Investigate cross disk download
**Package**: backend
**Branch**: `main`

### Summary

跨盘下载可彻底移除

### Main Changes

- 清单81+8
- 零耦合

### Git Commits

| Hash | Message |
|------|---------|
| `41cc6c6` | chore(task): archive 08-31-adjust-task-concurrency-width |

### Testing

- [OK] grep

### Status

[OK] **Completed**


## Session 58: Remove cross disk download
<!-- trellis-session: v=2 fp=5dff022b453c4fba -->

**Date**: 2026-09-01
**Task**: Remove cross disk download
**Package**: backend
**Branch**: `main`

### Summary

移除跨盘下载 0.0.10

### Main Changes

- 8后端+3前端
- cross_transfer 81+8

### Git Commits

| Hash | Message |
|------|---------|
| `9004274` | refactor: remove cross disk download and bump to 0.0.10 |

### Testing

- [OK] vet 0 type-check 0 docker 105MB

### Status

[OK] **Completed**


## Session 59: Rollback to 65b868b
<!-- trellis-session: v=2 fp=dfd9140b0c1aa6f5 -->

**Date**: 2026-09-01
**Task**: Rollback to 65b868b
**Package**: backend
**Branch**: `main`

### Summary

回滚远端错误至65b868b

### Main Changes

- git reset --hard 65b868b
- git push --force

### Git Commits

| Hash | Message |
|------|---------|
| `65b868b` | chore(task): archive 09-01-remove-cross-disk-download |

### Testing

- [OK] git log 65b868b

### Status

[OK] **Completed**


## Session 60: Investigate GitHub local gap
<!-- trellis-session: v=2 fp=1ab2e494f596df57 -->

**Date**: 2026-09-01
**Task**: Investigate GitHub local gap
**Package**: backend
**Branch**: `main`

### Summary

GitHub与本地0差距

### Main Changes

- HEAD 1a77f58一致
- 0.0.10一致

### Git Commits

| Hash | Message |
|------|---------|
| `1a77f58` | chore(task): archive 09-01-rollback-to-65b868b |

### Testing

- [OK] git log/status

### Status

[OK] **Completed**


## Session 61: Deploy latest and verify
<!-- trellis-session: v=2 fp=9d25a353d3b925a3 -->

**Date**: 2026-09-01
**Task**: Deploy latest and verify
**Package**: backend
**Branch**: `main`

### Summary

拉取latest 0.0.10部署验证

### Main Changes

- docker pull latest 18bf16f
- docker run latest health ok

### Git Commits

| Hash | Message |
|------|---------|
| `1a77f58` | chore(task): archive 09-01-rollback-to-65b868b |

### Testing

- [OK] docker ps latest/curl health

### Status

[OK] **Completed**


## Session 62: Investigate webhook removal
<!-- trellis-session: v=2 fp=dd87da947f6f7c0e -->

**Date**: 2026-09-01
**Task**: Investigate webhook removal
**Package**: backend
**Branch**: `main`

### Summary

Webhook可彻底移除

### Main Changes

- 清单8+4
- 零耦合

### Git Commits

| Hash | Message |
|------|---------|
| `1a77f58` | chore(task): archive 09-01-rollback-to-65b868b |

### Testing

- [OK] grep

### Status

[OK] **Completed**


## Session 63: Remove webhook completely
<!-- trellis-session: v=2 fp=7a7469a43b2cb579 -->

**Date**: 2026-09-01
**Task**: Remove webhook completely
**Package**: backend
**Branch**: `main`

### Summary

移除Webhook 0.0.11

### Main Changes

- 8+4 files
- trigger daily|interval only

### Git Commits

| Hash | Message |
|------|---------|
| `5f72e15` | refactor: remove webhook trigger completely and bump to 0.0.11 |

### Testing

- [OK] vet 0 type-check 0 docker 105MB

### Status

[OK] **Completed**


## Session 64: Build latest locally
<!-- trellis-session: v=2 fp=ebe6aa59c6d79781 -->

**Date**: 2026-09-01
**Task**: Build latest locally
**Package**: backend
**Branch**: `main`

### Summary

本地构建0.0.11验证

### Main Changes

- go vet 0 type-check 0
- docker 105MB local 0.0.11-local health 200

### Git Commits

| Hash | Message |
|------|---------|
| `5f72e15` | refactor: remove webhook trigger completely and bump to 0.0.11 |

### Testing

- [OK] go build 27M web build 102

### Status

[OK] **Completed**


## Session 65: Investigate trigger cleanup
<!-- trellis-session: v=2 fp=3d031fa257cc2844 -->

**Date**: 2026-09-01
**Task**: Investigate trigger cleanup
**Package**: web
**Branch**: `main`

### Summary

触发条件已干净，部署未同步

### Main Changes

- 代码 0 第三方
- 容器 0.0.9 -> 0.0.11

### Git Commits

| Hash | Message |
|------|---------|
| `5f72e15` | refactor: remove webhook trigger completely and bump to 0.0.11 |

### Testing

- [OK] grep vet

### Status

[OK] **Completed**


## Session 66: Interval hours to minutes
<!-- trellis-session: v=2 fp=da10f840f175f590 -->

**Date**: 2026-09-01
**Task**: Interval hours to minutes
**Package**: backend
**Branch**: `main`

### Summary

间隔小时改分钟 0.0.12

### Main Changes

- backend minutes compat
- 前端 间隔分钟

### Git Commits

| Hash | Message |
|------|---------|
| `03e90f7` | refactor: interval hours -> minutes and bump to 0.0.12 |

### Testing

- [OK] vet 0 type-check 0 docker 105MB

### Status

[OK] **Completed**


## Session 67: 调查上游合并安全性：不建议 merge，改按需 cherry-pick
<!-- trellis-session: v=2 fp=a1457bbbdabcf38b -->

**Date**: 2026-09-09
**Task**: 调查上游合并安全性：不建议 merge，改按需 cherry-pick
**Package**: backend
**Branch**: `main`

### Summary

只读模拟合并 origin/main(374affd, v0.5.4-beta)：merge-tree 68 冲突文件（50 个为本方已删功能 delete/modify，18 个保留文件 content 冲突，automation 5 文件方向性冲突不可自动解决）。结论 C：不合并，按需移植 8e332f3 认证刷新/1c71fec 连接检测/c7a424c 189Cloud 认证子集/353b830 上传目录错位；de83b46 上传批次化与本方精简 worker 语义相撞不建议。全程只读，工作区干净，未改业务代码。

### Git Commits

(No commits - planning session)

### Testing

- [OK] git merge-tree --write-tree 模拟 + 保留文件上下游 diff 对照

### Status

[OK] **Completed**

### Next Steps

- 如需上游修复：逐项建任务 cherry-pick 并 bump 0.0.13


## Session 68: 适配合并上游保留功能修复并发布 0.0.13
<!-- trellis-session: v=2 fp=9cec962eb22e55a7 -->

**Date**: 2026-09-09
**Task**: 适配合并上游保留功能修复并发布 0.0.13
**Package**: backend
**Branch**: `main`

### Summary

按 investigate-upstream-merge-safety 结论移植上游 4 组修复：8e332f3 oauth 状态码细分+UA、1c71fec 连接检测(ping 重试/分类，剔除 Baidu)、c7a424c 189Cloud 认证子集(auth_response.go 重构+token 先落库+会话错误降级)、353b830 上传目录错位。适配点：RecoverAccount 保持 void 签名、115 保留 600s 超时、Init 移除内联 Ping、驱动嵌入 AuthRefreshControl(guard nil 行为不变)、未移植 internal/auth 守卫接线与 de83b46 批次化。vet/test(除存量 internal/file 失败)/type-check/build 全绿，0.0.13 镜像三 tag 同 digest 67d3b57f，tag+release+容器重建 health ok。

### Git Commits

| Hash | Message |
|------|---------|
| `221c340` | fix: adapt upstream auth/account/upload fixes, bump to 0.0.13 |

### Testing

- [OK] GOWORK=off go vet ./... 全绿；go test httpx/account/driver/115_Open/189Cloud 全绿；web type-check+build 通过；存量失败 internal/file name_align_test 在 clean HEAD 复现(与本任务无关)；容器 /api/health ok

### Status

[OK] **Completed**

### Next Steps

- 存量 internal/file name_align 中文集号解析失败可另建任务修复；如需 189Cloud 实测认证刷新可后续验证


## Session 69: 修复 189Cloud HTTP 400 认证判定回归并发布 0.0.14
<!-- trellis-session: v=2 fp=f5952dcbffacb375 -->

**Date**: 2026-09-09
**Task**: 修复 189Cloud HTTP 400 认证判定回归并发布 0.0.14
**Package**: backend
**Branch**: `main`

### Summary

0.0.13 验收失败：天翼云盘报 DRIVER_ERROR HTTP 400 UserInvalidOpenToken/unifyAccountInfo is null 且不恢复(调度排7440分钟后)。根因是移植 c7a424c 时照搬收窄判定(401||200+payload)，丢掉 0.0.12 的'400+失效payload→CodeAuthExpired→WithRetry被动刷新'链路。修复 rawJSON/rawForm 恢复 400 分支(payload 精确匹配不影响其它400)，taskauth 异步事件 ctx 加固，新增7例回归单测全绿。0.0.14 三 tag digest 3b2ae4fb，tag+release+容器重建 health ok。API 实测受阻(admin密码已改)，待用户 UI 复测；DB 只读核对认证态 active/无错误记录印证态机无感知。

### Git Commits

| Hash | Message |
|------|---------|
| `add540a` | fix: restore 189Cloud auth-expired classification on HTTP 400, bump to 0.0.14 |

### Testing

- [OK] go vet 全绿；drivers/189Cloud 7 例回归单测全绿；全量测试仅存量 internal/file 失败(已备案)

### Status

[OK] **Completed**

### Next Steps

- 用户 UI 复测天翼云盘列表；可选清理 data/litepan.db 中已删功能残留表(strm_tasks/offline_download_tasks 等)


## Session 70: 记录 admin/123456 并完成 0.0.14 天翼云盘实测验收
<!-- trellis-session: v=2 fp=c1afaaee0e7573d0 -->

**Date**: 2026-09-09
**Task**: 记录 admin/123456 并完成 0.0.14 天翼云盘实测验收
**Package**: backend
**Branch**: `main`

### Summary

DB 证实 08-30 后本实例从未改密（用户改密发生在别处）；经授权用 pkg/security.HashPassword 生成哈希重置为 123456（重置前全库备份 /tmp/litepan-backup-20260909-201827.db）。破案：/api/auth/login 仅接受 form 表单编码，JSON 请求被解析为空用户名（日志 username=""）——此前所有登录失败均源于此。实测：登录成功→files/list 返回真实目录→日志还原恢复链（20:16:30 用户UI请求触发被动刷新+凭据回写→20:19:14 列表直接成功），0.0.14 修复闭环确认。AGENTS.md 更新凭据+form编码注意事项，清理笔误产物 data/litean.db 与 .tmp-hashgen。

### Git Commits

(No commits - planning session)

### Testing

- [OK] form 编码登录 200；files/list success:true 含真实目录；account_auth_states active/last_refresh 20:16:30

### Status

[OK] **Completed**

### Next Steps

- 无遗留；可选：清理 DB 已删功能残留表


## Session 71: 清理孤儿表与残留键并发布 0.0.15
<!-- trellis-session: v=2 fp=69b165e7b48b1879 -->

**Date**: 2026-09-09
**Task**: 清理孤儿表与残留键并发布 0.0.15
**Package**: backend
**Branch**: `main`

### Summary

逐表对照代码引用分类：strm×3/offline_download 纯孤儿，media_organize/cache_retention 仅 backup.go 引用，quarktv_bindings 为死代码(repo+domain+装配)。notifications/api_keys/fuse_mounts 复核为活功能保留(用户原以为 notifications 是残留,实为站内通知)。实施:迁移0022(幂等DROP×7+删configs strm_base_url/strm_token)+backup.go 清引用+删 quarktv 死代码。0.0.15 三 tag digest da66cbe8,tag+release,部署前备份 data/backups/manual-pre-0022-20260909-202635.db,启动自动迁移成功(台账22),表17→10、键10→8,health/login/files-list 三连通过。

### Git Commits

| Hash | Message |
|------|---------|
| `9bb17de` | chore: drop removed-feature orphan tables via migration 0022, bump to 0.0.15 |

### Testing

- [OK] go vet 全绿;go test 仅存量 internal/file 失败(备案);迁移0022 幂等执行;健康/登录/列表实测通过

### Status

[OK] **Completed**

### Next Steps

- 无遗留


## Session 72: 本轮维护总结报告归档
<!-- trellis-session: v=2 fp=92f5a2c2ee4214d7 -->

**Date**: 2026-09-09
**Task**: 本轮维护总结报告归档
**Package**: backend
**Branch**: `main`

### Summary

产出 report.md 总结 0.0.12→0.0.15 全程：5 任务、3 发布(0.0.13 适配上移植/0.0.14 189Cloud 回归修复/0.0.15 孤儿表清理)、上游合并决策C(不整体merge小步移植)、回归教训(收窄型判定须连同兜底机制一起评估,400vs401差异用回归单测锁死)、运维知识(form编码登录/admin123456落档)、DB卫生(17→10表)。遗留:internal/file存量2例测试失败、上游守卫接线未移植。

### Git Commits

| Hash | Message |
|------|---------|
| `e2c492a` | chore: record journal |

### Status

[OK] **Completed**

### Next Steps

- 无


## Session 73: 修复命名对齐中文集号解析并发布 0.0.16
<!-- trellis-session: v=2 fp=bda3f41ebafbdec4 -->

**Date**: 2026-09-09
**Task**: 修复命名对齐中文集号解析并发布 0.0.16
**Package**: backend
**Branch**: `main`

### Summary

internal/file 2 例存量失败根因：(1) 精简时 guessit 引擎被换成简化 parseEpisodeNumber，组合中文数字(二十八)完全不支持(Trim+Atoi 把整串删空)；(2) 兜底'最后一个数字'没去扩展名，.mp4 的 4 被当集号(即测试看到的 episode=4)。修复:标准进位中文数字解析(十二/二十八/一百零五,无单位按位拼接)+兜底用 stem。新增组合数字与扩展名忽略 2 组单测。全模块 go test 首次全绿。0.0.16 三 tag digest f91294c2,tag+release,部署三连验证通过。

### Git Commits

| Hash | Message |
|------|---------|
| `c996c21` | fix: Chinese numeral episode parsing in name align, bump to 0.0.16 |

### Testing

- [OK] internal/file 全部 8 测试通过;go vet 全绿;go test ./... 全模块零失败(首次)

### Status

[OK] **Completed**

### Next Steps

- 无遗留


## Session 74: 调查上游未移植项：守卫接线建议移植、批次化挂起
<!-- trellis-session: v=2 fp=dda8f1de9da3d0a4 -->

**Date**: 2026-09-09
**Task**: 调查上游未移植项：守卫接线建议移植、批次化挂起
**Package**: backend
**Branch**: `main`

### Summary

守卫接线(c7a424c 单提交,本方 internal/auth 零分叉):control.go 新增 130 行,内联刷新 sync.Once 去重+复用窗口+冷却+失败入状态机,initializeDriver 防冷启动风暴,driverexec 网络熔断(3次失败退避30s);移植面=auth 9 文件整取+driverexec+4测试文件(~370行)+仅 1 处手改(RecoverAccount 还原 error 签名);wire_http 无需改(自装配);manager 预埋挂载点将首次激活。结论 B 建议移植。批次化 de83b46:真实源码 30 文件,前端 1088 行与本方 705 行删减正面相撞,纯 UX 无正确性修复(目录错位已移植),结论 C 挂起,若做需独立任务三步走。

### Git Commits

(No commits - planning session)

### Testing

- [OK] 只读取证:git log/diff/grep 对照,工作区干净

### Status

[OK] **Completed**

### Next Steps

- 待用户决定是否移植守卫接线(建议下一轮)


## Session 75: 接入上游认证守卫并发布 0.0.17
<!-- trellis-session: v=2 fp=8f8fc353d800b269 -->

**Date**: 2026-09-09
**Task**: 接入上游认证守卫并发布 0.0.17
**Package**: backend
**Branch**: `main`

### Summary

整取 upstream internal/auth(control.go+9文件+4测试文件,本方零分叉)与 driverexec 网络熔断,激活 0.0.13 预埋 SetAuthGuards。内联刷新统一收口:sync.Once 请求内去重/recentlyRefreshed 复用窗口/冷却封锁/失败入状态机;initializeDriver 账号锁防冷启动风暴;RecoverAccount 回归 error 签名。两处适配:oauth_integration_test 驱动矩阵 123/Baidu/OneDrive→template(已删驱动替代);template 骨架对齐统一代理契约(postOAuthJSON→OAuthProxyHTTPError,classifyRefreshError→ClassifyOAuthRefreshError,上游疏漏修正)。auth 包 31 测试全绿,全模块零失败。0.0.17 三 tag digest 7f691b58,部署三连+启动日志正常。

### Git Commits

| Hash | Message |
|------|---------|
| `68f4c8e` | feat: wire auth refresh guards from upstream c7a424c, bump to 0.0.17 |

### Testing

- [OK] go vet 全绿;go test ./... 零失败;auth 31 测试通过;health/登录/files-list 三连通过;启动日志无错误

### Status

[OK] **Completed**

### Next Steps

- 上游未移植项仅剩上传批次化(P3,挂起待真实需求)


## Session 76: 移植上传批次化并发布 0.0.18
<!-- trellis-session: v=2 fp=900a7914eb079b29 -->

**Date**: 2026-09-09
**Task**: 移植上传批次化并发布 0.0.18
**Package**: backend
**Branch**: `main`

### Summary

按调查报告三步走移植 de83b46：迁移0023(batch_id/batch_name+索引,上游0022编号冲突用本方下一号)；后端 types/manager/delete/lifecycle/sse/persist/queue/state/worker+domain+repo 批次化,manager 2处冲突调和(去 runningDownloads/completedOfflineGroups 保批次广播集),worker 丢弃跨盘/离线函数块,manager_test 清孤儿 import(net/http/httptest/playback);前端 uploadTaskTree 新文件+9纯替换+TaskPanel 7冲突逐块调和(去 relay/offline 分支保 upload 树)+activeRelayCount 按 upload-only 改写。批次单测全绿,全模块零失败,type-check/build 通过。0.0.18 三 tag digest 05ac313f,迁移23落库(upload_tasks 33/34列),部署三连通过。

### Git Commits

| Hash | Message |
|------|---------|
| `6f375f5` | feat: port upload task batching from upstream de83b46, bump to 0.0.18 |

### Testing

- [OK] go vet 全绿;go test ./... 零失败;web type-check+build 通过;迁移0023 执行;health/登录/列表三连

### Status

[OK] **Completed**

### Next Steps

- 用户 UI 实测文件夹上传批次体验;上游未移植项全部清零


## Session 77: 调查大批量上传卡死：吞吐塌陷非死锁
<!-- trellis-session: v=2 fp=2941f1efeee77cbf -->

**Date**: 2026-09-09
**Task**: 调查大批量上传卡死：吞吐塌陷非死锁
**Package**: backend
**Branch**: `main`

### Summary

791 文件批次(xwechat_files,40MB)5分钟仅完成46个(6%)→观感卡死。根因:①upload_task_concurrency=1(configs实测,单并发);②189账号级500ms操作间隔门持锁串行(全账号2req/s天花板,数据面PUT已豁免);③目录解析缓存TTL仅30s,数百微信哈希目录长跑反复重List。量化模型2500-3000门限操作≈20-25分钟起步,吞吐曲线(1/8/27/10每分钟)吻合。排除:SSE托底drop-1不死锁(上游同款无后续修复)、后端零错误、暂停时仍在推进。次要:pause不写updated_at(745行0值)。修复建议:并发2-4/TTL10min/批次预解析/pause补时间戳,待用户确认后实施。

### Git Commits

(No commits - planning session)

### Testing

- [OK] upload_tasks 表逐项核对+configs 实测+源码路径核实(delay/transport/target_dir/sse/upload),全程只读

### Status

[OK] **Completed**

### Next Steps

- 待用户确认并发策略后建修复任务


## Session 78: 修复大批量上传目录解析吞吐并发布 0.0.19
<!-- trellis-session: v=2 fp=cab802bbc0c99e85 -->

**Date**: 2026-09-09
**Task**: 修复大批量上传目录解析吞吐并发布 0.0.19
**Package**: backend
**Branch**: `main`

### Summary

承接调查报告实施(并发=1 用户设置保留,500ms 门保留)：①target_dir 缓存 TTL 30s→10min(消长跑重复 List)②批次目录预解析(CreateBatch 后台去重+字典序预热唯一 rel_dir,父前缀先行,防重复预热,失败 worker 兜底),上传命中缓存不再边传边解析。另修正上轮调查误报:pause 未写 updated_at 系 datetime 缺 unixepoch 查询笔误,原始值核对 791 行时间戳全部正常,该项撤销。新增单测 2 组(预热去重/排序/缓存吸收、批次收集分组),全模块 go test 零失败。0.0.19 三 tag digest 1fa84073,部署三连通过。

### Git Commits

| Hash | Message |
|------|---------|
| `cd5720c` | perf: speed up bulk upload dir resolution, bump to 0.0.19 |

### Testing

- [OK] go vet 全绿;go test ./... 零失败;health/登录/列表三连

### Status

[OK] **Completed**

### Next Steps

- 用户复测大批量上传(745 个 paused 任务可续传观察吞吐变化)
