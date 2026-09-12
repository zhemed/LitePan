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
- WebDAV: `internal/share/dav` 已移除（`2026-08-30 remove-share`）。`GET /internal/cover-source/{token}`（cover-extract）与 `Route("/tools/quarktv/cleanup/cover-extract")`、`Route("/emby")`、`/fnos` 均已移除，仅保留 `Route("/tools/local-upload", 4 handler)`；`Route("/cross-transfer", 5 handler: routes/scan/scan/stream/probe/execute)` 与 `Deps{ CrossTransfer}` 已在 `2026-08-30 nocross` 移除。

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

### 版本号：单一来源（2026-09-12，v0.0.44 起）

**`internal/buildinfo.Version` 是全站版本号的唯一真值。任何地方都不得再持有版本字面量。**

- 装配：`api.Deps.Version` ← `app/wire_http.go` 注入 `buildinfo.Version`（与 `backuprestore.Options{Version: ...}` 同一模式）
- 暴露：`GET /api/public/system-config` 返回 `version` 字段
- 前端：`web/src/stores/appInfo.ts` 运行期读取；组件经该 store 取值，**不设版本 fallback**（取不到就只显示应用名）
- `internal/httpx.AppVersion` / `DefaultUserAgent` 亦由该值派生（故它们是 `var` 而非 `const`；`drivers/115_Open` 的 `ossUserAgent` 已随之改为 `var`）
- 发版时**只改 `version.go` 一处**；`README.md` 与 `docker-compose.yml` 的镜像 tag 需同步为同一版本
- 历史教训：此前 `internal/buildinfo`、`web/src/version.ts`、`internal/httpx/user_agent.go` **三处**各自持有 `v0.5.2-Beta` 副本，靠"记得一起改"维持一致性 → 界面与 User-Agent 长期报着上游版本号。回归检查：`grep -rn "v0\.5\.2" web/src internal/ drivers/` 必须零命中。

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
