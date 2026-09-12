# 清理死代码 P3：删除复扫确认的 182 行 + 1 依赖，并把方法论写入 spec

## Goal

执行 `09-12-rescan-dead-code`（复扫报告）给出的 **P1 + P2** 清理建议，并把本轮最有价值的方法论固化进 spec。

**范围就是复扫报告 §9 表中标为"可安全清理"的 P1/P2 项**（≈182 行 + 1 依赖）；**不含** P3 的两项（孤儿端点 `/accounts/{id}/refresh-auth`、`OfflineHandoffClientID`）与 P4（`drivers/template` 链路）—— 那些属功能决策，用户未授权。

## Background（复扫报告的可执行结论）

| 优先级 | 项 | 规模 | 依据（复扫已实证） |
|---|---|---|---|
| **P1** | `internal/settings/service_test.go` **整文件** | 28 行 | 全文只有一个 **从未被实例化** 的 mock（`memoryConfigRepo`），**无任何 `func Test*`** → 删除不损失任何测试覆盖 |
| **P1** | `internal/domain/pause_reason.go` **整文件** | 17 行 | `grep -rn "PauseReason"` 排除本文件后**为空** —— 类型 + 3 常量 + 2 函数在**含测试**的全仓库零引用 |
| **P2** | `web/src/components/base/TimeWindowField.vue` | 122 行 | 茎名引用计数 0；**构建产物交叉验证 0 命中**（对照 `TimeWheelPicker` 命中 1）→ 确未进 bundle |
| **P2** | `@fontsource-variable/noto-serif-sc` | `package.json` 1 行 + lock | `web/src`/`index.html`/`vite.config.ts` 零引用；CSS 无 `Noto Serif SC` 字体族；产物无 noto 字体 |
| **P2** | `internal/automation/service_test.go` 的 10 个符号 | ~15 行 | `unused` 报 14 条中属本文件的 10 条（`type apiKeyRepo` + 8 methods + `automationRunRepo.count`），均**从未被实例化**；该文件仍有在用的测试 |

**同批要一并沉淀的方法论**（复扫报告 §6.2 与 §8）：`deadcode`（生产可达性）与 `golangci-lint --enable=unused`（包内未使用）**视角互补、结果零重叠、必须并用**；且**涉及接口的方法下结论前必须先核查"是否被实例化"**——本轮与上一版都因跳过这一步而误判过同一批 mock。

## Requirements

### D1 删除 Go 侧两个死文件

- 删 `internal/settings/service_test.go`（整文件；**该文件不含任何测试函数**，删除不损失覆盖率。注：删后 `settings` 包将**没有测试文件**，这是现状的真实反映 —— 它本来就没有测试）
- 删 `internal/domain/pause_reason.go`（整文件）

### D2 删除前端死文件与未使用依赖

- 删 `web/src/components/base/TimeWindowField.vue`
- **一并删** `web/src/composables/useTimeWindowSchedule.ts`（**规划期修订**，见下方 Notes：原判"仍被引用、不可误删"的前提已在实施中被否证）
- 从 `web/package.json` 移除 `@fontsource-variable/noto-serif-sc`，并更新 `package-lock.json`
- **必须保留** `@vue/devtools-api`（复扫已排除的误报：它是 `pinia` 的 peerDependency + `vue-router` 的 dependency）
- 重建 embed：`cd web && npm run build`，并**验证 `internal/api/web/**` 零 churn**（反证被删文件确实不在 bundle 中）

### D3 删除测试层死符号（10 个）

- `internal/automation/service_test.go`：删 `type apiKeyRepo` 及其 8 个方法（`List`/`Get`/`GetByHash`/`Count`/`Create`/`Update`/`Delete`/`TouchLastUsed`）+ `automationRunRepo.count`
- **必须保留** `automationRunRepo` 的其它方法与类型本身（复扫确认它**被实例化 4 处**，是活桩）
- 删除后该文件**其余测试必须仍能编译并通过**

