# Implementation Plan: Remove STRM

## Overview

Bottom-up deletion in 9 phases, each with `verify` command. Single owner, sequential, commit once at Phase 9.

## Phase 1: Domain & Store (backend bottom)

- [ ] 1.1 Delete `internal/domain/strm.go`, `strm_dir_cache.go`
  - `grep -l "strm" internal/domain/*.go` must become 0
  - `verify: GOWORK=off go vet ./internal/domain`

- [ ] 1.2 Delete `internal/store/strm_repo.go`, `strm_dir_cache_repo.go`
  - Edit `internal/store/store.go`: remove `StrmTasks StrmBranches StrmDirCache` fields, `strmRepo` init in `New()`
  - Edit `internal/store/migrate.go`: comment STRM `CREATE TABLE strm_tasks/strm_branches/strm_dir_cache` (retain comment for compat) or delete
  - `verify: GOWORK=off go vet ./internal/store && GOWORK=off go test ./internal/store -run Test -count=1 -skip TestMigrate` (memory DB)

## Phase 2: Service Layer

- [ ] 2.1 `rm -rf internal/strm internal/strmscrape`
  - `verify: ls internal/strm internal/strmscrape 2>&1 | grep "No such"` 

- [ ] 2.2 Edit `internal/app/wire_strm.go` → delete file; `internal/app/app.go` remove `strm *strm.Service`, `strmScrape *strmscrape.Service`, `strmDir`, `accountLifecycle` strm branches
  - `internal/app/wire_services.go` / `wire_core.go` / `wire_store.go` remove strm wiring
  - `internal/app/account_lifecycle.go` remove `strm.PauseByAccount`, `RemoveTasksByAccount`, `ClearDirCache`
  - `verify: GOWORK=off go vet ./internal/app`

## Phase 3: Config & Automation

- [ ] 3.1 Edit `internal/config/config.go`: delete `StrmDir`, `StrmDirForData()`, `LITEPAN_STRM_DIR` handling; `config_test.go` delete 3 `StrmDir` cases
  - `verify: GOWORK=off go test ./internal/config -run TestLoad*`

- [ ] 3.2 Edit `internal/automation/service.go`, `service_run.go`, `service_validate.go`: remove `AutomationActionStrm/StrmScrape` cases, `strm_tasks` option loading (around `automation.Options{strm_tasks: ...}`), `domain/automation.go` constants
  - `internal/settings/registry.go` remove strm settings
  - `verify: GOWORK=off go vet ./internal/automation`

## Phase 4: API Layer

- [ ] 4.1 `rm internal/api/strm_admin.go internal/api/strm_play.go internal/api/strm_scrape.go`
- [ ] 4.2 Edit `internal/api/router.go`: remove `Deps{Strm, StrmScrape, StrmDir}`, `r.Route("/strm")`, `r.Route("/admin/strm")`, `r.Get("/strm/play")`, `Handler{strm, strmScrape, strmDir}` fields
  - Edit `internal/api/tools.go` remove `strm` helpers if any
  - `verify: GOWORK=off go vet ./internal/api && GOWORK=off go build -o /tmp/litepan ./cmd/litepan` (expect 0)

## Phase 5: Frontend API & Stores

- [ ] 5.1 `rm web/src/api/strm.ts web/src/api/strmScrape.ts` + edit `web/src/api/automation.ts` remove `strm_tasks` from `AutomationOptions`
  - `verify: cd web && npx vue-tsc -b --noEmit 2>&1 | head`

## Phase 6: Frontend Components & Views

- [ ] 6.1 `rm web/src/components/admin/StrmSettingsPanel.vue web/src/components/admin/StrmScrapePanel.vue web/src/components/admin/StrmScrapeScopePicker.vue web/src/components/admin/StrmScrapeSettings.vue web/src/composables/useStrmDirectoryPrompt.ts`
- [ ] 6.2 Edit `web/src/components/admin/TaskManagement.vue`: remove `import { *Strm* } from "@/api/strm"` , `STRM_TAB`, `StrmSettingsPanel`, `strm` drawer logic, `tasks: StrmTask[]`
- [ ] 6.3 Edit `web/src/views/AdminView.vue`: remove `defaultTab: "strm"` → set `"cache"` or `"organize"`, remove `tabs: {strm, scrape}`
- [ ] 6.4 Edit `web/src/components/file/FileBrowser.vue`: remove `import { generateCurrentDirectoryStrm }`, `useStrmDirectoryPrompt`, `strmGenerating`, `strmPrompt`, template `<div class="strm-prompt-bar">` and style block `.strm-prompt-bar`
- [ ] 6.5 Edit `web/src/components/file/FileTable.vue`: remove `generate-current-directory-strm` emit/menus
  - `verify: cd web && npm run type-check` (vue-tsc -b) PASS, `grep -r -i strm web/src` == 0

