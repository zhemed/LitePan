# 彻底移除磁力与临时下载

## Goal

按 `report.md` 清单彻底移除磁力下载端口与临时下载目录相关所有内容，并发布 `0.0.7`。

## Background

- 报告结论：与 `local_upload` 零耦合，可彻底移除。

## Requirements

- **后端**：`internal/settings/registry.go` 删 3 键常量 + 2 spec，`go.mod` 删 `anacrolix/torrent` + `go mod tidy`，`docker-compose.yml/.fnos.yml` 删 `42069` 端口段。
- **前端**：`web/src/components/upload/UploadTaskSettingsPanel.vue` 整文件删，`web/src/components/admin/SystemSettings.vue` 删 3 键名数组。
- **版本**：`README v0.0.6→v0.0.7`，`docker 0.0.7`。

## Constraints

- 所有写操作在 `task.py start` 后。
- `GOWORK=off go vet` `type-check` 必须 0。

## Acceptance Criteria

- [ ] `grep builtin_offline --include=*.go | wc -l ==0`（registry 除外应 0）
- [ ] `grep anacrolix go.mod ==0`
- [ ] `grep 42069 docker-compose.yml ==0` 或 保留但注释说明
- [ ] `ls UploadTaskSettingsPanel.vue` 不存在
- [ ] `GOWORK=off go vet 0` `type-check 0` `docker build 104MB`

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.
