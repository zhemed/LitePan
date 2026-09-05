# 调查上传任务重试次数与重试规则

## Goal

调查本地 `LitePan` 工作区上传任务（`internal/upload` + `drivers/*` + `web` 上传面板）是否存在重试次数与重试规则，并输出完整证据链与结论。

## Background

- 用户要求只读调查：上传失败是否自动重试、重试几次、间隔/退避规则、哪些错误可重试、哪里可配置。
- 范围限 `/root/LitePan` 本地工作区，不涉及 `LitePan-own` 与远端。

## Requirements

- **后端扫描**：`internal/upload`（`manager.go/worker.go/queue.go/lifecycle.go/persist.go/types.go`）、`drivers/*` 上传路径、`internal/settings` 重试相关键。
- **判定维度**：是否存在 `maxRetries/Retry/attempt` 计数、退避（`backoff/sleep/ticker`）、可重试错误分类、并发/队列影响、是否可配置。
- **前端扫描**：`web/src` 上传面板重试按钮/状态展示、手动重试入口。
- **产出**：`report.md` 含文件:行证据、规则表、结论（有/无/部分）与改进建议。

## Constraints

- 只读调查，不改业务代码；写仅限任务目录 `report.md`。
- 遵循 `AGENTS.md 项目强制规则`：`task.py start` 后执行。

## Acceptance Criteria

- [ ] `report.md` 存在，含后端/前端/配置三节与结论
- [ ] 每条结论有文件:行来源，可复现 `grep` 命令

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.
