# Implementation Plan: Thoroughly refactor README from scratch

## Overview

按 `design.md` 7 章从零重写 `README.md`。

## Phase 1: 从零重写 README.md

- [ ] 1.1 `write README.md` 约 180 行，含 7 章：`顶部（banner+badge）/简介（精简定位+已移除 vs 仍保留对比表）/功能清单（7 项）/支持网盘（3 行表格）/快速开始（zhemed/litepan compose）/技术栈与构建/许可与致谢`
  - 顶部 `version-shield` → `v0.6.0-lite`，`docker-url` → `https://github.com/zhemed/LitePan`
  - 简介对比表：`已移除：STRM 44文件 / WebDAV 16文件 / cacheretention 15 / mediaorganize 30+ / 7 增强工具 / crosstransfer 4 / 8 驱动 13012 行` vs `仍保留：115/189/LocalFs + LocalUpload + FUSE + 离线下载 + 备份恢复`
  - 功能清单 7 项：`多网盘聚合（3 驱动）/文件管理/FUSE/从服务器上传/离线下载/自动联动（仅 delay）/备份恢复/系统设置`，无 `跨盘/整理/STRM/WebDAV`
  - 支持网盘表格 3 行：`115网盘Open / 天翼云盘 / 本机存储`，括号注 8 已移除
  - 快速开始 `image: zhemed/litepan:latest`，`admin/123456`，`FUSE` 权限说明，`WARNING` 保留 1 处 `ponphil/litepan:latest`
  - 技术栈 `Go 1.26.6 / chi v5 / modernc.org/sqlite / Vue 3.5.41 / Vite 8.2.2`，`GOWORK=off go vet/build` 指令
  - `verify: cat README.md | head -n 80` 含 `zhemed/LitePan` 与 3 驱动

## Phase 2: Sweep & Verification

- [ ] 2.1 `grep -c "feature-crosstransfer" README.md` == 0
- [ ] 2.2 `grep -c "跨盘秒传" README.md` == 0（或仅在“已移除”对比表中 1 处）
- [ ] 2.3 `grep -c "115.*天翼.*本机" README.md` >=1
- [ ] 2.4 `cat README.md | wc -l` 约 150-220
- [ ] 2.5 `GOWORK=off go vet ./...` PASS

## Phase 3: Commit & Archive

- [ ] 3.1 `git add README.md && git commit -m "docs(readme): thoroughly refactor for three-drivers"`
- [ ] 3.2 `task.py archive 08-30-refactor-readme-thorough --skip-branch-validation && git add .trellis/tasks/archive/... && git commit`
- [ ] 3.3 `add_session.py --commit <hash>`

## Rollback

- `git revert <docs commit>`

## Validation Commands

```bash
cat README.md | head -n 60
grep -c "跨盘秒传" README.md
GOWORK=off go vet ./...
```
