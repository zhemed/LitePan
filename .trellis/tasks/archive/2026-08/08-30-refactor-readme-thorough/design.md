# Design: Thoroughly refactor README from scratch

## Overview

旧 `README.md` 为 `Ponphil` 全功能版（11 驱动 + STRM + WebDAV + 跨盘 + 整理）的静态复制，与 `zhemed/LitePan` 的 `three-drivers` 现状严重割裂。需按 `docs-as-code` 原则，以 `drivers/all.go 3 导入` 与 `CloudToolsPanel 8→1` 为唯一事实源，重绘 7 章结构。

## Boundaries

| 章 | 重写要点 | 事实源 |
|---|---|---|
| **顶部** | `banner` 保留，`badge` 的 `docker-pulls/version/docker-url` 指向 `zhemed/LitePan`，`version` 为 `v0.6.0-lite` | `git remote github → zhemed/LitePan` |
| **简介** | 1 段精简版定位 + 2 列对比表 `已移除（13012 行）` vs `仍保留` | `git log --stat` 的 4 次 `refactor`（strm/share/cache/aux/crosstransfer/drivers） |
| **功能清单** | 8 项 → 7 项：`多网盘聚合（3 驱动）/文件管理/FUSE/从服务器上传/离线下载/自动联动（仅 delay）/备份恢复/系统设置` | `internal/crosstransfer` 已 404，`CloudToolsPanel` 仅 `LocalUpload` |
| **支持网盘** | 独立表格 3 行：`115_Open/189Cloud/LocalFs`，括号注 8 已移除 | `GET /admin/drivers → 3` |
| **快速开始** | `image: zhemed/litepan:latest`，`ports/volumes/devices/pid` 保留，`extra_hosts tmdb` 删，`admin/123456` 明文 | `docker images 118M`，`adminauth` 实际 |
| **技术栈** | `Go 1.26.6 / chi v5 / modernc.org/sqlite / Vue 3.5.41 / Vite 8.2.2`，`go vet/build` 指令 | `go.mod`，`web/package.json` |
| **许可** | 保留 `PolyForm NC`，链接改为 `zhemed/LitePan` | `LICENSE` |

## Data Flow

```
旧 README（Ponphil 全功能） → 读 drivers/all.go 3 导入 + router 仅 local-upload + CloudToolsPanel 仅 LocalUpload → 新 README 7 章 → 用户 git clone zhemed/LitePan → docker compose up → /api/health 200
```

## Compatibility

- **图片**：`banner.png` 保留，`feature-crosstransfer.png` 等 2 图不再引用但文件可保留，不影响 `vite`（仅 `README` 引用）
- **链接**：`ACKNOWLEDGEMENTS.md / THIRD_PARTY_NOTICES.md` 中 8 已删驱动的致谢保留原文，仅 `README` 中标注已移除
- **i18n**：保持 `zh-CN`，不新增 `en`

## Tradeoffs

- **彻底重写 vs 增量补丁**：上次 `775bc62` 为增量补丁（`+12 -27`），用户明确要彻底重构，故本次 `~180 行` 全量重写，结构更清晰，但 `git blame` 会断
- **版本号**：`v0.5.2-Beta` 为上游，精简后 `v0.6.0-lite` 以区分，避免与 `Ponphil` 混淆

## Rollout / Rollback

- 单提交 `docs(readme): thoroughly refactor for three-drivers`，`git revert` 即恢复旧版
- 验证：`cat README.md | head -n 80` 目视 7 章结构，`grep -c "跨盘秒传" ==0`（除对比表外）

## File Map

1. `README.md`（144 行 → 约 180 行，全量重写）
