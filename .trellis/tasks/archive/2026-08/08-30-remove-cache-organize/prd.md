# Remove cache tasks and directory organization features

## Goal

彻底移除 LitePan 中 **缓存任务**（`cacheretention` 缓存保持任务及其 API/前端/自动化）与 **目录整理**（`mediaorganize` 媒体整理、`classifyorganize` 归档整理、`aiorganize` AI 整理、`coverextract` 封面提取及其关联）相关全部能力，使项目不再包含任何缓存任务、目录整理入口，且构建与容器仍正常。

## Requirements

- **缓存任务彻底删除**：
  - 删除目录 `internal/cacheretention/`（15 文件：`service.go/crud.go/coordinator.go/scanner.go/...`）与 `internal/fusereadcache/`（若仅服务于缓存任务则删，否则保留 `fusereadcache` 若仍被 `fusemount` 使用需评估）
  - 删除 `internal/domain/cache_retention.go`（`CacheRetentionTask`、`Repository`）
  - 删除 `internal/store/cache_retention_repo.go` 及 `store.go` 中 `CacheRetentionTasks` 字段与 `New()` 注册、迁移中 `cache_retention` 建表（`0010_cache_retention.sql`、`0011_cache_retention_run_fp.sql` 保留但不再读写）
  - 删除 `internal/api/cache_retention_admin.go` 及 `router.go` 中 `Deps{CacheRetention}` 与 `/admin/cache-retention` 路由
  - 删除 `internal/app/wire_cache_retention.go` 及 `app.go`/`wire_services.go` 中 `cacheRetention` 注入与 `account_lifecycle.go` 中 `retention.Pause/Remove`
  - 删除 `web/src/api/cacheRetention.ts` 与 `web/src/components/admin/CacheRetentionPanel.vue`
  - 删除 `internal/automation` 中 `AutomationActionCacheClear` 的 `cache_clear` 相关分支（若仅服务于缓存任务）
  - 清理 `grep -r -i "cacheretention\|CacheRetention"` 残留（约 21 文件）
- **目录整理彻底删除**：
  - 删除目录 `internal/mediaorganize/`（30+ 文件：`service.go/planner/*`）、`internal/classifyorganize/`（3 文件）、`internal/aiorganize/`（4 文件）、`internal/coverextract/`（若仅服务于整理则删）
  - 删除 `internal/domain/media_organize.go` 及其 `MediaOrganizeTask`、`Repository`，以及 `classifyorganize`/`aiorganize` 相关的 domain
  - 删除 `internal/store/media_organize_repo.go`、`store.go` 中 `MediaOrganizeTasks`、`core` 中相关
  - 删除 `internal/api/media_organize.go`、`ai_organize.go`、`classification.go` 及 `router.go` 中相关路由
  - 删除 `internal/app/wire_mediaorganize.go` 及 `app.go` 中 `mediaOrganize` 注入
  - 删除 `web/src/api/mediaOrganize.ts`、`web/src/components/admin/MediaOrganizePanel.vue`/`MediaOrganizeSettings.vue`、`web/src/composables/useOrganizePlanPreview.ts` 等
  - 删除 `internal/automation` 中 `AutomationActionOrganize` 相关
  - 清理 `grep -r -i "mediaorganize\|classifyorganize\|aiorganize"` 残留（约 40+ 文件）
- **前端与自动化联动**：
  - 删除 `web/src/components/admin/TaskManagement.vue` 中 `organize` tab 相关（保留 `automation` 等其他 tab）
  - 删除 `web/src/views/AdminView.vue` 中若存在的 `organize` 入口
  - 清理 `automation` 中 `organize` 动作的校验与执行（若彻底移除则整个 `organize` 动作类型移除）

## Constraints

- 保留 `internal/cache` 核心（文件列表缓存、命中追踪等，`cache.Service` 仍被 `file.Service` 使用，不得删除）
- 保留 `internal/file`、`internal/fusemount` 等核心业务
- 删除后 `grep -r -i "cacheretention\|mediaorganize"` 在 Go/TS/Vue 中除 Trellis 历史外为 0
- `GOWORK=off go vet ./...`、`go build`、`cd web && npm run type-check && npm run build` 必须通过
- 容器 `docker build -t litepan-go:nocache-organize .` 成功，`curl /api/health ok` 且无相关路由

## Acceptance Criteria

- [ ] `rm -rf internal/cacheretention` 后 `ls internal/cacheretention` 无此目录
- [ ] `rm -rf internal/mediaorganize internal/classifyorganize internal/aiorganize` 后无此目录
- [ ] `grep -r -i "cacheretention" --include="*.go" | grep -v ".trellis"` 为 0
- [ ] `grep -r -i "mediaorganize\|classifyorganize\|aiorganize" --include="*.go" --include="*.ts" --include="*.vue" | grep -v ".trellis"` 为 0
- [ ] `internal/api/cache_retention_admin.go`、`media_organize.go` 等已删除且 `router.go` 无相关路由，`curl -i /api/admin/cache-retention/tasks` 404
- [ ] `web/src/api/cacheRetention.ts`、`mediaOrganize.ts` 已删除且 `CacheRetentionPanel.vue`、`MediaOrganizePanel.vue` 已删除
- [ ] `GOWORK=off go vet ./...` PASS，`go build -o /tmp/litepan` PASS
- [ ] `cd web && npm run type-check` PASS，`npm run build` PASS
- [ ] `docker build -t litepan-go:nocache-organize .` PASS，`docker run` 后 `curl /api/health ok`

## Notes

- Complex task：需 `design.md` 与 `implement.md` 后方可 `task.py start`
- 影响面：约 60+ 文件，需按 trellis 单 Owner 串行执行

