# Clone LitePan-own into workspace

## Goal

在 `/root/LitePan` 工作区内创建单独目录（与当前 `main` 分支代码隔离），完整拉取 `https://github.com/zhemed/LitePan-own`，使其与已有的 `LitePan-own`（`zhemed/LitePan` 精简版）并存，便于对比与后续合并。

## Requirements

- **目录**：在 `/root/LitePan` 下创建 `LitePan-own`（或 `_external/LitePan-own`）单独目录，不污染 `main` 的 `drivers/all.go` 等已精简文件
- **拉取**：`git clone https://github.com/zhemed/LitePan-own.git <...>/LitePan-own`，`--depth 1` 可选，`gh` 已登录 `zhemed` 无需额外 `token`
- **隔离**：该目录为嵌套 `git` 仓库（`.git` 独立），父仓库 `.gitignore` 需忽略它（追加 `LitePan-own/` 或 `_external/`），避免 `git status` 误报
- **验证**：`ls <...>/LitePan-own` 含 `README.md`、`drivers/`、`internal/` 且 `git -C <...>/LitePan-own remote -v` 指向 `zhemed/LitePan-own`

## Constraints

- 仅在 `/root/LitePan` 内新增目录，不改 `drivers/all.go` 等已提交的 `118M` 精简内容
- 若 `/root/LitePan/LitePan-own` 已存在则先 `rm -rf` 或提示用户确认覆盖
- 操作经 Trellis（`task.py create/start/archive`）以满足 `AGENTS.md` 强制

## Acceptance Criteria

- [ ] `ls /root/LitePan/LitePan-own` 存在且 `cat .../README.md | head` 含 `LitePan-own`
- [ ] `git -C /root/LitePan/LitePan-own remote -v | grep zhemed/LitePan-own` 有输出
- [ ] `git -C /root/LitePan status --porcelain | grep LitePan-own` == 0（已被 `.gitignore` 忽略）
- [ ] `ls /root/LitePan-own` 仍为原有 sibling 克隆，不冲突

## Notes

- `LitePan-own` 为用户在 `zhemed` 下的另一精简版（与 `zhemed/LitePan` 本次新建的 `three-drivers` 不同），拉取后可用于对比 `drivers` 差异
- 本任务为轻量 infra，无需 `design.md`
