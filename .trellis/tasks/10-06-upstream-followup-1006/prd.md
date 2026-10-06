# 上游调查收尾与移植拍板（2026-10-06）

## Goal

1. **收尾留痕**：把本次只读调查（任务 `10-06-upstream-updates-1006`）的归档与 journal 落成提交 ——
   该任务的归档提交被 Trellis 闸门拦下（归档完成后无"进行中任务"，闸门因此拒绝 `git commit`），
   需要在本任务（进行中）名下补交。
2. **移植范围拍板**：把调查报告中 4 项"适用"修复交给用户选择要不要做、做哪些；
   项目随后按选择另开移植任务（本任务**不实施**任何代码改动）。

## Background

- 调查结论（`.trellis/tasks/archive/2026-10/10-06-upstream-updates-1006/research.md`）：
  上游 `42a3ee9a` → `e0e29c0e`（21 提交，无 merge、未重写历史）中，
  **适用 4 项**、**待拍板 1 项（2FA）**、混合 1 项、不适用 15 项。
- 适用 4 项：`7e0c0040` 上传强制 HTTP/1.1（`httpx.NewUploadClient` + `driver.Config.UploadUseHTTP2`）、
  `5df900c5` playback 多段 Range + Range 诊断日志、`214e027d` 189Cloud 同步盘根（`syncRootID`）、
  `52fcf32e` 日志详情恒返回（后端一行 + 前端两文件）。
- 待拍板：`5db84a63` 后台 2FA（TOTP，~1200 行含前端，改登录主链路）。
- 现场：`main = 778f5b87`（v0.0.48 已发布部署）；本任务开始前工作区仅调查任务目录（现已归档、处于已暂存状态）。

## Requirements

### R1 补交调查归档与 journal

- 提交 `.trellis/tasks/archive/2026-10/10-06-upstream-updates-1006/**`（`prd.md`、`research.md`、
  `task.json`、`.check-passed`），提交消息带 `[task:<本任务 slug>]` 与调查任务 slug 说明。
- 用 `add_session.py` 记录本次调查的 journal（Session 条目含结论摘要与验证项）。

### R2 把移植选择交给用户

- 用交互式提问组件（`ask_user_question`）给出候选：全做 / 只做优先级 1（上传 HTTP/1.1 + 日志详情）/
  先只做 playback 多段 Range / 都不做（仅留档）。
- 每个候选一句话说明影响；推荐项排第一并标注「（推荐）」。
- 用户选择后：**另开移植任务**（本任务只负责记录决定）。若用户选择"都不做"，则把决定写入本任务与 journal 后归档。

### R3 边界

- 本轮**不改任何代码/配置/部署/数据**，不发版、不动 `:5211` 容器与 `data/`。
- 网络动作：仅 `github.com`（`git push` 记账提交，若用户授权）。

## Acceptance Criteria

- [ ] 调查任务的归档文件已提交（提交消息含本任务 slug）
- [ ] `add_session.py` 已写入本次调查的 journal 条目（含 21 提交分类结论与零改动声明）
- [ ] 已用 `ask_user_question` 向用户提出移植范围选择，并记录用户的选择
- [ ] 零代码改动：`git status` 除 `.trellis/**` 外无改动
- [ ] 本任务归档（若用户已给出选择且无后续动作）；或保持进行中等候用户选择（在 journal 写明）

## Notes

- Scope `lightweight`（收尾与拍板，PRD-only）。
- 闸门现象记录：`task.py archive` 完成后闸门立即要求"进行中任务"，导致归档提交必须在下一次任务里补交
  （本次即为补交）。该现象写入 journal，供后续同类收尾参考。
