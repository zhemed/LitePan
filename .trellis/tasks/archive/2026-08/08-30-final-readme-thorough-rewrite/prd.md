# Final thorough README rewrite

## Goal

按用户“全部删除，内容由你重写”将 `README.md` 从 0 重写为极简理想版（`LitePan-own` 风格：首屏 1 句定位 + 1 个 GHCR 拉取即决策），与 `118M 3驱动` 实际一致，彻底抛弃 `Ponphil` 原版的 4 格堆砌与工程师脚注。

## Requirements

- **删除旧 README 全部内容**，从 `a name="readme-top"` 起重写，约 60-80 行
- **新结构 4 屏**：
  1. 顶部：`banner` + `GHCR v0.5.2-Beta + License` 2 徽章，去 `docker-pulls/Bilibili` 堆砌
  2. 一句话：`LitePan 精简版 · 115/天翼/本机存储 · 118M · FUSE`
  3. 快速开始：`docker pull ghcr.io/zhemed/litepan:v0.5.2-Beta` + 单 `compose`（`ghcr.io` 单镜像，`admin/admin`）
  4. 支持网盘：1 行 `115/天翼/本机` + 脚注 `已移除 8 驱动`
- **文案**：人话，去 `GET /admin/drivers → 3`、`+13012`、`mounts:shared` 等技术词
- `image:` 仅 `ghcr.io/zhemed/litepan:v0.5.2-Beta`，`ponphil` 仅 `WARNING` 1 处

## Acceptance Criteria

- [ ] `wc -l README.md` 60-80
- [ ] `grep -c "ghcr.io/zhemed/litepan:v0.5.2-Beta" >=2`
- [ ] `grep -c "115.*天翼.*本机" >=1`
- [ ] `GOWORK=off go vet 0`
