# research.md — 10.0.0.11「冷却等待中的任务无法暂停」取证记录

> 调查时间：2026-09-10 23:05–23:20 CST（用户当轮显式要求调查 11 生产机最新异常）
> 目标机：`10.0.0.11`（fnOS），LitePan `:5211`，**全程只读**（`mode=ro` + 只读 docker 命令）

---

## 1. 版本与场景（R3）

| 项目 | 值 |
|---|---|
| 镜像 | `ghcr.io/zhemed/litepan:v0.0.32`，ImageID `sha256:d6a7451d3333…`（与本地 `0.0.32` 构建产物一致） |
| 容器 | `StartedAt=2026-09-10T14:57:17Z`（22:57:17 CST），`RestartCount=0` |
| 0.0.32 新日志是否生效 | **是**：窗口内仅 1 条「上传暂缓」，随后 1 条「账号网络冷却已恢复」 |
| 账号 | `account_id=2` = 115网盘 |
| 批次 | 822 条 `server_local` 任务（自动化「定时全局备份」），当前 `paused 783 / success 39 / failed 0 / pending 0` |

**0.0.32 抑制效果实证**（对比 0.0.31 的 183 条刷屏）：

```
22:59:06.485 INFO 上传暂缓：账号网络冷却，稍后自动重试 account_id=2 name=Wub.ini retry_after_seconds=30
22:59:37.512 INFO 账号网络冷却已恢复 account_id=2 deferred_tasks=1 window_seconds=31
```

## 2. 时间线（R4，证据：容器日志 + DB 行）

| 时刻 | 事件 | 证据 |
|---|---|---|
| 22:57:17 | 生产机升级到 0.0.32 后启动 | `docker inspect` `StartedAt` |
| 22:57:55–22:58:26 | 恢复上传后连续 **19 个任务成功**（多为秒传） | DB `success.updated_at` 直方图 |
| **22:59:06.484** | 用户点「暂停」：**783 个任务**被批量置为 `paused` + `上传已暂停` | DB `updated_at` 同毫秒大量行 |
| **22:59:06.485** | 同一毫秒：`Wub.ini` 命中账号冷却，记录窗口首条「上传暂缓」，进入 **30 秒等待** | 容器日志 |
| 22:59:06.485→22:59:37.51 | `Wub.ini` **未被暂停**，等待持续了整整 31 秒 | 见 §3 |
| **22:59:37.511** | `Wub.ini` **上传完成**：`文件 'Wub.ini' 秒传成功`（102172 B，progress=100） | DB 行 `023ef0726a1a` |
| 22:59:37.512 | 冷却窗口结束汇总：`deferred_tasks=1 window_seconds=31` | 容器日志 |
| 之后 | 全队列回到「已暂停」（783 paused），无 pending 残留 | DB 状态计数 |

> 用户描述「有一个文件停留在 30 秒等待无法暂停，30 秒后冷却恢复才能完全暂停」= 上表第 4~7 行。

## 3. 服务端行为证明：**暂停根本没有到达这台任务的执行器**（R5 关键证据）

后端对「冷却等待中的暂停」是**立即响应**的，逐行可证：

1. `internal/upload/worker.go:107-127`：冷却分支在等待前 `canCooldownWait` 放行，`m.patch` 置 `pending`，随后
   ```go
   select {
   case <-ctx.Done():   return false          // ← 暂停/取消：立刻结束等待
   case <-time.After(seconds * time.Second):
   }
   return m.canCooldownWait(taskID)
   ```
2. `internal/upload/lifecycle.go:58-70`：`pause()` 置 `paused` 后 **`cancel()`** 执行中任务的 ctx。
3. `internal/upload/queue.go:118-127`：`acquireRunSlot()` 在任务获得执行权时 `st.cancel = cancel` —— 冷却等待期间该 `cancel` 有效。

**推论**：若暂停在 22:59:06.5–22:59:36 之间任一时刻到达 `Wub.ini`，`<-ctx.Done()` 会立刻触发 → 等待中断 → `executeUpload` 返回 false → 任务保持 `paused`，**不会**在 22:59:37 上传成功。

**但实测它确实在 22:59:37.511 上传成功**（等待完整走完 30 秒）⇒ **该任务的暂停请求从未生效在服务端**。

补充排除项（同样与观测不符，故排除）：
- 「暂停先到、被冷却 patch 覆盖、任务遗留 pending 无 runner」→ 该任务会永久卡在冷却提示且**不可能**在 22:59:37 成功；且 `deferred_tasks=1` 表明窗口内只有 1 次冷却命中（若被客户端恢复触发过第二次命中，这里会是 2）。
- 「暂停后服务端拒绝」：`pause()` 对 `pending/running` 一律生效，冷却等待期间任务正是 `pending`。

**结论：服务端没有可导致「暂停被延迟 30 秒」的路径；问题在客户端没有把暂停送达。**

## 4. 前端缺陷定位（R5）

`web/src/composables/upload/useUploadBatchActions.ts` 的 `pauseUploadTask()` 存在**只改本地状态、不发 HTTP** 的分支：

```ts
if (isQueuedRemoteResumeTask(task)) {          // task_id ∈ store.pendingRemoteResumeTaskIds
  store.pendingRemoteResumeTaskIds.delete(String(task.task_id));
  store.patchRemoteUploadTask(task.task_id, { status: "paused", message: getPausedMessage(task), error: "" });
  return;                                      // ← 没有 uploadApi.pauseTask()
}
```
（同文件 108-113 行；`isQueuedRemoteResumeTask` 定义在 30 行，集合定义在 `useUploadTaskStore.ts:62`）

