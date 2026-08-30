# Implementation Plan: Remove announcement

## Overview

按 `design.md` 自底向上删除。

## Phase 1: 后端服务

- [ ] 1.1 `rm -rf internal/announcement`
  - `verify: ls internal/announcement` → No such file
- [ ] 1.2 `rm internal/api/announcement.go`
- [ ] 1.3 `internal/app/wire_http.go`：删 `import announcement`、`announcement.New` 创建与 `Deps{Announcement}` 注入
  - `verify: grep -n "announcement" internal/app/wire_http.go` == 0
- [ ] 1.4 `internal/api/router.go`：删 `import announcement`、`Deps/Handler/announcement` 与 `2` 路由
  - `verify: grep -n "announcement" internal/api/router.go` == 0

## Phase 2: 前端

- [ ] 2.1 `rm web/src/components/admin/AdminAnnouncementModal.vue` 等
- [ ] 2.2 清理 `web/src/views/AdminView.vue` 等对 `announcement` 的引用
  - `verify: grep -rn "announcement" --include="*.ts" --include="*.vue" web/src | wc -l` == 0

## Phase 3: Sweep & Verification

- [ ] 3.1 `grep -r -i "announcement" --include="*.go" --include="*.ts" --include="*.vue" | grep -v ".trellis" | wc -l` == 0
- [ ] 3.2 `GOWORK=off go vet ./...` PASS
- [ ] 3.3 `cd web && npm run type-check` PASS, `npm run build` PASS
- [ ] 3.4 `docker build -t litepan-go:noannouncement .` PASS，`docker logs | grep announcement` == 0

## Phase 4: Commit & Archive

- [ ] 4.1 `git add -A && git restore --staged .trellis/tasks/08-30-remove-announcement && git commit -m "refactor(announcement): remove announcement completely"`
- [ ] 4.2 `task.py archive ... --skip-branch-validation && git add ... && git commit`
- [ ] 4.3 `add_session.py --commit <hash>`
