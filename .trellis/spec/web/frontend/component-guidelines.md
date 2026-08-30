# Component Guidelines

> SFC conventions for `web/src/{components,views,composables}`.

---

## SFC Structure

```vue
<script setup lang="ts">
import { ref, computed, onMounted } from "vue";
import { useFooStore } from "@/stores/foo";
import FooCard from "@/components/FooCard.vue";

const props = defineProps<{ accountId: number; path: string }>();
const emit = defineEmits<{ (e:"select", id:string):void }>();
const store = useFooStore();
const loading = ref(false);
onMounted(()=> store.load(props.accountId));
const visible = computed(()=> store.items.filter(i=> !i.hidden));
</script>

<template>
  <div class="foo-browser">
    <FooCard v-for="item in visible" :key="item.id" :item="item" @select="emit('select', $event)" />
  </div>
</template>

<style scoped>
.foo-browser { display:grid; gap:12px; }
</style>
```

- `<script setup lang="ts">` required; no Options API.
- `defineProps` typed; `defineEmits` typed.
- Styles: `scoped` by default; global in `src/styles/`.

---

## Props / Emits

- Props are **readonly data down**, emits are **events up**. No `v-model` mutation of props.
- Use `computed` for derived lists (e.g. `visibleDrivers` in `stores/accounts.ts`).
- Event names: `select`, `open`, `close`, `confirm` — kebab in template, camel in TS.

---

## Component Types

| Location | Role |
|----------|------|
| `src/components/FileBrowser.vue` | Main grid/list, pagination, selection |
| `src/components/Breadcrumb.vue` | `Crumb[]` path nav |
| `src/components/AccountSelector.vue` | Dropdown/floating switch (`compact_home_enabled`) |
| `src/components/media/*` | `PdfViewer.vue` (`pdfjs-dist`), `DocxPreview.vue` (`docx-preview`), `VideoPlayer.vue` (`hls.js`/`mpegts.js`), `ImagePreview.vue` (`heic-to`, `@panzoom/panzoom`) |
| `src/views/IndexView.vue` | File browsing page `/` |
| `src/views/LoginView.vue` | Login + password change |
| `src/views/AdminView.vue` | Admin tab shell (strm/cache/fuse/automation/apiKeys/logs...) — **11k lines, split via composables, not by file** |

---

## Media Preview Stack

- **PDF**: `pdfjs-dist` + `PdfViewer.vue` (canvas).
- **Office**: `docx-preview`, `@aiden0z/pptx-renderer`, `xlsx` (SheetJS), `rtf.js`, `chardet` for encoding.
- **Image**: `heic-to` for HEIC, `@panzoom/panzoom` for zoom.
- **Video**: `hls.js` (m3u8), `mpegts.js` (flv), `media-chrome` + `media-captions` for controls/captions, `libbitsub` for subtitles.
- **Archive**: `@zip.js/zip.js` for zip preview.

All loaded via `import()` async in viewer components to keep main chunk small (see `vite.config.ts: manualChunks`).

---

## Composables

```ts
// src/composables/useToast.ts
export function useToast(){ function toast(msg:string, type:"success"|"error"){ /* append to #toast */ } return {toast} }

// src/composables/useDeveloperUnlock.ts
export function useDeveloperUnlock(){ const unlocked=ref(false); async function init(){ /* fetch /auth/status, check dev flag */ } return {unlocked, init} }
```

- `useX` must be call-safe multiple times (idempotent).
- State inside composable should be `ref` returned, not global store unless shared.

---

## Styling

- Global: `src/styles/*.css` (variables, reset).
- Component: `<style scoped>`; use CSS vars (`--color-primary`).
- Admin mode: `body.admin-mode` class toggled by `useAuthStore.syncAdminBodyClass` — admin-only styles scoped behind it.
- `AdminView.vue` uses `gsap 3.15.0` for transitions.

---

## Anti-Patterns

- Fetching in `components/` — move to `stores/` or `api/`.
- Mutating `props` directly — emit event, let parent update store.
- Large `AdminView.vue` edits without checking tab isolation — each tab (STRM, Cache, Log...) has its own state; search `AdminView.vue` for `ref.*Tab` before adding.
- Hardcoding `isCustomElement` beyond `media-` (`vite.config.ts` already handles `media-*`).

---

## Adding New Component

1. Create `src/components/FooBar.vue` with `<script setup>` + `defineProps<{foo:FooItem}>()`.
2. Export not needed; import where used: `import FooBar from "@/components/FooBar.vue"`.
3. If it needs data, `const store=useFooStore(); store.load()` in `onMounted`, not in `<script>` top-level `await`.
4. Add story/manual test via `npm run dev` → navigate to `/` or `/admin` tab.

---

## Testing

- `npm run type-check` validates props/emits types.
- Visual: `npm run dev` + browser; no unit test harness yet (add `vitest` when introducing component tests).

