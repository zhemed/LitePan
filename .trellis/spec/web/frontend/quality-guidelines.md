# Frontend Quality Guidelines

> `web/` quality gates: `vue-tsc` + `vite build` + manual chunk policy.

---

## Commands

```bash
cd web
npm run type-check  # vue-tsc -b  (no emit, strict)
npm run build       # vue-tsc -b && vite build && node scripts/compress-build.mjs
npm run dev         # vite --port 5173, proxy /api → 127.0.0.1:5211
```

- **All PRs that touch `web/` must pass** `npm run type-check` and `npm run build`.
- Build output `internal/api/web` must be rebuilt before Go `make build` if frontend changed — `Dockerfile` does this automatically (`FROM node:20` → `npm ci && npm run build`).

---

## Type Safety

- `tsconfig.json: strict:true`, `@types/*` for `markdown-it`, `node`.
- `web/src/api/types.ts` is shared DTO contract — no `any` payloads.
- Props/emits typed via `defineProps<{...}>()` and `defineEmits<{(e:"select",id:string):void}>()`.
- `pinia` stores typed `ref<T>` — avoid `ref<any>`.

Example violation:

```ts
// Bad
function fetchFoo(): Promise<any> { return http.get("/admin/foo"); }
// Good
function fetchFoo(): Promise<FooItem[]> { return http.get<FooItem[]>("/admin/foo"); }
```

---

## Build Policy

- `vite.config.ts`:

```ts
build: {
  outDir: "../internal/api/web",
  emptyOutDir: true,
  chunkSizeWarningLimit: 3200,
  rollupOptions: {
    output: {
      manualChunks(id){
        if(id.includes('node_modules/three')) return 'three-vendor';
        if(id.includes('node_modules/vue')||id.includes('vue-router')||id.includes('pinia')) return 'vue-vendor';
      }
    }
  }
}
```

- `three` and `vue-vendor` must stay separate chunks — `three` is large and only used if 3D features added.
- `chunkSizeWarningLimit:3200` — CI warns if any chunk exceeds; `scripts/compress-build.mjs` gzips `internal/api/web/**/*.{js,css,html}`.

---

## Lint & Format

- No `eslint` in `web/` currently; rely on `vue-tsc` + `vite` warnings.
- When adding lint, keep `web/.eslintrc` minimal and don't conflict with Go `golangci`.

---

## Testing

- No `vitest` yet; when adding, place `web/src/__tests__/` and run `npm test`.
- Manual QA: `npm run dev` → login → browse `/` → admin tabs → file preview (pdf/docx/video) → check console for `TypeError`.

---

## Common Mistakes

- Editing `internal/api/web/*` directly — it is build artifact; edit `web/src/*` then `npm run build`.
- Forgetting `npm run type-check` after `api/types.ts` change — `vue-tsc` catches mismatched `Account` fields.
- Adding large dep (e.g. `three`) without updating `manualChunks` — bloats `vue-vendor`.
- Using `process.env` instead of `import.meta.env` in Vite code.

---

## Pre-Commit Checklist for Frontend

```bash
cd web
npm run type-check   # must be silent
npm run build        # must output ../internal/api/web and compress log
git diff --stat      # ensure only web/src changed, not web/node_modules
git status           # internal/api/web is expected to change after build
```

Trellis `trellis-check` will also require `make lint` (Go) + `npm run type-check` — run both before marking task complete.

---

## Performance

- Media viewers lazy-loaded via `() => import("@/components/media/PdfViewer.vue")` — keep main chunk < 1MB.
- Preview decoders (`pdfjs-dist`, `hls.js`) are async; don't `import` at top-level of `App.vue`.
