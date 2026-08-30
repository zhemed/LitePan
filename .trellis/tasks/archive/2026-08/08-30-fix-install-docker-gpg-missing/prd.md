# Fix install-docker gpg not found on Debian12

## Goal

修复 `https://raw.githubusercontent.com/zhemed/LitePan/main/install-docker.sh` 在 `10.0.0.99`（`root/1234`，`Debian12`）上 `bash: line 41: gpg: command not found` + `curl: (23) Failed writing body` 导致 `Docker 29.7.2` 安装失败，使 `curl -fsSL https://raw.githubusercontent.com/zhemed/LitePan/main/install-docker.sh | bash` 在最小化 `Debian12`（无 `gpg`）上亦能成功。

## Requirements

- **复现**：`10.0.0.99` 的 `Debian12` 最小化安装，无 `gpg`（`which gpg` 无），`bash install-docker.sh` 在 `配置 Docker 官方源...` 后 `curl ... | gpg --dearmor` 因 `gpg` 缺失而 `exit 1`（`set -e`），`curl` 的 `Failed writing body` 为管道破裂
- **修复**：
  - 在 `mkdir -p /etc/apt/keyrings` 前 `apt-get update -qq` 后 `apt-get install -y gnupg`（或 `gnupg2` 兼容）确保 `gpg` 可用；若 `apt-get` 本身需 `gpg` 则先 `apt-get install -y --no-install-recommends ca-certificates curl gnupg`
  - 或改用 `curl -fsSL https://download.docker.com/linux/debian/gpg | tee /etc/apt/keyrings/docker.asc` + `gpg --dearmor` 的替代为 `curl ... | gpg --dearmor` 前 `command -v gpg >/dev/null || apt-get install -y gnupg`
  - 保持 `set -euo pipefail` 下 `gpg` 缺失不直接 `exit`，而是先装 `gnupg`
- **验证**：
  - `bash -n install-docker.sh` 语法通过
  - 在 `10.0.0.99`（`root/1234`）上 `ssh root@10.0.0.99 "bash -c 'curl -fsSL https://raw.githubusercontent.com/zhemed/LitePan/main/install-docker.sh | bash'"` 最终 `Docker 29.7.2 + Compose v5.4.0`（`hold`）

## Constraints

- 仅改 `./install-docker.sh`（89 行 → ~95 行），不改 `LitePan` 的 `Go` 代码（`internal/*`）
- `install-docker.sh` 需 `bash -n` 通过，且 `grep -c "gnupg" >=1`
- `LitePan` 的 `data/install-docker.sh` 同步为 `install-docker.sh`（若 `data` 仍有则 `rm`，仅仓库根为主）

## Acceptance Criteria

- [ ] `./install-docker.sh` 含 `apt-get install -y gnupg` 或 `command -v gpg || apt-get install` 的容错
- [ ] `bash -n ./install-docker.sh && echo ok`
- [ ] `ssh root@10.0.0.99 "which gpg || echo missing"` 在修复后 `which gpg` 有路径
- [ ] `ssh root@10.0.0.99 "curl -fsSL https://raw.githubusercontent.com/zhemed/LitePan/main/install-docker.sh | bash" 2>&1 | tail` 含 `✅ 完成：Docker 29.7.2`
- [ ] `GOWORK=off go vet ./...` 仍 PASS

## Notes

- `10.0.0.99` 为 `Debian12` 最小化，未预装 `gnupg`，`gpg` 缺失导致 `curl | gpg` 管道破裂