## Phase 7: Deploy & Docs

- [ ] 7.1 Edit `Dockerfile`: remove `ENV LITEPAN_STRM_DIR`, `RUN mkdir -p /app/strm`, `VOLUME ["/app/strm"]` (keep data/mounts), edit `COPY --from=web` still OK
- [ ] 7.2 Edit `docker-compose.yml` + `docker-compose.fnos.yml`: remove `- ./strm:/app/strm` volume, remove `LITEPAN_STRM_DIR` env if any
- [ ] 7.3 Edit `README.md`: remove STRM table rows (2 rows: `STRM 直连播放` + `STRM 刮削`) and feature images, update badge counts
  - `verify: cat docker-compose.yml | grep strm` == 0, `cat Dockerfile | grep -i strm` == 0

## Phase 8: Sweep & Verification

- [ ] 8.1 `grep -r -i strm --include="*.go" --include="*.ts" --include="*.vue" | grep -v ".trellis" | grep -v ".git" | grep -v "journal"` == 0
  - If any remain, fix per file
- [ ] 8.2 `GOWORK=off go vet ./...` PASS
- [ ] 8.3 `GOWORK=off go test ./internal/config ./internal/automation -run Test -count=1` PASS
- [ ] 8.4 `cd web && npm run type-check` PASS, `npm run build` PASS (outputs to `internal/api/web`)
- [ ] 8.5 `GOWORK=off go build -trimpath -ldflags="-s -w" -o /tmp/litepan ./cmd/litepan` PASS
- [ ] 8.6 `docker build -t litepan-go:dev .` PASS (web stage + go stage), `docker run -d --name litepan-test -p 5212:5211 ... litepan-go:dev` + `curl -s http://127.0.0.1:5212/api/health` ok + `curl -i http://127.0.0.1:5212/api/strm/tasks` == 404

## Phase 9: Commit & Archive

- [ ] 9.1 `git diff --name-only` review (expect ~60 files deleted + 15 edited, no `strm` remain)
- [ ] 9.2 `git add -A` + `git commit -m "refactor(strm): remove STRM feature completely\n\nTask: 08-30-remove-strm"` (single commit per Trellis)
- [ ] 9.3 `python3 ./.trellis/scripts/task.py archive 08-30-remove-strm --skip-branch-validation` + `git add .trellis/tasks/archive/...` + `git commit -m "chore(task): archive 08-30-remove-strm"`
- [ ] 9.4 `python3 ./.trellis/scripts/add_session.py --title "Remove STRM completely" --commit <hash> --summary "..." ` → `chore: record journal`

## Rollback

- `git revert <refactor commit>` restores all STRM files; `docker build` again includes strm; old `strm_tasks` tables still on disk (if not dropped) so revert functional.
- If host `strm/` dir was manually `rm -rf`, restore from backup before revert.

## Validation Commands (Complete Set)

```bash
grep -r -i strm --include="*.go" --include="*.ts" --include="*.vue" | grep -v ".trellis" | wc -l   # expect 0
GOWORK=off go vet ./...
GOWORK=off go test ./internal/config -run TestLoad -count=1
cd web && npm run type-check && npm run build
GOWORK=off go build -o /tmp/litepan ./cmd/litepan && echo ok
docker build -t litepan-go:dev . && echo ok
docker run -d --name litepan-test -p 5212:5211 -v /root/LitePan/data:/app/data -v /root/LitePan/mounts:/app/mounts:shared --device /dev/fuse --privileged --pid host litepan-go:dev
curl -s http://127.0.0.1:5212/api/health | grep ok && curl -i http://127.0.0.1:5212/api/strm/tasks | grep 404 && echo ok
docker rm -f litepan-test
```
