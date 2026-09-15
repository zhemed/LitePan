# FUSE 挂载点相关全部内容能否彻底移除（调查）

## Goal

回答一个问题：**FUSE（挂载点 / 读缓存 / share-fuse / 构建与部署面）能不能从 LitePan 里彻底删掉？**

产出 `research.md`：构成盘点 → 使用面核实 → 真实耦合清单 → 分层删除清单（含"删不掉的是什么、为什么"）→ 风险与建议。

**本轮只读**：不改任何代码、配置、镜像、容器、数据库。是否执行删除由用户看到结论后另行决定（另开任务）。

## Requirements

### Q1 构成盘点：FUSE 到底包含哪些东西

必须逐类列全，并给出**文件与行数**：

| 类别 | 已知线索（待核实） |
|---|---|
| 挂载服务 | `internal/fusemount/`（7 文件：`service.go`/`service_account.go`/`state.go`/`validate.go`/`constants.go`/`mountpoint_{linux,other}.go`）|
| 读缓存 | `internal/fusereadcache/`（7 文件：`coordinator.go`/`evict.go`/`service.go`/`store.go`/`settings.go`/`constants.go`/`service_test.go`）|
| 分享后端 | `internal/share/fuse/`（11 文件，含 `compiled_fuse.go` + `compiled_stub.go` 构建标签双分支）——**它是否还可达？** `share/dav` 早在 2026-08-30 被删 |
| 领域/存储 | `internal/domain/fuse_mount.go`、`internal/store/fuse_mount_repo.go`、迁移 `0008_fuse_mounts.sql`（`fuse_mounts` 表）|
| HTTP 层 | `internal/api/fuse_admin.go`、`internal/api/fuse_read_cache_admin.go`（端点清单 + 是否注册在 router）|
| 装配 | `internal/app/wire_fuse_read_cache.go` + `wire_services.go`/`wire_http.go`/`app.go` 中的引用 |
| 前端 | `web/src/api/fuse.ts`、`web/src/components/admin/DashboardManagement.vue`（以及是否有 FUSE 面板/tab）|
| 构建与部署 | `Dockerfile`（`ARG BUILD_TAGS=fuse`、`fuse3`、`/etc/fuse.conf`）、`docker-compose*.yml`（`/dev/fuse`、`privileged`、`pid: host`、读缓存挂载注释）、`Makefile`（`build` 用 `-tags fuse` + `build-nofuse`）、`install-docker.sh` |
| 其它引用 | `internal/playback`(3)、`internal/auth`(2)、`internal/upload`(1)、`internal/settings`(1)、`internal/core/driverexec`(1)、`internal/domain`(3)、`internal/store`(3)、`internal/api`(6) —— **逐个定性**（是真耦合还是仅注释/字符串）|

### Q2 使用面核实：这台机器上到底用没用

必须用运行期证据（而不是"代码里存在"）回答：

- 实例库 `fuse_mounts` 表行数、是否有挂载记录（当前实测 0 行 —— 复核）
- 宿主是否真有 FUSE 挂载（`mount`/`findmnt`/`/proc/mounts` 中与 litepan 相关的条目）
- `data/fuse_read_cache/`、`mounts/` 目录内容与体积（是否为空壳）
- 容器日志中 FUSE 相关活动（挂载/卸载/读缓存命中）
- 前端界面是否真的暴露该功能（哪个页签/面板，是否可见可点）
- 上游 `Ponphil/LitePan` 里 FUSE 的角色（本项目是否需要）

### Q3 真实耦合：删掉会牵连什么

- **播放链路**：`playback` 对 FUSE 读缓存是否有依赖（若是读缓存主力路径，删除会导致功能降级还是仅少一层缓存？）
- **账号生命周期**：`auth` 的 2 处引用（账号删除/停用时卸载挂载点？）
- **上传**：`upload` 的 1 处
- **设置项**：`settings` 的 1 处（是否存在 FUSE 相关配置键，删后旧库残留键怎么处理）
- **driverexec**：`core/driverexec` 的 1 处
- **自动化**：automation 动作/规则里是否引用挂载
- **备份恢复**：`backuprestore` 的 components 是否含 `fuse_mounts`；旧备份恢复路径会不会因删表而失败
- **构建标签**：去掉 `-tags fuse` 后 `compiled_stub.go` 之外的代码是否还有 `//go:build fuse` 分支；`Makefile`/`Dockerfile` 的默认构建是否必须改
- **部署形态**：删掉 FUSE 后 `privileged: true` / `pid: "host"` / `/dev/fuse` 是否**都可去掉**（这是安全收益，需明确结论）

