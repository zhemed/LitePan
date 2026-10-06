# Design：发布 v0.0.49

## 目标与不变量

**目标**：把 `v0.0.48..main` 的四项移植（`548b56b2`）发布为 `v0.0.49` 并在本机 `:5211` 上线。

**不变量**

1. 发布内容 = `v0.0.48..main` 的全部提交（代码 1 个 + 记账 6 个），**无 merge**、无其它分支内容。
2. 不可逆动作（GHCR push、tag、release、容器更新）全部排在本地验证之后。
3. 不动数据面：无迁移（仍是 `0025`）→ 不做部署前备份，但**部署后实测复核库未变**。
4. 不动部署形态：端口、`data` 绑定、`restart`、网络模式、权限面全部保持。
5. 版本号仍只有 5 处（唯一真值 + README×2 + compose×2）。

## 关键决策

### KD1 本版与 v0.0.48 的差别

| 维度 | v0.0.48 | v0.0.49 |
|---|---|---|
| 前端 | 零改动（embed 不变） | **有改动**（日志页详情+复制），embed 已在代码提交里重建 |
| 数据库 | 无迁移 | 无迁移 |
| 用户可见面 | 无 | **日志页**：非错误条目可展开详情 + 「复制日志」按钮 |

→ 因此本版的浏览器验收要**多看一项**：日志页详情与复制按钮存在（v0.0.48 只需页脚版本）。

### KD2 镜像内容验证

沿用 v0.0.48 的双证据：二进制含 `v0.0.49` = 1、`v0.0.48` = 0；外加本轮新增字符串 **`同步盘`**（A3 的
`GetFileInfo` 返回值名）≥1 —— 证明镜像里确实是本轮代码，而不是重建旧提交。
前端资源 gz 后 embed，明文 grep 不到（v0.0.47 那次已实证），故**不对前端文案做镜像内断言**，
前端改动以部署后浏览器 DOM 为准。

### KD3 无备份的判据

迁移最新 `0025`、实例库已在 25、本版改动无迁移文件 → 免备份；用部署后 `schema_migrations`=25、
表数=9、`configs`=7 行三项实测反证（判据写在 PRD Background，不是事后解释）。

## 风险与处置

| 风险 | 处置 |
|---|---|
| 版本提交漏提/误提 | `git add -A -- <4 文件>` + `git show --stat` 回读 |
| 镜像与提交不一致 | KD2 双证据（版本串 + `同步盘`） |
| 三 tag 不同 digest | `docker tag` 后逐个 push，用 `gh api` 回读单条 version 的 tags+digest |
| 部署后页面异常（本版有前端改动） | 浏览器实测：页脚 `v0.0.49`、`window.__err` 空、日志页「复制日志」存在；异常则按 Rollback 回 `v0.0.48` |
| A4 改了后端 DTO | 部署后实测 `/api/logs` 返回非 ERROR 条目带 `details`（本地已验，线上再抽验一次） |

## Rollback

`docker-compose.yml` 换回 `ghcr.io/zhemed/litepan:v0.0.48` → `docker compose up -d`；
对外 `gh release delete v0.0.49 --cleanup-tag` + 删 GHCR 该 version。**数据侧无动作。**

## File Map

| 路径 | 动作 |
|---|---|
| `internal/buildinfo/version.go`、`README.md`(×2)、`docker-compose.yml`、`docker-compose.fnos.yml` | `v0.0.48` → `v0.0.49` |
| `.trellis/tasks/10-06-release-0-0-49/**` | 本任务记录 |

**零改动（越界即失败）**：`internal/**`（除 `version.go`）、`web/src/**`、`internal/api/web/**`、
`Dockerfile`、`Makefile`、`.golangci.yml`、`go.mod`/`go.sum`、`data/**`。
