# 候选清单 P3 / P4 关闭记录（零代码改动）

## 结论

| 项 | 上游依据 | 处置 | 依据摘要 |
|---|---|---|---|
| **P3** | `99ea858` 认证收口 | **归档不实施**（用户 2026-09-12 指示） | 见 §1 |
| **P4** | `df86352` 账号级目录缓存失效 | **本方不适用**（查证后判定） | 见 §2 |

---

## 1. P3「认证收口」：按用户指示归档

- 上游 `99ea858` 内容：`completeRefresh` 统一收尾、`schedule_calc` 调度计算复用、`scheduler` 日志漂移抑制（6 文件、约 220 行差异），属**重构 + 日志降噪**，非缺陷修复。
- 本方现状：`internal/auth` 已在 `0.0.17` 从上游同源移植（`grep completeRefresh` → 0，即未含该批改动）；认证链路当前**无已知缺陷**（0.0.32 的日志放大问题发生在冷却日志侧，已单独修复）。
- 用户决定：**归档不实施**。保留此记录，以便将来出现认证类问题（如刷新风暴、调度抖动）时可据此重启该项。

## 2. P4「账号级目录缓存失效」：本方不适用（可达性 + 重复能力）

上游 `df86352` 的实质改动是**把自动化动作「刷新目录」从"逐目录 List 预热"改成"整账号清理目录缓存"**，为此新增：

```go
// 上游新增
func InvalidateAccountDirKeys(c *Service, accountID int64)          // internal/cache/keys.go
func (s *Service) InvalidateDirectoryCaches(accountID int64)        // internal/file/service.go
```

本方核查（只读）：

1. **被改的自动化动作已不存在**：本方 `internal/domain/automation.go` 只有 `AutomationActionLocalUpload` 一个动作（`grep AutomationAction` → 仅 local_upload），上游该动作（cache_clear/刷新目录）在 `1bcfac8` 精简时已随缓存任务一并删除 ⇒ **上游此次改动的目标载体在本方不存在**。
2. **等价/更精确的能力已具备**：
   - 账号级整体失效：`cache.Service.InvalidateAccount(accountID)`（`internal/cache/service.go:197`），并按缓存类型 `InvalidateAccountType`；
   - 写路径按目录精确失效：`internal/file/service.go:62/242/413` 调用 `cache.InvalidateDirKeys`；
   - 事件驱动：`internal/cache/cleaner.go`（`ApplyMutation` → 失效受影响的文件/目录键）；
   - 管理端「清空缓存」：`POST /api/admin/clear-cache` → `ClearAll()`（`internal/api/cache.go:55`）。
3. **若强行移植**：新增的 `InvalidateAccountDirKeys` / `InvalidateDirectoryCaches` 在本方**没有任何调用者**，只会制造新的死代码——与本次死代码排查（`09-12-investigate-dead-code-sweep`）的目标相反。

**结论**：P4 **不实施**；如将来恢复"刷新目录"类自动化动作，可连同该 helper 一起从上游取回（届时才有调用者）。

---

## 3. 本轮动作

- 零代码改动（`git status` 仅本任务目录）。
- 候选清单至此：P1 ✅（0.0.38/0.0.39）、P2 ✅（0.0.43）、P5 ✅（0.0.43）、P6 ❌作废、**P3 归档、P4 不适用**；剩余 **P7 / P8 / P9**。
