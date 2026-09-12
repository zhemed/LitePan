# 归档死代码 P3/P4 处置：四项均保留（不实施），附重启条件

## Goal

按用户指示，把死代码复扫（`09-12-rescan-dead-code`）遗留的 **P3/P4 四项**处置决定**归档为"保留、不实施"**，并逐项给出「**当前状态核实 + 为何可先不做 + 何时值得重启**」，作为死代码候选清单的最终收尾。

**本轮零代码改动** —— 交付物就是这份决定记录，防止未来会话重新翻案或误判。

## Background

`09-12-rescan-dead-code` 的可清理项分两档：

- **P1/P2（≈182 行 + 1 依赖）** → 已由 `09-12-cleanup-rescan-findings` 全部清理完毕（`deadcode` 9→7、`unused` 14→0、`deps` 24→23）
- **P3/P4（4 项）** → 当时标注"待定 / 需决策"，本轮逐项查清并归档结论

**关键前提**：复扫报告用的措辞是"疑似未使用/待定"，但**经代码级核实，这四项没有一项是真正的"死代码"** —— 它们分别是*可达但未调用*、*活着但名字过时*、以及*为流程与测试而有意保留*。这正是复扫报告当初只把 P1/P2 算作"可安全清理"的原因。

## Requirements

### D1 逐项记录处置与依据（本 PRD 即交付物）

对四项各写清：**当前状态核实（附实测证据）→ 为何可先不做 → 重启条件**。

### D2 明确"保留"的性质判定（区别于死代码）

须写明每项的**严格性质**，避免未来把它当作"遗漏的死代码"再次清理。

## 处置记录

### P3-1 `/api/admin/accounts/{id}/refresh-auth` —— 孤儿端点，**性质：可达但未被调用**

**当前状态核实**

| 项 | 实测 |
|---|---|
| 注册位置 | `internal/api/router.go:186`，位于 **`requireAdmin`** 分组内 |
| Handler | `internal/api/auth_refresh.go`（30 行）→ `h.auth.Refresh(ctx, id, driver.CallerPassive)` |
| 后端能力 | **活**：同一 `Refresh` 也被 `internal/auth/gate.go:43`（统一守卫）与 `scheduler.go:360`（主动调度器）调用 |
| 调用方 | **无** —— 前端 `web/src/api/accounts.ts` 有 `toggle`/`set-default`/`refresh-profile`，**没有** refresh-auth |
| 邻居对照 | 左右两个兄弟端点 `set-default`、`refresh-profile` **前端都在用** |

**为何可先不做**

它不是死代码：路由已注册、handler 可达，任何持有管理员会话者都能调用。它是**未被调用的运维钩子** —— 语义上等于"手动触发一次统一守卫本来会自动做的事"。删除它属于**收窄 API 表面的产品决策**，且对任何外部脚本都是 **breaking change**；而保留成本仅 30 行 + 1 行路由。

**重启条件**：确定不再需要"手工刷新单个账号认证"这一排障手段，**且**愿意承担对外 breaking（或先提供替代入口）。

### P3-2a `OfflineHandoffClientID`（3 行）—— **性质：生产死 / 测试可达**

**当前状态核实**

- 生成 `offline-handoff:<group>:<index>` 形式的 client task ID
- 前缀常量 `offlineHandoffClientPrefix` **只在生成函数内部使用**（`manager.go:568/571`）→ 生产侧**既不生成也不解析**该格式 ID
- 调用方**仅** `internal/upload/manager_test.go` 的 3 处

**为何可先不做**

可删，但**收益极小**：省 3 行，代价是要改 3 处测试调用（改成字面量）。删除它不减少任何有效覆盖（测试验证的是通用 `ClientTaskID` 去重语义，与 ID 由谁生成无关），但改动面大于收益。

**重启条件**：下次批量清理 `internal/upload/manager.go`、或该测试文件重构时**顺带内联**，不值得单独立项。

### P3-2b `SourceTypeOfflineHandoff` —— **性质：生产活（必须保留）**

**当前状态核实**

