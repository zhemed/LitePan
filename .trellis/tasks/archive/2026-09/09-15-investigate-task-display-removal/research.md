# 仪表盘的「运行任务 / 任务总数 / 后台任务」能否彻底移除 —— 调查报告

> 调查任务：`09-15-investigate-task-display-removal`（只读，未改任何代码/配置/DB/容器）
> 结论日期：2026-09-15 · 基线：`main @ fad06dc`（v0.0.46）

---

## 0. 结论（TL;DR）

**可以彻底移除，而且必须移除 —— 这三处展示今天全部是"假数据"：两个计数器是硬编码 `0`，一个列表是恒空数组。**

| 展示 | 实际值 | 性质 |
|---|---|---|
| 控制台状态行「运行任务」 | 恒 `0` | `const enabledTaskCount = computed(() => 0)` —— **硬编码** |
| 概况卡片「任务总数」 | 恒 `0` | `const totalTaskCount = computed(() => 0)` —— **硬编码** |
| 右列面板「后台任务」 | 恒「0 个任务 · 0 个运行中」+ 恒「暂无后台任务」 | `const taskSummaries = computed(() => [] as Array<{…}>)` —— **恒空数组**，列表分支永不渲染 |

**来历（git 实锤）**：提交 `1bcfac8`（*refactor(cache,organize): remove cache tasks and directory organization*，2026-08-30）把这三个计算的**真实实现换成了桩**：

```diff
-const enabledTaskCount = computed(                    →  +const enabledTaskCount = computed(() => 0);
-const totalTaskCount  = computed(                     →  +const totalTaskCount  = computed(() => 0);
-const taskSummaries   = computed(() => [ …真实列表… ]) →  +const taskSummaries   = computed(() => [] as Array<{…}>);
```

即：**上游那批"后台任务"（缓存任务/目录整理等）被删时，仪表盘的展示没有一起删，而是就地改成了空壳**，于是从 v0.0.7 一路发到 v0.0.46 都在显示假的 0。

**三处都零后端依赖**：不调用任何"任务"API，没有专属的前端文件/导出（三个标识符在 `DashboardManagement.vue` 之外引用数均为 **0**），因此删除**不牵动任何后端代码**。

**唯一的信息损失**：仪表盘不再显示任务类计数。但**真实任务信息并未丢失**：
- 上传任务 → 文件页右上角任务面板（`TaskPanel.vue`）+ 首页页脚状态入口，走 `/api/files/upload/tasks*`（**活跃**）
- 自动化运行 → 「任务管理 → 自动联动」页（`TaskPanel`？不，是 `AutomationPanel.vue`，**活跃**）

> 顺带说明一个**真实数据源就摆在那里却没用**：`GET /api/files/upload/tasks/summary` 实测返回 `{"total":0,"counts":{}}`，前端 `uploadApi.tasksSummary()` 也确实在用（`useUploadTaskStream.ts:93`）—— 也就是说这三处如果还想保留，**正确做法是接这个真实接口**，而不是继续显示硬编码 0。本报告的删除建议基于"用户要的是删掉"，但这条替代路径已写明供选择。

---

## 1. Q1 逐处定位（文件:行号 + 渲染条件）

全部在 `web/src/components/admin/DashboardManagement.vue`（`<style scoped>`，故样式不外溢）：

| # | 展示 | 模板位置 | 计算属性 | 渲染条件 |
|---|---|---|---|---|
| T1 | **运行任务** | `:328-329`：`<div class="hero-metric"><strong>{{ enabledTaskCount }}</strong><span>运行任务</span></div>` | `:56` `computed(() => 0)` | 恒渲染（控制台状态行第 3 个指标）|
| T2 | **任务总数** | `:351-354`：`<article class="overview-card"><strong>{{ totalTaskCount }}</strong><span>任务总数</span></article>` | `:57` `computed(() => 0)` | 恒渲染（概况卡片第 1 张）|
| T3 | **后台任务** | 面板 `:441-467`（`<article>` 起于 441）：标题副行 `:445` `{{ totalTaskCount }} 个任务 · {{ enabledTaskCount }} 个运行中`；列表 `:449-465`（`v-if="taskSummaries.length"`）| `:97-106` `computed(() => [])` | 副行恒渲染；**列表分支恒不渲染**（空数组）；`v-else` 空态 `:466`「暂无后台任务」恒渲染 |

