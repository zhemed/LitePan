# Error Handling

> `internal/domain/errors.go` as single error taxonomy, mapped to HTTP via `internal/api`.
---

## Taxonomy

```go
// internal/domain/errors.go
const (
    CodeNotFound     = "not_found"
    CodeInvalid      = "invalid_arg"
    CodeConflict     = "conflict"
    CodeForbidden    = "forbidden"
    CodeUnauthorized = "unauthorized"
    CodeRateLimited  = "rate_limited"
    CodeInternal     = "internal"
)
func Errf(code string, format string, args ...any) error // wraps with code
func CodeOf(err error) string // extracts code, default CodeInternal
```

- Use `domain.Errf(domain.CodeInvalid, "name %q taken", name)` for user-facing validation.
- Use `domain.Errf(domain.CodeNotFound, "account %d not found", id)` for missing resources.
- `domain.TokenAuthFailureMessage(text)` etc. for auth-classification (`internal/domain/conn_error.go`).

Reference: `internal/domain/errors.go`, `internal/domain/conn_error.go`.

---

## Store Layer

- `internal/store/*: wrapDB(err)` converts `sql.ErrNoRows` → `domain.CodeNotFound`, sqlite unique → `CodeConflict`, busy/locked → `CodeInternal` with retry hint.
- Always `return wrapDB(err)` not raw `err`.
- `scanAccount`, `scanApiKey` etc. return `wrapDB` already; callers check `domain.CodeOf(err)==domain.CodeNotFound` when branching.

---

## Service Layer

- Validate early, return `domain.Errf` with `CodeInvalid`/`CodeConflict`.
- Example: `internal/account/service.go: if repo.NameTaken(...) { return domain.Errf(domain.CodeConflict, "名称已存在") }`
- Wrap driver errors: `fmt.Errorf("list %s failed: %w", parentID, err)` — preserve `%w` for `errors.Is`.
- Auth failures: set `domain.AuthStatus` + `AuthFailureKind` on `AuthState`, not just returned error (see `internal/auth/service.go`).

---

## API Layer

- `internal/api/errors.go: func writeDomainError(w http.ResponseWriter, err error)` maps:

| `domain.Code` | HTTP | JSON `code` |
|---|---|---|
| `CodeNotFound` | 404 | `not_found` |
| `CodeInvalid` | 400 | `invalid_arg` |
| `CodeConflict` | 409 | `conflict` |
| `CodeUnauthorized` | 401 | `unauthorized` |
| `CodeForbidden` | 403 | `forbidden` |
| `CodeRateLimited` | 429 | `rate_limited` |
| otherwise | 500 | `internal` |

- Handlers should `if err != nil { writeDomainError(w, err); return }` — never `http.Error` raw.
- For SSE/WebDAV, use `httpx` helpers + `logx` for non-JSON errors.

Reference: `internal/api/api_keys.go`, `internal/api/accounts.go` — all handlers end with `writeDomainError`.

---

## Driver Layer

- `internal/driver/errors` classification via `domain.TokenAuthFailureMessage`, `conn_error.go` for network vs auth.
- `Transport` retries on `CodeRateLimited` with `DelayController`, fails fast on `CodeForbidden`.

---

## Logging vs Returning

- **Return**: user-fixable (`invalid_arg`, `conflict`, `not_found`) → HTTP 4xx with message.
- **Log + return 500**: internal (`CodeInternal`) → `slog.Error("...", "error", err, "account_id", id)` via `logx.Manager.For(module)` then `domain.Errf(CodeInternal, "internal error")` to caller (hide details).

Example:

```go
if err := repo.Create(ctx, acc); err != nil {
    h.log.Error("create account failed", "error", err, "driver", acc.DriverType)
    writeDomainError(w, domain.Errf(domain.CodeInternal, "保存账号失败"))
    return
}
```

---

## Common Mistakes

- Returning `errors.New("account not found")` without code — API will return 500 instead of 404.
- Forgetting `wrapDB` in new `store/*` method — `sql.ErrNoRows` leaks as 500.
- Swallowing `%w` (`fmt.Errorf("failed")` without `%w`) — breaks `errors.Is/As` in scheduler.
- Using `panic` for validation — never; return `domain.Errf`.

---

## Testing

- Assert codes: `if domain.CodeOf(err) != domain.CodeConflict { t.Fatalf(...) }`
- Example: `internal/account/service_test.go`, `internal/store/account_repo_test.go` check `CodeNotFound`.
