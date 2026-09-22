# 发布 v0.0.48：上游三批移植上线

## Goal

把 `v0.0.47`（`1c23251b`）之后的功能改动——**上游三批修复的移植 + 评审跟进修复**——发布成 `v0.0.48`，并把本机 `:5211` 容器更新到该镜像：

- 对外：GHCR 三 tag（`v0.0.48` / `0.0.48` / `latest`）同 digest、`git tag v0.0.48`、GitHub Release
- 对内：容器镜像 `v0.0.47 → v0.0.48`，实例可正常登录/浏览，页脚版本号变 `v0.0.48`

发版是不可逆的对外动作 → **所有验证放在推送之前**。

## Background（实测基线，2026-09-21）

**发布内容**（`v0.0.47..main`，共 10 个提交，**无 merge 提交**）：

| 提交 | 内容 |
|---|---|
| `a54fd7ae` | 三批移植主体（115 驱动 / 性能可观测 / 清洁），17 文件 |
| `827b9a86` | spec 同步 3 处（配置表缓存边界、慢日志抑制、批次路径解析） |
| `2adc1bd1` | 评审跟进：修并发写缓存顺序（MAJOR）+ 补 A4/B3 单测，8 文件 |
| `ae162c5a` / `fb5bb313` / `d37945c5` | 移植任务归档与 journal（无代码） |
| `45ede01c` / `8af18923` / `b89383be` | 上一轮"上游更新调研"任务的归档与 journal（无代码） |
| `fc6ce288` | `v0.0.47` 发布任务的归档提交（无代码） |

即：**3 个代码/文档提交 + 7 个任务归档与 journal 记账提交**，全部由本仓库同一作者在同一 `main` 上线性提交。

**非 `.trellis` 改动共 20 个文件**（1181+/85-）：`Dockerfile`、`cmd/litepan/main.go`、
`drivers/115_Open/{full_list,ops}.go` + 2 个测试、`drivers/189Cloud/ops.go`、
`internal/adminauth/{service.go,service_test.go,config_cache_test.go}`、
`internal/api/{accounts,files,commit_writer,local_upload,oauth,router,slow_dashboard_log}.go` + 3 个测试。

**用户可见变化：无。** 本版是后端/性能/健壮性版本：B2 把"后台概况接口慢"的日志从 INFO 降到 DEBUG
并加 30 分钟抑制（日志更干净）；A1–A3 的 115 清单改动落在**当前无调用方**的代码路径上（移植口径，
非线上行为修复，已在 `full_list.go` 注释写明）；无前端改动 → **embed 不变**。

**当前发版面**（5 处 `v0.0.47`）：`internal/buildinfo/version.go`、`README.md`×2、
`docker-compose.yml`、`docker-compose.fnos.yml`。

**本轮无数据库变更**：迁移最新仍是 `0025_drop_fuse.sql`，实例库已在 25 → 镜像更新不执行任何迁移，
schema 不变，故按项目既有判据**不需要部署前备份**（备份是为不可逆迁移设的）。此判据要写进执行记录。

**本机实例现状**：`ghcr.io/zhemed/litepan:v0.0.47`，非特权（`Privileged=false`、`PidMode=""`、
`Devices=None`、仅 `data` 绑定），`:5211` health 200；库：`schema_migrations`=25、表数=9、`configs`=7 行。

## Requirements

### D1 版本号推进到 `v0.0.48`

`internal/buildinfo/version.go`（唯一真值）、`README.md`×2、`docker-compose.yml`、
`docker-compose.fnos.yml` —— 只改版本字符串，**不重建 embed**。

### D2 发布前质量门

`make lint` 0 issues、`GOWORK=off go vet ./...` exit=0、`GOWORK=off go test ./...` 全包 ok、
`cd web && npm run type-check` exit=0、`git status --short internal/api/web` 无输出。

### D3 提交并推送 `main`

版本提交消息形如 `chore(release): 版本号推进到 v0.0.48（…）`，用 `git add -A --` 显式限定 4 个文件
（**防漏提/误提**：上上轮曾出现"只提交了部分文件"的事故）。

### D4 构建镜像并推 GHCR（三 tag 同 digest）

`make docker-build DOCKER_IMAGE=ghcr.io/zhemed/litepan:v0.0.48` → 补 `:0.0.48`、`:latest` →
`gh auth token | docker login ghcr.io -u zhemed --password-stdin` → `docker push` ×3。

**镜像内交叉验证（本版必须做）**：`/app/litepan` 内 `v0.0.48` ≥1 且 `v0.0.47` = 0；并抽查本轮
新增的明文字符串（`启动失败`、`OAuth 转发重试均失败`、`映射目录`）≥1，证明镜像真带上了本轮代码，
而不是"重建了旧提交"。

### D5 git tag 与 GitHub Release

