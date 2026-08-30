# Remove announcement feature completely

## Goal

彻底移除 `LitePan` 中公告（`announcement`）相关的所有前后台内容，使 `docker logs` 不再出现 `WARN announcement fetch failed`（`https://www.litepan.top/announcement.json`），且构建与容器仍正常。

## Requirements

- **后端彻底删除**：
  - `rm -rf internal/announcement`（`announcement.go` + `announcement_test.go`，含 `DefaultURL = "https://www.litepan.top/announcement.json"`）
  - `rm internal/api/announcement.go`（`getAnnouncement` / `markAnnouncementRead` 2 handler）
  - `internal/api/router.go`：删 `import announcement`、`Deps{Announcement}`、`Handler{announcement}`、`announcement: d.Announcement`、`r.Get("/announcement")` 2 行
  - `internal/app/wire_http.go`：删 `import announcement`、`announcement.New(DefaultURL, ...)` 创建、`Deps{Announcement}` 注入
  - `internal/settings/registry.go` 的 `KeyAnnouncementReadVersion` 若仅服务于公告则删（需检查 `grep`）
- **前端彻底删除**：
  - `web/src/components/admin/AdminAnnouncementModal.vue`（如存在）及 `AdminView.vue` 中对 `announcement` 的 `useAnnouncement` 引入与 `<AdminAnnouncementModal>` 渲染
  - `web/src/composables/useAnnouncement.ts`（如存在）及 `web/src/api/announcement.ts`（如存在）
  - 清理 `web/src/api/client.ts` 等若引用 `announcement` 则一并清理

## Constraints

- 仅删公告，不动 `文件管理/FUSE/上传/备份` 等核心
- 删除后 `grep -r -i "announcement" --include="*.go" --include="*.ts" --include="*.vue" | grep -v ".trellis" | wc -l` == 0
- `GOWORK=off go vet ./...`、`go build`、`cd web && npm run type-check && npm run build` 必须通过
- `docker build -t litepan-go:noannouncement .` 成功，`docker logs` 无 `announcement fetch failed`

## Acceptance Criteria

- [ ] `ls internal/announcement` → No such file
- [ ] `ls internal/api/announcement.go` 无
- [ ] `grep -r -i "announcement" --include="*.go" | grep -v ".trellis" | wc -l` == 0
- [ ] `GOWORK=off go vet ./...` PASS，`go build -o /tmp/litepan` PASS
- [ ] `cd web && npm run type-check` PASS，`npm run build` PASS
- [ ] `docker build -t litepan-go:noannouncement .` PASS，`docker run` 后 `docker logs | grep -i "announcement fetch failed" | wc -l` == 0 且 `curl /api/health ok`
