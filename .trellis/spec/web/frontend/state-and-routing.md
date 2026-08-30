# State & Routing

> Pinia stores + vue-router 5 guards as implemented in `web/src/{stores,router}`.

---

## Router

```ts
// web/src/router/index.ts
const routes: RouteRecordRaw[] = [
  { path:"/", name:"home", component:()=>import("@/views/IndexView.vue"), meta:{title:"文件浏览"} },
  { path:"/login", name:"login", component:()=>import("@/views/LoginView.vue"), meta:{title:"管理员登录", guestOnly:true} },
  { path:"/admin", name:"admin", component:()=>import("@/views/AdminView.vue"), meta:{title:"管理后台", requiresAuth:true} },
  { path:"/:pathMatch(.*)*", redirect:"/" },
];
router.beforeEach(async(to)=>{
  if(to.meta.title) document.title = `${to.meta.title} - LitePan`;
  const auth=useAuthStore();
  if(to.meta.requiresAuth){
    if(auth.loaded && auth.sessionAdmin) return true;
    const status=await fetchAuthStatus();
    if(!status.is_admin) return {path:"/login", query:{redirect:to.fullPath}};
    auth.applyStatus(status); return true;
  }
  if(to.meta.guestOnly){
    if(auth.loaded && auth.sessionAdmin) return "/admin";
    const status=await fetchAuthStatus();
    if(status.is_admin){ auth.applyStatus(status); return "/admin"; }
  }
  if(to.name==="home"){
    if(!auth.loaded) { const s=await fetchAuthStatus(); auth.applyStatus(s); if(!s.public_index_enabled && !s.is_admin) return "/login"; }
  }
  return true;
});
```

- `public_index_enabled` gates `/` for anonymous (see `SystemConfig` in `api/auth.ts`).
- `must_change_password` forces admin stays on `/admin` until password changed (`auth.ts: isAdmin computed`).

Reference: `web/src/router/index.ts:1-60`.

---

## Stores

### auth.ts
```ts
export const useAuthStore = defineStore("auth", ()=>{
  const sessionAdmin=ref(false), username=ref(""), mustChangePassword=ref(false), publicIndexEnabled=ref(false), loaded=ref(false);
  const isAdmin=computed(()=>sessionAdmin.value && !mustChangePassword.value);
  function applyStatus(data:AuthStatus){ sessionAdmin.value=data.is_admin; username.value=data.username??""; mustChangePassword.value=data.must_change_password; ... loaded.value=true; document.body.classList.toggle("admin-mode", isAdmin.value); }
  async function load(){ try{applyStatus(await fetchAuthStatus())}catch{clear()} }
  return {sessionAdmin, isAdmin, loaded, applyStatus, load, clear};
});
```
- Called by `router.beforeEach` + `App.vue` on mount.
- `syncAdminBodyClass` toggles `admin-mode` for CSS.

### accounts.ts

- `accounts`, `drivers`, `visibleDrivers` (filters `internal_experimental` unless `devUnlocked`), `loading`.
- `loadAccounts()` dedup via `inflightLoad` Promise — multiple components mounting simultaneously share one request.
- `loadDrivers()` caches `drivers.value.length` check.

### browser.ts

- `currentAccountId`, `breadcrumb: Crumb[]`, `files: FileItem[]`, `loading/refreshing/error/responseTime/cacheRate`.
- `cloneCrumbs`, `ROOT={id:"", name:"根目录"}`.
- Actions: `loadFiles({silent, forceRefresh})` → `filesApi.list(accountId, crumbs.at(-1).id)`; `enter(dir)` pushes crumb; `up()` pops.

Reference: `web/src/stores/accounts.ts`, `auth.ts`, `browser.ts`.

---

## API Status Object

```ts
// web/src/api/auth.ts
interface AuthStatus { is_admin:boolean; username?:string; public_index_enabled:boolean; must_change_password?:boolean; password_change_reason?:string; }
interface SystemConfig { admin_username:string; session_timeout:number; public_index_enabled:boolean; compact_home_enabled:boolean; admin_home_return_mode:"sidebar"|"top_icon"; ... }
```

Returned by `GET /auth/status` and `GET /admin/system-config`. Stores cache it; router reuses `auth.loaded`.

---

## Composables

- `useToast(): { toast({message,type}) }` — global toast.
- `useDeveloperUnlock(): { unlocked, init() }` — checks `localStorage` + `GET /auth/status` for experimental driver visibility.
- `useStrmTask`, `useMediaOrganize` — polling `GET /admin/strm/tasks` etc. with interval.

---

## Adding New Route/Store

1. **Route**: add entry in `router/index.ts` with `meta:{requiresAuth|guestOnly|title}`.
2. **Guard**: if new auth rule, extend `beforeEach` (e.g. feature flag).
3. **Store**: `defineStore("foo", ()=>{ const items=ref<Foo[]>([]); async function load(){...} return {items, load} })`.
4. **Dedup**: if store is fetched by multiple components, add `let inflight` pattern.
5. **Persist**: use `localStorage` only for UI prefs (`header_effects_enabled` etc.), not for auth (session cookie).

---

## Anti-Patterns

- Reading `fetchAuthStatus()` directly in component without store — use `useAuthStore().isAdmin`.
- Mutating `auth.sessionAdmin.value` directly — use `applyStatus`/`applyLogin`.
- Forgetting `guestOnly` for `/login` — admin would see login page again.
- Not handling `public_index_enabled=false` → anonymous should redirect to `/login`.

---

## Testing

- Router guards tested via `npm run type-check` (types) + manual `npm run dev` with auth mock.
- Store logic unit-testable via `vitest` if added (not present; use `pinia testing` pattern when introducing).
