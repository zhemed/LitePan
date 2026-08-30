# Quality Guidelines

> Enforced by `.golangci.yml`, `Makefile`, and `depguard` — every PR must pass `make lint` + `make test`.

---

## Lint (`make lint`)

```bash
GOWORK=off golangci-lint run -c .golangci.yml ./...
```

Configured in `.golangci.yml` (v2):

```yaml
run:
  go: "1.26.4"
  modules-download-mode: readonly
linters:
  enable: [depguard, errcheck, govet, staticcheck]
  settings:
    depguard:
      rules:
        api-no-store:    api/** must not import litepan/internal/store
        core-no-api:     file/playback/upload must not import api
        drivers-pure:    drivers/** must not import file/auth/upload
        domain-pure:     domain/** must not import internal/*
        no-driver-leak:  internal/** must not import drivers (use driver interface)
        pkg-must-be-pure: pkg/** must not import internal
```

- **All PRs must pass** — CI runs `make lint` with `GOLANGCI_LINT_VERSION=v2.12.2`.
- Fix before review: `make lint` must be silent.
- **Do not** `//nolint` without comment explaining why.

---

## Tests (`make test`)

```bash
GOWORK=off go test -race ./...
```

- `-race` required — detects `DelayController`, `singleflight`, `automation` races.
- Package-level: `go test -race ./internal/store -run TestAccount` for fast feedback.
- Coverage not enforced, but new pure functions need unit tests; bug fixes need regression tests.

Test style (see `internal/adminauth/service_test.go`, `internal/store/account_repo_test.go`):

```go
func TestXxx(t *testing.T){
    t.Parallel()
    ctx := context.Background()
    db, _ := store.Open(ctx, store.Options{Memory:true})
    defer db.Close()
    _ = db.Migrate(ctx)
    // ...
}
```

For `logx`/`favorites`: use `t.TempDir()` file DB; for `store`: `Memory:true`.

---

## Type & Build

```bash
GOWORK=off go vet ./...                # covered by govet
GOWORK=off go build -tags fuse ./...   # -tags fuse required (FUSE optional)
GOWORK=off go build ./...              # build-nofuse
```

- `Dockerfile` builds with `CGO_ENABLED=0 GOTOOLCHAIN=local GOPROXY=https://goproxy.cn,direct go build -tags "${BUILD_TAGS}"`.
- Frontend type-check: `cd web && npm run type-check` (`vue-tsc -b`).

---

## 已移除门禁示例（2026-08-30 精简）

- `grep -r "embyproxy|quarktv|spacecleanup|coverextract" --include="*.go" | wc -l ==0`（增强工具已删，仅 `local_upload` 保留）。

## Import Guard Cheat Sheet

| From → To | Allowed? | Why |
|-----------|----------|-----|
| `internal/api` → `internal/store` | ❌ | API must use domain repos via Deps |
| `internal/file` → `internal/api` | ❌ | File service is lower layer |
| `drivers/*` → `internal/file/auth/upload` | ❌ | Drivers must stay pure |
| `internal/domain` → `internal/*` | ❌ | Domain is pure |
| `pkg/*` → `internal/*` | ❌ | pkg is shared util |
| `internal/*` → `drivers` | ❌ | Use `internal/driver` interface |

If `make lint` reports `depguard`, refactor to pass `domain.*Repository` or move helper to `pkg/`.

---

## Code Standards

- **Surgical changes**: touch only what PRD requires; match existing style even if you'd differ.
- **Min code**: no abstractions for single use; no speculative flexibility (`pkg/manager.go` is intentional shared).
- **Error handling**: always `return wrapDB(err)` / `domain.Errf`; check `errcheck` — unhandled `err` fails lint.
- **Logging**: use `logx.Manager.For(module)` not `slog.Default`; include `account_id`/`driver_name`.
- **Context**: propagate `ctx` from `Handler` → Service → Store/Driver; respect `ctx.Done()` in loops.
- **Naming**: `Service` struct + `NewService`, `Handler` for api, `Manager` for orchestrators, `Coordinator` for task schedulers.

---

## Common Mistakes

- Adding `import "litepan/drivers/Quark"` in `internal/*` — use `driver.Registry.Get("quark")`.
- Forgetting `GOWORK=off` — repo has no `go.work`, but `GOWORK=off` ensures correct module mode.
- Running `go test ./...` without `-race` — races in `DelayController` slip through.
- Leaving `web/node_modules` or `dist/*.tar.gz` untracked but built — they are `.gitignore`'d, don't `git add`.
- Editing `web/src` without `npm run type-check` — breaks `vite build` (`vue-tsc` strict).

---

## Pre-Commit Verification

Before `git commit` or `task.py archive`:

```bash
make lint
GOWORK=off go test -race ./...   # or at least changed packages
cd web && npm run type-check && npm run build
git diff --name-only              # review every changed line maps to PRD
```

Trellis hook `trellis-check` also runs this suite — fix locally first.

---

## Frontend Quality (web/)

```bash
cd web && npm run type-check   # vue-tsc -b
cd web && npm run build        # vite build + compress-build.mjs → ../internal/api/web
```

- No `any` without justification; use `api/types.ts` for shared DTOs.
- Pinia stores must handle `inflightLoad` deduplication (see `stores/accounts.ts`).
