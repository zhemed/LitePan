# Design: 发布 v0.0.46 并把实例切成非特权容器

## 目标与不变量

**目标**：把 FUSE 移除（含部署特权面收窄）发布为 `v0.0.46`，并让**本机实例真正以非特权方式运行**。

**不变量**

1. `v0.0.45` 及更早版本的一切对外痕迹（GHCR 版本、tag、release）**不动** —— 回滚退路。
2. 实例数据不丢：`data/` 只读校验 + 在线备份；`configs` 行数在迁移前后必须一致（实测 7）。
3. 推送类动作（GHCR / tag / release）只在**本地验证通过之后**发生。
4. 容器重建必须走 compose，不得手搓 `docker run`（否则丢 `com.docker.compose.*` 标签与网络归属）。

## 关键决策

### KD1 为什么这轮必须"发布 + 部署"一起做

FUSE 任务只改了仓库里的 compose（去特权）；容器仍在按**已存在的容器配置**运行（privileged/pid host/devices）。`docker compose up -d` 只有在镜像 tag 变化或配置变化时才会重建 —— 两者本轮都发生，因此**发布与部署天然是一步**：

```
compose(image v0.0.45 → v0.0.46, 已去特权) + docker compose up -d ⇒ 容器按新配置重建
```

### KD2 迁移 `0025` 是唯一不可逆动作 → 备份是硬前置

重建后启动即执行 `0025`：`DROP TABLE fuse_mounts` + 清 `fuse_*` 键 + 清 `fuse_mount_warn` 通知。实例上是 **0 行空表**，但仍按规矩做**在线备份**（`Connection.backup`，因为库有 `-wal`，裸 `cp` 会拿到不一致快照），并**回读校验**（`integrity_check`、version=24、`fuse_mounts` 在、`configs` 7 行）。

### KD3 回滚形态必须在发布说明里写明（本轮特有的坑）

`0025` 执行后，"回到旧镜像"是**不完整**的回滚：

| 回滚目标 | 结果 |
|---|---|
| `v0.0.45` 镜像 + 已迁移的库 | 应用能起、登录/浏览正常，但**创建备份会失败**（旧 `BackupCounts` 查 `fuse_mounts`），FUSE 相关接口会因缺表报错 |
| `v0.0.45` 镜像 + **备份恢复的库** | 完整回到旧状态（这是唯一完整回滚路径）|

→ release notes 必须写清"完整回滚 = 恢复 `manual-pre-0025-*` 备份"。

### KD4 验证必须是"部署面"的，而不是"代码面"的

本轮真正的验收对象是**运行中的容器配置**：`Privileged=false`、`PidMode=""`、`Devices=[]`、无 `/app/mounts` 绑定 —— 这是"删 FUSE"的**收益兑现**，必须用 `docker inspect` 实测，而不是靠读 compose。另外用 `POST /api/admin/backups` 打穿 `BackupCounts`（本轮最高风险点的真实调用路径）。

### KD5 残留目录清理放在部署**之后**

`data/fuse_read_cache/` 与 `./mounts/` 都是**未跟踪**目录（`data/` 被 gitignore；`mounts/` 不在 git 里）。清理时机放在容器重建成功之后：

- `./mounts/` 曾是 compose 的绑定源，**重建后**该绑定已消失 → 删除无副作用
- `data/fuse_read_cache/` 的读者（读缓存服务）已随代码删除 → 删除无副作用
- 删完再复验 health 200，确保不是"删了还在用的目录"

### KD6 不重建 embed、不 bump 之外零改动

FUSE 任务已重建 embed 并提交（`internal/api/web` 干净）。本轮源码改动**只有 5 处版本字符串**，因此 diff 应该极小；任何其它源码变化都视为越界。

## 风险与处置

| 风险 | 处置 |
|---|---|
| 迁移删错东西 | 已四场景实测 + 临时实例实跑；本实例 `fuse_mounts` 0 行；部署前备份 |
| 容器重建后起不来（去特权导致）| 已在**非特权临时容器**上实测通过（health/登录/备份/各端点）；若仍失败 → compose 回旧镜像 + 恢复备份 |
| 记忆中的"备份"其实不可用 | D6 回读校验（`integrity_check` + 结构断言），不通过就不进 D7 |
| 推送鉴权失效 | 先 `gh auth token \| docker login ghcr.io` 并验证；本地已有 `ghcr.io` auths |
| tag/release 指向漂移（0.0.39 教训）| 固定顺序 + 推送后 `gh api …/git/ref/tags/v0.0.46` 回读比对本地 sha |
| 删残留目录误删在用数据 | D9 前置核对（`mounts` 已不在 compose、读缓存代码已删）+ 删后复验 health |

## Rollout / Rollback

**Rollout**：D1 版本 → D2 质量门 → D3 推送源码 → D4 镜像三 tag → D5 tag/release → D6 备份 → D7 重建容器 → D8 部署面验证 → D9 清残留 → D10 归档。

**Rollback**

| 项 | 动作 |
|---|---|
| 对外 | `gh release delete v0.0.46 --cleanup-tag`；删 GHCR `v0.0.46` 那条 version（`v0.0.45` 及更早不动）|
| 容器 | compose image 改回 `v0.0.45` → `docker compose up -d`（本地镜像仍在）→ 但**备份创建会失败**（见 KD3）|
| 数据 | 从 `data/backups/manual-pre-0025-*.db` 覆盖 `data/litepan.db`（先停容器）→ 这才是完整回滚 |
| 源码 | `git revert <version-commit>` + push |

**触发回滚的硬条件**：质量门不绿、镜像构建失败、三 tag 非同 digest、tag sha 不一致、备份校验不过、容器起不来、`configs` 行数变化、备份创建失败、浏览器验收失败。

## File Map

| 路径 | 动作 |
|---|---|
| `internal/buildinfo/version.go` | `v0.0.45` → `v0.0.46` |
| `README.md` | 2 处 tag |
| `docker-compose.yml` / `docker-compose.fnos.yml` | 各 1 处 tag |
| `data/backups/manual-pre-0025-<ts>.db` | **新增**（部署前备份，不入库）|
| `data/fuse_read_cache/`、`mounts/` | **删除**（未跟踪的残留目录）|

**零改动（越界即失败）**：`internal/**`（除 `version.go`）、`web/src/**`、`internal/api/web/**`、`Dockerfile`、`Makefile`、`.golangci.yml`、`go.mod`/`go.sum`。
