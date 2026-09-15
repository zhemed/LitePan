# FUSE 挂载点相关全部内容能否彻底移除 —— 调查报告

> 调查任务：`09-15-investigate-fuse-removal`（只读，未改任何代码/配置/DB/容器）
> 结论日期：2026-09-15 · 基线：`main @ a4b920e`（v0.0.45）

---

## 0. 结论（TL;DR）

**可以彻底移除。** 代码层、数据表、前端、构建与部署面四类都能删净，没有"删不掉的硬依赖"。

三条决定性证据：

1. **前端早就没有"创建挂载"的入口了**：`web/src/api/fuse.ts` 导出 11 个函数，**只有 `fetchFuseMounts` 被调用（1 处，仪表盘只读卡片），其余 10 个零调用** —— 也就是说「本地挂载」这个功能**只有管理侧、没有消费侧**，和刚删掉的 API 秘钥是同一类问题，而且更彻底（连创建入口都没有）。
2. **实例上从未用过**：`fuse_mounts` 表 0 行、`/api/admin/fuse/status` 返回 `enabled:false`、`/api/admin/fuse/mounts` 返回 `[]`、读缓存 `block_count:0`/`used_bytes:0`、宿主与容器内**没有任何 FUSE 挂载**、全量日志里除我这次探测的 401 外**零 FUSE 活动**、通知表 0 行（含 `fuse_mount_warn` 0 行）。
3. **FUSE 是唯一需要 `privileged: true` + `pid: host` + `/dev/fuse` 的功能**：全仓 `/proc` 读取只有 `fusemount/mountpoint_linux.go` 一处（`/proc/self/mountinfo`），`fusermount3` 只有它调用。删掉后**容器可以退回非特权运行**，这是本轮最大的净收益。

同时有两个**必须一起处理的连带项**（否则删一半会炸，详见 §5）：

- `internal/store/backup.go` 的 `BackupCounts` 会 `SELECT COUNT(1) FROM fuse_mounts` —— 表删了但不改这里，**备份功能会直接报错**；
- `internal/playback` 的 `OpenRemoteReader`/`RemoteReader`/`remoteWindowReader` **唯一调用者就是 FUSE**（`share/fuse/nodes.go:393`）→ FUSE 一删，这一坨播放链路代码（≈1000 行 + 其测试）立刻变成死代码，需要同时决定处置。

---

## 1. Q1 构成盘点（受跟踪文件 32 个，3,719 行）

### 1.1 删除对象（31 个代码文件，3,719 行）

| 层 | 文件 | 行数 |
|---|---|---|
| 挂载服务 | `internal/fusemount/{service.go,validate.go,mountpoint_linux.go,mountpoint_other.go,state.go,service_account.go,constants.go}` | 1,073 |
| 读缓存 | `internal/fusereadcache/{store.go,service.go,settings.go,evict.go,constants.go,coordinator.go,service_test.go}` | 778 |
| FUSE 文件系统实现 | `internal/share/fuse/{nodes.go,write_support.go,manager_fuse.go,statfs.go,backend.go,types.go,compiled_fuse.go,manager_stub.go,compiled_stub.go,write_support_test.go,statfs_test.go}` | 1,248 |
| HTTP 层 | `internal/api/{fuse_admin.go,fuse_read_cache_admin.go}` | 349 |
| 装配 | `internal/app/wire_fuse_read_cache.go` | 31 |
| 领域/存储 | `internal/domain/fuse_mount.go`、`internal/store/fuse_mount_repo.go` | 155 |
| 前端 | `web/src/api/fuse.ts` | 85 |

> `internal/share/` 目录**只剩 `share/fuse/`**（分享功能在 2026-08-30 精简时已删）—— 删掉 FUSE 等于**整棵 `internal/share/` 消失**。

### 1.2 必须修改（不是删除）的位置 —— 共 20 处

