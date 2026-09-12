# 09-12-investigate-mediaorganize-necessity

## Goal

只读判定 `09-12-investigate-upstream-updates` 报告中的 **P6（移植上游 `20b66de` "修复目录整理误匹配"）在本方是否还有必要**：追溯 `internal/mediaorganize/rules` 的引用链与功能入口（API 路由 / 前端 / 自动化动作 / 其他模块 import），给出「跳过 / 清理孤儿包 / 仍应移植」的结论与依据。

## Background

- 用户提出：本方是精简分支，"目录整理"应该已经删了，为什么还会有 P6。
- 本方既有事实：`1bcfac8 refactor(cache,organize): remove cache tasks and directory organization` 已移除目录整理功能；但仓库里仍存在 `internal/mediaorganize/rules/`（19 个文件，约 3,796 行非测试代码）。
- 上游 `20b66de` 改动 `internal/mediaorganize/rules/tmdb.go` + `tmdb_test.go`（另有 2 个已删驱动的测试文件），当时按「改动文件是否存在于本方 tree」判定为"候选 P6"——**该判定只看了文件是否存在，未验证代码是否可达**，本任务即是对此的复核。

## Requirements

- **R1 只读**：不改任何代码/配置；仅检索、阅读与 `git log` 取证。
- **R2 引用链证据**：给出四类证据——① Go import 链（谁能到达该包）；② HTTP 路由；③ 前端入口；④ 自动化/任务动作类型。
- **R3 处置结论**：明确 P6 是否仍必要；若判定为"死代码"，给出可选的后续动作（清理孤儿包）及其边界、风险与不做清理的理由对照。
- **R4 记录**：产出 `research.md`（证据 + 命令 + 结论），并在记录中**更正**调查报告 P6 条目的判定口径（"文件存在"≠"代码可达"）。

## Constraints

- 不实施任何清理/删除（如需删除，另建任务并走完整流程）。
- 不触碰生产机、不改容器与数据。
- 结论以代码事实为准；无法验证的部分如实标注。

## Acceptance Criteria

- [x] 已给出 `internal/mediaorganize/rules` 的 import 引用链证据（谁 import、是否仅自测引用）
      → research.md §2.1：全仓仅 1 处**注释**提及；`go list -deps ./cmd/litepan | grep -c mediaorganize` = **0**（不在二进制依赖图）
- [x] 已核对 API 路由 / 前端 / 自动化动作三类入口是否存在
      → research.md §2.2：路由空、前端源码与构建产物均空、自动化动作只剩 `AutomationActionLocalUpload`
- [x] 已给出 P6 是否必要的明确结论
      → **不需要**：`rules` 为死代码（`1bcfac8` 同时移除了 `name_align` 对它的依赖），P6 从候选清单作废（9 项→8 项）
- [x] 已给出可选的清理建议（边界/风险/收益）或说明为何不清理
      → research.md §4：建议 A（删孤儿包 + 同步 spec，推荐）/ 建议 B（保留）；本轮不实施
- [x] `research.md` 完成并含命令与证据；本轮零代码改动
      → `git status` 仅新增任务目录；未改任何代码/配置
- [x] 已在记录中更正调查报告 P6 的判定口径
      → research.md §3 更正表（含报告 §3.2/§6、journal-1.md:252、spec `directory-structure.md:44` 三处过期表述）

## Notes

- **根因（为何上轮会误判 P6）**：上游调查报告用"改动文件是否出现在本方 tree"作为适用性判据——这只证明"文件存在"，不证明"代码可达"。本次补上可达性验证（import 链 + 依赖图 + 三类入口）。
- **可复用的判据（供后续调查沿用）**：候选移植项判定应为「文件存在 **且** 存在可达入口/消费者」；对 Go 包可用 `go list -deps ./cmd/litepan | grep <pkg>` 一票否决。
- **顺带发现**：spec `directory-structure.md:44` 与 `concurrency-and-scheduling.md:41` 仍把 `mediaorganize` 描述为保留/在用，与实际不符——建议在做清理任务时一并修正（本轮不动）。
- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