### Q4 结论与建议

- 明确回答"能不能彻底移除"，并**区分**：代码层 / 数据表 / 前端 / 构建与部署面 四类各自能否删净
- 列出**删不掉或不应删**的部分及理由（若有）
- 给出**分层删除清单**（建议顺序 + 每步验证点 + 回滚点），供后续任务直接引用
- 明确风险：数据迁移不可逆项、兼容性（旧备份/旧设置键）、用户可见变化
- 若结论是"可以删"，额外评估**部署简化收益**：镜像体积、安全面（去 privileged/pid host/devices）、文档与安装脚本简化

## Constraints

- **只读**：不改代码/配置/DB/容器/镜像；不建新分支；不 push
- 调查方法必须**证据优先**，且沿用本仓库既有教训：
  - 死代码判定用 `deadcode` + `golangci-lint --enable=unused` **并用**（零重叠），不单用其一
  - 涉及接口的方法先查**是否被实例化**，再看可达性
  - 引用计数前**剥注释**（曾因此误判 `useTimeWindowSchedule.ts` 为活代码）
  - 包级可达性用 `go list -deps ./cmd/litepan` 复核，不能只看文件是否存在
  - 涉及构建标签的代码要用**两种 build tags**各跑一次可达性（`-tags fuse` 与 nofuse）
- 结论要给**可复核的命令**，不写"大概/应该"
- 不改 `.trellis/spec/`（除非调查中发现规范本身有错，那也只**登记**不修改）
- 探索期命令保持只读（`git ls-files`/`grep`/`go list`/`go build -o /dev/null` 到临时目录/`docker inspect`/只读 sqlite）

## Acceptance Criteria

- [x] Q1 构成盘点齐全：每一类给出**文件清单 + 行数 + 受跟踪状态**，无"等等/若干"这类含糊表述
- [x] `internal/share/fuse/` 的**可达性有明确结论**（活代码还是死代码，判据给出命令与输出）
- [x] Q2 运行期证据齐全：表行数、目录内容、宿主挂载情况、日志、前端可见性**逐项有实测输出**
- [x] Q3 逐个定性：`playback`/`auth`/`upload`/`settings`/`driverexec`/`automation`/`backuprestore` 每一处引用都判定为「真耦合 / 仅字符串 / 仅注释」，并说明删除后果
- [x] 构建标签结论明确：`-tags fuse` 与 nofuse 两种构建下，FUSE 相关包的可达性各是什么；删标签需要动哪些文件
- [x] 部署面结论明确：删掉 FUSE 后 `privileged`/`pid: host`/`/dev/fuse` 能否全部去掉，逐项给理由
- [x] Q4 给出**合并结论**（能/不能彻底移除 + 分层清单 + 顺序 + 验证点 + 回滚点 + 风险）
- [x] 所有结论都附**可复跑命令**，且命令在本机实际执行过（research.md 内含关键输出片段）
- [x] 零写操作：`git status --short` 只多出本任务目录；容器/镜像/DB 无任何改动
- [x] 若发现"某个东西看起来能删、实际删不掉"，必须**写明反例**（本仓库历史上最贵的错误就是这类误判）

## Notes

- Scope 标 `lightweight`：调查类，PRD-only（与 `09-12-investigate-dead-code-sweep`、`09-12-investigate-mediaorganize-necessity` 同规格），产物为 `prd.md` + `research.md`。
- 动机推测（不写入结论，仅解释为何现在问）：项目已连续精简（仅 3 驱动、无 STRM/媒体整理/缓存整理/离线下载/夸克TV），而 FUSE 是**唯一还需要 `privileged: true` + `pid: host` + `/dev/fuse`** 的功能 —— 若不用，删掉能显著收窄部署权限面。
- 本任务**不做**：实际删除（另开任务）、上游同步、发版。
