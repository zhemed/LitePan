# Implementation Plan: 死代码清理 P3（复扫确认的 P1+P2）

## Overview

顺序：**基线实证 → D1（Go 文件）→ D3（Go 测试符号）→ D2（前端）→ D4（spec）→ 质量门 → 归档**。

每步后**立即复测数量**；任一不符预期即停在该步并按 design.md 的回滚表处理，**不带着疑点往前走**。

本机 `danger-full-access` 且审批关闭 —— 不请求任何 escalation。

---

## Phase 0: 基线实证（不继承上一轮结论）

- [ ] 0.1 `git status --short` 记录基线（期望：仅本任务目录）
- [ ] 0.2 **重新实证三条删除判据**（不引用复扫报告）：
      - `grep -n "apiKeyRepo\|memoryConfigRepo" internal/automation/service_test.go internal/settings/service_test.go` → **只有定义、无实例化**
      - `grep -rn "PauseReason" --include=*.go . | grep -v pause_reason.go` → **空**
      - `grep -rn "TimeWindowField" web/src` → **仅自身**
- [ ] 0.3 记录四项基线数值：`deadcode ./cmd/litepan`=**9**、`golangci-lint --enable=unused`=**14**、前端零引用文件=**1**、`dependencies`=**24**
- [ ] 0.4 记录 `internal/api/web` 当前状态（供后续 churn 比对）：`git status --short internal/api/web`（期望空）

**回滚点 R0**：以上数值即后续"精确吻合"的判据来源

---

## Phase 1: D1 删除两个 Go 死文件

- [ ] 1.1 `git rm internal/settings/service_test.go internal/domain/pause_reason.go`
- [ ] 1.2 **验证门 G1**：
      - `GOWORK=off go build ./...` 通过（编译期即证无调用方）
      - `GOWORK=off go vet ./...` exit=0
      - `GOWORK=off go test ./...` 全包 ok（**注意**：`settings` 包将不再有测试文件，输出应为 `no test files` 而非失败）
      - `deadcode ./cmd/litepan` → **必须 = 7**（9 − 2）
- **回滚点 R1**：`git checkout HEAD -- <两个文件>`

---

## Phase 2: D3 删除测试层死符号（10 个）

- [ ] 2.1 从 `internal/automation/service_test.go` 删除：`type apiKeyRepo` 及其 8 个方法、`automationRunRepo.count`
      —— **保留** `automationRunRepo` 类型与其余方法（活桩，被实例化 4 处）
- [ ] 2.2 **验证门 G3**：
      - `GOWORK=off go vet ./...` exit=0
      - `GOWORK=off go test ./internal/automation/` **通过**（证明删符号未破坏该文件其余测试）
      - `GOWORK=off go test ./...` 全包 ok
      - `golangci-lint run --enable=unused -c .golangci.yml ./...` → **必须 = 0**
- **回滚点 R3**：`git checkout HEAD -- internal/automation/service_test.go`

---

## Phase 3: D2 前端删除 + 依赖移除 + embed 零 churn 证明

- [ ] 3.1 `git rm web/src/components/base/TimeWindowField.vue`
- [ ] 3.2 **确认同族文件未被误删**：`ls web/src/composables/useTimeWindowSchedule.ts`
- [ ] 3.3 从 `web/package.json` 移除 `@fontsource-variable/noto-serif-sc`，运行 `npm install` 更新 lock
- [ ] 3.4 **确认未误删**：`grep -n "@vue/devtools-api" web/package.json` 仍在
- [ ] 3.5 `cd web && npm run type-check` → exit=0
- [ ] 3.6 `cd web && npm run build`
- [ ] 3.7 **验证门 G2（关键反证）**：`git status --short internal/api/web` → **必须为空**
      - 为空 → 被删组件**原本不在 bundle**，删除判定成立 ✅
      - **非空 → 判定错误，立即 `git checkout HEAD -- internal/api/web` 并恢复被删组件**
- [ ] 3.8 复测前端指标：零引用文件 = **0**；`dependencies` 计数 = **23**
- **回滚点 R2**：`git checkout HEAD -- web/src web/package.json web/package-lock.json internal/api/web` 后 `npm ci`

---

