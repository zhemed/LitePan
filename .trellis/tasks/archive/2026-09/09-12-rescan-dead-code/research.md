# 死代码复扫报告（09-12-rescan-dead-code）

> 复扫时点：2026-09-12，代码版本 `v0.0.44`（`745e3a3`）
> 上次基线：`09-12-investigate-dead-code-sweep`（`deadcode prod` 32）+ `remove-dead-code-p1`（删 23）→ **9**
> **本轮零代码改动**，只扫描与判定。

---

## 1. 结论摘要

| 面 | 结论 |
|---|---|
| **Go 符号级（生产视角）** | **9 → 9，0 新增、0 消除** —— 与 `0.0.41` 之后的基线**逐条完全一致**，期间 `0.0.42/0.0.43/0.0.44` 与多轮维护改动**未引入新死代码** |
| **上次基线的判定纠错** | B 簇（`pause_reason.go`）上次记为"**仅测试可达**"，实测为**全文件零引用（含测试）** —— 上次低估了：不是 2 个死函数，而是**整个文件**（类型 + 3 常量 + 2 函数）都死 |
| **前端** | **新增发现 1 个零引用文件**（`TimeWindowField.vue`，122 行）—— 上次 P2 迭代删 23 文件时**漏掉了它**；另发现 **1 个未使用直接依赖** |
| **测试层（`unused` 交叉验证新发现）** | **14 条未使用测试替身**，其中 `internal/settings/service_test.go` **整个文件（28 行）是死的**（只定义了一个从未实例化的 mock，**连一个测试函数都没有**）。上次把这批记为"接口实现桩，不是死代码"——**该结论经实测否证**（见 §6） |
| **误报排除** | `@vue/devtools-api` **不是**无用依赖（`pinia` 的 peerDependency + `vue-router` 的 dependency），上一版文本检索会误报，**已排除** |
| **保留项复核** | `pkg/jsonvalue.FlexibleString.UnmarshalJSON` 的保留理由**仍然成立**（3 个驱动的配置结构体在用，`encoding/json` 反射调用） |
| **端点** | 复现上次结论：**1 条孤儿端点** `/accounts/{id}/refresh-auth`，全仓库仅 `router.go` 注册处出现 |

---

## 2. 方法（可复现）

```bash
# 工具（本轮新增安装）
go install golang.org/x/tools/cmd/deadcode@latest      # v0.50.0，落在 /root/go/bin

# Go 符号级：生产视角 + 含测试视角（与上次同一命令，保证可比）
deadcode ./cmd/litepan      > /tmp/dc_prod.txt
deadcode -test ./...        > /tmp/dc_test.txt

# 前端：文件茎名引用计数（vue/ts/js/css；排除入口与声明文件）
# 依赖：package.json ∩ web/** 文本检索 + package-lock 依赖树交叉验证
# 构建产物：解压 .gz 后检索（验证"零引用"是否真的没进 bundle）
# 端点：router.go 路由字面量 × 前端文本叶子段匹配
```

**防误报原则**（沿用上次）：`deadcode` 用 RTA（含接口分派）比 grep 可靠；但**两者冲突时必须逐个核查调用链**，且工具报不出类型/常量，故对每簇都做了人工可达性复核。

---

## 3. Go 符号级：9 → 9 逐项对照

```bash
deadcode ./cmd/litepan     # → 9 行，与基线逐条一致
```

| 簇 | 文件:行 | 符号 | 本轮状态 | 与基线对比 |
|---|---|---|---|---|
| **A** | `internal/httpx/oauth.go:20` | `PostOAuthProxyJSON` | 仍在 | ✓ 一致 |
| **A** | `internal/httpx/oauth.go:43` | `OAuthProxyHTTPError` | 仍在 | ✓ 一致 |
| **A** | `internal/httpx/oauth.go:65` | `OAuthProxyResponseError` | 仍在 | ✓ 一致 |
| **A** | `internal/httpx/do_json.go:10` | `DoJSON` | 仍在 | ✓ 一致 |
| **A** | `internal/httpx/envelope.go:11` | `ParseDataEnvelope` | 仍在 | ✓ 一致 |
| **B** | `internal/domain/pause_reason.go:11` | `PauseReason.AutoResumable` | 仍在 | ✓ 一致（但**判定需纠错**，见 §4.2） |
| **B** | `internal/domain/pause_reason.go:15` | `ValidAutoPauseReason` | 仍在 | ✓ 一致（同上） |
| **C** | `internal/upload/manager.go:570` | `OfflineHandoffClientID` | 仍在 | ✓ 一致 |
| **保留** | `pkg/jsonvalue/flexible_string.go:12` | `FlexibleString.UnmarshalJSON` | 仍在 | ✓ 一致（保留理由复核通过，见 §4.4） |