| 文件 | 要改什么 |
|---|---|
| `internal/api/router.go` | import、`Deps.Fuse`、`Handler.fuse`、`NewRouter` 赋值、**`/api/admin/fuse` 下的 11 条路由** |
| `internal/api/local_fs.go` | `add(fusemount.MountRoot)`（本地目录浏览器的候选落点）+ 注释 |
| `internal/api/slow_dashboard_log.go` | 慢请求抑制白名单里的 `"/api/admin/fuse/mounts"`（**及其测试 `slow_dashboard_log_test.go:19`**）|
| `internal/app/app.go` | `fuse` 字段、启动 `Start`、关闭 `Stop` + `shutdownFuseBudget`（12s）|
| `internal/app/account_lifecycle.go` | `fuse`/`readCache` 字段与 `OnAccountDeleted` 两段调用 |
| `internal/app/wire_services.go` | `fuseSvc`/`fuseReadCache` 构造、`ApplyConfiguredMountRoot`、`Register`、`PrepareMountRoot`、`SetUploads`、`SetStartupGate` |
| `internal/app/wire_http.go` | `Fuse: svc.fuse` |
| `internal/store/store.go` | bundle 的 `FuseMounts` 字段与 `&fuseMountRepo{}` |
| `internal/store/backup.go` | ① sanitize 语句 `UPDATE fuse_mounts SET state='unmounted'…`；② **`BackupCounts` 的 `tables` 列表里的 `"fuse_mounts"`（必须改，否则备份报错）** |
| `internal/settings/registry.go` | 4 个键常量 + 4 条 spec（`FUSE 读缓存*`，category=performance）|
| `internal/domain/notification.go` | `NotificationCategoryFuseMountWarn` |
| `internal/playback/remote_reader.go` | `fuseReaderUA` 常量与 `ua==""` 兜底分支（连带 §5.2 的整块处置）|
| `internal/upload/maintenance.go` | 注释提到"FUSE 写入"的临时路径（纯注释）|
| `web/src/components/admin/DashboardManagement.vue` | import、`fuseMounts` ref、`mountedFuseCount`/`totalFuseCount`、`loadOverview` 里的请求、**「FUSE 挂载点 0/0」卡片**、`firstLoad` 条件 |
| `web/src/components/common/LocalDirBrowserModal.vue` | 快速路径 `/app/mounts` |
| `Dockerfile` | `ARG BUILD_TAGS=fuse`、`fuse3` 安装、`/etc/fuse.conf` 改写、`mkdir /app/mounts`、`VOLUME ["/app/mounts"]` |
| `Makefile` | `build` 的 `-tags fuse`、`build-nofuse` 目标（合并为一个）|
| `docker-compose.yml` | `./mounts:/app/mounts:shared`、`devices: /dev/fuse`、`pid: "host"`、**`privileged: true`**、读缓存挂载注释 |
| `docker-compose.fnos.yml` | `mounts` 绑定、`/dev/fuse`、`pid: "host"`、`privileged: true` |
| `README.md` | 快速开始里的 `pid`/`privileged`/`/dev/fuse`/`mounts` 四行 |
| `go.mod` / `go.sum` | `github.com/hanwen/go-fuse/v2`（唯一消费者是 `share/fuse`）、`golang.org/x/sys`（唯一直接使用点是 `mountpoint_linux.go`）→ `go mod tidy` |
| **新增** `internal/store/migrations/0025_drop_fuse.sql` | `DROP TABLE IF EXISTS fuse_mounts;` + `DELETE FROM configs WHERE key IN ('fuse_enabled','fuse_mount_root','fuse_read_cache_enabled','fuse_read_cache_max_gb','fuse_read_cache_retention_days','fuse_read_cache_eviction_policy');`（幂等，沿用 0022/0024 风格）|
| `.trellis/spec/**` | 15 处引用（`directory-structure.md` 目录树/生命周期示例、`database-guidelines.md` 表清单与仓储示例、`quality-guidelines.md` 的 `-tags fuse`、`logging-guidelines.md` 的 `ModuleFuse`、`api-client.md` 的 `fuse.ts`、`index.md` 的 FUSE 简介）|

> `internal/store/migrations/0008_fuse_mounts.sql` **保留不动**（追加式迁移约定：历史迁移是记录，删掉会破坏已部署实例的升级路径）。

---

## 2. Q2 使用面核实（运行期证据，全部实测）

