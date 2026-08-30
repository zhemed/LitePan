# Implementation Plan: Remove cross-drive instant transfer

## Overview

按 `design.md` 自底向上删除，确保 `go vet` 逐步通过。

## Phase 1: 后端服务

- [ ] 1.1 `rm -rf internal/crosstransfer`（4 文件）
  - `verify: ls internal/crosstransfer` → No such file
- [ ] 1.2 `rm internal/api/cross_transfer_admin.go`
  - `verify: ls internal/api/cross_transfer_admin.go` → No such file
- [ ] 1.3 `internal/app/wire_services.go`：删 `import crosstransfer`、`servicesBundle.crossTransfer`、`crosstransfer.New` 创建与注入
  - `verify: grep -n "crosstransfer" internal/app/wire_services.go` == 0
- [ ] 1.4 `internal/app/wire_http.go`：若含 `CrossTransfer` 注入则删
  - `verify: grep -n "crossTransfer\|CrossTransfer" internal/app/wire_http.go` == 0

## Phase 2: 路由

- [ ] 2.1 `internal/api/router.go`：删 `import crosstransfer`、`Deps{CrossTransfer}`、`Handler{crossTransfer}`、`h.crossTransfer: d.CrossTransfer`、`r.Route("/cross-transfer", 5 handler)`
  - `verify: grep -n "crosstransfer\|cross-transfer" internal/api/router.go` == 0
  - `verify: grep -n "crossTransfer" internal/api/router.go` == 0

## Phase 3: 前端

- [ ] 3.1 `rm web/src/api/crossTransfer.ts`
- [ ] 3.2 `rm web/src/components/admin/CrossDriveTransfer.vue web/src/components/admin/CrossTransferTree.vue web/src/components/admin/CrossTransferProbeNoticeDialog.vue`（存在则删）
- [ ] 3.3 清理 `web/src/views/AdminView.vue` 或 `FileBrowser.vue` 中对 `CrossDriveTransfer` 的 `import/route/menu`
  - `verify: grep -rn "crossTransfer|CrossTransfer|CrossDrive" --include="*.ts" --include="*.vue" web/src | wc -l` == 0

## Phase 4: Sweep & Verification

- [ ] 4.1 `grep -r -i "crosstransfer|CrossTransfer" --include="*.go" --include="*.ts" --include="*.vue" | grep -v ".trellis" | wc -l` == 0
- [ ] 4.2 `GOWORK=off go vet ./...` PASS
- [ ] 4.3 `cd web && npm run type-check` PASS, `npm run build` PASS
- [ ] 4.4 `GOWORK=off go build -o /tmp/litepan ./cmd/litepan` PASS
- [ ] 4.5 `docker build -t litepan-go:nocross .` PASS, `docker run -d -p 5217:5211 ...` + `curl /api/health 200` + `curl -i /api/cross-transfer/routes` 404 + `curl -b cookie /api/admin/tools/local-upload/config 200`

## Phase 5: Commit & Archive

- [ ] 5.1 `git add -A && git restore --staged .trellis/tasks/08-30-remove-crosstransfer && git commit -m "refactor(crosstransfer): remove cross-drive instant transfer"`
- [ ] 5.2 `task.py archive 08-30-remove-crosstransfer --skip-branch-validation && git add .trellis/tasks/archive/... && git commit`
- [ ] 5.3 `add_session.py --commit <hash>`

## Rollback

- `git revert <refactor commit>`