> **0 消除、0 新增**：`0.0.41` 之后所有改动（含 `0.0.42` 前端删 23 文件、`0.0.43` 上游移植、`0.0.44` 版本号重构、以及多轮文档/配置维护）**均未产生新的生产不可达符号**。

---

## 4. 逐簇可达性复核

### 4.1 A 簇（5 个）：仍属"驱动脚手架链"，保留理由未变

**证据**：

```bash
grep -rn "drivers/template" --include=*.go . | grep -v /tasks/
# → 仅 internal/auth/oauth_integration_test.go:14:  _ "litepan/drivers/template"   （空导入，测试文件）
```

- 生产侧**没有任何**对 `drivers/template` 的 import —— 它只被一个测试文件空导入
- A 簇 5 个符号的生产侧使用者**全部是 `drivers/template`**；其余是自身与测试
- 驱动注册表 `internal/driver/registry.go` 走 `init()` 注册，template **未注册**

**判定**：A 簇与 `drivers/template` 是**同一条链**。上次决策是"template 作为驱动开发脚手架与 auth 集成测试替身 → 保留，故 A 簇一并保留"，**该理由本轮仍然成立**（驱动接口未变、template 仍在被集成测试引用）。
**风险提示**：若确定不再新增驱动，可整链清理（A 簇 5 个符号 + template 的 6 文件/438 行）；这是**决策问题**，不是技术障碍。

### 4.2 B 簇（2 个）：**上次判定有误 —— 整文件都是死代码**

**证据**：

```bash
grep -rn "PauseReason" --include=*.go . | grep -v /tasks/ | grep -v pause_reason.go
# → 空（零引用）

cat internal/domain/pause_reason.go      # 17 行
#   type PauseReason string
#   const PauseReasonUser / PauseReasonAccountDisabled / PauseReasonAuthFailure
#   func (r PauseReason) AutoResumable() bool
#   func ValidAutoPauseReason(r PauseReason) bool
```

- 该类型的 3 个常量、2 个函数，**在包括测试在内的全部 Go 文件中零引用**
- `deadcode -test ./...` 的输出里**同样列出**这两个函数 —— 与"仅测试可达"的定义相反，证明测试也够不到它们
- 交叉印证：上次记的"仅测试覆盖"找不到任何测试引用

**判定**：**上次低估了规模** —— 不是"2 个仅测试可达的函数"，而是**整个 `internal/domain/pause_reason.go`（17 行）为真死代码**（`deadcode` 不报类型与常量，故只暴露出 2 行）。
**建议**：**整文件删除**（风险极低：零引用，且 `deadcode` 双视角一致）。

### 4.3 C 簇（1 个）：函数可删，但同文件的常量**仍在生产路径**

**证据**：

```bash
grep -rn "OfflineHandoffClientID" --include=*.go .
# → internal/upload/manager.go:570（定义） + manager_test.go ×3（调用）

grep -rn "SourceTypeOfflineHandoff" --include=*.go .
# → manager.go:405,477,479（生产逻辑分支） + types.go:64（定义） + 若干测试
```

- `OfflineHandoffClientID` 是**仅测试可达**（3 处调用全在 `manager_test.go`）
- 但**同源的 `SourceTypeOfflineHandoff` 常量仍被生产代码分支使用**（`manager.go:405/477/479`）

**判定**：`OfflineHandoffClientID` **可删**（低风险）；但**不可**顺手把 `SourceTypeOfflineHandoff` 一起删 —— 后者在生产的 source-type 分支里活着（很可能是为兼容历史任务的 `offline_handoff` 记录）。需先判定那些分支是否还有现实语义（属**功能决策**，超出死代码范畴）。

