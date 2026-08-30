# Implementation Plan: Remove aux enhanced tools keep server upload

## Overview

按 `design.md` 边界，前后端并行删除 7 项增强工具，仅留 `LocalUpload`。分 5 阶段，自上而下删，避免中间态 `go vet` 失败。

## Phase 1: 前端面板精简（CloudToolsPanel）

- [ ] 1.1 `web/src/components/admin/CloudToolsPanel.vue`：
  - 删 `import ProxyToolsPanel, QuarkTVToolCard, AIToolCard, ClassificationToolCard, CleanupToolCard, CoverExtractToolCard` 6 行
  - 删 `cardTitles` 中除 `"从服务器上传"` 外 7 项，缩为 `const cardTitles = ["从服务器上传"]`
  - 删模板中 `<ProxyToolsPanel .../> / <QuarkTVToolCard .../> / <AIToolCard .../> / <ClassificationToolCard .../> / <CleanupToolCard .../> / <CoverExtractToolCard .../>` 6 个标签，仅留 `<LocalUploadToolCard :search-query="searchQuery" />`
  - `verify: grep -n "AIToolCard|ClassificationToolCard|CleanupToolCard|CoverExtractToolCard|ProxyToolsPanel|QuarkTV" web/src/components/admin/CloudToolsPanel.vue` == 0
- [ ] 1.2 `rm web/src/components/admin/AIToolCard.vue web/src/components/admin/ClassificationToolCard.vue web/src/components/admin/CleanupToolCard.vue web/src/components/admin/CloudToolCard.vue web/src/components/admin/CoverExtractToolCard.vue web/src/components/admin/ProxyToolsPanel.vue web/src/components/admin/ProxyWorkspace.vue web/src/components/admin/QuarkTVToolCard.vue web/src/components/admin/QuarkTVBindModal.vue web/src/components/admin/TmdbHostsHelpTip.vue`（存在则删）
  - `verify: ls web/src/components/admin/AIToolCard.vue` → No such file
- [ ] 1.3 `web/src/api/cloudTools.ts`：仅保留 `localUploadApi`（`getConfig/saveConfig/browse/createTasks`）与对应 `LocalUploadMapping` 类型，删 `quarktvApi / cleanupApi / coverExtractApi` 等
- [ ] 1.4 `rm web/src/api/coverExtract.ts web/src/api/emby.ts web/src/api/fnos.ts`（若存在且仅服务于增强工具）
  - `verify: grep -rn "coverExtract\|quarktv\|emby" --include="*.ts" web/src | grep -v "localUpload" | wc -l` == 0

## Phase 2: 后端服务删除（internal/*）

- [ ] 2.1 `rm -rf internal/embyproxy internal/fnosproxy internal/quarktv internal/spacecleanup internal/coverextract`
  - `verify: ls internal/embyproxy internal/quarktv` → No such file
- [ ] 2.2 `rm internal/api/emby.go internal/api/fnos.go internal/api/quarktv.go internal/api/space_cleanup.go internal/api/cover_extract.go internal/api/proxy_tools.go`（存在则删）
- [ ] 2.3 `internal/settings/registry.go`：删 `KeyEmby* / KeyFnos* / KeyQuarkTV* / KeySpaceCleanup* / KeyCoverExtract*` 的 `const` 与 `intSpec/boolSpec` 注册（约 10+ 处），保留 `KeyLocalUpload*`
  - `verify: grep -n "KeyEmby\|KeyFnos\|KeyQuarkTV\|KeyCoverExtract" internal/settings/registry.go` == 0

## Phase 3: App 装配（wire）

- [ ] 3.1 `internal/app/wire_services.go`：
  - 删 `import embyproxy/fnosproxy/quarktv` 3 行
  - 删 `servicesBundle` 的 `embyProxy/fnosProxy/quarktv` 字段
  - 删 `embyProxySvc/fnosProxySvc/quarktvSvc` 创建与 `playbackSvc.SetDownloadResolverHook`、`lifecycle.quarktv` 相关
  - `verify: grep -n "embyproxy\|fnosproxy\|quarktv" internal/app/wire_services.go` == 0