```go
// manager.go:477  —— 它是服务器上传的【默认】来源类型
if sourceType == "" { sourceType = SourceTypeOfflineHandoff }
// manager.go:479  —— 校验只允许 offline_handoff 或 server_local
// manager.go:405  —— 决定任务文案「等待离线文件上传」
```

**为何可先不做**

**它不是死代码**，而且被用作默认值。`offline_handoff` 这个名字来自**已删除的"离线下载"功能**，属**命名漂移**而非死代码；改名要动 DB 存量记录里的来源类型值，风险远大于收益。**任何清理动作都不得把它与同文件的 `OfflineHandoffClientID` 一并删除** —— 二者名字同源、命运相反。

**重启条件**：出现"来源类型语义混乱导致的**实际缺陷**"时，作为**独立重构**（含数据迁移与兼容期）评估，而非死代码清理。

### P4 `drivers/template`（438 行）+ httpx A 簇五符号（三个文件 115 行）—— **性质：有意脚手架 + 测试载体**

**当前状态核实**

| 项 | 实测 |
|---|---|
| 规模 | `drivers/template/` 6 文件 **438 行**；A 簇所在 `internal/httpx/{oauth.go,do_json.go,envelope.go}` 共 **115 行** |
| 引用者 | **仅** `internal/auth/oauth_integration_test.go:14`（空导入 `_`），生产侧零 import |
| 包自述 | "新驱动脚手架：复制本目录为 `drivers/<名>/` …**勿注册本包**" —— 且实测 `registry.go` 确无 template 条目 |
| spec 依赖 | `spec/backend/backend/driver-development.md:71` 明确要求 **`cp -r drivers/template drivers/FooCloud`** 作为驱动创建第一步 |
| 测试依赖 | 该空导入支撑 `TestOAuthDriversUseUnifiedGuard`（`internal/auth` 5 个测试文件中唯一的 OAuth 守卫测试） |

**为何可先不做**（三条硬的）

1. **它是本项目 spec 记载的驱动创建流程载体** —— 删 template 会让 `driver-development.md` 的流程失效，须连 spec 一起改
2. **它是"统一 OAuth 守卫"的唯一测试载体** —— 真实 OAuth 驱动（123/百度/OneDrive）已在精简版移除，测试注释明载"用同走标准代理信封与统一分类的 template 骨架驱动验证统一守卫"。删 template = `TestOAuthDriversUseUnifiedGuard` 失去替身 = **静默丢掉该安全相关路径的唯一测试覆盖**
3. **A 簇与它是同一条链**（A 簇五符号的生产侧使用者只有 template），故命运绑定；但链的另一端挂着上述测试价值

**重启条件**：**先补一个不依赖 template 的 OAuth 守卫测试**（恢复覆盖），**再**评估整链删除；或明确"不再新增驱动"且接受该守卫覆盖下降。

## Constraints

- **零代码改动**：本任务只写决定记录；`git status` 除任务目录外为空
- **不改任何被讨论的代码**：`refresh-auth` 路由、`OfflineHandoffClientID`、`SourceTypeOfflineHandoff`、`drivers/template`、httpx A 簇一律不动
- **不改 spec**：四项均保留，无规范变更
- 不修改归档任务与历史 journal（`09-12-rescan-dead-code` 的"待定"标注是当时状态的历史证据）

## Acceptance Criteria

