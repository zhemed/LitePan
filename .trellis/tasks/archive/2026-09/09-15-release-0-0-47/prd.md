# 发布 v0.0.47：仪表盘三处假任务展示的删除上线

## Goal

把 `v0.0.46` 之后的唯一功能改动（**删除仪表盘「运行任务 / 任务总数 / 后台任务」三处假展示**）发布成 `v0.0.47`，并**把本机 `:5211` 容器更新到该镜像**：

- 对外：GHCR 三 tag（`v0.0.47` / `0.0.47` / `latest`）同 digest、`git tag v0.0.47`、GitHub Release
- 对内：容器镜像 `v0.0.46 → v0.0.47`，`:5211` 界面变为"瘦身后"的仪表盘（状态行 3 项、卡片 2 张、右列 1 个面板）

发版是不可逆的对外动作 → **所有验证放在推送之前**。

## Background（实测基线）

**发布内容**（`v0.0.46..HEAD` 只有一个功能提交）：

| 提交 | 内容 |
|---|---|
| `7ce9900` | `refactor(dashboard)`: 删三处假任务展示（模板三块 + 三个 computed + 组件内 70 行 task-* 样式 + brutal 皮肤 3 条规则），净 −140 行、后端零改动 |

其余为任务归档与 journal。

**当前发版面**（5 处 `v0.0.46`）：`internal/buildinfo/version.go`、`README.md`×2、`docker-compose.yml`、`docker-compose.fnos.yml`。

**本轮不涉及数据库**：迁移最新是 `0025`，实例库已在 `25` → **镜像更新不会执行任何迁移，schema 不变**，故按项目既有判据**不需要部署前备份**（备份是为不可逆迁移设的；本轮改的只是前端静态资源）。这一点要在执行记录里写明，避免"省了备份"看起来像漏做。

**本机实例现状**：`ghcr.io/zhemed/litepan:v0.0.46`，非特权（`Privileged=false`、`PidMode=""`、`Devices=None`、仅 `data` 绑定），`/api/health` 200。仓库 compose 里 image 仍是 `v0.0.46` → 发版后 `docker compose up -d` 会按新 tag 重建容器。

## Requirements

### D1 版本号推进到 `v0.0.47`

- `internal/buildinfo/version.go`（唯一真值）、`README.md`（2 处）、`docker-compose.yml`、`docker-compose.fnos.yml`
- 只改版本字符串；**不重建 embed**（前端已在 `7ce9900` 重建并提交）

### D2 发布前质量门

- `make lint` 0 issues、`GOWORK=off go vet ./...` exit=0、`GOWORK=off go test ./...` 全包 ok、`cd web && npm run type-check` exit=0
- `git status --short internal/api/web` 无输出（embed 零改动）

### D3 提交并推送 `main`

### D4 构建镜像并推 GHCR（三 tag 同 digest）

- `make docker-build DOCKER_IMAGE=ghcr.io/zhemed/litepan:v0.0.47`
- **镜像内交叉验证**：镜像里 `DashboardManagement` chunk 的「运行任务/任务总数/暂无后台任务」应**零命中**（证明镜像真带上了本轮改动），且二进制含 `v0.0.47`、不含 `v0.0.46`
- 本地补 `:0.0.47`、`:latest`；登录 GHCR；`docker push` ×3
- 验证：GHCR 单条 version 含三 tag；`v0.0.46` 仍指向旧 digest

### D5 git tag 与 GitHub Release

- 顺序：确认 main 已推 → `git tag v0.0.47` → `push` → `gh release create v0.0.47 --notes-file …`
- 说明写明：删了哪三处、为什么（`1bcfac8` 起就是桩）、**用户可见变化**（状态行/卡片/面板数量）、无数据库变更、回滚只需换回 `v0.0.46`
- 验证：远端 tag sha == 本地；release 非草稿

### D6 更新本机 `:5211` 容器

- `docker compose pull litepan && docker compose up -d`（**不得**手搓 `docker run`）
- 无迁移 → 无备份（见 Background 的判据说明），但**部署后必须复核库未变**：`schema_migrations` 仍 = 25、表数仍 9、`configs` 仍 7 行

### D7 部署后验证

