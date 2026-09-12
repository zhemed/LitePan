# 完全移除 API 秘钥功能（前端 + 后端 + 数据表 + 接线）

## Goal

用户确认本项目**完全不用 API 秘钥**，移除整个功能。

核实结论（2026-09-12）：该功能**只有管理侧、没有消费侧** —— 能在后台增删改查秘钥，但**没有任何入站请求会校验它**。属上游 `Ponphil/LitePan` 的「开放 API / 第三方调用」残件：精简时调用侧被移除，管理侧被留下。

移除范围覆盖**四层**：前端界面、后端 API 与包、数据表、以及六处接线。

## Background（实测证据）

**为什么判定"无消费侧"**

| 证据 | 实测 |
|---|---|
| 入站鉴权中间件 | 全仓库搜 `Header.Get("Authorization"/"X-API-Key"/…)` → **零命中** |
| `GetByHash`（按哈希查秘钥，校验必经步骤） | 仅接口声明 + 仓储实现，**零调用者** |
| `TouchLastUsed`（记最后使用时间） | 同上，**零调用者** → `last_used_at` 永远是空的 |
| `automation.apiKeys` | `service.go:145` 是除声明/构造外**唯一引用** → 只赋值、**从不读取** |
| 实例 DB | `api_keys` 表 **0 行** |
| 界面文案自证 | `ApiKeySettings.vue:255`：「秘钥供**后续**自动联动等外部调用使用」——"后续"即尚未接线 |

**待移除对象全景**

| 层 | 路径 | 性质 |
|---|---|---|
| 后端包 | `internal/apikey/{service.go,keyutil.go}` | **整包删除** |
| 后端 handler | `internal/api/api_keys.go` | **整文件删除**（5 个 handler） |
| 领域层 | `internal/domain/api_key.go` | **整文件删除**（类型 + 仓储接口） |
| 存储层 | `internal/store/api_key_repo.go` | **整文件删除**（仓储实现） |
| 接线 | `internal/api/router.go` | `Deps.ApiKeys`、`Handler.apiKeys`、**5 条路由**、import |
| 接线 | `internal/app/wire_http.go` | `apiKeySvc` 构造、`SetApiKeys` 调用、Deps 传参 |
| 接线 | `internal/automation/service.go` | `apiKeys` 字段、`SetApiKeys`、构造赋值 |
| 接线 | `internal/store/store.go` | bundle 的 `ApiKeys` 字段与初始化 |
| 接线 | `internal/store/backup.go` | `SanitizePortableBackup` 里的 `UPDATE api_keys SET last_used_at=NULL` |
| 接线 | `internal/backuprestore/service.go` | 全量备份 components 列表中的 `"api_keys"` |
| 数据 | `api_keys` 表 | 新增迁移 `DROP TABLE IF EXISTS` |
| 前端组件 | `web/src/components/admin/ApiKeySettings.vue` | **整文件删除（517 行）** |
| 前端 API | `web/src/api/apiKeys.ts` | **整文件删除** |
| 前端接线 | `web/src/views/AdminView.vue` | tab 标签 `"api-keys": "API 秘钥"` |
| 前端接线 | `web/src/components/admin/SystemSettings.vue` | import、`API_KEYS_TAB` 分支、toolbar 状态、模板用法 |
| 前端文案 | `web/src/components/admin/BackupRestorePanel.vue` | 帮助文案中的「API 密钥、」（备份含敏感项举例） |
| embed | `internal/api/web/**` | 前端变更后重建 |

## Requirements

### D1 后端删除四个文件（整包/整文件）

- 删 `internal/apikey/`（整包：`service.go` + `keyutil.go`）
- 删 `internal/api/api_keys.go`
- 删 `internal/domain/api_key.go`
- 删 `internal/store/api_key_repo.go`

### D2 摘除六处接线（逐个改，不顺手扩大）

- `internal/api/router.go`：删 `Deps.ApiKeys` 字段、`Handler.apiKeys` 字段、`NewRouter` 赋值、`/api-keys` 路由组 5 条路由、`litepan/internal/apikey` import
- `internal/app/wire_http.go`：删 `apiKeySvc` 构造块、`svc.automation.SetApiKeys(apiKeySvc)` 调用、`Deps{ApiKeys: …}` 传参、`litepan/internal/apikey` import
- `internal/automation/service.go`：删 `apiKeys` 字段、`SetApiKeys` 方法、构造里的赋值、`litepan/internal/apikey` import
- `internal/store/store.go`：删 `ApiKeys` 字段与 `&apiKeyRepo{db: db}` 初始化
- `internal/store/backup.go`：删 `UPDATE api_keys SET last_used_at=NULL` 这一条语句（**只删这一条**，不动同函数其它 sanitize 语句）
- `internal/backuprestore/service.go`：全量备份 `components` 列表去掉 `"api_keys"`

### D3 数据表清理（新增迁移）

- 新增 `internal/store/migrations/0024_drop_api_keys.sql`，内容 `DROP TABLE IF EXISTS api_keys;`
- **沿用 `0022_drop_removed_feature_tables.sql` 的既有风格**（幂等、带注释说明）
- 须确认迁移被正确登记（`schema_migrations`），并验证**重复执行安全**

