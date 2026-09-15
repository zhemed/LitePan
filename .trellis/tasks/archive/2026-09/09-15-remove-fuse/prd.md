# 彻底移除 FUSE 挂载点相关全部内容

## Goal

按调查结论（`.trellis/tasks/archive/2026-09/09-15-investigate-fuse-removal/research.md`）把 FUSE **从代码、数据表、前端、构建与部署面四层彻底删除**：

- 功能面：后台「本地挂载」——把网盘账号挂成本地目录（`internal/fusemount`）、FUSE 读缓存（`internal/fusereadcache`）、FUSE 文件系统实现（`internal/share/fuse`，`internal/share/` 仅剩它）
- 交付判据：删除后 `internal/share/` 整棵消失、`fuse_mounts` 表不存在、前端不再有「FUSE 挂载点」卡片、**容器可以去掉 `privileged: true` + `pid: "host"` + `/dev/fuse` 而正常运行**

依据（实测）：该功能**只有管理侧、没有消费侧** —— 前端 11 个 API 函数只有 `fetchFuseMounts` 被调用（仪表盘只读卡片），**创建/挂载入口根本不存在**；实例上 `fuse_mounts` 0 行、`enabled:false`、读缓存 0 块、宿主与容器内无任何 FUSE 挂载、全量日志零 FUSE 活动。

## Requirements

### D1 前端删除（先做，独立于 Go 编译）

- 删 `web/src/api/fuse.ts`（85 行）
- `web/src/components/admin/DashboardManagement.vue`：删 import、`fuseMounts` ref、`mountedFuseCount`/`totalFuseCount`、`loadOverview` 里的请求、**「FUSE 挂载点 0/0」卡片**、`firstLoad` 条件中的 `fuseMounts.value.length`
- `web/src/components/common/LocalDirBrowserModal.vue`：快速路径去掉 `/app/mounts`

### D2 HTTP 层删除

- 删 `internal/api/fuse_admin.go`（267 行，9 个 handler）、`internal/api/fuse_read_cache_admin.go`（82 行，3 个 handler）
- `internal/api/router.go`：删 import、`Deps.Fuse`、`Handler.fuse`、`NewRouter` 赋值、**`/api/admin/fuse` 下 11 条路由**（`Route("/fuse", …)` 整块）
- `internal/api/local_fs.go`：删 `add(fusemount.MountRoot)` 与相关 import、注释
- `internal/api/slow_dashboard_log.go` + 其测试：慢请求白名单去掉 `"/api/admin/fuse/mounts"`

### D3 装配层删除

- `internal/app/wire_services.go`：删 `fuse`/`fuseReadCache` 字段与构造、`ApplyConfiguredMountRoot`、`Register`、`PrepareMountRoot`、`SetUploads`、`SetStartupGate`、传给 api/app 的字段
- `internal/app/app.go`：删 `fuse` 字段、`Start` 调用、`Shutdown` 中的 `Stop` 与 `shutdownFuseBudget`（12s）
- `internal/app/account_lifecycle.go`：删 `fuse`/`readCache` 字段与 `OnAccountDeleted` 两段级联
- `internal/app/wire_http.go`：删 `Fuse: svc.fuse`
- 删 `internal/app/wire_fuse_read_cache.go`（31 行）

### D4 存储层删除（含**必须同步**的 SQL）

- `internal/store/store.go`：删 `FuseMounts` 字段与 `&fuseMountRepo{}`
- `internal/store/backup.go`：① 删 sanitize 语句 `UPDATE fuse_mounts SET state='unmounted', last_error=''`；② **`BackupCounts` 的 `tables` 列表删 `"fuse_mounts"`**（漏此处 → `internal/backuprestore/service.go:213` 创建备份时直接报 `no such table`）
- 删 `internal/store/fuse_mount_repo.go`（111 行）

### D5 删包（最后）

- 删 `internal/fusemount/`（7 文件 1,073 行）、`internal/fusereadcache/`（7 文件 778 行）、`internal/share/`（整棵，11 文件 1,248 行）

### D6 领域 / 设置 / 通知 / playback 收尾

- 删 `internal/domain/fuse_mount.go`（44 行）
- `internal/settings/registry.go`：删 4 个键常量（`KeyFuseReadCache*`）与 4 条 spec（description 里还写着"需在「文件共享 → 本地挂载」页配置"——那个页面早已不存在）
- `internal/domain/notification.go`：删 `NotificationCategoryFuseMountWarn`
- `internal/playback/remote_reader.go`：删 `fuseReaderUA` 常量与 `ua == ""` 兜底分支（保留其它逻辑）
- `internal/upload/maintenance.go`：注释更新（不再提 FUSE 写入）

