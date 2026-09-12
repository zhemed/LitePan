# Design: 完全移除 API 秘钥功能

## Overview

一次**六层贯穿的删除**：前端界面 → HTTP 路由/handler → 领域类型与仓储接口 → 存储实现 → 数据表（新迁移）→ 备份链路。

与上轮"死代码清理"的本质区别：**这次删的东西是可达的**（后台 tab 点得到、5 条 API 路由调得通），所以**没有任何工具能提示风险** —— `deadcode`/`unused` 全绿。安全性完全依赖两件事：

1. **删前实证"没有消费侧"**（已做，见 PRD Background）
2. **删后验证"没删到近名物"**（本设计的核心防线）

## Boundaries

| 层 | 删 | **明令不动**（近名陷阱） |
|---|---|---|
| **前端界面** | `ApiKeySettings.vue`(517 行)、`apiKeys.ts`；`AdminView.vue` 的 tab 标签；`SystemSettings.vue` 的 import/tab/toolbar/模板 | `DynamicForm.vue:72` 的 `"api_key"` —— 它是**通用敏感字段关键词表**（与 `token`/`password`/`secret`/`bearer` 并列），用于表单掩码启发式 |
| **前端文案** | `BackupRestorePanel.vue` 帮助文案里的「API 密钥、」 | 同文件的 `LITEPAN_SECRET_KEY` 提示（讲环境变量密钥，与本功能无关） |
| **HTTP 层** | `internal/api/api_keys.go` 整文件；`router.go` 的 `Deps.ApiKeys`/`Handler.apiKeys`/5 条路由/import | router 其余路由与中间件 |
| **领域层** | `internal/domain/api_key.go` 整文件（类型 + `ApiKeyRepository` 接口） | 其它 domain 类型 |
| **存储层** | `internal/store/api_key_repo.go` 整文件；`store.go` 的 bundle 字段与初始化；`backup.go` 的 `UPDATE api_keys …` **一条语句** | `backup.go` 同函数内其余 sanitize 语句（`upload_tasks`/`notifications`/`automation_runs`/`fuse_mounts`/`automation_rules`/`configs`） |
| **服务包** | `internal/apikey/` 整包（`service.go` + `keyutil.go`） | — |
| **自动化接线** | `automation/service.go` 的 `apiKeys` 字段 + `SetApiKeys` + 构造赋值 | automation 其余逻辑 |
| **备份链路** | `backuprestore/service.go` components 列表中的 `"api_keys"` | 同列表的 `settings`/`accounts`/`credentials`/`tasks`/`favorites`/`secret_key`（**且保持顺序**） |
| **数据库** | 新增迁移 `0024_drop_api_keys.sql` | **历史迁移一律不动**（尤其 `0022`）；`schema_migrations` 机制不动 |
| **构建/文档** | 无 | `Dockerfile`、`.golangci.yml`、`README.md`、`docker-compose*.yml`、`buildinfo/version.go` |

## Data Flow（删除后的形态）

```
删除前                                   删除后
─────────                                ─────────
后台「其他设置」→「API 秘钥」tab          该 tab 不存在
  └─ ApiKeySettings.vue (517 行)              （组件删除）
      └─ apiKeys.ts → /api/admin/api-keys      （客户端删除）
          └─ internal/api/api_keys.go          （handler 删除）
              └─ apikey.Service (CRUD)         （整包删除）
                  └─ domain.ApiKey             （领域类型删除）
                      └─ store/apiKeyRepo      （仓储删除）
                          └─ api_keys 表       （0024 迁移 DROP）
automation.apiKeys 字段（只写不读）            字段与 SetApiKeys 删除
backup.sanitize: UPDATE api_keys …             语句删除
backup.components: [..., "api_keys", ...]      列表项删除
```

**没有任何消费侧被牵连** —— 因为本来就没有消费侧（无入站鉴权中间件、`GetByHash`/`TouchLastUsed` 零调用者）。

## Key Decisions

### KD1 为什么连数据表一起 DROP

- **有明确先例**：`0022_drop_removed_feature_tables.sql` 就是为已删功能删表（STRM、媒体整理、缓存保留、离线下载、夸克TV），风格是幂等 `DROP TABLE IF EXISTS`
- **数据本身无价值**：表里存的是**没有任何接口会校验**的秘钥；留着它等于留一个永久误导后来者的表（"有 api_keys 表 ⇒ 大概有 API 鉴权"）
- **可逆性可接受**：迁移只 DROP 不删文件，DB 有备份链路；且本机实例 `api_keys` **0 行**
- **只新增不改历史**：不动 `0022` 等既有迁移（项目既有结论：改历史会破坏已部署实例的升级路径与 `schema_migrations` 一致性）

### KD2 旧备份的兼容性（已核查，**不会坏**）

`buildFullSources` 用 `SnapshotTo` 导出**整库快照**，因此**旧备份里含 `api_keys` 表**。恢复路径实测为：

```
PrepareRestore → prepareStagedDatabase → staged.Migrate(ctx)   ← service.go:428
restoreRollback                        → db.Migrate(ctx)       ← pending.go:223
```

即**恢复时会对还原出来的库重跑迁移**，所以：
- 恢复旧备份 → 该库的 `schema_migrations` 只到 `0023` → **`0024` 再次执行 → 孤儿表被自动清掉** ✅
- 恢复新备份 → 本就不含该表

