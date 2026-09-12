# Design: 死代码清理 P3（复扫确认的 P1+P2）

## Overview

本任务只做**删除**，不新增任何逻辑。核心设计问题不是"怎么删"，而是**"怎么证明删对了"** —— 删死代码的风险全在"误判仍被引用"，而误判正是上一轮报告犯过的错（把从未实例化的测试 mock 当成必要的接口桩）。

因此设计围绕**三条独立验证**展开：

1. **删除前再实证一次**（不依赖上一轮报告的结论，重跑判定命令）
2. **删除后工具复测**（`deadcode` 9→7、`unused` 14→0，数量变化必须与预期**精确吻合**）
3. **构建产物零 churn**（删前端组件后重建 embed，产物若有任何变化即说明它**原本在 bundle 里** → 判定为误判，回滚）

第 3 条是本设计里最有价值的一条：它是**唯一能自证"前端删除判定"正确与否的客观反证**，且项目在上一轮 `remove-dead-frontend-p2` 已用过同一手法（"构建产物零 churn 反证不在 bundle"）。

## Boundaries

| 对象 | 删 | 保留（**明令不动**） |
|---|---|---|
| **Go 生产代码** | `internal/domain/pause_reason.go`（整文件 17 行） | 其余全部；尤其 `internal/upload/manager.go:570` 的 `OfflineHandoffClientID`、`types.go:64` 的 `SourceTypeOfflineHandoff` |
| **Go 测试代码** | `internal/settings/service_test.go`（整文件 28 行）；`internal/automation/service_test.go` 的 10 个符号 | `automationRunRepo` 类型与其余方法（**活桩，被实例化 4 处**）；该文件其余测试 |
| **前端** | `web/src/components/base/TimeWindowField.vue`（122 行） | `web/src/composables/useTimeWindowSchedule.ts`（同族但**仍被引用**） |
| **依赖** | `@fontsource-variable/noto-serif-sc`（`package.json` + lock） | `@vue/devtools-api`（**已排除的误报**：pinia 的 peerDependency + vue-router 的 dependency） |
| **构建配置** | 无 | `.golangci.yml`（`unused` 保持 ad-hoc，**不进强制门**）、`vite.config.ts`、`Dockerfile` |
| **版本声明** | 无 | `README.md` / `docker-compose*.yml` 的镜像 tag（保持 `v0.0.44`，见 KD3） |
| **API 表面** | 无 | `/accounts/{id}/refresh-auth` 路由与 handler（**P3 待定，未授权**） |
| **spec** | 无 | 既有 spec 全部；本任务**新增**一份指南并登记索引 |

## Verification Strategy（本设计的核心）

```
删除前（再实证，不信上一轮结论）
  ├─ grep -n "apiKeyRepo\|memoryConfigRepo" → 确认仍只有定义、无实例化
  ├─ grep -rn "PauseReason" | grep -v pause_reason.go → 确认为空
  ├─ grep -rn "TimeWindowField" web/src → 确认仅自身
  └─ 记录基线：deadcode=9、unused=14、前端零引用=1、deps=24
        │
删除（D1 → D2 → D3 → D4）
        │
删除后（三项复测，数量必须精确吻合）
  ├─ ① deadcode ./cmd/litepan        → 必须 = 7（9 − B 簇 2）
  ├─ ② golangci-lint --enable=unused → 必须 = 0（14 − 14）
  ├─ ③ 前端零引用 = 0、deps = 23
  ├─ ④ 重构产物零 churn：npm run build 后 git status internal/api/web 为空
  └─ ⑤ 质量门：make lint / go vet / go test / vue-tsc / vite build 全绿
        │
任一不吻合 → 该步判定为误判，回滚该项并重新判定
```

**为什么数量必须"精确吻合"而不是"变少就行"**：如果删除后 `deadcode` 降到了 6 而不是 7，说明**多删了一个正在被间接引用的符号**；如果 `unused` 没归零，说明**少删了**或引入了新的未使用符号。精确数量是唯一能把"删干净了"与"删过头了"区分开的判据。

## Key Decisions

### KD1 删除前重跑判定，不继承上一轮结论

上一轮报告（与我的初稿）在测试 mock 上**误判过一次**。所以本设计的第 0 步是**用命令重新证明"它们确实没被实例化"**，而不是引用"复扫报告说它是死的"。理由：**引用结论无法发现结论本身错**。

### KD2 用"构建产物零 churn"证明前端删除判定

`TimeWindowField.vue` 是否真不在 bundle 里，**不能靠 grep**（grep 只能证明"没有显式 import"，证明不了"没有动态引用/间接引用"）。`npm run build` 后对比 `internal/api/web/**`：
- **零变化** → 它原本就不在 bundle 中 → 删除安全
- **有变化** → 它其实在 bundle 里 → **立即回滚该文件**

这是本任务唯一一个**能用客观产物反证判定**的环节，必须做。

### KD3 不做版本 bump、不发版（**刻意偏离**，已留痕）

