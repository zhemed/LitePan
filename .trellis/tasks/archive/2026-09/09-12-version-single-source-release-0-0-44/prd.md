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

- [x] `grep -rn "v0\.5\.2\|v0\.5\.2-Beta"`（排除 `node_modules`/归档/workspace）零命中 —— 实测 `web/src`+`internal/`+`drivers/`+`cmd/`+`pkg/` 全零命中。**该验收项在实施期抓出了侦察时遗漏的第三处硬编码**（`internal/httpx/user_agent.go`），详见 D1b
- [x] `grep -rn "APP_VERSION" web/src` 零命中 —— 实测零命中；`version.ts` 已删除 `APP_VERSION` 与 `APP_VERSION_BADGE`
- [x] `GET /api/public/system-config` 返回体含 `"version":"v0.0.44"`，且原有 3 个字段均未丢失 —— 实测（拉回的发布镜像内）：`{"compact_home_enabled":false,"header_effects_enabled":true,"index_account_switch_mode":"dropdown","version":"v0.0.44"}`
- [x] 新增 Go 单测覆盖版本字段，`go test ./internal/api/` 通过 —— 新增 `internal/api/public_version_test.go`，两个子测试（回传注入值 / 既有字段不丢失）均 PASS
- [x] `web/src/stores/appInfo.ts` 存在，`load()` 具备 inflight 去重与已加载短路；两个组件均经 store 取值 —— 实测 store 已建；`AppFooter.vue`、`AdminAccountChip.vue` 均改为经 store 取值并各自触发 `load()`
- [x] `web/src/version.ts` 的 `GITHUB_URL` 为 `https://github.com/zhemed/LitePan` —— 实测 `version.ts:5` 已改
- [x] `internal/buildinfo/version.go` 为 `v0.0.44`；`README.md`（2 处）与 `docker-compose.yml`（1 处）均为 `v0.0.44` —— 实测；且 `grep v0\.0\.44` 在**全部代码**中只命中 `version.go` 一处，单一真值达成
- [x] `cd web && npm run build` 后 `internal/api/web/**` 已更新并提交 —— 实测 109 个产物变更（54 新/54 删/1 改）并已随提交推送；解压复核产物内 0 处含上游版本号
- [x] 质量门全绿 —— 实测 `make lint` **0 issues**；`go vet` exit=0；`go test` **28 包全 ok 无失败**；`vue-tsc -b` exit=0
- [x] GHCR 上出现 `v0.0.44` + `0.0.44` + `latest` 且同 digest —— 实测 GHCR API 返回**单条 version（id 1240621421）挂载全部三个 tag**，digest 均为 `sha256:a864057d46acfaa980f8064f7724b867342090827b91b99fb5161c50b035c903`
- [x] `git tag v0.0.44` 已推送且远端 commit 与本地一致 —— 实测远端 `593d1253aa5dc2e1e72fb12701cecfa92fd1bdc4` == 本地，未重演 0.0.39 的旧 main 头问题
- [x] `gh release view v0.0.44` 存在 —— 实测 `tag=v0.0.44  name=v0.0.44 — 版本号收敛为单一来源（后端），前端运行期读取  published=2026-09-12T12:11:13Z`
- [x] 临时验证容器（5212）三项全过；验证后容器已删除 —— 实测 health 200、version=`v0.0.44`、表单登录 200（新库默认 `admin/admin`）；**且删除本地镜像后从 GHCR 真拉回来重测一遍**，digest 与构建一致、三项再验全过；容器与临时目录均已清理
- [x] `ss -ltnp` 确认 `5211` 仍空闲、`3080/3081` 仍由 DSH 监听；仓库内无新增 `data/`/`mounts/` —— 实测 5211/5212 均空闲、DSH 2 个监听在位、`data`/`mounts` 不存在、`docker ps -a` 无 litepan 容器
- [x] `git status --short` 改动逐条可追溯到 D1–D5，无越界文件 —— 实测提交 `593d125` 含 13 个受跟踪文件改动 + 2 个新文件 + embed 更新；`internal/domain`/上传/缓存/鉴权零改动

## Notes

- Scope 标注 `cross-layer`（API DTO → api client → Pinia store → 组件，四层贯穿），故按复杂任务补齐 `design.md` + `implement.md` 后再 `start`。
- 方案 C 后，版本真值仍是**手工维护的 Go 常量**（发版时改一处）。若日后希望由 tag 自动注入，那是被搁置的方案 A（`Dockerfile ARG` + `-ldflags` + Vite define），本任务不预埋。
- **本任务不做**：LitePan 的正式部署（用户明确搁置，5211 保持空闲）；`internal/api/web` 与 Docker 构建产物的字节级一致性（Dockerfile 内会重新构建前端，属既有权衡）。
- 发版属不可逆对外动作，已获用户明确授权（"一并完整发版 v0.0.44"）。

## 检查记录（trellis-check，2026-09-12）

**Step 1 变更识别**：提交 `593d125` 含 13 个受跟踪文件改动 + 2 个新文件（`internal/api/public_version_test.go`、`web/src/stores/appInfo.ts`）+ 109 个 embed 产物变更（54 新/54 删/1 改）。业务路径零改动：`internal/domain`、上传、缓存、鉴权均未触碰。

