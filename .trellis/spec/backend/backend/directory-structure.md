# Directory Structure

> How Go code is organized in LitePan. Reality as of `go.mod: module litepan` / `go 1.26.6`.

---

## Top-Level Layout

```
.
├── cmd/litepan/main.go          # entry, flag parsing, app wiring
├── drivers/                     # pluggable cloud drivers (pure, no business deps)
│   ├── 115_Open/ 123_Open/ 139Cloud/ 189Cloud/ Baidu_Open/ Guangya/
│   │   Quark/ OneDrive/ OpenList/ WebDAV/ LocalFs/ template/
│   └── all.go, B-driver registration via internal/driver/registry.go
├── internal/                    # business monolith, 47 packages
│   ├── api/                     # chi router, handlers, go:embed web
│   ├── app/                     # wiring: wire_*.go, account_lifecycle.go
│   ├── config/                  # LITEPAN_* env loading, DB/Strm paths
│   ├── core/                    # shared core types (sparse)
│   ├── domain/                  # pure domain structs + repository interfaces
│   ├── store/                   # sqlite impl of domain repositories (modernc.org/sqlite)
│   ├── driver/                  # driver abstractions: Manager, DelayController, Config, registry
│   ├── file/ playback/ upload/  # file operations, streaming, upload manager
│   ├── strm/ strmscrape/ mediaorganize/ aiorganize/ classifyorganize/
│   ├── fusemount/ fusereadcache/ cache/ cacheretention/
│   ├── auth/ adminauth/ apikey/ account/ accountprofile/
│   ├── logx/ httpx/ eventbus/ notification/ announcement/
│   └── ... (automation, backuprestore, share, quarktv, etc.)
├── pkg/                         # pure utils: jsonvalue, secretkey, singleflight, strutil, timeutil, speedsmoother
├── web/                         # Vue 3 + Vite frontend, builds to internal/api/web
├── docs/pictures/               # README assets
├── Dockerfile / docker-compose.yml / Makefile / .golangci.yml
└── .trellis/                    # Trellis workflow (this spec)
```

Reference: `internal/app/app.go` wires 30+ services via `Deps` structs; `drivers/*/driver.go` each implements `driver.Meta`.

---

## 已移除（2026-08-30 精简后 현황）

- `internal/crosstransfer`（4 文件）`2026-08-30 nocross` 已移除，`cross_transfer_admin.go` 5 handler 与 `Route("/cross-transfer")` 均已删，仅 `local-upload` 保留
- `internal/strm`、`strmscrape`、`cacheretention`、`mediaorganize/classifyorganize/aiorganize`、`share/dav`、`embyproxy/fnosproxy/quarktv/spacecleanup/coverextract` 已 `rm -rf`，`internal/cache` 核心与 `mediaorganize/rules` 保留。镜像 `128M → 119M`，`CloudToolsPanel` 仅 `LocalUpload`。

## Module Ownership

| Path | Owns | Must NOT import |
|------|------|-----------------|
| `internal/domain` | Pure structs, `Repository` interfaces, `Code*` errors, constants | `internal/*` (enforced `domain-pure`) |
| `internal/store` | SQLite impl of `domain.*Repository`, `DB`, `Migrate`, helpers `wrapDB/boolToInt/parseTS` | `internal/api` |
| `internal/api` | HTTP layer: `chi.Router`, `Handler{Deps}`, `* _admin.go` handlers, `web` embed | `internal/store` (enforced `api-no-store`) |
| `internal/driver` | `Manager`, `Registry`, `DelayController`, `Config`, `Lister` interfaces | `internal/file/auth/upload` is forbidden for `drivers` |
| `drivers/*` | One driver per directory, `Config()` + `GetAddition()` + `ListFiles` etc. | `internal/file, internal/auth, internal/upload` (drivers-pure) |
| `pkg/*` | Pure helpers, no business logic | `litepan/internal` (pkg-must-be-pure) |
| `internal/app` | Lifecycle: `accountLifecycle.OnAccountDeleted` cascades to fuse/readCache/strm/retention/media/favorites/offline/uploads | — |

Guard source: `.golangci.yml` → `linters.settings.depguard.rules`.

---

## Naming Conventions

