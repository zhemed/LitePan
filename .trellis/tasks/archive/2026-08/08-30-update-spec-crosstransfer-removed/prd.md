# Update spec for crosstransfer removal

## Goal

将 `.trellis/spec` 中仍描述跨盘秒传（`internal/crosstransfer` / `CrossDriveTransfer`）的章节，更新为与 `litepan-go:nocross 119M` 现状一致：跨盘秒传已彻底移除。

## Requirements

- **后端 `spec/backend/backend` 7 篇**：
  - `directory-structure.md`：`internal/crosstransfer` 标注为 `已移除（2026-08-30 nocross）`，`crosstransfer` 从 `internal/` 拥有表移除
  - `api-layering.md`：`Deps` 注入表删 `CrossTransfer`，`Route("/cross-transfer", 5 handler)` 标注已移除，仅保留 `local-upload` 等
  - `database-guidelines.md`：如曾列 `crosstransfer` 表则标注无专有表，`upload_tasks source_type=cross_transfer` 保留历史兼容
  - `quality-guidelines.md`：`grep` 门禁示例中 `crosstransfer|cross-transfer|CrossDrive` 标注为已删
- **前端 `spec/web/frontend` 5 篇**：
  - `directory-structure.md`：`CrossDriveTransfer.vue / CrossTransferTree.vue` 标注已移除，`AdminView` 的 `cross-transfer` 导航已删
  - `api-client.md`：`crossTransfer.ts` 5 导出标注已删
  - `component-guidelines.md`：跨盘相关卡片一节更新

## Constraints

- 仅改 `.trellis/spec` 下 Markdown，不改 `internal/*` / `web/*` 代码
- 保持 7 章节结构完整

## Acceptance Criteria

- [ ] `grep -r "CrossDriveTransfer|crossTransfer|crosstransfer" .trellis/spec --include="*.md" | wc -l` == 0（或仅在“已移除”备注中）
- [ ] `cat .trellis/spec/backend/backend/api-layering.md | grep -A2 "Deps"` 不含 `CrossTransfer`
- [ ] `python .trellis/scripts/task.py archive` 成功

## Notes

- 触发：`08-30-remove-crosstransfer` 已 `go vet 0 / build 33M / docker 119M / /cross-transfer 404`，但 `spec` 仍为精简前快照