| 证据 | 实测结果 | 命令 |
|---|---|---|
| `fuse_mounts` 表行数 | **0** | `python3 -c "sqlite3 … SELECT COUNT(*) FROM fuse_mounts"` |
| FUSE 相关配置键 | **0 个**（configs 里只有 admin_*/public_index_enabled/session_timeout 共 7 键）| 同上 |
| `/api/admin/fuse/status` | `enabled:false`、`compile_support:true`、`mount_root:/app/mounts` | `curl -b <cookie> …/api/admin/fuse/status` |
| `/api/admin/fuse/mounts` | `[]` | 同上 |
| 读缓存 | `enabled:false`、`block_count:0`、`used_bytes:0`、`root_path:/app/data/fuse_read_cache` | `curl …/api/admin/fuse/read-cache` |
| 宿主挂载 | `/proc/mounts` 中**无** litepan 相关 FUSE 条目（只有内核 `fusectl` 伪文件系统）| `grep -i fuse /proc/mounts` |
| 容器内挂载 | `docker exec litepan grep -i fuse /proc/mounts` → 无；`/app/mounts` **空目录** | 同上 |
| 读缓存目录 | `data/fuse_read_cache/`：`blocks/` 空 + `index.sqlite` 4KB（启动时建的空索引）| `ls -la data/fuse_read_cache` |
| 容器日志 | 唯一 FUSE 行是本次调查探测的 401；应用自身**零 FUSE 活动** | `docker compose logs --since 720h \| grep -i fuse` |
| 通知 | `notifications` 表 0 行（`fuse_mount_warn` 0 行）| sqlite 只读查询 |
| 前端可见性 | 仅**仪表盘一张只读卡片**「FUSE 挂载点 0/0」；设置页「其他设置」Tab 实测**不含** FUSE 读缓存设置（category=performance 不在渲染分组内）| `bw open …/admin?page=settings&tab=services` + 代码核对 |

---

## 3. Q3 真实耦合定性（逐处判定）

| 位置 | 判定 | 说明 |
|---|---|---|
| `internal/api/fuse_admin.go` / `fuse_read_cache_admin.go` | **真耦合**（本功能自身）| 11 个端点，均在 `requireAdmin` 组内（实测未登录 401）|
| `internal/store/backup.go:96` `UPDATE fuse_mounts …` | **真耦合** | sanitize 语句，表没了一起删 |
| `internal/store/backup.go:116` `tables{"fuse_mounts"}` | **真耦合（危险）** | `BackupCounts` 会 COUNT 该表 → **不删 SQL 就报错** |
| `internal/api/local_fs.go:44` `add(fusemount.MountRoot)` | **真耦合（弱）** | 只是本地目录浏览器的候选落点，删掉后回退到 dataDir / `/data` / `/mnt` … |
| `internal/api/slow_dashboard_log.go:17` | **真耦合（弱）** | 慢请求白名单，删键即可 |
| `internal/app/{app,wire_services,wire_http,account_lifecycle}.go` | **真耦合** | 生命周期 Start/Stop/账号删除级联/装配 |
| `internal/settings/registry.go` 4 键 | **真耦合** | 键与 spec；UI 不渲染（见 Q2），但 `/api/admin/settings` 会返回 |
| `internal/domain/notification.go` 1 常量 | **真耦合** | 仅 `fusemount` 使用 |
| `internal/playback/remote_reader.go` `fuseReaderUA` | **真耦合（弱）** | `ua==""` 时的默认 UA；唯一调用方（FUSE）删后走不到 |
| `internal/playback/account_range_limiter.go` | **注释引用** | 限流器被 HTTP range proxy 与 FUSE 共用，**不删** |
| `internal/playback/remote_window_reader.go` | **真耦合（连带死代码）** | 见 §5.2 |
| `internal/upload/maintenance.go:5` | **仅注释** | `ActiveTempPaths` 顺带发现**本来就零调用**（deadcode/unused 都没报，因为它挂在导出类型上）|
| `internal/auth/*_test.go`、`internal/core/driverexec/exec_test.go`、`internal/domain/conn_error.go` | **仅字符串** | 都是 `"connection refused"` 字面量，与 FUSE 无关（误命中）|
| `internal/automation/**` | **零引用** | 自动化规则/动作里没有任何挂载相关项 |
| `internal/backuprestore/**` | **零引用** | 全量备份 components 不含 `fuse_mounts`（早已只在 store/backup.go 里）|
| `drivers/LocalFs/**` | **零引用** | 本机驱动与 FUSE 无关（`DefaultRoot:"/"`，用户自选目录）|

---

## 4. Q3b 构建标签矩阵（`-tags fuse` 到底控制什么）

`internal/share/fuse/` 的标签结构（逐文件实测）：

```
backend.go / nodes.go / statfs.go / write_support.go / manager_fuse.go / compiled_fuse.go   → //go:build fuse
manager_stub.go / compiled_stub.go                                                        → //go:build !fuse
types.go                                                                                  → 无标签
```

