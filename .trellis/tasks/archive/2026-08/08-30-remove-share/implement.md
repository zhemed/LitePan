# Implementation Plan: Remove File Share

## Overview

Bottom-up: delete `internal/share` first, then api/auth/settings/cache/log, then playback, then frontend, then docs, then verification.

## Phase 1: Share Backend (dav/fuse)

- [ ] 1.1 `rm -rf internal/share` (27 files: dav 16 + fuse 11)
  - `verify: ls internal/share 2>&1 | grep "No such file"`
- [ ] 1.2 Edit `internal/api/router.go`: remove `import "litepan/internal/share/dav"`, `davLog = d.Logs.For(logx.ModuleWebDAV)`, `r.Post("/webdav-config", h.adminWebDAVConfig)`, and mount bypass `r.Mount("/dav", ...)` + comment `chi 不认 WebDAV 方法`
  - `verify: grep -n "share/dav\|davLog\|webdav-config\|/dav" internal/api/router.go` == 0

## Phase 2: Auth/Settings/Cache/Log

- [ ] 2.1 Edit `internal/api/auth.go`: remove `adminWebDAVConfig` handler (73-80)
- [ ] 2.2 Edit `internal/adminauth/service.go`: remove `KeyWebDAVEnabled`, `WebDAVEnabled` field, `WebDAVConfigRequest`, `UpdateWebDAVConfig`, `webdavEnabled()`, `SystemConfig` webdav_enabled assignment, and related const map
  - `verify: grep -n "WebDAV\|webdav" internal/adminauth/service.go` == 0
- [ ] 2.3 Edit `internal/settings/registry.go`: remove `KeyWebDAVCacheEnabled` and `boolSpec` line
- [ ] 2.4 Delete `internal/cache/webdav_keys.go`; edit `internal/cache/keys.go` remove `prefixWebDAVMeta`, edit `internal/cache/cleaner.go` remove `InvalidateWebDAV*`, edit `internal/cache/service.go` WebDAV comment is ok, edit `internal/logx/module.go` remove `ModuleWebDAV` and its cases
  - `verify: grep -r -n "webdav" --include="*.go" internal/cache internal/logx | grep -v ".trellis"` == 0

## Phase 3: Playback/File

- [ ] 3.1 Edit `internal/playback/pick.go`: remove `WebDAV bool` field from `Intent`, update `if !intent.WebDAV` branches, remove `WebDAV` from tests; edit `playback/redirect.go` remove `intent.WebDAV` branch, `playback/service_test.go` remove WebDAV test cases
  - `verify: grep -n "WebDAV" internal/playback/*.go` == 0 (except maybe comments, but aim 0)

## Phase 4: Frontend

- [ ] 4.1 `rm web/src/components/admin/FileShareManagement.vue` (and `WebDAVSettings.vue` if exists)
  - `verify: ls web/src/components/admin/FileShare*` no file
- [ ] 4.2 Edit `web/src/views/AdminView.vue`: remove `import FileShareManagement`, `adminPageLoaders.share`, `{key:"share", label:"文件共享"}`, `share: {defaultTab:"webdav"...}`, `<FileShareManagement v-else-if="page==='share'" />`, and `['settings','cross-transfer','share']` includes
  - `verify: grep -n "FileShare\|share.*page" web/src/views/AdminView.vue` == 0
- [ ] 4.3 Edit `web/src/components/admin/...` if any remaining share references (e.g., `ApiKeySettings.vue` may have share? No, but check)
  - `verify: grep -r -n "FileShare\|WebDAV" --include="*.ts" --include="*.vue" web/src | grep -v ".trellis" | wc -l` == 0 (excluding `mounts:shared` is not in web/src)

## Phase 5: Docs & Deploy

- [ ] 5.1 Edit `README.md`: remove FileShare/WebDAV feature docs if any (currently none after STRM removal, but check)
- [ ] 5.2 Check `Dockerfile` and `docker-compose.yml`: ensure no `internal/share` references (already clean after STRM removal, but double-check `grep -i "share" Dockerfile` should be 0 except `mounts:shared` which is ok)
  - `verify: grep -i "internal/share" Dockerfile` == 0, `grep -i "FileShare" README.md` == 0

## Phase 6: Sweep & Verification

- [ ] 6.1 `grep -r -i -n "internal/share\|FileShare" --include="*.go" --include="*.ts" --include="*.vue" | grep -v ".trellis" | wc -l` == 0
  - Note: `grep -i "share"` will also hit `shared` in `mounts:shared`, so use precise pattern `FileShare`/`internal/share`/`webdav` for Go/TS/Vue, but for share feature the precise check is `internal/share` and `FileShare`
- [ ] 6.2 `GOWORK=off go vet ./...` PASS
- [ ] 6.3 `GOWORK=off go test ./internal/cache ./internal/playback -count=1` PASS
- [ ] 6.4 `cd web && npm run type-check` PASS, `npm run build` PASS
- [ ] 6.5 `GOWORK=off go build -trimpath -ldflags="-s -w" -o /tmp/litepan ./cmd/litepan` PASS
- [ ] 6.6 `docker build -t litepan-go:noshare .` PASS, `docker run -d --name litepan-test3 -p 5214:5211 ... litepan-go:noshare` + `curl /api/health ok` + `curl -i /dav/ 404` + `curl -i /api/admin/webdav-config 404`

## Phase 7: Commit & Archive

- [ ] 7.1 `git diff --name-only | grep -v ".trellis/tasks" ` review (expect ~30 files: 27 deleted + 10 edited)
- [ ] 7.2 `git add -A` + `git restore --staged .trellis/tasks/08-30-remove-share` + `git commit -m "refactor(share): remove file share (WebDAV dav/fuse) completely"`
- [ ] 7.3 `python3 ./.trellis/scripts/task.py archive 08-30-remove-share --skip-branch-validation` + `git add .trellis/tasks/archive/...` + `git commit -m "chore(task): archive 08-30-remove-share"`
- [ ] 7.4 `python3 ./.trellis/scripts/add_session.py --title "Remove file share completely" --commit <hash> ...` → `chore: record journal`

## Rollback

- `git revert <refactor commit>` restores `internal/share` and UI; `configs` table `webdav_enabled` rows still present (if not deleted) so revert functional.
- If `FileShareManagement.vue` was deleted, revert restores it.

## Validation Commands

```bash
grep -r -n "internal/share\|FileShare" --include="*.go" --include="*.ts" --include="*.vue" | grep -v ".trellis" | wc -l  # expect 0
grep -r -n "webdav" --include="*.go" internal/cache internal/logx internal/api | grep -v ".trellis" | wc -l  # expect 0 (excluding drivers/WebDAV)
GOWORK=off go vet ./...
GOWORK=off go test ./internal/cache -count=1
cd web && npm run type-check && npm run build
GOWORK=off go build -o /tmp/litepan ./cmd/litepan && echo ok
docker build -t litepan-go:noshare . && echo ok
docker run -d --name litepan-test3 -p 5214:5211 -v /root/LitePan/data:/app/data --device /dev/fuse --privileged --pid host litepan-go:noshare
curl -s http://127.0.0.1:5214/api/health | grep ok && curl -i http://127.0.0.1:5214/dav/ | grep 404 && echo ok
docker rm -f litepan-test3
```
