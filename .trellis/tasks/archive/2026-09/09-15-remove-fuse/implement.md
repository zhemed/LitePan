# Implementation Plan: 彻底移除 FUSE

## Overview

按 design **KD1 的固定顺序**（叶子优先）执行，每步后跑 `GOWORK=off go build ./...`，保证任一步失败都停在**可解释**的状态。不可逆动作只有"删文件"，全部在质量门通过前完成；迁移文件**写好但不执行**。

本机 `danger-full-access`、审批关闭 —— 不请求 escalation。

---

## Phase 0: 基线快照

- [ ] 0.1 `git status --short`（期望仅本任务目录）、`git log --oneline -1`（期望 `dbb0d59`）
- [ ] 0.2 死代码基线：`deadcode ./cmd/litepan` = **7**、`golangci-lint --enable=unused` = **0 issues**
- [ ] 0.3 前端基线：零引用文件 = **0**、`grep -rn "api/fuse" web/src` = 1 处（`DashboardManagement.vue`）
- [ ] 0.4 锁定近名物（改动后复核）：`internal/upload/{temp.go,manager.go,maintenance.go}` 的 `TempRegistry`/`CleanupOrphanTempFiles`、`internal/playback/{range_proxy,streamer,response,seeker,headers,transport}.go`、`drivers/LocalFs/*`、`internal/store/migrations/0008_fuse_mounts.sql`（记 `sha256`）
- [ ] 0.5 记录 32 个待删文件的行数合计（**3,719**）与 `internal/share/` 仅含 `fuse/` 的事实

**回滚点 R0**：以上即后续比对依据

---

## Phase 1: D1 前端删除（独立）

- [ ] 1.1 `git rm web/src/api/fuse.ts`
- [ ] 1.2 `DashboardManagement.vue`：删 import、`fuseMounts` ref、`mountedFuseCount`/`totalFuseCount`、`loadOverview` 中 `fetchFuseMounts()` 请求与 `assignSettled(results[1], …)`、**卡片 `<article>`**、`firstLoad` 里的 `fuseMounts.value.length`
- [ ] 1.3 `LocalDirBrowserModal.vue`：`quickPaths` 去掉 `"/app/mounts"`
- [ ] 1.4 **验证门 G1**：`cd web && npm run type-check` exit=0（类型系统会抓出残留引用）；`grep -rn "fuse" web/src` 零命中
- **回滚点 R1**：`git checkout HEAD -- web/src`

---

## Phase 2: D2 HTTP 层摘除

- [ ] 2.1 `git rm internal/api/fuse_admin.go internal/api/fuse_read_cache_admin.go`
- [ ] 2.2 `router.go`：删 import、`Deps.Fuse`、`Handler.fuse`、赋值、`Route("/fuse", …)` 整块（11 条路由）
- [ ] 2.3 `local_fs.go`：删 `fusemount` import、`add(fusemount.MountRoot)`、注释里的"FUSE 挂载根"表述
- [ ] 2.4 `slow_dashboard_log.go` + `slow_dashboard_log_test.go`：白名单去掉 `"/api/admin/fuse/mounts"`
- [ ] 2.5 **验证门 G2**：`go build ./...` 通过（预期此处会暴露 app 层还在传 `svc.fuse`）
- **回滚点 R2**：`git checkout HEAD -- internal/api`

---

## Phase 3: D3 装配层摘除

- [ ] 3.1 `git rm internal/app/wire_fuse_read_cache.go`
- [ ] 3.2 `wire_services.go`：删 `fuse`/`fuseReadCache` 字段与构造、`wireFuseReadCacheOrNil` 调用、`ApplyConfiguredMountRoot`、`PrepareMountRoot`、`SetStartupGate`、`Register`、`SetUploads`、返回结构里的两个字段
- [ ] 3.3 `app.go`：删 `fuse` 字段、`Start`、`Shutdown` 里的 `Stop` 与 `shutdownFuseBudget`
- [ ] 3.4 `account_lifecycle.go`：删 `fuse`/`readCache` 字段与两段级联、import
- [ ] 3.5 `wire_http.go`：删 `Fuse: svc.fuse`
- [ ] 3.6 **验证门 G3**：`go build ./...` 通过
- **回滚点 R3**：`git checkout HEAD -- internal/app`

---

## Phase 4: D4 存储层（含**最高风险点**）

- [ ] 4.1 `store.go`：删 `FuseMounts` 字段与 `&fuseMountRepo{db: db}`
- [ ] 4.2 `backup.go`：删 sanitize 的 `UPDATE fuse_mounts …`；**`BackupCounts` 的 `tables` 删 `"fuse_mounts"`**
- [ ] 4.3 `git rm internal/store/fuse_mount_repo.go`
- [ ] 4.4 **新增回归测试** `TestBackupCountsAfterMigrations`（迁移后的库上 `BackupCounts` 必须成功且 `Tasks` 来自现有表）
- [ ] 4.5 **验证门 G4a**：`go test ./internal/store/` 通过
- [ ] 4.6 **验证门 G4b（反向验证测试有效性）**：临时把 `"fuse_mounts"` 加回 `tables` → 该测试**必须失败**（`no such table`）→ 再改回来
- **回滚点 R4**：`git checkout HEAD -- internal/store`

