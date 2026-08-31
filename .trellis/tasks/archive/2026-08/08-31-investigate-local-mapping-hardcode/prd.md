# 调查本地映射多选是否为硬编码

## Goal

以截图 `24500b 137x39`（`本地映射（多选）`）为靶，核查 `web/src/components/admin/AutomationPanel.vue` 的映射选项是硬编码还是动态拉取，并追溯 `2b44969/283b875` 变更。

## Background

- 用户疑似看到 `本地映射（多选）` 仍为 3 固定值，怀疑硬编码。
- 现状 `0.0.5` 已由硬编码 `['我的文件','杂物间','pve_backup']` 动态化为 `fetchLocalMappings()` 拉 `GET /api/admin/tools/local-upload/config`。

## Requirements

- **现状**：`grep localMappingOptions` 在 `AutomationPanel.vue` 的定义（`ref([])` vs 常量数组）、`fetchLocalMappings` 实现、`openConfig` 是否 `void fetchLocalMappings()`。
- **历史**：`git log --oneline --grep=mapping` 与 `git show 283b875/2b44969` 对比硬编码 → checkbox → AppSelect multiple 动态。
- **后端**：`GET /api/admin/tools/local-upload/config` 来源 `settings KeyLocalUploadMappings`，是否动态。
- **产出**：`report.md` 含现状代码行、历史演进表、是否硬编码结论。

## Constraints

- 只读，不改代码；写仅限 `report.md`。
- 遵循 `AGENTS.md`：`task.py start` 后执行。

## Acceptance Criteria

- [ ] `report.md` 存在，含现状、历史、后端三表与结论（是/否硬编码）

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.
