# 统一版本号单一来源（后端运行期提供）并发布 v0.0.44

## Goal

版本号自上游继承后**从未更新**：`internal/buildinfo/version.go` 与 `web/src/version.ts` 双双停留在 `v0.5.2-Beta`，而 `Dockerfile` 的构建命令也没用 `-ldflags` 注入。结果是界面上「当前版本」「关于 LitePan」长期显示上游版本号，与实际发布 tag（`v0.0.43`）不符，且两处独立字面量必然再次漂移。

按用户选定的**方案 C**：后端成为唯一来源（公共端点运行期暴露 `buildinfo.Version`），前端改为运行期读取并**彻底移除版本字面量**；顺带修正 `version.ts` 中指向错误仓库的 `GITHUB_URL`；最后完整发布 `v0.0.44`。

## Background

只读侦察（2026-09-12）确认的全部现状：

| 事实 | 证据 |
|---|---|
| 后端版本常量是上游值 | `internal/buildinfo/version.go:4` → `var Version = "v0.5.2-Beta"`（注释已声明可用 `-ldflags` 覆盖，但从未覆盖） |
| 后端该值**仅**用于备份元数据 | 唯一消费点 `internal/app/wire_http.go:38` → `backuprestore.Options{Version: ...}` |
| 前端第二处硬编码 | `web/src/version.ts:3` → `APP_VERSION = "v0.5.2-Beta"`；注释自称"唯一来源"，实为与后端重复 |
| 界面可见位置 | `AppFooter.vue`（`APP_VERSION_BADGE` → "当前版本"徽标）、`AdminAccountChip.vue:196-197`（「关于 LitePan」→ `APP_VERSION`） |
| 后端**未**暴露版本 | `api.Deps` 无 `Version` 字段；`publicSystemConfig` 只返回 3 个 UI 开关（`internal/api/public.go:34-38`） |
| Dockerfile 未注入 | `Dockerfile:31` → `go build ... -ldflags="-s -w"`（无 `-X litepan/internal/buildinfo.Version=`） |
| 无 CI | 仓库无 `.github/`；镜像靠手工构建推送 |
| `v0.0.43` 出现处仅 3 个文件 | `docker-compose.yml:3`、`README.md:11`、`README.md:34` |
| 发布前提就绪 | `gh` 已认证为 `zhemed` 且 token 含 `write:packages`/`delete:packages`；`docker buildx v0.37.1` 可用；daemon active |
| 前端已有现成端点与 DTO | `web/src/api/public.ts:16` → `systemConfig: () => http.get<PublicSystemConfig>("/public/system-config")`（**公共、免鉴权**） |
| GITHUB_URL 指错仓库 | `web/src/version.ts:5` → `https://github.com/Ponphil/LitePan`（上游）；README 徽标指向 `zhemed/LitePan` |

**既有发版仪式**（考据自 `journal-3.md` 三次发版复盘）：

1. 质量门 vet / test / build / web 三连全绿
2. 构建镜像 → 推 **3 个 tag 同 digest**：`v0.0.4X` + `0.0.4X` + `latest`（GHCR 历史 tag 已确认此模式）
3. **顺序硬要求**：`push main` → `git tag` → `push tag` → **最后** `gh release create`
   （0.0.39 的教训：`gh release create` 先执行会让 GitHub 自动创建的 tag 指向旧 main 头，需删远端 tag 重推并在 API 侧复核）
4. 本地容器重建三连验证（health / 表单登录 / 任务汇总）

## Requirements

### D1 后端成为版本唯一来源

- `internal/api/router.go`：`Deps` 增加 `Version string` 字段；`Handler` 增加 `version string`；`NewRouter` 中 `version: d.Version`
- `internal/app/wire_http.go`：构造 `api.Deps` 时传 `Version: buildinfo.Version`（与该文件既有的 `backuprestore.Options{Version: buildinfo.Version}` 同一模式，不新增隐式依赖）
- `internal/api/public.go`：`publicSystemConfig` 响应增加 `"version": h.version`
- **不改变**该端点既有 3 个字段的语义与形状（纯新增字段，向后兼容）

### D1b 第三处硬编码：默认 User-Agent（实施期由验收项 1 抓出）

侦察时遗漏、由本任务验收项「`internal/` 内 `v0.5.2` 零命中」在执行中抓出的第三处：

