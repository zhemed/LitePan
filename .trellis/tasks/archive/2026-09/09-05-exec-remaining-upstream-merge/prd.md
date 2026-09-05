# 执行剩余上游合入并分批发版

## Goal

父任务：先合入c7a424c认证修复发0.0.14，再合入A路文件夹上传链发0.0.15

## Requirements

1. 子任务1（先）：合入 `c7a424c` 认证修复，按调查报告排除清单挑拣，发 `0.0.14`。
2. 子任务2（后，以子任务1已合入的 main 为基）：合入 `de83b46` 的 A路文件夹上传链，发 `0.0.15`。B路（任务树/SSE）不在本父任务范围。
3. 每个子任务独立走完：合入→构建→测试→bump→打tag→推GHCR→推github→本地部署验证→归档。

## Acceptance Criteria

- [ ] `v0.0.14`：认证修复在线，GHCR/github 同步，容器健康
- [ ] `v0.0.15`：文件夹上传链在线，GHCR/github 同步，容器健康

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.