**同文件里与被删项无关、但有联动关系的**：`taskSummaries` 的类型形状（`title/icon/count/enabled/detail/progress/tone/updated`）只服务这一个面板；`OverviewResult` 联合类型（`:43-48`）**不含**任何任务类型，不受影响。

---

## 2. Q2 数据源定性（每处都判定为"假"）

| 展示 | 判定 | 证据 |
|---|---|---|
| T1 运行任务 | **硬编码恒零** | `DashboardManagement.vue:56` —— 无 ref、无请求、无 props |
| T2 任务总数 | **硬编码恒零** | 同文件 `:57` |
| T3 后台任务（计数）| **硬编码恒零** | 副行直接引用上面两个 `computed` |
| T3 后台任务（列表）| **恒空数组** | 同文件 `:97-106` —— `computed(() => [] as Array<{…}>)`，无数据源 |

**"是否存在专属前端文件/导出"核查**（命令见 §9）：

```
enabledTaskCount   组件外引用 = 0 处
totalTaskCount     组件外引用 = 0 处
taskSummaries      组件外引用 = 0 处
```

**"是否存在后端端点/字段"核查**：仪表盘 `loadOverview()`（`:112-121`）只发 5 个请求 —— `accountsApi.list()`、`fetchCacheStats()`、`fetchNotifications()`、`fetchUnreadCount()`、`logsApi().stats()` —— **没有任何任务请求**。全仓也**没有**通用 `/api/admin/tasks` 端点（`grep -c '"/tasks"' internal/api/router.go` = **0**）；`upload/tasks*` 那 10 条属于上传子系统，见 §3。

**构建产物交叉验证**：这四个文案确实随包发布（同一 chunk 各命中 1 次），说明它们是**live 代码里的假数据**，而不是"未被构建的遗留"：

```
zcat internal/api/web/assets/DashboardManagement-*.js.gz | grep -o "暂无后台任务\|运行任务\|任务总数\|个任务\|个运行中" | sort | uniq -c
      1 个任务
      1 个运行中
      1 任务总数
      1 暂无后台任务
      1 运行任务
```

---

## 3. Q3 反例清单（这些"任务"是活的，绝不能删）

| 近名物 | 位置 | 证据（为什么不能删）|
|---|---|---|
| **上传任务面板** | `web/src/components/upload/TaskPanel.vue`（`FileBrowser.vue:1043` 渲染，`:34` 引入）| 文件页真实功能；`uploadApi` 提供 `uploadTaskLabel`/`openUploadTaskPanel` 等（`FileBrowser.vue:170-177`）|
| **上传任务 API** | `internal/api/router.go:235-247`（`/api/files/upload/*` 共 10 条）+ `internal/upload/**` + `upload_tasks` 表 | 实测 `GET /api/files/upload/tasks` → 200 `{"data":[]}`、`…/tasks/summary` → 200 `{"total":0,"counts":{}}`、`…/runtime` → 200；前端经 `useUploadTaskStream.ts:93` 调用 `uploadApi.tasksSummary()` |
| **自动化（自动联动）** | `TaskManagement.vue` → `AutomationPanel.vue`（nav `tasks` 页签，`AdminView.vue:21,55-58`）+ `internal/automation/**` + `automation_rules`/`automation_runs` | 「任务管理」页 = 自动联动，是**现行功能**；与会话中"任务总数/后台任务"只共享"任务"这个词 |
| **首页页脚任务入口** | `AppFooter.vue:38,130`（`useHomeFooterStatus().openTaskPanel`）| 打开的是**上传**任务面板（`:129` `status.uploadTaskLabel`）|
| **同文件其它卡片/面板** | 概况卡片「缓存空间」(cacheStats)、「未读通知」(unreadCount)；面板「存储账号」(accounts)、「日志与通知」(logStats)；状态行「接入/在线」(accounts)、「待确认错误」(recentErrorCount) | 全部接真实数据（`loadOverview` 的 5 个请求 + `accounts`），**不在删除范围** |
| **皮肤样式里的同名类** | `web/src/styles/upload-task-panel.css:400,416` 的 `.task-list`/`.task-row` | 属于**上传面板**的全局样式；与 `DashboardManagement.vue` 的 `<style scoped>` 同名但**互不影响**，且前者要保留 |

---

## 4. Q4 运行期证据

**实例数据**（`data/litepan.db` 只读）：`upload_tasks` 0 行、`automation_rules` 0 行、`automation_runs` 0 行（本机无账号，故三处必然显示 0 —— 这也说明"看数字为 0"不足以判定，必须看**代码**）。

