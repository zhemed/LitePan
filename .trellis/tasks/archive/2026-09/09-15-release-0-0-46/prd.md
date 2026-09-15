# 发布 v0.0.46：把 FUSE 移除发出去，并把本机实例切成非特权容器

## Goal

把 `v0.0.45` 之后的**用户可见改动**（FUSE 本地挂载功能整体移除 + 部署特权面收窄）发布成 `v0.0.46`，并**把本机 `:5211` 实例真正切到非特权运行**：

- 对外：GHCR 三 tag（`v0.0.46` / `0.0.46` / `latest`）同 digest、`git tag v0.0.46`、GitHub Release
- 对内：容器重建后**不再有** `privileged: true` / `pid: "host"` / `/dev/fuse` / `mounts` 绑定；迁移 `0025` 在实例库上执行
- 收尾：清掉 `data/fuse_read_cache/` 与宿主 `./mounts/` 两个已无用途的空残留目录

发版是不可逆的对外动作 → **所有验证放在推送之前**；容器重建会执行 `DROP TABLE`（本实例为空表）→ **部署前必须备份**。

## Background（实测基线）

**发布内容**（`v0.0.45..HEAD` 共 15 个提交，功能改动 3 个）：

| 提交 | 内容 |
|---|---|
| `726fb7a` | `refactor(fuse)`: 彻底移除 FUSE（三个包 / 11 条端点 / 前端卡片 / 迁移 0025 / Dockerfile+Makefile+compose+README 去特权与构建标签 / go.mod 去 go-fuse）|
| `0d28d94` | `refactor(playback)`: 删除失去唯一入口的 RemoteReader 圈（-977 行）|
| `bcd38f0` | `chore(cleanup)`: 僵尸 `EXPOSE 42069` + `ActiveTempPaths` + `TempRegistry` 链 |

**当前发版面**（全部 `v0.0.45`）：`internal/buildinfo/version.go`、`README.md` ×2、`docker-compose.yml`、`docker-compose.fnos.yml`（**3 文件 4 处**，另加 `version.go` 共 5 处）。

**本机实例现状**：容器 `litepan` 跑 `ghcr.io/zhemed/litepan:v0.0.45`，带 `privileged` + `pid: host` + `/dev/fuse` + `./mounts` 绑定；实例库 `schema_migrations` 最大 = **24**、`fuse_mounts` **仍存在且 0 行**、`configs` **7 行**、`fuse_*` 配置键 0 个。

**仓库里的 compose 已经是非特权版**（FUSE 任务里改的），只是 image tag 还写着 `v0.0.45` → 发布后 `docker compose up -d` 会按新配置重建容器。

**回滚的关键前提（必须在发布说明与验证里讲清）**：迁移 `0025` 一旦执行，回到 `≤v0.0.45` 的镜像**不完整**（旧代码的 `BackupCounts` 会 `SELECT COUNT(1) FROM fuse_mounts` → 创建备份报错；旧 FUSE 服务也会因缺表而异常）。**完整回滚只能靠部署前备份恢复整库**。

## Requirements

### D1 版本号推进到 `v0.0.46`

- `internal/buildinfo/version.go`（唯一真值）、`README.md`（2 处）、`docker-compose.yml`、`docker-compose.fnos.yml`
- 只改版本字符串，不动其它内容

### D2 发布前质量门

- `make lint` 0 issues、`GOWORK=off go vet ./...` exit=0、`GOWORK=off go test ./...` 全包 ok、`cd web && npm run type-check` exit=0
- **不重建 embed**：FUSE 任务已重建并提交（`internal/api/web` 工作区干净），本轮前端零改动

### D3 提交并推送 `main`

### D4 构建镜像并推 GHCR（三 tag 同 digest）

- `make docker-build DOCKER_IMAGE=ghcr.io/zhemed/litepan:v0.0.46`
- 本地补 `:0.0.46`、`:latest`；`gh auth token | docker login ghcr.io`；`docker push` ×3
- 验证：GHCR 单条 version 同时含三 tag；镜像内二进制 `grep v0.0.46` = 1、`v0.0.45` = 0

### D5 git tag 与 GitHub Release

- 顺序固定：确认 main 已推 → `git tag v0.0.46` → `git push origin v0.0.46` → `gh release create v0.0.46 --notes-file …`
- 说明须写明：移除内容、**升级提示（可去掉 privileged/pid/devices/mounts）**、迁移 `0025` 的行为、以及**回滚需恢复整库**这一条
- 验证：远端 tag sha == 本地；release 非草稿

### D6 部署前备份（不可逆动作前的最后一道）

- 用 **SQLite 在线备份 API**（禁止裸 `cp`）把 `data/litepan.db` 快照到 `data/backups/manual-pre-0025-<时间戳>.db`
- 校验：`integrity_check=ok`、`version=24`、**`fuse_mounts` 表存在**（0 行）、`configs` 7 行

### D7 用非特权 compose 重建本机 `:5211` 容器

- `docker compose pull litepan && docker compose up -d`
- **不得**手搓 `docker run`（会丢 compose 网络标签）

### D8 部署后验证（本任务的核心证明）