| 观察 | `-tags fuse`（默认/镜像用的）| 无标签（`make build-nofuse`）|
|---|---|---|
| `go-fuse` 在 `go list -deps ./cmd/litepan` 中 | ✅ 9 个子包 | ❌ 0 个 |
| `internal/{fusemount,fusereadcache,share/fuse}` 在依赖图中 | ✅ | **✅ 仍然在** |
| 挂载能否工作 | 能 | 不能（`stubManager.Mount` 返回 `CodeNotImplement`："当前程序未编译 FUSE 支持…"）|
| `deadcode ./cmd/litepan` | 7（无 FUSE 项）| 7（同）|

**结论**：`fuse` 标签只是**关掉实现**，不是移除机制 —— 路由、装配、DB 表、前端在两种构建下都在。要"彻底移除"必须删包 + 删装配，删完后标签本身也变得无意义（`Makefile` 的两个目标可合并、`Dockerfile` 的 `ARG BUILD_TAGS` 可删）。

---

## 5. 连带影响（删一半会炸的地方）

### 5.1 必须同步改的 SQL（唯一"删了会 runtime 报错"的点）

`internal/store/backup.go:115-117`：

```go
tables := []string{ "fuse_mounts", "automation_rules" }   // ← COUNT(1) FROM <table>
```

表被迁移删掉后，`BackupCounts`（创建备份时调用）会直接 `no such table: fuse_mounts`。**这一处是最容易漏的坑。**

### 5.2 playback 的二级死代码（≈1,000 行，需单独决定处置）

```
internal/share/fuse/nodes.go:393 → f.b.deps.Playback.OpenRemoteReader(...)   ← 全仓库唯一调用点
```

`OpenRemoteReader` 一删，以下立刻失去唯一生产者：
`internal/playback/remote_reader.go`（RemoteReader / newRemoteReader）、`internal/playback/remote_window_reader.go`、`internal/playback/local_reader.go`（`openLocalFileReader`）、`internal/playback/remote_reader_test.go`。

而 HTTP 播放链路用的是另一套：`internal/api/files.go` → `playback.Resolve` + `playback.ServeHTTP`（`range_proxy.go`/`streamer.go`/`response.go`），**与 FUSE 无关**，删 FUSE 不影响播放。

> 建议：FUSE 删除任务里**先只删 `fuseReaderUA` 与默认分支**保住编译，然后单独跑一次 `deadcode`+`unused` 确认 `RemoteReader` 圈的处置（一起删 or 保留待复用），**不要在同一刀里顺手删**。

### 5.3 删除后残留的"垃圾"（不在代码里，需人工清）

- `data/fuse_read_cache/`（68K：空 `blocks/` + 空索引 sqlite）→ 可删
- 宿主 `./mounts/`（**当前是空目录**）→ compose 不再挂载后，可删
- 若某台机器上**真的挂着** LitePan 建的 FUSE 挂载点：升级前必须先在宿主 `fusermount -u` 卸载（本机无此情况）
- 旧库中可能的 `fuse_*` 配置键与 `fuse_mount_warn` 通知行 → 由新迁移的 `DELETE` 清键；通知行**建议一并清**（否则前端会渲染未知类别）—— 本机 0 行，其他实例需在迁移里加一句

---

## 6. Q3c 部署面：删掉后能收窄多少（净收益）

| 当前 | 删除后 | 依据 |
|---|---|---|
| `privileged: true` | **可去** | 全仓唯一需要 `CAP_SYS_ADMIN` 的是 FUSE 挂载（`unix.Unmount` / go-fuse mount）|
| `pid: "host"` | **可去** | 唯一 `/proc` 读取是 `/proc/self/mountinfo`（自身命名空间），其余无处依赖宿主 PID 命名空间 |
| `devices: /dev/fuse` | **可去** | `fusemount` 是唯一使用者 |
| `./mounts:/app/mounts:shared` | **可去** | 挂载根；LocalFs 驱动不依赖它（`DefaultRoot:"/"`，用户自选）|
| 镜像内 `fuse3` / `/etc/fuse.conf` 改写 / `VOLUME /app/mounts` | **可去** | `fusermount3` 唯一调用点在 `mountpoint_linux.go` |
| `EXPOSE 42069/tcp,udp` | 与 FUSE 无关，**建议一并清** | 查实它是**内置离线下载（磁力/BT）端口**，该功能已于 `08-31-remove-builtin-offline` 移除（`anacrolix/torrent` 已出 `go.mod`），`Dockerfile:53` 的 `EXPOSE` 是那次清理的残留 |
| `network_mode: host`（README/compose.fnos）| 与 FUSE 无关，**保留** | NAS 变体用于避开 NAT；本机 compose 用的是 `ports` |

