# Final thorough README refactor

## Goal

按用户“全部删除，内容由你重写”将 `README.md` 从零彻底重写为 `97 行` 的 `ghcr.io/zhemed/litepan:v0.5.2-Beta` 单列流，与 `118M 3驱动` 实际一致。

## Requirements

- 删除旧 `README.md` 全部内容，从 `a name="readme-top"` 起重写
- 新结构：`顶部（GHCR badge）/ 一句话定位 / 支持网盘（3 行）/ 功能清单（7 项）/ 快速开始（ghcr compose）/ 技术栈 / 许可` 6 屏
- `image:` 统一 `ghcr.io/zhemed/litepan:v0.5.2-Beta`，`ponphil/litepan:beta` 仅 `WARNING` 1 处
- 全文 90-120 行，`grep -c "feature-crosstransfer" ==0`，`grep -c "ghcr.*v0.5.2-Beta" >=2`

## Acceptance Criteria

- [ ] `wc -l README.md` 90-120
- [ ] `grep -c "ghcr.io/zhemed/litepan:v0.5.2-Beta" >=2`
- [ ] `grep -c "feature-crosstransfer" ==0`
- [ ] `GOWORK=off go vet 0`