**Step 2 规范对齐**：实施前已读 `spec/web/frontend/api-client.md`（按域模块 + 类型集中）、`state-and-routing.md`（Pinia setup 风格 + `inflightLoad` 去重）、`spec/backend/backend/quality-guidelines.md`；实现方式与三处约定逐条对齐（版本经 store 而非组件直连、DTO 加在既有 `public.ts` 而非新建重复类型、Deps 注入而非在 handler 内直接 import）。

**Step 3 项目质量门（全部实测）**

| 检查 | 命令 | 结果 |
|---|---|---|
| Lint | `make lint` | `0 issues.` exit=0 |
| Vet | `GOWORK=off go vet ./...` | exit=0 |
| Test | `GOWORK=off go test ./...` | 28 包 `ok`，**无失败** |
| 新测试 | `go test ./internal/api/ -run PublicSystemConfig -v` | 2 个子测试 PASS |
| 类型检查 | `cd web && npm run type-check` | `vue-tsc -b` exit=0 |
| 构建 | `cd web && npm run build` | 成功；解压复核 102 个产物内 0 处含上游版本号 |

**Step 4 清单核对**

- 代码质量：lint / vet / test / type-check 全绿；新增代码无 debug 输出、无 `//nolint`、无 `any`。
- **测试覆盖：本任务实际新增了测试**（与近期几个维护任务不同）—— `internal/api/public_version_test.go` 覆盖"注入什么就回传什么"与"既有字段不丢失"两个断言，前者正是本次改动的核心契约。
- **Spec 同步：已执行** —— 在 `spec/backend/backend/api-layering.md` 的 "Service Injection via Deps" 后新增「版本号：单一来源」小节，写明真值位置、注入路径、前端读取方式、发版只改一处，并附历史教训与回归 grep 命令。**理由**：不定这条约定，后人会再写死一份版本号，本次修的 bug 就会复发。
- 范围纪律：改动逐条对应 D1–D5b；未顺手格式化、未重命名、未重构无关代码。

**Step 5 跨层一致性（本任务正是跨层改动，逐层核对）**

- 读写链路完整：`buildinfo.Version` → `Deps` → `Handler.version` → `/public/system-config` → `PublicSystemConfig` DTO → `appInfo` store → 两个组件。四层全部实测贯通（API 实测 + 构建产物解压检索）。
- 错误路径已处理：取版本失败时 store 捕获并保持空串，组件隐藏版本展示 —— **不设 fallback 字面量**，否则等于把硬编码换个地方藏。
- 反向兼容已验证：既有 3 个字段在真实容器响应中均存在（实测）。

**诚实记录：偏离、发现与未决事项**

1. **规划期遗漏一处（由验收项抓出）**：侦察时只找到 Go 与 TS 两处硬编码，漏了 `internal/httpx/user_agent.go`。是验收项「`internal/` 内 `v0.5.2` 零命中」在执行中把它逼出来的。已补写 PRD 的 D1b 段落而非静默修掉 —— 这是**规划漏项**，不是我顺手扩大了范围。
2. **驱动层唯一例外**：`drivers/115_Open/upload.go` 的 `const ossUserAgent` 必须改 `var`（常量无法由运行期值派生）。只改关键字，值与 3 处用法均未动，已在 Constraints 中显式声明。
3. **匿名访问受 `public_index_enabled` 门控**（实测发现）：全新库该开关默认关闭，匿名请求 `/public/system-config` 得 401。**判定为可接受**：此时匿名用户被路由守卫送到登录页、根本不渲染 footer，不存在"该显示却没有"的场景；store 的降级路径恰好吃住这个 401。但这是实施期才确认的行为，属设计假设的实测验证。
4. **未顺手修 gofmt**：`internal/api/router.go` 与 `internal/app/wire_http.go` **原本就不符合 gofmt**（上游继承，分别建议改 64/50 行）。我只保证新增的 8 行风格一致，不做全文件格式化 —— 否则会制造与本次改动无关的大 diff。
5. **本机是全新库**：默认口令是 `admin/admin`（`must_change_password:true`、`password_change_reason:default_credentials`），**不是** `AGENTS.md` 记录的 `123456` —— 那是上一台机器经授权重置后的值。此处如实记录以免混淆。
6. **UI 层未做浏览器级验证**（本机无无头浏览器）：以「API 实测返回 v0.0.44」+「构建产物解压后确认已含 `public/system-config` 且不含任何版本字面量」作为替代证据。若需像素级确认，需后续在真实浏览器打开首页 footer 与后台「关于」。
7. **镜像仅 linux/amd64**，与既有发布惯例一致（`DOCKER_PLATFORM=linux/amd64`）；未做多架构。
8. **已做"拉回再验"**：删本地镜像 → 从 GHCR 真拉 → digest 与构建一致（`sha256:a864057d…`）→ 再跑一遍三项验证全过。避免"只验证了本地构建产物"的假安全感。
9. **未决观察（不在本任务范围，未擅自改）**：README 首屏声称 `118M`，实测 v0.0.44 **压缩后 41.8 MiB**、`docker images` 未压缩 **161MB** —— 三个口径互不一致。`118M` 是 `v0.0.1` 时代写下的数字（当时镜像内容少得多）。是否更新、以及该以哪个口径对外宣称，需你决定。
