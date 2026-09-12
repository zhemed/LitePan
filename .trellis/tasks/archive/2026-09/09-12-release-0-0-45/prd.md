# 发布 v0.0.45：把「API 秘钥功能完全移除」发出去并更新本机实例

## Goal

把已合入 `main` 的**用户可见功能移除**（API 秘钥：前端页签 + 5 个接口 + 整包 + 数据表）发布成可部署版本 **v0.0.45**，并把本机 `:5211` 实例更新到该版本 —— 用户 2026-09-12 明确批准（版本号选 `v0.0.45`、容器「一并更新」）。

发版是**不可逆的对外动作**，因此**所有验证都放在推送之前**：代码质量门不绿不提交，镜像构建不成功不推 GHCR，tag 未推送不建 release，备份未就绪不重启容器。

## Background（实测事实）

**发布内容（main 已含，无需再改业务代码）**

| 提交 | 内容 |
|---|---|
| `b61ee9d` | `refactor(api-key): 完全移除 API 秘钥功能`（删 4 个后端文件/包 + 2 个前端文件 + 6 处接线 + 新增迁移 `0024` + 重建 embed + spec 同步）|
| `0fac3f1` / `0d41eb4` | 任务归档与 journal |

**当前发版面（全部是 `v0.0.44`）**

- 版本号单一来源：`internal/buildinfo/version.go: var Version = "v0.0.44"`（前端**不硬编码**版本，运行期经 `GET /api/public/system-config` 读取）
- `README.md:11`、`README.md:34`、`docker-compose.yml:3`、`docker-compose.fnos.yml:3`
- 本地镜像 `ghcr.io/zhemed/litepan:v0.0.44`（`sha256:a864057d…`）；GHCR 上**单条 version 含 3 个 tag**（`v0.0.44` / `0.0.44` / `latest`）同 digest —— 本轮沿用同一规则

**本机部署形态（必须原样保留，只换镜像）**

由**仓库自己的** `docker-compose.yml` 管理（`com.docker.compose.project=litepan`，`config_files=/root/LitePan/docker-compose.yml`），实测运行中容器配置：

| 项 | 值 |
|---|---|
| 镜像 | `ghcr.io/zhemed/litepan:v0.0.44`（`ImageID sha256:a864057d…`） |
| 端口 | `5211:5211`（`0.0.0.0`） |
| 绑定 | `/root/LitePan/data:/app/data`、`/root/LitePan/mounts:/app/mounts:shared` |
| 设备/权限 | `/dev/fuse`、`privileged: true`、`pid: host`、`security_opt: label=disable` |
| 重启策略 | `unless-stopped` |
| 环境 | `TZ=Asia/Shanghai` + 镜像内 `LITEPAN_DATA_DIR=/app/data`、`LITEPAN_LISTEN=:5211`、`LITEPAN_LOG_LEVEL=info` |

**实例库现状（迁移生效点）**

- `data/litepan.db`：`schema_migrations` 最大 = **23**，`api_keys` 表**仍存在且 0 行**，`configs` **7 行**
- 容器下次启动时会执行迁移 `0024_drop_api_keys.sql`（幂等 `DROP TABLE IF EXISTS`，已在四场景 + 临时实例实测通过）→ **本轮部署会真删这张表**，故必须先备份

## Requirements

### D1 版本号推进到 `v0.0.45`（1 处单一来源 + 4 处引用）

- `internal/buildinfo/version.go`：`v0.0.44` → `v0.0.45`
- `README.md`（2 处）、`docker-compose.yml`（1 处）、`docker-compose.fnos.yml`（1 处）：同样改到 `v0.0.45`
- **只改版本号**，不动其它任何内容（本轮功能改动已在 `b61ee9d` 合入）

### D2 发布前质量门

- `make lint` → `0 issues.`；`GOWORK=off go vet ./...` → exit=0；`GOWORK=off go test ./...` → 全包 `ok`；`cd web && npm run type-check` → exit=0
- **不重建 embed**：本轮前端源码零改动，且版本号不在前端源码里（运行期读取）→ 重建只会产生零 churn 的噪声。若执行，判据是产物内容哈希不变

### D3 提交并推送 `main`

- 一次提交（版本号 + README + 2 个 compose），推 `origin main`，推送后本地与 `origin/main` 同步

### D4 构建镜像并推送 GHCR（3 tag 同 digest）

- `make docker-build DOCKER_IMAGE=ghcr.io/zhemed/litepan:v0.0.45`（走项目自己的入口；需拉 `node:20-bookworm-slim`/`golang:1.26.6-bookworm`/`debian:bookworm-slim`，耗时较长 → 后台任务跑）
- 本地补 tag `:0.0.45`、`:latest`
- `gh auth token | docker login ghcr.io -u zhemed --password-stdin` → `docker push` ×3
- 验证：GHCR 上**单条 version 同时含三个 tag**（= 同 digest）

### D5 git tag 与 GitHub release

