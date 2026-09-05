# 审计上游更新与所需合入完整性

## Goal

fetch最新后逐提交核对20+上游提交的已合入/部分合入/有意跳过状态，验证353b830/c7a424c/de83b46所需项无遗漏

## Requirements

1. `git fetch origin`（只读）确认作者有无新增提交；列出分叉点以来上游全部提交。
2. 逐提交给出状态：已合入（含本地版本）/ 部分合入（说明哪部分、落在哪版）/ 有意跳过（说明原因：模块已删/无关）/ 待定（需用户定夺）。
3. 实证抽查"所需项"：关键符号与行为在本地存在（`mappingIndex` 越界保护、`AuthRefreshControl`、`BatchPause` 路由、`uploadTaskTree` 接线、`migration 0022`、SSE `kind` 字段等），不只看提交记录。
4. 输出遗漏清单（若有）与建议；只读，不合入、不改代码。

## Acceptance Criteria

- [ ] 上游提交全表（含 fetch 后新增）状态齐备
- [ ] 所需项逐项实证存在
- [ ] 遗漏/待定项明确列出或确认为零
- [ ] 全程只读，业务代码零改动

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.