### 4.4 保留项（1 个）：保留理由**复核通过**

**证据**：

```bash
grep -rn "FlexibleString" --include=*.go .
# → drivers/115_Open/config.go:10   type flexString = jsonvalue.FlexibleString
# → drivers/189Cloud/config.go:5    type flexString = jsonvalue.FlexibleString
# → drivers/LocalFs/driver.go:17    CacheTTL jsonvalue.FlexibleString `json:"cache_ttl" ...`
```

- `FlexibleString` 被**三个驱动的配置结构体**使用（其中 LocalFs 是带 json tag 的字段）
- `UnmarshalJSON` 实现 `json.Unmarshaler`，由 `encoding/json` **反射调用** —— `deadcode(RTA)` 观察不到
- **删除它会破坏配置反序列化**（数字/字符串/null 三态兼容）

**判定**：**保留正确，理由未过期**。这正是上次"判据修正：凡实现标准接口的方法先人工复核"的范例，本轮复核确认。

---

## 5. 前端复扫

### 5.1 零引用文件：1 个（**本轮新增发现**）

| 文件 | 规模 | 证据 |
|---|---|---|
| `web/src/components/base/TimeWindowField.vue` | 122 行 | ① 茎名引用计数 **0**；② 全仓库仅自身出现 `TimeWindowField*` 标识符；③ **构建产物中 0 个文件含 `TimeWindowField`，而对照组件 `TimeWheelPicker` 在产物中存在（1 个文件）** → 确属未进 bundle |

- `web/src` 参与判定 199 个文件（排除入口/声明），**仅此 1 个零引用**
- **上次未提及**：`09-12-remove-dead-frontend-p2`（删 23 文件、迭代到不动点）与 `investigate-dead-code-sweep` 的记录中均无 `TimeWindow` —— 说明它是在那之后**新产生**的孤儿，或当时被漏判
- 同族的 `composables/useTimeWindowSchedule.ts` **仍被引用**（不在零引用列表中），故**不可**误删该 composable

### 5.2 依赖：1 个未使用 + **1 个误报已排除**

| 包 | 结论 | 证据 |
|---|---|---|
| `@fontsource-variable/noto-serif-sc` | **未使用** | `web/src`、`index.html`、`vite.config.ts` 全域零引用；CSS 亦无 `Noto Serif SC` 字体族（实际用的是 `Inter, "PingFang SC", "Microsoft YaHei", system-ui`）；构建产物中**无** noto 相关字体文件（只有 FontAwesome 的 woff2/ttf） |
| `@vue/devtools-api` | **误报，排除** | `package-lock` 依赖树显示它是 **`pinia` 的 peerDependency（^8.1.5）** 与 **`vue-router` 的 dependency（^8.1.5）**；`package.json` 直接声明 `^8.2.1` 是**满足 peer 依赖的正确做法**，**不得删除** |

> 依赖共 24 个；文本检索会漏判 peer 依赖类，故本轮**必须**用 `package-lock` 交叉验证 —— 上表第 2 行即该方法拦下的误报。

---

## 6. 测试层死代码：14 条（`unused` 交叉验证的**新发现**）

`deadcode` 只看"从 main 是否可达"，天然漏掉**测试内部**的死代码。本轮补跑交叉验证后暴露出一整类此前从未纳入视野的死代码：

```bash
golangci-lint run --enable=unused -c .golangci.yml ./...   # → 14 issues (unused)
```

| 文件 | 条目 | 规模 |
|---|---|---|
| `internal/settings/service_test.go` | `type memoryConfigRepo` + `Get`/`Set`/`All` | **整个文件 28 行**（无任何测试函数） |
| `internal/automation/service_test.go` | `type apiKeyRepo` + `List`/`Get`/`GetByHash`/`Count`/`Create`/`Update`/`Delete`/`TouchLastUsed` + `automationRunRepo.count` | 10 个符号（该文件本身仍有在用的测试，318 行） |

### 6.1 关键核实：它们**不是**"接口实现桩"，而是真死代码