- `internal/httpx/user_agent.go:8` 持有 `AppVersion = "v0.5.2-Beta"`，并派生 `DefaultUserAgent = AppName + "/" + AppVersion`；其注释原文写着"保持与前端 web/src/version.ts 的品牌版本一致" —— 恰恰是"靠人工保持一致"这件事断了
- 影响面：`internal/api/oauth.go:56`、`internal/httpx/oauth.go:27`、`drivers/template/{auth,transport}.go`、`drivers/115_Open/upload.go` —— 即**发给上游服务的 User-Agent 也一直报着错误版本**
- 修法：`AppVersion` / `DefaultUserAgent` 改为由 `buildinfo.Version` **运行期派生**，不再持有字面量
- **连带必改**：`buildinfo.Version` 是 `var`（`-ldflags -X` 只能覆盖 var，不能是 const），故 `DefaultUserAgent` 不能再是 const；而 `drivers/115_Open/upload.go:587` 原本是 `const ossUserAgent = httpx.DefaultUserAgent`，必须同步改为 `var ossUserAgent = ...`（该符号仅用于 3 处 `req.Header.Set`，运行期取值，语义无变化）

### D2 前端运行期读取，彻底移除版本字面量

- `web/src/api/public.ts`：`PublicSystemConfig` 接口增加 `version: string`
- 新增 `web/src/stores/appInfo.ts`：setup 风格 Pinia store，暴露 `version`（初值空串）与 `load()`；`load()` 必须做**并发去重**（`inflight` Promise 模式，与 `stores/accounts.ts` 一致）且已加载则短路
- `web/src/components/layout/AppFooter.vue`：改为经 store 读取版本，徽标值由 `APP_NAME` + store 版本组合；`onMounted` 触发 `load()`
- `web/src/components/admin/AdminAccountChip.vue`：`{{ APP_VERSION }}` 改为 store 值
- `web/src/version.ts`：**删除** `APP_VERSION` 与 `APP_VERSION_BADGE` 两个导出版本字面量；保留 `APP_NAME`/`APP_URL`/`COLLAB_*`
- **版本未加载时的降级**：只显示 `APP_NAME`，**不得**回退到任何硬编码版本字符串（否则等于把字面量藏进 fallback）
- 遵守 `spec/web/frontend/api-client.md` 与 `state-and-routing.md`：组件经 store 取共享状态，不直接 `fetch`，不新增重复 DTO

### D3 修正 GITHUB_URL

- `web/src/version.ts` 的 `GITHUB_URL`：`https://github.com/Ponphil/LitePan` → `https://github.com/zhemed/LitePan`（与 README 徽标一致）

### D4 版本号升到 v0.0.44

- `internal/buildinfo/version.go`：`v0.5.2-Beta` → `v0.0.44`（此任务即新版本）
- `README.md` 2 处、`docker-compose.yml` 1 处：镜像 tag `v0.0.43` → `v0.0.44`

### D5 重建并提交 embed 产物

- `cd web && npm run build`，提交 `internal/api/web/**` 的变更（受跟踪的 `go:embed` 目标）
- 单机 `go build` 的产物必须与 `web/src` 一致，否则非 Docker 部署会继续显示旧版本

### D6 发布 v0.0.44

1. 构建：`docker build --platform linux/amd64 -t ghcr.io/zhemed/litepan:v0.0.44 .`
2. 本地补 tag：`0.0.44`、`latest`（与 `v0.0.44` 同一 digest）
3. 登录：`gh auth token | docker login ghcr.io -u zhemed --password-stdin`
4. 推送 3 个 tag
5. **严格顺序**：`git push origin main` → `git tag v0.0.44` → `git push origin v0.0.44` → `gh release create v0.0.44`
6. 远端复核：tag 指向的 commit 与本地一致

### D7 发版验证（不构成部署）

- 用**临时容器**验证镜像：端口 **5212**（不占 5211）、数据目录用 `/tmp/litepan-verify`（不在仓库创建 `data/`）
- 验证三连：`GET /api/health`、`GET /api/public/system-config` 返回 `version=v0.0.44`、`POST /api/auth/login` **表单编码** `admin/123456` 成功
- 验证后 `docker rm -f` 清理；**5211 保持空闲**（部署仍按用户指示搁置）

## Constraints

