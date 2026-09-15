# Design: 彻底移除 FUSE

## 目标与不变量

**目标**：把 FUSE（挂载服务 / 读缓存 / FUSE 文件系统实现 / 前端 / 数据表 / 构建与部署面）删净。

**不变量（每一步都必须成立）**

1. **每一步之后 `go build ./...` 都能过**（或至少停在"已明确知道还差哪一处"的可解释状态）。
2. **本机 `:5211` 实例不受影响**：不动容器、不动数据、不发版。
3. **不误删近名物**：`internal/upload` 的 `TempRegistry`、`playback` 的 HTTP 链路、`drivers/LocalFs`、历史迁移 `0008` 全部保留。
4. **迁移不可逆**：人工只新增迁移，不改历史文件；本轮不执行迁移（执行发生在下次发布部署时）。

## 关键决策

### KD1 删除顺序：叶子优先，与创建顺序相反

依赖方向是单向的（`api/app → fusemount → {fusereadcache, share/fuse}`，`store` 独立）。因此**先摘上层引用、再删下层包**：

```
① 前端（独立于 Go）
② HTTP 层（api 两个 handler + router + local_fs + slow_dashboard_log）
③ 装配层（app.go / wire_services / account_lifecycle / wire_http，删 wire_fuse_read_cache.go）
④ 存储层（store.go + backup.go 两处 + 删 fuse_mount_repo.go）
⑤ 删三个包（fusemount / fusereadcache / share 整棵）
⑥ 领域/设置/通知/playback 收尾（此时才删 domain/fuse_mount.go —— 它被 ⑤ 的包引用）
⑦ 迁移 0025
⑧ 构建与部署面（Dockerfile / Makefile / compose×2 / README / go mod tidy）
⑨ embed 重建 + 全量质量门 + 基线复核
⑩ spec 同步
⑪ 非特权临时容器验证
```

**为什么 ⑥ 必须在 ⑤ 之后**：`internal/domain/fuse_mount.go` 的类型被 `fusemount`/`share/fuse`/`store` 使用 → 先删它必然编译失败。这正是上轮 API 秘钥任务（先摘接线后删包）的同一教训。

### KD2 历史迁移不动，新增 0025 做三件事

`internal/store/migrations/0008_fuse_mounts.sql` 保留（追加式约定：已部署实例的 `schema_migrations` 台账不能缺号）。
新增 `0025_drop_fuse.sql` **幂等**三连：

```sql
DROP TABLE IF EXISTS fuse_mounts;
DELETE FROM configs WHERE key IN ('fuse_enabled','fuse_mount_root','fuse_read_cache_enabled',
  'fuse_read_cache_max_gb','fuse_read_cache_retention_days','fuse_read_cache_eviction_policy');
DELETE FROM notifications WHERE category='fuse_mount_warn';   -- 防止旧库留下渲染不出类别的通知
```

第三句的必要性：`NotificationCategoryFuseMountWarn` 常量随代码删除后，旧库里若有该类通知会变成"未知类别"。本机 0 行，但迁移要对所有实例成立。

### KD3 **最高风险点**：`BackupCounts` 的 SQL 与它的回归测试

`internal/store/backup.go`：

```go
tables := []string{ "fuse_mounts", "automation_rules" }   // SELECT COUNT(1) FROM <table>
```

表被删后这一句会让**创建备份**直接 `no such table: fuse_mounts`（调用链 `backuprestore/service.go:213 → store.BackupCounts`）。这是本轮唯一"删了会 runtime 报错"的地方，因此：

- 删 `"fuse_mounts"` 后，**主动补一个回归测试**（`internal/store/backup_test.go`）：迁移后的库上调用 `BackupCounts` 必须成功。
- 并**反向验证一次**：临时把 `"fuse_mounts"` 加回去、跑该测试，确认它会失败 —— 证明这个测试真能抓住这类漏改（不是"摆设测试"）。

### KD4 `fuseReaderUA` 只摘兜底，不动 `RemoteReader` 圈

`internal/playback/remote_reader.go` 的 `OpenRemoteReader` 唯一调用者是 `share/fuse/nodes.go:393`。删 FUSE 后这一坨（`RemoteReader`/`remoteWindowReader`/`local_reader` + 测试）会成为死代码，但**本轮不删**：

- 它需要**独立的死代码复核**（`deadcode` + `unused` 并用，确认没有其它间接入口）；
- 删除它会让本轮的 diff 与风险面翻倍（尤其 `remote_reader_test.go` 有 12 个测试）。

→ 本轮只删 `fuseReaderUA` 常量与 `ua == ""` 的兜底分支，保持编译与 HTTP 播放链路不变。

### KD5 部署面收窄：删 `privileged` 的依据是"唯一消费者已消失"

| 项 | 删的依据（实测）|
|---|---|
| `privileged: true` | 全仓唯一需要 `CAP_SYS_ADMIN` 的是 FUSE 挂载 |
| `pid: "host"` | 全仓唯一 `/proc` 读取是 `fusemount/mountpoint_linux.go` 的 `/proc/self/mountinfo`（自身命名空间）|
| `devices: /dev/fuse` | `fusermount3` 唯一调用点同上 |
| `./mounts:/app/mounts:shared` | 挂载根；LocalFs 驱动 `DefaultRoot:"/"` 自选目录，不依赖它 |

