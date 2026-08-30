# Keep only 115 Tianyi LocalFs drivers

## Goal

存储管理（`drivers/` 与 `internal/driver` 前端驱动卡片）**仅保留 3 驱动**：`115_Open`（115网盘）、`189Cloud`（天翼云盘）、`LocalFs`（本机存储），将其余 8 驱动的前后台、文档、测试等所有相关内容彻底移除，使构建仅包含 3 驱动。

## Requirements

- **保留 3 驱动**：
  - `drivers/115_Open`（115网盘）、`drivers/189Cloud`（天翼云盘，`189Cloud`）、`drivers/LocalFs`（本机存储）
  - `drivers/all.go` 仅 `import _ "litepan/drivers/115_Open"` / `189Cloud` / `LocalFs` 3 行
  - `internal/driver` 的 `registry.go` 自动发现保留的 3 个 `Config.Name`（`115_open`、`189cloud`、`local_fs` 或实际 `Config.Name`）
  - 前端 `web/src/api/drivers.ts` / `DriverPicker` 自动仅展示 3 卡片（后端 `GET /admin/drivers` 返回 3）
- **彻底移除 8 驱动**：
  - `drivers/123_Open`（123网盘）、`drivers/139Cloud`（139云盘）、`drivers/Baidu_Open`（百度网盘）、`drivers/Guangya`（广雅？）、`drivers/OneDrive`、`drivers/OpenList`、`drivers/Quark`（夸克网盘）、`drivers/WebDAV`（WebDAV 远端挂载，**非** `internal/share/dav` 已删的本地 WebDAV）
  - 每个驱动目录含 `driver.go/config.go/auth.go/ops.go/transport.go/upload.go/...` 全删
  - `drivers/template` 保留（模板，不计入）
- **关联清理**：
  - `README.md` / `docs/` 中驱动列表提及的 8 驱动名称同步删或标注已移除
  - `web` 中若有驱动硬编码 `icon/logo` 映射（如 `DriverPicker` 的 `logoSrc`）无需手动改，后端不返回则不展示
  - `internal/driver` 的 `ProvideHashes` 等驱动声明随驱动一起消失，无需额外改

## Constraints

- 仅删驱动，不动 `internal/share/fuse`（本地挂载）、`internal/cache`、`internal/file` 等
- 删除后 `ls drivers/` 仅 `115_Open 189Cloud LocalFs all.go template` 5 项（`template` 保留）
- `grep -r "123_Open|139Cloud|Baidu_Open|Guangya|OneDrive|OpenList|Quark|WebDAV" --include="*.go" drivers/ | wc -l` == 0（除 `all.go` 的 3 保留外）
- `GOWORK=off go vet ./...`、`go build`、`cd web && npm run type-check && npm run build` 必须通过
- `docker build -t litepan-go:three-drivers .` 成功，`curl /api/health 200` 且 `GET /admin/drivers` 仅返回 3 驱动

## Acceptance Criteria

- [ ] `ls drivers/` 仅 `115_Open 189Cloud LocalFs all.go template`（`123_Open` 等 8 目录已 `rm -rf`）
- [ ] `cat drivers/all.go` 仅 3 个 `import _ "litepan/drivers/...`（115、189、LocalFs）
- [ ] `grep -r "drivers/123" --include="*.go" | wc -l` == 0
- [ ] `GET /admin/drivers -b cookie` 返回 `3` 项且 `name` 为 `115_open/189cloud/localfs`（或实际 Config.Name）
- [ ] `GOWORK=off go vet ./...` PASS，`go build -o /tmp/litepan` PASS
- [ ] `cd web && npm run type-check` PASS，`npm run build` PASS
- [ ] `docker build -t litepan-go:three-drivers .` PASS，`curl /api/health ok`

## Notes

- `WebDAV` 驱动（远端挂载）与已删 `internal/share/dav`（本地 WebDAV 服务）不同，需区分；本次删的是 `drivers/WebDAV` 远端驱动
- `129`? 用户未提及 `123_Open` 等 8 个均删，仅留 115/189/LocalFs