- **不改业务逻辑**：不动 `internal/domain`、上传、缓存、鉴权等任何业务路径；本任务只加一个响应字段 + 前端展示链路
- **驱动的唯一例外（D1b 连带）**：`drivers/115_Open/upload.go` 仅把 `const ossUserAgent` 改为 `var ossUserAgent`（因常量不能再由运行期值派生），**不改其值、不改使用方式、不改任何驱动行为**
- **版本唯一来源**：前端不得残留任何版本字面量（含 fallback）；后端只有 `buildinfo.Version` 一处真值
- **不引入构建期注入**：`Dockerfile`/`Makefile` 的 `-ldflags`/`ARG` 接线属方案 A，用户未选，**本任务不做**
- **不占用 5211、不动 DSH（3080/3081）**；仓库内不得生成 `data/`/`mounts/`（用临时目录）
- **发版顺序不得颠倒**（0.0.39 教训）；三 tag 必须同 digest
- **登录必须用表单编码**：`curl -d 'username=admin&password=123456'`，发 JSON 会被静默解析为空用户名（AGENTS.md 明载）
- 不修改归档任务与历史 journal
- 前端改动须过 `npm run type-check`；Go 改动须过 `make lint` + `go vet` + `go test`

## Acceptance Criteria

- [ ] `grep -rn "v0\.5\.2\|v0\.5\.2-Beta"`（排除 `node_modules`/归档/workspace）零命中，即上游版本号已从后端常量与前端字面量中彻底消失
- [ ] `grep -rn "APP_VERSION" web/src` 零命中（硬编码与 `APP_VERSION_BADGE` 均已删除）
- [ ] `GET /api/public/system-config` 返回体含 `"version":"v0.0.44"`，且原有 3 个字段（`index_account_switch_mode`/`compact_home_enabled`/`header_effects_enabled`）均未丢失
- [ ] 新增 Go 单测覆盖版本字段（注入值可被断言），`GOWORK=off go test ./internal/api/` 通过
- [ ] `web/src/stores/appInfo.ts` 存在，`load()` 具备 inflight 去重与已加载短路；`AppFooter.vue`、`AdminAccountChip.vue` 均经 store 取值
- [ ] `web/src/version.ts` 的 `GITHUB_URL` 为 `https://github.com/zhemed/LitePan`
- [ ] `internal/buildinfo/version.go` 为 `v0.0.44`；`README.md`（2 处）与 `docker-compose.yml`（1 处）镜像 tag 均为 `v0.0.44`
- [ ] `cd web && npm run build` 后 `internal/api/web/**` 已更新并提交（`git status` 无遗留未提交的 embed 变更）
- [ ] 质量门全绿：`make lint` 0 issues、`GOWORK=off go vet ./...`、`GOWORK=off go test ./...`、`cd web && npm run type-check`
- [ ] GHCR 上 `ghcr.io/zhemed/litepan` 出现 tag 组 `v0.0.44` + `0.0.44` + `latest`，且三者指向**同一 digest**
- [ ] `git tag v0.0.44` 已推送到 origin，且 `gh api repos/zhemed/LitePan/git/ref/tags/v0.0.44` 指向的 commit 与本地一致
- [ ] `gh release view v0.0.44` 存在
- [ ] 临时验证容器（5212）三项全过：health 200、`public/system-config` 返回 `version=v0.0.44`、表单登录成功；验证后容器已删除
- [ ] `ss -ltnp` 确认 `5211` 仍空闲、`3080/3081` 仍由 DSH 监听；仓库内无新增 `data/`/`mounts/`
- [ ] `git status --short` 改动逐条可追溯到 D1–D5，无越界文件

## Notes

- Scope 标注 `cross-layer`（API DTO → api client → Pinia store → 组件，四层贯穿），故按复杂任务补齐 `design.md` + `implement.md` 后再 `start`。
- 方案 C 后，版本真值仍是**手工维护的 Go 常量**（发版时改一处）。若日后希望由 tag 自动注入，那是被搁置的方案 A（`Dockerfile ARG` + `-ldflags` + Vite define），本任务不预埋。
- **本任务不做**：LitePan 的正式部署（用户明确搁置，5211 保持空闲）；`internal/api/web` 与 Docker 构建产物的字节级一致性（Dockerfile 内会重新构建前端，属既有权衡）。
- 发版属不可逆对外动作，已获用户明确授权（"一并完整发版 v0.0.44"）。