### D4 把方法论写入 spec

- 新增 `.trellis/spec/guides/dead-code-guide.md`，内容须含：
  1. **两种视角必须并用**：`deadcode ./cmd/litepan`（生产可达性）vs `golangci-lint run --enable=unused`（包内未使用）；给出两者**零重叠**的实例证据
  2. **下结论前的两条硬核查**：① 涉及接口的方法 → 先查"是否被实例化/赋给接口变量"（`grep` 实例化处）；② 报出的类型/常量 → `deadcode` **只报函数**，故类型与常量需人工补查
  3. **`unused` 不进强制门**：本仓库仍以命令行 ad-hoc 方式使用（**不改 `.golangci.yml`**），理由与前端浏览器验收同源——它是排查工具，不是门禁
  4. 完整**复现命令**
- 在 `.trellis/spec/guides/index.md` 的 "Available Guides" 表登记该指南

## Constraints

- **只删复扫确认的项**：不得顺手删 P3/P4 任何内容（孤儿端点、`OfflineHandoffClientID`、`SourceTypeOfflineHandoff`、`drivers/template` 链路）
- **不做版本 bump、不发版**：本次是**纯删除不可达代码，无任何行为变化**；`README.md` / `docker-compose*.yml` 的镜像 tag **保持 `v0.0.44`**，与线上正在运行的实例一致。若日后要发版，另开任务走完整发版流程（该偏离已在检查记录中留痕）
- **不改 `.golangci.yml`**：`unused` 仍为 ad-hoc 排查工具，不进强制门
- **不改运行中的实例**（`litepan` 容器、`data/litepan.db`）与 DSH
- **不得删除任何仍被引用的东西**：`useTimeWindowSchedule.ts`、`@vue/devtools-api`、`automationRunRepo` 其余方法
- 质量门全绿后才归档：`make lint`、`GOWORK=off go vet ./...`、`GOWORK=off go test ./...`、`cd web && npm run type-check`、`npm run build`
- 不修改归档任务与历史 journal

## Acceptance Criteria

- [x] `internal/settings/service_test.go` 与 `internal/domain/pause_reason.go` **已不存在** —— 实测 `git status` 显示两者为 `D`；`go build` 通过即证无调用方
- [x] `web/src/components/base/TimeWindowField.vue` 与 `web/src/composables/useTimeWindowSchedule.ts` **均已不存在** —— 实测两者均已 `git rm`（后者见规划期修订 1）
- [x] `web/package.json` 不再含 `@fontsource-variable/noto-serif-sc`；`package-lock.json` 已同步；`@vue/devtools-api` **仍在** —— 实测 `npm install` 输出 `removed 1 package`；`devtools-api` 在 package.json 与 lock 中均仍在
- [x] `internal/automation/service_test.go` 不再含 `apiKeyRepo` 与 `automationRunRepo.count`；`automationRunRepo` 类型与其它方法**仍在** —— 实测该文件 318 → 297 行（−21）；`go test ./internal/automation/` **通过**（证明其余测试未破）
- [x] **`golangci-lint run --enable=unused` 从 14 降至 0** —— 实测 **14 → 0** ✅
- [x] **`deadcode ./cmd/litepan` 从 9 降至 7** —— 实测 **9 → 7** ✅（B 簇 2 条消除；A 簇 5 条 + 保留项 1 条如期仍报）
- [x] 前端零引用文件从 1 降至 **0**；`dependencies` 从 24 降至 **23** —— 实测（**剥注释后**重算）零引用 = **0**；deps = **23** ✅
- [x] **embed 零 churn**：`npm run build` 后 `git status --short internal/api/web` 为空 —— 实测两次构建后均为 **0 个变更文件**（反证两个被删前端文件都不在 bundle）
- [x] 质量门全绿：`make lint` 0 issues、`go vet` exit=0、`go test ./...` 27 包 ok、`vue-tsc -b` exit=0、`npm run build` 成功 —— 全部实测通过
- [x] `spec/guides/dead-code-guide.md` 已新增且含四要素；`guides/index.md` 已登记 —— 实测 **126 行 / 8 个小节**，含"两工具零重叠""接口方法先查实例化""引用计数须剥注释""构建产物零 churn""迭代到不动点""unused 不进强制门"+ 复现命令 + 删除前清单；index.md:28 已登记
- [x] **未越界**：P3/P4 涉及的文件与符号**零改动** —— 实测 `git status` 中无 `manager.go`/`types.go`/`router.go`/`drivers/template`/`version.go`/`README`/`docker-compose` 任何条目
- [x] 运行中实例未受影响 —— 实测 `litepan | Up 2 hours`、`/api/health` HTTP 200
- [x] **附加**：按"迭代到不动点"再扫一轮，**未发现连带孤儿**（剥注释后零引用仍为 0）

