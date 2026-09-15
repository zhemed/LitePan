# Design: 发布 v0.0.47（仪表盘瘦身上线）

## 目标与不变量

**目标**：把仪表盘三处假任务展示的删除发布为 `v0.0.47`，并把本机容器更新到该镜像。

**不变量**

1. `v0.0.46` 及更早的对外痕迹（GHCR version、tag、release）**不动** —— 回滚退路。
2. **数据库零变更**：本轮无新迁移，`schema_migrations`/表数/`configs` 行数在部署前后必须完全一致。
3. 容器配置零变更：本轮的改动**不含**部署形态（权限、挂载、端口、重启策略），部署前应逐项一致。
4. 推送类动作（GHCR / tag / release）只在本地验证通过之后。

## 关键决策

### KD1 本轮不需要部署前备份 —— 但要把判据写清

项目此前两轮都在部署前做了 SQLite 在线备份，因为那两轮各自带了**不可逆迁移**（`0024` 删 api_keys 表、`0025` 删 fuse_mounts 表）。本轮**没有任何迁移**（最新迁移 `0025` 早已应用，实例在 25），`docker compose up -d` 只是换镜像 + 换静态资源。

→ 不做备份；但按"不能因为省了就看起来像漏做"的原则，PRD/设计/执行记录都写明判据，并在部署后**实测库未变**（version/表数/行数三项）。

### KD2 镜像内验证：必须证明"改动真的进了镜像"

上一轮（v0.0.46）我用的是「镜像内二进制 grep 版本号」；本轮改动是**前端静态资源**，所以镜像内的验证要针对产物：

```
docker run --rm --entrypoint /bin/sh ghcr.io/zhemed/litepan:v0.0.47 -c \
  'zcat /app/…? '     # 注意：镜像里前端是 go:embed 进二进制的，不是散文件
```

`internal/api/web` 是 `go:embed` 进二进制的 → 镜像里没有独立的 chunk 文件。因此**镜像内的判据改为**：

```sh
grep -c "运行任务" /app/litepan   # 期望 0（嵌入的 gz 资源是压缩的，需先确认可 grep）
grep -c "v0.0.47"  /app/litepan   # 期望 1
```

若压缩后的资源 grep 不到明文字符串（很可能，因为 `.gz`），则退化为**二进制版本号 + 镜像 ID 与本地构建树一致**两条，并**在部署后的实例上**用浏览器实测三处文案消失（那才是最终判据）。→ 这一点在执行时按实测结果选择，不预设。

### KD3 部署后最有价值的验证是"用户可见面"

本轮的收益就是界面变化，因此核心验收必须在**真实例**上用 `bw` 做 DOM 断言 + 截图：

- 状态行指标数 = **3**
- 概况卡片数 = **2**
- 右列 `.dashboard-panel` 数 = **1**
- 页面 innerText 不含 `运行任务` / `任务总数` / `后台任务` / `暂无后台任务`
- `window.__err` 为空、布局无塌陷

### KD4 回滚极简

无迁移、无权限变化 → 回滚 = `docker-compose.yml` 改回 `v0.0.46` + `docker compose up -d`（本地镜像仍在）。release/tag/GHCR 侧按既有三件套删除。

## 风险与处置

| 风险 | 处置 |
|---|---|
| 版本号漏改（`docker-compose.fnos.yml` 最易漏）| D1 后用 `grep -rn "v0\.0\.46"` 全扫源码与部署文件，零命中才算过 |
| embed 未随本轮提交（本轮不需要重建）| 提交前 `git status --short internal/api/web` 必须无输出；镜像构建时 Dockerfile 内部会再构建一次前端，等价于二次校验 |
| 镜像里带了旧前端 | 部署后浏览器实测三处文案消失（最终判据）+ 首页页脚版本号 |
| 容器重建失败 | 已是非特权配置且上轮实测通过；失败则 compose 回 `v0.0.46` |
| tag 与 release 指向漂移 | 固定顺序 + 推送后 `gh api …/git/ref/tags/v0.0.47` 回读比对 |

## Rollout / Rollback

**Rollout**：D1 版本 → D2 质量门 → D3 推源码 → D4 镜像三 tag → D5 tag/release → D6 更新容器 → D7 验证 → D8 归档。

**Rollback**：`docker-compose.yml` 改回 `v0.0.46` → `docker compose up -d`；对外 `gh release delete v0.0.47 --cleanup-tag` + 删 GHCR version。**数据侧无需任何动作**（本轮不改库）。

## File Map

| 路径 | 动作 |
|---|---|
| `internal/buildinfo/version.go` | `v0.0.46` → `v0.0.47` |
| `README.md` | 2 处 tag |
| `docker-compose.yml` / `docker-compose.fnos.yml` | 各 1 处 tag |

**零改动（越界即失败）**：`internal/**`（除 `version.go`）、`web/src/**`、`internal/api/web/**`、`Dockerfile`、`Makefile`、`.golangci.yml`、`go.mod`/`go.sum`、`data/**`。