---

## Phase 5: D5 删三个包

- [ ] 5.1 `git rm -r internal/fusemount/ internal/fusereadcache/ internal/share/`
- [ ] 5.2 **验证门 G5**：`go build ./...` 与 `go vet ./...` 通过；`ls internal/ | grep -E "fusemount|fusereadcache|share"` 空
- **回滚点 R5**：`git checkout HEAD -- internal/fusemount internal/fusereadcache internal/share`

---

## Phase 6: D6 领域 / 设置 / 通知 / playback 收尾

- [ ] 6.1 `git rm internal/domain/fuse_mount.go`
- [ ] 6.2 `settings/registry.go`：删 4 个键常量 + 4 条 spec
- [ ] 6.3 `domain/notification.go`：删 `NotificationCategoryFuseMountWarn`
- [ ] 6.4 `playback/remote_reader.go`：删 `fuseReaderUA` 与 `ua == ""` 兜底（其余不动）
- [ ] 6.5 `upload/maintenance.go`：注释改为不含 FUSE 表述
- [ ] 6.6 **验证门 G6**：`go build ./...` + `go vet ./...` 通过
- **回滚点 R6**：按文件 `git checkout HEAD -- …`

---

## Phase 7: D7 迁移

- [ ] 7.1 新增 `internal/store/migrations/0025_drop_fuse.sql`（幂等三连，见 design KD2）
- [ ] 7.2 **验证门 G7a**：临时库（含 `fuse_mounts` + `fuse_*` 键 + 一条 `fuse_mount_warn` 通知）跑迁移 → 表不存在、键清空、通知清空；**重复执行不报错**
- [ ] 7.3 **验证门 G7b**：`git status` 中**没有**任何历史迁移文件的改动（`0008` 零改动）
- **回滚点 R7**：删除新迁移文件

---

## Phase 8: D8 构建与部署面

- [ ] 8.1 `Dockerfile`：去 `ARG BUILD_TAGS=fuse`（`go build` 改为不带标签）、`fuse3`+`fuse.conf`、`mkdir /app/mounts`、`VOLUME /app/mounts`
- [ ] 8.2 `Makefile`：`build` 去 `-tags fuse`；`build-nofuse` 与 `build` 合并（保留一个目标名，避免留下无人引用的别名）
- [ ] 8.3 `docker-compose.yml`：删 `./mounts:/app/mounts:shared`、读缓存注释、`devices`、`pid`、`privileged`
- [ ] 8.4 `docker-compose.fnos.yml`：同上（保留 `network_mode: host`）
- [ ] 8.5 `README.md`：快速开始同步
- [ ] 8.6 `go mod tidy` → **验证门 G8a**：`go.mod` 不再有 `go-fuse`；**除 go-fuse/x-sys 相关外无版本变化**（`git diff go.mod` 逐行确认）
- [ ] 8.7 **验证门 G8b**：`docker compose config` 输出不含 `privileged`/`pid: host`/`/dev/fuse`/`/app/mounts`；`docker compose config` 解析成功
- **回滚点 R8**：`git checkout HEAD -- Dockerfile Makefile docker-compose.yml docker-compose.fnos.yml README.md go.mod go.sum`

---

## Phase 9: D9 重建 embed + 全量质量门

- [ ] 9.1 `cd web && npm run build`（**预期有 churn**：仪表盘卡片去掉后 `DashboardManagement` chunk 变化）
- [ ] 9.2 **验证门 G9a**：`git status --short internal/api/web | wc -l` > 0；构建产物 `zcat assets/*.gz | grep -c "FUSE 挂载点"` = **0**
- [ ] 9.3 **验证门 G9b（全量）**：`make lint` 0 issues、`go vet ./...` exit=0、`go test ./...` 全包 ok、`vue-tsc -b` exit=0
- [ ] 9.4 **验证门 G9c（基线）**：`deadcode ./cmd/litepan` ≤ 7、`unused` = 0、前端零引用 = 0、`gofmt` 无新增脏文件
- [ ] 9.5 **验证门 G9d（残余扫描）**：`internal/`+`web/src/`+`drivers/`+`Makefile`+`Dockerfile`+`docker-compose*.yml` 中 `grep -i fuse` 零命中（历史迁移与迁移 0025 之外）
- **回滚点 R9**：`git checkout HEAD -- internal/api/web` 后重建

---

## Phase 10: D10 非特权临时容器验证

