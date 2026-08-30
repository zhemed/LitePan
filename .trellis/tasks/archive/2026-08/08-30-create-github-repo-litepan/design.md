# Design: Create GitHub repo LitePan and sync

## Overview

本地 `main`（`70ee23d` 等 15+ commits）领先 `Ponphil/LitePan`，需在 `zhemed` 账户下新建 `LitePan`（`public`）并推送。`gh` 已登录 `zhemed`，`token` 含 `repo` 权限。

## Boundaries

| 层 | 操作 | 保留 |
|---|---|---|
| **GitHub** | `gh repo create zhemed/LitePan --public --confirm`（若 404 则新建，200 则跳过） | `Ponphil/LitePan` 的 `origin` 历史（可选改 `upstream`） |
| **Git** | `git remote add github https://github.com/zhemed/LitePan.git` 或 `git remote set-url origin` | `data/*` 等 `gitignore` 不推 |
| **推送** | `git push -u origin main`（或 `github main`）含 `drivers/all.go` 3 驱动等 | `git lfs` 若需则 `git lfs push` |

## Data Flow

```
本地 main (70ee23d, 118M) → git push → https://github.com/zhemed/LitePan.git main → gh api → 公开访问
```

## Compatibility

- `gh` 的 `credential.helper` 已为 `gh auth`，`https` 推送无需额外 `token` 输入
- `git log origin/main..HEAD` 展示 15+ 待推提交，用户可在 `prd` 确认后执行

## Tradeoffs

- **新建 vs 复用 `LitePan-own`**：用户明确要 `LitePan`，非 `LitePan-own`，故新建
- **`origin` 改向 vs 新增 `github` remote**：推荐 `git remote rename origin upstream && git remote add origin https://github.com/zhemed/LitePan.git`，使 `origin` 即新仓库，`upstream` 保留 Ponphil，便于后续 `fetch` 上游

## Rollout / Rollback

- 单 `git push`，`gh repo delete zhemed/LitePan --confirm` 可回滚（需用户确认）
- 若 `zhemed/LitePan` 已存在且非空，`push --force-with-lease` 前需确认

## File Map

1. `gh repo create/view`
2. `git remote -v / git log`
3. `git push`