### D4 前端删除与摘除 + 重建 embed

- 删 `web/src/components/admin/ApiKeySettings.vue`、`web/src/api/apiKeys.ts`
- `AdminView.vue`：移除 `"api-keys": "API 秘钥"` tab 标签
- `SystemSettings.vue`：移除 import、`API_KEYS_TAB` 常量与分支、`apiKeySettingsRef`/`apiKeyToolbar`/`apiKeyAddDisabled` 等状态与所有模板用法
- `BackupRestorePanel.vue`：帮助文案去掉「API 密钥、」（保持与"已移除功能"一致）
- `cd web && npm run build` 重建 embed（**本轮预期会有 churn** —— 被删组件原本确实在 bundle 里）

## Constraints

- **不得删除与本功能无关的同名/近名内容**（死代码排查的教训）：
  - `web/src/components/form/DynamicForm.vue:72` 的 `"api_key"` 是**通用敏感字段关键词表**（与 `token`/`password`/`secret` 并列），用于表单字段掩码启发式 —— **绝不可删**
  - `BackupRestorePanel.vue` 的 `LITEPAN_SECRET_KEY` 提示与本功能无关，**保留**
  - `internal/store/api_key_repo.go` 之外的 storage 代码不动
- **迁移不得删除历史迁移**：`0022` 等历史文件是追加式记录，只**新增** `0024`，不改既有迁移（项目既有结论：删迁移会破坏已部署实例的升级路径与 `schema_migrations` 一致性）
- **不改 `.golangci.yml`**、不改 `Dockerfile`、不改 `README` 的驱动/功能列表之外的内容
- **不做版本 bump、不发版**（与上一轮清理一致）：`README.md`/`docker-compose*.yml` 的镜像 tag 保持 `v0.0.44`，与线上实例一致；若要发布另开任务。**注意**：本轮是**用户可见的功能移除**（后台少一个 tab），比上轮的纯死代码删除更够格发版 —— 是否发版由用户决定
- **不改运行中的实例容器**；迁移会在下次启动时自动执行（属预期行为，需在验收中说明）
- 质量门全绿：`make lint`、`go vet`、`go test`、`npm run type-check`、`npm run build`
- 不修改归档任务与历史 journal

## Acceptance Criteria

- [x] 四个后端文件/包**已删除**：`internal/apikey/`、`internal/api/api_keys.go`、`internal/domain/api_key.go`、`internal/store/api_key_repo.go`
- [x] 两个前端文件**已删除**：`ApiKeySettings.vue`、`apiKeys.ts`
- [x] **全仓库 `grep -rn "apikey\|ApiKey"` 仅剩允许保留项**（`DynamicForm.vue` 的敏感字段关键词 + 本任务文档）；`api_keys` 仅出现在新迁移文件与历史迁移中
- [x] `internal/api/router.go` 不再有 `/api-keys` 路由与 `ApiKeys` 字段；`wire_http.go` 不再构造 `apiKeySvc`
- [x] `internal/store/backup.go` 的 sanitize 语句列表**少了 `api_keys` 那条、其余不变**（其余语句逐条比对仍在）
- [x] `internal/backuprestore/service.go` 的 components 列表**不再含 `"api_keys"`**，其余项顺序不变
- [x] 新迁移 `0024_drop_api_keys.sql` 存在、内容为幂等 `DROP TABLE IF EXISTS api_keys;`；历史迁移 `0022` **零改动**
- [x] **迁移实测生效**：新建临时库跑迁移后 `api_keys` 表**不存在**；且**重复执行不报错**
- [x] `web/src/components/form/DynamicForm.vue` 的 `"api_key"` **未被误删**（该行仍与 `token`/`password` 等并列）
- [x] embed 已重建（`internal/api/web/**` 有 churn —— 被删组件原本在 bundle 中）
- [x] 质量门全绿：`make lint` 0 issues、`go vet` exit=0、`go test ./...` 全包 ok、`vue-tsc -b` exit=0、`npm run build` 成功
- [x] `deadcode ./cmd/litepan` **不高于**改动前（7），`unused` 仍为 **0**（不得引入新的不可达符号）
- [x] **未越界**：`Dockerfile`、`.golangci.yml`、`README.md`、`docker-compose*.yml`、`internal/buildinfo/version.go` 零改动
- [x] 运行中实例未受影响：`/api/health` 仍 200（容器不重建；新迁移在下次启动时才生效，属预期）

## Notes

- Scope 标注 `cross-layer`（前端 → API → 领域 → 存储 → 迁移 → 备份链路，六层贯穿），故按复杂任务补齐 `design.md` + `implement.md` 后再 `start`。
- 本任务**不**做：发版、写新功能替代、清理 `offline_handoff` 命名漂移、处理 P4（template 链路）。
- 与上一轮的关系：本任务是"管理侧存在、消费侧缺席"这类**功能级死代码**的处置（`deadcode`/`unused` 查不出来，因其 CRUD 路径从 UI 完全可达）；上轮 spec 新增的 `dead-code-guide.md` 明确覆盖不到这一类，故本条也值得回填进该指南（**本任务暂不做**，避免扩大范围）。