- **Package**: `internal/<feature>` lowercase, e.g. `strmscrape`, `cacheretention`, `fusereadcache`.
- **Handler file**: `internal/api/<feature>*.go` — e.g. `strm_admin.go`, `space_cleanup.go`, `cross_transfer_admin.go`.
- **Driver file**: `drivers/<DriverName>/{driver,config,auth,ops,transport,upload}.go`.
- **Service**: `internal/<feature>/service.go` with `type Service struct{...}` and `NewService`.
- **Store repo**: `internal/store/<feature>.go` with `type <feature>Repo struct{db *DB}` + `wrapDB`.
- **Domain type**: `internal/domain/<feature>.go` with `type <Feature> struct{ID int64}` + `type <Feature>Repository interface`.
- **Test**: `<name>_test.go` alongside impl, `t.Parallel()` when safe.

---

## Where New Code Goes

| Task | Location | Example |
|------|----------|---------|
| New HTTP endpoint | `internal/api/<feature>.go` + wire in `internal/api/router.go` (`Deps` → `Handler`) + `internal/app/wire_http.go` | `internal/api/cover_extract.go` |
| New business workflow | `internal/<feature>/service.go` + `internal/domain/<feature>.go` (types) + `internal/store/<feature>.go` (persistence) | `internal/mediaorganize/service.go` + `domain/media_organize.go` |
| New driver `FooCloud` | `drivers/FooCloud/{driver,config,auth,ops,transport}.go` + register in `internal/driver/registry.go` + `drivers/all.go` | `drivers/Quark/driver.go` (see `Config(){Name:"quark"}`) |
| Shared helper | `pkg/<util>/` only if no `internal` imports needed | `pkg/singleflight`, `pkg/strutil` |
| DB migration | `internal/store/migrate.go` + `store.Open.Migrate` | `internal/store/db.go` |
| Background job/interval | `internal/<feature>/service.go` with `context.Context` + `DelayController.Wait` + lifecycle hook in `internal/app/account_lifecycle.go` | `internal/cacheretention/coordinator.go` |

---

## Wire / Dependency Injection

- `internal/app/wire_*.go` (`wire_core.go`, `wire_http.go`, `wire_services.go`, `wire_store.go`, `wire_strm.go` etc.) assemble `Deps` structs.
- `internal/api.NewRouter(Deps)` receives only services/interfaces, never `*store.DB` directly.
- Adding a new service: 1) define in `internal/<feature>/service.go`, 2) construct in `wire_services.go`, 3) add field to `api.Deps` and `app.accountLifecycle` if it needs disable/delete hooks.

Reference: `internal/app/app.go: accountLifecycle{ fuse, readCache, strm, retention, media, favorites, offline, uploads, quarktv }` + `OnAccountDeleted` ordering.

---

## Frontend Build Integration

- `web/` is a separate `npm` package (`web/package.json` `litepan-web`).
- `web/vite.config.ts: build.outDir = "../internal/api/web"` — `npm run build` in `web/` writes directly into Go embed dir.
- `internal/api/router.go: //go:embed web` embeds `internal/api/web` — **never commit `internal/api/web` manually**, it is build artifact (but currently tracked; rebuild with `make build` which runs node stage in `Dockerfile`).

---

## Anti-Patterns

- **Putting business logic in `drivers/`** — drivers must stay pure API translators; logic like `strm` generation belongs in `internal/strm`.
- **Importing `internal/store` from `internal/api`** — pass `domain.AccountRepository` etc. via `Deps` instead.
- **Adding files directly under `internal/`** — always under `internal/<feature>/`.
- **Copying utils into `internal/`** — if reusable without business imports, put in `pkg/`.

---

## Examples

- Well-organized feature: `internal/strm/{service.go, coordinator.go}` + `domain/strm.go` + `store/strm*.go` + `api/strm_admin.go` — full vertical slice.
- Driver example: `drivers/115_Open/driver.go: type Driver struct{...}; func (d *Driver) Config() driver.Config { Name:"115_open", ... }`
- Lifecycle example: `internal/app/account_lifecycle.go: OnAccountDeleted` deletes in order `fuse → readCache → strm → retention → media → favorites → offline → uploads → quarktv`.