**浏览器实测**（真实例 `:5211`，v0.0.46，`bw` + DOM 断言）：

```json
{"hero":["0=接入","0=在线","0=运行任务","0=待确认错误"],
 "cards":["0=任务总数","0 B=缓存空间=清理","0=未读通知"],
 "panels":["存储账号 |  | 0 个接入 · 0 个在线 |  | 刷新 | 还没有添加存储账号",
           "后台任务 |  | 0 个任务 · 0 个运行中 |  | 暂无后台任务",
           "日志与通知 |  | 近 24 小时 | 135 | 日志总数 | 0 | 未读通知 | …"],
 "taskRows":0,"taskList":0}
```

`taskRows=0` / `taskList=0` 直接证明**列表分支从未渲染**（`v-if="taskSummaries.length"` 永远为假）。

**删除后的可见面变化**（实测当前布局推算）：

| 区域 | 现在 | 删除后 |
|---|---|---|
| 控制台状态行 | 接入 / 在线 / **运行任务** / 待确认错误（4 项）| 接入 / 在线 / 待确认错误（**3 项**）|
| 概况卡片 | **任务总数** / 缓存空间 / 未读通知（3 张）| 缓存空间 / 未读通知（**2 张**）|
| 右列面板 | **后台任务** / 日志与通知（2 个）| 日志与通知（**1 个**）|

---

## 5. Q5 分层删除清单（供后续任务直接引用）

### 第 1 层：模板（3 处）

| 对象 | 动作 |
|---|---|
| T1 | 删 `:327-330` 的整个 `.hero-metric` 块（运行任务）|
| T2 | 删 `:347-355` 的整个 `.overview-card` 块（任务总数，`<article>` 起于 347）|
| T3 | 删 `:441-467` 的整个「后台任务」`<article class="dashboard-panel">` |

### 第 2 层：脚本（3 个计算属性）

- 删 `:56` `enabledTaskCount`、`:57` `totalTaskCount`
- 删 `:97-106` `taskSummaries`
- 复核：删后 `grep -n "taskSummaries\|enabledTaskCount\|totalTaskCount" web/src/` 应**零命中**

### 第 3 层：样式

| 位置 | 动作 | 注意 |
|---|---|---|
| `DashboardManagement.vue:823-829` | 共享规则 `.account-list, .dashboard-side, .task-list, .notice-list { … }` —— **只删 `:825` 的 `.task-list,` 这一行** | 其余三个选择器都还在用，不能整块删 |
| `DashboardManagement.vue:984-1052` | 连续块 `.task-row*`（含 `:last-child`、`__icon`、`__icon--purple/amber`、`__main`、`__title`、`small`）与 `.task-progress*` —— **整块删** | 块尾到 `.log-snapshot`（`:1055`）之前 |
| `web/src/styles/skins/brutal.css:66-79` | 3 条 `.dashboard-page .task-row` / `.task-progress` / `.task-progress span` —— 删 | 同处的 `.dashboard-page .account-row`（`:81`）**必须保留** |
| `web/src/styles/upload-task-panel.css` | **不动** | 那是上传任务面板的样式 |

### 第 4 层：后端

**无需改动**（核实结论：三处零后端依赖；`upload/*` 与 `automation/*` 端点原样保留）。

### 验证点

1. `cd web && npm run type-check` exit=0（删掉未使用的 computed/模板后不会有类型残留）
2. `grep -rn "taskSummaries\|enabledTaskCount\|totalTaskCount" web/src/` 零命中
3. `grep -c "暂无后台任务\|运行任务\|任务总数" web/src/` → 0
4. `npm run build` 重建 embed；产物中三处文案**零命中**（对照：现在各 1 命中）
5. 浏览器验收：状态行 4→3 项、卡片 3→2 张、右列面板 2→1 个，**布局无塌陷**；上传任务面板与「任务管理 → 自动联动」**照旧可用**
6. 回归：`make lint`/`go test`（前端改动不影响 Go，但按项目质量门一并跑）

### 回滚点

单提交 `git revert`；前端可 `git checkout HEAD -- web/src` + 重建 embed。

---

## 6. 风险与建议

