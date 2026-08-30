# Driver Development

> Pluggable drivers under `drivers/`: 115_Open, 123_Open, 139Cloud, 189Cloud, Baidu_Open, Guangya, Quark, OneDrive, OpenList, WebDAV, LocalFs, template.

---

## Contract

All drivers implement `internal/driver.Meta` plus optional capabilities:

```go
// internal/driver/driver.go
type Config struct {
    Name string // e.g. "quark", "115_open", "localfs"
    DisplayName, Description string
    CardTags []string
    SortOrder int
    AuthLabel, CardColor, CardLogo string // frontend card
    DefaultRoot string // e.g. "0"
    AuthType AuthType // none/token/cookie
    OAuthName string // for unified OAuth refresh
    TokenLifetime time.Duration
    RefreshAdvance time.Duration
    HealthCheckInterval time.Duration
    ProvideHashes []string // sha1/md5 for cross-transfer rapid
    RapidUploadHashes []string
    UploadConflictPolicies []string
    QRDevices []FieldOption // scan login variants
    InternalExperimental bool
    SupportsAccountProfile bool
}
type Meta interface {
    Config() Config
    GetAddition() any // ptr to Addition struct for JSON form
    Init(ctx context.Context) error
    Drop(ctx context.Context) error
    Ping(ctx context.Context) error
}
type Lister interface { ListFiles(ctx context.Context, parentID string) ([]domain.FileItem, error) }
type FullListLister interface { ListAllFiles(ctx context.Context, rootID string) ([]FullListEntry, error); ResolveDirPath(...) }
type AccountProfileProvider interface { GetAccountProfile(ctx context.Context) (*domain.AccountProfile, error) }
// plus uploader, downloader, deleter, mover, etc. in drivers/<name>/ops.go
```

Reference: `internal/driver/driver.go`, `internal/driver/registry.go`, `drivers/template/driver.go`.

---

## Directory Layout per Driver

```
drivers/<Name>/
├── driver.go      # Config() + struct + Init/Drop/Ping
├── config.go      # Addition struct (JSON fields), FieldOption for forms
├── auth.go        # token/cookie refresh, QR login (qrlogin.go for some)
├── ops.go         # List/CreateDir/Rename/Delete/Move + File ops
├── transport.go   # http.Client, retry, rate-limit via driver.DelayController
├── upload.go      # UploadSession, RapidUpload check, chunking
├── download.go    # (LocalFs etc.)
├── profile.go     # GetAccountProfile if SupportsAccountProfile
├── models.go      # API models (XOR template)
└── qrlogin.go     # QR code polling where applicable
```

Reference: `drivers/Quark/{driver.go, config.go, auth.go, ops.go, transport.go, upload.go, qrlogin.go}`; `drivers/LocalFs/{driver,ops,move,download}.go`.

---

## Adding a New Driver `FooCloud`

1. **Copy template**: `cp -r drivers/template drivers/FooCloud` then edit `config.go: type Addition struct{...}` (fields become frontend form).
2. **Implement `driver.go`**:
   ```go
   func (d *Driver) Config() driver.Config {
       return driver.Config{
           Name: "foo_open", DisplayName: "Foo Cloud",
           AuthType: driver.AuthToken, OAuthName: "foo",
           TokenLifetime: 7200*time.Second, RefreshAdvance: 300*time.Second,
           ProvideHashes: []string{"sha1"}, RapidUploadHashes: []string{"sha1"},
       }
   }
   func (d *Driver) GetAddition() any { return &Addition{} }
   ```
3. **Fill `auth.go`, `transport.go`, `ops.go`, `upload.go`** — use `driver.DelayController.Gate(accountID).Wait(ctx, interval)` before each API call (see `internal/driver/delay.go`).
4. **Register**: add import in `drivers/all.go` and `internal/driver/registry.go: Registry.Register(new FooCloud.Driver)`. Sort via `Config.SortOrder`.
5. **Frontend**: card appears automatically via `GET /admin/drivers` → `web/src/stores/accounts.ts: visibleDrivers` respects `internal_experimental`.
6. **Test**: ping via `internal/account/ping.go` + `drivers/FooCloud/driver_test.go` if needed.

---

## Transport & Rate Limit

- Use `transport.go: http.Client` with `httpx` helpers + `driver.DelayController` per account.
- **Every driver API call** must:
  ```go
  if err := driver.WaitRequestInterval(ctx, d.gate, d.Addition.OperationDelay); err != nil { return err }
  // plus ExtraAPIDelay from ctx if set by task (automation/strm)
  ```
  Reference: `internal/driver/delay.go: WaitRequestInterval`.

- Auth refresh: `internal/driver/manager.go` + `internal/auth` scheduler handles `AuthToken` (proactive `RefreshAdvance`) vs `AuthCookie` (periodic `HealthCheckInterval` + `Ping`).

---

## Pure-Driver Rule (Critical)

`drivers/*` **must not** import `litepan/internal/file`, `internal/auth`, `internal/upload`, or `internal/store`. It may only import `litepan/internal/domain`, `internal/driver`, `internal/httpx`, `pkg/*`.

Enforced by `.golangci.yml: drivers-pure`. If you need business logic, put it in `internal/<feature>` and pass data via `domain.FileItem` / `domain.UploadTaskRecord`.

---

## Cross-Transfer / Rapid Upload

- `Config.ProvideHashes` (source can supply) + `Config.RapidUploadHashes` (dest accepts) drive `internal/crosstransfer/service.go`.
- Upload path: `internal/upload.Manager` orchestrates `drivers/*: upload.go UploadSession` + `domain.UploadTaskRecord` state machine (`Pending→Uploading→Success/Failed`).

---

## Common Mistakes

- Adding `time.Sleep` directly instead of `DelayController.Wait` — breaks per-account serialization.
- Missing `GetAddition()` pointer return → frontend form breaks (reflection).
- Hardcoding root ID `"0"` when driver uses `Config.DefaultRoot` — use config.
- Importing `internal/file` to reuse helpers — copy pure helpers to `pkg/` instead.

---

## Testing

- `internal/driver/manager_test.go`, `internal/driver/delay.go` unit tests pattern.
- Driver `Ping` and `ListFiles` should be mocked via `transport.go` interface for `go test -race ./drivers/FooCloud`.
