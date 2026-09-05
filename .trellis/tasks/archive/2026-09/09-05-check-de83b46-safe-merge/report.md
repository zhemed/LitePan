# de83b46 上传批次化安全合入评估报告

评估时间：2026-09-05（UTC+8）· 子任务2（父任务：调查剩余上游提交能否安全合并）
上游提交：`de83b46`（08-30《上传任务批次化》，源码约40文件，+1924/−244）

## 结论

**不能直接合入（整提交 cherry-pick 必败）。建议拆 A/B 两路，A 路可独立合入，B 路成本高按需定夺。**

- 逐文件 `apply --check`：29 PASS / 5 FAIL（`api/router.go`、`upload/manager.go`、`upload/worker.go`、`TaskPanel.vue`、`useUploadTaskStore.ts`）+ 测试 `manager_test.go` FAIL。
- FAIL 根因是本地 `0.0.6/0.0.10` 两次精简删掉了同一批文件里的跨盘/离线分支，补丁上下文对不上——不是逻辑冲突，是基线分叉。
- 补丁新增行**零引用**已删概念（无 `CrossTransfer/offlineDownload/relay`），PASS 部分合入后不复活已删功能。

## FAIL 文件移植成本

| 文件 | 失败点 | 移植量 |
|---|---|---|
| `api/router.go` | 1行`batch-pause`路由，锚点下方的 offline 路由块本地已删 | 1行，手工加到`batch-delete`旁即可 |
| `upload/manager.go` | struct：`broadcastDirty`→`broadcastAllDirty+DirtyTaskIDs+DeletedTaskIDs`；另含`BatchPause`新方法 | 中：struct 手工改 + 方法 hunk 可拆出单合 |
| `upload/worker.go` | 两处：`retainBatchRootMetadata`回填 + `broadcast()`→`broadcast(taskID)`，上下文含已删的`finishCrossTransferDownloadError` | 小：两处手工改 |
| `TaskPanel.vue` | 274行任务树重写 vs 本地精简版（删过360行跨盘/离线） | 大：需以本地版为基重写 |
| `useUploadTaskStore.ts` | 69行hunk含batch字段；`addLocalUploadTasks`等为纯新增可拆 | 中：拆 hunk 移植 |
| `manager_test.go` | 209行新测试，本地删过425行跨盘测试 | 中：手工对齐 |

## 原子性约束（不可拆错）

- `sse.go`（PASS）新代码引用 manager 新字段 → **sse 与 manager struct 必须同批**；但 `broadcast(taskIDs...)` 为 variadic，旧无参调用方不破。
- `upload.go` 的 `batchPauseUploadTasks`（PASS）调用 `Manager.BatchPause`（FAIL文件内）→ **路由+handler+BatchPause 必须同批**，否则编译不过。
- 前端 `uploadTaskTree.ts`（新，PASS）、`folderPlanner`（PASS）等可独立于 TaskPanel 存在。

## A/B 两路建议

- **A路·文件夹上传链（风险低-中，建议做）**：`CloudLocalUploadPanel`拖拽文件夹部分（PASS）+ `useUploadFolderPlanner`/`useLocalUploadDispatcher`/`useUploadFileInput`/`confirmUpload`/`FileBrowser`（PASS）+ `types`（PASS）+ `local_upload.go` API（PASS）+ store 纯新增 hunk（手工拆）+ migration/repo（见下）。补的是本地缺失的"文件夹拖拽上传"能力。
- **B路·任务树折叠+增量SSE（风险中-高，按需）**：上面 FAIL 表 + `sse/manager`原子组。解决的是大批量任务渲染性能，用户任务量不大可暂缓。
- **无争议可直接合**：`migration 0022`（本地停在0021，新文件PASS）、`upload_task_repo`、`domain/upload_task`、`delete/lifecycle/persist/progress/queue/state/types`（PASS）、`api/upload.ts`、`uploadTaskStream`等。

## 只读确认

全程 `git show/diff/apply --check/grep`，未改业务代码，未合入。工作区除任务目录外干净。
