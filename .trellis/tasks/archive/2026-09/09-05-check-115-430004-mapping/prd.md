# 调查115错码430004映射NotFound的作用

## Goal

查b2ecd58两行映射在transport错误链中的位置、有无时的行为差异、下游对CodeNotFound的处理，给出合并不合的依据

## Requirements

1. 还原上游 `b2ecd58` 在 `drivers/115_Open/transport.go` 的 2 行上下文：`430004` 在 115 错误码分支中的位置、相邻分支的映射惯例。
2. 追踪本地无此映射时的兜底路径：默认分支返回什么 code，`CodeNotFound` 与兜底 code 在下游（文件列取/删除/上传建目录/重试/前端提示）各走什么分支，给出行为差异表。
3. 查作者提交语境（同提交其他文件、提交说明）判断 430004 命中的真实场景（删云端已删文件？列取失效目录？STRM 链路？）。
4. 给出结论：合（收益/风险）或不合（理由）；只读，不合入、不改代码。

## Acceptance Criteria

- [ ] 映射位置与相邻惯例说明
- [ ] 有/无映射行为差异（下游分支级）
- [ ] 触发场景判定（含未能独立验证的标注）
- [ ] 合并建议与风险，全程只读零改动

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.
