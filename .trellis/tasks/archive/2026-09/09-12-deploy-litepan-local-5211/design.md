# Design: 部署 LitePan v0.0.44 到本机 :5211

## Overview

单容器部署，运行时拓扑极简：

```
宿主机 /root/LitePan/                     容器 litepan (ghcr.io/zhemed/litepan:v0.0.44)
├── docker-compose.yml  ──── compose up ──►  ENTRYPOINT /app/litepan
├── data/  (gitignored) ◄── bind mount ───►  /app/data   ← DB/日志/密钥落这里
└── mounts/ (gitignored)◄── bind mount ───►  /app/mounts ← 网盘挂载点/FUSE
                                              :5211 (published)
/i\net/dev/fuse ────────── device ────────►  FUSE 能力
```

`data/` 是**唯一需要保护的东西**：`litepan.db`（账号、设置、密钥）与 `secret.key` 都在其中。因此本设计把"回滚"定义为**只 down 容器、绝不动 `data/`**。

## Boundaries

| 对象 | 改 | 不改 |
|---|---|---|
| **宿主机文件系统** | 创建 `/root/LitePan/data/`、`/root/LitePan/mounts/`（由 compose 自动创建，均 gitignored） | 仓库任何受跟踪文件；DSH 相关路径 |
| **Docker** | 新增容器 `litepan`、镜像引用 `v0.0.44`、端口发布 `5211` | 其他容器；DSH 使用的主机进程；Docker daemon 配置 |
| **端口** | 发布 `5211` | `3080`/`3081`（DSH）；不发布 `42069`（镜像内 EXPOSE，但 compose 未映射） |
| **仓库** | 无 | `docker-compose.yml`、`README.md`、任何源码、`AGENTS.md` |
| **管理员口令** | 无 | 默认口令保持原样，用户自行修改 |

## Key Decisions

### KD1 选根 `docker-compose.yml` 而非 README/fnos 变体

根文件用**相对路径**（`./data`、`./mounts`）与 `ports: "5211:5211"`，自包含且可随仓库搬移；README 快速开始与 `docker-compose.fnos.yml` 同源，硬编码 `/vol1/1000/docker/litepango/...` 这类**飞牛 NAS 绝对路径**，在本机执行会在根目录创建一个无意义的 `/vol1` 树。故本机只用根文件，且不修改它。

### KD2 `ports` 而非 `network_mode: host`

NAS 变体用 host 网络（因为 NAS 上要让容器直接接管 42069 磁力端口并避开 NAT）。本机是 DSH 开发机，host 网络会把容器全部端口（含 42069）直接摊到主机上，且与 DSH 同处一个网络命名空间、风险面更大。用 `ports` 只暴露 5211，边界清晰；本机也没有必用磁力链接的场景。

### KD3 为什么必须验证"重启后数据仍在"

"容器起来了、页面能打开"**不能**证明持久化正确 —— 如果 `data/` 实际落在容器可写层（例如 volume 拼写错误、或 compose 里写成匿名卷），一切照常工作，直到容器被重建才丢库。所以设计里把持久化验证拆成三步：① 宿主机 `data/litepan.db` 存在；② 容器内外为同一 `inode`（证明是 bind mount 而非拷贝/匿名卷）；③ `docker compose restart` 后 DB 大小不变且服务仍可用。

### KD4 保留 compose 的 `privileged` + `pid: host`

这两项看着"重"，但它们是项目既定配置且 FUSE 链路需要（`/dev/fuse` 已单独 `devices` 挂入；`pid: host` 供 FUSE daemon 与宿主进程可见性）。**本任务不擅自削弱它们** —— 改变部署形态属于另一个决策（若日后要收紧，需单独评估 FUSE 是否仍工作）。

### KD5 不改默认口令

用户明确"我去修改默认密码"。改密在业务上会触碰 `adminauth` 数据，且 `AGENTS.md` 规定"后续改密必先建 Trellis 任务并 `ask_user_question`"。本任务只**如实记录**首次启动的凭据状态（`must_change_password:true`）与入口地址，改密由用户在界面上完成。

## Compatibility

- **DSH 共存**：DSH 的 `3080`/`3081` 由 node 进程持有；LitePan 只用 `5211`，且以 `ports` 发布，不存在抢占。部署前后均核对 DSH 监听数。
- **FUSE**：宿主 `/dev/fuse`（`crw-rw-rw-`）存在，镜像内已装 `fuse3` 并设置 `user_allow_other`，compose 已 `devices` 挂入。本任务不创建任何网盘账号或挂载点，故 FUSE 只验证"存在"，不做实际挂载。
- **重启自愈**：compose 的 `restart: unless-stopped` 保证宿主重启后容器自动拉起（除非被显式 stop）。
- **数据格式**：全新库由应用自身迁移创建，不依赖任何外部初始化脚本。

## Tradeoffs

- **`privileged` 的权限面 vs FUSE 可用性**：保持 privileged 换取 FUSE 与项目一致的部署形态；代价是容器权限较高。这是项目既有取舍（README 与两个 compose 文件均如此），本任务不改变它。
- **只发布 5211 vs 也发布 42069**：本机不发布 42069，换取更小的暴露面；代价是磁力链接相关功能在容器外不可达（本机无此需求）。
- **`./data` 在仓库目录内 vs 放到 `/opt` 之类**：放仓库目录内最省事且 `.gitignore` 已覆盖，代价是"删仓库"会连带删数据。已在 Rollback 中把"绝不 `rm -rf` 仓库目录"写成硬约束。

## Rollout / Rollback

**Rollout**：`docker compose up -d` → 等健康 → 逐项验证 → 留下运行中的服务（这是交付物）。

**Rollback（保留数据）**：

| 情形 | 动作 |
|---|---|
| 容器起不来 / 端口冲突 | `docker compose down`，检查 `docker compose logs`；数据不动 |
| 验证不通过 | 先 `docker logs litepan` 取证，再 `docker compose down`；**不留半启动容器** |
| 想换成别的版本 | `docker compose down` → 改 `docker-compose.yml` 的 tag（**需另建任务**，本任务不改该文件）→ `up -d` |
| 彻底移除 | `docker compose down`；如需连数据一起清，手动 `rm -rf data mounts`（**本任务严禁**） |

**不可回滚项**：无。首次启动只会**新建** `data/`，不会覆盖任何既有数据（本机此前无 `data/`，已实测确认）。

## File Map

| 路径 | 动作 | 说明 |
|---|---|---|
| `/root/LitePan/data/` | **新增**（gitignored） | DB、日志、secret.key —— 唯一需保护的持久化数据 |
| `/root/LitePan/mounts/` | **新增**（gitignored） | FUSE/网盘挂载点 |
| 容器 `litepan` | **新增** | 镜像 `ghcr.io/zhemed/litepan:v0.0.44` |
| 主机端口 `5211` | **新增监听** | 应用入口 |
| `docker-compose.yml` | 零改动 | 已含正确 tag 与挂载 |
| 仓库受跟踪文件 | 零改动 | 本任务不产生任何 commit（除任务目录） |
