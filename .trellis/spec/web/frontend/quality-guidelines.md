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

### Browser acceptance: the only coverage for rendered behavior

The frontend has **no automated tests at all** (no `vitest`/`jest`, no `test` script). `vue-tsc` only checks
types and `vite build` only proves the bundle builds — **neither observes anything that exists only after
render**. For rendered behavior there are exactly two options: look at it in a browser, or verify nothing.
This section defines when the former is required.

#### When browser acceptance is required

The change touches `web/src/**` **and** its effect is only visible after rendering:

| Trigger | Why `type-check` cannot cover it |
|---|---|
| Async / runtime-fetched values in the UI (e.g. the version badge filled by a store after load) | Types are correct while the value may never arrive, or render empty |
| Conditional rendering and state machines (task terminal buckets, paused badge, cooldown countdown) | Whether a branch is actually reached is invisible to the type system |
| store ↔ component wiring, shared state across components, `watch`/reactivity | "Reactivity did not fire" and "value is empty" are indistinguishable by type |
| Router guard branches (`public_index_enabled`, `must_change_password`) | The outcome depends on runtime config |
| Release / deployment wrap-up | Must confirm the surface the user actually sees |

**Not required** for: backend/driver/store changes (`go test` + `curl` already cover them), docs- or
config-only changes, copy or CSS-class-only edits, and type errors that `type-check` already reports.

#### How to run it

```bash
bw open http://127.0.0.1:5211/ --text=1500   # real render (runs JS), returns post-render text
bw els                                       # list interactive elements to pick a click target
bw fill admin --into='input[placeholder="请输入用户名"]'
bw click '.submit-btn'                       # buttons: use a CSS selector
bw text 1200                                 # post-render text
bw shot /tmp/ui.png --full                   # screenshot; inspect it with read_image
```

`bw` is the resident headless Chromium (systemd `browser-cdp.service`, CDP 9222) usable from any session —
see the BROWSER-CDP section of the global `AGENTS.md`.

**Two pitfalls measured in this repo (2026-09-12):**

- **`bw click <text>` can hit the wrong element.** Clicking `登录` matched the page heading `管理员登录`
  instead of the `<button>`, so the form never submitted. Use a CSS selector for buttons; keep text matching
  for links whose text is unique.
- **Vue form inputs.** `bw fill` does write the value; if a form submits empty, dispatch `input`/`change`
  events via `bw eval` before clicking. *Not independently isolated:* in the observed run the click failure
  came first, so treat this as a troubleshooting order, not a proven requirement.

#### Boundaries (do not confuse these)

- **It is acceptance, not testing.** It neither replaces nor reduces `npm run type-check`, `npm run build`,
  `make lint`, or `go test`.
- **Never put it in CI, never use it as an automated gate decision.** Selectors break when the UI changes and
  it needs a running instance — it is non-deterministic tooling.
- **State the evidence, not a verdict.** Record "screenshot inspected" / "post-render DOM text was X" —
  never "automated assertion passed".
- Record the result in the task's `prd.md` check log. Write screenshots to `/tmp` and delete them afterwards
  so the git worktree stays clean.

#### Full manual QA pass (when the change touches a main flow)

`npm run dev` (or a deployed instance) → login → browse `/` → admin tabs → file preview (pdf/docx/video)
→ **check console for `TypeError`**

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
