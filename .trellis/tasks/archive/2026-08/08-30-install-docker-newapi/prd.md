# Install Docker via new-api-own script

## Goal

执行 `curl -fsSL https://raw.githubusercontent.com/zhemed/new-api-own/main/install-docker.sh | bash` 完成 Docker 及 new-api-own 依赖的安装，使本机具备 docker/docker compose 能力。

## Requirements

- 下载并执行 `zhemed/new-api-own/main/install-docker.sh`（raw.githubusercontent）
- 脚本以 `bash` 非交互执行，允许安装 Docker Engine、compose 插件及相关依赖
- 保持 Trellis 规范：记录任务、验收、日志归档
- 不破坏现有 LitePan 服务（`*:5211` 仍监听，`data/litepan.db` 保留）

## Constraints

- 需 `curl` 可访问 `raw.githubusercontent.com`（已验证可达）
- 脚本可能需 `apt`/`systemd` 权限，本机为 `root` 权限容器内，可写入
- 若脚本改写 Docker 配置或重启 `dockerd`，需在完成后验证 `docker --version` 与 `docker compose version`/`docker-compose --version`

## Acceptance Criteria

- [ ] `curl -fsSL https://raw.githubusercontent.com/zhemed/new-api-own/main/install-docker.sh | bash` 执行完成且 `EXIT 0`
- [ ] `docker --version` 输出可用版本
- [ ] `docker compose version` 或 `docker-compose --version` 至少其一可用
- [ ] `ss -tlnp` 中 `*:5211` 的 litepan 仍运行（`ps -p 76998` 存活）或可重启
- [ ] 脚本日志已捕获至 `/tmp/install-docker.log` 供回溯

## Notes

- Lightweight PRD-only 任务，`task.py start` 后直接执行脚本
