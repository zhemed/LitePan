# 调查任务设置填充

## Goal

以截图 `b33a32 1583x719`（单项 `任务并发 5` 稀疏）与 `2ffaf0 895x414`（旧 4 项 `并发/限速/磁力/临时目录` 充实）为对比，调查 `UploadTaskSettingsPanel` 仅 `任务并发` 单项时是否可通过填充/布局优化避免不协调，并给出不改“并发格式”（`− 5 +`）的方案。

## Background

- `0.0.7` 已删 `builtin_offline` 3 键，`UploadTaskSettingsPanel` 从 `4 项`（`并发+限速+磁力+临时目录`）精简至 `1 项`（`任务并发`），`grid: 1fr 1fr` 导致单卡片左置、右侧大面积空白，不协调。
- 用户提供例子仅作填充参考，明确“并发格式不需要修改，已标注原有格式”。
- 需评估填充方式：空白占位、说明卡、居中、增宽、或新增中性信息卡。

## Requirements

- **现状**：`web/src/components/upload/UploadTaskSettingsPanel.vue` 当前 `template` 仅 `1 项` `task-settings__item--stepper`，`grid 2 列`，`width 720px`。
- **例子**：`2ffaf0` 中 `并发+限速` 同行、`磁力` 宽卡、`临时目录` 宽卡 的布局可作填充密度参考，但并发控件本身 `− 5 +` 不动。
- **方案**：提出 2-3 种填充方案（如：① 单项居中+增宽+说明 ② 新增“说明/提示”占位卡 ③ 将 `grid` 改为 `1 列` 并增 `min-height`），评估 `vue-tsc` 与 `Go` 无影响，仅 `web` CSS。
- **产出**：`report.md` 含现状截图描述、3 方案对比（效果/改动/风险）、推荐。

## Constraints

- 只读调查，不改代码；写仅限 `report.md`。
- 遵循 `AGENTS.md`：`task.py start` 后执行。
- 不改“并发”控件结构，仅布局/填充。

## Acceptance Criteria

- [ ] `report.md` 存在，含现状、例子对比、3 方案与推荐

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.
