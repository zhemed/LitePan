# 把浏览器验收判据写进前端 spec（填补零自动化测试的验证空白）

## Goal

前端**没有任何自动化测试**，而 `spec/web/frontend/quality-guidelines.md:74` 又白纸黑字要求手工 QA —— 这条要求在归档任务中的**执行次数为 0**。结果是：**"渲染之后才存在的行为"长期没有验证手段**，只有"用浏览器看"与"什么都不做"两种可能，实际一直落在后者。

本机现已具备常驻无头浏览器（`bw`，systemd `browser-cdp.service`）。本任务把"**何时必须做浏览器验收、怎么跑、边界在哪**"写成**判据**固化进 spec，并把本轮实测踩到的两个坑一并沉淀，使这条要求从"写了但没人执行"变为"可执行、有触发条件、有明确边界"。

## Background（2026-09-12 取证）

| 证据 | 实测结果 |
|---|---|
| 前端自动化测试 | `package.json` **无 `test` 脚本**；devDependencies **无 vitest/jest**；`web/src/__tests__` **不存在** |
| spec 现行要求 | `quality-guidelines.md:73-74`：`No vitest yet…` / `Manual QA: npm run dev → login → browse / → admin tabs → file preview (pdf/docx/video) → check console for TypeError` |
| 该要求的执行史 | 归档任务中 `手工 QA`/`人工 QA`/`manual QA`/`手工验收` grep **零命中** —— 从未被执行并记录 |
| 缺口的实际暴露 | 仅 2 个任务显式写下"UI 未验证"（`09-12-version-single-source-release-0-0-44`、`09-12-deploy-litepan-local-5211`，均为本会话产物） |
| `bw` 可用性 | `bw doctor` 全绿（Chromium 153 / CDP 9222 / systemd active）；`open/text/els/fill/click/eval/links/tabs/key/shot/gui` 12 项子能力实测通过；登录态跨命令保持 |
| 现有前端门 | `npm run type-check`（`vue-tsc`，仅类型）+ `npm run build`（能否打包）—— **均无法覆盖运行期行为** |

**已验证的正面样例**：`v0.0.44` 的界面版本号此前只能靠"构建产物不含版本字面量 + API 返回正确"**间接推断**；本轮用 `bw` 登录后目视确认 footer 实际渲染 `当前版本: LitePan v0.0.44`。这正是本判据要覆盖的典型场景。

## Requirements

### D1 在既有 spec 中扩写「Testing」段

文件：`.trellis/spec/web/frontend/quality-guidelines.md`，在 `## Testing` 段内（`No vitest yet` 之后、原 Manual QA 清单之前）新增内容，须包含：

1. **问题陈述**：前端零自动化测试 ⇒ 渲染后行为的验证只有"浏览器"或"什么都没做"
2. **触发判据表**（何时**必须**做浏览器验收）：逐条写清"为什么 type-check 管不到"
   - 异步/运行期取值的展示（如版本号经 store 异步填入 footer）
   - 条件渲染与状态机（上传任务终态桶展开、暂停徽标、冷却倒计时等）
   - store ↔ 组件接线、跨组件共享状态、watch/响应式
   - 路由守卫分支（`public_index_enabled`、`must_change_password` 等）
   - 发布/部署收尾（确认"用户最终看到的那一面"）
3. **明确不需要浏览器的场景**：纯后端/驱动/存储改动、纯文档与配置改动、仅改文案或 CSS 类名、`type-check` 即可发现的类型错误
4. **怎么跑**：给出可直接复制的 `bw` 命令序列（`open/els/fill/click/text/shot`），并指向全局 `AGENTS.md` 的 BROWSER-CDP 段
5. **保留原 Manual QA 清单**：`npm run dev`（或已部署实例）→ login → browse `/` → admin tabs → file preview → **check console for `TypeError`**

### D2 边界必须写死（防止被误用）

- **它是验收手段，不是测试**：不替代也不减少 `npm run type-check`、`npm run build`、`make lint`、`go test`
- **不得进 CI、不得作为门禁的自动判定项**：选择器随 UI 改动失效、依赖应用正在运行，属非确定性工具
- **结论必须带证据来源**：写"截图目视确认"/"渲染后 DOM 文本为 X"，**不得**写成"自动化断言通过"

### D3 沉淀本轮实测的两个坑

- **`bw click <文本>` 会点错目标**：实测点击「登录」命中了页面标题「管理员**登录**」而非 `<button>`，表单未提交 → **按钮一律用 CSS 选择器**（如 `.submit-btn`），文本匹配只留给唯一目标的链接
- **Vue 表单输入**：`bw fill` 能写入值；若出现"填了值但提交为空"，用 `bw eval` 补发 `input`/`change` 事件后再提交。**须如实标注**：本轮未能单独隔离验证"补发事件是否必需"（点击失败在先），此处记为**排障顺序**而非确定结论

### D4 验收记录约定

- 浏览器验收结果写进任务 `prd.md` 的检查记录
- 截图写到 `/tmp` 并在验收后清理，**不污染仓库**（避免在 git 工作区留下未跟踪 PNG）

## Constraints

- **只改 `.trellis/spec/web/frontend/quality-guidelines.md` 一个文件**；不改 `spec/web/frontend/` 其它文件、不改 `spec/backend/**`、不改全局 `AGENTS.md`
- **不改任何代码**：不动 `web/src/**`、`internal/**`、compose、README
- **不改强制门本身**：`vue-tsc` + `vite build` 仍是前端主门，本任务只**新增可选验收判据**，不改变既有门槛
- **不虚构能力**：只写本轮实测通过的 `bw` 用法；未验证的能力（如 `--full` 之外的高级选项、并发多会话行为）不写入
- 不改运行中的容器与服务；不触碰 `data/litepan.db`
- 不修改归档任务与历史 journal

