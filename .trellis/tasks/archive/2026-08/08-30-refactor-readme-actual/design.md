# Design: Refactor README for actual stripped build

## Overview

当前 `README.md` 为 `Ponphil` 原版 Go 版（`v0.5.2-Beta`）的全功能文档，含 `STRM`、`WebDAV`、`crosstransfer`、`mediaorganize` 等已删能力，与 `litepan-go:three-drivers 118M` 的仅 3 驱动 + `LocalUpload` 现状严重不一致。需先做减法（删 4 格中 2 格及挂载段），再做加法（明确 3 驱动与新镜像）。

## Boundaries

| 段 | 删除 | 重构为 | 依据 |
|---|---|---|---|
| **顶部 badge** | `ponphil/litepan` 的 `docker-pulls`/`version` 指向 | `zhemed/LitePan` 的 `ghcr.io` 或 `docker.io/zhemed/litepan`，`version` 改为 `v0.6.0-lite` 或 `three-drivers` | `git remote github → zhemed/LitePan` |
| **功能简述表格** | `跨盘秒传` 格（`feature-crosstransfer.png`）+ `目录整理` 格（`feature-organize.png`） | 表格 4→2 格：仅 `多网盘聚合（115/189/LocalFs）` + `自动联动（仅 delay）`，或 4→3 格加 `从服务器上传` | `internal/crosstransfer`、`mediaorganize` 已 `rm -rf` |
| **挂载与更多功能** | `支持 WebDAV 与 FUSE` 中的 `WebDAV`、`302 直链、缓存保持、命名对齐` | `支持 FUSE 本地挂载` + `从服务器上传（LocalUpload 唯一保留的增强工具）` | `internal/share/dav`、`cacheretention` 已删 |
| **快速开始** | `image: ponphil/litepan:beta`、`extra_hosts` 的 `tmdb` 段、`fuse_read_cache` 的 `WebDAV` 提及 | `image: zhemed/litepan:latest`（或 `litepan-go:three-drivers`），`ports 5211/42069`、`volumes data/mounts:shared`、`devices /dev/fuse` 保留，`extra_hosts` 删 | `docker images litepan-go:three-drivers 118M` |
| **支持网盘** | 泛指“多网盘” | 明确列表 `115网盘Open / 天翼云盘 (189Cloud) / 本机存储 (LocalFs)`，其余 8 驱动标注 `已移除（2026-08-30 three-drivers）` | `drivers/all.go` 仅 3 导入 |
| **许可/反馈** | `ponphil/litepan` 链接 | `zhemed/LitePan` 链接，`B 站` 等保留 |

## Data Flow

```
Before: README 全功能（11 驱动 + STRM + WebDAV + 跨盘 + 整理） → 用户按文档部署 ponphil/litepan:beta → 功能与代码不一致
After:  README 仅 3 驱动 + LocalUpload + FUSE → 用户 git clone zhemed/LitePan → docker build -t litepan-go:three-drivers → curl /api/health 200
```

## Compatibility

- **图片**：`docs/pictures/banner.png` 保留，`feature-crosstransfer.png` 等 2 图若不再引用可 `git rm` 但不影响 `vite`（`internal/api/web` 仅 `web` 构建产物）
- **链接**：`ACKNOWLEDGEMENTS.md / THIRD_PARTY_NOTICES.md` 中 8 已删驱动的致谢若提及，标注已移除即可，不删文件
- **i18n**：保持 `zh-CN`，不新增 `en` 段

## Tradeoffs

- **4 格删 2 格 vs 重绘表格**：删 2 格最简，保留 `多网盘聚合` 与 `自动联动` 的 2 格，符合 `118M` 的“少而精”定位；若需突出 `从服务器上传`，可改为 3 格（聚合/上传/联动）
- **版本号**：`v0.5.2-Beta` 为上游版本，精简后建议 `v0.6.0-lite` 或 `three-drivers` 以区分，避免与 `Ponphil` 混淆

## Rollout / Rollback

- 单提交 `docs(readme): refactor for three-drivers actual`，`git revert` 即恢复原 `README`
- 容器验证：`docker run -p 5211:5211 zhemed/litepan:latest` 后按新 `README` 的 `compose` 可启动

## File Map

1. `README.md`（144 行 → 约 120 行）
2. `docs/pictures/feature-crosstransfer.png` / `feature-organize.png`（若删则 `git rm`）
