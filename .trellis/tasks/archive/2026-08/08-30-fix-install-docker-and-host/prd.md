# Fix new-api-own install-docker script and host on LitePan

## Goal

拉取 `https://raw.githubusercontent.com/zhemed/new-api-own/main/install-docker.sh`（49 行，`Docker 29.7.2 + Compose v5.4.0`）到本地 `/tmp/install-docker.sh` 检查为何“有的系统能安装，有的系统安装不了”，修复兼容性后，上传到本工作区的 `LitePan`（`litepan-go:localupload 118M` 的 `LocalUpload` 或直接 `data/`落盘）使他人可通过 `LitePan` 的 `http://IP:5211` 链接直接拉取，无需再 `curl raw.githubusercontent`。

## Requirements

- **拉取与检查**：
  - `curl -fsSL https://raw.githubusercontent.com/zhemed/new-api-own/main/install-docker.sh -o /tmp/install-docker.sh` 已拉取 49 行，`set -euo pipefail`，`REQUIRED_DOCKER=29.7.2` 等
  - 检查点：`grep -qiE 'debian|ubuntu' /etc/os-release` 的 `debian/ubuntu` 判定、`command -v docker` 存在时跳过 `get.docker.com` 导致官方源未建、`apt-get install docker-ce=5:29.7.2*` 的版本 `candidate` 缺失（如 `Debian 12 bookworm` 的 `containerd.io` 冲突）、`docker-ce-rootless-extras` 在 `Ubuntu` 的 `has no candidate`、`systemctl enable` 在非 `systemd` 容器中的失败
- **修复**：
  - 若 `docker` 已存在但版本非 `29.7.2`，仍强制 `curl -fsSL https://get.docker.com | sh` 重建官方源（而非跳过）
  - `apt-get update` 前 `mkdir -p /etc/apt/keyrings` 与 `curl -fsSL https://download.docker.com/linux/debian/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg` 的显式建源（替代 `get.docker.com` 的隐式），并 `echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/debian $(. /etc/os-release && echo $VERSION_CODENAME) stable" > /etc/apt/sources.list.d/docker.list`
  - `apt-get install` 时对 `docker-ce-rootless-extras` 的缺失做 `|| true` 或按 `lsb_release` 区分 `Ubuntu` 与 `Debian` 的包名
  - 版本校验：`docker --version | grep -oP "Docker version \K[0-9.]+"` 与 `docker compose version | grep -oP` 的 `REQUIRED` 比对保留
- **上传到 LitePan**：
  - 修复后脚本落盘为 `/tmp/install-docker-fixed.sh`（49→~70 行），`chmod +x`
  - Via `LitePan`：`POST /api/admin/tools/local-upload/upload` 或直接 `cp` 到 `data/` 并通过 `LitePan` 的 `FileBrowser` 生成直链（`http://IP:5211/api/files/download?account_id=...&file_id=...`）或 `LitePan` 的 `data` 目录映射到 `LitePan-own` 的 `LocalUpload` 自动化（若 `LitePan` 的 `local_upload` 映射含 `/app/data`）
  - 使他人可 `curl -fsSL http://IP:5211/api/files/download?... | bash` 替代 `raw.githubusercontent`

## Constraints

- 仅改 `/tmp/install-docker.sh` 的兼容性，不改 `LitePan` 的 `Go` 代码（除非需 `local_upload` 映射）
- `LitePan` 的 `data` 为 `bind mount`（`/root/LitePan/data`），`LitePan-own` 的 `fix` 分支已验证 `115G 4分钟` 增量，`LitePan` 的 `three-drivers` 不影响 `install-docker.sh` 的 `apt` 逻辑
- `LitePan` 的 `admin/admin`（`must_change:true`）需先 `POST /api/auth/login` 取 `cookie` 再 `POST /api/admin/tools/local-upload/upload`

## Acceptance Criteria

- [ ] `/tmp/install-docker.sh` 已拉取 49 行，`cat /tmp/install-docker.sh | head` 含 `REQUIRED_DOCKER="29.7.2"`
- [ ] `/tmp/install-docker-fixed.sh` 存在且 `grep -c "get.docker.com" | wc -l` >=1 且 `grep -c "signed-by=/etc/apt/keyrings/docker.gpg"` >=1
- [ ] 在 `Debian 12 bookworm` 与 `Ubuntu 22.04` 的 `docker --version` 非 `29.7.2` 时，`bash /tmp/install-docker-fixed.sh` 均能 `apt-get install` 成功（`docker --version` 最终 `29.7.2`）
- [ ] 修复后脚本已上传到 `LitePan`：`curl -s -b /tmp/c.txt http://127.0.0.1:5211/api/admin/tools/local-upload/config | grep` 含新映射或 `ls /root/LitePan/data/install-docker-fixed.sh` 存在且 `curl -fsSL http://127.0.0.1:5211/api/files/download?... | head` 可拉取
- [ ] `GOWORK=off go vet ./...` 仍 PASS（未改 Go）

## Notes

- 原脚本在 `zhemed` 的 `10.0.0.99` 闭环已验 `Debian 12` 成功，但在 `Ubuntu 20.04` 的 `docker.io` 已装场景下跳过 `get.docker.com` 导致 `has no candidate`
- 上传到 `LitePan` 后，他人可 `curl -fsSL http://<LitePan-IP>:5211/install-docker.sh | bash` 替代 `raw.githubusercontent`
