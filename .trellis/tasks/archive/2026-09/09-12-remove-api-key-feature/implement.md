# Implementation Plan: 完全移除 API 秘钥功能

## Overview

按 design.md **KD3 的固定顺序**执行：每步后 `go build ./...`，保证任一步失败都停在一个**可编译**的状态。

顺序不是随意的 —— **必须先摘接线、后删包**，否则删包瞬间三处接线编译失败，中途无法分辨"是我删错了"还是"还没删完"。

本机 `danger-full-access`、审批关闭 —— 不请求 escalation。

---

## Phase 0: 基线快照

- [x] 0.1 `git status --short` 记录基线（期望仅本任务目录）
- [x] 0.2 记录死代码基线：`deadcode ./cmd/litepan` = **7**、`unused` = **0**（改动后必须不劣于）
- [x] 0.3 记录前端基线：零引用文件 = **0**、`dependencies` = **23**
- [x] 0.4 记录待删清单核验：四个后端文件/包 + 两个前端文件**确实存在**
- [x] 0.5 **锁定近名物**（改动后要复核）：`DynamicForm.vue:72` 的 `"api_key"` 行、`backup.go` 的 6 条其它 sanitize 语句、`backuprestore` components 列表其余 6 项 —— 逐一留底

**回滚点 R0**：以上基线即后续比对依据

---

## Phase 1: 前端删除与摘除（独立，不影响 Go 编译）

- [x] 1.1 `git rm web/src/components/admin/ApiKeySettings.vue web/src/api/apiKeys.ts`
- [x] 1.2 `AdminView.vue`：移除 `"api-keys": "API 秘钥"` tab 标签
- [x] 1.3 `SystemSettings.vue`：移除 `ApiKeySettings` import、`API_KEYS_TAB` 常量与分支、`apiKeySettingsRef`/`apiKeyToolbar`/`apiKeyAddDisabled` 等状态、以及模板中所有相关用法
- [x] 1.4 `BackupRestorePanel.vue`：帮助文案去掉「API 密钥、」
- [x] 1.5 **验证门 G1**：`cd web && npm run type-check` **exit=0**（类型系统会抓出任何残留引用）
- **回滚点 R1**：`git checkout HEAD -- web/src`

---

## Phase 2: HTTP 层摘除

- [x] 2.1 `git rm internal/api/api_keys.go`（5 个 handler）
- [x] 2.2 `internal/api/router.go`：删 `Deps.ApiKeys`、`Handler.apiKeys`、`NewRouter` 赋值、`/api-keys` 路由组 5 条、`litepan/internal/apikey` import
- [x] 2.3 **验证门 G2**：`GOWORK=off go build ./...` 通过
- **回滚点 R2**：`git checkout HEAD -- internal/api`

---

## Phase 3: 接线摘除（wire_http + automation）

- [x] 3.1 `internal/app/wire_http.go`：删 `apiKeySvc` 构造块、`svc.automation.SetApiKeys(apiKeySvc)`、`Deps{ApiKeys: …}` 传参、`litepan/internal/apikey` import
- [x] 3.2 `internal/automation/service.go`：删 `apiKeys` 字段、`SetApiKeys` 方法、构造里的赋值、`litepan/internal/apikey` import
- [x] 3.3 **验证门 G3**：`GOWORK=off go build ./...` 通过
- **回滚点 R3**：`git checkout HEAD -- internal/app internal/automation`

---

## Phase 4: 领域层 + 存储层删除

- [x] 4.1 `git rm internal/domain/api_key.go internal/store/api_key_repo.go`
- [x] 4.2 `internal/store/store.go`：删 `ApiKeys` 字段与 `&apiKeyRepo{db: db}` 初始化
- [x] 4.3 `internal/store/backup.go`：**只删** `UPDATE api_keys SET last_used_at=NULL` 这一条语句
- [x] 4.4 `internal/backuprestore/service.go`：components 列表去掉 `"api_keys"`（其余 6 项**顺序不变**）
- [x] 4.5 **验证门 G4**：`GOWORK=off go build ./...` 通过；且 `grep -c 'UPDATE \|DELETE FROM ' internal/store/backup.go` 语句数**恰好少 1**
- **回滚点 R4**：`git checkout HEAD -- internal/domain internal/store internal/backuprestore`

---

## Phase 5: 删除 apikey 包（最后）

- [x] 5.1 `git rm -r internal/apikey/`
- [x] 5.2 **验证门 G5**：`GOWORK=off go build ./...` 与 `GOWORK=off go vet ./...` 均通过
- **回滚点 R5**：`git checkout HEAD -- internal/apikey`

---

## Phase 6: 数据表迁移（新增，不改历史）

