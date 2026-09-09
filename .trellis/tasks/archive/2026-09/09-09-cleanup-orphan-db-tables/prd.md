# 09-09-cleanup-orphan-db-tables

## Goal

清理 `data/litepan.db` 中已删功能的孤儿表与孤儿配置键。以迁移方式实施（新迁移 `0022`），随 `0.0.15` 发布自动落地，保证现有库与全新安装收敛一致。

## 取证分类（逐表对照代码）

**确认 DROP（7 张，零活引用或纯死代码）**：

| 表 | 代码引用 | 结论 |
|---|---|---|
| `strm_tasks` / `strm_branches` / `strm_remote_dir_cache` | 0 处 | 纯孤儿 |
| `offline_download_tasks` | 0 处 | 纯孤儿 |
| `media_organize_tasks` | 仅 `store/backup.go` 恢复清洗语句 | 删表 + 同步删该引用 |
| `cache_retention_tasks` | 仅 `store/backup.go` 恢复清洗语句+计数 | 同上 |
| `quarktv_bindings` | 仅死代码：`store/quarktv_binding_repo.go`（无任何消费方）+ `store.go` 字段 + `domain/quarktv.go` + `domain/notification.go` 一个无消费常量 | 删表 + 删死代码 |

**明确保留（复核为活功能）**：`notifications`（6 条路由+service 全接线，站内通知非公告）、`api_keys`（全接线）、`fuse_mounts`（fusemount 全接线）、`account_auth_states`/`automation_*`/`upload_tasks`/`cloud_accounts`/`configs`/`schema_migrations`。

**孤儿配置键**：`strm_base_url`、`strm_token`（STRM 残留）→ 迁移内 DELETE；`public_index_enabled`/`session_timeout`/`upload_task_concurrency`/`admin_*` 均活，保留。

## Requirements

1. 新迁移 `internal/store/migrations/0022_drop_removed_feature_tables.sql`：`DROP TABLE IF EXISTS` ×7 + 删除 2 个 STRM 残留键（启动时自动执行，现有库与全新库一致收敛）。
2. `internal/store/backup.go`：从 `SanitizePortableBackup` 语句与 `BackupCounts` 表清单移除 media_organize_tasks/cache_retention_tasks。
3. 删除死代码：`store/quarktv_binding_repo.go`、`domain/quarktv.go`、`store.go` 的 `QuarkTVBindings` 字段与装配、`domain/notification.go` 的 `NotificationCategoryQuarkTVWarn`。
4. 质量门：`GOWORK=off go vet ./...` + `go test ./internal/... ./drivers/...`（存量 `internal/file` 失败除外，已备案）。
5. 版本 `0.0.14 → 0.0.15`（README/compose×2），走标准发布管线；**部署前**再做一次 DB 备份。
6. 部署后验证：7 表 + 2 键消失、其余表完好、health ok、admin 登录、files/list 正常、启动日志无迁移报错。

## Constraints

- 不动 `notifications`/`api_keys`/`fuse_mounts`（活功能）。
- 不改历史迁移文件（0022 幂等 DROP，ledger 记账）。
- 备份先于一切写库动作；备份放 `data/backups/`（已 gitignore，不入 git）。

## Acceptance Criteria

- [ ] 0022 迁移随 0.0.15 启动执行，7 表 + 2 键消失，保留表完好
- [ ] vet/test 全绿（存量备案项除外）
- [ ] 三 tag 镜像推送、tag+release、容器运行 0.0.15
- [ ] 部署前 DB 备份落盘
- [ ] health/login/files-list 三连验证通过
- [ ] 归档 + journal
