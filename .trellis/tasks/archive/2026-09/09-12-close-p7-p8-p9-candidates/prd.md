# 09-12-close-p7-p8-p9-candidates

## Goal

按用户 2026-09-12 指示，**归档上游候选清单 P7 / P8 / P9（不实施）**，并逐项给出"当前状态核实 + 为何可先不做 + 何时值得重启"的记录，作为候选清单的最终收尾。本轮**零代码改动**。

## Background

- 候选来源：`.trellis/tasks/archive/2026-09/09-12-investigate-upstream-updates/research.md` §6。
- **P7**（`ae75882` 子集）：上传取消语义（`isClientGone`/`isCancelError`）、`local_upload` 批次名收敛、`upload/resume.go` 定时器简化。
- **P8**（`adca0ee`）：铃铛通知改服务端 SSE 实时推送（替代轮询），涉及新端点 + 前端订阅。
- **P9**（`46a0a89`）：播放诊断日志 + 直读文案。
- 本次核实要点（只读）：
  1. **P7 的取消语义本方已内联等价**：`internal/api/upload.go:34/64/102` 三处已是 `errors.Is(r.Context().Err(), context.Canceled) || strings.Contains(strings.ToLower(err.Error()), "context canceled")`，上游改动只是把它抽成两个 helper ⇒ 零行为差异。
  2. **P7 的批次名部分属本方可有意行为**：`internal/api/local_upload.go:288-290` 仍派生 `batchName`（文件夹上传显示为单条批次需要它）；上游删除该逻辑，照搬会改变本方面板显示（且与本方 `0.0.36` 决策不冲突）。
  3. **P8 是功能增强非修复**：通知功能在线且用轮询，无缺陷/性能问题报告。
  4. **P9 是排障能力**：`internal/playback/service.go` 现无诊断日志（`grep 诊断` → 0），当前无播放故障驱动。

## Requirements

- **R1 只读取证**：不改代码；证据＝文件阅读 + `grep` 行号定位。
- **R2 逐项记录**：P7/P8/P9 各给出"属性（修复/增强/工具）+ 当前状态 + 不实施理由 + 重启条件"。
- **R3 清单收尾**：给出候选清单（P1~P9）的最终状态表。
- **R4 留档**：写入 `research.md` 并记录 journal。

## Constraints

- 零代码改动；不触碰生产机；不改容器/DB。
- 不实施任何 P7/P8/P9 的改动（如将来重启，另建任务）。

## Acceptance Criteria

- [x] P7 已核实并记录：取消语义已内联等价（附行号）、批次名属有意行为、定时器属代码卫生 ⇒ 不实施
      → research.md §1：`internal/api/upload.go:34/64/102` 已是等价内联判定（上游仅抽 helper）；`internal/api/local_upload.go:288-290` 的 `batchName` 是本方面板需要用到的有意行为（照搬上游删除会造成显示回归）；`resume.go` 定时器简化无行为差异
- [x] P8 已记录属性（功能增强）与不实施理由、重启条件
      → research.md §2：属性＝实时推送增强（新端点 + 前端订阅），本方轮询无缺陷/性能问题；重启条件＝需要秒级通知或减少轮询
- [x] P9 已记录属性（排障能力）与不实施理由、重启条件
      → research.md §3：属性＝诊断日志；本方 `internal/playback/service.go` 无诊断（grep=0）且无播放故障驱动；重启条件＝出现播放/直链故障需定位
- [x] 候选清单最终状态表已产出（哪些已落地、哪些作废/不适用、哪些归档）
      → research.md §4：3 项已落地（P1/P2/P5）、2 项作废/不适用（P4/P6）、4 项按用户决定归档（P3/P7/P8/P9）
- [x] `research.md` 已产出且本轮零代码改动（`git status` 仅任务目录）
      → research.md 89 行；`git status --porcelain` 仅本任务目录

## Notes

- `scope=lightweight`，无代码产出。
- **本次核实再次体现"先查证再决定"**：P7 的三项里，一项本方已等价实现（移植＝白做）、一项是本方有意差异（照搬＝回归）、一项是纯卫生；若盲目移植会既浪费又有风险。
- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