- 本任务**纯删除不可达代码，零行为变化**（删的都是从 main 不可达或从未被实例化的东西）→ 没有用户可见变化，不构成发版理由
- 更重要的是：**若 bump 到 `v0.0.45` 而不推镜像**，`README.md`/`docker-compose*.yml` 就会指向一个**不存在的镜像** —— 这正是本仓库反复出现、并在 `09-12-version-single-source-release-0-0-44` 里刚修好的那类不一致
- 因此：**镜像 tag 保持 `v0.0.44`**（与线上正在运行的实例一致），**代码版本也不动**，避免制造"代码说 0.0.45、镜像只有 0.0.44"的新裂缝
- 若日后要把它发布出去，**另开任务**走完整发版流程（build → 推 GHCR 三 tag → tag → release → 本地容器验证）
- 该偏离已在 PRD Constraints 与检查记录中显式声明，**不是遗忘**

### KD4 `unused` 不进强制门

`golangci-lint --enable=unused` 在本任务中只作为**排查工具**用命令行调用，**不写入 `.golangci.yml`**。理由：
- 它与 `deadcode` 一样存在**系统性误报类**（接口实现桩、测试替身、反射调用），本轮就亲手处理过 14 条需要人工裁决的
- 强行进门会让**每一处接口桩都要写 `//nolint`**，反而鼓励无注释的抑制
- 与前端"浏览器验收不进 CI"同源：**排查工具与门禁是两回事**

### KD5 交付顺序：先 Go 后前端，spec 最后

Go 侧改动的验证最快（`go test` 秒级），前端要跑 `vue-tsc` + `vite build`（分钟级）且会触发生成物。故先做 Go（D1/D3），再做前端（D2），最后写 spec（D4）—— 一旦前端产物 churn 异常，前面已验证的 Go 部分不受影响，回滚面最小。

## Compatibility

- **Go**：删除的两处均无调用方，编译期即可验证（`go build` 失败即说明判据错）
- **前端**：`TimeWindowField.vue` 无 import；`useTimeWindowSchedule.ts` 保留不动 → 无组件依赖破坏
- **依赖**：移除 `noto-serif` 后 `package-lock.json` 由 `npm install` 重算；`@vue/devtools-api` 保留 → pinia/vue-router 的 peer 约束仍满足
- **既有测试**：`automation/service_test.go` 删符号后其余测试必须仍编译通过（`go test` 验证）；`settings` 包删后**无测试文件**（现状如此，非本任务造成）
- **embed 产物**：预期零 churn；若 churn 则说明前端判定错误 → 回滚
- **运行中实例**：完全不涉及（不重建容器、不碰 `data/`）

## Tradeoffs

- **不 bump 版本 vs 沿用"改动即 bump"惯例**：选择不 bump，换取"代码-镜像-文档三者仍一致"。代价是提交历史里这次改动不带版本号 —— 由 commit message 与任务档案说明。
- **删 `settings/service_test.go` 整文件 vs 只删其中的 mock**：整文件删更干净（文件除 mock 外无任何内容），代价是 `settings` 包**从此没有测试文件**。但这是**真实状态的诚实反映**（它本来就没有测试），留着死 mock 只会让人误以为该包有测试。
- **`unused` 只做 ad-hoc vs 写进配置**：ad-hoc 保住"零 `//nolint`"的洁癖，代价是**不会自动发现**未来新增的测试层死代码 —— 需靠定期复扫（本任务的 spec 指南就是为此）。

## Rollout / Rollback

**Rollout**：D1 → D2 → D3（每步后立即复测）→ D4（spec）→ 质量门 → 归档。

**Rollback**（逐项独立，均可用 git 恢复）：

| 项 | 回滚 |
|---|---|
| Go 两个文件 | `git checkout HEAD -- internal/settings/service_test.go internal/domain/pause_reason.go` |
| 测试符号 | `git checkout HEAD -- internal/automation/service_test.go` |
| 前端文件 | `git checkout HEAD -- web/src/components/base/TimeWindowField.vue` |
| 依赖 | `git checkout HEAD -- web/package.json web/package-lock.json` 后 `npm ci` |
| embed（若 churn） | `git checkout HEAD -- internal/api/web` 并**回滚对应前端删除** |
| spec | `git checkout HEAD -- .trellis/spec/guides/` |

**触发回滚的硬条件**：任一复测数量不符预期、或 `go build`/`vue-tsc`/`vite build` 失败、或 embed 出现 churn。

## File Map

| 路径 | 动作 | 归属 |
|---|---|---|
| `internal/settings/service_test.go` | **删除**（28 行） | D1 |
| `internal/domain/pause_reason.go` | **删除**（17 行） | D1 |
| `web/src/components/base/TimeWindowField.vue` | **删除**（122 行） | D2 |
| `web/package.json`、`package-lock.json` | 移除 `@fontsource-variable/noto-serif-sc` | D2 |
| `internal/automation/service_test.go` | 删 10 个死符号（约 15 行） | D3 |
| `.trellis/spec/guides/dead-code-guide.md` | **新增** | D4 |
| `.trellis/spec/guides/index.md` | 加 1 行指南表条目 | D4 |
| `internal/api/web/**` | **预期零改动**（若改则回滚） | D2 验证 |
| `README.md`、`docker-compose*.yml`、`internal/buildinfo/version.go` | **零改动**（见 KD3） | — |
| `refresh-auth` 路由、`OfflineHandoffClientID`、`drivers/template` | **零改动**（P3/P4 未授权） | — |
