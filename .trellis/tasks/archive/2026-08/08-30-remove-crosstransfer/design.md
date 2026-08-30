# Design: Remove cross-drive instant transfer

## Overview

跨盘秒传为垂直功能：`crosstransfer.Service` 负责指纹秒传、扫描、试探、执行，`internal/api/cross_transfer_admin.go` 暴露 5 个 `NDJSON` 流接口，`web` 的 `CrossDriveTransfer.vue` 提供跨盘 UI。删除需自底向上，保持 `upload` 普通上传不受影响。

## Boundaries

| 层 | 删除 | 保留 |
|---|---|---|
| **服务** | `internal/crosstransfer/*` 4 文件 | `internal/upload` 的 `SourceTypeCrossTransfer` 常量（历史兼容，仅注释） |
| **API** | `internal/api/cross_transfer_admin.go` + `router.go` 的 `Deps/CrossTransfer`、`Handler.crossTransfer`、`r.Route("/cross-transfer", 5 handler)` + `import crosstransfer` | `internal/api/local_upload.go` 等非跨盘 |
| **App** | `internal/app/wire_services.go` 的 `crossTransfer` 字段与 `crosstransfer.New`、`wire_http.go` 的 `Deps CrossTransfer` | `wire_services` 的 `favorites/file/playback` 等 |
| **Driver** | 无（`ProvideHashes` 保留，驱动声明） | `internal/driver/driver.go` 的 `ProvideHashes/RapidUploadHashes` |
| **前端** | `web/src/api/crossTransfer.ts`、`CrossDriveTransfer.vue`、`CrossTransferTree.vue`、`CrossTransferProbeNoticeDialog.vue` | `web/src/api/cloudTools.ts` 的 `localUploadApi` 等 |

## Data Flow Removal

```
Before: web CrossDriveTransfer → POST /cross-transfer/scan → cross_transfer_admin.go → crosstransfer.Service.ScanSources → driver RapidUpload
After: 404 (chi 404)

Before: web CrossDriveTransfer → POST /cross-transfer/execute → crosstransfer.Service.ExecuteStream → upload.Manager.CreateWithSource
After: 404
```

- `upload.Manager` 仍接收 `SourceTypeCrossTransfer` 的历史值，但不再有调用方；保留常量避免 `store` 中旧 `upload_tasks` 的 `source_type` 解析失败

## Compatibility

- **DB**：无 `crosstransfer` 专有表，`upload_tasks` 的 `source_type` 仍存 `cross_transfer` 字符串，查询时保留
- **Config**：无专有配置键
- **Driver**：`ProvideHashes` 保留，驱动仍可声明秒传能力，但无消费方

## Tradeoffs

- **彻底删 vs 隐藏**：彻底删减镜像与维护成本，符合用户“彻底移除”
- **保留 `SourceTypeCrossTransfer` 常量**：避免旧 DB 行解析 `source_type` 时 `unknown` 导致 `upload` 列表失败

## Rollout / Rollback

- 单提交 `refactor(crosstransfer): remove cross-drive instant transfer`
- `git revert` 即恢复

## File Map (Deletion Order)

1. `internal/crosstransfer`、`internal/api/cross_transfer_admin.go`
2. `internal/app/wire_services.go`、`wire_http.go`、`internal/api/router.go`
3. `web/src/api/crossTransfer.ts`、`CrossDriveTransfer.vue` 等
4. `grep` sweep + `go vet` + `web build` + `docker build`
