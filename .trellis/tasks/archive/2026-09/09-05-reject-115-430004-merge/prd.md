# 否决430004合入并归档结论

## Goal

记录否决b2ecd58两行映射合入的决定：STRM已删无执行器、无115账号，零收益，避免重复调查

## Requirements

1. 落文字结论：否决上游 `b2ecd58` 中 `drivers/115_Open/transport.go` 的 `430004→CodeNotFound` 2 行合入。
2. 否决理由（用户拍板）：STRM 模块已整体移除，该映射的最大执行器（扫描跳过+防误删 130 行）不存在；全库无 115 账号，驱动无触发方；本地仅剩删除幂等一路边角收益，总体零收益。
3. 记录重开条件：绑 115 账号且出现真实 complaint，或重引入消费 `CodeNotFound` 的 115 链路时，可重开（补丁 `apply --check` 已验证，随时可合）。
4. 纯文档任务：不改业务代码，只写 `report.md` 后归档。

## Acceptance Criteria

- [ ] `report.md` 含决定、理由、重开条件
- [ ] 任务归档，`github/main` 同步
- [ ] 业务代码零改动

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.
