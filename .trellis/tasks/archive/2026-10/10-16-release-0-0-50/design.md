# Design：v0.0.50 发布

## 与 v0.0.49 的差别

| 维度 | v0.0.49 | v0.0.50 |
|---|---|---|
| 后端 | — | `internal/playback/handoff.go`(新) + `service.go`(改)：安卓两跳握手 |
| 前端 | — | 零改动（不重建 embed） |
| 数据库 | 无迁移 | 无迁移 |
| 用户可见 | — | 安卓手机可用 302 直链下载（此前报 ERR_INVALID_RESPONSE） |

## 关键决策

- **KD1 镜像内验证特征串**：`HeyTapBrowser`（握手 UA 标记表里的独有字符串）—— 证明镜像含本轮逻辑；
  同时断言 `v0.0.50`=1、`v0.0.49`=0。
- **KD2 部署目标**：**只更新飞牛生产机**（用户要真机验证的就是它）；本机 `:5211` 保持 v0.0.49 不动。
- **KD3 服务端两跳模拟**：从飞牛本机用三种 UA 打 `/api/files/download`（浏览器 UA → 空附件；组件 UA → 302；
  再用组件 UA 取 CDN → 206）。这能在**没有安卓真机**的情况下验证"改法成立"；
  真机只剩余一个未知量：安卓是否真的发起第二跳。
- **KD4 无备份**：迁移仍 0025 → 免备份；用部署后 `schema_migrations`/表数/`configs` 计数反证库未变。

## Rollback

飞牛 compose 镜像换回 `ghcr.io/zhemed/litepan:v0.0.49` → `docker compose up -d`；
对外 `gh release delete v0.0.50 --cleanup-tag` + 删 GHCR 该 version。数据侧无动作。

## File Map

| 路径 | 动作 |
|---|---|
| `internal/buildinfo/version.go`、`README.md`×2、`docker-compose.yml`、`docker-compose.fnos.yml` | `v0.0.49` → `v0.0.50` |
| `.trellis/tasks/10-16-release-0-0-50/**` | 本任务记录 |

**零改动**：`internal/**`（除 version.go）、`web/src/**`、`internal/api/web/**`、`Dockerfile`、`Makefile`、
`.golangci.yml`、`go.mod`、`data/**`、本机 `:5211` 容器。
