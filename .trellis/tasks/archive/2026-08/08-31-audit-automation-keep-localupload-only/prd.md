# 排查自动化动作仅保留本地上传

## Goal

以截图 `de6ccfb 512x440 5项` 为准，排查 `添加执行动作` 中除 `本地上传` 外的 4 项（`刷新目录` `执行整理任务` `延迟等待` `Emby全局刷库`）是否可仅保留 `本地上传` 并安全彻底移除相关所有内容（含后端/前端/DB/配置），输出逐项依赖与风险。

## Background

- 截图：`刷新目录（绿刷）` `执行整理任务（蓝整理）` `延迟等待（橙钟）` `本地上传（蓝上传）` `Emby全局刷库（紫Emby）` 5 项。
- 现状 `LitePan main v0.0.3` 的 `domain/automation.go` 保留 `3 枚`：`delay / local_upload / emby_refresh`，而 `organize/cache_clear` 已在 `精简` 中移除但 `delay/emby` 仍在；`service_run.go` 已 `delay/local_upload` 2 分支，`Emby` 仅常量；`web/AutomationPanel` 仍渲染 5 项（含已移除的整理/刷新）。
- 用户问题：能不能只留 `本地上传`，其他能不能安全地相关所有内容彻底移除。

## Requirements

- **映射**：每项 `刷新目录/整理/延迟/Emby` 对应 `domain 常量`、`service_run 分支`、`service_validate 分支`、`internal/*` 服务（`mediaorganize/cache/embyproxy`）、`web` 定义（`AutomationPanel ACTION_DEFINITIONS`、图标、描述）、`configs` 键、`事件/订阅` 依赖。
- **安全性**：判定是否“安全” = 无其他功能依赖、无 DB 残留被外键引用、无 `type-check/go vet` 破坏、无 `LocalUpload` 精简后依赖（如 `delay` 是否被 `LocalUpload` 串联需要）。
- **产出**：`audit.md` 含 5 项对照表（保留/可删/风险）+ 彻底移除清单（文件/行/配置）+ 建议（是/否/保留哪几项）。

## Constraints

- 只读排查，不实际删代码；写仅限 `audit.md`。
- 遵循 `AGENTS.md`：`task.py start` 后执行。

## Acceptance Criteria

- [ ] `audit.md` 存在，5 项逐项有“可否只留本地上传”结论与依据
- [ ] 列出若只留 `local_upload` 需删除的文件/常量/分支/前端定义清单
- [ ] 给出最终建议：可/不可/或折中（如保留 `delay`）

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.
