# PRD：发布 v0.0.50（安卓 302 两跳握手上线）

## Goal

把 `40307ad8`（安卓 302 两跳握手修复）发布为 `v0.0.50`，并把**飞牛生产机**的 LitePan 容器更新到该镜像，
使手机上能以 302 直链下载（字节仍直连 115 CDN）。用户当轮授权：**发 v0.0.50 + 部署到飞牛 + 推送源码**。

## Background（实测基线）

- 发布内容：`v0.0.49..main` = `40307ad8`（fix）+ 3 个记账提交（排查归档、两段 journal）。
  代码改动仅 `internal/playback/{handoff.go,handoff_test.go,service.go}` + spec 一节。
- **无数据库变更**：迁移仍 `0025`。
- 生产机（飞牛，`docker-compose.fnos.yml`：`network_mode: host`，数据在 `/vol1/1000/docker/litepan/data`）
  当前跑 `v0.0.49`；本机 `:5211` 也保持 v0.0.49（**本轮不动本机实例**，用户要的是飞牛）。
- 当前发版面 5 处 `v0.0.49`：`internal/buildinfo/version.go`、`README.md`×2、两个 compose。

## Requirements

### D1 版本号 → `v0.0.50`（5 处），不重建 embed（前端零改动）

### D2 质量门：`make lint` 0 / `go vet` exit=0 / `go test ./...` 全 ok / `vue-tsc -b` exit=0；
`git status --short internal/api/web web/src` 无输出

### D3 提交并推送 `main`（含此前 4 个未推提交：用户已授权推送）

### D4 构建镜像 → 镜像内验证（二进制含 `v0.0.50`、不含 `v0.0.49`，且含握手逻辑特征串 `HeyTapBrowser`）
→ 推 GHCR 三 tag（`v0.0.50`/`0.0.50`/`latest`）同 digest

### D5 `git tag v0.0.50` + push + `gh release create`（说明：修了什么、无数据库变更、回滚方式）

### D6 更新飞牛容器：`docker compose pull && up -d`（先存基线：容器形态 + 库计数）

### D7 部署后验证：
- health 200 / 登录 200 / `version=v0.0.50`；库未变；容器非镜像字段未变；0 ERROR
- **服务端两跳模拟（关键）**：从飞牛本机
  ① 安卓浏览器 UA 请求 `/api/files/download` → 期望 `200` + `Content-Disposition: attachment` + 0 字节 + 无 `Location`；
  ② 同资源用安卓系统下载组件 UA 再请求 → 期望 `302` + `Location` 指向 115 CDN；
  ③ 用**组件 UA** 去取该 CDN 链接 → 期望 `206`（证明签名匹配、可直连下载）
- 电脑端行为不变：桌面 UA 首次请求 → 直接 `302`

### D8 收尾：`trellis-check` → `mark-check` → 勾选 → `pre-archive` → `archive` → journal → 推送到 `origin/main`

## Constraints

- 不越界：`internal/**`（除 `version.go`）、`web/src/**`、`Dockerfile`、`Makefile`、`.golangci.yml`、`go.mod`
  本版零改动；不改部署形态（host 网络、端口、`restart`、`data` 绑定）；**不动本机 `:5211` 实例**
- 网络动作（用户本轮已授权）：`github.com`（push/tag/release/gh api）、`ghcr.io`（推送/拉取）、
  飞牛生产机（SSH + 其容器更新）；**不访问其它主机**
- 凭据纪律：生产机地址/凭据**不写入仓库、任务记录、journal**（任务与 journal 只用"飞牛生产机"指代）

## Acceptance Criteria

- [x] 版本串 5 处 = `v0.0.50`；源码/部署文件里 `v0.0.49` 零残留
- [x] 质量门全绿；前端与 embed 本版零改动
- [x] `main` 已推送（含此前 4 个未推提交），`origin/main` 与本地一致
- [x] 镜像含 `v0.0.50`、不含 `v0.0.49`、含 `HeyTapBrowser`；GHCR 三 tag 同 digest
- [x] tag 远端 sha == 本地；release 非草稿
- [x] 飞牛容器 = `v0.0.50`；health/登录/version 三连；库与容器形态未变；0 ERROR
- [x] **服务端两跳模拟三条全过**（200 空附件 → 302 → 206）
- [x] 桌面 UA 首次仍直接 302（不回归）
- [x] 任务归档 + journal + `main == origin/main`
- [x] 真机验证交由用户（不在本轮判据内，但要在报告里写明预期现象）

## Notes

- 真机若失败（安卓不发起第二跳）→ 回滚 = compose 换回上一版镜像 + `up -d`，或回退该提交。
- Scope 预判：`infra`。
