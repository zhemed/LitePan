# 09-12-remove-dead-frontend-p2

## Goal

执行死代码排查报告（`09-12-investigate-dead-code-sweep` §9）的 **P2 清理**：删除前端 **12 个零引用文件（2,237 行）**；删除前逐个验证"是否仍有可达入口 / 功能是否由其它在线组件覆盖"，随后跑前端三连 + Go 门禁，发布 `0.0.42` 并完成本地部署验证。

## Background

- 报告依据：`.trellis/tasks/archive/2026-09/09-12-investigate-dead-code-sweep/research.md` §5（方法：`web/src` 全量 basename/stem 引用计数 + 全仓二次检索，12 项均为 0 引用）。
- 12 个文件：`api/spaceCleanup.ts`(71)、`components/admin/AdminStartupBanner.vue`(23)、`CacheRuntimeStats.vue`(86)、`CacheSettingsPanel.vue`(141)、`FuseManagement.vue`(1071)、`WebDAVSettings.vue`(122)、`composables/useConditionalPolling.ts`(46)、`useLiveElapsedClock.ts`(49)、`useStartupCountdown.ts`(54)、`useVirtualPosterWall.ts`(170)、`utils/coverPoster.ts`(356)、`utils/tmdbHit.ts`(48)。
- **本次新增的功能覆盖复核**（删除前的关键判断）：
  | 文件 | 复核结论 |
  |---|---|
  | `FuseManagement.vue`(1071) | FUSE 功能**另有在线 UI**：`api/fuse.ts` 被 `DashboardManagement.vue`/`SystemSettings.vue`/`AuxToolsManagement.vue`/`stores/accounts.ts` 使用 → 该组件是重复遗留 |
  | `CacheSettingsPanel.vue`/`CacheRuntimeStats.vue` | 缓存设置在 `SystemSettings.vue` 中被 `filterOutCacheSettings` **显式过滤**（不再展示），这两个面板无入口 |
  | `WebDAVSettings.vue` | WebDAV 后端仍在（`internal/share`），但**前端已无任何 WebDAV UI 入口**（除该文件自身） |
  | `AdminStartupBanner.vue`/`useStartupCountdown.ts` | 启动/恢复倒计时 UI 无其它实现，属已删流程遗留 |
  | 其余 6 个 | 均属已删功能（垃圾清理/海报墙/封面/TMDB）或通用工具（轮询/计时）无引用 |
- `AdminView.vue` 只加载 5 个视图（dashboard/accounts/settings/tasks/tools），12 个文件均不在其中 → 删除**零运行时影响**。

## Requirements

- **R1 删除**：删除上述 12 个文件（`git rm`），并重建前端产物（`npm run build` 会更新 `internal/api/web/assets`，属预期）。
- **R2 门禁**：`cd web && npm run type-check && npm run build && npm run check:memo` 全绿；`go build ./...`、`go vet ./...`、`go test ./...` 全绿（嵌入产物变化后需复跑）。
- **R3 复核**：删除后再次全量扫描 `web/src`，确认**无新增零引用文件**（仅允许原本就存在的入口/声明白名单）；并确认 `web/src` 中 `spaceCleanup`/`FuseManagement`/`WebDAVSettings`/`CacheSettingsPanel`/`CacheRuntimeStats`/`AdminStartupBanner` 等标识符无残留引用。
- **R4 保留项**：`web/src/constants/cacheSettings.ts`（被 `SystemSettings.vue` 使用）**不得删除**；`web/src/api/fuse.ts`（在线）同样保留 —— 二者不在删除清单内。
- **R5 发版**：bump `0.0.42`（README + docker-compose）；镜像三 tag；顺序 **push main → tag → push tag → 最后 release**；本地容器重建 + health/登录/任务汇总三连。
- **R6 记录**：PRD 勾选附证据；journal 记录删除清单、覆盖复核结论与"后端仍在但前端无入口"的说明（WebDAV/缓存）。

## Constraints

- 不扩张范围：不动 `drivers/template`+httpx OAuth（P3）、不动 `/accounts/{id}/refresh-auth`（P4）。
- 不改任何后端代码；不改前端**在线**组件。
- 不触碰生产机；版本规则 `0.0.x` 递增 → `0.0.42`。
- 门禁：`pre-start`/`mark-check`/`pre-archive` 全过；pre-archive 被拒时不得继续 archive。

## Acceptance Criteria

- [x] 12 个文件已删除，且删除前每个文件均有"无可达入口/功能已被覆盖"的复核结论
      → 第一层 12 个已删；每个都有复核结论（见 Background 表 + Notes）
- [x] `web/src` 复扫无新增零引用文件；被删标识符无残留引用
      → 复扫发现 **11 个传递孤儿**（第二层 7 + 第三层 4，其引用者全部属于已删集合，已用 `git grep HEAD` 逐个核实）；迭代删除后**收敛到 0 个零引用文件**（源文件 223 → 204）
- [x] 前端三连（type-check/build/check:memo）与 `go build/vet/test` 全绿
      → 全绿；**构建产物零 churn**（反证这些文件本就不在 bundle 内）；`go test ./...` exit 0（42 包）
- [x] 未删除 `constants/cacheSettings.ts` 与 `api/fuse.ts`（R4 保留项）
      → 两者均保留（`cacheSettings.ts` 被 `SystemSettings.vue` 引用、`api/fuse.ts` 被 3 个在线组件引用）
- [x] 版本号 `0.0.42` 已更新（`README.md` + `docker-compose.yml`）
      → README 2 处 + docker-compose 1 处
- [x] 镜像 `ghcr.io/zhemed/litepan:0.0.42` 三 tag 已推送；tag 指向修复提交（API 复核）；release 已创建（顺序正确）
      → 三 tag 同 digest `sha256:6617d652…`（与 0.0.41 相同，属预期：前端产物未变）；tag `v0.0.42` = `75be52b`（API 复核一致）；release https://github.com/zhemed/LitePan/releases/tag/v0.0.42；顺序 push main → tag → push tag → release ✔
- [x] 本地容器已重建到 0.0.42，health / form 登录 / 任务汇总三连通过
      → ImageID `0c11de670d67`、`Restarts=0`；health ok；登录 ok；任务汇总 `total=13 success=13`
- [x] 删除清单与覆盖复核结论已写入 journal
      → Session 129

## Notes

- **实际删除量：23 个文件 / -3,017 行**（原报告列 12 个；迭代到不动点后新增 11 个传递孤儿）。
- **方法论修正（重要）**：死代码清理必须**迭代到不动点**——删除第一层后要重新扫描，因为"只被死文件引用"的模块会成为新的孤儿。本轮三层分别是 12 → 7 → 4 → 0。
- **反证证据**：删除前后前端构建产物**零 churn**，说明这些文件从未进入 bundle（与"零引用"判定一致）。
- 删除的 UI 面板（WebDAV/缓存/启动横幅）未来若需恢复，可从 git 历史或上游内容对照取回；本轮不影响任何在线功能。
- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.

- `scope=lightweight`：纯前端死文件删除 + 产物重建 + 发版。
- **风险说明**：删除的 UI 面板（WebDAV/缓存/启动横幅）未来若需恢复，可从 git 历史取回或按内容对照从上游移植；本轮不影响任何在线功能。
- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