上一版报告（§10 第 5 条）把这批记为"**接口实现桩，不是死代码**"。本轮实测**否证**了该结论：

```bash
grep -n "apiKeyRepo" internal/automation/service_test.go
# → 仅 305（type 定义）+ 309~318（method 定义），**无任何实例化、无任何传参**
grep -n "memoryConfigRepo" internal/settings/service_test.go
# → 仅 7（type 定义）+ 11/16/21（method 定义），同样零实例化

grep -rn "apiKeyRepo\|memoryConfigRepo" --include=*.go .    # 全仓库：仅上述两文件
```

- **"接口实现桩"的前提是被实例化并赋给接口变量**；实测两者**从未被实例化**（无 `&apiKeyRepo{...}`、无 `memoryConfigRepo{...}`），也不被同包其它测试文件引用
- `internal/settings/service_test.go` 全文 28 行**只有这个未使用的 mock，没有任何 `func Test*`** —— **整文件都是死代码**
- 对照：同包的 `automationRunRepo` **被实例化 4 处**，是活桩，故其方法**不在**本次清理建议内（只有 `count` 未被调用）

### 6.2 为什么 `deadcode` 与 `unused` 结论不同（互补而非冲突）

| 工具 | 视角 | 本轮命中 |
|---|---|---|
| `deadcode ./cmd/litepan` | 从 main 出发的生产可达性（RTA） | 9 条（生产死） |
| `golangci-lint --enable=unused` | **包内**（含 `_test.go`）未使用标识符（U1000） | 14 条（**测试死**） |

- 两者**零重叠**：`unused` 因把同包测试文件算作"使用方"，故看不到那 8 条仅测试可达的生产符号；`deadcode` 则完全看不见测试内部的死代码
- **结论：单一工具不足以覆盖，两者必须并用** —— 这是本轮最有方法论价值的一条

---

## 7. 端点：1 条孤儿（复现上次结论）

```bash
# router.go 路由字面量 95 个，叶子段在前端文本中找不到的：1 个
#   /accounts/{id}/refresh-auth
grep -rn "refresh-auth" . --include=*.go --include=*.ts --include=*.vue
# → 仅 internal/api/router.go:186 注册处，无任何调用方、无测试
```

- 前端 `web/src/api/accounts.ts` 的调用**全部是字面路径**（`toggle`/`set-default`/`refresh-profile` 等），**不存在**动态拼路径掩盖的情况 → 该判定**可靠**
- 该端点为管理员路由（`/admin/accounts/{id}/refresh-auth` → `h.refreshAccountAuth`），可能是为**外部 API 调用方**预留
- **判定**：**"待定"**（与上次一致）。删除前需确认是否有 API-key 使用者；属功能决策。

---

## 8. 误报排除与未做项

**已排除的误报**：

1. `@vue/devtools-api` —— 实为 pinia 的 peerDependency + vue-router 的 dependency（§5.2），**不得删除**
2. `drivers/template` 本身未出现在 `deadcode` 输出（工具只报函数，不报包）—— 其"测试专用"结论由 import 分析得出，非工具输出

**❗ 本轮自我纠错（一条曾被误判为"非死代码"的项**）：

- **`automation/service_test.go` 与 `settings/service_test.go` 里的 14 个符号**：本报告初稿与**上一版报告 §10 第 5 条**都把它们记为"接口实现桩，不是死代码 / RTA 看不到测试内接口分派"。
  **该结论错误** —— 经实例化核查（§6.1），两者**从未被实例化**，不具备"实现接口供测试注入"的前提，实为**真死代码**。
  教训与上次 `FlexibleString` 那条**同源**：**凡涉及接口的方法，必须先核实"它究竟有没有被实例化/赋给接口变量"，再下结论**；"看起来像桩"不能代替核查。

**未做 / 待定**：

- 端点使用率的**完整**核对仍无法穷尽（95 条路由的叶子段匹配是启发式；动态拼接路径理论上仍可能漏判）→ **保留"待定"**，但本轮把可疑集收缩到 1 条且已人工确认其可靠性
- `SourceTypeOfflineHandoff` 生产分支的**现实语义**（是否仍需兼容历史记录）→ 属功能决策，未判定
- `drivers/template` 是否继续保留 → 属决策，未判定
- 未评估删除后的二进制体积变化（预期 ≈0：生产死代码本不入二进制；测试死代码影响更小）

