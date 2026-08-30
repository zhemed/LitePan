# Build ghcr image and thoroughly refactor README

## Goal

如 `LitePan-own` 的 `ghcr.io/zhemed/litepan-own:0.0.1`，为本仓库的精简版构建可被他人 `docker pull` 的镜像 `ghcr.io/zhemed/litepan:v0.5.2-Beta`（与 `zhemed/litepan:latest` 同步），并**彻底重构 `README.md`**（非增量补丁）——上次 `ac578c2` 的 7 章重构仍被用户认为“不理想”，需从信息架构、视觉、文案三层重做，使 `README.md` 与 `three-drivers 118M` 实际一致且可直接 `docker compose` 启动。

## Requirements

- **镜像**：
  - 基于当前 `main`（`b94f91a` 含 `local_upload` 适配）的 `litepan-go:three-drivers 118M`（或 `localupload 118M`）重新 `docker build -t ghcr.io/zhemed/litepan:v0.5.2-Beta`，`docker tag` 同时打 `ghcr.io/zhemed/litepan:latest` 与 `zhemed/litepan:latest`（`hub.docker.com` 可选）
  - `docker login ghcr.io` 使用 `gh` 的 `oauth_token`（`hosts.yml` 的 `ghp_...`，`scope: write:packages`），`docker push ghcr.io/zhemed/litepan:v0.5.2-Beta && docker push ghcr.io/zhemed/litepan:latest`
  - `README.md` 的 `image:` 统一改为 `ghcr.io/zhemed/litepan:v0.5.2-Beta`（主）与 `ghcr.io/zhemed/litepan:latest`（浮动），旧 `image: zhemed/litepan:latest` 与 `ponphil/litepan:beta` 仅在 `WARNING` 段保留 1 处对比
- **README 彻底重构（非 775bc62/ac578c2 的补丁式）**：
  - 抛弃 `Ponphil` 原版的 `4 格` 表格结构，按精简版的**真实用户旅程**重排 6 章：`1 顶部（banner+badge 指向 zhemed/LitePan）/ 2 一句话定位（精简版 3 驱动 118M）/ 3 支持网盘（3 行表格）/ 4 核心功能（7 项清单，无 STRM/WebDAV/跨盘/整理）/ 5 快速开始（ghcr.io 完整 compose + 3 映射示例）/ 6 技术栈与验证 / 7 许可与致谢`，每章 1 屏，不堆表
  - 文案：`多网盘聚合（仅 3 驱动）` 等标题去括号，去 `已移除（2026-08-30 精简）` 的大段对比表，改为 `支持网盘` 小节的 1 行脚注“其余 8 驱动已移除”
  - 视觉：`badge` 的 `docker-pulls` 改为 `ghcr`，`version` 固定 `v0.5.2-Beta`，`banner` 保留，`feature-*.png` 仅保留 `feature-browser.png` 与 `feature-automation.png` 的 1 行 2 图（或单列），不堆 4 图
  - 删除 `extra_hosts` 的 `tmdb`、`fuse_read_cache` 的 `WebDAV` 提及等已删功能的任何残留

## Constraints

- 仅改 `README.md` 与 `docker` 镜像标签，不改 `internal/*` / `web/*` 代码（`118M` 已定）
- `ghcr.io` 推送需 `docker login ghcr.io -u zhemed --password-stdin <<< $TOKEN`，`TOKEN` 取 `gh auth token`，`scope` 需 `write:packages`
- 新 `README.md` 约 `140-180` 行，`grep -c "feature-crosstransfer" ==0`，`grep -c "ponphil/litepan:beta"` ==0（除 `WARNING` 外），`grep -c "ghcr.io/zhemed/litepan:v0.5.2-Beta" >=2`

## Acceptance Criteria

- [ ] `docker images ghcr.io/zhemed/litepan --format "{{.Tag}}"` 含 `v0.5.2-Beta` 与 `latest`，`docker pull ghcr.io/zhemed/litepan:v0.5.2-Beta` 在新环境可拉取（`ghcr.io` 公开）
- [ ] `README.md` 无 `ponphil/litepan:beta`（除 1 处对比），`image:` 2 处均为 `ghcr.io/zhemed/litepan:v0.5.2-Beta` / `ghcr.io/zhemed/litepan:latest`
- [ ] `README.md` 无 `feature-crosstransfer.png` / `feature-organize.png` 的 `<img>`，仅保留 `feature-browser.png` 等 1-2 图
- [ ] `README.md` 有独立 `支持网盘` 表格 3 行，`功能清单` 7 项与 `118M` 一致，无 `STRM/WebDAV/跨盘/整理` 等已删描述
- [ ] `cat README.md | head -n 80` 目视 6 章结构清晰，`GOWORK=off go vet` 仍 PASS

## Notes

- 上次 `ac578c2` 的 7 章重构被认为“不理想”，本次按用户旅程（`支持网盘 → 功能 → 快速开始`）重排信息架构，非增量补丁
- 镜像 `v0.5.2-Beta` 复用 `LitePan-own` 的 `0.0.1` 定版思路，`Beta` 浮动指向最新