- 严格顺序：① 确认 `main` 已推 ② `git tag v0.0.45` ③ `git push origin v0.0.45` ④ `gh release create v0.0.45 --title v0.0.45 --notes <本轮说明>`
- 验证：远端 tag 的 sha **等于**本地 `git rev-parse v0.0.45`（`0.0.39` 的直接教训），`gh release view` 存在，release 说明写明「移除 API 秘钥功能」

### D6 本机 `:5211` 实例更新到 `v0.0.45`（用户批准）

- **部署前备份**：用 SQLite 在线备份（实例在跑，**不得裸 `cp`**）把 `data/litepan.db` 快照到 `data/backups/manual-pre-0024-<时间戳>.db`，并校验可打开、含 `api_keys` 表
- `docker compose pull litepan && docker compose up -d`（compose 内已是 `v0.0.45`）
- 验证：`/api/health` 200；`/api/public/system-config` 的 `version` = `v0.0.45`；实例库 `schema_migrations` = **24**、`api_keys` 表与其索引**消失**、`configs` **仍 7 行**（数据零损）；容器配置与部署前**逐项一致**（仅镜像/容器 ID 变化）
- 浏览器验收（`quality-guidelines.md` 命中「发布/部署收尾」条款）：`:5211` 后台设置页**只有 3 个页签**、**无「API 秘钥」**、登录仍正常

### D7 归档收尾

- `skill trellis-check` 走查 → `mark-check` → 勾选验收项 → `pre-archive` → `archive` → `add_session.py` → `git push`

## Constraints

- **不越界**：`Dockerfile`、`.golangci.yml`、`Makefile`、任何业务代码（`internal/**`、`web/src/**`）本轮**零改动**
- **不改部署形态**：不新增/删除 compose 服务，不改端口、绑定、设备、`privileged`/`pid`/`restart` 等任何一项
- **不清理 GHCR 历史版本**（保留回滚余地），不删旧 tag、不删旧 release
- 不改 `data/litepan.db` 内容：表删除由容器启动迁移自动完成（属预期），人工只做**只读备份与只读校验**
- 不做上游同步、不发 `1.0.0`（版本规则：`0.0.44` → `0.0.45`，用户 2026-09-12 选定）
- 本机 `danger-full-access`、审批关闭 —— **不请求任何 escalation**
- 归档与历史 journal 不改

## Acceptance Criteria

- [x] `internal/buildinfo/version.go` = `v0.0.45`；`README.md`×2、`docker-compose.yml`、`docker-compose.fnos.yml` 均为 `v0.0.45`
- [x] 全仓 `v0.0.44` 仅剩历史/归档/任务记录（源码与部署文件**零命中**）
- [x] 质量门全绿：`make lint` 0 issues、`go vet` exit=0、`go test ./...` 全包 ok、`vue-tsc -b` exit=0
- [x] 版本提交已推 `main`，`git log origin/main -1` = 该提交、工作区干净
- [x] 镜像 `ghcr.io/zhemed/litepan:v0.0.45` 构建成功且可执行（`docker run --rm --entrypoint /app/litepan … --help` 或 `docker inspect` 校验）
- [x] GHCR 单条 version 同时含 `v0.0.45` / `0.0.45` / `latest` —— **三 tag 同 digest**
- [x] git tag `v0.0.45` 已推；远端 sha **等于**本地；`gh release view v0.0.45` 存在且说明写明本轮移除内容
- [x] 部署前备份存在、可打开、含 `api_keys` 表（`data/backups/manual-pre-0024-*.db`）
- [x] 本机容器已跑 `v0.0.45`：`/api/health` 200、`/api/public/system-config` 的 `version` = `v0.0.45`
- [x] 实例库迁移到 24：`api_keys` 表与 `idx_api_keys_status` **消失**、`configs` 仍 7 行（数据零损）
- [x] 浏览器验收通过：`:5211` 后台设置页 3 个页签（账号安全/首页设置/其他设置）、**无「API 秘钥」**、登录正常
- [x] 容器配置与部署前一致：binds/ports/devices/pid/privileged/restart/environment **逐项相同**（仅镜像变）
- [x] 未越界：`Dockerfile`、`.golangci.yml`、`Makefile`、`internal/**` 业务代码、`web/src/**` 零改动
- [x] 任务已归档、journal 已记录、`main` 与 `origin/main` 同步

## Notes

- 本轮发版的正当性：不是纯死代码删除，而是**用户可见变更**（后台少一个 tab）——`09-12-remove-api-key-feature` 的执行记录里已写明「比上轮更够格发版」。
- 关联：`09-12-version-single-source-release-0-0-44`（上一轮发版流程，本任务沿用其门禁设计：G5 推送前全验证、G6 三 tag 同 digest、G6c 远端 tag sha 一致）。
- 回滚三件套：① 容器 → compose 改回 `v0.0.44` 后 `docker compose up -d`；② 数据 → 从 D6 备份恢复（`api_keys` 本就是 0 行空表）；③ 对外 → `gh release delete v0.0.45 --cleanup-tag` + 删 GHCR 版本（保留 `v0.0.44` 三个 tag 不动）。
- 本任务**不做**：上游同步、1.0.0、GHCR 旧版本清理、`error-handling.md` 僵尸段落重写（已在上一任务登记为待办）。