即：**这个容器可以从"特权 + 宿主 PID 命名空间 + 设备直通"退回普通容器**，只需要 `data` 与（可选的）LocalFs 目录绑定。

---

## 7. 删除任务分层清单（建议顺序，每步可编译/可验证）

| 步 | 内容 | 验证点 | 回滚点 |
|---|---|---|---|
| ① | 前端：删 `web/src/api/fuse.ts`，改 `DashboardManagement.vue`（卡片/computed/fetch/import）、`LocalDirBrowserModal.vue` 快速路径 | `npm run type-check` 通过；仪表盘无「FUSE 挂载点」卡片 | `git checkout HEAD -- web/src` |
| ② | HTTP 层：删 `fuse_admin.go`/`fuse_read_cache_admin.go`，摘 `router.go`（11 路由 + Deps/Handler）、`slow_dashboard_log.go`(+测试) | `go build ./...` 通过；`/api/admin/fuse/*` 变 404 | `git checkout HEAD -- internal/api` |
| ③ | 装配：`app.go`/`account_lifecycle.go`/`wire_services.go`/`wire_http.go` 摘除；删 `wire_fuse_read_cache.go` | `go build ./...` 通过 | `git checkout HEAD -- internal/app` |
| ④ | store：`store.go` 字段、`backup.go` 两处（**含 `BackupCounts` 的 tables**）；删 `fuse_mount_repo.go` | `go test ./internal/store/` 通过；**实测 `BackupCounts` 仍可用** | `git checkout HEAD -- internal/store` |
| ⑤ | 领域/设置/通知/playback：删 `domain/fuse_mount.go`、`settings/registry.go` 4 键+spec、`domain/notification.go` 常量、`playback` 的 `fuseReaderUA` 兜底 | `go build ./...` + `go test ./...` | 同上按文件 |
| ⑥ | 删包：`internal/fusemount/`、`internal/fusereadcache/`、`internal/share/`（整棵）| `go build ./...` + `go vet ./...` | `git checkout HEAD -- internal/{fusemount,fusereadcache,share}` |
| ⑦ | 迁移 `0025_drop_fuse.sql`（幂等 DROP + DELETE configs 键 + 清 `fuse_mount_warn` 通知）| 临时库跑迁移：表不存在、重复执行安全；历史迁移零改动 | 删新迁移文件 |
| ⑧ | 构建/部署面：`Dockerfile`、`Makefile`、两个 compose、`README.md`；`go mod tidy` | `make lint`/`go vet`/`go test` 全绿；`docker compose config` 无 fuse/privileged | `git checkout HEAD -- Dockerfile Makefile docker-compose*.yml README.md go.mod go.sum` |
| ⑨ | 重建 embed + 全量质量门 + 基线复核（`deadcode` 不高于当前 7、`unused` 0、前端零引用 0）| 全部通过 | 见上 |
| ⑩ | spec 同步（15 处，注意其中 `strm/quarktv` 等**早已过期**，一并修准）| grep 零残留 | `git checkout HEAD -- .trellis/spec` |
| ⑪ | 部署与本机实例验证（去掉 privileged/pid/devices 后容器仍正常 + 备份/播放/上传回归）| `/api/health`、登录、播放、上传、创建备份 四连通过 | compose 回滚 + 旧镜像 |

**风险最高的两步**：④（`BackupCounts` 的 SQL）与 ⑪（去掉 `privileged` 后重启容器）。

**建议拆成两个任务**：`FUSE 主体删除`（①②③④⑤⑥⑦⑧⑨⑩）与紧随其后的 `playback RemoteReader 处置 + 二级死代码复核`，避免一刀过大。

---

## 8. 反例与"不要动"的东西（防止误删）