## Notes

- Scope 标注 `multi-deliverable`（6 个可独立验证的交付物：2 个 Go 文件、1 个前端文件、1 个依赖、10 个测试符号、1 份 spec 指南），故按复杂任务补齐 `design.md` + `implement.md` 后再 `start`。
- 本任务**不**做：版本 bump、发版、P3/P4 决策项、引入 vitest、把 `unused` 加进强制门、清理上一轮复扫报告的其它"待定"项。
- 与复扫任务的衔接：`09-12-rescan-dead-code` 的验收档案已把可清理项与待决策项分开列明，本任务**只取其"可安全清理"部分**。

## 检查记录（trellis-check，2026-09-12）

**Step 1 变更识别**：6 个受跟踪文件 —— 删 4（`internal/settings/service_test.go` 28 行、`internal/domain/pause_reason.go` 17 行、`web/src/components/base/TimeWindowField.vue` 122 行、`web/src/composables/useTimeWindowSchedule.ts`）+ 改 3（`internal/automation/service_test.go` −21 行、`web/package.json`、`web/package-lock.json`）+ 新增 2（spec 指南与索引行）。**`internal/api/web` 零 churn**。

**Step 3 项目质量门（全部实测）**

| 检查 | 结果 |
|---|---|
| `make lint` | `0 issues.` |
| `GOWORK=off go vet ./...` | exit=0 |
| `GOWORK=off go test ./...` | **27 包 ok**，无失败 |
| `cd web && npm run type-check` | `vue-tsc -b` exit=0 |
| `cd web && npm run build` | 成功；**embed 零 churn** |
| 四项数量终测 | `deadcode` 9→**7**、`unused` 14→**0**、`deps` 24→**23**、churn 0→**0**（**全部与预期精确吻合**） |

**Step 4 清单核对**

- 测试覆盖：删掉的是**测试层的死代码**（未实例化的 mock、无测试函数的文件），**不减少任何有效覆盖**；`settings` 包本就没有测试，`automation` 包删符号后其余测试仍通过（`go test ./internal/automation/` ok）。本任务未新增功能，故无需新增单测。
- **Spec 同步：已执行** —— 新增 `guides/dead-code-guide.md`（126 行 / 6 条规则 + 命令 + 删除前清单），把本轮与上一轮的**全部实证教训**固化：两工具零重叠、接口方法先查实例化、`deadcode` 只报函数、**引用计数须剥注释**、构建产物零 churn 反证、迭代到不动点、`unused` 不进强制门。
- 范围纪律：**未越界**（P3/P4 与版本文件零改动，已 grep 验证）；未做版本 bump；未改 `.golangci.yml`。
- 跨层一致性：N/A（纯删除，无数据流变更）。但**构建产物零 churn** 本身就是一次跨层验证（源码层删除 → 构建层产物不变 → 证明无运行期引用）。

**诚实记录**

