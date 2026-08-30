# Update spec for drivers keep only three

## Goal

将 `.trellis/spec` 中驱动相关章节从 11 驱动更新为与 `litepan-go:three-drivers 118M` 现状一致：仅保留 `115_Open`、`189Cloud`、`LocalFs`，其余 8 驱动标记为已移除。

## Requirements

- **后端 `spec/backend/backend` 7 篇**：
  - `directory-structure.md`：`drivers/` 列表 11→3，仅 `115_Open 189Cloud LocalFs + all.go + template`，其余 8 标注 `已移除（2026-08-30 three-drivers）`
  - `driver-development.md`：`Pluggable drivers under drivers/` 列表 11→3，`Config` 示例保留，`Adding a New Driver` 章节保留 `template` 为基
  - `api-layering.md`：`GET /admin/drivers` 示例返回 3 项标注
- **前端 `spec/web/frontend` 5 篇**：
  - `directory-structure.md`：驱动卡片相关描述同步为 3
- **共享 `guides/index.md`**：驱动检查项同步为 3

## Constraints

- 仅改 `.trellis/spec` 下 Markdown，不改 `drivers/` 代码
- 保持 7 章节结构完整

## Acceptance Criteria

- [ ] `grep -r "123_Open|139Cloud|Baidu_Open|Quark|WebDAV" .trellis/spec --include="*.md" | wc -l` == 0（或仅在“已移除”备注中）
- [ ] `cat .trellis/spec/backend/backend/driver-development.md | grep -A2 "Pluggable drivers"` 仅 3 驱动
- [ ] `python .trellis/scripts/task.py archive` 成功

## Notes

- 触发：`08-30-keep-only-three-drivers` 已 `go vet 0 / build 32M / docker 118M / drivers 3`，但 `spec` 仍为 11 驱动快照
