# Backend Development Guidelines

> Go 1.26.6 backend for LitePan — single Go module `litepan`, chi router, `modernc.org/sqlite`, FUSE, and pluggable drivers.

---

## Overview

LitePan backend is a monolithic Go service that mounts as single binary `./cmd/litepan/main.go`. All business logic lives under `internal/` with strict import guards enforced by `.golangci.yml`. Frontend is built separately in `web/` and embedded via `go:embed` into `internal/api/web`.

---

## Guidelines Index

| Guide | Description | Status |
|-------|-------------|--------|
| [Directory Structure](./directory-structure.md) | Module layout, `internal/` ownership, driver contract | Ready |
| [API Layering](./api-layering.md) | chi routing, `Deps` injection, handler/service/store separation | Ready |
| [Driver Development](./driver-development.md) | `driver.Meta/Lister/Config`, adding a new driver | Ready |
| [Database Guidelines](./database-guidelines.md) | `modernc.org/sqlite`, `store.DB`, migrations, `domain` repositories | Ready |
| [Error Handling](./error-handling.md) | `domain.Code*`, `domain.Errf`, HTTP mapping, auth failures | Ready |
| [Logging Guidelines](./logging-guidelines.md) | `internal/logx.Manager`, `slog`, levels, module tags | Ready |
| [Quality Guidelines](./quality-guidelines.md) | `make lint`, `golangci-lint`, `go test -race`, `depguard` rules | Ready |
| [Concurrency & Scheduling](./concurrency-and-scheduling.md) | `DelayController`, `singleflight`, `automation`, task lifecycles | Ready |

---

## Pre-Development Checklist

Before writing backend code, read the relevant guidelines:

- New HTTP endpoint → [api-layering.md](./api-layering.md) + [error-handling.md](./error-handling.md)
- New business module under `internal/` → [directory-structure.md](./directory-structure.md)
- New/changed driver in `drivers/` → [driver-development.md](./driver-development.md)
- DB schema or query → [database-guidelines.md](./database-guidelines.md)
- Logging or log retention → [logging-guidelines.md](./logging-guidelines.md)
- Changing `internal/domain` types → [directory-structure.md](./directory-structure.md) + [database-guidelines.md](./database-guidelines.md)
- Touching goroutines, intervals, rate limits → [concurrency-and-scheduling.md](./concurrency-and-scheduling.md)

Also read `guides/index.md` when work spans layers (API→Service→Driver→DB).

---

## Quality Check

After writing code:

1. `git diff --name-only` — every changed line must trace to the task PRD.
2. Run `make lint` (`golangci-lint run -c .golangci.yml ./...`) — must pass `depguard/errcheck/govet/staticcheck`.
3. `GOWORK=off go test -race ./...` — add regression test for bug fixes.
4. `GOWORK=off go vet ./...` implicitly covered by `govet`.
5. No new import violations: `pkg` must not import `internal`, `drivers` must not import `internal/file/auth/upload`, `internal/api` must not import `internal/store`.

---

**Language**: All docs and code comments are bilingual Chinese/English, but spec is in **English** per Trellis convention.
