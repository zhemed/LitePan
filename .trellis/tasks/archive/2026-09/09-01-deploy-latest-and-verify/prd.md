# 拉取最新镜像部署并验证

## Goal

拉取 `ghcr.io/zhemed/litepan:latest`（`0.0.10 sha256:18bf16f`）重新部署 `litepan` 容器并验证 `health` 与 `3驱动`。

## Background

- 本地 `latest` 刚 `pull` 到 `18bf16f`，但容器 `litepan` 仍 `0.0.9`。
- 需 `docker compose pull/up` 或 `docker run` 更新。

## Requirements

- **拉取**：`docker pull ghcr.io/zhemed/litepan:latest` 已 `up to date`（18bf16f）。
- **部署**：`docker rm -f litepan` + `docker run/compose up -d` 以 `latest/0.0.10` 启动，`ports 5211` `volumes data/mounts:shared`。
- **验证**：`curl /api/health` `200` `status ok`，`docker ps` 镜像 `0.0.10`，`logs` 无错误。

## Constraints

- 遵循 `AGENTS.md`：`task.py start` 后执行。

## Acceptance Criteria

- [ ] `docker ps --format` 显示 `0.0.10` / `latest 18bf16f`
- [ ] `curl /api/health` `200`
- [ ] `docker logs` 无 `offline/cross` 报错

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.
