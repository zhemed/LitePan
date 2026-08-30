# Correct install-docker hosting to repo and update README

## Goal

纠正上次误将 `install-docker-fixed.sh` 仅托管到 `LitePan` 运行实例的 `http://IP:5211/install-docker.sh`（需先有 LitePan 的死循环），改为**提交到 `zhemed/LitePan` GitHub 仓库根**，使他人可 `curl -fsSL https://raw.githubusercontent.com/zhemed/LitePan/main/install-docker.sh | bash` 替代旧的 `zhemed/new-api-own` 源，并同步更新 `README.md` 的 `快速开始` 中 URL。

## Requirements

- **仓库提交**：
  - `cp /tmp/install-docker-fixed.sh ./install-docker.sh`（89 行修复版，`signed-by` + `rootless` 容错）置于 `zhemed/LitePan` 根（与 `README.md` 同级），`chmod +x`
  - `git add install-docker.sh && git commit -m "feat: add install-docker.sh for zhemed/LitePan (from new-api-own fixed)"`
- **移除实例托管**（可选保留 `router.go` 的 `GET /install-docker.sh` 作为兼容，但 `README` 主推 `raw.githubusercontent`）：
  - 保留 `internal/api/router.go` 的 `serveInstallDocker` 及其 `GET /install-docker.sh` 路由（已验证 `curl http://127.0.0.1:5211/install-docker.sh` 89 行），或按需移除（本次保留，仅 `README` 改主推）
- **README 更新**：
  - `README.md` 的 `快速开始` 中 `docker pull` 前的 `curl` 示例（若有）或 `LitePan` 的 `install-docker.sh` 引用，由 `https://raw.githubusercontent.com/zhemed/new-api-own/main/install-docker.sh` 改为 `https://raw.githubusercontent.com/zhemed/LitePan/main/install-docker.sh`
  - 若 `README` 无 `install-docker.sh` 引用，则新增 `一键安装 Docker` 小节：`curl -fsSL https://raw.githubusercontent.com/zhemed/LitePan/main/install-docker.sh | bash`
  - 保留 `ghcr.io/zhemed/litepan:v0.5.2-Beta` 的 `compose` 示例不变

## Constraints

- 仅改 `install-docker.sh`（新增）与 `README.md` 的 URL，不改 `internal/*` 代码（`router.go` 的 `serveInstallDocker` 已加，保留）
- `install-docker.sh` 需 `bash -n` 语法通过，且 `grep -c "signed-by" >=1`
- `README.md` 的 `grep -c "raw.githubusercontent.com/zhemed/LitePan/main/install-docker.sh" >=1` 且 `grep -c "new-api-own" ==0`（除 `WARNING` 外）

## Acceptance Criteria

- [ ] `ls install-docker.sh` 存在且 `wc -l ==89`，`bash -n install-docker.sh && echo ok`
- [ ] `git ls-remote https://github.com/zhemed/LitePan.git HEAD` 的 `sha` 与本地 `HEAD` 一致且含 `install-docker.sh`
- [ ] `curl -fsSL https://raw.githubusercontent.com/zhemed/LitePan/main/install-docker.sh | head -n 5` 返回 `#!/bin/bash` 的修复版（`v0.5.2-Beta`）
- [ ] `README.md` 含 `https://raw.githubusercontent.com/zhemed/LitePan/main/install-docker.sh` 且 `grep -c "new-api-own" ==0`
- [ ] `GOWORK=off go vet ./...` 仍 PASS

## Notes

- 上次误将脚本仅放 `data/` 并 `router` 直链，需 `LitePan-IP` 先有 LitePan，成死循环；本次纠正为仓库根提交，使 `curl raw.githubusercontent` 无需 `LitePan` 即可拉
