# API Layering

> chi-based HTTP layer: how requests flow from `net/http` → `internal/api` → `internal/<service>` → `internal/store`.

---

## Stack

- Router: `github.com/go-chi/chi/v5` + `chi/middleware` in `internal/api/router.go: NewRouter(Deps)`.
- Auth: `internal/adminauth.Service` + `internal/api/admin_middleware.go` (`Authorization: Bearer <key>` or cookie session, `X-API-Key`).
- Embed: `//go:embed web` `webFS embed.FS` serves `internal/api/web/index.html` as SPA fallback.
- Handler pattern: `type Handler struct{ log *slog.Logger, accountSvc *account.Service, ... }` + method per route, constructed once per `NewRouter`.

---

## Request Flow

```
Client → chi.Router (Recoverer, RequestID, Logger)
  → admin_middleware.go (session/apiKey check, sets ctx admin)
  → Handler.<Feature>(w,r) in internal/api/<feature>.go
    → internal/<feature>.Service (business validation)
      → domain.Repository (interface, e.g. domain.AccountRepository)
        → internal/store/<feature>.go (sqlite, wrapDB)
    → resp helpers: api/errors.go, api/commit_writer.go, api/request.go
  → JSON/stream/SSE response
```

Reference: `internal/api/router.go: type Deps struct{ AccountSvc *account.Service, Files *file.Service, Uploads *upload.Manager, ... DataDir, StrmDir }` + `NewRouter` registers `GET /api/...`, `POST /admin/...`, `GET /files`, `WebDAV`.

---

## Handler Conventions

```go
// internal/api/accounts.go — typical handler shape
func (h *Handler) ListAccounts(w http.ResponseWriter, r *http.Request) {
    // 1. Parse & validate (use api/request.go helpers where applicable)
    // 2. Call service: h.accountSvc.List(r.Context())
    // 3. Map domain error → HTTP via domain.Code* helpers (see error-handling.md)
    // 4. Write JSON via resp.go helpers: writeJSON(w, http.StatusOK, data)
}
```

- **No `store` import**: handlers must import `litepan/internal/domain` and service interfaces only. Direct `store` import is blocked by `golangci depguard: api-no-store`.
- **No business logic**: validation that touches DB or cross-service belongs in `internal/<feature>/service.go`, not handler.
- **Context**: always `r.Context()`; add `driver.WithExtraAPIDelay(ctx, ms)` if caller supplied per-task delay (see `pkg`).
- **Streaming**: use `api/commit_writer.go` for large file transfer; `internal/playback` for HLS/mpegts; `internal/share/dav` for WebDAV.

---

## Route Registration

- All routes wired in `internal/api/router.go: NewRouter`.
- Public: `GET /auth/status`, `POST /auth/login`, `GET /` (SPA), `GET /api/public/*`.
- Admin: grouped under `r.Route("/admin", func(r chi.Router){ r.Use(h.adminOnly) ... })` — see `internal/api/admin_middleware.go`.
- WebDAV: `internal/share/dav` 已移除（`2026-08-30 remove-share`）。`GET /internal/cover-source/{token}`（cover-extract）与 `Route("/tools/quarktv/cleanup/cover-extract")`、`Route("/emby")`、`/fnos` 均已移除，仅保留 `Route("/tools/local-upload", 4 handler)`。

Example registration:

```go
r.Get("/admin/accounts", h.ListAccounts)
r.Post("/admin/accounts", h.CreateAccount)
r.Post("/admin/accounts/{id}/toggle", h.ToggleAccount)
```

Reference: `internal/api/accounts.go:6205`, `admin_middleware.go:1187`.

---

## DTO & Validation

- Request payloads: `json.RawMessage` for dynamic `Actions`/`TriggerConfig` (e.g. `domain.AutomationRule.Actions json.RawMessage`), otherwise typed struct per handler.
- Validation: handler does `strings.TrimSpace` + domain helper `scan*` rejects empty/invalid; service does deeper checks (e.g. `store.accountRepo.NameTaken`).
- Response: `resp.go: writeJSON(w, code, payload)` + `api/errors.go: writeDomainError(w, err)` maps `domain.CodeNotFound→404`, `CodeInvalid→400`, `CodeConflict→409`.

---

## Service Injection via Deps

- `internal/api/router.go: type Deps struct{ Logs *logx.Manager, AccountSvc *account.Service, Files *file.Service, ... OnSettingsUpdated func(map[string]string) }` **(2026-08-30 精简后，仅保留 LocalUpload，`EmbyProxy/FnosProxy/QuarkTV/SpaceCleanup/CoverExtract` 已移除，`Route("/tools/*")` 仅 `local-upload`)**
- `internal/app/wire_http.go` builds `Deps` from `store.New(db)` + all services.
- Adding new dependency: extend `Deps`, add field to `Handler`, wire in `app/wire_http.go`, update `api/router_test.go` if needed.

---

## Anti-Patterns

- **Fat handlers**: >80 line handler that does DB queries or driver calls directly — extract to service.
- **Leaking driver types to frontend**: handler must translate `drivers/*` models to `api/types` DTOs.
- **Ignoring `r.Context()` cancellation**: driver calls must respect `ctx.Done()` (see `driver/delay.go: accountGate.Wait`).
- **Writing to `internal/api/web` manually**: only `web/` build writes there via `vite outDir`.

---

## Testing

- `internal/api/*_test.go` uses `httptest.NewRequest` + `chi.NewRouter` seeded with mock services.
- Example: `internal/api/announcement_test.go`, `api/router_test.go` verify middleware and embed fallback.
- Run `GOWORK=off go test -race ./internal/api -run TestX` per package; full suite via `make test`.

---

## Cross-Layer Thinking

- Adding a field that flows `domain → store → service → api → web/api/*.ts`: update `domain/<feature>.go` first, then `store` scan helper, then service, then handler DTO, then frontend type — all in one PR, ordered bottom-up. See `guides/cross-layer-thinking-guide.md`.
