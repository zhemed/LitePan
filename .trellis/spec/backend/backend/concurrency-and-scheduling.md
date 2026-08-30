# Concurrency & Scheduling

> LitePan's background work: `DelayController`, `singleflight`, `automation`, and per-account task coordinators.

---

## Rate Limiting: `DelayController`

```go
// internal/driver/delay.go
type DelayController struct{ accounts map[int64]*accountDelay }
func (dc *DelayController) Gate(id int64) RequestIntervalGate // per-account gate
func WaitRequestInterval(ctx, gate, baseMS int) error {
    ms := baseMS + ExtraAPIDelayMS(ctx) // task-level extra from automation/strm
    return gate.Wait(ctx, time.Duration(ms)*time.Millisecond)
}
```

- **Every driver API call** must `WaitRequestInterval(ctx, gate, cfg.OperationDelay)` — see `drivers/Quark/transport.go`.
- Per-account serialization: `DelayController` ensures `interval` gap per `accountID`, not global.
- `WithExtraAPIDelay(ctx, ms)` injects task-level throttle (e.g. `strm` scan interval override).

Test: `internal/driver/delay_test.go`.

---

## Singleflight & Caching

- `pkg/singleflight` deduplicates concurrent `ListFiles`/`Ping` for same `accountID+path`.
- `internal/cache` + `internal/cacheretention` — `HitTracker`, `ScanDepth`, `RefreshInterval`, `ApiInterval`.
- Use `cache.Service` for file list memoization; `cacheretention.Coordinator` for periodic refresh.

---

## Coordinators & Task Lifecycle

| Coordinator | Package | State Machine |
|-------------|---------|---------------|
| `strm.Coordinator` | `internal/strm` | `pending → running → success/failed`, pause by account |
| `cacheretention.Coordinator` | `internal/cacheretention` | `running/paused`, `PauseByAccount` |
| `mediaorganize.Service` | `internal/mediaorganize` | preview → confirm → move |
| `upload.Manager` | `internal/upload` | `pending/uploading/success/failed`, respects `UploadTaskConcurrency` |
| `automation.Service` | `internal/automation` | `trigger(daily/interval/webhook) → actions[organize/strm/scrape/cache_clear/delay/emby_refresh]` |

All coordinators integrate with `internal/app/account_lifecycle.go`:

```go
func (a accountLifecycle) OnAccountDisabled(ctx, id) { strm.PauseByAccount(id); retention.PauseByAccount(id) }
func (a accountLifecycle) OnAccountDeleted(ctx, id) { fuse.OnAccountDeleted(id); readCache.InvalidateAccount(id); strm.RemoveTasksByAccount(id); ... }
```

Reference: `internal/app/account_lifecycle.go:87`, `internal/app/app.go`.

---

## Automation Rules

- `internal/automation/service.go` evaluates `AutomationRule{TriggerType, TriggerConfig, Actions}`.
- `domain.AutomationAction` has `Condition: always/prev_success/prev_failed`.
- Schedule via `NextRunAt` + `internal/automation/scheduler.go` (interval/daily ticker).

---

## Context & Cancellation

- Long-running jobs (STRM scan, cache retention, media organize) must:
  ```go
  select {
  case <-ctx.Done(): return ctx.Err()
  case <-time.After(interval): // work
  }
  ```
  and check `ctx.Err()` before DB writes.
- `accountLifecycle.OnAccountDisabled` cancels via `context.WithCancel` per account.

---

## Anti-Patterns

- **Global `time.Sleep`** in driver or service — use `DelayController.Wait` with `ctx`.
- **Unbounded goroutines** per file — use `pkg/singleflight` + bounded worker pool via `UploadTaskConcurrency` config.
- **Ignoring `ExtraAPIDelay`** — STRM scans must respect user-configured per-task interval.

---

## Testing

- `internal/cacheretention/*_test.go`, `internal/strm/*_test.go` use `context.WithCancel` + `t.TempDir`.
- Race: `go test -race ./internal/...` must pass — coordinators share maps behind mutex.

---

## Adding a New Background Task

1. Define `domain.FooTask{Status, PausedReason, ...}` + `FooTaskRepository` in `domain/`.
2. Implement `store/foo.go` + migration.
3. Implement `internal/foo.Service` with `Start(ctx)/PauseByAccount/RemoveTasksByAccount`.
4. Wire in `internal/app/wire_services.go` + `app.accountLifecycle`.
5. Expose via `internal/api/foo_admin.go` handlers.

Reference: `internal/strm/service.go` as canonical example.