1. **规划期修订 1（实施中发现，先改 PRD 再动手）**：复扫报告称 `useTimeWindowSchedule.ts`"仍被引用、不可误删"，本 PRD 据此把它列入保留清单。删掉 `TimeWindowField.vue` 后零引用复算立刻报出它 —— 严格核查确认**它本来就是死的**（8 个导出零消费方，唯一的"引用"是已删文件里的**两句注释**）。**根因是复扫那套茎名文本计数把注释算作引用**（假阴性）。已按 Plan↔Execute 回退修正 PRD 后一并删除，并把"引用计数须剥注释"写进 spec 指南。
2. **我第一次的 G4 自检命令写错了**：用了 `grep -qE "...\|..."`，而 `-E` 模式下 `\|` 是字面竖线，导致误报"要素缺失"。修正模式后复核通过 —— 是**检查命令**的问题，不是文件的问题。
3. **不跑 `gofmt -w`**：`internal/automation/service_test.go` **改前就不符合 gofmt**（4 处 map 字面量对齐，与本次删除无关；改前改后 hunk 数均为 4）。按范围纪律不做全文件格式化，避免制造无关 diff。
4. **删 `settings/service_test.go` 的副作用**：`settings` 包**从此没有测试文件**。这是**现状的诚实反映**（它本来就没有测试），留着死 mock 反而会让人误以为该包有覆盖。已如实记录而非粉饰。
5. **刻意不做版本 bump 与发版**（见 design.md KD3）：纯删除不可达代码、无行为变化；且 bump 而不推镜像会让 README/compose 指向不存在的镜像 —— 这正是本仓库刚修好的那类不一致。镜像 tag 保持 `v0.0.44`，与线上实例一致。若要发布，另开任务。
6. **P3/P4 未动**：孤儿端点 `/accounts/{id}/refresh-auth`、`OfflineHandoffClientID`、`SourceTypeOfflineHandoff`、`drivers/template` 链路均保持原样，属待决策项。

## 规划期修订记录（实施中发现，**先改 PRD 再动手**）

**修订 1：`useTimeWindowSchedule.ts` 由"必须保留"改为"一并删除"**

- **原判**：复扫报告 §5.1 称"同族的 `composables/useTimeWindowSchedule.ts` **仍被引用**，故不可误删"，本 PRD 据此把它写入"必须保留"清单
- **否证**：删掉 `TimeWindowField.vue` 后，零引用复算立刻报出 `useTimeWindowSchedule.ts`。严格核查结果——
  - `grep -rn "useTimeWindowSchedule" web/src`（排除自身）：**空**（无 import、无注释、无任何提及）
  - `grep -rnE 'from "[^"]*useTimeWindowSchedule"|import\("[^"]*useTimeWindowSchedule'`：**无任何 import（含动态）**
  - 该文件 8 个导出（`TimeWindowMode`/`ScheduleMode`/`TimeWindowFields`/`TimeWindowWheelPayload`/`formatTimeWindowDisplay`/`applyTimeWindowFromTask`/`timeWindowPayload`/`useTimeWindowSchedule`）**全部无消费方**
  - **根因**：它此前唯一的"引用"是已删文件 `TimeWindowField.vue` 第 5 行与第 15 行的**两句注释**（"与 useTimeWindowSchedule 的 … 对应"）。复扫那套"茎名文本引用计数"**把注释也算作引用** → **假阴性**，故它当时未被列入零引用清单
- **处置**：按本任务"清理已证实死代码"的目标一并删除；并已把"**引用计数必须排除注释**"这一判据补进 D4 的 spec 指南，避免复发
- **说明**：本修订在**动手删除之前**完成（Plan↔Execute 回退），不是事后补账

**修订 2：D2 的零 churn 验证适用范围**

- 该验证（`npm run build` 后 `internal/api/web` 必须无变化）对**两个**被删前端文件同时成立；若任一个出现 churn，即说明它在 bundle 中，需回滚对应文件（判定见 implement.md G2）
