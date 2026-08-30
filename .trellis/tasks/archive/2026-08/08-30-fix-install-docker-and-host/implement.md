# Implementation Plan: Fix new-api-own install-docker script and host on LitePan

## Overview

按 `design.md` 拉取、检查、修复、验证、上传。

## Phase 1: 拉取与检查

- [ ] 1.1 `curl -fsSL https://raw.githubusercontent.com/zhemed/new-api-own/main/install-docker.sh -o /tmp/install-docker.sh && wc -l /tmp/install-docker.sh && head -n 30 /tmp/install-docker.sh`
  - `verify: cat /tmp/install-docker.sh | head | grep REQUIRED_DOCKER`
- [ ] 1.2 `grep -n "command -v docker" /tmp/install-docker.sh` 与 `grep -n "signed-by" /tmp/install-docker.sh` 检查为何跳过建源
  - `verify: grep -c "get.docker.com" /tmp/install-docker.sh` ==1 且 `grep -c "signed-by" ==0`（原脚本缺显式建源）

## Phase 2: 修复

- [ ] 2.1 `cp /tmp/install-docker.sh /tmp/install-docker-fixed.sh` 后编辑：
  - 若 `command -v docker` 已存在但 `docker --version` 非 `29.7.2`，仍 `curl -fsSL https://get.docker.com | sh`
  - `apt-get update` 前插入 `mkdir -p /etc/apt/keyrings && curl -fsSL https://download.docker.com/linux/debian/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg && echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/debian $(. /etc/os-release && echo $VERSION_CODENAME) stable" > /etc/apt/sources.list.d/docker.list`
  - `apt-get install` 的 `docker-ce-rootless-extras` 后加 `|| true` 或按 `ID=ubuntu` 区分
  - `verify: cat /tmp/install-docker-fixed.sh | grep -c "signed-by"` >=1

## Phase 3: 验证（本地）

- [ ] 3.1 `bash -n /tmp/install-docker-fixed.sh && echo ok` 语法检查
- [ ] 3.2 在 `Debian 12 bookworm`（当前 `10.0.0.11` 或容器）与 `Ubuntu 22.04` 容器中 `bash /tmp/install-docker-fixed.sh 2>&1 | tail -n 20` 验证 `Docker 29.7.2 + Compose v5.4.0`（已 `hold`）
  - `verify: docker --version | grep 29.7.2 && docker compose version | grep v5.4.0`

## Phase 4: 上传到 LitePan

- [ ] 4.1 `cp /tmp/install-docker-fixed.sh /root/LitePan/data/install-docker-fixed.sh && chmod +x /root/LitePan/data/install-docker-fixed.sh`
  - `verify: ls -lh /root/LitePan/data/install-docker-fixed.sh`
- [ ] 4.2 若 `LitePan` 的 `LocalUpload` 已映射 `/app/data`，则 `curl -s http://127.0.0.1:5211/api/auth/login -d "username=admin&password=admin" -c /tmp/c.txt` 后 `curl -b /tmp/c.txt http://127.0.0.1:5211/api/files/list?account_id=...&parent_id=...` 找到 `install-docker-fixed.sh` 的 `file_id`，再 `curl -s http://127.0.0.1:5211/api/files/download?account_id=...&file_id=... | head` 验证直链
  - `verify: curl -fsSL http://127.0.0.1:5211/api/files/download?account_id=... | head` 含 `REQUIRED_DOCKER`
- [ ] 4.3 在 `README.md` 或 `AGENTS.md` 中记录 `LitePan` 直链：`curl -fsSL http://<LitePan-IP>:5211/install-docker.sh | bash`（若 `data` 映射到 `LitePan-own` 的 `3` 个 `ro` 则需 `FileBrowser` 直链）

## Phase 5: Commit & Archive

- [ ] 5.1 `git add /tmp/install-docker-fixed.sh` 不入 `LitePan` 主仓库（`_extracted/` 已忽略），但 `data/install-docker-fixed.sh` 为 `bind mount` 不 `git`，仅 `cp` 后 `ls` 验证
- [ ] 5.2 `task.py archive 08-30-fix-install-docker-and-host --skip-branch-validation && git add .trellis/tasks/archive/... && git commit`
- [ ] 5.3 `add_session.py --commit <hash>`

## Rollback

- `rm /root/LitePan/data/install-docker-fixed.sh` + `rm /tmp/install-docker-fixed.sh`
