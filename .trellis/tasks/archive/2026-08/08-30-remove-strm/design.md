# Design: Remove STRM

## Overview

STRM is a vertical feature spanning `domain → store → service → api → web → automation → config → docker`. Removal must be **bottom-up** (domain/store first) then **middle** (service/app) then **top** (api/web) to keep intermediate builds green, and leave DB tables as no-op to avoid destructive migration.

## Boundaries

| Layer | Files to Delete | Keep |
|---|---|---|
| **domain** | `internal/domain/strm.go`, `strm_dir_cache.go` (structs `StrmTask`, `StrmBranch`, `StrmDirCache`, `StrmTaskRepository` etc.) | `domain/file.go`, `automation.go` but remove `AutomationActionStrm/StrmScrape` constants |
| **store** | `internal/store/strm_repo.go`, `strm_dir_cache_repo.go`; `store.go: Stores{StrmTasks, StrmBranches, StrmDirCache}` fields + `New()` wiring; `migrate.go` STRM `CREATE TABLE` (keep as comment or delete, but `Migrate` must stay idempotent) | Other repos |
| **service** | `internal/strm/*`, `internal/strmscrape/*` entire dirs | `internal/file`, `playback`, `quota` |
| **app** | `internal/app/wire_strm.go`, `app.go: strm.Service/Coordinator` fields, `wire_services.go` strm wiring, `account_lifecycle.go` strm pause/remove | `app.go` other lifecycles |
| **api** | `internal/api/strm_admin.go`, `strm_play.go`, `strm_scrape.go`; `router.go` `Deps{Strm, StrmScrape, StrmDir}` + `r.Route("/strm")` + `r.Route("/admin/strm")` etc.; `tools.go` strm helpers | `api/files.go`, `play/`, `favorites` |
| **config** | `internal/config/config.go: StrmDir, StrmDirForData, LITEPAN_STRM_DIR` + `Load()` branch + `config_test.go` 3 cases | `DataDir, DBPath, ListenAddr` |
| **automation** | `internal/automation/service.go` `strm_tasks` option loading, `service_run.go` `case AutomationActionStrm/StrmScrape`, `service_validate.go` validation, `domain/automation.go` constants, `web/src/api/automation.ts` `strm_tasks` | `organize`, `cache_clear` actions |
| **frontend** | `web/src/api/strm.ts`, `strmScrape.ts`; `web/src/components/admin/Strm*` (4); `web/src/composables/useStrmDirectoryPrompt.ts`; `web/src/views/AdminView.vue` tab `strm`; `web/src/components/file/FileBrowser.vue` prompt bar + `generateCurrentDirectoryStrm`; `FileTable.vue` menu | `CacheRetentionPanel`, `TaskManagement` but remove `STRM_TAB` imports |
| **deploy** | `Dockerfile: ENV LITEPAN_STRM_DIR, RUN mkdir -p /app/strm, VOLUME ["/app/strm"]`; `docker-compose*.yml: - ./strm:/app/strm` + `LITEPAN_STRM_DIR` if any; `internal/api/web` already built (will be rebuilt) | `data`, `mounts`, `fuse` |
| **docs** | `README.md` STRM table rows (2) + `docs/pictures/feature-strm*` references | Keep other feature tables |

## Data Flow Removal

```
Old: web FileBrowser → POST /admin/strm/tasks → api/strm_admin.go → strm.Service.Create → store.StrmTasks.Create → sqlite strm_tasks
New: (deleted)
Old: automation rule → actions[ {type: "strm"} ] → service_run.go → strm.Coordinator.Run
New: automation validates and drops type "strm"/"strm_scrape" with error "unknown action type"
Old: GET /strm/play/{account_id}/{file_key} → api/strm_play.go → strm.Service.Signer → redirect 302
New: route 404
```

- **Write flow** (`web → api → service → store`) and **read flow** (`store → service → api`) both removed; no shared types remain, so intermediate DTOs can be deleted without orphan.

## Compatibility & Migration

- **DB**: Keep tables `strm_tasks`, `strm_branches`, `strm_dir_cache` on disk; `store.Migrate` will not drop them (avoid `DROP TABLE` surprise for users with existing STRM files). New code simply never opens them. Optionally add comment `-- STRM tables retained for backward compat, unused since v0.6`.
- **Config**: Users with `LITEPAN_STRM_DIR` env will have it ignored (log `WARN unknown env LITEPAN_STRM_DIR ignored` if desired, but spec says silently ignore to keep startup clean).
- **Automation**: Old JSON rules containing `actions: [{type:"strm"}]` will be rejected by `service_validate.go` with `domain.CodeInvalid`; user must edit rule to remove strm actions. No auto-migration.
- **FS**: Host `./strm` dir and container `/app/strm` mount are removed from compose; existing host `strm/` dir is left on disk (user manually `rm -rf strm` if desired).

## Tradeoffs

- **Delete vs Feature flag**: Choose delete (徹底移除) per user request, not flag-off. Flag would keep 167 files dead code and `depguard` exceptions. Delete reduces binary size (~STRM 44 files) and `web` chunk size.
- **DROP TABLE vs retain**: Retain avoids data loss panic; cost is dead tables occupying <1MB. Documented in migration comment.
- **Bottom-up order**: Guarantees `go vet` passes after each layer; top-down would leave dangling imports and break build mid-task.

## Rollout / Rollback

- **Rollout**: Single commit `refactor(strm): remove STRM feature completely` (this task) → `docker build` → `docker run` on `:5211` → smoke `curl /api/health` + `grep -r strm` must be empty (except Trellis history).
- **Rollback**: `git revert <commit>` restores all 167 files; DB tables already exist, so revert instantly functional. If user deleted host `strm/` dir, need restore from backup.
- **Verification before merge**: `GOWORK=off go build`, `npm run type-check`, `npm run build`, `docker build`, `grep -r -i strm` 0, `curl /api/strm/tasks` 404.

## File Map (Deletion Order)

1. `internal/domain/strm*.go`
2. `internal/store/strm*.go` + `store.go` + `migrate.go`
3. `internal/strm/` + `internal/strmscrape/`
4. `internal/app/wire_strm.go` + `app.go` + `account_lifecycle.go`
5. `internal/api/strm*.go` + `router.go`
6. `internal/config/config.go` + test + `automation` + `settings`
7. `web/src/api/strm*.ts` + components + composables + `AdminView.vue` + `FileBrowser.vue`
8. `Dockerfile` + `docker-compose*.yml` + `README.md`
9. `grep` sweep + `go vet` + `web build` + `docker build` + `curl` smoke

## Risks

- **Import orphan**: `internal/api/tools.go` imports `strm` for helper `strmBaseURL` — must remove helper or replace with plain string.
- **Frontend `automation` options**: `web/src/api/automation.ts` `strm_tasks` removal must sync with `internal/automation` validation or UI will show 500 on old rules.
- **Spec drift**: `spec/backend/backend` currently documents STRM (e.g., `quality-guidelines` mentions `strm`?), but our spec already generic; still run `trellis-update-spec` after.