| 风险 | 说明 | 建议 |
|---|---|---|
| 仪表盘不再有"任务"类计数 | 三处删掉后，仪表盘只反映账号/缓存/通知/日志 | 可接受：真实任务入口在文件页面板、页脚、任务管理页**都还在**；若想保留展示，正确做法是接 `GET /api/files/upload/tasks/summary`（实测可用），而不是留假 0 |
| 右列面板变少导致留白 | 右列将从 2 个面板变 1 个 | 建议同任务里目视确认；必要时用 `bw` 截图与左列对比，或把「日志与通知」上移 |
| 皮肤样式残留 | `brutal.css` 的 3 条规则删掉后无引用；若漏删只是死样式，不会报错 | 一并删；但**别碰**同处的 `.account-row` 规则 |
| 误伤近名物 | `task-list/task-row` 在上传面板里是**另一套全局样式**；`tasksSummary` 是上传 API | 删除范围严格限定在 `DashboardManagement.vue` 与本报告的 3 条皮肤规则 |

---

## 7. 一句话回答

**能删，而且这本就是一次"删功能没删干净"的收尾**：三处展示自 2026-08-30 起就是硬编码 0 与恒空数组（`1bcfac8` 留下的桩），零后端依赖、零跨文件引用；删除只影响仪表盘的 3 个视觉元素，上传任务与自动化两条真实链路完全不受影响。

---

## 8. 与既有清理的关系

- `09-15-remove-fuse` 删掉了同文件同区域的「FUSE 挂载点」卡片；本轮若执行，仪表盘就是**一次整体瘦身**（卡片 3→2、状态行 4→3、右列面板 2→1）。
- 同类"工具不报"的死代码：`deadcode`/`unused` 对前端导出与硬编码值都**不可见**（本项目 `guides/dead-code-guide.md` 已有"引用计数须剥注释""工具不报 ≠ 活代码"两条），本轮再次印证：**唯一可靠的判据是读渲染链路 + 运行期实测**。

---

## 9. 可复核命令（本报告每条结论的复跑入口）

```bash
cd /root/LitePan

# Q1 逐处定位
grep -n "enabledTaskCount\|totalTaskCount\|taskSummaries" web/src/components/admin/DashboardManagement.vue
sed -n '300,340p;440,470p' web/src/components/admin/DashboardManagement.vue

# Q2 定性（硬编码 / 恒空 / 跨文件引用）
sed -n '56,57p;97,106p' web/src/components/admin/DashboardManagement.vue
for s in enabledTaskCount totalTaskCount taskSummaries; do
  printf '  %-18s 组件外引用 = %s\n' "$s" "$(grep -rn "\b$s\b" web/src/ | grep -vc 'DashboardManagement.vue')"
done
grep -n "type OverviewResult" -A 6 web/src/components/admin/DashboardManagement.vue
grep -c '"/tasks"' internal/api/router.go        # → 0，无通用任务端点

# Q2 来历（谁把真实实现换成桩）
git log --oneline -S "taskSummaries" -- web/src/components/admin/DashboardManagement.vue
git show 1bcfac8 -- web/src/components/admin/DashboardManagement.vue | grep -n "taskSummaries\|enabledTaskCount\|totalTaskCount"

# Q2 构建产物交叉验证
zcat internal/api/web/assets/DashboardManagement-*.js.gz | grep -o "暂无后台任务\|运行任务\|任务总数" | sort | uniq -c

# Q3 反例（活跃链路）
grep -c 'upload/tasks' internal/api/router.go    # → 10
curl -s -b <cookie> http://127.0.0.1:5211/api/files/upload/tasks/summary
grep -rn "TaskPanel" web/src/components/file/FileBrowser.vue
grep -n "automation" web/src/views/AdminView.vue | head -3

# Q4 运行期
python3 -c "import sqlite3;c=sqlite3.connect('file:data/litepan.db?mode=ro',uri=True);[print(t, c.execute('SELECT COUNT(*) FROM '+t).fetchone()[0]) for t in ('upload_tasks','automation_rules','automation_runs')]"
bw open http://127.0.0.1:5211/admin?page=dashboard\&tab=overview --wait=3000
bw eval "JSON.stringify({hero:[...document.querySelectorAll('.hero-metric')].map(m=>m.innerText.replace(/\n/g,'=')),cards:[...document.querySelectorAll('.overview-card')].map(c=>c.innerText.replace(/\n/g,'=')),taskRows:document.querySelectorAll('.task-row').length})"

# 删除后的样式核查
grep -n "task-list\|task-row\|task-progress" web/src/components/admin/DashboardManagement.vue
grep -rn "dashboard-page .task" web/src/styles/
```