## Acceptance Criteria

- [x] `quality-guidelines.md` 的 `## Testing` 段含**触发判据表**，且每条写明"为什么 `type-check` 管不到" —— 实测表格 6 行（表头 + 5 个触发场景），表头即 `Why \`type-check\` cannot cover it`
- [x] 同段含**"不需要浏览器"的反向清单** —— 实测 `**Not required** for:` 一行覆盖四类：backend/driver/store、docs-/config-only、copy 或 CSS-class-only、type-check 已报的类型错误
- [x] 同段含**可直接复制的 `bw` 命令序列** —— 实测 6 条命令（`open`/`els`/`fill`/`click`/`text`/`shot`），含 `--into='input[placeholder="请输入用户名"]'` 与 CSS 选择器 `.submit-btn`
- [x] 原 **Manual QA 清单内容仍在** —— 实测第 133–134 行仍为 `npm run dev` → login → browse `/` → admin tabs → file preview (pdf/docx/video) → **check console for `TypeError`**
- [x] **边界三句写死** —— 实测三项逐一命中：`neither replaces nor reduces`（不替代/不减少既有门）、`Never put it in CI`（不进 CI/不作自动判定）、`never "automated assertion passed"`（结论带证据不带断言）
- [x] **两个实测坑已记录** —— 实测：① 误点现象写明"`登录` 命中了页面标题 `管理员登录` 而非 `<button>`，表单未提交"；② Vue 表单补发 `input`/`change`，并如实标注 `Not independently isolated`（点击失败在先，属排障顺序而非已证结论）
- [x] 含**截图写到 `/tmp` 并清理**的记录约定 —— 实测第 128 行 `Write screenshots to \`/tmp\` and delete them afterwards so the git worktree stays clean`
- [x] `git diff --stat` 仅 1 个受跟踪文件 —— 实测 `1 file changed, 61 insertions(+), 1 deletion(-)`，仅 `quality-guidelines.md`
- [x] 运行中实例未受影响 —— 实测 `litepan | Up About an hour`，`/api/health` HTTP 200
- [x] `git status --short` 仅该文件 + 本任务目录 —— 实测无越界改动

## Notes

- Scope 标注 `lightweight`：单文件、纯文档判据沉淀，无技术方案分歧（判据已在前序对话中与用户逐条确认），故 PRD-only。
- 本任务的来源：用户在评估"是否有必要调用浏览器验收"后确认采纳我的建议 —— **不进强制门**，而是写成**条件触发的可选验收判据**。
- 本任务**不**做：引入 vitest 或任何前端测试框架、修改 CI/门禁脚本、把工作区已有的 2 条"UI 未验证"记录回填到归档任务（历史不改写）。

## 检查记录（trellis-check，2026-09-12）

**Step 1 变更识别**：单文件 `.trellis/spec/web/frontend/quality-guidelines.md`，**+61 / −1**（104 → 164 行），新增 4 个小节（触发判据 / 怎么跑 / 边界 / 完整人工 QA）。零代码改动。

**Step 2 规范对齐**：本次改的正是 spec 自身。写入前已通读该文件全文（104 行），据此决定：① 沿用本文件的**英文**（该文件 100% 英文，与 `api-layering.md` 的中英混排不同），仅引用中文 UI 串原文以保证可复现；② 保留原有 `## Testing` 的两条既有内容（`No vitest yet` 与 Manual QA 清单），只在其间扩写，**不删除既有要求**。

**Step 3 项目质量门**：代码质量门 **N/A** —— 纯 spec 文档，未触及工程代码，不产生可测行为。本任务的"质量门"是**写入内容与本轮实测事实一致**：命令序列逐条取自实测通过的命令，两个坑均附实测现象，未验证的部分显式标注为"未隔离验证"而非当作结论。

**Step 4 清单核对**

- 测试覆盖：**N/A**（无可测行为）。
- Spec 同步：**本次即 spec 同步本身** —— 把"何时必须做浏览器验收"从口头共识固化为可查判据。
- 范围纪律：单文件；未改强制门（`vue-tsc` + `vite build` 仍是前端主门）；未改全局 `AGENTS.md`；未引入任何测试框架。
- 跨层一致性：N/A。

**诚实记录**

1. **只写实测通过的能力**：`bw` 的命令序列取自本轮实际跑通的 6 条（含 `--into=` 与 CSS 选择器点击）；`bw doctor` 列出的 `tesseract OCR`、`--full` 之外的选项、并发多会话行为等**未实测项一律未写入**，避免把未验证能力写成规范。
2. **第二条坑如实降级**：Vue 表单补发事件这一条，我**无法隔离验证**它是否必需（实际失败原因是点击命中了标题，先于该步）。故文中写为"troubleshooting order, not a proven requirement"，而不是断言 `bw fill` 对 Vue 无效。
3. **未回填历史**：`09-12-version-single-source-release-0-0-44` 与 `09-12-deploy-litepan-local-5211` 里那两条"UI 未验证"记录**不改写** —— 归档是当时状态的历史证据；且前者本轮已在浏览器中实际确认过（footer 渲染 `LitePan v0.0.44`），事实已在对话与 journal 中留痕。
4. **未改变任何既有门槛**：本任务只新增"条件触发的可选验收判据"，前端主门与 Go 侧门一律未动 —— 这是与用户逐条确认后的范围。
