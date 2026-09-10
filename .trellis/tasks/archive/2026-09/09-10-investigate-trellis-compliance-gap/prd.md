# investigate-trellis-compliance-gap

## Goal

调查本会话（2026-09-09 ~ 09-10）未**严格**执行 trellis 强制流程的原因，并给出可执行的纠正措施。只调查不改码（除本任务档案本身）。

## Background

项目规则（AGENTS.md）明确：任何写操作必须走
`skill trellis-start → task.py create → prd/design/implement → task.py start → trellis-check → task.py archive → add_session.py`。
用户观察到执行不严格，要求查明原因。

## Requirements

1. 以**权威来源**为准逐条对照：`AGENTS.md` 规则条文、`.trellis/workflow.md` 阶段定义、`trellis-start` / `trellis-check` 技能内容（skill 载入为只读证据）。
2. 取证实际操作偏差（可复现的证据，不靠回忆）：
   - PRD 是否先于 `task.py start`（文件时间戳 vs 归档时间）
   - `design.md` / `implement.md` 是否存在（复杂任务的强制产物）
   - 是否调用过 `trellis-start` / `trellis-check` 技能
   - 会话编号与 journal 的一致性
3. 根因分析（机制层，不写"疏忽"了事）：为什么 CLI 路径会绕过技能与阶段门。
4. 纠正措施：可落地、可验证（含"如何防止再次发生"的机制建议）。

## Constraints

- 只读调查；除本任务档案外不改任何代码/配置。
- 结论必须有证据支撑，无法验证的部分明确标注。

## Acceptance Criteria

- [x] 权威要求逐条列出（AGENTS.md 条文 + workflow.md:167/205/218/66 + 两个 skill 关键步骤）
- [x] 偏差清单 7 项 + 证据（design/implement 0/42、PRD mtime=归档时刻、jsonl 0 个、dev_type/scope 未设、会话号漂 1）
- [x] 根因 5 条（含关键发现：恢复用的流程定义本身就缺 trellis-start/trellis-check 两步）
- [x] 纠正措施 A-F（A 会话开始载入上下文/B 先 PRD 后 start + set-scope/C 归档前跑 check/D 流程门脚本/E 写入规范/F 会话号口径），D/E 需用户同意
- [x] 质量门已跑（vet 全绿、全模块零失败、类型检查通过；本任务无代码变更）；归档 + journal + push
