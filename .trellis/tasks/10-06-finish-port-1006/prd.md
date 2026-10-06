# PRD：移植收尾（归档提交与推送）

## Goal

承接 2026-10-06 两个任务的**记账提交**并推送，使仓库回到干净且与 `origin/main` 同步的状态：

1. `10-06-port-upstream-1006`（移植上游 4 项修复，代码提交 `548b56b2`）与
   `10-06-upstream-followup-1006`（调查收尾与拍板）的**归档移动**（文件已改名暂存，未提交）；
2. 本次移植的 journal 条目；
3. `git push origin main`（用户已授权）。

## Background

- 代码与测试已在 `548b56b2` 落地并通过全量门（lint 0 / vet exit=0 / 26 包 ok / type-check 0 /
  deadcode 7 / unused 0 / gofmt 15 既有）。
- 闸门口径（本轮实证，写入 journal）：任务已全部归档时，闸门会因"本轮无进行中任务"拒绝任何写操作，
  因此**归档提交必须在下一个进行中任务名下补交**；本任务即为该承接任务。
- 本任务**保持进行中**（不归档）：归档它自身会再次触发同一死锁；下一轮可由当时的任务一并归档。

## Requirements

- R1：提交两个任务的归档移动（`.trellis/tasks/archive/2026-10/**`），提交消息带本任务锚点。
- R2：用 `add_session.py --no-commit` 写本次移植的 journal 条目，再手工提交（含锚点）。
- R3：`git push origin main`，并回读 `origin/main` 与本地 `main` 一致。

## Acceptance Criteria

- [ ] 两个任务的归档移动已提交，工作区无未跟踪/未提交的 `.trellis/**` 改动
- [ ] journal 条目已写入且已提交
- [ ] `main == origin/main`（推送完成）
- [ ] 零代码改动（本任务只做记账与推送）

## Notes

- Scope `lightweight`（记账/推送）。
- 本任务不归档（原因见 Background）。
