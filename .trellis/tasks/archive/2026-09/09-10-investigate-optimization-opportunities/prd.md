# investigate-optimization-opportunities

## Goal

调查上传系统还有哪些优化空间：列表 API 5.4MB 全量载荷（字段构成/分页/过滤/汇总）、前端 fetchUploadTasks 热路径（7814 条多次全量遍历）、SSE 增量与快照体积、后端 List/persist/DB 索引、渲染路径；给出量化收益与优先级建议

## Requirements

- TBD

## Acceptance Criteria

- [x] 实测取证：列表 5.41MB/95ms、SSE 首帧 5.41MB 快照、字段占比、DB 索引齐备
- [x] 可优化项排序：①前端计算（徽标 4 次全量 filter/批量暂停 N-patch/树重建）②载荷瘦身（cleanup_local_path 0 引用+result 去重，-11%）③分页过滤汇总（5.41MB→0.3-0.6MB）④工作集治理
- [x] 不建议动：吞吐（已达设计上限）、序列化、DB

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.
