# investigate-pause-mass-failures

## Goal

调查大批量上传暂停后导致大批量失败的原因：取证 DB 中 paused/failed 任务时间线与错误模式、日志中暂停/恢复事件与取消错误、暂停实现（runner ctx 取消→任务被标 failed?）与恢复路径（189 resumeState/批根目录），定位根因并给出结论

## Requirements

- TBD

## Acceptance Criteria

- [x] 根因：账号网络冷却 × worker 无冷却感知（零 I/O 空转判死），451 失败/秒实证
- [x] 暂停排除：5 次中途暂停实验均未触发冷却；时间线显示暂停与风暴同秒发生（误读）
- [x] 恢复验证：失败任务重传双双成功；冷却触发源无日志可归因（如实说明）
- [x] 修复建议三步：worker 冷却等待重试 / IsNetworkError 去自指 / 批次熔断安全网

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.