- `/api/health` 200；登录 200；`/api/public/system-config` 的 `version` = **`v0.0.46`**
- 实例库：`schema_migrations` = **25**、`fuse_mounts` 与其索引**消失**、`fuse_*` 配置键 0、`configs` **仍 7 行**、表数 **10 → 9**
- **创建一次备份成功**（打穿 `BackupCounts` 真实路径，且证明迁移后旧逻辑缺陷未被带进来）
- 容器配置：**`Privileged=false`、`PidMode=""`、`Devices=[]`、无 `/app/mounts` 绑定**；端口仍是 5211
- 日志无 `level=ERROR`；旧端点 `/api/admin/fuse/*` 返回 404
- 浏览器验收：仪表盘 **3 张卡片**（无「FUSE 挂载点」）、设置页「其他设置」无 FUSE 读缓存项、布局无塌陷

### D9 清理两个空残留目录

- 删 `data/fuse_read_cache/`（空 `blocks/` + 空索引 sqlite）与宿主 `./mounts/`（空目录）
- **前置核对**：两者确为空/无引用（`mounts` 已不在 compose 里；`fuse_read_cache` 的代码已删）
- 删后复验：容器仍在跑、`/api/health` 200

### D10 收尾

- `skill trellis-check` → `mark-check` → 勾选验收项 → `pre-archive` → `archive` → `add_session.py` → `git push`
- spec 若需同步（例如版本号出现在规范里）一并处理

## Constraints

- **不越界**：`internal/**`、`web/src/**`、`Dockerfile`、`Makefile`、`.golangci.yml` 本轮**零改动**（只改版本字符串与 docs，及清理未跟踪的残留目录）
- 不改部署形态之外的东西：端口、`data` 绑定、`network_mode`（fnos 变体）、`restart` 策略保持
- 不清理 GHCR 历史版本（保留回滚余地）、不删旧 tag/release
- 迁移不可逆 → D6 备份是硬前置，未完成不得执行 D7
- 本机 `danger-full-access`、审批关闭 —— 不请求 escalation
- 不改归档任务与历史 journal

## Acceptance Criteria

- [x] `internal/buildinfo/version.go` = `v0.0.46`；`README.md`×2、两个 compose 均为 `v0.0.46`；`v0.0.45` 在源码与部署文件中零命中（历史/归档除外）
- [x] 质量门全绿：`make lint` 0 issues、`go vet` exit=0、`go test ./...` 全包 ok、`vue-tsc -b` exit=0；`internal/api/web` 零改动
- [x] 版本提交已推 `main`，工作区干净
- [x] 镜像构建成功且**镜像内二进制**含 `v0.0.46`、不含 `v0.0.45`
- [x] GHCR 单条 version 同时含 `v0.0.46` / `0.0.46` / `latest`（三 tag 同 digest）
- [x] `git tag v0.0.46` 已推且远端 sha == 本地；`gh release view v0.0.46` 存在、非草稿，说明含升级提示与回滚前提
- [x] 部署前备份存在、可开、`version=24`、`fuse_mounts` 表在、`configs` 7 行
- [x] 容器已更新到 `v0.0.46`：health 200、登录 200、`system-config.version` = `v0.0.46`
- [x] 实例库迁移到 25：`fuse_mounts`+`idx_fuse_mounts_*` 消失、`fuse_*` 键 0、`configs` 仍 7 行、表数 9
- [x] **创建备份成功**（`POST /api/admin/backups` 2xx）
- [x] 容器配置实测：`Privileged=false`、`PidMode=""`、`Devices=[]`、无 `/app/mounts` 绑定
- [x] 日志 0 ERROR；`/api/admin/fuse/{status,mounts,read-cache}` 均 404
- [x] 浏览器验收通过：仪表盘 3 张卡片无 FUSE、设置页无 FUSE 读缓存项、布局正常
- [x] `data/fuse_read_cache/` 与 `./mounts/` 已删除，删后 health 仍 200
- [x] 未越界：**本任务自身 diff 只有 4 个版本文件**（`internal/buildinfo/version.go`、`README.md`、两个 compose）；`web/src/**`、`internal/api/web/**`、`Makefile`、`.golangci.yml`、`go.mod` 零改动。说明：`Dockerfile`/`internal/upload/**` 的改动来自上一任务 `aa9d244` 的**补交**（其 `bcd38f0` 漏加暂存），已在 implement.md 的执行记录中披露
- [x] 任务归档、journal 记录、`main` 与 `origin/main` 同步

## Notes

- Scope `infra`（构建/推送/部署/运维），按复杂任务补 `design.md` + `implement.md`。
- 上一轮发版 `09-12-release-0-0-45` 的流程（G 门设计：推送前全验证、三 tag 同 digest、远端 tag sha 回读）本轮沿用。
- 本轮发版正当性：**用户可见的功能移除**（后台少一张卡片、少一套接口）+ **部署权限面显著收窄**（去 privileged/pid host/devices）。
- 建议顺序：D1 → D2 → D3 → D4 → D5 → D6 → D7 → D8 → D9 → D10；D6 未完成绝不进 D7。
