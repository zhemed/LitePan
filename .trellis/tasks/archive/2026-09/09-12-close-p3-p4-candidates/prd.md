# 09-12-close-p3-p4-candidates

## Goal

关闭上游候选清单中的 **P3**（认证收口 `99ea858`，用户指示归档不实施）与 **P4**（账号级目录缓存失效 `df86352`，查证后判定**本方不适用**）；给出证据并留档归档。本轮**零代码改动**。

## Background

- 候选来源：`.trellis/tasks/archive/2026-09/09-12-investigate-upstream-updates/research.md` §6。
- **P3**：上游 `99ea858`（`completeRefresh` 统一收尾 + `schedule_calc` 复用 + `scheduler` 日志漂移抑制，6 文件 ~220 行）属重构+日志降噪；本方 `internal/auth` 在 `0.0.17` 已从同源移植（不含该批），当前认证链路无已知缺陷。
- **P4**：上游 `df86352` 的实质是**把自动化动作「刷新目录」从逐目录预热改为整账号清理目录缓存**，为此新增 `cache.InvalidateAccountDirKeys` 与 `file.Service.InvalidateDirectoryCaches`。
- **查证要点（本任务新发现）**：
  1. 本方自动化**只有一个动作** `local_upload`（`internal/domain/automation.go:20`），上游所改的"刷新目录/清缓存"动作在 `1bcfac8` 精简时已随缓存任务删除 ⇒ **改动目标不存在**；
  2. 本方已具备等价且更精确的能力：`cache.Service.InvalidateAccount`（`internal/cache/service.go:197`）、写路径按目录失效（`internal/file/service.go:62/242/413`）、事件驱动失效（`internal/cache/cleaner.go`）、管理端清空缓存（`POST /api/admin/clear-cache` → `ClearAll`）；
  3. 强行移植只会得到**没有调用者的 helper＝新死代码**，与死代码排查目标相反。

## Requirements

- **R1 只读取证**：不改代码；证据＝`grep`/文件阅读/router 与动作常量核对。
- **R2 P3 处置**：按用户指示"归档不实施"，记录理由与"将来何时可重启"的条件。
- **R3 P4 处置**：给出"不适用"结论与逐条证据（被改动作不存在 + 已有等价能力 + 移植会产生死代码）。
- **R4 留档**：写入 `research.md` 并记录 journal；更新候选清单收尾状态（P1/P2/P5 已完成、P6 作废、P3 归档、P4 不适用、剩 P7/P8/P9）。

## Constraints

- 零代码改动；不触碰生产机；不改容器/DB。
- 若将来恢复"刷新目录"类自动化动作，P4 可随之重启（届时 helper 才有调用者）。

## Acceptance Criteria

- [x] P3 已按"归档不实施"处置并记录理由与重启条件
      → research.md §1：理由＝上游属重构+日志降噪非缺陷修复、本方 0.0.17 已同源移植（不含该批）、认证链路当前无已知缺陷；重启条件＝出现认证类问题（刷新风暴/调度抖动）时
- [x] P4 已给出"不适用"结论，且含逐条证据（动作已删 / 已有等价能力 / 移植即成死代码）
      → research.md §2：① 上游所改的自动化动作在本方不存在（`internal/domain/automation.go:20` 仅 `local_upload`，`grep AutomationAction` 佐证）；② 已有 `cache.InvalidateAccount`（`cache/service.go:197`）、写路径 `InvalidateDirKeys`（`file/service.go:62/242/413`）、事件驱动 `cache/cleaner.go`、管理端 `clear-cache → ClearAll`（`api/cache.go:55`）；③ 强移植将产生无调用者 helper
- [x] `research.md` 已产出并含可复现命令；本轮零代码改动（`git status` 仅任务目录）
      → research.md 已产出（含 grep/函数定位）；`git status --porcelain` 仅本任务目录
- [x] 候选清单收尾状态已更新并写入 journal
      → 状态：P1 ✅(0.0.38/0.0.39)、P2 ✅(0.0.43)、P5 ✅(0.0.43)、P6 ❌作废、P3 归档、P4 不适用、**剩 P7/P8/P9**；Session 132 记录

## Notes

- `scope=lightweight`，无代码产出。
- **本任务的方法论价值**：再次验证"候选移植项必须查证**目标载体是否存在**"——P4 的文件在树里，但它要修的功能早已随自动化动作删除（同 P6 的教训）。
- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
