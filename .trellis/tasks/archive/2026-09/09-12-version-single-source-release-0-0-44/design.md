# Design: 版本号单一来源（方案 C）+ 发布 v0.0.44

## Overview

把"版本号"从**两处独立硬编码**（Go 常量 + TS 字面量）收敛为**一处真值**：后端 `internal/buildinfo.Version`。前端不再在构建期固化版本，而是在运行期向公共端点取。

链路只增加一段：`buildinfo.Version` → `api.Deps.Version` → `Handler.version` → `GET /api/public/system-config` 的 `version` 字段 → 前端 `appInfo` store → `AppFooter` / `AdminAccountChip`。

选方案 C 而非 A（构建期注入）的直接后果要写明：**版本真值仍是手工维护的 Go 常量**，发版时改一处；但它同时消灭了"前端字面量"与"后端字面量"两处漂移源中的一处，并把界面版本从"构建时快照"变成"运行期事实"—— 只要后端是新的，界面就必然显示后端真实版本，不可能再对不上。

## Boundaries

| 层 | 改 | 不改 |
|---|---|---|
| **buildinfo** | `internal/buildinfo/version.go` 值 → `v0.0.44` | 该文件的 `-ldflags` 覆盖机制与包结构 |
| **api（Deps/Handler）** | `router.go` 的 `Deps` + `Handler` + `NewRouter` 各加一处 `version` | 既有任何依赖字段与路由注册 |
| **api（handler）** | `public.go` 的 `publicSystemConfig` 响应**新增一个字段** | 既有 3 个字段的语义/类型/形状 |
| **app（装配）** | `wire_http.go` 构造 `api.Deps` 时传 `Version` | 该文件其余装配顺序与 `backuprestore` 的既有 `Version` 传递 |
| **前端 api** | `public.ts` 的 `PublicSystemConfig` 加 `version: string` | 端点路径、`publicApi` 既有方法 |
| **前端 store** | 新增 `stores/appInfo.ts` | `auth.ts`/`accounts.ts`/`browser.ts` |
| **前端 组件** | `AppFooter.vue`、`AdminAccountChip.vue` 改经 store 取值 | 组件的其余结构与样式 |
| **前端常量** | `version.ts` 删 2 个导出版本字面量、改 `GITHUB_URL` | `APP_NAME`/`APP_URL`/`COLLAB_*` |
| **embed 产物** | `internal/api/web/**` 由 `npm run build` 重建 | 构建脚本与 `vite.config.ts` |
| **发布物** | `README.md`×2、`docker-compose.yml`×1 的镜像 tag | 正文其余内容 |
| **业务** | 无 | `internal/domain`、驱动、上传、缓存、鉴权等一切业务路径 |

## Data Flow

```
internal/buildinfo.Version  = "v0.0.44"        ← 唯一真值（发版时手工改这一处）
        │
        ├─► internal/app/wire_http.go    api.Deps{ Version: buildinfo.Version }
        │
        ├─► internal/api/router.go       Deps.Version ──► Handler.version
        │
        └─► internal/api/public.go       GET /api/public/system-config
                                          { version, index_account_switch_mode,
                                            compact_home_enabled, header_effects_enabled }
                                                   │  （公共端点，免鉴权，与既有 3 字段同处）
                                                   ▼
             web/src/api/public.ts   PublicSystemConfig{ version: string }
                                                   │
                                                   ▼
             web/src/stores/appInfo.ts   version: Ref<string>（初值 ""）+ load()（inflight 去重）
                                                   │
                        ┌──────────────────────────┴──────────────────────────┐
                        ▼                                                     ▼
             components/layout/AppFooter.vue              components/admin/AdminAccountChip.vue
             「当前版本」徽标 = APP_NAME + version             「关于 LitePan」= version
```

**降级语义**：`version === ""` 时只渲染 `APP_NAME`（不渲染任何版本字符串）。这样"版本还没到"与"版本是空的"表现一致，且**不引入 fallback 字面量** —— 否则等于把硬编码从常量挪到 fallback，问题没解决。

## Key Decisions

### KD1 为什么用 `Deps.Version` 注入而不是在 `public.go` 里直接 import `buildinfo`

- `wire_http.go` 已有同模式的先例（`backuprestore.Options{Version: buildinfo.Version}`），照抄即可保持装配层集中的既有风格。
- handler 依赖显式传入 → 单测可注入任意版本串做断言（`Handler{version: "vX"}`），不需要为了测试去改全局变量。
- 符合 `spec/backend/backend/api-layering.md` 的 "`Deps` injection" 约定。

### KD2 为什么新开 store 而不是复用 `auth.ts`

`publicSystemConfig` 是**公共**端点，首页匿名访问也要显示版本；`auth` store 持有的是会话态（`sessionAdmin`/`mustChangePassword`），把版本塞进去会让"未登录也能显示版本"依赖 auth 的加载时序。新 store 职责单一，且能被"首页"与"管理后台"两处共享。

### KD3 并发去重是必需的，不是可选优化

`AppFooter`（首页）与 `AdminAccountChip`（管理后台顶部）可能同页挂载（管理员在首页）。两个组件各自 `onMounted → load()` 会打两次同样的请求。按 `state-and-routing.md` 的既有约定（`stores/accounts.ts` 的 `inflightLoad` 注释：多组件同时挂载共享一次请求）实现去重。

### KD4 embed 产物必须重建

