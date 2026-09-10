# fix-cooldown-orphan-task

## Goal

修复用户实测发现的缺陷（0.0.24 引入）：任务显示「等待中 / 账号网络冷却中，30 秒后自动重试」后**永远不再重试**（孤儿任务）。

## 现象与根因

**现象**（用户 5000 文件批次实测，20:21:16）：`bulk5k_0850.bin` 卡在「pending + 账号网络冷却中，30 秒后自动重试」，35 秒后无任何变化；批次其余 4383 条已暂停、616 条成功、0 失败。

**根因**（`internal/upload/worker.go` 冷却分支）：
```go
if seconds, cooling := driverexec.IsCooldownError(err); cooling {
    m.patch(... st.Status = StatusPending ...)   // 置回 pending
    select { case <-ctx.Done(): case <-time.After(wait): }
    return                                        // ← 协程退出！
}
```
任务虽被置回 pending，但**执行它的 goroutine 已返回**；pending 任务的运行依赖各自在 `acquireRunSlot` 等待的协程，无人重新接管 → 任务永久滞留（提示语承诺的"自动重试"从未发生）。

**附带竞态**：冷却分支无条件把状态改成 pending，可能覆盖同一时刻用户发起的暂停（pause 先执行 → 被 patch 覆盖 → 任务既没在跑、也不是暂停态）。

## Requirements

1. `runTask` 支持**重入队列**：`executeUpload` 返回 `requeue`，为真时重新 `acquireRunSlot` 并再次执行（冷却等待结束后自动重试真正生效）
2. 冷却分支**竞态加固**：仅当任务仍处于可重试状态（非暂停/取消/停止）时才置 pending；等待期间被暂停/取消则保持其状态且不重入
3. 保持既有语义：冷却不判死、并发=1 时整队等待、暂停可中断等待
4. 测试覆盖：冷却 → 返回 requeue 且状态为 pending；暂停态 → 不 requeue；等待期间取消 → 不覆盖暂停

## Acceptance Criteria

- [x] 冷却后任务真正重入队列自动重试；暂停态不再被覆盖（canCooldownWait）
- [x] 单测 3 例全过（requeue=true/暂停不覆盖/取消停止拒绝）+ 既有套件全绿
- [x] 0.0.30 三 tag digest c8aae4f9 + release + 部署三连；实测孤儿任务部署后被接管并成功上传
- [x] 门禁 + 归档 + journal + push
