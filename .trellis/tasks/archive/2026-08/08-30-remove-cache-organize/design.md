# Design: Remove Cache Tasks and Directory Organization

## Overview

Cache tasks (`cacheretention`) and directory organization (`mediaorganize`/`classifyorganize`/`aiorganize`/`coverextract`) are vertical features with DB, service, API, frontend, automation. Removal must be bottom-up, keep `internal/cache` core.

## Boundaries

| Layer | Delete | Keep |
|---|---|---|
| **cache tasks** | `internal/cacheretention/*` (15 files), `internal/store/cache_retention_repo.go`, `internal/domain/cache_retention.go`, `internal/api/cache_retention_admin.go`, `internal/app/wire_cache_retention.go`, `web/src/api/cacheRetention.ts`, `web/src/components/admin/CacheRetentionPanel.vue` | `internal/cache/*` (14 files core), `internal/fusereadcache` (if used by fusemount, keep) |
| **media organize** | `internal/mediaorganize/*` (30+ files), `internal/domain/media_organize.go`, `internal/store/media_organize_repo.go`, `internal/api/media_organize.go`, `internal/app/wire_mediaorganize.go`, `web/src/api/mediaOrganize.ts`, `web/src/components/admin/MediaOrganizePanel.vue`, `MediaOrganizeSettings.vue`, `web/src/composables/useOrganizePlanPreview.ts` | `internal/file` core |
| **classify/ai** | `internal/classifyorganize/*` (3 files), `internal/aiorganize/*` (4 files), `internal/api/ai_organize.go`, `classification.go`, `internal/coverextract` (if only for organize) | `internal/domain` other |
| **app** | `internal/app/account_lifecycle.go` retention/media branches, `app.go: cacheRetention/mediaOrganize/aiOrganize/classifyOrganize` fields, `wire_services.go` wiring, `wire_http.go` Deps | `files`, `uploads`, `fuse`, `automation` other |
| **automation** | `AutomationActionCacheClear`, `AutomationActionOrganize` constants and handlers in `service_run.go`, `service_validate.go` | `AutomationActionDelay`, `EmbyRefresh` |
| **frontend** | `TaskManagement.vue` organize tab, `AdminView.vue` organize entry, `DashboardManagement.vue` organize stats | `CacheRetentionPanel` removed, `AutomationPanel` keep but without organize options |
| **store** | `store.go` fields `CacheRetentionTasks`, `MediaOrganizeTasks`, `migrate.go` keep SQL for backward compat | `Accounts`, `UploadTasks` etc. |

## Data Flow Removal

```
Old: web CacheRetentionPanel → POST /admin/cache-retention/tasks → api/cache_retention_admin.go → cacheretention.Service.Create → store.CacheRetentionTasks → sqlite
New: 404

Old: web MediaOrganizePanel → POST /admin/media-organize/tasks → api/media_organize.go → mediaorganize.Service → store.MediaOrganizeTasks
New: 404

Old: automation rule → actions[ {type:"cache_clear"} | {type:"organize"} ] → service_run.go → runCacheClear/runOrganize
New: validation rejects cache_clear/organize with CodeInvalid
```

- No shared types between cache/organize and other domains, so deletion is isolated.

## Compatibility

- **DB**: Keep tables `cache_retention_tasks`, `media_organize_tasks` etc. on disk, `store.Migrate` retains their CREATE TABLE, but code never reads/writes. No DROP.
- **Config**: No config for these features, so no env handling.
- **Automation**: Old JSON rules with `cache_clear`/`organize` will be rejected by `service_validate.go` with `CodeInvalid`; user must edit.
- **FS**: No FS data for these, just tasks.

## Tradeoffs

- **Delete vs flag**: Delete per user request, not flag.
- **Keep internal/cache core**: `internal/cache` is used by `file.Service` for DirCache, must keep; only `cacheretention` is task layer.

## Rollout / Rollback

- Single commit `refactor(cache,organize): remove cache tasks and directory organization`
- Rollback: `git revert` restores files; DB tables still exist.

## File Map (Deletion Order)

1. `internal/cacheretention`, `internal/mediaorganize`, `internal/classifyorganize`, `internal/aiorganize`, `internal/coverextract` (if)
2. `internal/domain/cache_retention.go`, `media_organize.go`, `internal/store/cache_retention_repo.go`, `media_organize_repo.go`, `store.go`, `migrate.go` (keep SQL)
3. `internal/api/cache_retention_admin.go`, `media_organize.go`, `ai_organize.go`, `classification.go`, `router.go`
4. `internal/app/wire_cache_retention.go`, `wire_mediaorganize.go`, `app.go`, `account_lifecycle.go`, `wire_services.go`
5. `internal/automation` organize/cache_clear
6. `web` api and components
7. `grep` sweep + `go vet` + `web build` + `docker build`

## Risks

- `internal/cache` is core, must not delete; only `cacheretention` is task.
- `TaskManagement.vue` has multiple tabs, removing organize must keep other tabs functional.
- `automation` has many tests for organize, must remove those tests or they will fail.