- [x] 6.1 新增 `internal/store/migrations/0024_drop_api_keys.sql`，内容：
      ```sql
      -- 移除 API 秘钥功能（2026-09-12）：该功能只有管理侧、无消费侧，已整体移除。
      -- 幂等：DROP TABLE IF EXISTS，可重复执行。
      DROP TABLE IF EXISTS api_keys;
      ```
- [x] 6.2 确认迁移编号与既有最大编号不冲突（既有最大 = `0023`）
- [x] 6.3 **验证门 G6a（迁移实测）**：用**临时库**跑迁移（不碰实例库）→ `api_keys` 表**不存在**；且**重复执行不报错**
- [x] 6.4 **验证门 G6b**：`internal/store/migrations/0022_drop_removed_feature_tables.sql` **零改动**（`git status` 无该文件）
- **回滚点 R6**：删除新迁移文件

---

## Phase 7: 重建 embed + 全量质量门

- [x] 7.1 `cd web && npm run build`（**预期有 churn** —— 被删组件原本在 bundle 中，与上轮"零 churn"相反，这是正常的）
- [x] 7.2 确认 churn 内容合理：`git status --short internal/api/web | wc -l` > 0，且**含被删组件的 chunk**
- [x] 7.3 **验证门 G7（全量）**：
      - `make lint` → `0 issues.`
      - `GOWORK=off go vet ./...` → exit=0
      - `GOWORK=off go test ./...` → 全包 ok
      - `cd web && npm run type-check` → exit=0
- [x] 7.4 **近名物复核（本任务最重要的一道）**：
      - `grep -n '"api_key"' web/src/components/form/DynamicForm.vue` → **仍在**
      - `internal/store/backup.go` 其余 6 条 sanitize 语句**逐条仍在**
      - `backuprestore/service.go` components 其余 6 项**仍在且顺序不变**
      - 上述任一被误删 → **立即回滚该文件**
- [x] 7.5 **残余引用扫描**：`grep -rn "apikey\|ApiKey" --include=*.go .`（排除本任务文档）→ 应为**空**；`grep -rn "api_keys"` → 仅剩新迁移与历史迁移
- [x] 7.6 **基线复核**：`deadcode` **不高于 7**、`unused` **= 0**、前端零引用 **= 0**
- [x] 7.7 **越界检查**：`Dockerfile`、`.golangci.yml`、`README.md`、`docker-compose*.yml`、`buildinfo/version.go` **零改动**
- [x] 7.8 实例未受影响：容器**不重建**，`/api/health` 仍 200

---

## Phase 8: 归档收尾

- [x] 8.1 `skill trellis-check` 走查
- [x] 8.2 `flow_gate.py mark-check 09-12-remove-api-key-feature --note "<质量门+近名物复核摘要>"`
- [x] 8.3 勾选 `prd.md` 全部验收项
- [x] 8.4 `flow_gate.py pre-archive` → `task.py archive … --skip-branch-validation`
- [x] 8.5 提交 → `add_session.py` → `git push`

---

## Validation Commands（汇总）

```bash
export PATH=/usr/local/go/bin:/root/go/bin:$PATH
cd /root/LitePan

# 每步之后
GOWORK=off go build ./...

# 迁移实测（临时库，不碰实例）
t=$(mktemp -d); # 用 cmd 启动一次或直接跑 store 迁移测试 → 断言 api_keys 不存在、重复执行安全

# 近名物复核（关键）
grep -n '"api_key"' web/src/components/form/DynamicForm.vue
grep -c 'UPDATE \|DELETE FROM ' internal/store/backup.go

# 残余扫描
grep -rn "apikey\|ApiKey" --include=*.go . | grep -v "^./.trellis"
grep -rn "api_keys" --include=* . 2>/dev/null | grep -v node_modules | grep -v "/tasks/" | grep -v migrations

# 全量质量门
make lint && GOWORK=off go vet ./... && GOWORK=off go test ./...
cd web && npm run type-check && npm run build
```

## Review Gates

| 门 | 位置 | 判据 |
|---|---|---|
| G1 | Phase 1.5 | 前端 `vue-tsc -b` exit=0（抓残留引用） |
| G2–G5 | Phase 2–5 | 每步 `go build ./...` 通过（保证可编译状态） |
| G6a | Phase 6.3 | 临时库迁移后 `api_keys` 不存在 + 重复执行安全 |
| G6b | Phase 6.4 | `0022` 及全部历史迁移零改动 |
| G7 | Phase 7.3–7.7 | 全量质量门 + **近名物复核** + 残余扫描 + 基线不劣化 + 无越界 |

## Rollback

见 design.md → **Rollout / Rollback**。要点：代码层全部可 `git revert`；**已执行的 DROP 不可自动回滚**（本机表为空，无实际损失；确需恢复可从含整库快照的旧备份还原）。

## Out of Scope

- 发版（版本 bump、GHCR 推送、tag、release）—— 交用户决定后另开任务
- 回填 `dead-code-guide.md`（"功能级死代码"这一类的识别方法）
- `offline_handoff` 命名漂移、P4 的 template 链路
- 任何"替代 API 鉴权方案"的新功能设计

