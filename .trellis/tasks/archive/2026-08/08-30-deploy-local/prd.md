# Deploy LitePan to local machine

## Goal

本地部署 LitePan 到本机（`/root/LitePan`），验证 `http://127.0.0.1:5211` 可访问，管理员可登录，工程化流程严格遵循 Trellis spec。

## Requirements

- **部署方式**：优先本机 Go 原生（`go build -tags nofuse` 避免 FUSE 内核依赖），若本机 FUSE 可用则 ` -tags fuse`；不依赖 `docker compose`（本机 `docker compose` 插件未安装），但保留 `Dockerfile` 构建产物可选。
- **前端产物**：`web/` 必须 `npm ci && npm run build` 生成 `internal/api/web`（`vite outDir ../internal/api/web`），供 Go `//go:embed web` 嵌入；否则 `/` 返回 404。
- **运行时目录**：`LITEPAN_DATA_DIR=/root/LitePan/data`、`LITEPAN_STRM_DIR=/root/LitePan/strm`、`LITEPAN_LISTEN=:5211`，确保 `data/ strm/ mounts/` 存在且可写；`data/litepan.db` 自动创建。
- **端口**：`5211` 未占用（`ss -tlnp` 已确认仅 22/3080/3081），`42069` Magnet 端口可选不暴露。
- **健康检查**：启动后 `GET /api/health` 或 `GET /auth/status` 返回 200，`GET /` 返回 SPA `index.html`。
- **管理员**：首次启动默认 `admin/admin`，`POST /auth/login` 可登录获取 session。

## Constraints

- 遵循 `spec/backend/backend/*` 与 `spec/web/frontend/*`：不改 `pkg`/`domain` 导入边界；前端构建必须 `vue-tsc -b` 通过。
- 单二进制，不写 `docker-compose.override`；不占用 `3080/3081`（DSH Web GUI）。
- 日志落 `data/log/`，级别 `info`，不泄露 token。

## Acceptance Criteria

- [ ] `cd web && npm ci && npm run build` 产出 `../internal/api/web/index.html` 且 `vue-tsc` 无报错
- [ ] `GOWORK=off go build -trimpath -ldflags="-s -w" -o /tmp/litepan ./cmd/litepan` 成功（或 `make build-nofuse`）
- [ ] 启动 `LITEPAN_DATA_DIR=/root/LitePan/data LITEPAN_LISTEN=:5211 /tmp/litepan &` 后 `curl -s http://127.0.0.1:5211/auth/status` 返回 `{"is_admin":...}` 且 HTTP 200
- [ ] `curl -s http://127.0.0.1:5211/ | grep -q "LitePan"` 或 `index.html` 存在
- [ ] `curl -s -X POST http://127.0.0.1:5211/auth/login -d "username=admin&password=admin" -c /tmp/cookie | grep -q admin`（或 JSON `is_admin:true`）
- [ ] 进程常驻后台，日志无 panic，`ss -tlnp` 可见 `0.0.0.0:5211` 或 `127.0.0.1:5211`
- [ ] 停止可控：`kill <pid>` 后端口释放，`data/litepan.db` 保留

## Notes

- Lightweight task: PRD-only，无需 `design.md/implement.md`，验收即 `in_progress` → `check` → `archive`。
- DSH 宿主已提供 `DSH_SESSION_ID`，`get_context.py` 自动以 `dsh_session-...` 绑定任务，无需手动 `TRELLIS_CONTEXT_ID`。
- 后续若需容器化，再建子任务 `deploy-docker` 走 `docker build -t litepan-go:dev .` + `docker run -p 5211:5211 ...`。