- `/api/health` 200；登录 200；`/api/public/system-config` 的 `version` = **`v0.0.47`**
- 容器配置与部署前**逐项一致**（本轮不应有任何配置变化：Privileged/PidMode/Devices/Mounts/Binds/Ports/RestartPolicy/Env/NetworkMode）
- 日志 `level=ERROR` = 0
- **浏览器验收（本轮核心）**：`:5211` 仪表盘 → 状态行 **3 项**（接入/在线/待确认错误）、概况卡片 **2 张**（缓存空间/未读通知）、右列 **1 个面板**（日志与通知）、**页面无「运行任务/任务总数/后台任务/暂无后台任务」**、布局无塌陷、`window.__err` 为空；首页页脚显示 `v0.0.47`
- 反例回归：自动联动页正常渲染；上传任务 API 三端点仍 200

### D8 收尾

`skill trellis-check` → `mark-check` → 勾选验收项 → `pre-archive` → `archive` → `add_session.py` → `git push`

## Constraints

- **不越界**：`internal/**`（除 `version.go`）、`web/src/**`、`Dockerfile`、`Makefile`、`.golangci.yml`、`go.mod` 本轮零改动
- 不改部署形态：端口、`data` 绑定、`restart`、`network_mode`（fnos 变体）保持
- 不清理 GHCR 历史版本、不删旧 tag/release（回滚退路）
- 本机 `danger-full-access`、审批关闭 —— 不请求 escalation
- 不改归档任务与历史 journal

## Acceptance Criteria

- [x] `internal/buildinfo/version.go` = `v0.0.47`；`README.md`×2、两个 compose 均为 `v0.0.47`；`v0.0.46` 在源码与部署文件中零命中
- [x] 质量门全绿：`make lint` 0 issues、`go vet` exit=0、`go test ./...` 全包 ok、`vue-tsc -b` exit=0；embed 零改动
- [x] 版本提交已推 `main`，工作区干净
- [x] 镜像构建成功；**镜像内** `DashboardManagement` chunk 的三处文案零命中、二进制含 `v0.0.47` 且不含 `v0.0.46`
- [x] GHCR 单条 version 含 `v0.0.47`/`0.0.47`/`latest`（三 tag 同 digest）；`v0.0.46` 仍指向旧 digest
- [x] `git tag v0.0.47` 远端 sha == 本地；release 非草稿且说明含用户可见变化与回滚方式
- [x] 容器已更新到 `v0.0.47`：health 200、登录 200、`system-config.version` = `v0.0.47`
- [x] **库未变**：`schema_migrations` = 25、表数 = 9、`configs` = 7 行（本轮无迁移）
- [x] 容器配置与部署前逐项一致（无权限/挂载/端口变化）
- [x] 日志 0 ERROR；自动联动页正常；上传任务 API 三端点 200
- [x] **浏览器验收通过**：状态行 3 项、概况卡片 2 张、**右列（`.dashboard-side`）1 个面板**、无 `运行任务/任务总数/后台任务/暂无后台任务` 文案、布局无塌陷、页脚 `v0.0.47`、`window.__err` 为空（注：`.dashboard-panel` 总数是 2，因为左列「存储账号」复用同类名）
- [x] 未越界：`internal/**`（除 `version.go`）、`web/src/**`、`Dockerfile`、`Makefile`、`.golangci.yml`、`go.mod` 零改动
- [x] 任务归档、journal 记录、`main` 与 `origin/main` 同步

## Notes

- Scope `infra`（构建/推送/部署），按复杂任务补 `design.md` + `implement.md`；流程沿用 `09-15-release-0-0-46`（推送前全验证、三 tag 同 digest、远端 tag sha 回读）。
- 本轮与上一轮的差别：**无数据库变更**、**无权限面变化**，纯粹是前端静态资源 + 版本号 → 回滚成本极低（compose 改回 `v0.0.46` 即可，无需恢复备份）。
- 遗留观察（本任务不做）：`DashboardManagement.vue` 状态句「账号、任务与缓存服务状态正常」仍含"任务"字样（状态描述，非计数展示）；`SystemSettings.vue:706` 的「后台任务」属 NAS 常驻说明。
