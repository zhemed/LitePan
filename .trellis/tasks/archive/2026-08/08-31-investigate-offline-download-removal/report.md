# 报告：离线下载是否可彻底移除

**任务**：`08-31-investigate-offline-download-removal`  
**时间**：2026-08-31  
**基线**：`main 7439a18 v0.0.5`  
**结论**：**可彻底移除**（`local_upload` 不依赖），**106 处后端 + 约 40 处前端** 需清理，若确认不再使用离线下载则安全

---

## 一、清单（全量 `grep offline_download|OfflineDownload`）

| 层 | 文件数 | 行数 | 文件 |
|---|---|---|---|
| **domain** | 2 | 10 | `internal/domain/offline_download.go` `internal/domain/automation.go:20 offline_download 触发器` |
| **store** | 3 | 15 | `internal/store/offline_download_repo.go` `store.go:14 OfflineDownloads` `backup.go:94 DELETE offline_download_tasks` |
| **driver** | 2+8 | 20 | `internal/driver/offline_download.go`（`Capabilities/Provider` 接口） + `drivers/115_Open/offline_download.go` 等 8 驱动的 `OfflineDownload*` 实现（115/123/139/Guangya/Baidu 等） |
| **service** | 2 | 30 | `internal/offlinedownload/service.go`（`Service` 650 行，`eventbus OfflineDownloadCompleted` 650-658） `service_test.go` |
| **app** | 3 | 8 | `internal/app/wire_services.go:48 Repo OfflineDownloads` `wire_http.go:60 OfflineDownloads` `internal/api/router.go:56 118 279-282` |
| **api** | 1 | 4 | `internal/api/offline_download.go` `list/refresh/delete/batchDelete` 4 handler |
| **eventbus** | 1 | 2 | `internal/eventbus/events.go:33 OfflineDownloadCompleted` |
| **automation** | 3 | 10 | `service_validate.go:75,98 offline_download触发器` `service_offline_download.go:16 Subscribe onOfflineDownloadCompleted` `service_offline_download_test.go` |
| **前端 API** | 2 | 8 | `web/src/api/offlineDownload.ts` `web/src/types/offline-download.ts` |
| **前端 逻辑** | 3 | 20 | `web/src/composables/useOfflineDownloads.ts` `useUploadPanelActions.ts:offline` `useUploadTaskStore.ts: offline` |
| **前端 视图** | 1 | 15 | `web/src/components/file/FileBrowser.vue:18 153 179 193 501 779 908` 离线面板/能力/任务展示 |

**总数**：`106 go` + `~40 ts/vue`，`grep --exclude-dir=_extracted/.trellis/node_modules` 后仍 `~100`。

---

## 二、依赖图

```
触发器 automation.TriggerOfflineDownload
  → service_offline_download.go Subscribe(eventbus.OfflineDownloadCompleted)
    → eventbus 来自 offloadeddownload/service.go:658 Publish
      → offloadeddownload 依赖 driver.OfflineDownloadProvider
        → drivers/115_Open/offline_download.go 等 8 驱动
  → automation → 仅触发 local_upload 等动作（可配），与 offline_download 逻辑解耦

API /files/offline-download/* 
  → offloadeddownload.Service → 驱动能力探测 → 前端 useOfflineDownloads → FileBrowser 离线面板

Store offline_download_tasks
  → 仅被 offloadeddownload 读写，无外键
```

**与 `local_upload` 关系**：**零耦合**。`local_upload` 走 `upload.Manager + driver.Upload`，不走 `OfflineDownloadProvider`；自动化 `local_upload` 可配 `daily/interval/webhook/offline_download` 触发器，但 `offline_download` 仅是触发源之一，删后 `local_upload` 仍可 `daily 02:00` 正常。

---

## 三、能否彻底移除

| 判断 | 结论 | 依据 |
|---|---|---|
| **local_upload 是否依赖** | **否** | `service_run.go runLocalUpload` 仅 `files/settings/uploads`，无 `offlinedownload` |
| **DB 外键** | **无** | `offline_download_tasks` 独立表，仅 `backup.go DELETE` 清理 |
| **驱动** | **可删或保留** | 若删则 115 等 `offline_download.go` 能力探测返回 `unsupported`，前端不再显示离线入口；若保留驱动层仅删触发器亦可，但用户问“所有内容”则含驱动 |
| **前端** | **可删** | `FileBrowser` 离线 `badge/panel` 仅 UI，删后不影响 `Browser/Upload` |
| **自动化** | **可删** | `TriggerOfflineDownload` 为 4 触发器之一，删后仅留 `daily/interval/webhook`，`validate` 白名单需同步 |

**安全条件**：确认 **不再使用** `离线下载`（URL/磁力/种子 → 云盘）功能。若仍需 `115 离线` 则应保留驱动层。

---

## 四、彻底移除清单（若执行）

**后端 14 文件/段**：
- `internal/domain/offline_download.go` 整文件
- `internal/domain/automation.go:20` `AutomationTriggerOfflineDownload`
- `internal/store/offline_download_repo.go` + `store.go:14,31` `OfflineDownloads` + `backup.go:94`
- `internal/driver/offline_download.go` 整文件
- `drivers/115_Open/offline_download.go` 等 8 驱动的 `offline_download.go`/`builtin_magnet.go` 中 `OfflineDownload` 相关（可保留驱动其余）
- `internal/offlinedownload/` 整目录 `service.go/service_test.go/types.go/...`
- `internal/app/wire_services.go:48` `wire_http.go:60` `router.go:56,118,279-282`
- `internal/api/offline_download.go` 整文件
- `internal/eventbus/events.go:33` `OfflineDownloadCompleted`
- `internal/automation/service_offline_download.go` + `service_offline_download_test.go` + `service_validate.go:75,98` `offline_download` 分支

**前端 7 文件/段**：
- `web/src/api/offlineDownload.ts` + `web/src/types/offline-download.ts`
- `web/src/composables/useOfflineDownloads.ts` + `upload/useUploadPanelActions.ts: offline` 分支 + `useUploadTaskStore.ts: offline`
- `web/src/components/file/FileBrowser.vue` 离线相关 `import/useOfflineDownloads` `18,153,179,193,201,207,210,216,220,502,511,519,779,824,908`

**DB**：`offline_download_tasks` 表（`store` 已删则自动不再读写，可 `DROP TABLE` 或保留空）

---

## 五、建议

| 方案 | 操作 | 适用 |
|---|---|---|
| **A 彻底移除** | 按 §四 14+7 清单 `rm -rf`，`go vet/type-check` 后 `0.0.6` | **推荐**：若离线下载确不再用，`local_upload` 不受影响，`118MB` 进一步缩 |
| **B 仅移除触发器** | 仅删 `domain TriggerOfflineDownload` + `automation offline_download.go`，保留驱动/API 供手动离线 | 保留 `115 磁力` 手动入口 |
| **C 保留** | 不删 | 仍需离线 |

**风险**：A 方案下历史 `automation_rules trigger_type=offline_download` 需 `UPDATE status=paused` 或 `validate` 报错提示重配；`go vet` 已 `0`，`type-check` 需清前端 `offline` 导入。

---

## 附：取证

```bash
grep -R -n "OfflineDownload" --include="*.go" --exclude-dir=_extracted | wc -l
ls internal/offlinedownload/ drivers/*/offline_download.go
grep -R "offline_download" web --include="*.ts" --include="*.vue" | wc -l
```
