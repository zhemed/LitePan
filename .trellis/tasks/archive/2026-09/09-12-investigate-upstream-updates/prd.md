# 09-12-investigate-upstream-updates

## Goal

只读调查上游 `Ponphil/LitePan` 自本方上次同步点以来（本地远程跟踪 ref `374affd` 2026-09-05 → 上游最新 `v0.5.5-beta`）的全部新增提交，逐条判定在本方**精简分支**（仅 3 驱动、无 STRM / 跨盘 / Emby / 分类 / 清理等）下的取舍（移植 / 跳过 / 观察），重点识别与本方 `0.0.32~0.0.37` 上传链路修复**重叠或冲突**的上游改动，产出 `research.md` 报告与后续任务候选清单。**本轮不实施任何代码改动。**

## Background

- 本方 `zhemed/LitePan` 是上游 `Ponphil/LitePan`（`git remote origin`）的**精简私有部署分支**（README：「基于 `Ponphil/LitePan` 精简：仅 3 驱动，无 `STRM` 等」）。
- 既有决策链（journal-2.md）：Session 67《调查上游合并安全性》结论 = **不整体 merge，改按需 cherry-pick**；Session 68 移植 4 组修复（`8e332f3` oauth 状态码/UA、`1c71fec` 连接检测、`c7a424c` 189 认证子集、`353b830` 上传目录错位）→ `0.0.13`；Session 74/75 移植 `internal/auth` 守卫接线 + `driverexec` 网络熔断 → `0.0.17`；`0.0.18` 移植 `de83b46 上传任务批次化`。
- 双方共同祖先 `4c160d9`（2026-08-29「部分显示优化」）；此后本方 `main` 独有 345 个提交，上游独有 20+ 个提交（相对本地已过期的 ref）。
- **本地 `origin/main` ref 停在 `374affd`（2026-09-05）**；GitHub API 显示上游已推进到 `46a0a89f`（2026-09-12T03:23:08Z，tag `v0.5.5-beta`），因此真实增量大于本地可见的 20 条，必须先 fetch。
- 本轮之前刚完成 6 项缺陷修复（`0.0.32~0.0.37`），**全部落在「上传任务批次化」派生链路**：`internal/upload/*`（`state.go`/`worker.go`/`breaker.go`/`lifecycle.go`）、`internal/core/driverexec`（账号冷却）、`internal/file/*`（冷却日志抑制）、`web/src/composables/upload/*`、`TaskPanel.vue`。上游同期存在 `收口5-上传任务`（2026-09-09）等提交，可能触及同一批文件 → 需重点对照。
- 上游版本号体系（`v0.5.x-beta`）与本方（`v0.0.x`）独立，不构成对本方版本规则的约束（本方规则：`0.0.1` 基线、`0.0.x` 递增、不跳 `1.0.0`）。

## Requirements

- **R1 只读原则**：不得修改上游仓库、不得 merge、不得 cherry-pick、不得改本方任何代码/配置/数据/镜像/前端构建；`git fetch origin` 仅允许更新 `.git` 内远程跟踪 ref（不产生工作区改动，任务结束时须有 `git status` 干净证据，除任务文档与 journal）。
- **R2 时效与基准**：以上游 `main` 的**最新**提交为基准（截至本任务执行日 2026-09-12），给出 `origin/main` 最新 sha/时间、上游 tag 与 release 状态，并说明本地 ref 与远端真实的差距（先 fetch 再比对）。
- **R3 全量清单**：列出「上次同步点 → 上游最新」的全部上游新增提交（sha 前缀、日期、标题、改动规模/涉及文件数），**不遗漏、不抽样**。
- **R4 四类分类**：逐条标注 —— (a) **本方已移植**（历史 cherry-pick 的 5 组 + 批次化）；(b) **与本方精简策略无关**（STRM / 跨盘 / Emby / 刮削 / 秒传 / 多驱动 / 纯 UI 主题等）；(c) **候选移植**（本方可直接受益的修复 / 稳定性 / 性能改进）；(d) **与本方 `0.0.32~0.0.37` 修复重叠或冲突**（同文件/同语义/修法差异）。给出每类的数量统计。
- **R5 重叠与冲突取证**：将本方 6 项修复链路与上游同路径改动做**并排比对**（可用 `git diff`、`git log -p` 逐条核对），明确：上游是否也修了同类问题、修法差异、若未来移植的冲突点与风险（例如上游改 `internal/upload/state.go` 冷却判定、`breaker.go` 熔断键、`TaskPanel.vue` 批次树渲染等）。
- **R6 结论与建议**：给出候选移植项的**优先级排序**（收益 / 风险 / 工作量）、最小验证方案（单测或只读核对点），并明确「本轮零实施」；如某项需要实施，定义后续独立任务的范围。
- **R7 外部来源交叉**：除 `git` / GitHub API 外，按全局检索约束做多路公开检索（上游发布动态、已知问题、社区反馈），关键论断需 ≥2 独立来源或在无法交叉时**如实标注未能独立验证**。
- **R8 记录**：产出 `research.md`（含执行命令、时间、关键输出摘要、提交清单表、对照证据、建议）；若发现新的上游契约知识（API/DB 字段/驱动契约变化），同步 `.trellis/spec` 或标注为后续任务输入。

