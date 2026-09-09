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
