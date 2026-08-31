# 移除自动化其他4项仅保留本地上传

## Goal

按 `audit.md` 方案A，将 `添加执行动作` 5项中的 `刷新目录/执行整理任务/延迟等待/Emby全局刷库` 4项相关的所有内容彻底移除，仅保留 `本地上传`，并发布 `0.0.4`。

## Background

- 审计结论：4 项与 `local_upload` 无强依赖，可安全移除；当前 `domain/service_run` 已半残，前端仍 5 项展示导致“可选但不支持”的不一致。
- 需按 `§4 清单` 后端 7处 + 前端 10处 彻底清理，保持 `go vet/type-check/docker 118MB` 通过。

## Requirements

- **后端**：`domain` 删 `delay/emby_refresh` 仅留 `local_upload`；`service_run` 删 `case delay` + `runDelay` + `actionDisplayName delay`；`validate` 删 `case delay` + 白名单仅 `local_upload`；`service.go` 删 `emby_configs` 空桩；`service_test.go` 若有 `emby` 用例同步。
- **前端**：`AutomationPanel.vue` 删 `ACTION_DEFINITIONS` 4 块（刷新/整理/延迟/Emby）仅留本地上传，删 `import fetchEmbyLibraries`、`Emby/整理` 模板与逻辑 `1117/1326/1393` 等；`api/automation.ts` 删 `organize_tasks/emby_configs`；`api/emby.ts` 整文件删除（若保留 Emby 配置则改为保留，但按彻底移除则删）。
- **可选**：`internal/mediaorganize` 与 `embyproxy` 若确定整功能不保留则删目录（本次按自动化动作维度，`mediaorganize` 目录保留但自动化不再引用，`emby` API 全删）。
- **版本**：`README v0.0.3 → v0.0.4`，`docker 0.0.4/v0.0.4/latest`，`git tag v0.0.4`。

## Constraints

- 所有写操作在 `task.py start` 后，遵循 `AGENTS.md`。
- `internal/cache` 全局缓存保留，不删。

## Acceptance Criteria

- [ ] `grep -rn AutomationActionDelay|EmbyRefresh` 仅注释或 0（`domain` 仅 `local_upload`）
- [ ] `grep -n "case domain.AutomationAction" service_run.go` 仅 `local_upload`
- [ ] `grep -n "fetchEmbyLibraries\|organizeTaskOptions" AutomationPanel.vue ==0`
- [ ] `ls web/src/api/emby.ts` 不存在
- [ ] `cd web && npm run type-check ==0` `GOWORK=off go vet ./... ==0` `docker build 118MB`
- [ ] `README v0.0.4` `docker push 0.0.4` `git tag v0.0.4` 完成，`curl /api/health 200`

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.
