# 彻底移除离线下载

## Goal

按 `report.md` 14+7 清单彻底移除离线下载相关所有内容（domain/store/driver/service/api/eventbus/automation/frontend），并发布 `0.0.6`。

## Background

- 报告结论：离线下载与 `local_upload` 零耦合，可彻底移除。

## Requirements

- **后端**：删 `internal/domain/offline_download.go`、`internal/store/offline_download_repo.go`+`store.go`+`backup.go`、`internal/driver/offline_download.go`、`drivers/*_Open/offline_download.go` 8 驱动、`internal/offlinedownload/`、`internal/api/offline_download.go`+`router.go`、`internal/eventbus/events.go OfflineDownloadCompleted`、`internal/automation service_offline_download.go`+`validate` 分支、`app/wire`。
- **前端**：删 `web/src/api/offlineDownload.ts`+`types/offline-download.ts`、`composables/useOfflineDownloads.ts`+`useUploadPanelActions offline`+`useUploadTaskStore offline`、`FileBrowser.vue` 离线相关。
- **版本**：`README v0.0.5→v0.0.6`，`docker 0.0.6`。

## Constraints

- 所有写操作在 `task.py start` 后，遵循 `AGENTS.md`。
- `GOWORK=off go vet` `type-check` 必须 0。

## Acceptance Criteria

- [ ] `grep OfflineDownload --include=*.go --exclude-dir=_extracted --exclude-dir=.trellis | wc -l ==0`
- [ ] `grep offline_download --include=*.ts --exclude-dir=_extracted | wc -l ==0`
- [ ] `GOWORK=off go vet 0` `type-check 0` `docker build 118MB` `README v0.0.6`

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.
