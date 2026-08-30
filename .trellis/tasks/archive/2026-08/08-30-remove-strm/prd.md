# Remove all STRM-related features

## Goal

彻底移除 LitePan 中 **STRM 直连播放、STRM 刮削** 及所有相关能力（任务、API、前端、配置、自动化联动、存储、文档），使项目不再包含任何 `strm` 字样、路由、表、配置与 UI 入口，构建与测试仍通过，容器可正常运行。

## Requirements

- **后端彻底删除**：
  - 删除目录 `internal/strm/`（44 文件：service/coordinator/runner/scanner/metadata/writer/...）与 `internal/strmscrape/`（24 文件）
  - 删除 `internal/domain/strm.go` 与 `strm_dir_cache.go`（结构体与 `Repository`）
  - 删除 `internal/store/strm_repo.go`、`strm_dir_cache_repo.go` 及 `store.go` 中 `StrmTasks/StrmBranches/StrmDirCache` 注册、迁移中 STRM 建表语句
  - 删除 `internal/api/strm_admin.go`、`strm_play.go`、`strm_scrape.go` 及 `router.go` 中 `/strm/*`、`/strm/play/*`、`/admin/strm/*`、`/admin/strm-scrape/*` 路由
  - 删除 `internal/app/wire_strm.go` 及 `app.go`/`wire_*.go` 中 `strm.Service/Coordinator`、`strmScrape.Service` 注入与生命周期 `account_lifecycle.go` 中 `strm.Pause/Remove/ClearDirCache`
  - 删除 `internal/config` 中 `StrmDir`/`StrmDirForData`、`LITEPAN_STRM_DIR` 环境变量及 `config_test.go` 中相关用例
  - 删除 `internal/automation` 中 `AutomationActionStrm`、`AutomationActionStrmScrape` 及 `service.go/service_validate.go` 中相关分支与 `options` 中 `strm_tasks`
  - 删除 `internal/settings` 中 STRM 相关注册项（`strm` 设置）、`internal/embyproxy`、`file`、`spacecleanup` 等处对 `strm` 的引用
  - 删除 `drivers/LocalFs` 中 STRM 特殊搬运逻辑（若仅服务于 STRM 输出）
  - 清理所有 `grep -r -i strm` 结果（167 个文件）中残留引用，包含注释、日志 `ModuleStrm`、错误码
- **前端彻底删除**：
  - 删除 `web/src/api/strm.ts` 与 `strmScrape.ts`
  - 删除组件 `web/src/components/admin/{StrmSettingsPanel,StrmScrapePanel,StrmScrapeScopePicker,StrmScrapeSettings}.vue` 及 `TaskManagement.vue` 中 `STRM_TAB`、`scrape` tab 与 `strm.ts` 导入
  - 删除 `web/src/composables/useStrmDirectoryPrompt.ts`
  - 删除 `web/src/views/AdminView.vue` 中 `defaultTab: "strm"` 及 STRM 相关 tab 配置
  - 删除 `web/src/components/file/FileBrowser.vue` 中 `generateCurrentDirectoryStrm`、`strmPrompt`、`strmGenerating`、`strm-prompt-bar` 模板与样式
  - 删除 `web/src/components/file/FileTable.vue` 中 `generate-current-directory-strm` 事件与菜单
  - 删除 `web/src/api/automation.ts` 等处 `strm_tasks` 选项
  - 清理 `web/src` 中所有 `strm` 引用（`grep -r strm web/src` 约 30+ 文件）
- **配置与部署**：
  - 删除 `Dockerfile` 中 `STRM` 相关 `ENV`、`VOLUME`、`RUN mkdir -p /app/strm` 及 `docker-compose.yml`/`docker-compose.fnos.yml` 中 `strm:/app/strm` 挂载
  - 删除 `.dockerignore` 中若有 `strm` 例外（当前无，但需检查）
  - 删除 `README.md`、`docs/` 中 STRM 功能介绍段落与图片引用
  - 保留 `data/`、`mounts/`，仅移除 `strm/` 宿主机目录与容器挂载（用户若已有 `/app/strm` 数据可手动备份后删除）
- **兼容性**：
  - 不自动删除用户已生成的 `strm/` 目录内文件，仅移除代码；旧 `strm_tasks` 表在新版启动时不再读写（可保留或在迁移中 `DROP TABLE IF EXISTS`，选保留避免数据丢失恐慌）
  - 确保 `make lint` 的 `depguard` 不再报 `strm` 相关导入

## Constraints

- 不得遗漏任何 `strm` 大小写引用（`grep -ri strm` 最终为 0，除 Trellis 历史 `tasks/archive` 与 `journal` 外）
- 删除后 `GOWORK=off go vet ./...`、`GOWORK=off go test ./...`（无 `strm` 测试）仍通过；`cd web && npm run type-check && npm run build` 通过且产物不含 `strm` 块
- 容器 `docker build -t litepan-go:dev .` 成功，`docker run` 后 `curl /api/health ok` 且无 `/api/strm` 路由（应 404）
- 自动化中 `strm`/`strm_scrape` 动作需从校验与执行器中移除，否则创建旧规则会 panic
- 需同步更新 `spec/backend/backend` 与 `spec/web/frontend`（若任务后 spec 已过时，需 `trellis-update-spec`）

## Acceptance Criteria

- [ ] `grep -r -i strm --include="*.go" --include="*.ts" --include="*.vue" | grep -v ".trellis" | grep -v ".git" ` 结果为空（除 `drivers/all.go` 修复注释外）
- [ ] `internal/strm` 与 `internal/strmscrape` 目录已删除
- [ ] `internal/domain/strm*`、`internal/store/strm*` 已删除且 `store.go` 不再注册
- [ ] `internal/api/strm*` 已删除且 `router.go` 无 `/strm` 路由，`curl -i http://127.0.0.1:5211/api/strm/tasks` 返回 404
- [ ] `internal/app/wire_strm.go` 已删除且 `app.go` 启动不再引用 `strm`
- [ ] `internal/config/config.go` 无 `StrmDir`，`docker-compose.yml` 无 `strm:` 卷
- [ ] `web/src/api/strm*.ts` 已删除且 `web/src/components/admin/Strm*` 已删除，`AdminView.vue` 默认 tab 非 `strm`
- [ ] `web` 构建 `npm run build` 产物 `internal/api/web` 不含 `Strm` 文案，`grep -r strm web/dist` 无命中
- [ ] `GOWORK=off go build -o /tmp/litepan ./cmd/litepan` 成功（42M 级）
- [ ] `docker build -t litepan-go:dev .` 成功，`docker run -d --name litepan-test -p 5212:5211 ... litepan-go:dev` 后 `curl /api/health ok` 且无 STRM 日志 `ModuleStrm`
- [ ] `make lint`（若本机有 `golangci-lint`）或 `go vet ./...` 通过

## Notes

- Complex task：需 `design.md`（边界、数据流、兼容、回滚）与 `implement.md`（有序清单、验证、回滚点）后方可 `task.py start`
- 影响面广（约 167 文件），按 `trellis` 单 Owner 串行执行，禁止并发改 `router/store/domain` 同一文件
