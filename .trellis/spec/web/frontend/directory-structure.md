# Frontend Directory Structure

> `web/` is a standalone Vite app that builds into Go's `internal/api/web`.

---

## Layout

```
web/
├── package.json / package-lock.json
├── vite.config.ts          # outDir ../internal/api/web, proxy /api, manualChunks
├── tsconfig.json / env.d.ts
├── index.html
├── public/                 # static assets copied verbatim
├── scripts/compress-build.mjs # gzip/brotli after vite build
└── src/
    ├── main.ts             # createApp(App).use(pinia).use(router).mount
    ├── App.vue             # root layout, <router-view>
    ├── version.ts          # build version
    ├── router/index.ts     # createRouter, beforeEach guards
    ├── stores/             # Pinia stores (one per domain)
    │   ├── auth.ts         # sessionAdmin, mustChangePassword, publicIndexEnabled
    │   ├── accounts.ts     # accounts, drivers, visibleDrivers, loadAccounts()
    │   └── browser.ts      # currentAccountId, breadcrumb, files, loading
    ├── api/                # typed HTTP clients
    │   ├── client.ts       # http.get/post/put/del/form, baseURL /api, error mapping
    │   ├── types.ts        # shared DTOs (Account, FileItem, DriverInfo, etc.)
    │   ├── accounts.ts / auth.ts / files.ts / strm.ts / ...
    │   └── public.ts       # publicApi.listAccounts for non-admin
    ├── views/              # route pages
    │   ├── IndexView.vue   # file browser /
    │   ├── LoginView.vue   # /login
    │   └── AdminView.vue   # /admin (heavy, 11k lines, tabbed)
    ├── components/         # reusable SFCs
    │   ├── FileBrowser.vue, Breadcrumb.vue, AccountSelector.vue, ...
    │   └── media/ (PdfViewer, DocxPreview, VideoPlayer via hls.js/mpegts)
    ├── composables/        # composition functions
    │   ├── useToast.ts, useDeveloperUnlock.ts, useStrmTask.ts, ...
    ├── constants/          # enums, driver meta
    ├── types/              # global types
    ├── assets/             # images, logos
    ├── styles/             # global CSS
    └── utils/              # formatters, validators
```

Reference: `web/src/main.ts`, `web/src/router/index.ts`, `web/vite.config.ts`.

---

## Module Ownership

| Path | Owns |
|------|------|
| `src/api/*.ts` | One file per backend domain, exporting `const fooApi = { list: ()=>http.get(...) }`. Uses `api/client.ts` only. |
| `src/stores/*.ts` | Pinia store per domain, `ref` + `computed` + async actions. Handles dedup `inflightLoad`. |
| `src/views/*.vue` | Page component per route, composes stores + components. `AdminView.vue` is the admin tab container. |
| `src/components/` | Pure presentational or media viewers (PdfViewer, HlsPlayer). No direct `http` calls — receive props. |
| `src/composables/` | `useX` functions that wrap store/api + UI state (toast, dev unlock, pagination). |
| `src/router/` | Single `router/index.ts` with `beforeEach` auth gate. |

---

## Adding New Code

| Task | Location |
|------|----------|
| New page | `src/views/FooView.vue` + add route in `src/router/index.ts` + link in `App.vue` if needed |
| New API domain | `src/api/foo.ts` + extend `src/api/types.ts` DTOs |
| Shared state | `src/stores/foo.ts` (`defineStore("foo", ()=>"`) |
| Reusable UI | `src/components/FooCard.vue` |
| Logic reuse | `src/composables/useFoo.ts` |

---

## Build Integration (Critical)

- `web/vite.config.ts`:

```ts
build: {
  outDir: "../internal/api/web",
  emptyOutDir: true,
  chunkSizeWarningLimit: 3200,
  rollupOptions: {
    output: {
      manualChunks(id){
        if(id.includes('node_modules/three')) return 'three-vendor';
        if(id.includes('node_modules/vue')||id.includes('pinia')||id.includes('vue-router')) return 'vue-vendor';
      }
    }
  }
}
server: { proxy: { "/api": { target: "http://127.0.0.1:5211" } } }
```

- `npm run build` does `vue-tsc -b && vite build && node scripts/compress-build.mjs`.
- Result is consumed by Go: `internal/api/router.go: //go:embed web` — **do not edit `internal/api/web` directly**.
- CI/Docker: `Dockerfile` runs `npm ci && npm run build` in first stage (`FROM node:20`), then `COPY --from=web /src/internal/api/web`.

---

## Anti-Patterns

- Importing `web/node_modules` paths outside `web/` — backend never imports frontend.
- Adding `fetch("/api/...")` inline in component — use `src/api/*.ts`.
- Putting business state in component `ref` instead of Pinia when shared across routes.
- Forgetting `vite.config.ts` manualChunks — `three` etc. would bloat main chunk.

---

## Examples

- Store dedup: `web/src/stores/accounts.ts: let inflightLoad; function loadAccounts(){ if(inflightLoad) return inflightLoad; ... }`
- Router guard: `web/src/router/index.ts: beforeEach(async(to)=>{ if(to.meta.requiresAuth){ status=await fetchAuthStatus(); if(!status.is_admin) return "/login" }})`
- API client: `web/src/api/client.ts: export const http={ get:(url)=>fetch(url).then(r=>r.json()), post: ... }` (typed generics).
