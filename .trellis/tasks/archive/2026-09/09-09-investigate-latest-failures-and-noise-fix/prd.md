# investigate-latest-failures-and-noise-fix

## Goal

调查最新失败原因（区分 0.0.20 部署前/后、新错误模式 vs 既有瞬时类）并处置：manager.injectAuth 认证加载 WARN 增加 ctx.Err() 短路降噪（1 行）；必要时错误重试是否生效一并核实；0.0.21 发布

## Requirements

- TBD

## Acceptance Criteria

- [ ] TBD

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.
