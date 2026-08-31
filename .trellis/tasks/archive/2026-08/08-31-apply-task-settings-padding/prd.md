# 填充任务设置单项

## Goal

按 `report.md` 方案A，将 `UploadTaskSettingsPanel` 单项 `任务并发` 全宽居中填充，`− 5 +` 格式不动，发布 `0.0.8`。

## Background

- 报告：`grid 1fr 1fr` 单项左置空白，需 `1fr` 单列居中。

## Requirements

- **CSS**：`UploadTaskSettingsPanel.vue` `193-197` `grid-template-columns:1fr 1fr` → 单项时 `1fr` 居中，`item--stepper:only-child { max-width:420px; margin:0 auto; grid-column:1/-1 }` 或等价 `grid:has`。
- **不改**：`− 5 +` 步进器结构、文案、逻辑。
- **版本**：`README v0.0.7→v0.0.8`，`docker 0.0.8`。

## Constraints

- 仅 `web` CSS，`go vet` 无影响，`task.py start` 后执行。

## Acceptance Criteria

- [ ] 单项时居中全宽，左右对称，不改并发控件
- [ ] `cd web && npm run type-check 0` `npm run build` 成功，`docker build` 成功

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.
