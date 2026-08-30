# Design: Build ghcr image and thoroughly refactor README

## Overview

`LitePan-own` 的 `ghcr.io/zhemed/litepan-own:0.0.1` 为定版，`zhemed/LitePan` 需同款 `ghcr.io/zhemed/litepan:v0.5.2-Beta`。`README` 上次 7 章重构仍被认为信息堆砌，需按用户旅程重排为 6 章单列流。

## Boundaries

| 产出 | 旧 | 新 |
|---|---|---|
| **镜像标签** | `zhemed/litepan:latest`（`hub.docker.com`） | `ghcr.io/zhemed/litepan:v0.5.2-Beta`（主） + `ghcr.io/zhemed/litepan:latest`（浮动），`hub` 可选同步 |
| **README 顶部** | `ponphil/litepan:beta` 的 `docker-pulls`/`version` | `zhemed/LitePan` 的 `ghcr` badge，`version → v0.5.2-Beta` |
| **README 结构** | 7 章堆表（简介对比表 + 功能 7 项 + 支持网盘 + 挂载 + 快速开始 + 技术栈 + 许可） | 6 章单列流：`顶部/简介（1 句定位）/支持网盘（3 行表）/功能清单（7 项清单）/快速开始（ghcr compose）/技术栈/许可`，去对比大表，`已移除` 仅在支持网盘脚注 1 行 |
| **README 图片** | `banner.png` + `feature-browser.png` + `feature-automation.png` + 残留 `feature-crosstransfer/organize` 引用已删但结构仍堆 | 仅 `banner.png` + `feature-browser.png`（或 + `feature-automation.png`）1 行 2 图，不堆 4 图 |

## Data Flow

```
本地 main (b94f91a, 118M) → docker build -t ghcr.io/zhemed/litepan:v0.5.2-Beta → docker login ghcr.io (gh auth token) → docker push v0.5.2-Beta + latest → ghcr.io 公开 → README 的 image: ghcr.io/...:v0.5.2-Beta → 用户 docker compose up -d → /api/health 200
```

## Compatibility

- `gh` 的 `oauth_token`（`hosts.yml ghp_...`）含 `write:packages`，`docker login ghcr.io -u zhemed --password-stdin` 可复用
- `README` 的 `快速开始` 仍保留 ` ponphil/litepan:latest 是 Python 旧版` 的 `WARNING` 1 处，用于防错
- `internal/api/web` 的 `104 files` 不因 `README` 改变而变

## Tradeoffs

- **ghcr vs hub**：`LitePan-own` 用 `ghcr.io`，本仓库跟 `ghcr` 保持一致，`hub` 可后续 `docker tag + push` 同步，非必须
- **彻底重写 vs 补丁**：上次 `ac578c2` 为补丁式（`+74 -40`），本次 `~160 行` 全量重写，`git blame` 会断但信息架构更清晰，符合用户“彻底重构”诉求

## Rollout / Rollback

- 单提交 `docs(readme): thoroughly refactor for ghcr v0.5.2-Beta` + 单 `docker push`，`git revert` 即恢复旧 `README`
- 镜像 `v0.5.2-Beta` 为定版，`latest` 为浮动，`docker pull` 验证

## File Map

1. `README.md`（163 行 → 约 150 行，全量重写）
2. `docker` 镜像：`ghcr.io/zhemed/litepan:v0.5.2-Beta` + `ghcr.io/zhemed/litepan:latest`