---

## 9. 清理建议（按优先级，**本任务不执行**）

| 优先级 | 项 | 规模 | 风险 | 说明 |
|---|---|---|---|---|
| **P1** | `internal/settings/service_test.go` **整文件** | 28 行 | **极低** | 全文只有一个从未实例化的 mock，**无任何测试函数** → 整文件删除不影响测试覆盖 |
| **P1** | `internal/domain/pause_reason.go` **整文件** | 17 行 | **极低** | 全文件（类型+3 常量+2 函数）零引用，双视角一致；上次记录低估了规模 |
| **P2** | `web/src/components/base/TimeWindowField.vue` | 122 行 | **低** | 零引用且未进构建产物；须**只删该文件**，勿动 `useTimeWindowSchedule.ts` |
| **P2** | `@fontsource-variable/noto-serif-sc` 依赖 | `package.json` 1 行 + lock | **低** | 全域零引用、无字体族、无产物；删后需 `npm install` 更新 lock |
| **P2** | `automation/service_test.go` 的 10 个死符号（`apiKeyRepo` 全套 + `automationRunRepo.count`） | 约 15 行 | **低** | 该文件仍有在用的测试；**只删这些符号**，勿动 `automationRunRepo` 的其它方法 |
| **P3** | `internal/upload/manager.go:570` `OfflineHandoffClientID` | 1 函数 | 低 | 仅测试可达；同文件常量**不可**连带删除 |
| **P3** | `/accounts/{id}/refresh-auth` 端点 | 1 路由 + 1 handler | 中 | **待定**：可能是外部 API 预留；删除前须确认无 API-key 使用者 |
| **P4（决策）** | A 簇 5 符号 + `drivers/template` | ~438 + ~150 行 | 中 | 仅当确定不再新增驱动时整链清理；否则保留脚手架 |

> **合计可安全清理 ≈ 182 行**（P1 的 45 行 + P2 的 137 行）+ **1 个依赖**；**需决策的**另有 P3/P4 三项。
> 其中 **14 个测试层符号（含整个 `settings/service_test.go`）是本轮新发现**，上次清理完全未覆盖。

---

## 10. 复现命令（关键）

```bash
# 工具
go install golang.org/x/tools/cmd/deadcode@latest
export PATH=/usr/local/go/bin:/root/go/bin:$PATH

# Go 符号级（与基线同命令）
deadcode ./cmd/litepan      # → 9 行
deadcode -test ./...        # → 15 行（多出的 12 个是测试层）

# 交叉验证：测试层死代码（deadcode 看不见的部分）
golangci-lint run --enable=unused -c .golangci.yml ./...   # → 14 issues (unused)

# 测试替身是否真死（"是否被实例化"的核查 —— 下结论前必做）
grep -n "apiKeyRepo" internal/automation/service_test.go      # 仅定义，无实例化
cat internal/settings/service_test.go                         # 28 行，无任何 Test 函数

# B 簇真死判定
grep -rn "PauseReason" --include=*.go . | grep -v pause_reason.go      # → 空
cat internal/domain/pause_reason.go                                    # → 17 行

# A 簇脚手架链
grep -rn "drivers/template" --include=*.go .                           # → 仅 1 个测试文件空导入

# 保留项复核
grep -rn "FlexibleString" --include=*.go .                             # → 3 个驱动配置在用

# 前端
grep -rn "TimeWindowField" web/src                                     # → 仅自身
find internal/api/web -name '*.gz' -exec sh -c 'gzip -dc "$1" | grep -l TimeWindowField' _ {} \; 2>/dev/null  # 0 命中
grep -rn "fontsource\|noto-serif\|Noto Serif" web/src web/index.html web/vite.config.ts   # → 空

# 依赖误报排除
python3 -c "import json; lock=json.load(open('web/package-lock.json'))['packages']; print([(k,v.get('peerDependencies',{}).get('@vue/devtools-api')) for k,v in lock.items() if v.get('peerDependencies',{}).get('@vue/devtools-api')])"
```