## Phase 4: D4 方法论写入 spec

- [ ] 4.1 新增 `.trellis/spec/guides/dead-code-guide.md`，含四要素：
      ① **两视角必须并用**（`deadcode` 生产可达性 vs `unused` 包内未使用，附两者**零重叠**的本轮实例：9 条 vs 14 条）
      ② **两条硬核查**：涉及接口的方法 → 先查是否被实例化；`deadcode` **只报函数**，类型与常量需人工补查
      ③ **`unused` 不进强制门**（ad-hoc 排查工具，理由同浏览器验收）
      ④ 完整复现命令
- [ ] 4.2 在 `.trellis/spec/guides/index.md` 的 "Available Guides" 表登记该指南
- [ ] 4.3 **验证门 G4**：`grep -n "dead-code-guide" .trellis/spec/guides/index.md` 命中；新文件非空且四要素齐备
- **回滚点 R4**：删新文件并还原 index.md

---

## Phase 5: 质量门（全量回归）

- [ ] 5.1 `make lint` → `0 issues.`
- [ ] 5.2 `GOWORK=off go vet ./...` → exit=0
- [ ] 5.3 `GOWORK=off go test ./...` → 全包 ok，无失败
- [ ] 5.4 `cd web && npm run type-check` → exit=0
- [ ] 5.5 `cd web && npm run build` → 成功且**仍零 churn**
- [ ] 5.6 **四项数量终测**与 Phase 0 基线对照：
      `deadcode`=7、`unused`=0、前端零引用=0、`deps`=23
- [ ] 5.7 实例未受影响：`docker compose ps` Up + `/api/health` 200
- [ ] 5.8 **越界检查**：`refresh-auth` 路由、`OfflineHandoffClientID`、`SourceTypeOfflineHandoff`、`drivers/template` 均**零改动**

---

## Phase 6: 归档收尾

- [ ] 6.1 `skill trellis-check` 走查（覆盖核对 / spec 同步 / 范围纪律 / 跨层一致性）
- [ ] 6.2 `flow_gate.py mark-check 09-12-cleanup-rescan-findings --note "<质量门+四项数量摘要>"`
- [ ] 6.3 勾选 `prd.md` 全部验收项
- [ ] 6.4 `flow_gate.py pre-archive` → `task.py archive ... --skip-branch-validation`
- [ ] 6.5 提交实质改动（含 embed 若有）→ `add_session.py` → `git push`

---

## Validation Commands（汇总）

```bash
export PATH=/usr/local/go/bin:/root/go/bin:$PATH
cd /root/LitePan

# 数量判据（删除前 vs 删除后）
deadcode ./cmd/litepan                                   # 9 → 7
golangci-lint run --enable=unused -c .golangci.yml ./... # 14 → 0

# 前端
ls web/src/composables/useTimeWindowSchedule.ts          # 必须存在
grep -n "@vue/devtools-api" web/package.json             # 必须存在
cd web && npm run type-check && npm run build
cd .. && git status --short internal/api/web             # 必须为空（零 churn）

# 质量门
make lint && GOWORK=off go vet ./... && GOWORK=off go test ./...

# 越界检查
git status --short | grep -E "manager.go|types.go|drivers/template|router.go" || echo "无越界 ✅"
```

## Review Gates

| 门 | 位置 | 判据 |
|---|---|---|
| G1 | Phase 1.2 | build/vet/test 通过 + `deadcode` **= 7** |
| G3 | Phase 2.2 | automation 包测试通过 + `unused` **= 0** |
| G2 | Phase 3.7 | **embed 零 churn**（唯一能反证前端判定的客观证据） |
| G4 | Phase 4.3 | spec 指南存在且四要素齐备 + 索引已登记 |
| G5 | Phase 5 | 全量质量门全绿 + 四项数量精确吻合 + 无越界 |

## Out of Scope

- 版本 bump 与发版（见 design.md KD3）
- P3 待定项（孤儿端点 `/accounts/{id}/refresh-auth`）
- P4 决策项（`drivers/template` + httpx OAuth 链路）
- `OfflineHandoffClientID`（删除会牵动 `manager_test.go` 的 3 处调用，属功能判断）
- 把 `unused` 写入 `.golangci.yml`
