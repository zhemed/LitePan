# cleanup-p0-docker-cache

## Goal

执行残留盘点报告的 **P0 磁盘清理**：回收 Docker 构建缓存与悬空镜像（预计 ≈16GB），不触碰运行容器、带 tag 镜像与卷。来源：`09-10-investigate-residual-maintenance` research.md §P0。

## Requirements

1. 清理前记录基线：`docker system df`、宿主机 `df -h /`、运行容器状态（镜像 tag、启动时间）
2. 执行：
   - `docker builder prune -f`（构建缓存）
   - `docker image prune -f`（仅悬空镜像，不加 `-a`，保留所有带 tag 镜像）
3. **硬性禁止**：不删运行容器及其镜像（`ghcr.io/zhemed/litepan:latest`）、不删任何带 tag 的版本镜像、不删卷、不重启/重建容器
4. 清理后复核：`docker system df` 对比、`df -h /` 对比、服务三连（health/登录/文件列表）、容器仍在运行且启动时间未变

## Constraints

- 只做 P0（构建缓存 + 悬空镜像）；P1（云端 190 文件/本地 2G/残留容器）不在本次范围
- 全过程可审计：清理前后数据落盘到 research.md

## Acceptance Criteria

- [x] 基线记录完成（构建缓存 17.05GB/悬空 3/根分区 28G 已用/容器 StartedAt）
- [x] 两条 prune 成功；带 tag 镜像 78→78 无损失，悬空 3→0
- [x] 实测回收 ≈17GB（根分区 28G→11G；构建缓存 17.05GB→1.01GB）
- [x] 三连通过、StartedAt 未变、卷未触碰
- [x] 质量门 + 门禁 + 归档 + journal + push
