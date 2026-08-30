# Remove aux enhanced tools keep server upload

## Goal

辅助工具（`AuxToolsManagement.vue`）→ **仅增强工具**（`CloudToolsPanel.vue`）中 **仅保留“从服务器上传”（`LocalUploadToolCard.vue` + `internal/api/local_upload.go` + `settings KeyLocalUpload*`）**，将其余 7 项增强工具的前后台能力彻底移除；**备份管理（`BackupRestorePanel` / `internal/backuprestore`）本次不动**，与增强工具完全隔离。用户已确认：**彻底删除前后端**（非仅隐藏前端），且**本次仅增强工具**。

## Background

- 当前 `CloudToolsPanel.vue` 的 `cardTitles` 硬编码 8 项：`["Emby 反代", "飞牛影视反代", "夸克 TV 接管", "AI 辅助识别", "目录整理分类", "从服务器上传", "垃圾清理工具", "视频海报生成"]`，分别对应 `ProxyToolsPanel / QuarkTVToolCard / AIToolCard / ClassificationToolCard / LocalUploadToolCard / CleanupToolCard / CoverExtractToolCard`
- 后端对应：`internal/embyproxy`、`internal/fnosproxy`、`internal/quarktv`、`internal/spacecleanup`、`internal/coverextract` 仍在 `wire_services/wire_http` 中装配；`internal/aiorganize / classifyorganize` 已在 `1bcfac8` 删除但前端 `AIToolCard / ClassificationToolCard` 残留

## Requirements

- **保留**：
  - 前端：`LocalUploadToolCard.vue`（含 `matches('从服务器上传')` 搜索、`localUploadApi`）及其在 `CloudToolsPanel.vue` 中的唯一引用
  - 后端：`internal/api/local_upload.go` 4 个 handler（`getLocalUploadConfig / updateLocalUploadConfig / browseLocalUpload / createLocalUploadTasks`）与 `router.go` 的 `Route("/tools/local-upload", ...)`，及 `settings KeyLocalUploadEnabled/KeyLocalUploadMappings`
  - 辅助工具的 `备份管理` Tab（`BackupRestorePanel`）不受影响
- **彻底移除（7 项）**：
  - `Emby 反代 + 飞牛影视反代` → `ProxyToolsPanel.vue / ProxyWorkspace.vue / TmdbHostsHelpTip.vue` + `internal/embyproxy + internal/fnosproxy` + `settings` 中 `emby/* fnos/* KeyEmby* KeyFnos*` + `api/emby.go fnos.go proxy_tools.go` 等路由
  - `夸克 TV 接管` → `QuarkTVToolCard.vue + QuarkTVBindModal.vue` + `internal/quarktv` + `api/quarktv.go` + `router /tools/quarktv`
  - `AI 辅助识别` → `AIToolCard.vue`（前端残留，`internal/aiorganize` 已删）
  - `目录整理分类` → `ClassificationToolCard.vue`（前端残留，`internal/classifyorganize` 已删）
  - `垃圾清理工具` → `CleanupToolCard.vue + CloudToolCard.vue` + `internal/spacecleanup` + `api/space_cleanup.go` + `router /tools/cleanup`
  - `视频海报生成` → `CoverExtractToolCard.vue` + `internal/coverextract` + `api/cover_extract.go` + `router /tools/cover-extract` + `spacecleanup` 的 `CoverExtractStats` 依赖

## Constraints

- 仅改增强工具，不动 `备份管理`、`辅助工具` 外的 `任务管理/系统设置` 等
- 保留 `internal/cache` 核心与 `file/name_align.go` 的 `mediaorganize/rules`（与封面无关，已定）
- 删除后 `grep -r "AIToolCard|ClassificationToolCard|CleanupToolCard|CoverExtractToolCard|ProxyToolsPanel|QuarkTV" --include="*.ts" --include="*.vue" web/src` 为 0，`grep -r "embyproxy|fnosproxy|quarktv|spacecleanup|coverextract" --include="*.go" | grep -v ".trellis" | wc -l` 为 0（除 `local_upload` 保留）

## Acceptance Criteria

- [ ] `CloudToolsPanel.vue` 仅渲染 `LocalUploadToolCard`，`cardTitles` 仅 `["从服务器上传"]`，`AuxToolsManagement.vue` 仍仅 `增强工具/备份管理` 两 Tab
- [ ] `web/src/components/admin/{AIToolCard,ClassificationToolCard,CleanupToolCard,CoverExtractToolCard,ProxyToolsPanel,QuarkTVToolCard,QuarkTVBindModal}.vue` 已删除，`web/src/api/cloudTools.ts` 仅保留 `localUploadApi`，`web/src/api/coverExtract.ts` 等若为空则删除
- [ ] 后端 `internal/embyproxy, fnosproxy, quarktv, spacecleanup, coverextract` 目录已 `rm -rf`，`wire_services.go/wire_http.go/router.go/settings/registry.go` 无引用，`curl -b cookie /api/admin/emby/config 404` 等同理（`quarktv/cleanup/cover-extract` 均 404），而 `GET /api/admin/tools/local-upload/config 200`
- [ ] `GOWORK=off go vet ./...` PASS，`cd web && npm run type-check && npm run build` PASS，`docker build -t litepan-go:aux-keep-upload` PASS 且体积 <125MB，`/api/health 200`

## Out of Scope

- **备份管理**（`AuxToolsManagement.vue` 的 `备份管理` Tab、`BackupRestorePanel.vue`、`internal/backuprestore`、`api/backup*.go`、`router /admin/backups`）：本次完全不动，与增强工具非同一工具
- 从服务器上传本身的逻辑优化
- `internal/mediaorganize/rules` 的保留（已定）

## Key Decisions

- 删除范围：用户于 2026-08-30 确认 **彻底删除前后端**，非仅隐藏前端