| 项 | 为什么不删 |
|---|---|
| `internal/store/migrations/0008_fuse_mounts.sql` | 历史迁移是追加式记录；删掉会破坏已部署实例的升级路径与 `schema_migrations` 台账（项目既有结论）|
| `internal/playback/{range_proxy,streamer,response,seeker,headers,transport,pick,range,cache,service}.go` | HTTP 播放主力链路，与 FUSE 无关 |
| `internal/playback/account_range_limiter.go` | 与 HTTP range proxy 共用（只有注释提到 FUSE）|
| `internal/upload/**`（含 `TempRegistry`/`CleanupOrphanTempFiles`）| 上传自身在用；FUSE 只是**读**它的 `TempDir()`/`TempRegistry`。删 FUSE 后 `Track` 变无用，但 `Snapshot` 仍被清理任务使用 |
| `drivers/LocalFs/**` | 本机驱动，与 FUSE 无关（不依赖 `/app/mounts`）|
| `EXPOSE 42069`、`network_mode: host` | 与 FUSE 无关；42069 是已删功能的残留（可另案清），`network_mode` 是 NAS 变体需要 |
| `internal/api/local_fs.go` 其它候选路径 | 删的只是 `fusemount.MountRoot` 这一行 |
| `data/`、`data/backups/`、旧备份文件 | 与 FUSE 无关；反向删除会伤数据 |

**方法论提醒（本项目最贵的错误类型）**：这次与上次 API 秘钥一样，`deadcode` 与 `unused` **都看不见**这批代码（路由与装配让它"可达"）—— 两次实测：`deadcode` 7 行中 FUSE 相关 **0**、`unused` 0 issues。所以"工具没报 = 还在用"是错的判据；**功能性未使用必须靠"入口可达性 + 运行期证据"判定**，这也再次印证 `guides/dead-code-guide.md` 需要补一节「功能级死代码」。

---

## 9. 可复核命令（本报告每条结论的复跑入口）

```bash
export PATH=/usr/local/go/bin:/root/go/bin:$PATH GOWORK=off GOPROXY=https://goproxy.cn,direct
cd /root/LitePan

# ① 构成盘点
git ls-files | grep -i fuse | grep -v '^.trellis'
for f in $(git ls-files | grep -i fuse | grep -v '^.trellis'); do printf '%6d  %s\n' "$(wc -l < "$f")" "$f"; done

# ② 前端可达性（11 个导出 vs 实际调用）
grep -n "^export function" web/src/api/fuse.ts
for fn in fetchFuseStatus updateFuseConfig fetchFuseReadCache updateFuseReadCache clearFuseReadCache \
          createFuseMount updateFuseMount deleteFuseMount mountFuse unmountFuse fetchFuseMounts; do
  printf '  %-22s %s\n' "$fn" "$(grep -rn "\b$fn\b" web/src/ | grep -v 'api/fuse.ts' | wc -l)"
done

# ③ 构建标签矩阵
for t in fuse ""; do
  echo "--- tags='${t:-<none>}'"
  go list -tags "$t" -deps ./cmd/litepan 2>/dev/null | grep -c 'go-fuse'
  go list -tags "$t" -deps ./cmd/litepan 2>/dev/null | grep -E 'litepan/internal/(fusemount|fusereadcache|share/fuse)$'
done

# ④ 运行期证据（只读）
python3 -c "import sqlite3;c=sqlite3.connect('file:data/litepan.db?mode=ro',uri=True);print('fuse_mounts=',c.execute('SELECT COUNT(*) FROM fuse_mounts').fetchone()[0])"
grep -i fuse /proc/mounts
docker exec litepan sh -c 'grep -i fuse /proc/mounts; ls -A /app/mounts'
docker compose logs --since 720h litepan 2>&1 | grep -ci 'fuse\|挂载'
# 有会话时（表单编码登录）
curl -s -X POST http://127.0.0.1:5211/api/auth/login -d 'username=admin&password=123456' -c /tmp/c.txt -o /dev/null
curl -s -b /tmp/c.txt http://127.0.0.1:5211/api/admin/fuse/status
curl -s -b /tmp/c.txt http://127.0.0.1:5211/api/admin/fuse/mounts

# ⑤ 危险点定位（BackupCounts）
grep -n -A 6 'func (db \*DB) BackupCounts' internal/store/backup.go

# ⑥ playback 唯一调用点
grep -rn "OpenRemoteReader" --include='*.go' .

# ⑦ 两工具盲区复核
deadcode ./cmd/litepan | grep -ci fuse          # → 0
golangci-lint run --enable=unused | tail -1     # → 0 issues.
```

---

## 10. 一句话回答

**能彻底删，而且比删 API 秘钥更值得做**：它没有任何消费侧、实例上从未启用，却长期让容器以 `privileged` + 宿主 PID 命名空间 + 设备直通的方式运行；删掉后只剩两个必须同步处理的连带项（`BackupCounts` 的 SQL、playback 的 `RemoteReader` 圈），以及一批部署文件与 spec 的同步。
