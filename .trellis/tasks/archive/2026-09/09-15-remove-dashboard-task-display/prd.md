# 删除仪表盘三处「假任务」展示（运行任务 / 任务总数 / 后台任务）

## Goal

按调查结论（`.trellis/tasks/archive/2026-09/09-15-investigate-task-display-removal/research.md` §5）删掉仪表盘上三处**恒为零/恒为空**的任务展示及其专属样式：

| # | 展示 | 真实值 |
|---|---|---|
| T1 | 控制台状态行「运行任务」 | `computed(() => 0)` 硬编码 |
| T2 | 概况卡片「任务总数」 | `computed(() => 0)` 硬编码 |
| T3 | 右列面板「后台任务」（含恒空列表模板） | `computed(() => [])` 恒空数组，列表分支永不渲染 |

**后端零改动**（调查已核实：仪表盘不发任何任务请求，全仓无通用 `/tasks` 端点）。上传任务与自动化两条真实链路**不得受影响**。

## Background

- 三处于 `1bcfac8`（2026-08-30 移除缓存任务与目录整理）被就地改成桩，此后一直显示假 0；`dashboard` 面板的空态文案「暂无后台任务」与副行「0 个任务 · 0 个运行中」是用户实际看到的全部内容。
- 删除后可见面变化：状态行 **4 → 3** 项、概况卡片 **3 → 2** 张、右列面板 **2 → 1** 个。
- 前端无自动化测试，验收只能靠 `vue-tsc` + 构建产物扫描 + **浏览器实测**（这正是本仓库 `web/frontend/quality-guidelines.md` 的条款：改动触及 `web/src/**` 且效果只在渲染后可见）。

## Requirements

### D1 模板（`web/src/components/admin/DashboardManagement.vue`）

- 删「运行任务」`.hero-metric` 块（调查记录 `:327-330`，以实际文本锚点为准）
- 删「任务总数」`.overview-card` 块（`:347-355`）
- 删整个「后台任务」`<article class="dashboard-panel">`（`:441-467`）

### D2 脚本（同文件）

- 删 `enabledTaskCount`、`totalTaskCount`（`:56-57`）
- 删 `taskSummaries`（`:97-106`）
- 复核：`grep -rn "taskSummaries\|enabledTaskCount\|totalTaskCount" web/src/` **零命中**

### D3 样式

- 同文件共享规则里只有 `.task-list,` 属于被删面板（`:825`）→ **只删这一行**，`.account-list` / `.dashboard-side` / `.notice-list` 保留
- 同文件 `.task-row*` / `.task-progress*` 连续块（`:984-1052`）→ 整块删
- `web/src/styles/skins/brutal.css` 的 3 条 `.dashboard-page .task-row` / `.task-progress` / `.task-progress span`（`:66-79`）→ 删；**同处的 `.dashboard-page .account-row`（`:81`）必须保留**
- **不动** `web/src/styles/upload-task-panel.css`（那是上传任务面板的全局样式，同名类不冲突）

### D4 重建 embed + 质量门

- `cd web && npm run type-check` exit=0、`npm run build` 成功（**本轮会有 embed churn**，前端改了）
- `make lint` 0 issues、`GOWORK=off go vet ./...` exit=0、`GOWORK=off go test ./...` 全包 ok
- 基线：`deadcode` ≤ 7、`unused` = 0、**前端零引用 = 0**、`gofmt` 无新增脏文件

### D5 产物与零残留

- 构建产物中 `运行任务` / `任务总数` / `暂无后台任务` / `个运行中` **零命中**（对照：改前在 `DashboardManagement-*.js.gz` 各命中 1 次）
- `grep -rn "taskSummaries\|enabledTaskCount\|totalTaskCount" web/src/ internal/` 零命中

### D6 浏览器验收（真实渲染，必须做）

在**新代码**起的临时实例上（`:5311` + 数据副本，二进制直接 `go build` 后运行，不碰 `:5211`）：