**但"能删"不等于"删完一定跑得起来"** → KD6 用非特权临时容器实测，而不是靠推断。

### KD6 验证策略：临时实例（非特权）打穿关键路径

本机实例不能动（还在跑 v0.0.45）。改为：

1. `go build -tags "" ./cmd/litepan`（**注意不再需要 `fuse` 标签**）
2. 起临时容器：**去掉 privileged / pid / devices / mounts 四个设置**，端口 `127.0.0.1:5311`，数据用 `data/` 副本
3. 打穿：health → 登录（表单编码）→ **创建备份**（`BackupCounts` 的真实调用路径）→ 文件列表 / 通知 / 自动化规则 / 上传配置 / 设置读写
4. 这一步同时也是"去掉 `-tags fuse` 后程序仍完整可用"的证明

### KD7 前端：删卡片而不是"隐藏卡片"

仪表盘那张「FUSE 挂载点 0/0」卡片直接删（连同 `loadOverview` 的请求）。风险评估：卡片在一个 grid 里，少一张不破坏布局 —— 用 `bw` 在临时实例上**目视确认**仪表盘布局不塌、无空白缺口。

## 风险与处置

| 风险 | 处置 |
|---|---|
| 漏改 `BackupCounts` → 备份报错 | KD3：先删 + 补回归测试 + 反向验证测试有效性 |
| 删包顺序错导致中途大面积编译失败 | KD1 固定顺序，每步 `go build ./...` |
| 误删近名物（`TempRegistry`/HTTP 播放/LocalFs） | 不变量 4 + 验收项的"越界检查"逐条核对 |
| 去掉 `privileged` 后容器起不来 | KD6 实测；起不来就保留 `privileged`（收益让位于可用性），并记录原因 |
| `go mod tidy` 连带升级别的依赖 | 只允许删除 `go-fuse` 与其专属传递依赖；`go.mod` 其它版本号变化即为越界，需人工确认 |
| spec 里顺手改出新的错误 | 只改与 FUSE 相关的行，并把同句已知过期内容改准（有实测依据）|

## Rollout / Rollback

**Rollout**：纯代码/配置改动，按 KD1 顺序推进；质量门全绿后提交推送。**不发版、不部署**。

**Rollback**：

| 项 | 回滚 |
|---|---|
| 全部代码/配置 | `git revert <commit>`（单提交即可还原）|
| 数据表 | 本轮**不执行**迁移 → 实例库仍有 `fuse_mounts` 空表；即便将来执行过，`DROP` 的也是一张 0 行表，恢复旧版即重新建表（`0008` 仍在迁移链里）|
| 前端 embed | `git checkout HEAD -- internal/api/web` 后重新 `npm run build` |
| 临时验证实例 | 删容器与临时目录 |

**触发回滚的硬条件**：任一步 `go build` 失败且无法在 1 处内解释、质量门不绿、`BackupCounts` 回归测试失败、非特权实例关键路径失败、发现误删近名物。

## File Map

| 路径 | 动作 |
|---|---|
| `internal/fusemount/**`（7）、`internal/fusereadcache/**`（7）、`internal/share/**`（11） | **删除整包** |
| `internal/api/fuse_admin.go`、`fuse_read_cache_admin.go`、`internal/app/wire_fuse_read_cache.go`、`internal/domain/fuse_mount.go`、`internal/store/fuse_mount_repo.go`、`web/src/api/fuse.ts` | **整文件删除** |
| `internal/api/{router,local_fs,slow_dashboard_log}.go`(+测试)、`internal/app/{app,wire_services,account_lifecycle,wire_http}.go`、`internal/store/{store,backup}.go`、`internal/settings/registry.go`、`internal/domain/notification.go`、`internal/playback/remote_reader.go`、`internal/upload/maintenance.go`(注释) | 摘除接线 |
| `web/src/components/admin/DashboardManagement.vue`、`web/src/components/common/LocalDirBrowserModal.vue` | 删卡片/快速路径 |
| `internal/store/migrations/0025_drop_fuse.sql` | **新增**（幂等）|
| `internal/store/backup_test.go` | **新增回归测试**（`BackupCounts`）|
| `Dockerfile`、`Makefile`、`docker-compose.yml`、`docker-compose.fnos.yml`、`README.md` | 去 FUSE 与特权设置 |
| `go.mod` / `go.sum` | `go mod tidy` |
| `internal/api/web/**` | 重建 embed |
| `.trellis/spec/**`（15 处） | 同步 |

**零改动（越界即失败）**：`internal/store/migrations/` 下全部**历史**迁移（`0001`–`0024`，含 `0008_fuse_mounts.sql`）、`internal/upload/**`（除注释）、`drivers/**`、`internal/playback` 的 HTTP 链路与 `RemoteReader`/`remoteWindowReader`/`local_reader` 的类型部分、`.golangci.yml`、`internal/buildinfo/version.go`。
