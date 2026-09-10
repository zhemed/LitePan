# 设计

## runTask 重入循环（internal/upload/queue.go）

```go
slotKind, ok := m.acquireRunSlot(taskID, done, cancel)
if !ok { return }
for {
    requeue := m.executeUpload(runCtx, taskID)   // 冷却等待时为 true
    m.releaseSlot(slotKind)
    if !requeue { return }
    var acquired bool
    slotKind, acquired = m.acquireRunSlot(taskID, done, cancel)  // 重新排队等槽
    if !acquired { return }   // 已被暂停/取消/任务不存在 → 正常退出
}
```
`acquireRunSlot` 已具备"状态非 pending 即返回 false"的检查，因此重入对暂停/取消/删除天然安全。

## 冷却分支（internal/upload/worker.go）

```go
if seconds, cooling := driverexec.IsCooldownError(err); cooling {
    if !m.canCooldownWait(taskID) { return false }        // 暂停/取消/停止 → 不置 pending
    m.patch(taskID, func(st *taskState) {                  // 置 pending + 冷却提示 + 置顶
        st.Status = StatusPending
        st.Message = fmt.Sprintf("账号网络冷却中，%d 秒后自动重试", seconds)
        st.Error = ""; st.resumePriority = true; st.SpeedBytesPerSecond = 0
    })
    m.mu.Lock(); m.runCond.Broadcast(); m.mu.Unlock()
    select {
    case <-ctx.Done(): return false                       // 等待期间暂停/取消 → 保持其状态
    case <-time.After(time.Duration(seconds) * time.Second):
    }
    return m.canCooldownWait(taskID)                       // 仍是可重试状态 → 重入队列
}
```

`canCooldownWait`（新，纯判定）：
```go
m.mu.Lock(); defer m.mu.Unlock()
st, ok := m.tasks[taskID]
if !ok || m.stopping { return false }
if st.cancelMode == "pause" || st.Status == StatusPaused || st.Status == StatusCanceled { return false }
return st.Status == StatusRunning || st.Status == StatusPending
```

## 兼容/兜底

- 重启恢复路径（persist.go）对 status=pending 的任务会重新 `go m.runTask` → 现存孤儿任务在部署重启后自动恢复
- 用户侧也可对该任务"暂停→继续"手动恢复