- 仪表盘：状态行 **3 项**（接入 / 在线 / 待确认错误）、概况卡片 **2 张**（缓存空间 / 未读通知）、右列面板 **1 个**（日志与通知）；页面无「任务总数 / 运行任务 / 后台任务 / 暂无后台任务」文案
- 布局无塌陷（无空白缺口 / 无错位），截图目视确认
- **反例回归**：上传任务面板仍可打开（`bw` 点文件页任务入口）、「任务管理 → 自动联动」页仍正常渲染
- 无 JS 错误（`window.__err` 为空）

### D7 收尾

`skill trellis-check` → `mark-check` → 勾选验收项 → `pre-archive` → `archive` → `add_session.py` → `git push`

## Constraints

- **只动这三处展示**：`DashboardManagement.vue` 的其它卡片/面板/指标（接入、在线、待确认错误、缓存空间、未读通知、存储账号、日志与通知）零改动
- **后端零改动**：`internal/**` 只允许 `internal/api/web/**`（embed 重建产物）变化
- **近名物零改动**：`upload/TaskPanel.vue`、`styles/upload-task-panel.css`、`api/upload.ts`、`AutomationPanel.vue`、`TaskManagement.vue`、`AppFooter.vue` 一律不动
- 不改 `.golangci.yml`、`Makefile`、`Dockerfile`、`go.mod`
- **不发版、不 bump 版本、不动本机 `:5211` 容器**（发布与否交用户另定）
- 本机 `danger-full-access`、审批关闭 —— 不请求 escalation
- 不改归档任务与历史 journal

## Acceptance Criteria

- [x] 三处展示的模板块已删除（`.hero-metric` 运行任务、`.overview-card` 任务总数、「后台任务」`<article>`）
- [x] 三个计算属性 `enabledTaskCount`/`totalTaskCount`/`taskSummaries` 已删除，全仓零引用
- [x] 共享选择器只少 `.task-list,` 一行，`.account-list`/`.dashboard-side`/`.notice-list` 仍在；`.task-row*`/`.task-progress*` 块已整体删除
- [x] `brutal.css` 3 条 `.dashboard-page .task-*` 已删，`.dashboard-page .account-row` 仍在
- [x] `upload-task-panel.css` 零改动；`TaskPanel.vue`/`api/upload.ts`/`AutomationPanel.vue`/`TaskManagement.vue`/`AppFooter.vue` 零改动
- [x] 质量门全绿：`make lint` 0 issues、`go vet` exit=0、`go test ./...` 全包 ok、`vue-tsc -b` exit=0、`npm run build` 成功
- [x] 构建产物中 `运行任务`/`任务总数`/`暂无后台任务`/`个运行中` **零命中**（`zcat assets/DashboardManagement-*.js.gz | grep -c`）
- [x] 基线不劣化：`deadcode` ≤ 7、`unused` = 0、前端零引用 = 0、`gofmt` 零新增
- [x] **浏览器验收通过**：状态行 3 项、卡片 2 张、右列面板 1 个、无相关文案、布局无塌陷（截图）
- [x] **反例回归通过**：自动化页（任务管理→自动联动）实测正常渲染、`window.__err` 为空；**上传任务面板的点击触达未能在本实例完成**（0 账号 → 文件页只显示空态、无任务入口），按 PRD 允许的证据替代：7 个近名文件 git 零改动 + 上传 API 三端点 200 + 面板文案/样式仍在重建后的产物（已在 check 记录中写明）
- [x] 后端与部署文件零改动（仅 `internal/api/web/**` embed 产物 churn）；未发版、未动 `:5211`
- [x] 任务归档、journal 记录、`main` 与 `origin/main` 同步

## Notes

- Scope `lightweight`（单文件前端删除 + 3 条皮肤规则），与 `09-12-remove-dead-frontend-p2`、`09-15-remove-playback-remote-reader` 同规格。
- **替代路径已记录**（本任务不采用）：若想保留"任务"展示，正确做法是把这两处计数改接真实接口 `GET /api/files/upload/tasks/summary`（实测可用），而不是继续显示硬编码 0。用户已选择**删除**，该替代路径留档在 research.md §0。
- 与 `09-15-remove-fuse` 的关系：那一轮删了同文件同区域的「FUSE 挂载点」卡片；本轮完成后仪表盘右列只剩「日志与通知」，卡片只剩 2 张 —— 若布局出现留白，按 D6 目视确认并记录（**不顺手改版式**，除非塌陷）。
