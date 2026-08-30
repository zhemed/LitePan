# Database Guidelines

> `modernc.org/sqlite v1.57.0` (CGO_ENABLED=0), `internal/store` as only implementation of `internal/domain` repositories.

---

## Engine & Connection

- Driver: `modernc.org/sqlite` pure Go, `GOTOOLCHAIN=local`, `CGO_ENABLED=0` (see `Dockerfile` build: `go build -tags fuse`).
- Entrypoint: `internal/store/db.go: func Open(ctx, Options{DataDir, DBPath, Memory bool}) (*DB, error)` — creates `db.read` + `db.write` `*sql.DB` with `journal=WAL`.
- Config: `LITEPAN_DATA_DIR` → `filepath.Join(v, "litepan.db")` (override via `LITEPAN_DB_PATH`) in `internal/config/config.go`.

Reference: `internal/store/db.go`, `internal/config/config.go: Load()`.

---

## Schema & Migrations

- Migrations live in `internal/store/migrate.go: func (db *DB) Migrate(ctx) error` — idempotent `CREATE TABLE IF NOT EXISTS` + `ALTER TABLE` additions. No external migration tool.
- Tables include: `cloud_accounts`, `account_auth_states`, `strm_tasks`, `cache_retention_tasks`, `media_organize_tasks`, `upload_tasks`, `api_keys`, `account_profiles`, `system_configs`, etc.
- Adding a migration: append `_, err = tx.ExecContext(ctx, `ALTER TABLE x ADD COLUMN y TEXT`)` guarded by `SELECT` of `pragma_table_info`; keep `Migrate` ordered chronologically; test via `store.Open(Memory:true)` in `*_test.go`.

Reference: `internal/store/migrate.go`, `internal/store/db.go: wrapDB`.

---

## Repository Pattern

- `internal/domain/*.go` declares `type FooRepository interface{ List/Create/Get/Update/Delete... }` + structs.
- `internal/store/*.go` implements one file per aggregate: `accountRepo`, `authStateRepo`, `apiKeyRepo`, `strmRepo`, etc.
- **Store is the only implementor**: `internal/api` and services receive `domain.FooRepository` via `Deps`/wiring, never `*store.DB` directly.

Example:

```go
// domain
type Account struct{ ID int64; Name string; DriverType string; Config string; IsActive bool }
type AccountRepository interface { List(ctx) ([]*Account, error); Create(ctx, *Account) (int64,error) }

// store
type accountRepo struct{ db *DB }
func (r *accountRepo) Create(ctx, a *domain.Account) (int64,error){
    res, err := r.db.write.ExecContext(ctx, `INSERT INTO cloud_accounts(name,driver_type,config,is_active,...) VALUES (?,?,?,?,?,?)`, a.Name, a.DriverType, a.Config, boolToInt(a.IsActive), ...)
    return res.LastInsertId(), wrapDB(err)
}
```

Reference: `internal/domain/account.go`, `internal/store/account_repo.go`, `internal/store/api_key_repo.go`.

---

## Helpers

- `wrapDB(err)` maps `sql.ErrNoRows` → `domain.Errf(domain.CodeNotFound)`, sqlite busy → retryable.
- `boolToInt(b) int` / `parseTS(sql.NullString) time.Time` / `tsValue(time.Time) any` — use them, don't inline.
- `selectAccountCols = SELECT id,name,driver_type,...` constant per repo, reused in `Get/List`.
- Transactions: `tx, _ := db.write.BeginTx(ctx, nil); defer tx.Rollback(); tx.ExecContext(...); return tx.Commit()` — see `accountRepo.SetDefault`.

---

## Query Conventions

- Always `Context` variants: `QueryContext`, `ExecContext`, `QueryRowContext`.
- Reads via `db.read`, writes via `db.write` (same underlying file, split for clarity).
- `WHERE LOWER(name)=LOWER(?)` for case-insensitive uniqueness (see `NameTaken`).
- Ordering: `ORDER BY is_default DESC, sort_order, id` for accounts; `ORDER BY id DESC` for keys.

---

## Testing

- Use `store.Open(ctx, store.Options{Memory:true})` + `db.Migrate(ctx)` + `store.New(db)` to get all repos in-memory.
- Example: `internal/app/account_lifecycle_test.go` creates `favorites.Service` backed by file db, but `store` tests use memory.

```go
db, _ := store.Open(ctx, store.Options{Memory:true})
defer db.Close()
_ = db.Migrate(ctx)
repos := store.New(db)
id, _ := repos.Accounts.Create(ctx, &domain.Account{Name:"test", DriverType:"115_open"})
```

Run with `GOWORK=off go test -race ./internal/store`.

---

## Anti-Patterns

- **Importing `store` from `api`** — blocked by `depguard`; pass repo interface.
- **Raw `sql` in service** — service must call repo methods, not `db.write.ExecContext`.
- **Adding GORM or other ORM** — LitePan uses raw `database/sql` only.
- **Ignoring `wrapDB`** — losing `CodeNotFound` mapping breaks `api/errors.go` → 404.

---

## Adding a New Entity `Bar`

1. Define `type Bar struct{...}` + `type BarRepository interface` in `internal/domain/bar.go`.
2. Implement `type barRepo struct{db *DB}` in `internal/store/bar.go` + add to `internal/store/store.go: type Stores struct{ Bars BarRepository }`.
3. Add migration in `migrate.go: CREATE TABLE bars (...)`.
4. Wire via `internal/app/wire_store.go` → service receives `domain.BarRepository`.
