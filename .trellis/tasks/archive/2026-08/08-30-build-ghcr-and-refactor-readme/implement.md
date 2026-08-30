# Implementation Plan: Build ghcr image and thoroughly refactor README

## Overview

按 `design.md` 先推镜像后重写 `README`。

## Phase 1: 构建并推送 ghcr.io 镜像

- [ ] 1.1 `docker build -t ghcr.io/zhemed/litepan:v0.5.2-Beta -t ghcr.io/zhemed/litepan:latest -t litepan-go:three-drivers .`（复用 `118M` 的 `Dockerfile`，`web` 已 `104 files`）
  - `verify: docker images ghcr.io/zhemed/litepan --format "{{.Tag}}" | grep v0.5.2-Beta`
- [ ] 1.2 `gh auth token | docker login ghcr.io -u zhemed --password-stdin`（`scope: write:packages`）
  - `verify: cat ~/.docker/config.json | grep ghcr.io`
- [ ] 1.3 `docker push ghcr.io/zhemed/litepan:v0.5.2-Beta && docker push ghcr.io/zhemed/litepan:latest`
  - `verify: docker pull ghcr.io/zhemed/litepan:v0.5.2-Beta` 在干净环境可拉取（`ghcr.io` 公开）

## Phase 2: 彻底重构 README.md（6 章单列流）

- [ ] 2.1 `write README.md` 约 150 行：`顶部（banner + zhemed/LitePan badge, version v0.5.2-Beta）/ 简介（1 句定位：基于 Ponphil 精简，仅 3 驱动 118M）/ 支持网盘（3 行表格 115/189/LocalFs，脚注 8 已移除）/ 功能清单（7 项清单，无 STRM/WebDAV/跨盘/整理）/ 快速开始（ghcr.io 完整 compose，3 映射示例可选）/ 技术栈与验证 / 许可与致谢`
  - 删除 `Ponphil` 原版的 `4 格` 表格中 `跨盘/整理` 的 `<img>` 引用（已在上次删，此次保持 0）
  - `image:` 2 处统一 `ghcr.io/zhemed/litepan:v0.5.2-Beta`（主）与 `ghcr.io/zhemed/litepan:latest`（浮动），`WARNING` 段保留 1 处 `ponphil/litepan:latest` 对比
  - `verify: grep -c "ghcr.io/zhemed/litepan:v0.5.2-Beta" README.md` >=2
  - `verify: grep -c "feature-crosstransfer" README.md` ==0
  - `verify: grep -c "ponphil/litepan:beta" README.md` ==0（除 1 处对比外）

## Phase 3: Sweep & Verification

- [ ] 3.1 `cat README.md | head -n 80` 目视 6 章单列流，`wc -l README.md` 约 140-180
- [ ] 3.2 `grep -rn "跨盘秒传|目录整理" --include="*.md" README.md | wc -l` ==0（或仅在脚注 1 行）
- [ ] 3.3 `GOWORK=off go vet ./...` PASS
- [ ] 3.4 `docker pull ghcr.io/zhemed/litepan:v0.5.2-Beta 2>&1 | tail -n 5` → `Status: Downloaded` 或 `Status: Image is up to date`

## Phase 4: Commit & Archive

- [ ] 4.1 `git add README.md && git commit -m "docs(readme): thoroughly refactor for ghcr v0.5.2-Beta"`
- [ ] 4.2 `task.py archive 08-30-build-ghcr-and-refactor-readme --skip-branch-validation && git add .trellis/tasks/archive/... && git commit`
- [ ] 4.3 `add_session.py --commit <hash>`
- [ ] 4.4 `git push github main`（含 `README` 与 `3` 个 `ghcr` 标签的 `digest` 已推）

## Rollback

- `git revert <docs commit>` + `gh api -X DELETE repos/zhemed/packages/container/litepan/versions`（如需删 `ghcr` 标签，需 `gh` 手动）

## Validation Commands

```bash
grep -c "ghcr.io/zhemed/litepan:v0.5.2-Beta" README.md  # >=2
grep -c "feature-crosstransfer" README.md  # 0
docker images ghcr.io/zhemed/litepan --format "{{.Tag}}"
docker pull ghcr.io/zhemed/litepan:v0.5.2-Beta
GOWORK=off go vet ./...
```
