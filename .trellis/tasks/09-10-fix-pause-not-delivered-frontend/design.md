# design.md — 前端暂停交付链路修复（0.0.33）

## 1. 改动边界（最小行为差）

| # | 现状 | 目标 |
|---|---|---|
| A | 远程任务若 id 在 `pendingRemoteResumeTaskIds`，`pauseUploadTask()` 只改本地、不发 HTTP | 远程任务**一律**发 `pauseTask`；本地（浏览器内）任务维持本地暂停语义 |
| B | 批量暂停用 `task.status ∈ {pending,running}` 过滤（本地乐观态） | 用「非终态 + 远程」过滤（服务端终态才是唯一可靠排除依据），收集后清空待恢复集合 |
| C | 待恢复集合会把 `paused` 掩码成 `pending`，即便最后消息是冷却 | 冷却语义存在时，展示状态/阶段标签以服务端状态为准 |

**行为真正所在**：暂停是否送达服务端发生在 `useUploadBatchActions.pauseUploadTask`（A）与 `handleToggleUploadTasks` 的收集循环（B）；展示掩码发生在 `uploadTaskFormatters.getUploadTaskDisplayStatus` / `getUploadTaskPhaseLabel`（C）。不把修复放到调用方（TaskPanel）去「打补丁」。

## 2. 契约（前端内部）

新增无依赖纯函数模块 `web/src/composables/upload/uploadPausePlan.ts`（不 import 任何运行时代码，便于断言脚本转译）：

```ts
export const TERMINAL_UPLOAD_STATUSES = ["success", "skipped"] as const;
export function isTerminalUploadStatus(status: unknown): boolean;
export function isCooldownMessage(message: unknown): boolean;            // /冷却/
export function collectRemotePauseIds<T extends { task_id?: unknown; status?: unknown }>(
  tasks: readonly T[], isLocalTask: (t: T) => boolean): string[];
export function resolveUploadDisplayStatus(
  status: string, opts: { waitingResume: boolean; cooling: boolean }): string;
```

语义：

1. `collectRemotePauseIds`：跳过本地任务、跳空 id、跳过终态、按 id 去重；**不参考** `pending`/`paused`/`running` 的本地判断（陈旧 `paused` 也必须入选——服务端对已暂停任务是 no-op）。
2. `resolveUploadDisplayStatus`：`cooling === true` 时直接返回服务端 `status`（不做待恢复掩码）；否则维持既有规则（`waitingResume && status ∈ {paused,failed,canceled}` → `"pending"`）。
3. `isCooldownMessage` 与服务端文案耦合（`账号网络冷却中，N 秒后自动重试` / `上传暂缓：账号网络冷却…`），在 spec 中登记为契约。

## 3. 数据流（暂停方向）

```
用户点暂停（行内主按钮 / 选择批量）
  → useUploadBatchActions.pauseUploadTask(task)            ← A：统一入口
        ├─ 本地任务：pausedLocalUploadTaskIds + abort（不变）
        └─ 远程任务：delete(pendingRemoteResumeTaskIds)
                     patchRemoteUploadTask(status=paused)      ← 乐观
                     await uploadApi.pauseTask(id)             ← 必发
                     await stream.fetchUploadTasks()           ← 以服务端为准回填
  → 服务端 Manager.pause(): pending/running → paused，并 cancel() 中断冷却等待
  → SSE/轮询回填 → 行状态=已暂停（不再回弹）
```

批量路径（`handleToggleUploadTasks` 非 resume 分支）：

```
remoteIds = collectRemotePauseIds(unique, isLocalUploadTask)   ← B
for id of remoteIds: pendingRemoteResumeTaskIds.delete(id)     ← 停止客户端待恢复重试
await uploadApi.batchPause(remoteIds) → stream.fetchUploadTasks()
（本地任务仍逐个 pauseUploadTask(task, true)）
```

## 4. 取舍与兼容

- **为何保留乐观 patch**：网络往返期间 UI 立即反馈；失败时 `fetchUploadTasks()` 回填纠正，不引入「幽灵暂停」。
- **为何不引入「服务端状态查询」再决定**：批量场景会放大 HTTP 次数（历史教训：1620 次逐任务请求卡死页面，0.0.26）；用「非终态一律发」把判断交回服务端（幂等）。
- **`resumeMode` 判定不改**：混合选择的语义保持既有实现；陈旧状态导致的误判定调用 `resumeTask`，服务端对非 paused 任务为 no-op，无副作用。
- **回滚形态**：单文件级回退（`uploadPausePlan.ts` 新文件 + 三处调用点），无数据迁移、无配置开关。
- **兼容性**：不改后端字段/端点；0.0.30/0.0.31/0.0.32 服务端语义均兼容（暂停请求本就是老接口）。

## 5. 风险

| 风险 | 处置 |
|---|---|
| 批量暂停范围变大（原先被乐观过滤掉的任务也会发暂停） | 服务端 `pause()` 对非 pending/running 是 no-op，返回 `updated=[]`，安全 |
| 冷却消息掩码取消后，刚点「继续」的行短暂显示旧状态 | 服务端 ~100ms 内回填 pending；换取「不掩盖真相」，可接受 |
| 待恢复集合清空后用户以为「点了继续却没继续」 | 清空仅发生在**暂停**动作内（用户意图明确是暂停） |

## 6. 测试策略

- 纯函数断言（`npm run check:memo` 新增用例）：A/B/C 三类规则各覆盖正反例。
- 类型与构建门（`type-check`/`build`）保证改动跨文件一致。
- 端到端（本地实例，人工演练）：让任务处于冷却等待（或直接构造 pending + 冷却消息）→ 点暂停 → 断言服务端 `pauseTask` 被调用、行状态落为「已暂停」、30 秒内无新的「上传暂缓/已恢复」日志对。
