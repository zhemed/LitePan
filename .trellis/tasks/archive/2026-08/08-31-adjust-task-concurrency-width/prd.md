# 调整任务并发宽度

## Goal

以截图 `5f4f9b 915x233`（红框左右留白窄，灰卡接近 `915` 满宽）为准，将 `0.0.8` 的 `max 420px` 调至接近两图中灰卡满宽，左右留白与图中红框一致，`− 5 +` 格式不动，发布 `0.0.9`。

## Background

- `0.0.8` 将单项限 `420px` 居中，左右各 `~150px` 留白，用户框选显示左右红框窄（约 `18px` 内边距），需灰卡接近容器满宽。
- 容器 `720px`，`padding 18px`，灰卡应 `~684px` 而非 `420px`。

## Requirements

- **CSS**：`UploadTaskSettingsPanel.vue` `only-child` 的 `max-width/max 420px` → `width 100%` / `max-width none`，`grid 1fr` 保持，左右通过 `task-settings__body padding 18px` 自然留白。
- **不改**：`− 5 +` 步进器。
- **版本**：`README v0.0.8→v0.0.9`，`docker 0.0.9`。

## Constraints

- 仅 `web` CSS，`task.py start` 后执行。

## Acceptance Criteria

- [ ] 单项灰卡左右留白窄，与截图红框一致，`− 5 +` 不动
- [ ] `type-check 0` `build` 成功 `docker 0.0.9`

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.