### D7 数据迁移（新增，不改历史）

- 新增 `internal/store/migrations/0025_drop_fuse.sql`，内容幂等：
  - `DROP TABLE IF EXISTS fuse_mounts;`
  - `DELETE FROM configs WHERE key IN ('fuse_enabled','fuse_mount_root','fuse_read_cache_enabled','fuse_read_cache_max_gb','fuse_read_cache_retention_days','fuse_read_cache_eviction_policy');`
  - `DELETE FROM notifications WHERE category='fuse_mount_warn';`（避免旧库残留未知类别通知）
- **`0008_fuse_mounts.sql` 保留不动**（追加式约定，删历史迁移会破坏已部署实例的升级路径）

### D8 构建与部署面收窄

- `Dockerfile`：删 `ARG BUILD_TAGS=fuse`（`go build` 不再带标签）、`fuse3` 安装与 `/etc/fuse.conf` 改写、`mkdir /app/mounts`、`VOLUME ["/app/mounts"]`
- `Makefile`：`build` 去掉 `-tags fuse`，`build-nofuse` 与之合并
- `docker-compose.yml` / `docker-compose.fnos.yml`：删 `privileged: true`、`pid: "host"`、`devices: /dev/fuse`、`./mounts:/app/mounts:shared`（fnos 对应路径）、读缓存挂载注释
- `README.md`：快速开始同步上述四处
- `go mod tidy`：`github.com/hanwen/go-fuse/v2` 与 `golang.org/x/sys` 直接依赖应消失/降级

### D9 质量门与基线

- `make lint` 0 issues、`go vet` exit=0、`go test ./...` 全包 ok、`cd web && npm run type-check` exit=0、`npm run build`（重建 embed）
- 基线不劣化：`deadcode ./cmd/litepan` **不高于 7**（删除后应减少）、`unused` **0**、前端零引用 **0**
- **新增回归测试**：`internal/store/backup_test.go` 增加对 `BackupCounts` 的用例，锁定"迁移后仍可统计"（这正是本轮最高风险点，若漏改 D4② 该测试必失败）

### D10 非特权运行验证（本机，临时实例）

用**本轮从工作区构建的镜像**起一个临时容器，**不带 `privileged`、不带 `pid: "host"`、不挂 `/dev/fuse`**，数据用实例库副本，验证：

- `/api/health` 200、登录 200、`/api/public/system-config` 正常
- **创建一次备份成功**（打穿 `BackupCounts` 路径）
- 文件列表、通知、自动化规则、上传配置、设置读写各返回 200
- 容器内无 FUSE 相关报错；`docker compose config` 解析后不含 privileged/pid/devices

### D11 spec 同步

- 15 处 FUSE 引用（`directory-structure.md` 目录树/生命周期示例、`database-guidelines.md` 表清单与仓储示例、`quality-guidelines.md` 的 `-tags fuse`、`logging-guidelines.md` 的 `ModuleFuse`、`api-client.md` 的 `fuse.ts`、`index.md` 简介）改为实测现状；**同句中早已过期的 `strm/quarktv/retention/media` 等一并改准**（这些行本就要重写）

## Constraints

- **不做版本 bump、不发版、不更新本机 `:5211` 容器**：本轮只落地代码与配置；发布与否（是否发 `v0.0.46`、是否用非特权 compose 重建容器）由用户另定
- **不动 playback 的 `RemoteReader`/`remoteWindowReader` 圈**：它们是 FUSE 删除后的二级死代码，按计划**另开任务**处置（本轮只把 `fuseReaderUA` 兜底摘掉，保持编译）
- **不动** `internal/store/migrations/0008_fuse_mounts.sql`、`internal/upload/**` 的 `TempRegistry`/`CleanupOrphanTempFiles`、`drivers/LocalFs/**`、HTTP 播放链路（`range_proxy`/`streamer`/`response`/…）、`EXPOSE 42069`（与 FUSE 无关的独立残留，另案）
- **不改** `.golangci.yml`、`internal/buildinfo/version.go`、`README` 与 FUSE 无关的内容
- 删除对象共 **32 个受跟踪文件 / 3,719 行**（含测试 234 行）；不得顺手扩大
- 迁移不可逆（`DROP TABLE`）：实例上是 0 行空表，且部署前会先备份（属发布任务）
- 本机 `danger-full-access`、审批关闭 —— 不请求 escalation
- 不改归档任务与历史 journal

## Acceptance Criteria

