# 修复后台 /> 残留

## Goal
移除 `web/src/views/AdminView.vue:303` 孤立 `/>` 文本节点（`2d1412a` 移除 `AdminAnnouncementModal` 残留），消除后台所有页面面包屑下方异常 `/>`，并按版本基线发布 `0.0.2`。

## Background
- 用户截图 `Image sha256:56a26ef... 506x138`：`后台 / 存储管理` 右侧红框内 `/>`
- `git show 2d1412a -- web/src/views/AdminView.vue`：仅删 `AdminAnnouncementModal` 前 3 行，末行 `/>` 残留成 `web/src/views/AdminView.vue:303`
- `git blame 303`：原 `39f7261 增加远端通知入口` 的 modal 闭合符
- 当前 `HEAD web/src/views/AdminView.vue 298-305` 仍含该行，`GOWORK=off go vet` 不报错但 `vue-tsc` 视为文本节点渲染

## Requirements
- 删除 `web/src/views/AdminView.vue:303` 的 `    />` 单独行，前后空行规整（保留 `WarningBanner` 与 `AdminEmptyState` 各一空行间隔）
- 不改其他逻辑：`nav/page/crumbs/KeepAlive/SystemSettings` 等保持不变
- 版本按 `AGENTS.md`：`0.0.1` 稳定基线 → 本修复 `0.0.2`（fix 递增），同步 `README` 与 `ghcr.io/zhemed/litepan:0.0.2`/`v0.0.2` 双标签
- 遵循 `web` 前端规范：`vue-tsc -b` 与 `vite build` 必须通过，构建产物落 `internal/api/web` 并嵌入

## Constraints
- 所有写操作必须在 `task.py start` 后（`in_progress`）执行，`AGENTS.md 项目强制规则`
- 不引入新依赖，不改 `depguard`，单行修复最小化 diff

## Acceptance Criteria
- [ ] `grep -n "    />" web/src/views/AdminView.vue` 仅剩 2 处（`AdminEmptyState:309` 与 `component :is` 闭合），`303` 已删
- [ ] `grep -c "/>" web/src/views/AdminView.vue` 行为与修复前一致但不再含孤立文本节点；页面 `curl` 或本地 `vite` 预览不再出现 `/>`
- [ ] `cd web && npm run type-check` 通过
- [ ] `GOWORK=off go vet ./...` 通过
- [ ] `GOWORK=off go build -o /tmp/litepan` 通过，`docker build -t litepan-go:fix-adminview` / `ghcr.io/zhemed/litepan:0.0.2` 成功并 `public`
- [ ] `README` 与 `docker-compose.yml` 示例标签更新至 `0.0.2`/`v0.0.2`，`git tag v0.0.2` + `gh release` 完成
- [ ] 本地 `http://127.0.0.1:5211` 登录后台各 tab 无 `/>`，回归 `curl /api/health 200`