- [ ] 3.2 `internal/app/wire_http.go`：
  - 删 `import coverextract/spacecleanup` 2 行
  - 删 `coverExtractSvc` 创建 10 行
  - `spacecleanup.New` 整块删（或仅删 `CoverExtractStats/ClearCoverExtract` 若保留 `spacecleanup` 的其它回调则评估；本任务彻底删则整块删）
  - `api.NewRouter` 的 `Deps` 删 `EmbyProxy/FnosProxy/QuarkTV/SpaceCleanup/CoverExtract/ApiKeys.Auth` 等中对应 5 字段，仅留 `LocalUpload` 相关通过 `Files/Settings` 间接（`local_upload` 不需额外 Deps）
  - `verify: grep -n "coverextract\|spacecleanup\|quarktv\|embyProxy" internal/app/wire_http.go` == 0

## Phase 4: 路由与清理

- [ ] 4.1 `internal/api/router.go`：
  - `Deps` 删 `EmbyProxy/FnosProxy/QuarkTV/SpaceCleanup/CoverExtract` 5 字段，`Handler` 删对应 5 字段
  - `NewRouter` 的 `h := &Handler{..., coverExtract: d.CoverExtract, ...}` 删 5 行
  - `r.Route("/api", ...)` 中删 `r.Get("/internal/cover-source/{token}")`、`r.Route("/tools/local-upload")` 外的所有 `r.Route("/tools/{quarktv,cleanup,cover-extract}")` 与 `r.Route("/emby") /fnos /quarktv`，仅留 `Route("/tools/local-upload", 4 handler)`
  - `verify: grep -n "cover-extract\|spacecleanup\|quarktv\|emby" internal/api/router.go` == 0
- [ ] 4.2 `internal/api/local_upload.go` 保留，其余 `api/*.go` 已删
- [ ] 4.3 `rm -rf web/src/components/file/CloudLocalUploadPanel.vue` 若仅服务于已删工具则评估；`LocalUploadToolCard` 保留故 `CloudLocalUploadPanel` 若为 `Files` 列表的上传入口则保留

## Phase 5: Sweep & Verification

- [ ] 5.1 `grep -r -n "AIToolCard|ClassificationToolCard|CleanupToolCard|CoverExtractToolCard|ProxyToolsPanel|QuarkTVToolCard" --include="*.vue" --include="*.ts" web/src | wc -l` == 0
- [ ] 5.2 `grep -r -n "embyproxy|fnosproxy|quarktv|spacecleanup|coverextract" --include="*.go" | grep -v ".trellis" | wc -l` == 0
- [ ] 5.3 `GOWORK=off go vet ./...` PASS
- [ ] 5.4 `cd web && npm run type-check` PASS, `npm run build` PASS 109→~90 files
- [ ] 5.5 `GOWORK=off go build -o /tmp/litepan-aux ./cmd/litepan` PASS
- [ ] 5.6 `docker build -t litepan-go:aux-keep-upload .` PASS，`docker run -d -p 5211:5211 -v ./data:/app/data litepan-go:aux-keep-upload` + `curl /api/health 200` + `curl -b cookie /api/admin/tools/local-upload/config 200` + `curl -b cookie /api/admin/tools/cover-extract/files 404` + `curl -b cookie /api/admin/emby/config 404`

## Phase 6: Commit & Archive

- [ ] 6.1 `git status --porcelain` review  `git diff --stat`
- [ ] 6.2 `git add -A && git restore --staged .trellis/tasks/08-30-remove-aux-enhanced-keep-upload && git commit -m "refactor(aux-tools): remove enhanced tools keep local-upload"`
- [ ] 6.3 `task.py archive 08-30-remove-aux-enhanced-keep-upload --skip-branch-validation && git add .trellis/tasks/archive/... && git commit -m "chore(task): archive 08-30-remove-aux-enhanced-keep-upload"`
- [ ] 6.4 `add_session.py --commit <hash>`

## Rollback

- `git revert <refactor commit>` 恢复 7 项卡片与后端

## Validation Commands

```bash
grep -rn "AIToolCard" --include="*.vue" web/src | wc -l  # 0
grep -rn "embyproxy" --include="*.go" | wc -l  # 0
GOWORK=off go vet ./...
cd web && npm run type-check && npm run build
docker build -t litepan-go:aux-keep-upload . && echo ok
curl -b /tmp/c.txt http://127.0.0.1:5211/api/admin/tools/local-upload/config | grep enabled
```
