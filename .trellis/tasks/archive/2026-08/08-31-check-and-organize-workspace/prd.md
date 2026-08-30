# 检查并整理 LitePan 工作区

## Goal

对 `/root/LitePan` 做一次只读为主的全面巡检，输出结构、残留、忽略、构建产物与一致性清单，并对可安全清理项执行整理，保持 `工作区只有LitePan` 且 `Clean`。

## Background

- 刚移除 `nested LitePan-own 57M`，当前 `LitePan main ae853bc v0.0.3`，历史含 `STRM/share/cache/organize/announcement` 大量精简、前端重建、驱动精简与 `LitePan-own` 适配，`data/litepan.db` 与 `internal/api/web` 为产物。
- 用户要求建立任务检查并整理工作区，需给出可验证的清单而非口头结论。

## Requirements

- **结构巡检**：`ls -1` 顶层、`du -sh` 分大项、`git status --porcelain`、`git ls-files --others --exclude-standard`、`cat .gitignore` 命中、`ls _extracted`、`ls data`、`ls internal/api/web/assets` 数量、`web/node_modules` 是否存在、`docker images` 相关。
- **残留与一致性**：`grep -R` 残留 `strm/share/crosstransfer/announcement` 关键字（应 0）、`grep LitePan-own` 仅 `_extracted/.trellis` 历史、`go vet` / `type-check` 仍通过、`docker-compose.yml` vs `README` 标签一致性、`internal/api/web` 是否与 `web` 构建一致。
- **整理执行**（仅安全项）：`web/node_modules` 若无用可提示、`_extracted` 已忽略保留、`data/*.wal-shm` 保留、`internal/api/web/assets` 已受控不手删；清理仅针对 `*.tmp/*.bak/*__pycache__` 等明显残留，或 `git clean -nd` 预览。
- **产出**：`report.md` 含 ① 结构表 ② 巡检表（逐项 ✅/⚠️）③ 整理动作（如有）④ 结论与建议。

## Constraints

- 只读巡检优先；写操作仅限 `report.md` 与明显安全的 `git clean -f`（需列出预览并经任务内执行）。
- 遵循 `AGENTS.md 项目强制规则`：`task.py start` 后执行。

## Acceptance Criteria

- [ ] `report.md` 存在，含四节
- [ ] `git status` 最终 `Clean`（除任务目录 `planning/in_progress`）
- [ ] `go vet` 与 `type-check` 复验通过
- [ ] 给出是否需进一步整理的明确建议

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.
