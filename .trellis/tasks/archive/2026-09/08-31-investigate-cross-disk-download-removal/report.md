# 报告：跨盘下载是否可彻底移除

**任务**：`08-31-investigate-cross-disk-download-removal`  
**时间**：2026-08-31  
**基线**：`main 7439a18→07c2d51 v0.0.7`（`crosstransfer` 跨盘秒传已在 `08-31-remove-crosstransfer` 移除）  
**结论**：**可彻底移除**（与 `local_upload` 零耦合），**81 处 go + 8 处 ts/vue** 需清理，属 `upload` 核心分支

---

## 一、清单（ `grep 跨盘下载|cross_transfer|crosstransfer` ）

| 层 | 命中 | 文件 |
|---|---|---|
| **domain/upload** | 1 | `internal/upload/types.go:61 SourceTypeCrossTransfer="cross_transfer"` |
| **upload 核心** | 60+ | `internal/upload/manager.go:301,312,319,328,523` `worker.go:22 executeCrossTransferDownload 258行` `lifecycle.go:32,120` `queue.go:36,83,88,206` `persist.go:103,221,226` |
| **tests** | 20+ | `internal/upload/manager_test.go:56 blockingCrossTransferDriver, 474 TestCrossTransferDownloadReleasesSlot, 975,1065,1147` `lifecycle_test.go:90` `progress_test.go:240` |
| **settings** | 1 | `internal/settings/registry.go:97` `任务并发数` 描述含 `跨盘下载` 三队列之一 |
| **前端** | 8 | `web/src/composables/upload/useUploadBatchActions.ts:22` `useUploadTaskStore.ts:88,105` `web/src/components/upload/TaskPanel.vue:309,382,462,620,756,768`（`跨盘下载` 分类、标签、进度） |

**区分**：
- `crosstransfer` 跨盘**秒传**（`internal/crosstransfer` 服务）已在 `0.0.6` 前彻底移除，本次为 `cross_transfer` 跨盘**下载**（`upload` 源类型），两者不同但均属“跨盘”。
- `grep crosstransfer` 在 `backend` 现 `0`（秒传已清），`cross_transfer` 仍 `81`。

---

## 二、依赖图

```
FileBrowser 跨盘下载 UI
  → TaskPanel 跨盘下载分类 / 进度“跨盘下载中”
    → useUploadTaskStore / useUploadBatchActions 过滤 cross_transfer
      → upload.Manager SourceTypeCrossTransfer
        → worker.go executeCrossTransferDownload (下载源盘 → 本地 temp → 上传目标盘)
          → queue.go/lifecycle.go 阶段 PhaseDownloading → PhaseUploading
            → driver.ResolveDownload + UploadLocalFile
              → 与 local_upload 的 UploadLocalFile 复用驱动层，但源不同

settings KeyUploadTaskConcurrency 描述“上传、跨盘下载和内置下载三个队列” → 仅文案
```

**与 `local_upload` 关系**：**零耦合**。`local_upload` 为 `SourceTypeServerLocal` 本地文件，`cross_transfer` 为 `SourceTypeCrossTransfer` 云到云经本地中转，两者在 `manager.go:319 case Manual, CrossTransfer` 仅共享上传分支的 `SourceType` 枚举，逻辑独立。

---

## 三、能否彻底移除

| 判断 | 结论 | 依据 |
|---|---|---|
| **local_upload 是否依赖** | **否** | `runLocalUpload` 不走 `SourceTypeCrossTransfer` 分支 |
| **是否还有秒传残留** | **无** | `crosstransfer` 已 `0`，本次仅 `cross_transfer` 下载 |
| **DB** | **可删** | `upload_tasks source_type=cross_transfer` 独立，无外键 |
| **前端** | **可删** | `TaskPanel` 跨盘下载分类/标签仅 UI，删后仅剩 `upload/relay` |
| **并发描述** | **需同步** | `registry.go:97` 三队列文案需改为 `上传队列` |

**安全条件**：确认 **不再使用跨盘下载**（`A 盘文件 → B 盘` 经 `data/cross_transfer` 中转）。若仍需 `115→天翼` 互传则保留。

---

## 四、彻底移除清单（若执行）

**后端 8 文件/段**：
- `internal/upload/types.go:61` `SourceTypeCrossTransfer`
- `internal/upload/worker.go:22` `executeCrossTransferDownload` 236 行 + `finish*` `resolveCrossTransferTarget` `openCrossTransferTempFile`
- `internal/upload/manager.go:301,312,328` `SourceTypeCrossTransfer` 3 分支 + `cross_transfer` temp 路径
- `internal/upload/lifecycle.go:32,120` `PhaseDownloading` 跨盘判断
- `internal/upload/queue.go:36,83,88,206` 跨盘队列
- `internal/upload/persist.go:103,226` 跨盘持久化
- `internal/upload/manager_test.go:56 block/priority driver + 474,975,1065,1147` 4 用例 + `app/lifecycle_test.go:90`
- `internal/settings/registry.go:97` 文案 `跨盘下载和` 删除

**前端 3 文件**：
- `web/src/composables/upload/useUploadBatchActions.ts:22,27` 跨盘文案
- `web/src/composables/upload/useUploadTaskStore.ts:88,105` 过滤
- `web/src/components/upload/TaskPanel.vue:309,382,462,620,756,768` `label 跨盘下载` `status 跨盘下载中` `filter` + `navCategories`

**DB**：`upload_tasks` 中 `cross_transfer` 行（可保留或 `DELETE WHERE source_type='cross_transfer'`）

---

## 五、建议

| 方案 | 操作 | 适用 |
|---|---|---|
| **A 彻底移除** | 按 §四 8+3 清单 `rm`，`go vet/type-check` 后 `0.0.8` | **推荐**：若仅 `local_upload`，`105MB→~104MB` |
| **B 保留** | 不删 | 仍需跨盘互传 |
| **C 仅 UI** | 仅删前端 `跨盘下载` 分类 | 保留后端能力但隐藏入口 |

**风险**：A 方案下 `worker.go` 删 236 行后 `manager.go` 仅 `Manual/ServerLocal`，需 `go test -run TestCrossTransfer` 同步删除；`TaskPanel` 删分类后 `upload/relay` 仍正常。

---

## 附：取证

```bash
grep -R -n "SourceTypeCrossTransfer" --include="*.go" | wc -l
ls internal/upload/worker.go | grep -n executeCrossTransferDownload
grep -R "跨盘下载" web --include="*.vue" | head
```
