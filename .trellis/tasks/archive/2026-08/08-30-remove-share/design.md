# Design: Remove File Share

## Overview

File Share in LitePan is a **对外共享** vertical: `AdminView share page (FileShareManagement → WebDAVSettings + FuseManagement) → internal/api WebDAV config + dav server → internal/share (dav/fuse backends) → internal/cache WebDAV keys → internal/settings WebDAV cache toggle`. It is **separate** from `drivers/WebDAV` (WebDAV **客户端**挂载) and `internal/fusemount` (FUSE **挂载**). Removal must keep those two, delete only `internal/share` and its UI/API.

## Boundaries

| Layer | Delete | Keep |
|---|---|---|
| **share backend** | `internal/share/` all (dav 16 + fuse 11 files) | `internal/fusemount`, `drivers/WebDAV` (mount remote) |
| **api** | `internal/api/router.go: import share/dav`, `davLog`, `r.Post("/webdav-config")`, `r.Mount("/dav", ...)` bypass, `internal/api/auth.go: adminWebDAVConfig`, `internal/api/settings.go: InvalidateAllWebDAVCaches` | `internal/api/files.go`, `internal/api/auth.go` other handlers |
| **auth/settings** | `internal/adminauth/service.go: KeyWebDAVEnabled, WebDAVEnabled, WebDAVConfigRequest, UpdateWebDAVConfig, webdavEnabled()`, `internal/settings/registry.go: KeyWebDAVCacheEnabled` | `internal/adminauth` other keys, `internal/settings` other specs |
| **cache/log** | `internal/cache/webdav_keys.go`, `internal/cache/keys.go: prefixWebDAVMeta`, `cache/cleaner.go: InvalidateWebDAV*`, `cache/webdav_keys.go` whole, `internal/logx/module.go: ModuleWebDAV` | `internal/cache` other keys, `ModuleFileOp` |
| **playback/file** | `internal/playback/pick.go: WebDAV bool` field and `Intent.WebDAV` branches (`pick.go: !intent.WebDAV`), `playback/redirect.go: intent.WebDAV`, `file/service.go: DirCacheTTL` WebDAV comment | `playback` other intents (OriginalFile, etc.) |
| **frontend** | `web/src/components/admin/FileShareManagement.vue`, `WebDAVSettings.vue` (if separate), `web/src/views/AdminView.vue: share page loader, {key:"share"}`, `web/src/api/share.ts` if exists | `FuseManagement.vue` (local mount, not share/fuse), `FileBrowser`, `Dashboard` |
| **deploy/docs** | `README.md` FileShare/WebDAV docs, `docs/pictures/feature-*share*` if any | `Dockerfile`, `docker-compose.yml` keep `mounts:shared` (Docker propagation, not feature) |

## Data Flow Removal

```
Old: AdminView → FileShareManagement → POST /api/admin/webdav-config (adminWebDAVConfig → adminauth.UpdateWebDAVConfig → settings KeyWebDAVEnabled) → share/dav/server.go checks configWebDAVEnabled → davLog
New: (deleted) → admin returns 404

Old: WebDAV client → /dav/* → router bypass → share/dav/server.go (filesystem.go/path.go/webdav_cache.go) → cache/webdav_keys.go → file.Service
New: /dav/* → 404 page not found (chi no longer mounts dav)

Old: settings KeyWebDAVCacheEnabled toggle → cache/webdav_keys.go InvalidateAllWebDAVCaches → cache cleaner
New: toggle removed, cache keys file deleted
```

- **Write flow** (`WebDAV PUT → dav/write.go → temp.go → file.Service`) and **read flow** (`PROPFIND → listing.go/pathcache.go → webdav_cache.go`) both removed.
- **No shared types** between share and other domains, so deletion is isolated.

## Compatibility & Migration

- **DB**: No share-specific tables (share is stateless, lives in `configs` table as `webdav_enabled` key + `webdav_cache_enabled`). Old `configs` rows with `webdav_enabled` will be ignored (settings `String()` returns "" for missing key, defaults to true in old code, now not read). No migration needed.
- **Config**: Users with `webdav_enabled=false` previously disabled `/dav`; now `/dav` always 404 regardless of config. Log `WARN unknown config webdav_enabled ignored` not needed; silently ignore.
- **FS**: Host `./mounts` and `./data` remain; `internal/share/fuse` is not `internal/fusemount`, so no data loss for mounts. Existing WebDAV clients will get 404 for `/dav`.
- **Driver**: `drivers/WebDAV` remains functional (mount remote WebDAV as disk), tested separately.

## Tradeoffs

- **Delete vs flag**: Delete per user request "彻底移除", not flag-off. Flag would keep 27 files dead code.
- **Keep `drivers/WebDAV`**: Distinguish **对外 WebDAV 服务** (`internal/share/dav` → serve local files via WebDAV) vs **WebDAV 客户端** (`drivers/WebDAV` → mount remote). User said "文件共享" (share local files), so keep client driver.
- **Keep `mounts:shared`**: Docker volume propagation `shared` contains substring `share` but is not feature; must not be deleted or Docker mounts break.

## Rollout / Rollback

- **Rollout**: Single commit `refactor(share): remove file share (WebDAV dav/fuse) completely` → `docker build` → `docker run` on `:5211` → smoke `curl /api/health ok`, `curl -i /dav/ 404`, `curl -i /api/admin/webdav-config 404`, `grep -r -i "internal/share\|FileShare" 0`.
- **Rollback**: `git revert <commit>` restores 27 files + UI; DB configs with `webdav_enabled` still in `configs` table, so revert instantly functional.
- **Verification before merge**: `go vet ./...`, `npm run type-check`, `docker build`, `curl` smoke.

## File Map (Deletion Order)

1. `internal/share/` (rm -rf)
2. `internal/api/router.go` (remove import, davLog, routes)
3. `internal/api/auth.go` (remove handler)
4. `internal/adminauth/service.go` (remove KeyWebDAVEnabled etc.)
5. `internal/settings/registry.go` (remove KeyWebDAVCacheEnabled)
6. `internal/cache/webdav_keys.go` + `cache/keys.go` prefix + `cleaner.go` calls + `logx/module.go` ModuleWebDAV
7. `internal/playback/pick.go` WebDAV field (keep but make false) or remove; safest remove field and its branches
8. `web/src/components/admin/FileShareManagement.vue` (+ WebDAVSettings if separate) + `AdminView.vue` share page
9. `README.md` docs
10. `grep` sweep + `go vet` + `web build` + `docker build` + `curl` smoke

## Risks

- **Mounts:shared mis-deletion**: `grep -r "share"` will hit `mounts:shared` in `docker-compose.yml`; must not delete this line. Use precise pattern `FileShare`/`internal/share`/`WebDAV`/`webdav` for code, not generic `share`.
- **Playback WebDAV intent**: `intent.WebDAV` is used in `pick.go` and `redirect.go` for WebDAV clients needing original bytes. If removed, those clients will get proxied bytes; acceptable per "thorough removal" but must not break non-WebDAV playback.
- **AdminView share tab**: Removing `share` key from `AdminView.vue` must also remove `FileShareManagement` import and `adminPageLoaders.share`, otherwise `vue-tsc` will error `Cannot find module`.
