# 调查磁力与临时下载是否可移除

## Goal

以截图 `c8584f 884x237`（`磁力下载端口 42069 / 临时下载目录 data/builtin_offline`）为靶，审计其相关所有内容是否可安全彻底移除，并评估与 `local_upload` 的关联。

## Background

- 截图为 `系统设置 → 性能/其他` 内的 `磁力下载端口` 与 `临时下载目录`（`data/builtin_offline`），对应 `settings KeyBuiltinOffline*` 与 `anacrolix/torrent` 内置下载器。
- 此为 `builtin_offline`（磁力/BT）功能，与刚移除的 `offline_download`（云盘离线）不同，需单独审计。

## Requirements

- **全量扫描**：`grep -R builtin_offline|BTPort|TempDir|anacrolix.*torrent` 在 `backend/frontend` 的文件清单。
- **依赖**：`settings` 3 键、`internal/offlinedownload`？`torrent` 库、`docker 42069`、`data/builtin_offline` 卷、与 `local_upload` 是否耦合。
- **DB/配置**：`configs builtin_offline_*` 是否有存量，`docker-compose 42069` 是否必需。
- **产出**：`report.md` 含清单、依赖图、可否结论与待删列表。

## Constraints

- 只读，不改代码；写仅限 `report.md`。
- 遵循 `AGENTS.md`：`task.py start` 后执行。

## Acceptance Criteria

- [ ] `report.md` 存在，含清单、依赖、可否结论与待删文件表

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.
