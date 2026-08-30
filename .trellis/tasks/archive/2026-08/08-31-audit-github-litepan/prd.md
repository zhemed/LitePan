# 审核 GitHub LitePan

## Goal

对 `https://github.com/zhemed/LitePan` 远端做只读巡检，核对与本地 `main 66337e3 v0.0.3` 的一致性，涵盖分支、文件、README、Release、Tag、GHCR、Actions 与可见性。

## Background

- 本地已 `v0.0.3` 且 `push github main`，但远端 `README` 渲染、`Release` 描述、`GHCR` `latest/0.0.3` 是否同步、`docker-compose.yml` 是否已落地 `ghcr`、`install-docker.sh` 是否可 `curl`，需远端实地验证。
- 用户刚完成本地工作区整理，要求同样审计远端。

## Requirements

- **远端元数据**：`gh repo view zhemed/LitePan --json name,visibility,defaultBranch,updatedAt`、`gh api repos/zhemed/LitePan --jq .size/.pushed_at`、`git ls-remote https://github.com/zhemed/LitePan.git`。
- **文件**：`gh api repos/zhemed/LitePan/contents/README.md` 与 `raw.githubusercontent.com` `README` `docker-compose.yml` `install-docker.sh` HEAD 可达性与标签一致性（`v0.0.3`）。
- **发布**：`gh release list --repo zhemed/LitePan` `tag v0.0.3/v0.0.2/v0.0.1` `gh api repos/zhemed/LitePan/releases/tags/v0.0.3`、发布时间、产物。
- **镜像**：`gh api users/zhemed/packages/container/litepan/versions --jq tags` `0.0.3/v0.0.3/latest` `sha256` 与本地 `docker images` 一致，`public` 可见性。
- **一致性**：`git -C LitePan fetch --dry-run` 是否落后、`git log --oneline github/main..HEAD` 是否 0、`raw` 文件 `sha` 与本地 `sha256sum` 对比。

## Constraints

- 只读 `gh api / curl raw`，不 `push --force`、不改远端。
- 遵循 `AGENTS.md`：`task.py start` 后执行。

## Acceptance Criteria

- [ ] `report.md` 存在，含五节（元数据、文件、发布、镜像、一致性）与结论
- [ ] 每项有来源（`gh api / curl` 输出）与时间戳
- [ ] 给出远端是否与本地 `v0.0.3` 100% 一致的结论

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.
