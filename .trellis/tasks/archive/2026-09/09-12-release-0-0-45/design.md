# Design: 发布 v0.0.45（API 秘钥功能移除）

## 目标与不变量

**目标**：把 `main` 上的功能移除（`b61ee9d`）落成对外可部署的 `v0.0.45`，并把本机 `:5211` 实例换成该版本。

**不变量（任何时刻都必须成立）**

1. `v0.0.44` 这一版**始终可用**：GHCR 三 tag、git tag、release、备份都不动 → 回滚永远有退路。
2. 本机实例的**数据与部署形态不变**：`data/` 绑定、端口、设备、权限、重启策略一律不动，只换镜像。
3. 推送动作（GHCR / tag / release）**只在本地验证通过之后**发生。

## 关键决策

### KD1 只改版本号，不碰功能代码

本轮发布的内容早已在 `b61ee9d` 合入并验证过；`v0.0.45` 与 `v0.0.44` 的**唯一源码差异**就是版本字符串与文档。若本轮再顺手改功能，等于把「功能验证」和「发布验证」混在一起，出问题无法二分。

→ 改动面精确为 5 处字符串：`internal/buildinfo/version.go` ×1、`README.md` ×2、`docker-compose.yml` ×1、`docker-compose.fnos.yml` ×1。

### KD2 不重建 embed（与上一轮发版的关键差异）

上一轮（`0.0.44`）重建 embed 是因为前端源码改了；**本轮前端源码零改动**，且版本号**不在前端源码里** —— 它由 `GET /api/public/system-config` 运行期下发（`web/src/stores/appInfo.ts`）。因此：

- 提交里**不会有** `internal/api/web/**` 的 churn；页脚徽标显示的 `v0.0.45` 来自新镜像的**后端** `buildinfo.Version`；
- 若误做 `npm run build`，产物哈希应完全相同（内容寻址）→ 一旦出现真实 churn 就说明「版本号被硬编码回前端」的回归，需要停下来查。

这条也顺带**复用**上一轮建立的单一来源机制，而不是重新解释一遍。

### KD3 镜像三 tag 同 digest，沿用既定惯例

`ghcr.io/zhemed/litepan:` + `v0.0.45` / `0.0.45` / `latest` 指向**同一条 manifest**（`0.0.44` 就是这么做的：单条 version 含三 tag）。做法是本地 `docker tag` 三次后 `docker push` 三次 —— 三次推送的是同一本地镜像 ID，故 digest 必然一致。

### KD4 release 与 tag 的顺序固定为「先 tag 后 release」

`git tag` → `git push origin v0.0.45` → `gh release create v0.0.45`。理由：`gh release create` 在 tag 不存在时会**自己造一个打到默认分支 HEAD 的 tag**，历史上有过 tag 指向漂移的坑（`0.0.39`），所以 tag 必须先由我们显式创建并推送。

并且**推送后必须回读校验**：`gh api repos/zhemed/LitePan/git/ref/tags/v0.0.45 --jq .object.sha` == 本地 `git rev-parse v0.0.45`。

### KD5 部署用 compose 自己走，不手搓 `docker run`

运行中的容器实测带 `com.docker.compose.project=litepan` 标签，`config_files` 就是本仓库的 `docker-compose.yml`。所以换版只需：

1. compose 里把 image 改成 `v0.0.45`（已在 D1 完成）
2. `docker compose pull litepan && docker compose up -d`

**不要**用 `docker run` 重建：那会丢掉 compose 的 project/network 标签（容器不在 `litepan_default` 网络里），部署形态被悄悄改掉 —— 正是「只换镜像」这条不变量要防的。

**验证方式**：部署前后各取一次 `docker inspect` 的 `Mounts / PortBindings / Devices / Privileged / PidMode / RestartPolicy / Env`，逐项比对，只有镜像与容器 ID 允许不同。

### KD6 迁移 0024 由容器启动自动执行 —— 所以先备份

