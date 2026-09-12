# 复扫死代码：核对上次遗留的 9 处并检查 0.0.42~0.0.44 期间有无新增

## Goal

上次排查把 Go 侧 `deadcode prod` 从 **32 降到 9**（删 3 孤儿包 + 23 真死函数、前端删 23 文件），此后又经过 `0.0.42`/`0.0.43`/`0.0.44` 与多轮维护改动。本任务**复扫一遍**，回答三个问题：

1. 上次遗留的 **9 处**现在是什么状态？
2. `0.0.41` 之后有没有**新增**死代码？
3. 前端文件与依赖有没有新的零引用残留？

**零代码改动** —— 只扫描、复核、出报告；是否清理交由用户决定后再另开任务。

## Background（上次留下的精确基线）

`09-12-investigate-dead-code-sweep/research.md` 的结论是 `32 = 24 真死 + 8 仅测试可达`；随后 `remove-dead-code-p1` 删掉 23 个（真死 24 个中保留 1 个），故 `32 → 9`。

**遗留的 9 处 = 8 个"仅测试可达" + 1 个刻意保留**：

| 簇 | 文件 | 符号 | 性质 |
|---|---|---|---|
| **A**（5） | `internal/httpx/{oauth.go,do_json.go,envelope.go}` | `PostOAuthProxyJSON`、`OAuthProxyHTTPError`、`OAuthProxyResponseError`、`DoJSON`、`ParseDataEnvelope` | OAuth 代理链路（123/百度/OneDrive 已删）；现仅 `drivers/template` 与自身测试使用 |
| **B**（2） | `internal/domain/pause_reason.go` | `PauseReason.AutoResumable`、`ValidAutoPauseReason` | **已删任务类型**的暂停原因，仅测试覆盖 |
| **C**（1） | `internal/upload/manager.go` | `OfflineHandoffClientID` | **离线交接**（已删功能），仅测试覆盖 |
| **保留**（1） | `pkg/jsonvalue/` | `FlexibleString.UnmarshalJSON` | 实现 `json.Unmarshaler`，由 `encoding/json` **反射调用** → `deadcode(RTA)` 盲区，删除会改变解码语义 |

**本次复扫的重点判断**：B/C 两簇是**"已删功能的残留，靠测试保活"** —— 即测试在测试死代码。这与 A 簇（`drivers/template` 是**有意保留**的驱动开发脚手架，其依赖必须存活）性质不同，须分开结论。

**上次的未做项**（research.md §10）：后端**端点使用率**的完整核对，当时标注"待定"（前端路径拼装方式多样、动态拼接无法穷尽）。

**工具现状**：`deadcode`、`staticcheck`、`unused` **均未安装**（`golangci-lint` 已装，但项目 `.golangci.yml` 只启用 `depguard/errcheck/govet/staticcheck`，**不含 `unused`**）。

## Requirements

### R1 补齐工具

- 装 `deadcode`（`go install golang.org/x/tools/cmd/deadcode@latest`，落在 `/root/go/bin`），记录版本

### R2 Go 符号级复扫并与基线逐项比对

- 跑 `deadcode ./cmd/litepan`（生产视角），与上表 **9 处逐一比对**，输出三类结果：**仍在 / 已消除 / 新增**
- 若有**新增**，每个都必须给出：路径、符号、规模、可达性结论

### R3 交叉验证（避免单一工具的盲区）

- 用 `golangci-lint run --enable=unused`（**ad-hoc，不改 `.golangci.yml`**）跑一轮，与 `deadcode` 结果对照；两者不一致的项逐个核查调用链是否本身是死链
- 任一工具因版本/依赖 panic 不可用时，**如实记录并给替代证据**，不得跳过

### R4 对遗留 9 处做可达性复核（不得只凭工具输出下结论）

对每一项确认：动态引用、反射调用、注册表/`init()`、字符串常量拼接、测试独占使用。特别地：

- **A 簇**：确认 `drivers/template` 是否仍被使用（若它本身已成死代码，A 簇的保留理由即失效）
- **B/C 簇**：确认对应功能是否确已删除；若确已删，判定为"**测试在测死代码**"，给出"连测试一起清理"的建议与规模
- **保留项**：复核 `FlexibleString.UnmarshalJSON` 的反射路径是否仍成立（防止上次的结论过期）