`internal/api/web/**` 是**受跟踪**的 `go:embed` 目标（135 个文件）。前端源码改了却不重建，则任何非 Docker 的 `go build` 仍会嵌入旧前端、继续显示 `v0.5.2-Beta`。所以 `npm run build` + 提交是**功能的一部分**，不是"顺手重新生成"。

### KD5 端点向后兼容

只在既有响应上加字段，不改名、不删字段、不改类型。旧前端（不认识 `version`）照常工作；新前端配旧后端时 `version` 为 `undefined` → store 归一为空串 → 降级只显示应用名。**不做版本探测或特性开关**——这是单仓库单体，前后端永远同镜像发布。

## Compatibility

- **后端**：`/api/public/system-config` 纯字段新增，无破坏性变更；`Deps` 加字段不影响既有调用点（`wire_http.go` 是唯一装配处）。
- **前端**：`APP_VERSION_BADGE` 删除前已确认**只有** `AppFooter.vue` 引用；`APP_VERSION` 只有 `AdminAccountChip.vue` 引用（两者都在本任务改动范围内，无遗漏引用）。
- **`GITHUB_URL` 改动影响面**：仅 `AppFooter.vue` 的"项目地址"徽标链接。
- **Docker 构建**：`Dockerfile` 内 `npm run build` 会重新生成前端并 `COPY --from=web` 覆盖，故镜像前端与提交的 embed 产物可能因 Node 版本差异（镜像内 node:20 vs 本机 node 22）而非字节相同 —— 这是既有取舍，本任务不改变。
- **发布顺序**：必须 `push main` → `tag` → `push tag` → `release`。0.0.39 曾因先 `release` 导致 GitHub 自动 tag 指向旧 main 头，需删远端 tag 重推并 API 复核。

## Tradeoffs

- **方案 C vs A**：C 保留一个手工维护点（Go 常量），换来"界面版本 = 后端真实版本"的强一致与更小的构建改动面；A 能做到 tag 驱动零手工，但要动 `Dockerfile`(ARG/ldflags) + `Makefile` + Vite define 三处构建链，且用户已明确不选。**代价**：发版仍要记得改 `version.go` —— 缓解手段是本次发布的验收项本身就把"代码版本 = 镜像 tag = git tag"三者一起核对。
- **重建 embed 的巨大 diff**：一次性 135 文件变更，换来单机构建的正确性。不重建的替代方案（让 deploy 一律走 Docker）会把约束藏进流程而非仓库事实，更脆。
- **临时容器验证 vs 完整部署**：用 5212 + 临时数据目录验证镜像真实行为，既能证明"发布的镜像确实报 v0.0.44"，又不占用 5211、不在仓库生成 `data/`，尊重用户"部署先搁置"的指示。

## Rollout / Rollback

**Rollout 顺序**（每一步都可停下来检查）：

1. 代码改动 → 质量门全绿 → 重建 embed → 提交并 `push main`
2. 构建镜像 → 本地 3 tag → 推 GHCR 3 tag
3. `git tag v0.0.44` → `push tag` → **最后** `gh release create`
4. 临时容器三连验证 → 删容器

**Rollback**：

| 阶段 | 回滚方式 |
|---|---|
| 代码/embed | `git revert <commit>`（或 `git checkout <prev> -- <files>`）并重跑质量门 |
| GHCR 镜像 | 删该版本：`gh api -X DELETE /user/packages/container/litepan/versions/<id>`（token 有 `delete:packages`）；再把 `latest` 指回 `v0.0.43` 的 digest |
| git tag | `git push origin :refs/tags/v0.0.44` + `git tag -d v0.0.44` |
| GitHub release | `gh release delete v0.0.44 --yes` |
| 临时容器 | `docker rm -f litepan-verify`；`rm -rf /tmp/litepan-verify` |

**不可回滚项**：一旦 3 个 tag 推上公共 registry，他人可能已拉取 —— 故在第 4 步验证通过前不发布 release 说明；若镜像有致命问题，按上表删版本并重推。

## File Map

| 路径 | 动作 | 归属 |
|---|---|---|
| `internal/buildinfo/version.go` | 改 1 行（`v0.5.2-Beta` → `v0.0.44`） | D4 |
| `internal/api/router.go` | `Deps` + `Handler` + `NewRouter` 各加 `version` | D1 |
| `internal/api/public.go` | 响应加 `"version"` | D1 |
| `internal/app/wire_http.go` | 传 `Version: buildinfo.Version` | D1 |
| `internal/api/public_version_test.go` | **新增**：断言注入的版本出现在响应中 | D1 |
| `web/src/api/public.ts` | DTO 加 `version: string` | D2 |
| `web/src/stores/appInfo.ts` | **新增** store（inflight 去重） | D2 |
| `web/src/components/layout/AppFooter.vue` | 改用 store，删除 `APP_VERSION_BADGE` 引用 | D2 |
| `web/src/components/admin/AdminAccountChip.vue` | 改用 store | D2 |
| `web/src/version.ts` | 删 `APP_VERSION`/`APP_VERSION_BADGE`；改 `GITHUB_URL` | D2/D3 |
| `internal/api/web/**` | 重建（135 受跟踪文件） | D5 |
| `README.md`、`docker-compose.yml` | 镜像 tag → `v0.0.44` | D4 |
| `internal/domain`、驱动、上传、缓存、鉴权 | **零改动** | — |
