# Create GitHub repo LitePan and sync

## Goal

在 GitHub 上创建仓库 `zhemed/LitePan`（若不存在则新建，存在则直接同步），将本地 `/root/LitePan`（`litepan-go:three-drivers 118M`，已精简 115/天翼/本机存储等）完整同步到 GitHub，`origin` 指向新仓库，`main` 分支可被用户公开访问。

## Requirements

- **仓库创建**：
  - `gh repo create zhemed/LitePan --public --description "LitePan - 精简版云盘聚合（仅115/天翼/本机存储，移除STRM/共享/缓存整理/增强工具/跨盘秒传等）" --confirm`（若已存在则跳过创建，直接 `gh repo view` 确认）
  - 可见性：`public`（与 `LitePan-own` 一致），`clone` 需无需 `token`
  - 若 `zhemed/LitePan` 已存在且非空，需 `git remote` 检查后 `force-with-lease` 或提示用户确认覆盖
- **同步**：
  - 本地 `main` 已领先 `origin Ponphil/LitePan` 约 15+ commits（`refactor(drivers): keep only 115 189 LocalFs` 等），需将本地 `main` 推送到新仓库的 `main`
  - `git remote` 处理：保留 `origin` 指向 `Ponphil/LitePan` 为 `upstream`，新增 `github` 或直接将 `origin` 改为 `zhemed/LitePan`（二选一，需在 `prd` 明确）
  - 推送内容：`git push -u origin main`（或 `github main`），含所有 `refactor`、`docs(spec)`、`chore(task)` 提交与 `internal/api/web` 资产
  - 大文件：`git lfs` 已启用（`filter.lfs.required=true`），需 `git lfs push` 若有
- **验证**：
  - `gh repo view zhemed/LitePan --json name,visibility,url` 返回 `public`
  - `gh api repos/zhemed/LitePan --jq .clone_url` 与 `git ls-remote https://github.com/zhemed/LitePan.git HEAD` 的 `sha` 与本地 `git rev-parse HEAD` 一致
  - `curl -I https://github.com/zhemed/LitePan` 200

## Constraints

- 不改 `Ponphil/LitePan` 的 `origin` 历史，仅新增 `zhemed/LitePan` 为同步目标；若用户要求 `origin` 改向，需显式确认
- `gh` 已登录 `zhemed`（`hosts.yml` token 有效，`scope: repo`），`git credential helper` 为 `gh auth`
- 本地 `main` 为 `Clean`，`git status` 无未提交，`litepan.db` 等 `data/*` 已 `gitignore` 不推送
- 推送前 `git log --oneline origin/main..HEAD` 展示待推提交供用户确认

## Acceptance Criteria

- [ ] `gh repo view zhemed/LitePan --json nameWithOwner,visibility` 返回 `zhemed/LitePan` `PUBLIC`
- [ ] `git remote -v` 含 `https://github.com/zhemed/LitePan.git`（`origin` 或 `github`）
- [ ] `git push` 后 `git rev-parse HEAD` == `gh api repos/zhemed/LitePan/commits/main --jq .sha`（或 `git ls-remote`）
- [ ] `https://github.com/zhemed/LitePan` 可公开访问，`README.md` 与本地一致
- [ ] `GOWORK=off go vet` 仍 PASS（推送不改代码）

## Notes

- 当前 `origin` 为 `Ponphil/LitePan`，`main` 已 `70ee23d` 等 15+ 本地 commits 领先，需新建 `zhemed/LitePan` 承接
- 若用户后续需将 `Ponphil/LitePan` 设为 `upstream`，可 `git remote rename origin upstream && git remote add origin https://github.com/zhemed/LitePan.git`
