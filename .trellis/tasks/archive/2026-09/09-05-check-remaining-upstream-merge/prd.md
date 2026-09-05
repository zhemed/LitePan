# 调查剩余上游提交能否安全合并

## Goal

父任务：逐个评估c7a424c认证修复与de83b46上传批次化能否安全合入本地，给出合入/改写/放弃结论

## Requirements

1. 子任务1（优先）：评估 `c7a424c`（09-04《修复一系列认证问题》）中 `115_Open` / `189Cloud` 认证改动能否安全合入。
2. 子任务2（随后）：评估 `de83b46`（08-30《上传任务批次化》）能否安全合入，重点是 DB migration `0022`、SSE、任务树前端与本地上传精简的冲突。
3. 两个子任务均为只读调查：`git show/diff/apply --check/grep`，不合入、不改业务代码。
4. 每个子任务独立输出 `report.md`（结论 + 风险等级 + 合入建议）；父任务只做汇总，不直接实施。

## Acceptance Criteria

- [ ] 子任务1有明确结论（可合入 / 需改写 / 放弃）及风险等级
- [ ] 子任务2有明确结论及风险等级
- [ ] 全程只读，业务代码零改动

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.