- [ ] 10.1 构建二进制：`go build -o /tmp/fuse-verify/litepan ./cmd/litepan`（**不带标签**）+ 准备数据副本（`data/litepan.db`+wal+shm、`secret.key`）
- [ ] 10.2 起容器：**不带** `privileged` / `pid: host` / `/dev/fuse` / `mounts`，端口 `127.0.0.1:5311`
- [ ] 10.3 **验证门 G10a**：`/api/health` 200；登录 200（表单编码）；`/api/public/system-config` 正常
- [ ] 10.4 **验证门 G10b（关键）**：**创建一次备份成功**（`POST /api/admin/backups`）→ 证明 `BackupCounts` 不再引用被删表
- [ ] 10.5 **验证门 G10c**：文件列表 / 通知 / 自动化规则 / 上传配置 / 设置读写 各 200；日志无 FUSE 报错、无 `ERROR`
- [ ] 10.6 **验证门 G10d（浏览器）**：`bw` 打开 `:5311` 仪表盘 → **无「FUSE 挂载点」卡片**、布局无塌陷；设置页「其他设置」无 FUSE 读缓存项
- [ ] 10.7 清理：停容器、删临时目录与二进制（`data/` 与 `:5211` 全程未动）
- **回滚点 R10**：删容器与临时目录

---

## Phase 11: D11 spec 同步 + 归档

- [ ] 11.1 15 处 FUSE 引用改准（含同句已过期的 `strm/quarktv/retention/media`）
- [ ] 11.2 **验证门 G11**：`grep -rn -i "fuse" .trellis/spec/` 零命中
- [ ] 11.3 `skill trellis-check` 走查（lint/类型/测试、测试覆盖、spec 同步、范围纪律、跨层一致性）
- [ ] 11.4 `flow_gate.py mark-check remove-fuse --note "<质量门+验证摘要>"`
- [ ] 11.5 勾选 `prd.md` 全部验收项
- [ ] 11.6 `flow_gate.py pre-archive` → `task.py archive remove-fuse --skip-branch-validation`（auto-commit 已知会失败 → 手工提交）
- [ ] 11.7 `add_session.py` → `git push`

---

## Validation Commands（汇总）

```bash
export PATH=/usr/local/go/bin:/root/go/bin:$PATH GOWORK=off GOPROXY=https://goproxy.cn,direct
cd /root/LitePan

# 每步后
go build ./... && go vet ./...

# 残余扫描
grep -rn -i "fuse" internal/ web/src/ drivers/ Makefile Dockerfile docker-compose*.yml \
  | grep -v "internal/store/migrations/0008_fuse_mounts.sql" | grep -v "0025_drop_fuse.sql"

# 质量门
make lint && go test ./... && (cd web && npm run type-check && npm run build)

# 基线
deadcode ./cmd/litepan | wc -l ; golangci-lint run --enable=unused | tail -1

# 迁移实测（临时库）
#   用一次性 Go 测试对副本库跑 store.Open+Migrate，重复两次；再用只读 sqlite 断言

# 非特权验证
docker run -d --name fuse-verify -p 127.0.0.1:5311:5211 \
  -v /tmp/fuse-verify/data:/app/data --network litepan_default ghcr.io/…  # 无 privileged/pid/devices
curl -s -o /dev/null -w '%{http_code}\n' http://127.0.0.1:5311/api/health
```

## Review Gates

| 门 | 位置 | 判据 |
|---|---|---|
| G1 | 1.4 | `vue-tsc -b` exit=0 + `web/src` 零 fuse |
| G2–G6 | 2.5–6.6 | 每步 `go build ./...` 通过 |
| G4a/b | 4.5/4.6 | `BackupCounts` 回归测试通过 **且反向验证能失败** |
| G7a/b | 7.2/7.3 | 迁移幂等实测 + 历史迁移零改动 |
| G8a/b | 8.6/8.7 | go.mod 无 go-fuse + compose 无特权项 |
| G9a–d | 9.2–9.5 | 产物无 FUSE 文案 + 全量质量门 + 基线不劣化 + 残余零命中 |
| G10a–d | 10.3–10.6 | 非特权实例：health/登录/备份/各端点 200 + 浏览器无卡片 |
| G11 | 11.2 | spec 零 fuse |

## Rollback

见 design.md → **Rollback**。要点：单提交 `git revert` 即可全部还原；迁移本轮**不执行**；历史迁移 `0008` 全程不动，故旧版行为与升级路径不受影响。

## Out of Scope

- 发版 / bump 版本 / 重建本机 `:5211` 容器
- playback `RemoteReader`/`remoteWindowReader`/`local_reader` 二级死代码处置（另开任务）
- `EXPOSE 42069` 僵尸声明、`upload.ActiveTempPaths` 零调用者（另案）
- 任何"本地挂载替代方案"
