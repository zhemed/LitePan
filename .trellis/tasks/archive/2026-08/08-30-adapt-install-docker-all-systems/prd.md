# Adapt install-docker to all systems and remove hold comment

## Goal

将 `install-docker.sh` 从 `仅支持 Debian/Ubuntu` 适配为 **所有主流 Linux 发行版**（Debian/Ubuntu/CentOS/RHEL/Fedora/Arch/Amazon Linux 等）均可 `curl | bash`，并按用户要求**移除** `README.md` 与 `install-docker.sh` 中 `# → Docker 29.7.2 + Compose v5.4.0（已 hold）` 的注释行，继续完善通用逻辑。

## Requirements

- **适配所有系统**：
  - `if ! grep -qiE 'debian|ubuntu' /etc/os-release` 的 `exit 1` 改为 **非 Debian/Ubuntu 时走通用兜底**：`echo "未识别为 Debian/Ubuntu，走 get.docker.com 通用安装..." >&2 && curl -fsSL https://get.docker.com | sh`，并在 `get.docker.com` 后尝试 `apt-get / yum / dnf / pacman` 的版本 `hold` 逻辑按发行版分支
  - `Debian/Ubuntu` 分支保留 `signed-by` 建源与 `29.7.2` 版本 pin 逻辑；`CentOS/RHEL/Fedora` 分支走 `yum/dnf` 的 `docker-ce` 安装；`Arch` 走 `pacman -S docker docker-compose`；其他未知发行版直接 `get.docker.com | sh` 后校验版本
  - 保持 `set -euo pipefail` 下各分支的 `gpg` 缺失容错（`command -v gpg || apt-get install -y gnupg` 已在 `99` 上验证）
- **移除 hold 注释**：
  - `README.md:27` 的 `# → Docker 29.7.2 + Compose v5.4.0（已 hold）` 整行删
  - `install-docker.sh:89` 的 `echo "✅ 完成：Docker $DOCKER_VER + Compose $COMPOSE_VER 已就绪（已 hold 锁定版本）"` 中的 `（已 hold 锁定版本）` 后缀删，改为 `已就绪`
  - `install-docker.sh` 顶部 `修复点` 注释中 `已 hold` 相关描述同步删

## Constraints

- 仅改 `./install-docker.sh` 与 `README.md` 的上述行，不改 `LitePan` 的 `Go` 代码
- `install-docker.sh` 需 `bash -n` 通过，且 `grep -c "已 hold" ==0`
- `README.md` 的 `grep -c "已 hold" ==0`

## Acceptance Criteria

- [ ] `./install-docker.sh` 无 `已 hold` 字符串（`grep -c "已 hold" ==0`）
- [ ] `README.md` 无 `已 hold` 字符串
- [ ] `./install-docker.sh` 含 `Arch` / `CentOS` / `Fedora` 等非 `Debian` 分支（`grep -c "arch\|centos\|fedora" -i >=2`）
- [ ] `bash -n ./install-docker.sh && echo ok`
- [ ] 在 `Debian12`（`10.0.0.99`）上 `bash ./install-docker.sh 2>&1 | tail` 仍 `✅ 完成：Docker 29.7.2`
- [ ] `GOWORK=off go vet ./...` 仍 PASS

## Notes

- 上次 `d54f5e1` 修复了 `Debian12` 的 `gpg` 缺失，本次在此 94 行基础上扩展至全平台
