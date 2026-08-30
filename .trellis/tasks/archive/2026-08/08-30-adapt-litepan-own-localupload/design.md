# Design: Adapt LitePan-own local_upload to LitePan

## Overview

`LitePan-own` 的 `local_upload` 为自动化侧全量增量，需将 `domain` 常量、`service` 依赖、`runLocalUpload` 核心、`validate` 分支、`frontend` 类型与面板 5 处同步移植到 `zhemed/LitePan` 的 `three-drivers` 基座（当前仅 `delay`）。

## Boundaries

| 层 | 移植 | 不移植 |
|---|---|---|
| **Domain** | `automation.go: AutomationActionLocalUpload = "local_upload"` | `organize/strm/cache_clear/emby` 等已删 |
| **Service Options** | `Settings *settings.Service`、`DataDir string`、`Uploads *upload.Manager` | `organize/strm` 等 |
| **Run** | `service_run.go: runLocalUpload + fileHash + load/saveState`（185 行，含 `B mode`） | `runOrganize/runStrm/runCacheClear` 等 |
| **Validate** | `service_validate.go: case LocalUpload` 的 `mapping/mappings/account_id` 校验 | `emby` 等 |
| **Frontend API** | `automation.ts: AutomationActionType += "local_upload"` | `EmbyRefreshMode` 等已删 |
| **Frontend Panel** | `AutomationPanel.vue: 本地上传` 三选（`mapping/account/target/conflict`） | `ProxyToolsPanel` 等已删 |

## Data Flow

```
AutomationPanel (本地上传: mappings/account/target) → POST /admin/automation/validate → service_validate.go: case LocalUpload 校验
→ POST /admin/automation/rules → service_run.go: executeAction: case LocalUpload → runLocalUpload()
  → loadLocalUploadState(dataDir, mapping) → fileHash(relPath) → cloud existence check (s.files.List) → batch Create (s.uploads.CreateBatch 100) → saveLocalUploadState
```

## Compatibility

- `s.dataDir` 为 `cfg.DataDir`（`./data`），`local_upload_state_*.json` 落盘与 `zhemed/LitePan` 的 `data/litepan.db` 同级，`gitignore` 的 `*.json` 不忽略（需确保 `local_upload_state_*.json` 被 `/.gitignore` 的 `*.db` 等不误伤，或显式加入 `!local_upload_state_*.json` 排除）
- `s.files.List/CreateFolder` 与 `s.uploads.CreateBatch` 已在 `zhemed/LitePan` 存在，无需新增
- `settings.KeyLocalUploadMappings` 已在 `aux-keep-upload` 保留，`runLocalUpload` 复用

## Tradeoffs

- **全量移植 vs 最小移植**：全量 `185 行` 含 `B mode` 与 `cloud existence` 双重增量，虽大但与 `LitePan-own` 已验的 `115G 4分钟` 能力一致，故全量移植；`115_Open OSS 512M` 等驱动优化不移植，避免与 `three-drivers` 的当前驱动状态冲突
- **Frontend 复用**：`LitePan-own` 的 `AutomationPanel.vue` 的 `本地上传` 片段与 `zhemed/LitePan` 的 `AutomationPanel.vue`（已删 emby）结构接近，可直接 patch `mapping` 部分

## Rollout / Rollback

- 单提交 `feat(automation): adapt local_upload from LitePan-own`，`git revert` 即回退
- 容器：`docker build -t litepan-go:localupload 118M→~120M`，`POST /api/admin/automation/validate` 的 `local_upload` 动作返回 `ok`

## File Map

1. `internal/domain/automation.go`
2. `internal/automation/service.go` + `wire_services.go`
3. `internal/automation/service_run.go`
4. `internal/automation/service_validate.go`
5. `web/src/api/automation.ts`
6. `web/src/components/admin/AutomationPanel.vue`
