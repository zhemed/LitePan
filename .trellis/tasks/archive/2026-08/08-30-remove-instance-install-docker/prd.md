# Remove instance install-docker hosting

## Goal

彻底还原上次错误：将 `LitePan` 运行实例的 `http://IP:5211/install-docker.sh` 直链（`internal/api/router.go` 的 `GET /install-docker.sh` + `/root/LitePan/data/install-docker.sh`）移除，仅保留 `https://raw.githubusercontent.com/zhemed/LitePan/main/install-docker.sh` 的 GitHub 仓库根托管（`./install-docker.sh` 89 行）。

## Requirements

- **删除实例直链**：
  - `internal/api/router.go`：删 `r.Get("/install-docker.sh", h.serveInstallDocker)` 与 `r.Get("/install-docker-fixed.sh", ...)` 2 行，删 `func (h *Handler) serveInstallDocker` 整段（含 `os.ReadFile` 逻辑），删 `import "os"` 若仅为该 handler 引入
  - `rm /root/LitePan/data/install-docker.sh`（`bind mount` 的宿主机文件，容器内 `/app/data/install-docker.sh` 同步消失）
- **保留**：
  - `./install-docker.sh`（仓库根，89 行，`chmod +x`，`a0fe719` 已提交，`raw.githubusercontent` 主链）
  - `README.md` 的 `curl -fsSL https://raw.githubusercontent.com/zhemed/LitePan/main/install-docker.sh | bash` 小节
  - `internal/api/router.go` 的其余 `Deps` 注入与 `/api` 路由

## Constraints

- 仅删实例直链，不改 `./install-docker.sh` 仓库根与 `README.md`
- `grep -c "serveInstallDocker" internal/api/router.go` == 0
- `ls /root/LitePan/data/install-docker.sh` → No such file
- `curl -s http://127.0.0.1:5211/install-docker.sh` → 404（`spaHandler` 回退 `index.html` 或 `404`）
- `curl -fsSL https://raw.githubusercontent.com/zhemed/LitePan/main/install-docker.sh | head` 仍 `#!/bin/bash` 89 行

## Acceptance Criteria

- [ ] `grep -n "serveInstallDocker" internal/api/router.go` == 0
- [ ] `grep -n "install-docker" internal/api/router.go` == 0
- [ ] `ls /root/LitePan/data/install-docker.sh` → No such file
- [ ] `curl -s -o /dev/null -w "%{http_code}" http://127.0.0.1:5211/install-docker.sh` == 404（或 200 但非脚本）
- [ ] `curl -fsSL https://raw.githubusercontent.com/zhemed/LitePan/main/install-docker.sh | head -n 5` → `#!/bin/bash` 修复版
- [ ] `GOWORK=off go vet ./...` PASS

## Notes

- 上次 `aceca55` 的 `feat: serve install-docker.sh via LitePan` 为死循环方案，本次彻底还原
