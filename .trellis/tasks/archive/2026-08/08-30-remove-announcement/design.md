# Design: Remove announcement feature

## Overview

公告为垂直功能：`announcement.Service` 拉远端 `JSON`，`api/announcement.go` 暴露 2 接口，`web` 的 `AdminAnnouncementModal` 弹公告。删除需自底向上，保持 `settings` 的 `KeyAnnouncementReadVersion` 仅公告用则一并删。

## Boundaries

| 层 | 删除 | 保留 |
|---|---|---|
| **服务** | `internal/announcement/*` 2 文件 | `internal/cache` 核心 |
| **API** | `internal/api/announcement.go` | `internal/api/health.go` 等 |
| **App** | `wire_http.go` 的 `announcement.New` | `backuprestore` 等 |
| **Router** | `router.go` 的 `Deps/Handler/announcement` 与 `2` 路由 | `tools/local-upload` 等 |
| **前端** | `AdminAnnouncementModal.vue` 等 | `AdminView` 除公告外 |

## Data Flow Removal

```
Before: web Mounted → useAnnouncement.Fetch() → GET /api/admin/announcement → announcement.Service.Fetch(https://www.litepan.top/announcement.json) → Warn on fail
After: 404 (前端不请求，后端无路由)
```

## Compatibility

- **DB**：`settings` 的 `KeyAnnouncementReadVersion` 若仅公告用则删，无专有表
- **Config**：无

## File Map

1. `internal/announcement`、`internal/api/announcement.go`
2. `internal/app/wire_http.go`、`internal/api/router.go`
3. `web` 的 `AdminAnnouncementModal`
4. `grep` sweep + `go vet` + `web build` + `docker build`