**结论**：新旧备份都仍可正常恢复，无需额外兼容代码。

### KD3 删除顺序必须"先摘接线、后删包"（编译安全）

若先删 `internal/apikey/`，则 `router.go`/`wire_http.go`/`automation` 立刻编译失败，中途无法验证。故顺序固定为：

```
① 前端删除（独立，不影响 Go 编译）
② HTTP 层摘除（删 handler + 路由 + Deps/Handler 字段）
③ 接线摘除（wire_http 构造、automation 字段与方法）
④ 领域 + 存储删除（domain 类型、仓储实现、bundle 字段、backup 语句、backuprestore 列表）
⑤ 最后删 internal/apikey/ 整包
⑥ 新增迁移
⑦ 重建 embed + 全量质量门
```

每步后跑 `go build ./...`，保证**任一步失败都能停在一个可编译的状态**。

### KD4 不做版本 bump（与上轮一致），但本轮更够格发版

- 不为版本号而 bump：一旦写成 `v0.0.45` 却不推镜像，`README`/`compose` 就会指向不存在的镜像 —— 正是本仓库刚修好的那类不一致
- 但**如实标注差异**：上轮删的是纯死代码（零行为变化），**本轮删的是用户可见功能**（后台少一个 tab）→ 它比上轮更够格发版
- **是否发版交用户决定**；要发版则另开任务走完整流程（build → GHCR 三 tag → tag → release → 本地容器验证）

### KD5 运行中实例的行为要说清

- 容器**不重建**，本轮改动**不影响正在运行的 `:5211`**
- 新迁移会在**下次启动时**执行（`store.Migrate`）→ 届时不存在的 `api_keys` 表被 `DROP TABLE IF EXISTS` 安全跳过/清理
- 故验收项写的是"实例 health 仍 200"，而非"迁移已生效"

## Compatibility

- **前后端同镜像发布**：新旧不混用，故不存在"新前端配旧后端"的字段缺失问题
- **API 兼容**：`/api/admin/api-keys` 5 条路由消失 → 对任何外部脚本是 **breaking**。但已核实**无消费侧**（无中间件、无调用者），且秘钥本身不生效，故无可断的集成
- **备份/恢复**：见 KD2，新旧备份均可恢复
- **数据库**：`DROP TABLE IF EXISTS` 幂等，重复执行安全
- **`SanitizePortableBackup`**：删掉一条语句后其余 6 条不变 → 便携备份的语义不受影响

## Tradeoffs

- **删表 vs 只删代码留空表**：删表更干净（不留误导），代价是**不可逆的数据丢失** —— 但该数据无消费侧、本机为空，且项目有先例
- **删整包 vs 保留包只删路由**：整包删更彻底，代价是若将来要接回上游的开放 API 能力需重新引入（但从上游 cp 回来并不难，且届时形态应重新设计）
- **不动 `DynamicForm` 关键词**：保留一个"永远不会匹配到实际字段"的关键词，代价是极小的冗余，换来**绝不误伤通用掩码逻辑**

## Rollout / Rollback

**Rollout**：按 KD3 的七步顺序，每步可编译、可验证。

**Rollback**：

| 项 | 回滚 |
|---|---|
| 代码（全部） | `git revert <commit>`（或 `git checkout <prev> -- <paths>`）—— 四个删除文件、六处接线均可还原 |
| 数据表 | `git revert` 只是还原迁移文件；**已执行的 DROP 不会自动回滚** → 需从备份恢复（本机为空表，无实际损失） |
| 前端 embed | `git checkout HEAD -- internal/api/web` 后重建 |
| 迁移已执行 | 无"反向迁移"。若确需恢复表结构，从旧备份恢复即可（该备份含整库快照） |

**触发回滚的硬条件**：任一步 `go build`/`go vet` 失败、质量门不绿、或发现误删近名物（如 `DynamicForm` 的关键词被删）。

## File Map

| 路径 | 动作 |
|---|---|
| `internal/apikey/service.go`、`keyutil.go` | **删除**（整包） |
| `internal/api/api_keys.go` | **删除** |
| `internal/domain/api_key.go` | **删除** |
| `internal/store/api_key_repo.go` | **删除** |
| `web/src/components/admin/ApiKeySettings.vue` | **删除**（517 行） |
| `web/src/api/apiKeys.ts` | **删除** |
| `internal/api/router.go` | 改（Deps/Handler/5 路由/import） |
| `internal/app/wire_http.go` | 改（构造/SetApiKeys/传参/import） |
| `internal/automation/service.go` | 改（字段/方法/赋值/import） |
| `internal/store/store.go` | 改（bundle 字段 + 初始化） |
| `internal/store/backup.go` | 改（删 1 条 sanitize 语句） |
| `internal/backuprestore/service.go` | 改（components 列表去 1 项） |
| `internal/store/migrations/0024_drop_api_keys.sql` | **新增** |
| `web/src/views/AdminView.vue`、`web/src/components/admin/SystemSettings.vue` | 改（摘 tab 入口） |
| `web/src/components/admin/BackupRestorePanel.vue` | 改（文案去 1 词） |
| `internal/api/web/**` | 重建（预期 churn） |
| `web/src/components/form/DynamicForm.vue` | **零改动**（关键防线） |
| `Dockerfile`、`.golangci.yml`、`README.md`、`docker-compose*.yml`、`buildinfo/version.go` | **零改动** |
