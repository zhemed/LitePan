# 调查GitHub与本地差距

## Goal

对比 `https://github.com/zhemed/LitePan` 远端 `main` 与本地 `/root/LitePan` 工作区，定位差距（commits/tags/文件/镜像）并给出同步建议。

## Background

- 刚经历 `0.0.10 跨盘下载移除` 的 `reset --hard` 同步，工作区现 `1a77f58` 与远端 `github/main` 一致，但仍有 `2` 个 `untracked` 任务目录需核查。
- 用户要求建立新任务调查差距。

## Requirements

- **元数据**：`git log HEAD..github/main` / `github/main..HEAD` / `git status` / `git diff` / `gh repo view` / `gh release list` / `gh api packages` / `docker images` 对比。
- **文件**：`git ls-remote` vs 本地 `HEAD`，`raw` 文件 `sha256` 对比，`untracked` 任务目录说明。
- **版本**：`v0.0.10` tag 一致性，`GHCR` `0.0.10` digest 一致性。
- **产出**：`report.md` 含差距表与结论。

## Constraints

- 只读，不改远端；写仅限 `report.md`。
- 遵循 `AGENTS.md`。

## Acceptance Criteria

- [ ] `report.md` 存在，含差距表与是否一致结论

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.
