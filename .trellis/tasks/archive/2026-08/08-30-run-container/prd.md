# Run LitePan in Docker container

## Goal

将 LitePan 从本机原生 `/tmp/litepan :5211` 切换到 Docker 容器运行，保持 `http://127.0.0.1:5211` 可用、数据持久化（`./data/litepan.db` → `/app/data`），严格遵循 Trellis 流程与现有 `Dockerfile`/`docker-compose.yml`。

## Requirements

- 停止原生进程 `pid 76998`（释放 `*:5211`），数据保留在 `/root/LitePan/data`
- 构建本地镜像：`docker build -t litepan-go:dev .`（复用 `Dockerfile` 多阶段：`node:20` 构建 `web → internal/api/web` + `golang:1.26.6` 构建二进制）
- 运行容器：映射 `5211:5211`、`42069:42069/tcp+udp`，挂载 `./data:/app/data`、`./strm:/app/strm`、`./mounts:/app/mounts:shared`，传参 `TZ=Asia/Shanghai`，`--device /dev/fuse --privileged --pid host`（按 `docker-compose.yml`），`--name litepan`
- 若 `docker compose` 可用，优先 `docker compose up -d --build`（需为 `docker-compose.yml` 补 `build: .`）；否则 `docker run` 直启
- 兼容已安装 `Docker 29.7.2 + Compose v5.4.0`，`buildx v0.36.1`

## Constraints

- 不丢失 `data/litepan.db`、`data/log/`、`strm/` 内容（容器 volume 为 bind mount，非匿名卷）
- 不占用 `3080/3081`（DSH），仅 `5211/42069`
- 构建需 `GOPROXY=https://goproxy.cn,direct`、`npm registry https://registry.npmmirror.com`（`Dockerfile` 已设）
- 若容器已存在 `litepan`，先 `docker rm -f litepan`

## Acceptance Criteria

- [ ] 原生 `kill 76998` 后 `ss -tlnp | grep 5211` 无监听
- [ ] `docker build -t litepan-go:dev .` EXIT 0，`docker images litepan-go:dev` 可见
- [ ] `docker ps --filter name=litepan --format "{{.Names}} {{.Ports}} {{.Status}}"` 显示 `litepan` Up 且 `0.0.0.0:5211->5211`
- [ ] `curl -s http://127.0.0.1:5211/api/health | grep -q '"status":"ok"'`
- [ ] `curl -s http://127.0.0.1:5211/api/auth/status | grep -q is_admin`
- [ ] `curl -s http://127.0.0.1:5211/ | grep -q LitePan`（SPA）
- [ ] `POST /api/auth/login admin/admin` 在容器内仍返回 `is_admin:true`（cookie 持久）
- [ ] `docker logs litepan 2>&1 | grep -q "HTTP 服务已监听"`
- [ ] 数据持久：宿主机 `ls -lh data/litepan.db` 与容器内 `/app/data/litepan.db` 为同一 bind mount

## Notes

- Lightweight PRD-only，`task.py start` 后即 Implement
