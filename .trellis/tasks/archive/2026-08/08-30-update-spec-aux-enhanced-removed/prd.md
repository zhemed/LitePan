# Update spec for aux enhanced tools removal

## Goal

将 `.trellis/spec` 中仍描述 7 项已删增强工具（Emby 反代、飞牛影视反代、夸克 TV 接管、AI 辅助识别、目录整理分类、垃圾清理工具、视频海报生成）的章节，更新为与 `litepan-go:aux-keep-upload 119M` 现状一致：增强工具仅保留“从服务器上传”（`LocalUpload`），其余标记为已移除。

## Requirements

- **后端 `spec/backend/backend` 7 篇**：
  - `directory-structure.md`：`internal/` 列表中 `embyproxy/fnosproxy/quarktv/spacecleanup/coverextract` 标记为 `已移除（2026-08-30 aux-keep-upload）`，`internal/` 拥有表同步删
  - `api-layering.md`：`Deps` 注入表删 `EmbyProxy/FnosProxy/QuarkTV/SpaceCleanup/CoverExtract`，`Route("/tools/*")` 列表仅留 `local-upload`，`GET /internal/cover-source/{token}` 标注已删
  - `database-guidelines.md`：`quarktv_bindings` 表（如曾列）标注保留建表 SQL 但不再读写（与 `cache_retention` 一致）
  - `quality-guidelines.md`：`grep` 门禁示例中 `embyproxy|quarktv|...` 标注为已删增强工具，`local_upload` 为唯一保留
- **前端 `spec/web/frontend` 5 篇**：
  - `directory-structure.md`：`src/components/admin/CloudToolsPanel.vue` 的 `cardTitles` 8→1，列 `ProxyToolsPanel/QuarkTV/AI/Classification/Cleanup/CoverExtract` 为已移除，仅 `LocalUploadToolCard` 保留
  - `api-client.md`：`cloudTools.ts` 仅 `localUploadApi`，`coverExtract.ts/emby.ts/fnos.ts` 标注已删
  - `component-guidelines.md`：增强工具卡片一节更新为单卡片
- **共享 `guides/index.md`**：若含增强工具检查项，指向新 spec

## Constraints

- 仅改 `.trellis/spec` 下 Markdown，不改 `internal/*` / `web/*` 代码
- 保持 `spec` 的 7 章节结构（Scope/Trigger/Signatures/Contracts/Validation/Good-Bad/Tests/Wrong-Correct）对跨层契约处补齐
- 不新增 `internal/api/web` 资产

## Acceptance Criteria

- [ ] `grep -r "Emby 反代|飞牛影视|夸克 TV|AI 辅助识别|目录整理分类|垃圾清理|视频海报" .trellis/spec --include="*.md" | wc -l` == 0（或仅出现在“已移除”备注中）
- [ ] `cat .trellis/spec/backend/backend/api-layering.md | grep -A2 "Deps"` 不含 `EmbyProxy/FnosProxy/QuarkTV/SpaceCleanup/CoverExtract`
- [ ] `cat .trellis/spec/web/frontend/api-client.md | grep cloudTools` 仅 `localUploadApi`
- [ ] `python .trellis/scripts/task.py archive` 成功，`journal Session` 记录

## Notes

- 触发：`08-30-remove-aux-enhanced-keep-upload` 已 `go vet 0 / build 33M / docker 119M / local-upload 200 / cover-extract 404`，但 `spec` 仍为精简前快照
- 本任务为文档追齐，不涉及 `data/litepan.db` 与 `admin/123456`