顺序：确认 main 已推 → `git tag v0.0.48` → `git push origin v0.0.48` → `gh release create v0.0.48
--notes-file`。说明须写明：本版是后端/性能版、无用户可见变化、无数据库变更、回滚只需换回 `v0.0.47`。
**发布说明不得包含内网地址/主机名/凭据**（端口与镜像坐标在仓库 README/compose 里本就有，属项目配置）。

### D6 更新本机 `:5211` 容器

`docker compose pull litepan && docker compose up -d`（**不得**手搓 `docker run`）。
无迁移 → 无备份（判据见 Background），但**部署后必须复核库未变**。

### D7 部署后验证

- `/api/health` 200；登录 200；`/api/public/system-config` 的 `version` = `v0.0.48`
- **库未变**：`schema_migrations` = 25、表数 = 9、`configs` = 7 行
- **形态未变**：`docker inspect` 的 9 个非镜像字段（Mounts/Privileged/PidMode/Devices/Binds/
  PortBindings/RestartPolicy/Env/NetworkMode）与部署前逐项一致，只允许 Image/ImageName 变
- 日志 `level=ERROR` = 0（`WARN` 若为未开放匿名列表的 401 属预期，需逐条判读）
- **浏览器验收**：`:5211` 首页/仪表盘正常渲染、页脚 `v0.0.48`、`window.__err` 为空；上传任务 API
  三端点（`tasks`/`tasks/summary`/`runtime`）仍 200；本地映射上传配置端点 200

### D8 收尾

`skill trellis-check` → `mark-check` → 勾选验收项 → `pre-archive` → `archive` → `add_session.py` → `git push`

## Constraints

- **不越界**：`internal/**`（除 `version.go`）、`web/src/**`、`Dockerfile`、`Makefile`、
  `.golangci.yml`、`go.mod`/`go.sum` 本轮零改动
- 不改部署形态：端口、`data` 绑定、`restart`、`network_mode` 保持；不清理 GHCR 历史版本、
  不删旧 tag/release（回滚退路）
- **网络动作仅限**：`ghcr.io`（docker login/push/pull）、`github.com`（git push tag / release / gh api）；
  不访问其它主机
- 本机 `danger-full-access`、审批关闭 —— 不请求 escalation；不改归档任务与历史 journal

## Acceptance Criteria

- [x] 版本字符串 5 处均 = `v0.0.48`；`v0.0.47` 在源码与部署文件中零命中（`.git` 历史除外）
- [x] 质量门全绿：`make lint` 0 issues、`go vet` exit=0、`go test ./...` 全包 ok、`vue-tsc -b` exit=0；embed 零改动
- [x] **发布内容核对**：`v0.0.47..main` 无 merge 提交；非 `.trellis` 改动恰 20 个文件且全在移植任务 File Map 内；无越界文件
- [x] 版本提交已推 `main`，工作区干净（仅任务目录）
- [x] 镜像构建成功；**镜像内**二进制含 `v0.0.48`、不含 `v0.0.47`，且含本轮新增文案（`启动失败`/`OAuth 转发重试均失败`/`映射目录`）
- [x] GHCR 单条 version 含 `v0.0.48`/`0.0.48`/`latest`（三 tag 同 digest）；`v0.0.47` 仍指向旧 digest
- [x] `git tag v0.0.48` 远端 sha == 本地；release 非草稿且说明含"无用户可见变化 / 无数据库变更 / 回滚方式"
- [x] 容器已更新到 `v0.0.48`：health 200、登录 200、`system-config.version` = `v0.0.48`
- [x] **库未变**：`schema_migrations` = 25、表数 = 9、`configs` = 7 行
- [x] 容器形态与部署前逐项一致；日志 0 ERROR；上传任务 API 三端点 200；本地映射配置 200
- [x] 浏览器验收：`:5211` 正常渲染、页脚 `v0.0.48`、`window.__err` 为空
- [x] 未越界：`internal/**`（除 `version.go`）、`web/src/**`、`Dockerfile`、`Makefile`、`.golangci.yml`、`go.mod` 零改动
- [x] 任务归档、journal 记录、`main` 与 `origin/main` 同步（版本提交 `778f5b87`）

## Notes

- Scope `infra`（构建/推送/部署），按复杂任务补 `design.md` + `implement.md`；流程沿用
  `09-15-release-0-0-47`（推送前全验证、三 tag 同 digest、远端 tag sha 回读、部署后库/形态复核）。
- 与上一版的差别：**无前端改动**（故无"用户可见面"要断言）、**无数据库变更**，回滚成本同上一版
  （compose 改回 `v0.0.47` 即可）。
- 用户特别要求：**务必核对发布内容没有搞错**（无 merge、无越界、镜像与提交一致）——D4 的
  "镜像内交叉验证"与验收项的"发布内容核对"就是这条要求的具体落地。
