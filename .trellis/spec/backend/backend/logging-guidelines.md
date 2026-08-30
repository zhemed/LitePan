# Logging Guidelines

> `internal/logx.Manager` = stdout `slog.TextHandler` + async file queue → `Storage` → `litepan.db` + retention.

---

## Levels

```go
// internal/logx/logx.go
const LevelDebug=10, LevelInfo=20, LevelWarn=30, LevelError=40
func ParseLevel(s string) slog.Level // "debug"→Debug, "warn"→Warn, "error"→Error, default Info
func LevelToInt(l slog.Level) int
func LevelName(l int) string // DEBUG/INFO/WARNING/ERROR
func LevelEmoji(l int) string // 🔍/ℹ️/⚠️/❌ (frontend uses)
```

- Default `LITEPAN_LOG_LEVEL=info` (`internal/config/config.go`).
- Enable debug via `LITEPAN_LOG_LEVEL=debug` or `?debug` param in `internal/api/logs.go`.

Reference: `internal/logx/logx.go`, `internal/config/config.go`.

---

## Manager Creation

```go
// internal/logx/manager.go
mgr, _ := logx.New(logx.Options{
    Dir: filepath.Join(cfg.DataDir, "log"),
    Level: cfg.LogLevel,
    Stdout: os.Stdout,
})
defer mgr.Close()
log := mgr.For(logx.ModuleAPI) // or ModuleStrm, ModuleFuse, ModuleCache ...
```

- `Options.DisableFile` for tests; pass `Dir: t.TempDir()` in tests.
- `Manager.For(module)` returns `*slog.Logger` tagged with `module` attr.
- Modules: `ModuleSystem`, `ModuleAPI`, `ModuleStrm`, `ModuleFuse`, `ModuleCache`, `ModuleAuth`, etc. in `internal/logx/module.go`.

---

## Writing Logs

```go
log.Info("strm task started", "task_id", id, "account_id", accountID, "path", task.Path)
log.Warn("rate limited, retrying", "account_id", id, "driver", "quark", "retry_after", delay)
log.Error("refresh failed", "error", err, "account_id", accountID, "driver_name", driverName)

// With module automatically injected via recordToEntry:
// internal/logx/handler.go: switch key { case "module": e.Module=..., case "account_id": e.AccountID=..., case "driver_name": e.DriverName=... }
```

- **Always include**: `account_id` when account-scoped, `driver_name` when driver-scoped, `task_id`/`path` for tasks.
- **Never log**: tokens, cookies, raw `AccessToken`/`RefreshToken`/`Cookie` — they are filtered by `logx` attrs.

Reference: `internal/logx/handler.go: recordToEntry`, `internal/logx/storage.go: Enqueue`.

---

## Storage & Query

- File queue → SQLite `logs` table via `Storage.Enqueue(Entry)` async batch.
- API: `GET /admin/logs?level=error&module=strm&keyword=115&limit=50` via `internal/api/logs.go` → `logx.Manager.Query(QueryFilter)`.
- Retention: `internal/logx/manager.go: cleanupRetention` + `LITEPAN_LOG_RETENTION_DAYS` (default 30) via `internal/api/logs.go`.
- Frontend: `web/src/views/AdminView.vue` Log panel + `web/src/api/logs.ts`.

---

## Levels Usage

| Level | When | Example |
|-------|------|---------|
| DEBUG | High-volume tracing, only when `LITEPAN_LOG_LEVEL=debug` | `ListFiles` each dir, FUSE read cache hit/miss |
| INFO | Normal lifecycle | `account created`, `strm generation started`, `upload success` |
| WARN | Recoverable, retryable | `auth cooldown`, `rate limited`, `cache skip due to window` |
| ERROR | Requires attention, persisted as RecentErrors | `refresh failed`, `DB write failed`, `driver Ping failed` |

- `ERROR` increments `logx.Stats.RecentErrors` + `RecentUnacknowledgedErrors` shown in `GET /admin/logs/stats`.

---

## Common Mistakes

- Using `log.Printf` or `fmt.Println` — always `mgr.For(...).Info/Warn/Error`.
- Logging with `slog.LevelDebug` but leaving `LITEPAN_LOG_LEVEL=info` — will be silent; confirm level before diagnosing missing logs.
- Including `AccessToken` in log fields — use `account_id` + `driver_name` only.
- Calling `slog.Default()` directly in `internal/api` — use `d.Logs.For(logx.ModuleAPI)` via `Deps`.

---

## Testing

```go
dir := t.TempDir()
mgr, _ := logx.New(logx.Options{Dir: dir, Level: "debug", DisableStdout: true})
defer mgr.Close()
log := mgr.For(logx.ModuleTest)
log.Info("hello", "account_id", 1)
entries, _ := mgr.Query(ctx, logx.QueryFilter{Keyword: "hello", Limit: 10})
```

Reference: `internal/logx/manager_test.go`, `internal/api/logs.go`.