实例库现在还是 version 23、`api_keys` 表存在（0 行）。容器一启动就会跑 `0024_drop_api_keys.sql`。虽然该迁移已在**四场景 + 临时实例**实测（含幂等重跑与旧备份重放），但这是**本机实例库第一次真跑**，且 `DROP TABLE` 不可逆 → 部署前用 **SQLite 在线备份 API**（`VACUUM INTO` 或 `sqlite3` 的 backup）落一份快照到 `data/backups/manual-pre-0024-<ts>.db`。

**不得用 `cp data/litepan.db`**：实例在跑且存在 `-wal`（实测 53KB 未检查点数据），裸 cp 会拿到撕裂/过期的快照。备份后要**回读校验**：可打开、`schema_migrations` 最大 = 23、`api_keys` 表存在。

### KD7 浏览器验收：这次验的是"真部署面"

上一轮是在临时实例上验收「新前端会不会渲染坏」；本轮是在 **`:5211` 真实例**上验收**功能移除的最终效果**：后台设置页只剩 3 个页签、无「API 秘钥」。这正是 `web/frontend/quality-guidelines.md` 里「Release / deployment wrap-up → 必须确认用户真正看到的界面」那一条。

## 风险与处置

| 风险 | 处置 |
|---|---|
| 镜像构建缺依赖（首次拉 `node:20`/`golang:1.26.6`/`debian:bookworm-slim`）| 后台任务跑并留日志；构建失败就停在这里，**不推任何东西** |
| GHCR 推送鉴权失效 | 先 `gh auth token \| docker login ghcr.io` 并验证；`config.json` 已有 `ghcr.io` auths 条目（实测） |
| tag 与 release 指向漂移（0.0.39 教训）| KD4 的顺序 + 推送后回读 sha 比对 |
| 容器重建后起不来（配置漂移/FUSE 权限）| compose 原样不动，只改 image；起不来就 `docker compose logs` + 回滚 compose 到 `v0.0.44` |
| 迁移把不该删的东西删了 | 已四场景实测；且有 KD6 快照。部署后校验 `configs` 仍 7 行、表数 11→10（只少 `api_keys`） |
| 上游 `latest` 语义（用户拉 `latest` 会拿到新版本）| 这是既定惯例（`0.0.44` 已如此），本轮沿用；发布说明里写明移除内容 |

## Rollback

| 项 | 回滚动作 |
|---|---|
| 本机实例 | `docker-compose.yml` 改回 `v0.0.44` → `docker compose up -d`（旧镜像本地仍在，无需重新拉） |
| 数据 | 从 `data/backups/manual-pre-0024-<ts>.db` 恢复（`api_keys` 本就是空表，实际无数据损失） |
| 代码 | `git revert <version-commit>` + `git push`（只回滚版本号与文档） |
| 对外 | `gh release delete v0.0.45 --cleanup-tag`；GHCR 删该 version（`delete:packages` 权限已有）；**`v0.0.44` 三 tag 全程不动** |

**触发回滚的硬条件**：质量门不绿、镜像构建失败、GHCR 三 tag 非同 digest、tag sha 不一致、容器起不来、`configs` 行数变化、浏览器验收失败。

## File Map

| 路径 | 动作 |
|---|---|
| `internal/buildinfo/version.go` | 改 `v0.0.44` → `v0.0.45`（1 处） |
| `README.md` | 改 2 处 tag 引用 |
| `docker-compose.yml` | 改 1 处 tag 引用（同时是部署配置） |
| `docker-compose.fnos.yml` | 改 1 处 tag 引用 |
| `data/backups/manual-pre-0024-<ts>.db` | **新增**（部署前备份，不入库） |
| `.trellis/tasks/09-12-release-0-0-45/**` | 本任务产物 |

**零改动（越界即失败）**：`Dockerfile`、`.golangci.yml`、`Makefile`、`internal/**`（除 `version.go`）、`web/src/**`、`internal/api/web/**`。
