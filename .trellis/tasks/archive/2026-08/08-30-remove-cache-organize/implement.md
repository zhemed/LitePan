# Implementation Plan: Remove Cache and Organize

## Overview

Bottom-up deletion in 6 phases.

## Phase 1: Backend Services (cacheretention, mediaorganize, classify, ai, coverextract)

- [ ] 1.1 `rm -rf internal/cacheretention internal/mediaorganize internal/classifyorganize internal/aiorganize`
  - Also `rm -rf internal/coverextract` if exists and only for organize (check `grep -r "coverextract" --include="*.go" | grep -v ".trellis"` if only mediaorganize uses it, then delete)
  - `verify: ls internal/cacheretention internal/mediaorganize` → No such file

- [ ] 1.2 Delete `internal/domain/cache_retention.go` `internal/domain/media_organize.go` (and classify/ai domain if separate)
  - `verify: ls internal/domain/cache* internal/domain/media*` → No such

- [ ] 1.3 Delete `internal/store/cache_retention_repo.go` `internal/store/media_organize_repo.go`
  - Edit `internal/store/store.go`: remove `CacheRetentionTasks`, `MediaOrganizeTasks` fields and `New()` assignments
  - Keep `migrate.go` SQL (do not delete migrations)
  - `verify: grep -n "CacheRetention\|MediaOrganize" internal/store/store.go` == 0

## Phase 2: App Wiring

- [ ] 2.1 `rm internal/app/wire_cache_retention.go` `internal/app/wire_mediaorganize.go`
- [ ] 2.2 Edit `internal/app/app.go`: remove `cacheRetention`, `mediaOrganize`, `aiOrganize`, `classifyOrganize` fields, imports, `if a.cacheRetention != nil` Start, and `if a.mediaOrganize != nil` etc.
- [ ] 2.3 Edit `internal/app/account_lifecycle.go`: remove `retention` and `media` fields and their `Pause/Remove` logic
- [ ] 2.4 Edit `internal/app/wire_services.go`: remove `cacheretention`, `mediaorganize`, `aiorganize`, `classifyorganize` imports, `cacheRetention`, `mediaOrganize` fields, and wiring (wireCacheRetention, wireMediaOrganize, lifecycle retention/media)
  - `verify: grep -n "cacheretention\|mediaorganize\|aiorganize\|classifyorganize" internal/app/*.go` == 0

## Phase 3: API Layer

- [ ] 3.1 `rm internal/api/cache_retention_admin.go` `internal/api/media_organize.go` `internal/api/ai_organize.go` `internal/api/classification.go` (if classification is organize)
- [ ] 3.2 Edit `internal/api/router.go`: remove `Deps{CacheRetention, MediaOrganize, AiOrganize, ClassifyOrganize}`, `Handler` fields, and routes `r.Route("/admin/cache-retention", ...)`, `r.Route("/admin/media-organize", ...)`, etc.
  - `verify: grep -n "cache-retention\|media-organize\|ai_organize\|classification" internal/api/router.go` == 0

## Phase 4: Automation

- [ ] 4.1 Edit `internal/domain/automation.go`: remove `AutomationActionCacheClear`, `AutomationActionOrganize` constants
- [ ] 4.2 Edit `internal/automation/service.go`, `service_run.go`, `service_validate.go`: remove `case AutomationActionCacheClear`, `case AutomationActionOrganize`, and related functions `runCacheClear`, `runOrganize`, validation, and `MediaOrganize` field
  - `verify: grep -n "CacheClear\|Organize" internal/automation/*.go` == 0 (allow `organize` in comments? aim 0)

## Phase 5: Frontend

- [ ] 5.1 `rm web/src/api/cacheRetention.ts` `web/src/api/mediaOrganize.ts` (`web/src/api/aiOrganize.ts` if exists) and `web/src/components/admin/CacheRetentionPanel.vue` `MediaOrganizePanel.vue` `MediaOrganizeSettings.vue` `web/src/composables/useOrganizePlanPreview.ts`
- [ ] 5.2 Edit `web/src/components/admin/TaskManagement.vue`: remove `organize` tab imports and logic (keep other tabs)
- [ ] 5.3 Edit `web/src/views/AdminView.vue`: remove organize page entry if any
- [ ] 5.4 Edit `web/src/components/admin/DashboardManagement.vue`: remove organize/cache retention stats
  - `verify: grep -r -n "cacheretention\|CacheRetention\|mediaOrganize\|MediaOrganize" --include="*.ts" --include="*.vue" web/src | wc -l` == 0

## Phase 6: Sweep & Verification

- [ ] 6.1 `grep -r -i -n "cacheretention\|mediaorganize\|classifyorganize\|aiorganize" --include="*.go" --include="*.ts" --include="*.vue" | grep -v ".trellis" | wc -l` == 0
- [ ] 6.2 `GOWORK=off go vet ./...` PASS
- [ ] 6.3 `GOWORK=off go test ./internal/cache -count=1` PASS (core cache, not cacheretention)
- [ ] 6.4 `cd web && npm run type-check` PASS, `npm run build` PASS
- [ ] 6.5 `GOWORK=off go build -trimpath -ldflags="-s -w" -o /tmp/litepan ./cmd/litepan` PASS
- [ ] 6.6 `docker build -t litepan-go:nocache-organize .` PASS, `docker run -d --name litepan-test -p 5215:5211 ...` + `curl /api/health ok` + `curl -i /api/admin/cache-retention/tasks` 404

## Phase 7: Commit & Archive

- [ ] 7.1 `git diff --name-only | grep -v ".trellis/tasks"` review
- [ ] 7.2 `git add -A` + `git restore --staged .trellis/tasks/08-30-remove-cache-organize` + `git commit -m "refactor(cache,organize): remove cache tasks and directory organization"`
- [ ] 7.3 `python3 ./.trellis/scripts/task.py archive 08-30-remove-cache-organize --skip-branch-validation` + `git add .trellis/tasks/archive/...` + `git commit`
- [ ] 7.4 `python3 ./.trellis/scripts/add_session.py --title ... --commit ...`

## Rollback

- `git revert <refactor commit>` restores all; DB tables still exist.

## Validation Commands

```bash
grep -r -i "cacheretention" --include="*.go" | grep -v ".trellis" | wc -l  # 0
grep -r -i "mediaorganize" --include="*.go" --include="*.ts" --include="*.vue" | grep -v ".trellis" | wc -l  # 0
GOWORK=off go vet ./...
GOWORK=off go test ./internal/cache -count=1
cd web && npm run type-check && npm run build
GOWORK=off go build -o /tmp/litepan ./cmd/litepan && echo ok
docker build -t litepan-go:nocache-organize . && echo ok
```
