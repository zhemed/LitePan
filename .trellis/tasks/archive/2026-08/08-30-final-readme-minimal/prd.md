# Final README minimal 66 lines

## Goal

按用户“彻底重写”将 `README.md` 从 0 重写为 `66 行` 极简版（`LitePan-own` 风格：`66` 行，`GHCR` 单镜像，`3 驱动`），与 `118M` 实际一致，彻底抛弃 `Ponphil` 旧骨架。

## Requirements

- 删除旧 `README.md` 全部内容，从 `a name="readme-top"` 起重写，约 60-70 行
- 新结构 4 屏：`顶部（banner + GHCR badge）/ 支持网盘（3 行）/ 快速开始（ghcr compose）/ 能做什么（4 行）/ 许可`，无 `已移除` 大表
- `image:` 仅 `ghcr.io/zhemed/litepan:v0.5.2-Beta`，`ponphil` 仅 `CAUTION` 1 处

## Acceptance Criteria

- [ ] `wc -l README.md` 60-70
- [ ] `grep -c "ghcr.io/zhemed/litepan:v0.5.2-Beta" >=2`
- [ ] `GOWORK=off go vet 0`
