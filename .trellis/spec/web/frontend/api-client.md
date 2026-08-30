# API Client

> Typed `fetch` wrapper in `web/src/api/client.ts` + per-domain modules in `web/src/api/*.ts`.

---

## Base Client (`src/api/client.ts`)

```ts
// Simplified shape
export const http = {
  get: <T>(url:string, params?:Record<string,any>)=>$T,
  post: <T>(url:string, body?:any)=>$T,
  put: <T>(url:string, body?:any)=>$T,
  del: <T>(url:string)=>$T,
  form: <T>(url:string, body:URLSearchParams)=>$T,
};
// Internally: fetch("/api"+url, {headers:{"Content-Type":"application/json"}, credentials:"include"})
// On 401 → redirect to /login (handled by caller store, not client directly)
// On !ok → throw with getApiErrorMessage(err) parsed from {code,message}
export function getApiErrorMessage(e: any): string
```

- Base URL: `/api` (dev proxy `vite.config.ts: proxy["/api"].target=http://127.0.0.1:5211`).
- Credentials: `include` (cookie session).
- Error shape: backend returns `{code, message}` via `writeDomainError`; client throws `ApiError{status, code, message}`.

Reference: `web/src/api/client.ts`.

---

## Per-Domain Modules

Each file exports an object with typed methods, never raw `fetch`:

```ts
// web/src/api/accounts.ts
export const accountsApi = {
  list: ()=> http.get<Account[]>("/admin/accounts"),
  get: (id:number)=> http.get<Account>(`/admin/accounts/${id}`),
  create: (p:AccountPayload)=> http.post<Account>("/admin/accounts", p),
  update: (id,p)=> http.put<Account>(`/admin/accounts/${id}`, p),
  remove: (id)=> http.del(`/admin/accounts/${id}`),
  toggle: (id)=> http.post<Account>(`/admin/accounts/${id}/toggle`),
  setDefault: (id)=> http.post<Account>(`/admin/accounts/${id}/set-default`),
};
// web/src/api/public.ts
export const publicApi = { listAccounts:()=> http.get<Account[]>("/public/accounts") };
// web/src/api/files.ts, strm.ts, auth.ts, etc. follow same pattern
```

Full list: `accounts.ts, announcement.ts, apikey.ts, auth.ts, automation.ts, drivers.ts, files.ts, public.ts, strm.ts, cache.ts, logs.ts, ...` (11+ modules).

---

## Types (`src/api/types.ts`)

Central DTOs shared between client and store, mirrors `internal/domain`:

```ts
export interface Account { id:number; name:string; driver_type:string; config:Record<string,any>; is_active:boolean; is_default:boolean; sort_order:number; }
export interface FileItem { id:string; name:string; size:number; is_dir:boolean; mime_type:string; thumb_url?:string; }
export interface DriverInfo { name:string; display_name:string; card_tags:string[]; auth_label:string; internal_experimental:boolean; }
export interface StrmTask { id:number; account_id:number; path:string; status:string; file_count:number; ... }
```

- Change requires **triple update**: `internal/domain/*.go` struct → `internal/store` scan → `internal/api` handler DTO → `web/src/api/types.ts` + store handling.

---

## Usage in Stores/Components

```ts
// stores/accounts.ts
import { accountsApi } from "@/api/accounts";
import { driversApi } from "@/api/drivers";
const accounts = await accountsApi.list(); // typed Account[]

// components/FileBrowser.vue
import { filesApi } from "@/api/files";
const res = await filesApi.list(currentAccountId.value, dirId);
```

- **Never** `fetch("/api/...")` in `components/` or `views/` — always via `api/*.ts`.
- **Never** `getApiErrorMessage` inline — stores use `getApiErrorMessage(e)` + `toast`.

Example: `web/src/stores/browser.ts: loadFiles()` catches `getApiErrorMessage(err)` and sets `error.value`.

---

## Auth & WebDAV

- Login: `auth.ts: login({username,password,remember}) => http.form<LoginResult>("/auth/login", body)` (`URLSearchParams`).
- WebDAV: `share/dav` uses same client but with `PROPFIND` via `http` override; `web/src/api/webdav.ts` if present.

---

## Adding New Endpoint

1. **Backend**: add handler in `internal/api/foo.go` + register in `router.go` (`/admin/foo`).
2. **Type**: add `FooItem`/`FooPayload` to `web/src/api/types.ts`.
3. **Client**: create/update `web/src/api/foo.ts`:

```ts
export const fooApi = {
  list: ()=> http.get<FooItem[]>("/admin/foo"),
  create: (p:FooPayload)=> http.post<FooItem>("/admin/foo", p),
};
```

4. **Store**: add `useFooStore` if shared state, or call `fooApi` directly in view.
5. **Verify**: `npm run type-check` must pass; `vite build` must not warn.

---

## Anti-Patterns

- `any` payload — always type `FooPayload`.
- Hardcoding `/api` prefix — client already prefixes; pass `"/admin/foo"` only.
- Ignoring error `code` — use `getApiErrorMessage` to show `message`, not `status`.
- Duplicating DTOs per file — put shared `FileItem/Account` in `types.ts`.

---

## Testing

- `npm run type-check` validates `api/*.ts` vs `types.ts`.
- Manual `npm run dev` with backend running; use browser devtools Network to verify `Authorization` header / cookie.

