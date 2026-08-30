# Refactor README for actual stripped build

## Goal

先移除 `README.md` 中与当前代码不一致的已删功能描述（STRM、跨盘秒传、目录整理、增强工具 7 项、WebDAV 共享、8 驱动等），再按 `litepan-go:three-drivers 118M` 的**实际精简后**能力重构 `README.md`，使文档与镜像一致，可直接 `git clone https://github.com/zhemed/LitePan.git` 后 `docker compose` 启动。

## Requirements

- **先移除（与 `grep 0` 的代码现状对齐）**：
  - 功能简述表格中的 `跨盘秒传`、`目录整理` 2 格（含 `feature-crosstransfer.png` / `feature-organize.png`）—— `internal/crosstransfer`、`mediaorganize` 已删
  - `挂载与更多功能` 段的 `WebDAV`、`302 直链、缓存保持、命名对齐` 等已删描述（`internal/share/dav`、`cacheretention` 等）
  - 驱动相关：原 `多网盘聚合` 的“多网盘”泛指，需改为仅 3 驱动；`THIRD_PARTY_NOTICES` 中 8 已删驱动的致谢若提及则同步
  - 镜像标签：`ponphil/litepan:beta` / `latest 是 Python 旧版` 的指向已过时，改为 `zhemed/LitePan` 与 `litepan-go:three-drivers`
- **再重构（按实际）**：
  - 顶部 `banner` 与 `badge` 保留，但 `version-shield` 改为 `v0.6.0-lite`（或 `three-drivers`），`docker-pulls` 指向 `zhemed/litepan`（或 `ghcr.io/zhemed/litepan` 若后续推）
  - `功能简述` 表格 4→2 或 4→3 格：保留 `多网盘聚合（仅 115/天翼云盘/本机存储）` 与 `自动联动（仅 delay）`，`跨盘秒传/目录整理` 去除，图片仅保留 `feature-browser.png` / `feature-automation.png`（或替换为精简后截图）
  - `挂载与更多功能` 改为 `FUSE 本地挂载 + 从服务器上传`（`LocalUpload` 唯一保留的增强工具），去除 `WebDAV` 与已删 7 项
  - `快速开始` 的 `docker-compose.yml` 示例：`image: zhemed/litepan:latest`（或 `litepan-go:three-drivers`），`ports` 保留 `5211 42069`，`volumes` 保留 `data/mounts`，去除 `fuse_read_cache` 可选映射中已删提及，`extra_hosts` 的 `tmdb` 段落去除（`classifyorganize` 已删）
  - `支持网盘` 列表：明确列出 `115网盘Open / 天翼云盘 (189Cloud) / 本机存储 (LocalFs)` 3 项，其余 8 标注已移除
  - `许可` 段保留 `PolyForm Noncommercial`，但 `ponphil/litepan` 链接改为 `zhemed/LitePan`

## Constraints

- 仅改 `README.md`（与 `docs/pictures` 若删图则一并），不改 `internal/*` / `web/*` 代码
- 保持 `README.md` 的 `zh-CN` 为主、`a name="readme-top"` 锚点、`AGENTS.md` 的 `所有操作必须调用 trellis` 不受影响
- `grep -r "跨盘秒传|目录整理|STRM|WebDAV.*dav" README.md | wc -l` == 0（或仅在“已移除”备注中）
- `cat README.md | grep -c "115.*天翼.*本机"` >=1

## Acceptance Criteria

- [ ] `README.md` 无 `跨盘秒传`、`目录整理` 表格格子与对应图片引用
- [ ] `README.md` 无 `WebDAV`（`internal/share/dav`）与 `缓存保持` 描述
- [ ] `README.md` 的 `快速开始` 示例 `image:` 为 `zhemed/litepan` 或 `litepan-go:three-drivers`，非 `ponphil/litepan:beta`
- [ ] `README.md` 明确列出仅 3 驱动：`115网盘Open / 天翼云盘 / 本机存储`，其余 8 标注已移除
- [ ] `README.md` 的 `功能简述` 与 `挂载与更多功能` 与当前 `118M` 镜像能力一致，`GOWORK=off go vet` 仍 PASS（文档不影响构建）

## Notes

- 本任务为文档重构，`data/litepan.db` 与 `admin/123456` 不变
- 图片 `docs/pictures/feature-crosstransfer.png` 等若不再引用，可 `git rm` 但保留 `banner.png`