### R5 前端与其它面复扫

- 前端：文件级可达性（basename/stem 引用计数，排除入口与声明文件）—— 上次删 23 文件后的现状，是否有**新的**零引用文件
- 依赖：`web/package.json` 的 dependencies 与 `web/**` 实际引用对照（未使用的依赖）
- 构建产物、注册表、配置项：沿用上次 §6~§8 的方法做快速复核

### R6 输出 `research.md`

须含：① 结论摘要；② 与 9 处基线逐项比对的**对照表**；③ 新增死代码（若有）的分类清单与证据；④ 误报排除与未确认项（标"待定"）；⑤ 按优先级排序的清理建议（规模 + 风险 + 依赖关系）；⑥ **可复现命令**。
**每项结论必须附可复核证据**（命令 + 输出摘要），不得仅凭"名字看起来像"下判断。

### R7 零代码改动

- `git status` 除本任务目录外为空；不删除、不修改任何源码与配置

## Constraints

- **只扫不删**：本任务产出报告与建议，**不执行任何清理**（清理须另开任务）
- **不改 `.golangci.yml`**：交叉验证用命令行 `--enable=unused`，不写入项目配置
- **不改运行中的实例**（`litepan` 容器、`data/litepan.db`）与 DSH
- **不改归档任务与历史 journal**（上次的 `research.md` 是历史证据，只读引用）
- 装工具属宿主写操作，已在本任务范围内；**不得**顺带升级/改动其它工具链
- 不确定的结论标"**待定**"，不得为凑结论而推断

## Acceptance Criteria

- [x] `deadcode` 已安装并记录版本；`deadcode ./cmd/litepan` 跑出结果 —— 实测 `golang.org/x/tools v0.50.0`（`go version -m` 读取，装在 `/root/go/bin`）；`deadcode ./cmd/litepan` **exit=0，9 行**；`deadcode -test ./...` 15 行
- [x] 产出与基线 **9 处的逐项对照表**（仍在 / 已消除 / 新增），每项附证据 —— research.md §3：**9 条逐项一致，0 新增、0 消除**（含路径:行、符号、与基线对比列）
- [x] `golangci-lint run --enable=unused` 交叉验证完成，与 `deadcode` 的**不一致项已逐个核查**并说明 —— 实测 **14 issues (unused)**，与 `deadcode` 的 9 条**零重叠**；已在 §6.2 说明原因（视角不同、互补），并对 14 条**逐个核查了"是否被实例化"**
- [x] A/B/C 三簇与保留项**各自给出可达性结论**（含 `drivers/template` 是否仍被使用的判定）—— §4.1 A 簇：`drivers/template` **仅被 1 个测试文件空导入**、注册表未注册 → 脚手架链保留理由成立；§4.2 B 簇：**全文件零引用**；§4.3 C 簇：函数仅测试可达、同源常量**仍在生产分支**；§4.4 保留项：3 个驱动配置在用，反射路径成立
- [x] B/C 簇若确为"测试在测死代码"，给出**清理建议 + 规模 + 风险** —— §9 表：B 簇实为**全文件真死**（非仅测试，纠正上次判定）→ P1 整文件 17 行、风险极低；C 簇 `OfflineHandoffClientID` → P3、风险低，并明确"同文件 `SourceTypeOfflineHandoff` **不可**连带删除"
- [x] 前端复扫完成：新的零引用文件清单 + 未使用依赖清单 —— §5：**1 个新增零引用文件**（`TimeWindowField.vue` 122 行，经构建产物交叉验证未进 bundle）+ **1 个未使用依赖**（`@fontsource-variable/noto-serif-sc`）；并**排除 1 个误报**（`@vue/devtools-api` 是 pinia 的 peer 依赖）
- [x] `research.md` 已产出，含结论摘要、对照表、证据、误报排除、待定项、优先级建议、复现命令 —— 实测 **296 行、10 个章节**（1 结论 / 2 方法 / 3 对照 / 4 逐簇复核 / 5 前端 / 6 测试层 / 7 端点 / 8 误报与待定 / 9 建议 / 10 复现）
- [x] **零代码改动** —— 实测 `git status --short` 仅 `?? .trellis/tasks/09-12-rescan-dead-code/`；`git diff` **0 行**
- [x] 运行中实例未受影响 —— 实测 `litepan | Up About an hour`，`/api/health` HTTP 200
- [x] **附加**：补跑 `unused` 后**新发现一整类死代码** —— 14 个测试层符号，其中 `internal/settings/service_test.go` **整文件（28 行）是死的**（只有未使用的 mock、**无任何 `Test*` 函数**）；且该批**上次被记为"接口实现桩，不是死代码"，本轮实测否证**（两者从未被实例化）

