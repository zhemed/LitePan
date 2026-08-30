# Implementation Plan: Refactor README for actual stripped build

## Overview

按 `design.md` 先做减法（删 2 格及挂载段），再做加法（明确 3 驱动与新镜像）。

## Phase 1: 移除已删功能描述

- [ ] 1.1 `README.md` 表格：删 `<h3>跨盘秒传</h3>` 所在 `<td>` 整块（含 `feature-crosstransfer.png`）与 `<h3>目录整理</h3>` 所在 `<td>` 整块（含 `feature-organize.png`），表格从 2 行 4 格改为 1 行 2 格（或 1 行 3 格若加上传）
  - `verify: grep -c "跨盘秒传" README.md` == 0（或仅在“已移除”备注）
  - `verify: grep -c "feature-crosstransfer" README.md` == 0
- [ ] 1.2 `README.md` `## ▎ 挂载与更多功能`：删 `WebDAV` 与 `302 直链、缓存保持、命名对齐` 句，改为 `支持 FUSE 本地挂载，另有从服务器上传（LocalUpload 唯一保留的增强工具）等能力。`
  - `verify: grep -c "WebDAV" README.md` == 0（除徽章外）
- [ ] 1.3 `README.md` `快速开始` 的 `extra_hosts` 段（含 `tmdb`）整块删（`classifyorganize` 已删，`tmdb` 无意义）
  - `verify: grep -c "themovedb" README.md` == 0

## Phase 2: 重构为实际

- [ ] 2.1 顶部 `badge` 区域：`version-shield` 改为 `v0.6.0-lite` 或 `three-drivers`，`docker-pulls-shield` 的 `docker-url` 由 `https://hub.docker.com/r/ponphil/litepan` 改为 `https://github.com/zhemed/LitePan` 或 `https://hub.docker.com/r/zhemed/litepan`（若已推），`license-shield` 保留
  - `verify: grep -c "ponphil/litepan" README.md` == 1（仅许可段的旧版提示中保留一处，其余改 zhemed）或 0
- [ ] 2.2 `功能简述` 表格：保留 `多网盘聚合` 格但 `<p>` 改为 `仅支持 115网盘Open / 天翼云盘 (189Cloud) / 本机存储 (LocalFs) 3 驱动`，`自动联动` 格 `<p>` 改为 `仅支持 delay 等待，其余 Emby/整理等已移除`
  - `verify: grep -c "115.*天翼.*本机" README.md` >=1
- [ ] 2.3 `挂载与更多功能` 下新增 `支持网盘` 列表：`115网盘Open / 天翼云盘 / 本机存储` 3 项，括号标注 `已移除 8 驱动：123/139/百度/广雅/OneDrive/OpenList/Quark/WebDAV（2026-08-30 three-drivers）`
- [ ] 2.4 `快速开始` 的 `compose` 示例：`image: ponphil/litepan:beta` → `image: zhemed/litepan:latest`（或 `litepan-go:three-drivers`），`fuse_read_cache` 的 `WebDAV` 提及删，`ports/volumes/devices` 保留
- [ ] 2.5 `反馈` 段：`ponphil/litepan` 的 `latest 是 Python 旧版` 提示保留 1 处即可，其余链接改为 `zhemed/LitePan`

## Phase 3: Sweep & Verification

- [ ] 3.1 `grep -rn "跨盘秒传|目录整理|feature-crosstransfer|feature-organize" --include="*.md" | wc -l` == 0（或仅在“已移除”备注）
- [ ] 3.2 `cat README.md | head -n 100` 人工目视表格为 2 格且图片可加载
- [ ] 3.3 `GOWORK=off go vet ./...` PASS（文档不影响构建），`git diff --stat` 仅 `README.md` (+/- 约 30 行) 与 `docs/pictures` 若删图
- [ ] 3.4 `docker build -t litepan-go:three-drivers .` 已在上任务完成，此处仅 `git status` 确认 `Clean` 外仅 `README.md`

## Phase 4: Commit & Archive

- [ ] 4.1 `git add README.md docs/pictures/feature-crosstransfer.png docs/pictures/feature-organize.png（若删） && git commit -m "docs(readme): refactor for three-drivers actual"`
- [ ] 4.2 `task.py archive 08-30-refactor-readme-actual --skip-branch-validation && git add .trellis/tasks/archive/... && git commit`
- [ ] 4.3 `add_session.py --commit <hash>`

## Rollback

- `git revert <docs commit>`

## Validation Commands

```bash
grep -c "跨盘秒传" README.md  # 0
grep -c "115.*天翼" README.md  # >=1
cat README.md | head -n 60
GOWORK=off go vet ./...
```
