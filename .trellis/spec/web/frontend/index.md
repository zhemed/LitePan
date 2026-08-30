# Frontend Development Guidelines

> Vue 3 + Vite frontend in `web/` — builds into Go embed `internal/api/web` for single-binary deployment.

---

## Overview

- `web/package.json: "litepan-web"` — `vue 3.5.41`, `vue-router 5.2.0`, `pinia 4.0.3`, `vite 8.2.2`, `vue-tsc 3.3.11`, `hls.js`, `mpegts.js`, `pdfjs-dist`, `docx-preview`, etc.
- Build: `vite build` → `../internal/api/web` (see `web/vite.config.ts: outDir`), compressed via `scripts/compress-build.mjs`, embedded by `internal/api/router.go: //go:embed web`.
- Dev: `cd web && npm run dev` proxies `/api` → `http://127.0.0.1:5211` (`LITEPAN_API_PROXY`).

---

## Guidelines Index

| Guide | Description | Status |
|-------|-------------|--------|
| [Directory Structure](./directory-structure.md) | `src/{api,stores,views,router,components,composables}` | Ready |
| [State & Routing](./state-and-routing.md) | Pinia stores, `vue-router` guards, auth flow | Ready |
| [API Client](./api-client.md) | `src/api/client.ts` (`http`), typed DTOs in `api/types.ts` | Ready |
| [Component Guidelines](./component-guidelines.md) | SFC structure, props/emits, styling, media previews | Ready |
| [Quality Guidelines](./quality-guidelines.md) | `vue-tsc`, Vite build, manual chunks, type safety | Ready |

---

## Pre-Development Checklist

Before writing frontend code:

- New page → [directory-structure.md](./directory-structure.md) + [state-and-routing.md](./state-and-routing.md)
- New API call → [api-client.md](./api-client.md) — add typed method to `src/api/*.ts`, not inline `fetch`
- Shared state → [state-and-routing.md](./state-and-routing.md) — use Pinia store with `inflightLoad` dedup
- Component/reuse → [component-guidelines.md](./component-guidelines.md) — check `components/` for existing
- Changing `api/types.ts` → also update `internal/api` DTO + re-run `npm run type-check`

Also read `guides/index.md` for cross-layer data flow thinking.

---

## Quality Check

```bash
cd web
npm run type-check   # vue-tsc -b
npm run build        # vite build → ../internal/api/web
```

- No `any` in `src/api/types.ts` without comment.
- Stores must deduplicate concurrent loads (`inflightLoad` pattern).
- Router guards must handle `public_index_enabled` + `is_admin` correctly.

---

**Language**: Source comments may be Chinese, spec in English.
