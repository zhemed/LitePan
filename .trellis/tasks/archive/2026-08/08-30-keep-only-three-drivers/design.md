# Design: Keep only 115 Tianyi LocalFs drivers

## Overview

驱动为插件式：`drivers/<Name>/driver.go` 的 `init()` 向 `internal/driver/registry.go` 注册 `Config.Name`，`drivers/all.go` 通过空导入触发 `init`。`GET /admin/drivers` 直接读 `registry`。删除时仅需 `rm -rf` 目录 + 改 `all.go`，无需改 `internal/driver` 逻辑。

## Boundaries

| 层 | 删除 | 保留 |
|---|---|---|
| **drivers/** | `123_Open`、`139Cloud`、`Baidu_Open`、`Guangya`、`OneDrive`、`OpenList`、`Quark`、`WebDAV` 8 目录（含各 `driver.go/config.go/auth.go/ops.go/transport.go/upload.go/...`） | `115_Open`、`189Cloud`、`LocalFs` 3 目录 + `all.go`（3 导入）+ `template`（模板） |
| **drivers/all.go** | 8 个 `import _ "litepan/drivers/XXX"` 行 | 3 个保留驱动的导入 |
| **internal/driver** | 无 | `driver.go` 的 `Config` 结构、`registry.go` 自动发现保留的 3 个 |
| **API** | 无（`internal/api/drivers.go` 的 `GET /admin/drivers` 自动仅返回 3） | `internal/api/drivers.go` 保留 |
| **web** | 无（`DriverPicker` 自动仅展示 3） | `web/src/api/drivers.ts` 保留 |
| **docs** | `README.md` / `docs/pictures` 中 8 驱动提及（若存在） | 3 保留驱动的文档 |

## Data Flow

```
Before: drivers/all.go imports 11 → registry 11 Config → GET /admin/drivers → 11 cards
After:  drivers/all.go imports 3 → registry 3 Config → GET /admin/drivers → 3 cards (115/189/LocalFs)
```

- `internal/store` 的 `cloud_accounts` 表中旧 `driver_type` 为已删驱动的行保留，但 `ListAccounts` 后 `registry` 找不到对应 `Config` 会显示 `未知驱动`，不 panic

## Compatibility

- **DB**：`cloud_accounts.driver_type` 旧值（如 `quark`）不再有 `Config`，前端显示为 `未知`，`Ping/List` 会 `driver not found`，用户需手动删该账号
- **Config**：`drivers/<Name>/config.go` 的 `Addition` 随目录一起删，无残留 `settings` 键（驱动配置存 `cloud_accounts.config` JSON，非全局 `settings`）
- **Build**：`go vet` 的 `depguard` 仍满足（`drivers` 不导入 `internal/file` 等），`drivers/template` 不参与构建

## Tradeoffs

- **彻底删 vs 仅改 all.go**：用户要求“相关的所有内容彻底移除”，故 `rm -rf` 目录，非仅注释 `all.go`，减镜像与维护成本，日后恢复需 `git revert`
- **保留 template**：`drivers/template` 为脚手架，不计入业务，保留以便后续加新驱动

## Rollout / Rollback

- 单提交 `refactor(drivers): keep only 115 189 LocalFs`，`git revert` 即恢复 8 驱动
- 容器：`docker build -t litepan-go:three-drivers` 后 `curl -b cookie /api/admin/drivers | jq .[].name` 仅 3

## File Map (Deletion Order)

1. `rm -rf drivers/123_Open drivers/139Cloud drivers/Baidu_Open drivers/Guangya drivers/OneDrive drivers/OpenList drivers/Quark drivers/WebDAV`
2. `edit drivers/all.go` 缩为 3 导入
3. `grep` sweep + `go vet` + `web build` + `docker build`