且批量暂停的收集循环**按 UI 本地（乐观）状态过滤**：

```ts
for (const task of unique) {
  if (!["pending", "running"].includes(String(task.status))) continue;   // ← 本地状态过期即被跳过
  if (isLocalUploadTask(task) || isQueuedRemoteResumeTask(task)) { await pauseUploadTask(task, true); continue; }
  ...
}
await uploadApi.batchPause(remoteIds);
```
（同文件 180-196 行；任务面板行内按钮最终也走 `handleToggleUploadTasks` —— `web/src/components/upload/TaskPanel.vue:864`）

**可见症状链条**（与用户描述完全吻合）：
1. `Wub.ini` 处于服务端 `pending` + 「账号网络冷却中，30 秒后自动重试」；
2. 客户端该行要么仍是**乐观态 `paused`**（此前点过暂停/来自本地 patch），要么其 id 仍在 `pendingRemoteResumeTaskIds`（此前点过「继续」后被调度器反复重试）；
3. 展示层 `uploadTaskFormatters.ts:32-34` 会把「在待恢复集合中且状态为 paused/failed/canceled」的行显示成 `pending`（「等待继续」），消息沿用服务端的冷却文案 → **用户看到的就是「停在 30 秒等待」**；
4. 用户点暂停时：命中上述两个分支之一 ⇒ **不发送 `pauseTask`**（且 `silent=true`，连失败提示都没有）→ 服务端继续跑冷却等待；
5. 下一次 `stream.fetchUploadTasks()` 用服务端状态回填（`useUploadTaskStore.ts:301-309` 合并规则），本地「已暂停」被覆盖 → 行又跳回「冷却中」⇒ **表现为「无法暂停」**；
6. 30 秒窗口结束 → `Wub.ini` 秒传成功（任务终态，不再有待恢复集合的干扰）→ 界面才安静下来 ⇒ **表现为「30 秒后才能完全暂停」**。

**证据强度**：服务端事实链（第 3 节）为**强证据**；客户端分支定位为**强（代码可读 + 症状唯一吻合）**，但**具体是「乐观态过滤」还是「待恢复集合」触发，需浏览器端复现确认**（两者都在同一函数内，修复面相同）。

## 5. 影响面（R4 补充）

| 维度 | 评估 |
|---|---|
| 本次规模 | **仅 1 个任务**（`deferred_tasks=1`），其余 783 个立即暂停成功 |
| 数据风险 | **无**：该文件为秒传命中，最终 `success`；无重复上传、无残留 pending |
| 行为风险 | **有**：暂停后该任务仍继续上传最长 30 秒（若为大文件则可能持续更久）——用户以为已停，实际仍在写云端 |
| 复现条件 | 目标任务必须**正处在账号冷却等待**（或本地状态与服务器不一致），此时点暂停即可复现 |
| 是否普遍 | 只在「点暂停的瞬间有任务处于冷却等待/本地状态过期」时出现；批量暂停其余任务不受影响 |

## 6. 修复建议（R6，本任务不实施）

| # | 缺陷 | 位置 | 建议 |
|---|---|---|---|
| A | 暂停只改本地、不发 HTTP（假定「待恢复」任务尚未在服务端运行，实际可能正在冷却等待/运行） | `web/src/composables/upload/useUploadBatchActions.ts:108-113` | 删除该捷径或改为「先查服务端状态，非 paused 就调用 `pauseTask`」；`silent` 不应吞掉暂停失败 |
| B | 批量暂停按**本地乐观状态**过滤，状态过期即跳过 | 同上 180-196 行 | 按服务端快照（或任务真实 id 列表）收集，不依赖乐观态；对 `success/terminal` 之外的远程任务一律发暂停请求 |
| C | 展示层把「待恢复集合 + 已暂停」渲染成 `pending`，掩盖了与服务端的差异 | `uploadTaskFormatters.ts:32-34` | 待恢复态与「冷却等待」应显式区分（冷却来自服务端 message，不应被本地集合改写状态语义） |
| D | 服务端可选加固（防御性） | `internal/upload/lifecycle.go`/`worker.go` | 冷却等待期间若被暂停，落库终态校验已有（0.0.30）；可增加「暂停后 1 秒内断言状态为 paused」的自检日志，便于定位同类客户端问题 |

**最小复现**：客户端保持一个任务的 id 在 `pendingRemoteResumeTaskIds`（点一次「继续」），同时让该任务在服务端处于冷却等待（账号退避中），然后点「暂停」→ 观察该任务无 `pauseTask` 请求、行状态在刷新后回弹，服务端等待走完后仍会上传。

**验收点（修复任务）**：
1. 冷却等待中点暂停：**立即**发出暂停请求，任务落为 `paused`，且不再有后续上传尝试（可在等待中点暂停验证 30 秒内不再产生「上传暂缓/已恢复」日志对）；
2. 批量暂停不再依赖本地乐观状态：一个处于冷却等待的任务也必须被暂停；
3. 前端不再出现「本地显示已暂停、服务端仍在跑」的分歧（刷新后状态一致）；
4. 保留 0.0.32 冷却日志抑制语义不变。

## 7. 未决与合规（R7）

- **未决**：触发路径（乐观态 vs 待恢复集合）需浏览器复现确认；本次无法从服务端侧完全区分（服务端不记录暂停请求明细）。
- **只读合规**：仅 `docker ps/inspect/logs/stats`、`ls/df/cat`、SQLite `mode=ro` 的 `SELECT`；未重启容器（`RestartCount=0`、`StartedAt` 未变）、未改配置/数据、未部署；口令仅内存传参未落盘。
- **副作用**：无（本轮未登录应用，未产生任何写入）。
