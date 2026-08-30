# Remove all file sharing related features

## Goal

彻底移除 LitePan 中 **文件共享** 相关全部能力（`internal/share` 的 WebDAV DAV 服务与 FUSE 共享后端、`FileShareManagement` 页面及 `WebDAV`/`本地挂载` 两 tab、`WebDAVSettings`、`share` 相关 API/缓存/设置），使项目不再暴露任何 `/dav`、`/api/admin/webdav-config`、`FileShareManagement` 等入口，且构建与容器仍正常。

## Requirements

- **后端彻底删除**：
  - 删除目录 `internal/share/` 整体（`dav/` 16 文件：`server.go/filesystem.go/path.go/webdav_cache.go/...` + `fuse/` 11 文件：`backend.go/nodes.go/...`）
  - 删除 `internal/api/router.go` 中 `share/dav` 导入、`davLog`、`r.Post("/webdav-config", h.adminWebDAVConfig)`、`r.Mount("/dav", ...)` 旁路及 WebDAV 认证/日志
  - 删除 `internal/api/auth.go` 中 `adminWebDAVConfig` handler 与 `WebDAVConfigRequest` 调用
  - 删除 `internal/adminauth/service.go` 中 `KeyWebDAVEnabled`、`WebDAVEnabled` 字段、`WebDAVConfigRequest`、`UpdateWebDAVConfig()`、`webdavEnabled()` 及 `SystemConfig` 中 `webdav_enabled`
  - 删除 `internal/settings/registry.go` 中 `KeyWebDAVCacheEnabled` 及 `boolSpec` 定义
  - 删除 `internal/cache/webdav_keys.go`、`cache/keys.go` 中 `prefixWebDAVMeta`、`cache/cleaner.go` 中 `InvalidateWebDAV*` 调用、`cache/webdav_keys.go` 整体
  - 删除 `internal/logx/module.go` 中 `ModuleWebDAV` 及相关 case
  - 删除 `internal/file/service.go` 中 WebDAV 路径缓存 TTL 注释中 WebDAV 提及（若仅注释可保留，但需无 `webdav` 字样）
  - 删除 `internal/playback` 中 `WebDAV` intent 字段的共享相关用法（保留海报取帧逻辑，但移除 WebDAV 分支）
  - 清理 `grep -r -i -n "share/dav\|webdav" --include="*.go"` 残留（约 60+ 处）
- **前端彻底删除**：
  - 删除 `web/src/components/admin/FileShareManagement.vue`（文件共享主页面）
  - 删除 `web/src/components/admin/WebDAVSettings.vue`（若存在，或为 `FileShareManagement` 内嵌则随主文件删除）
  - 删除 `web/src/views/AdminView.vue` 中 `share` 页面路由：`adminPageLoaders.share`、`FileShareManagement` 异步组件、`{key:"share", label:"文件共享"}`、`share: {defaultTab:"webdav", tabs:{webdav,fuse}}` 及 `<FileShareManagement v-else-if="page==='share'" />`
  - 删除 `web/src/api` 中若存在的 `share` 相关 API（`share.ts` 若存在）
  - 清理 `web/src` 中 `share` 相关 `grep -r -i "share" --include="*.ts" --include="*.vue"` 残留（排除 `shared` 在 `docker-compose.yml` 的 `mounts:shared` 为容器挂载传播选项，非功能，可保留；但需区分 `FileShare` 与 `shared`）
- **配置与部署**：
  - 删除 `README.md`、`docs/` 中文件共享/WebDAV 功能介绍
  - 保留 `drivers/WebDAV`（远端 WebDAV **挂载**驱动，非本地共享，与 `internal/share/dav` 的 **对外 WebDAV 服务** 不同，予以保留）
  - 保留 `internal/fusemount`（本地 FUSE 挂载，非 `internal/share/fuse` 的共享后端）
  - 保留 `docker-compose.yml` 中 `mounts:shared`（为 Docker 传播模式，非功能文案）
- **兼容性**：
  - 不删除用户已通过 WebDAV 上传的文件（仅移除服务，文件仍在对应网盘账号中）
  - 旧 `webdav_enabled` 配置项在新版启动时忽略（`log WARN unknown config webdav_enabled ignored` 或静默）

## Constraints

- 不得误删 `drivers/WebDAV`（WebDAV **客户端**驱动）与 `internal/fusemount`（FUSE **挂载**），仅删 `internal/share`（**对外共享**服务）
- 删除后 `grep -r -i "internal/share\|FileShare\|webdav" --include="*.go" --include="*.ts" --include="*.vue" | grep -v ".trellis" | grep -v ".git" | grep -v "node_modules"` 必须为 0（排除 `mounts:shared` 的 `shared` 若被误判，需精确匹配 `FileShare`/`WebDAV`/`internal/share`）
- `GOWORK=off go vet ./...`、`go build`、`cd web && npm run type-check && npm run build` 必须通过
- 容器 `docker build -t litepan-go:noshar .` 成功，`curl /api/health ok` 且 `curl -i http://127.0.0.1:5211/dav/` 返回 404（原 WebDAV 应 401/207，现应 404），`curl /api/admin/webdav-config` 404

## Acceptance Criteria

- [ ] `rm -rf internal/share` 后 `ls internal/share` 无此目录
- [ ] `grep -r -n "internal/share" --include="*.go" | grep -v ".trellis"` 为 0
- [ ] `internal/api/router.go` 无 `share/dav` 导入、`davLog`、`webdav-config`、`Mount.*dav` 旁路
- [ ] `internal/adminauth/service.go` 无 `KeyWebDAVEnabled`、`WebDAVEnabled`、`UpdateWebDAVConfig`
- [ ] `internal/cache/webdav_keys.go` 已删除，`grep -r webdav --include="*.go" | grep -v ".trellis"` 为 0（排除驱动 WebDAV 的 `drivers/WebDAV` 保留，故此条仅限 `internal/cache`/`internal/api` 等共享侧，驱动可保留）
- [ ] `web/src/components/admin/FileShareManagement.vue` 已删除，`AdminView.vue` 无 `share` 页面及 `FileShareManagement` 引用，`grep -r FileShare --include="*.vue" --include="*.ts"` 为 0
- [ ] `GOWORK=off go vet ./...` PASS，`go build -o /tmp/litepan` PASS
- [ ] `cd web && npm run type-check` PASS，`npm run build` PASS 且产物无 `FileShare` 字样
- [ ] `docker build -t litepan-go:noshar .` PASS，`docker run -d --name litepan-test2 -p 5213:5211 ... litepan-go:noshar` 后 `curl /api/health ok` 且 `curl -i /dav/` 404、`curl -i /api/admin/webdav-config` 404
- [ ] `docker logs litepan` 无 `ModuleWebDAV`

## Notes

- Complex task：需 `design.md`（边界、数据流、兼容、回滚）与 `implement.md`（有序清单、验证）后方可 `task.py start`
- 影响面：`internal/share` 27 文件 + `FileShareManagement` 等前端，约 30+ 文件