---

## 执行记录（实施后回填，仅记录事实与偏差）

**计划与实际一致的验证门**：G1（`vue-tsc -b` exit=0）、G2–G5（每步 `go build ./...` 通过；Phase 3 处观察到预期的中间态报错 `unknown field ApiKeys in struct literal of type api.Deps`，修完即通过）、G6a/G6b、G7 全部通过。

**实际比计划多做的两件事（下文单列，非计划外扩张）**：

1. **迁移实测从"临时库一次"扩到四场景 + 真实启动**（计划 6.3 只要求临时库一次）：
   - ① 实例库**只读副本**：迁移后 `api_keys` 与其索引消失，其余各表行数**逐表零变化**；
   - ② **幂等重跑**：同一库再跑 `Migrate` 无报错，版本仍 24；
   - ③ **旧备份重放**：把 `schema_migrations` 回退到 23 并重建 `api_keys`（含 1 行数据）后重跑 → 表与索引再次被清除（即"含旧表的备份恢复"路径成立）；
   - ④ **全新库**：1→24 台账完整，终态无 `api_keys`；
   - ⑤ **真实启动**：`go build -tags fuse` 起临时实例（`127.0.0.1:5311` + 数据副本），启动日志正常、库自动升到 version 24、`configs` 7 行未损。
2. **浏览器验收**（`.trellis/spec/web/frontend/quality-guidelines.md` 要求：改动触及 `web/src/**` 且效果只在渲染后可见 —— 本任务删了页签与条件分支，命中该条款）：在临时实例上 `bw` 实测 —— 设置页仅剩 **3 个页签**（账号安全/首页设置/其他设置），「保存改动」按钮与卡片布局完好；`首页设置`/`其他设置`（key=`services`）切页正常；备份管理弹窗文案已是「管理员登录信息、网盘账号数据等敏感信息」（无「API 密钥」）；SPA 内程序化切换三个页签期间 `window.__err` 为空（0 个 JS 错误）；截图目视确认无空白缺口。验收后临时实例、临时目录、临时二进制、仓库内截图**全部清理**，线上容器未动（`/api/health` 200）。
   - 附：构建产物交叉验证 —— HEAD 的 `SystemSettings-*.js.gz` 含 6×「API 秘钥」/19×`api-keys`、`AdminView-*.js.gz` 含 1×「API 秘钥」，新产物对应 chunk **命中数均为 0**。

**偏差 1（内容层面，无害）**：实现里 `0024_drop_api_keys.sql` 的**注释文字**比 6.1 的示意稿更详细（写明"后台页 + 5 个接口 + `internal/apikey` 包 + 仓储接口"的删除范围），SQL 主体与计划完全一致（仅 `DROP TABLE IF EXISTS api_keys;`），风格对齐 `0022`。

**偏差 2（计划外但必要：spec 同步）**：`trellis-check` 的 Spec Sync 要求本次一并执行，共改 7 个 spec 文件，全部是**指向已删文件/已不存在符号的引用修正**（不改任何规则语义）：
`api-layering.md`（鉴权描述由 `Authorization: Bearer`/`X-API-Key` 改为实测的**仅 Cookie 会话**）、`database-guidelines.md`（表清单改为实测表；仓储示例 `apiKeyRepo`→存活示例）、`directory-structure.md`（目录树去掉 `apikey/`）、`error-handling.md`（引用文件与函数名改为存活项）、`api-client.md`（`web/src/api/*.ts` 清单改为实测 19 个文件）、`component-guidelines.md`（`AdminView.vue` 一行改为实测 4 页签）、`dead-code-guide.md`（示例目标改为通用写法 + 历史案例）。

**偏差 3（发现但故意不修，登记为待办）**：`error-handling.md`、`api-layering.md`、`api-client.md` 中仍有多处**既有僵尸内容** —— `writeDomainError`、`internal/api/errors.go`、`CodeInvalid/CodeConflict/CodeUnauthorized/CodeForbidden` 在当前代码中**均不存在**（实测出口为 `internal/api/resp.go: writeErr`，HTTP 码由 `internal/domain/errors.go: codeTable + AppError.HTTPStatus()` 决定，响应字段是 `error_type`）。按"只修与已删功能相关的引用、不顺手扩大范围"的纪律**未重写**这些段落，仅在本任务触及的那一行下方加了 ⚠️ 标注，留给单独的 spec 同步任务。

**不做永久回归测试的理由**：本轮是纯删除，无新函数、无行为分支；编译期即保证 `internal/apikey` 不可能再被引用，迁移守卫由 `newTestStore`（每次 `go test` 都跑全量 `Migrate`）隐含覆盖 —— 与先例 `0022_drop_removed_feature_tables.sql` 一致（该迁移亦无专属测试）。