- [x] `internal/fusemount/`、`internal/fusereadcache/`、`internal/share/` **三个目录不存在**；`internal/domain/fuse_mount.go`、`internal/store/fuse_mount_repo.go`、`internal/api/fuse_admin.go`、`internal/api/fuse_read_cache_admin.go`、`internal/app/wire_fuse_read_cache.go`、`web/src/api/fuse.ts` 均已删除
- [x] `grep -rn -i "fuse"` 在 `internal/`、`web/src/`、`drivers/`、`Makefile`、`Dockerfile`、`docker-compose*.yml` 中**零命中**（允许保留：历史迁移 `0008`、本任务文档、以及迁移 `0025` 里必须提及的旧表名/键名）
- [x] `internal/store/backup.go` 的 `BackupCounts` 不再引用 `fuse_mounts`，且 sanitize 语句列表少了 `UPDATE fuse_mounts` 那条、**其余语句逐条仍在、顺序不变**
- [x] 新迁移 `0025_drop_fuse.sql` 存在且幂等（`DROP TABLE IF EXISTS` + `DELETE`）；临时库实测：`fuse_mounts` 不存在、`fuse_*` 配置键清空、重复执行不报错；历史迁移 **零改动**
- [x] **新增的 `BackupCounts` 回归测试通过**，且"故意漏改 D4② 会失败"已被论证（可用临时回退验证一次）
- [x] 前端：仪表盘不再有「FUSE 挂载点」卡片、`LocalDirBrowserModal` 无 `/app/mounts`、`web/src/api/fuse.ts` 已删、`vue-tsc -b` exit=0
- [x] `internal/playback` 只删了 `fuseReaderUA` 与其兜底分支；`RemoteReader`/`remoteWindowReader`/`local_reader` **未被改动**（留给后续任务）
- [x] 质量门全绿：`make lint` 0 issues、`go vet` exit=0、`go test ./...` 全包 ok、`vue-tsc -b` exit=0、`npm run build` 成功且 `internal/api/web/**` 有 churn
- [x] 基线不劣化：`deadcode ./cmd/litepan` ≤ 7（且 FUSE 相关 0）、`unused` = 0、前端零引用 = 0、`gofmt` 无新增脏文件
- [x] `go.mod` 不再直接依赖 `github.com/hanwen/go-fuse/v2`；`go mod tidy` 后 `go build ./...` 通过
- [x] **非特权验证通过**：临时容器（本轮构建的镜像、去掉 privileged/pid/devices/mounts 的 compose 设置、数据副本）health 200、登录 200、**创建备份成功**、文件列表/通知/自动化/上传配置/设置读写均 200
- [x] `docker compose config` 输出**不含** `privileged`、`pid: host`、`/dev/fuse`、`/app/mounts`
- [x] 越界检查：`internal/upload/**`（除 `maintenance.go` 注释）、`drivers/**`、`.golangci.yml`、`internal/buildinfo/version.go`、playback 的 `RemoteReader` 圈 **零改动**
- [x] spec 同步完成：**无任何"活跃"FUSE 引用**（实测 `grep -rn -i "fuse" .trellis/spec/` 仅剩 5 处**说明"已于 2026-09-15 移除"的历史注记** —— `database-guidelines.md`×2、`index.md`、`quality-guidelines.md`、`api-client.md`；原判据写的是"零命中"，但要把"已移除"这件事记进规范就必然出现该词，故按"活跃引用为零"验收）
- [x] 本机 `:5211` 容器**未被改动**（仍 v0.0.45 运行中），未发版、未 bump 版本
- [x] 任务归档、journal 记录、`main` 与 `origin/main` 同步

## Notes

- Scope 标 `cross-layer`（前端 → HTTP → 装配 → 存储 → 领域/设置 → 包 → 迁移 → 构建部署 → spec，九层贯穿），按复杂任务补 `design.md` + `implement.md`。
- 与上一轮（API 秘钥）同类：**功能级死代码**（管理侧存在、消费侧缺席），`deadcode`/`unused` 均查不出（实测 FUSE 相关 0）。差别在于本轮还**收窄了部署权限面**。
- 后续任务（不在本轮）：① playback `RemoteReader`/`remoteWindowReader`/`local_reader` 二级死代码处置；② `EXPOSE 42069` 僵尸声明与 `ActiveTempPaths` 零调用者的清理；③ 是否发版并把本机容器改用非特权 compose。

## 明确不做

- **不重建 `mounts/` 目录逻辑**：删除后 `/app/mounts` 只是普通目录，LocalFs 用户仍可自选任意目录。
- **不写新的本地挂载替代方案**：用户未提出需求，且该功能本来就无消费入口。
