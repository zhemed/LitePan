# Remove cross-drive instant transfer

## Goal

彻底移除 LitePan 中“跨盘秒传”（Cross-Drive Transfer / RapidUpload）相关的所有前后台能力，使项目不再包含跨盘探测、秒传、路由、前端入口，且构建与容器仍正常。

## Requirements

- **后端服务彻底删除**：
  - `rm -rf internal/crosstransfer`（4 文件：`service.go/routes.go/methods.go/cleanup_test.go`）
  - `rm internal/api/cross_transfer_admin.go`（5 Handler + NDJSON 流）
  - `internal/app/wire_services.go`：删 `import crosstransfer`、`servicesBundle.crossTransfer` 字段、`crosstransfer.New` 创建、`crossTransfer: crossTransferSvc` 注入
  - `internal/app/wire_http.go`：删 `Deps{ CrossTransfer }` 注入（若有）
  - `internal/api/router.go`：删 `Deps{CrossTransfer}`、`Handler{crossTransfer}`、`crossTransfer: d.CrossTransfer`、`r.Route("/cross-transfer", 5 handler)`、`internal/crosstransfer` import
  - `internal/upload`：保留 `SourceTypeCrossTransfer` 常量与 `SourceAccountID` 字段（仅为历史兼容，`upload.Manager` 不再接收跨盘来源），但 `upload` 中 `blockingCrossTransferDriver` 等测试专用 mock 保留；`driver.Config` 的 `ProvideHashes/RapidUploadHashes` 保留（驱动声明，非跨盘专有）
- **前端彻底删除**：
  - `rm web/src/api/crossTransfer.ts`（`listCrossTransferRoutes/scan/probe/execute` 等 5 导出）
  - `rm web/src/components/admin/CrossDriveTransfer.vue` + `web/src/components/admin/CrossTransferTree.vue` + `CrossTransferProbeNoticeDialog.vue`（如存在）
  - `web/src/components/admin/CrossTransfer*` 相关在 `AdminView` 或 `FileBrowser` 的入口/按钮、路由、菜单项一并删
  - 清理 `web/src/api/files.ts` 等若引用 `crossTransfer` 则一并清理

## Constraints

- 保留 `internal/file`、`internal/upload` 原有非跨盘逻辑（普通上传、离线下载等）
- 保留 `internal/driver` 的 `ProvideHashes` 字段（驱动能力声明，非仅跨盘）
- 删除后 `grep -r -i "crosstransfer|CrossTransfer|CrossDrive|跨盘" --include="*.go" --include="*.ts" --include="*.vue" | grep -v ".trellis" | wc -l` 为 0（除 `upload.SourceTypeCrossTransfer` 常量注释外，视作 0）
- `GOWORK=off go vet ./...`、`go build`、`cd web && npm run type-check && npm run build` 必须通过
- `docker build -t litepan-go:nocross .` 成功，`curl /api/health 200` 且 `curl -i /api/cross-transfer/routes` 404

## Acceptance Criteria

- [ ] `ls internal/crosstransfer` → No such file
- [ ] `ls internal/api/cross_transfer_admin.go` 无
- [ ] `grep -r "crosstransfer" --include="*.go" | grep -v ".trellis" | wc -l` == 0
- [ ] `web/src/api/crossTransfer.ts` 已删，`CrossDriveTransfer.vue` 已删
- [ ] `internal/api/router.go` 无 `CrossTransfer` 字段与 `/cross-transfer` 路由，`curl -b cookie /api/cross-transfer/routes` 404
- [ ] `GOWORK=off go vet ./...` PASS，`go build -o /tmp/litepan` PASS
- [ ] `cd web && npm run type-check` PASS，`npm run build` PASS
- [ ] `docker build -t litepan-go:nocross .` PASS，`curl /api/health ok`
