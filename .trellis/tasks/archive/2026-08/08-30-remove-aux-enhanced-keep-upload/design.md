# Design: Remove aux enhanced tools keep server upload

## Overview

增强工具 `CloudToolsPanel.vue` 当前聚合 8 项卡片，7 项需彻底下线。前端为卡片级 `defineAsyncComponent` + `searchQuery` 过滤，后端为 `wire_services` 的 `quarktv/embyproxy/fnosproxy` 与 `wire_http` 的 `coverextract/spacecleanup`。保留的 `LocalUpload` 为独立 `local_upload.go + settings KeyLocalUpload*`，与其它工具无共享表，可孤立删除。

## Boundaries

| 层 | 删除 | 保留 | 说明 |
|---|---|---|---|
| **前端 CloudToolsPanel** | `ProxyToolsPanel.vue / ProxyWorkspace.vue / TmdbHostsHelpTip.vue`、`QuarkTVToolCard.vue / QuarkTVBindModal.vue`、`AIToolCard.vue`、`ClassificationToolCard.vue`、`CleanupToolCard.vue / CloudToolCard.vue`、`CoverExtractToolCard.vue` 的 `import` 与 `<ProxyToolsPanel|QuarkTV|AI|Classification|Cleanup|CoverExtract>` 标签 | `LocalUploadToolCard.vue` 唯一 import 与渲染 | `cardTitles` 缩为 `["从服务器上传"]` |
| **前端 API** | `web/src/api/cloudTools.ts` 中除 `localUploadApi` 外的 `emby/fnos/quarktv/spaceCleanup/coverExtract` 相关 export，`web/src/api/coverExtract.ts`、`emby.ts`、`fnos.ts` 若仅服务于增强工具则删 | `localUploadApi.getConfig/saveConfig/browse/createTasks` | `types.ts` 中相关 DTO 同步删 |
| **后端 proxy** | `internal/embyproxy/*`、`internal/fnosproxy/*`（含 `service.go` 2 个 + `fnos_test.go`） | 无 | `wire_services.go` 的 `embyProxySvc/fnosProxySvc` 字段、创建、注入 `api.Deps`、`lifecycle` 无关 |
| **后端 quarktv** | `internal/quarktv/*`（`service.go/quarktv_binding*` 等 6+ 文件）+ `internal/api/quarktv.go` | 无 | `wire_services` 的 `quarktvSvc` 及其 `playback.SetDownloadResolverHook` 需评估：若 `quarktv` 仅为工具，则连 hook 一并删 |
| **后端 cleanup** | `internal/spacecleanup/*`（`service.go + cleanup.go + service_test.go`）+ `internal/api/space_cleanup.go` | 无 | `wire_http` 的 `spaceCleanupSvc` 及其 `FUSE/Backup` 回调，若无其它调用方可全删 |
| **后端 coverextract** | `internal/coverextract/*`（`service.go 1006 行`）+ `internal/api/cover_extract.go` | 无 | 刚在 `08-30-fix-coverextract-nil` 恢复，本任务再彻底下线；`wire_http` 的 `coverExtractSvc` 及其在 `spacecleanup` 的 `CoverExtractStats` 依赖一并删 |
| **settings** | `internal/settings/registry.go` 的 `KeyEmby* / KeyFnos* / KeyQuarkTV* / KeySpaceCleanup* / KeyCoverExtract*` 及 `intSpec/boolSpec` 定义 | `KeyLocalUploadEnabled / KeyLocalUploadMappings` 保留 | `grep -r "KeyEmby" registry.go` 为 0 |
| **router** | `internal/api/router.go` 的 `Deps{EmbyProxy,FnosProxy,QuarkTV,SpaceCleanup,CoverExtract}` 字段、`Handler{embyProxy,...}`、`r.Route("/tools/local-upload")` 外的所有 `Route("/tools/*")` 与 `Route("/emby"...)` `/fnos` `/quarktv` `/cleanup` `/cover-extract`，以及 `r.Get("/internal/cover-source/{token}")` | `Route("/tools/local-upload", ... 4 handler)` 保留 | `api/local_upload.go` 的 `coverExtractSource` 无 |

## Data Flow

```
Before: web CloudToolsPanel → ProxyToolsPanel/QuarkTV/AI/Classification/Cleanup/CoverExtract → POST /api/admin/tools/{quarktv,cleanup,cover-extract} /api/admin/emby/* → wire_http → embyproxy/fnosproxy/quarktv/spacecleanup/coverextract → driver/file
After:  web CloudToolsPanel → LocalUploadToolCard → POST /api/admin/tools/local-upload/* → local_upload.go → file.Service.UploadLocal → driver → 200
        其它 7 路径 → 404（chi 404 text/plain）
```

## Compatibility

- **DB**：`spacecleanup` / `coverextract` 为内存态（`coverextract` 的 `files/frames/tokens` 为 `map`），`quarktv` 的 `quarktv_bindings` 表若存在则保留建表 SQL 但不再读写（`store.Migrate` 保留 `CREATE TABLE`，与 `cache_retention` 策略一致），不 `DROP`
- **Config**：`settings` 中旧键（`emby_proxy_enabled` 等）保留在库但不再 `Get/Set`，前端不再展示
- **FUSE/Upload**：`spacecleanup` 的 `UploadActivePaths / OfflineTempRoots / FuseCacheStats` 回调若全删，需确认 `wire_http` 不再引用 `svc.uploads` 等
- **Playback hook**：`quarktvSvc.ResolveHook` 若随 `quarktv` 删除，`playbackSvc.SetDownloadResolverHook` 传入 `nil` 或删该行

## Tradeoffs

- **彻底删 vs 隐藏**：用户已选彻底删，减 `~2k 行` 前端 + `~3k 行` 后端，镜像预计 `125MB → ~110MB`，但日后恢复需 `git revert`
- **coverextract 再删**：刚在 `fix-coverextract-nil` 恢复，紧接着删，看似反复，但符合用户「仅留上传」的最终态，避免残留 `ffmpeg` 下载逻辑

## Rollout / Rollback

- 单提交 `refactor(aux-tools): remove enhanced tools keep local-upload`，`git revert` 即恢复 7 项
- 容器：`docker build -t litepan-go:aux-keep-upload` 后 `curl -b cookie /api/admin/tools/local-upload/config 200` 且 `curl -b cookie /api/admin/tools/cover-extract/files 404` 为门禁

## Risks

- `ProxyToolsPanel` 内含 `Emby 反代` 与 `飞牛影视反代` 共用 `ProxyWorkspace`，删时需一并删 `ProxyWorkspace.vue` 但保留 `LocalUploadToolCard` 的 `workspace` 不受影响（后者自带 `ProxyWorkspace` 的简化版，非共用）
- `quarktv` 的 `ResolveHook` 删后，`playback` 的夸克 TV 直链回落为本体，需确认 `原文件读取` 仍稳
