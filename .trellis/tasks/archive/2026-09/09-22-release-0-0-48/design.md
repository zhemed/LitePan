# Design: 发布 v0.0.48

## 目标与不变量

**目标**：把 `v0.0.47..main`（上游三批移植 + 评审跟进）发布为 `v0.0.48`，并在本机 `:5211` 上线。

**不变量**

1. **发布内容与预期逐字节一致**：被发布的代码 = `v0.0.47..main` 里那 20 个非 `.trellis` 文件，
   不多不少；无 merge 提交、无其它分支内容（用户明确要求"排查好别合并错"）。
2. **不可逆动作（GHCR push / tag / release / 容器更新）一律排在本地验证之后**。
3. **不动数据面**：本轮无迁移、无 schema 变化 → 不做部署前备份，但必须**实测复核库未变**。
4. **不动部署形态**：端口、`data` 绑定、`restart`、网络模式、权限面全部保持不变。
5. 版本号仍只有 5 处（单一真值 + README×2 + compose×2）。

## 关键决策

### KD1 为什么本轮不做部署前备份

备份是给"不可逆迁移/数据改写"兜底的。本轮 `Dockerfile`/`internal`/`cmd` 改动里**没有任何迁移文件**
（迁移最新仍是 `0025`，实例库已在 25），`docker compose up -d` 只是换个镜像重建容器。
故按项目既有判据免备份，但要用部署后实测（`schema_migrations`=25、表数=9、`configs`=7 行）反证库确实未变 ——
这条判据写在 PRD Background，不是事后解释。

### KD2 镜像内容验证：用"二进制明文 + 本轮新增文案"双证据

v0.0.47 那次实测发现"镜像内 grep 前端文案"是**无效判据**（前端资源 gz 后 `go:embed`，明文 grep 不到）。
本轮改的是后端，因此改用两条**有效**证据：

1. 二进制里的版本字符串：`v0.0.48` ≥1、`v0.0.47` = 0（Go 字符串在 rodata 明文可 grep）；
2. 本轮新增的中文明文（`启动失败`、`OAuth 转发重试均失败`、`映射目录`）各 ≥1 —— 证明镜像带的是
   **本轮提交**的代码，而不是"重建了一个旧提交却打成新 tag"。

前端无改动 → 不做前端资源断言（embed 与 `web/src` 应零 diff）。

### KD3 "别合并错"的机械核对清单

发布前按顺序跑，任一条不符即停止：

| 检查 | 命令 | 期望 |
|---|---|---|
| 无 merge 提交 | `git log --merges --oneline v0.0.47..main` | 空 |
| 提交集合 | `git log --oneline v0.0.47..main` | 6 条（4 条本任务 + 2 条上一任务归档/journal） |
| 代码文件集合 | `git diff --name-only v0.0.47..main -- . ':!.trellis'` | 恰 20 个，且全在移植任务 File Map 内 |
| 越界文件 | 同上交集 `web/src`/`go.mod`/`.golangci.yml`/`Makefile`/`internal/**`（除 `version.go`） | 空 |
| 分支一致 | `git rev-parse main origin/main` | 同 sha |
| 无新迁移 | `ls internal/store/migrations \| tail -1` | `0025_*` |
| 版本串一致 | `grep -rn "v0\.0\.47" version.go README.md docker-compose*.yml` | 恰好 5 处（改前）→ 0 处（改后） |

## 风险与处置

| 风险 | 处置 |
|---|---|
| 版本提交漏提/误提（上上轮真实事故） | `git add -A -- <4 个文件>` 显式限定；提交后 `git show --stat` 回读文件清单 |
| 镜像与提交不一致（旧提交打成新 tag） | KD2 的双证据（版本串 + 本轮新增文案） |
| GHCR 三 tag 不同 digest | 本地 `docker tag` 后 `docker push` 三个 tag，再用 `gh api` 读回单个 version 的 tags+digest |
| 新旧镜像混用导致容器起不来 | 部署后立刻 health/登录/版本三项断言；失败即按 Rollback 回 `v0.0.47` |
| 发布说明泄露现场信息 | 说明只写本版内容/无用户可见变化/无迁移/回滚方式；不含主机名、内网地址、凭据（端口与镜像坐标仓库里本就有） |
| 内存缓存（B1）上线后权限行为异常 | 部署后实测登录、`/api/admin/system-config`、改设置后回读（非独占键不陈旧）+ 复核 ERROR=0 |

## Rollout / Rollback

**Rollout**：版本号（本地）→ 质量门 → 推 `main` → 构建+推镜像 → tag/release → `docker compose pull && up -d` → 验证。

**Rollback**：`docker-compose.yml` 改回 `ghcr.io/zhemed/litepan:v0.0.47` → `docker compose up -d`；
对外 `gh release delete v0.0.48 --cleanup-tag`（或仅删 tag）+ 删 GHCR 该 version。**数据侧无需任何动作。**

## File Map

| 路径 | 动作 |
|---|---|
| `internal/buildinfo/version.go` | `v0.0.47` → `v0.0.48` |
| `README.md`（2 处）、`docker-compose.yml`、`docker-compose.fnos.yml` | 同上 |
| `.trellis/tasks/09-22-release-0-0-48/**` | 本任务记录 |

**零改动（越界即失败）**：`internal/**`（除 `version.go`）、`web/src/**`、`internal/api/web/**`、
`Dockerfile`、`Makefile`、`.golangci.yml`、`go.mod`/`go.sum`、`data/**`。