## Notes

- Scope 标注 `lightweight`：纯调查，无技术方案分歧，与上次 `09-12-investigate-dead-code-sweep`（同为 `lightweight`）一致，故 PRD-only。
- 本任务**不**做：任何删除、`go mod tidy`、依赖升级、`.golangci.yml` 变更、DB 清理。
- 与上一次的衔接：上次 `research.md` §10 遗留的"端点使用率核对"标注待定 —— 本任务**尝试**推进，若仍无法穷尽，继续标"待定"并说明卡点，不强行给结论。

## 检查记录（trellis-check，2026-09-12）

**Step 1 变更识别**：**零代码改动** —— `git diff` 为空，`git status` 仅本任务目录（`research.md` + `prd.md`）。宿主侧新增 `deadcode` 工具（`/root/go/bin/deadcode`，在仓库外）。

**Step 3 项目质量门**：代码质量门（lint/vet/test/type-check）**N/A** —— 本任务只读扫描，未改动任何代码，故无需回归；本轮反而**额外跑了** `golangci-lint --enable=unused` 作为排查手段（ad-hoc，**未写入 `.golangci.yml`**）。

**Step 4 清单核对**

- 测试覆盖：**N/A**（无代码改动）。但本轮**发现了测试层的死代码**并给出清理建议，属"测试资产卫生"而非"测试覆盖"。
- Spec 同步：**建议但本任务未执行** —— 本轮最有价值的方法是"**`deadcode` 与 `unused` 必须并用，且涉及接口的方法下结论前必须核查是否被实例化**"。这条方法论值得写进 spec（与 `api-layering.md` 的"版本号单一来源"同级）。**但本任务的 PRD 未声明 spec 改动**，为守范围纪律**不擅自扩大**，改为在下方提出建议、交由用户决定是否另开任务。
- 范围纪律：**零代码改动**，只装了一个分析工具；未删除任何代码；未改 `.golangci.yml`；未触碰实例。
- 跨层一致性：N/A。

**诚实记录：我自己的两次判断失误（均已纠正）**

1. **我把 14 条测试层死代码先判成了"非死代码"**：初稿 §7 里我沿用了上一版报告"接口实现桩，RTA 看不到接口分派"的说法。**经实例化核查后否证** —— 那些类型**从未被实例化**，不具备"实现接口供注入"的前提，实为真死代码。research.md §8 已用醒目段落记录该纠错，并写明教训：**"看起来像桩"不能代替"是否被实例化"的核查**。
2. **同类错误上一版也犯过**：上一版 §10 第 5 条同样把它们排除。**两次都是同一个思维捷径** —— 看到"测试里的 mock"就默认它是必要的。本轮把这条写成了可复用的判据。

**其它需要你知道的**

3. **上次基线另有一处低估**：B 簇被记为"仅测试可达"，实测为**全文件零引用（含测试）** → 可清理规模比记录的大（整个文件 17 行，而非 2 个函数）。原因是 `deadcode` 只报函数、不报类型与常量。
4. **两项"待定"未强行给结论**：孤儿端点 `/accounts/{id}/refresh-auth`（可能是外部 API 预留）与 `SourceTypeOfflineHandoff` 的生产分支语义，均属**功能决策**，已标注待定并说明卡点。
5. **`deadcode -test` 多出的 12 条**曾被我看作"RTA 盲区"，实为 `deadcode` **正确地**报出了测试层死代码 —— 说明该输出不该被忽略。
6. **本轮未评估删除后的体积变化**（预期 ≈0：生产死代码本不入二进制）。
