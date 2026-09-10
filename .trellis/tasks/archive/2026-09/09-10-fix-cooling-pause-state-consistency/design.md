# design.md — 批次 1：冷却/暂停状态一致性

## 1. 改动边界

| 现状 | 目标 |
|---|---|
| `worker.go` 先 `canCooldownWait()` 判定，再 `m.patch()` 无条件写 `pending` | 用 `beginCooldownWait()` **在锁内一次性**判定+写入；判定不通过则零改动返回 |
| `patch()` 解锁后用**活体指针** `snap := st` 持久化 | 用**值拷贝** `snap := *st` 持久化，落库内容锁定为该次迁移 |

**行为真正所在**：状态迁移的原子性在 `internal/upload/state.go`（新增守卫）+ `internal/upload/worker.go`（调用点）。不在调用方（service/handler）打补丁。

## 2. 契约

```go
// state.go
// beginCooldownWait 原子进入冷却等待：仅当任务仍可重试时写回 pending + 冷却文案。
// 返回 false 表示任务已暂停/取消/停止/不存在——调用方必须保持其现有状态（暂停优先）。
func (m *Manager) beginCooldownWait(taskID string, seconds int) bool

// canCooldownWait 保持原语义，用于等待结束后的「是否重入队」判定。
func (m *Manager) canCooldownWait(taskID string) bool
```

不变式（Invariants）：
- I1 `pause()/cancel()` 对同一任务的任何后续冷却迁移**始终优先**（不会被改写回 pending）。
- I2 任意时刻「内存状态」与「最后一次持久化行」在同一状态迁移之后一致；不存在 DB=pending 而内存=paused 的分叉。
- I3 启动恢复对 `pending` 行续传的语义不变（仍是「未终态即可续传」），但由于 I1/I2，用户暂停的任务不会被写回 pending。

## 3. 数据流

```
worker 收到 CooldownError
  → beginCooldownWait(taskID, seconds)         [m.mu 内：判定 + 迁移 + 值快照]
        ├─ false → return false（保持 paused/取消态；不等待、不重入队）
        └─ true  → broadcast → select{ ctx.Done()（暂停/取消） | 30s }
                     └─ 结束 → canCooldownWait(taskID) → true 重入队 / false 退出（保持暂停）
```

## 4. 取舍

- **为何新增函数而非给 `patch` 加参数**：`patch` 被 10 处复用，语义是「无条件迁移」；冷却重入队需要「条件迁移 + 持久化」，单独函数更清晰，也避免把守卫逻辑散进通用工具。
- **为何持久化仍在锁外**：SQLite 写在锁内会拉长持锁时间（进度类 patch 频率高）。用值快照即可保证「落库内容 == 该次迁移」，写入顺序由单写连接串行化；不一致的根因是活体读取，不是顺序。
- **不改 `persist.go` 恢复逻辑**：恢复语义本身正确（pending 可续传）；修好写入侧即消除分叉。改动恢复逻辑会引入「按消息文本判断是否暂停」的脆弱耦合。
- **不加配置开关**：无第二形态需求。

## 5. 风险与回滚

| 风险 | 处置 |
|---|---|
| 守卫条件写错导致冷却后不再重试（任务滞留 pending 无 runner） | 守卫仅在「已暂停/取消/停止/终态」时拒绝；正常 pending/running 一律放行；测试覆盖守卫放行路径 |
| 值快照让 `Result`/`resumeData` 仍共享 map | 仅读不写，风险与改动前一致；本次不扩大改动面 |
| 回滚 | 单文件级回退（`state.go` 新增函数 + `worker.go` 调用点），无数据迁移 |

## 6. 测试策略

- 守卫拒绝矩阵：paused / canceled / `cancelMode="pause"` / 不存在 / `m.stopping` ⇒ false 且字段零变更。
- 守卫放行：pending/running ⇒ true 且 `status=pending`、`message` 含冷却文案、`resumePriority=true`。
- 竞态一致性：先 `beginCooldownWait` 再 `pause()`，随后断言内存 `paused` 且 fake repo 最后一行 `paused`（非 pending）；反向顺序（pause → beginCooldownWait）断言守卫拒绝、行状态仍为 paused。
- 并发烟雾：`go test -race ./internal/upload/`。
