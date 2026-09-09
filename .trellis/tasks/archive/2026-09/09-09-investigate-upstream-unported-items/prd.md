# investigate-upstream-unported-items

## Goal

调查两项上游未移植项的移植可行性与价值：(1) internal/auth 守卫接线——SetAuthGuards/内联刷新冷却/网络熔断/HandleRetryFailure 的完整机制与本方被动刷新的差距；(2) de83b46 上传批次化(任务树分层/大批量渲染优化)与本方精简 worker 的冲突面。产出逐项 A/B/C 结论与建议

## Requirements

- TBD

## Acceptance Criteria

- [x] 守卫接线机制拆解 + 移植面清单（整取/手改/无需三类）→ research.md 项 1
- [x] de83b46 真实源码面统计（~30 文件）与冲突实证 → research.md 项 2
- [x] 逐项 A/B/C 结论：守卫接线 B 建议移植；批次化 C 暂不移植 → research.md 总建议
- [ ] 归档 + journal + push

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.
