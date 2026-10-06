# PRD：发布 v0.0.49（上游 4 项修复上线）

## Goal

把 `v0.0.48`（`778f5b87`）之后的四项移植（提交 `548b56b2`）发布为 `v0.0.49`，并把本机 `:5211` 容器更新到该镜像：

- 对外：GHCR 三 tag（`v0.0.49` / `0.0.49` / `latest`）同 digest、`git tag v0.0.49`、GitHub Release
- 对内：容器 `v0.0.48 → v0.0.49`，实例可正常登录/浏览，页脚版本变 `v0.0.49`

**用户授权**：上一轮末我问"要不要发 v0.0.49 上线这四项"，用户答"推送吧" → 发版 + 上线。

## Background（实测基线）

**发布内容**（`v0.0.48..main`）：代码提交 `548b56b2`（A1 上传强制 HTTP/1.1、A2 playback 多段 Range、
A3 189Cloud 同步盘根、A4 日志详情恒返回+前端复制 + embed 重建）+ 归档/journal 记账提交
（`4562f199`、`443d99c1`）。

**用户可见变化**：日志页非错误级别条目也能展开详情，并新增「复制日志」按钮（前端已重建 embed）。
其它三项为后端/驱动内部行为（上传更快、多段 Range 兼容、189 同步盘可浏览）。

**当前发版面**（5 处 `v0.0.48`）：`internal/buildinfo/version.go`、`README.md`×2、`docker-compose.yml`、
`docker-compose.fnos.yml`。

**无数据库变更**：迁移最新仍是 `0025`，实例库已在 25 → 不需部署前备份；部署后必须实测复核库未变。

**本机实例现状**：`ghcr.io/zhemed/litepan:v0.0.48`（Up），非特权、仅 `data` 绑定、`:5211`；
库：`schema_migrations`=25、表数=9。

## Requirements

### D1 版本号推进到 `v0.0.49`

5 处（`internal/buildinfo/version.go` 唯一真值、`README.md`×2、`docker-compose.yml`、`docker-compose.fnos.yml`）；
只改版本字符串，**不重建 embed**（embed 已在 `548b56b2` 重建）。

### D2 发布前质量门

`make lint` 0 issues、`go vet ./...` exit=0、`go test ./...` 全包 ok、`vue-tsc -b` exit=0；
`git status --short internal/api/web web/src` 无输出（本版这两处零改动）。

### D3 提交并推送 `main`

`git add -A --` 显式限定 4 个文件；提交后 `git show --stat` 回读；推送后回读 `origin/main`。

### D4 构建镜像并推 GHCR（三 tag 同 digest）

`make docker-build DOCKER_IMAGE=ghcr.io/zhemed/litepan:v0.0.49` → 补 `:0.0.49`/`:latest` → 登录 → push ×3。
**镜像内交叉验证**：二进制含 `v0.0.49` ≥1、`v0.0.48` = 0，且含本轮新增字符串 `同步盘`（A3）——
证明镜像带的是本轮代码而非旧提交。

### D5 git tag 与 GitHub Release

`git tag v0.0.49` → push → `gh release create v0.0.49 --notes-file`；说明写明用户可见变化、
无数据库变更、回滚方式（换回 `v0.0.48`）；不含内网地址/主机名/凭据。

### D6 更新本机 `:5211` 容器

`docker compose pull litepan && docker compose up -d`（**不得**手搓 `docker run`）。

### D7 部署后验证

health 200；登录 200；`public/system-config.version` = `v0.0.49`；库未变（`schema_migrations`=25、表数=9、
`configs`=7 行）；容器 9 个非镜像字段与部署前逐项一致（仅 Image/ImageName 变）；`level=ERROR` = 0；
`/api/files/upload/tasks|summary|runtime`、本地映射配置、日志接口全 200；
浏览器：页脚 `v0.0.49`、`window.__err` 空、日志页存在详情/「复制日志」。

### D8 收尾

`skill trellis-check` → `mark-check` → 勾选验收 → `pre-archive` → `archive` → `add_session.py` → `git push`。

## Constraints

- 不越界：`internal/**`（除 `version.go`）、`web/src/**`、`Dockerfile`、`Makefile`、`.golangci.yml`、
  `go.mod` 本版零改动；不改部署形态；不清理 GHCR 历史版本与旧 tag/release（回滚退路）
- 网络动作仅 `ghcr.io`（login/push/pull）与 `github.com`（push tag / release / gh api）——用户本轮已授权
- 本机 `danger-full-access`、审批关闭；不改归档任务与历史 journal

## Acceptance Criteria

- [x] 版本字符串 5 处 = `v0.0.49`；`v0.0.48` 在源码与部署文件零命中
- [x] 质量门全绿；`internal/api/web` 与 `web/src` 本版零改动
- [x] 发布内容核对：`v0.0.48..main` 无 merge；改动仅限本轮移植与记账文件
- [x] 镜像构建成功；镜像内二进制含 `v0.0.49`、不含 `v0.0.48`，且含 `同步盘`
- [x] GHCR 单条 version 含 `v0.0.49`/`0.0.49`/`latest`（同 digest）；`v0.0.48` 仍指向旧 digest
- [x] `git tag v0.0.49` 远端 sha == 本地；release 非草稿
- [x] 容器 `v0.0.49`：health 200 / 登录 200 / version 正确；库未变；形态未变；0 ERROR；端点回归 200
- [x] 浏览器：页脚 `v0.0.49`、`window.__err` 空、日志页有「复制日志」
- [x] 任务归档 + journal + `main == origin/main`