## Constraints

- 沿既有决策：**不整体 merge**；上游改动的取舍一律以「本方精简后的模块边界」为准，已删除模块（STRM / 跨盘 / 8 驱动 / Emby / 分类 / 清理 / 海报）相关改动不做深挖，直接归为「不适配」。
- 不连接生产机 `10.0.0.11`（本轮无授权、无业务需要）；不触碰本地容器/数据。
- 不向上游提交 Issue/PR（除非用户后续明确要求）。
- 不臆测：无法从提交内容判定的条目，标注「需进一步核对」并给出核对方法，不写成结论。
- 本任务只做**调查**；任何修复/移植须另建任务并完整走 Trellis 九步流程。

## Acceptance Criteria

- [x] 已 fetch 上游最新并给出基准（`origin/main` 最新 sha/时间 + tag/release 状态 + 本地 ref 与远端差距说明）
      → research.md §3.1：`46a0a89`（09-12 11:23）、tag `v0.5.5-beta`；fetch 报 **forced update**、历史已重写（无共同祖先）
- [x] 已产出「上次同步点 → 上游最新」**全量**新增提交清单（数量 + sha + 日期 + 标题 + 改动规模）
      → 共 **47** 个提交（`f71a522..origin/main`），逐条列于 research.md §3.2
- [x] 已按 (a) 已移植 /(b) 无关 /(c) 候选 /(d) 重叠冲突 四类逐条分类并给出数量统计
      → research.md §3.3：5 / 20 / 9 / 1（+12 低价值可选，合计 47）
- [x] 已完成本方 `0.0.32~0.0.37` 六项修复与上游相关提交的**并排对照**（同文件/同语义/修法差异/移植风险）
      → research.md §5.1（六项修复上游状态逐条取证）+ §5.2（`5255775` 收口5 逐行判定为纯重构）
- [x] 已给出候选移植项优先级排序（收益/风险/工作量）+ 最小验证方案 + 后续任务建议；明确本轮零实施
      → research.md §6（P1~P9）+ §9；§7 列明不实施项
- [x] `research.md` 完成（含命令、证据、时间线），且本轮**零代码改动**（`git status` 干净，仅任务文档/journal 变更）
      → research.md §2/§10；`git status --porcelain` 仅 `.trellis/tasks/09-12-investigate-upstream-updates/`，`git diff --stat HEAD` 为空
- [x] 外部来源交叉检索已完成并如实标注可信度（含「未能独立验证」项）
      → research.md §10（8 路 tavily + 官方 changelog）与 §8（5 项未验证/限制）

## Notes

- 沿用同类调查任务（`09-10-investigate-11-cooldown-storm`）的产物约定：`prd.md` + `research.md`，`scope=lightweight`（本任务无代码产出）。
- **执行中的重要发现（超出初始预期，已写入 research.md §1/§3.1/§9）**：上游在 2026-09-05 之后 **force-push 重写了历史**，双方已无共同祖先，`main..origin/main` 类提交差集不再可用；同步口径自此改为「内容级 tree 对照移植」。该发现使 R3 的清单口径以「自 fork 点（`f71a522`）以来的 47 提交」为准（原表述「上次同步点」在重写后无法按提交差集定义，属证据驱动的口径校准，非放宽要求）。
- **本轮零实施**：未 merge/cherry-pick/改码；唯一仓库级操作是 `git fetch origin`（仅更新远程跟踪 ref，不进工作区）。`git status` 证据见上。
- **Spec 同步判定**：本轮无 API/DB/驱动契约变更，`.trellis/spec` 无需更新；「上游同步口径」知识点已写入 research.md §2/§9 并列为后续任务输入（§9.5），是否固化为 `.trellis/spec` 待用户确认，避免超出本轮调查范围。
- 版本影响提示：若本轮判定需要移植，按规则另建任务并在该任务内 bump `0.0.x`（`0.0.1` 基线、不跳 `1.0.0`）。
- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.