- [x] 四项各自的「当前状态核实 + 为何可先不做 + 何时值得重启」**均已记录** —— 见上方「处置记录」四节，每节均含三要素（P3-1 / P3-2a / P3-2b / P4）
- [x] 每项均标注**严格性质**，与"死代码"区分 —— P3-1「可达但未被调用」、P3-2a「生产死 / 测试可达」、P3-2b「生产活（必须保留）」、P4「有意脚手架 + 测试载体」
- [x] 关键判据**附可复核证据**（文件:行号、grep 结论、调用方计数）—— 共引用 7 处行号，本轮**逐一抽查全部命中**：`router.go:186`（refresh-auth）、`gate.go:43` 与 `scheduler.go:360`（`Refresh` 活调用）、`manager.go:477`（默认来源类型）、`manager.go:568`（前缀常量仅内部使用）、`driver-development.md:71`（`cp -r drivers/template`）、`oauth_integration_test.go:14`（唯一空导入）
- [x] **零代码改动** —— 实测 `git status --short` 仅 `?? .trellis/tasks/09-12-close-deadcode-p3-p4/`；`git diff` **0 行**；受跟踪文件改动 **0**
- [x] **基线不变** —— 实测 `deadcode ./cmd/litepan` **仍为 7**、`golangci-lint --enable=unused` **仍为 0**（证明本轮确实没动代码）
- [x] 运行中实例未受影响 —— 实测 `/api/health` HTTP 200、容器 `Up`
- [x] 与复扫任务衔接说明：P1/P2 已清理完毕，本任务收尾 P3/P4 → **死代码候选清单全部关闭** —— 已在 Notes 写明，并把 `deadcode` 剩余 7 的构成（A 簇 5 + C 簇 1 + 保留项 1）与去留**一一对应**，确认无遗漏项

## Notes

- Scope 标注 `lightweight`：纯决定归档、零代码改动，与先例 `09-12-close-p7-p8-p9-candidates`（同为轻量、同为"候选清单收尾"）一致，故 PRD-only。
- 本任务**不**做：任何删除、新增测试、改 spec、处理 `offline_handoff` 命名漂移。
- 候选清单最终状态：**P1/P2 已清理（182 行 + 1 依赖）**；**P3/P4 四项归档保留**。`deadcode` 剩余 **7** 的构成与去留一一对应：A 簇 **5**（随 P4 保留）+ C 簇 **1**（`OfflineHandoffClientID`，P3-2a 归档保留）+ 保留项 **1**（`FlexibleString.UnmarshalJSON`，实现 `json.Unmarshaler` 由反射调用，必须留）。**7 = 5 + 1 + 1，全部有明确处置，无遗漏项。**

## 检查记录（trellis-check，2026-09-12）

**Step 1 变更识别**：**零代码改动** —— `git diff` 为 0 行、受跟踪文件改动 0；本轮唯一产出是本任务目录下的 `prd.md`（决定记录）。

**Step 3 项目质量门**：代码质量门 **N/A**（未改任何代码）。本任务的"质量门"是**引用准确性**：PRD 中引用的 **7 处文件:行号已逐一抽查，全部命中**（见 Acceptance 第 3 条）。另复测 `deadcode` = 7、`unused` = 0 作为"确实没动代码"的**反向证据**。

**Step 4 清单核对**

- 测试覆盖：**N/A**（零代码改动，无可测行为）。
- Spec 同步：**无需** —— 四项均保留，无规范变更；P4 的保留恰恰是**为了维持** `driver-development.md` 既有流程的有效性。
- 范围纪律：未改任何被讨论的代码、未新增测试、未改 spec；零代码改动已实测确认。
- 跨层一致性：N/A。

**诚实记录**

1. **本任务纠正了复扫报告的一个措辞陷阱**：报告把这四项与 P1/P2 并列在同一张"清理建议"表里（标注为 P3/P4、风险低/中），容易让人读成"剩下这些也是死代码，只是优先级低"。**实际经核实，四项没有一项是真正的死代码** —— 分别是*可达但未被调用*、*生产活*、*为流程与测试有意保留*。把"性质判定"单独列出（Acceptance 第 2 条）就是为了挡住这个误读。
2. **我自己在初稿里算错过一笔账**：写"剩余 7 的构成"时漏了 C 簇，写成 5 + 1 = 6。已自查修正为 **5（A 簇）+ 1（C 簇）+ 1（保留项）= 7**，并在 Notes 中逐项对应到处置结论，确保**没有一项是"没人管"的**。
3. **未做但已给出路径**：若日后真要清理 P4，正确顺序是**先补不依赖 template 的 OAuth 守卫测试、再删链** —— 这条已写入 P4 的重启条件，避免未来把它当纯清理任务执行而静默降覆盖。
4. **与先例一致**：本任务的形式与 `09-12-close-p7-p8-p9-candidates`（上游候选清单收尾）完全对齐 —— 零代码改动、逐项「现状核实 + 为何不做 + 重启条件」、作为清单最终收尾。
