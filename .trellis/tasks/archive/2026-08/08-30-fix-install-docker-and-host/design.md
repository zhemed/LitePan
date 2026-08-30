# Design: Fix new-api-own install-docker script and host on LitePan

## Overview

原脚本 `49 行` 的 `set -euo pipefail` + `REQUIRED_DOCKER=29.7.2` 在 `Debian/Ubuntu` 上因 `已装 docker` 时跳过 `get.docker.com` 而官方源未建，导致 `apt-get install docker-ce=5:29.7.2*` 的 `has no candidate`。需显式建源与版本容错，并落盘到 `LitePan` 的 `LocalUpload` 使 `http://IP:5211` 可直链。

## Boundaries

| 层 | 改 | 不改 |
|---|---|---|
| **脚本** | `/tmp/install-docker.sh` → `/tmp/install-docker-fixed.sh`（`get.docker.com` 强制、`signed-by` 建源、`rootless-extras` 容错） | `LitePan` 的 `Go` 代码（`internal/*`） |
| **LitePan** | `data/install-docker-fixed.sh` 落盘 + `LocalUpload` 映射（`/_extracted` 已 `gitignore`，`data` 为 `bind mount`） | `drivers` 3 驱动、`internal/cache` 等 |
| **验证** | `Debian 12 bookworm` 与 `Ubuntu 22.04` 的 `docker --version` 非 `29.7.2` 时 `bash` 均成功 | `LitePan-own` 的 `9` 个自有 commits |

## Data Flow

```
https://raw.githubusercontent.com/zhemed/new-api-own/main/install-docker.sh
  ↓ curl -o /tmp/install-docker.sh (49 行)
检查：grep -qiE debian|ubuntu /etc/os-release + command -v docker 已存在时跳过 get.docker.com → has no candidate
  ↓ 修复：强制 get.docker.com + 显式 signed-by 建源 + rootless-extras || true
/tmp/install-docker-fixed.sh (~70 行)
/tmp/install-docker-fixed.sh -- 验证：bash 在 Debian 12/Ubuntu 22.04 均 Docker 29.7.2
  ↓ 上传：cp 到 /root/LitePan/data/ 或 POST /api/admin/tools/local-upload/upload
LitePan: http://IP:5211/api/files/download?account_id=...&file_id=... → 他人 curl | bash
```

## Compatibility

- `get.docker.com | sh` 已处理 `VERSION_CODENAME`，显式 `signed-by` 仅加固，不冲突
- `docker-ce-rootless-extras` 在 `Debian 12` 存在但 `Ubuntu 20.04` 的 `docker.io` 已装时 `has no candidate`，`|| true` 容错
- `LitePan` 的 `data` 为 `bind mount`，`LitePan-own` 的 `fix` 分支已验证 `115G` 增量，`LitePan` 的 `three-drivers` 不影响 `apt`

## Tradeoffs

- **强制重建源 vs 跳过**：强制 `get.docker.com` 多 `10s`，但保证 `29.7.2` 的 `candidate` 存在，符合用户“有的系统能装有的不能”的根因
- **显式 `signed-by` vs 隐式**：显式增加 5 行，但使 `bookworm` 的 `signed-by` 路径明确，避免 `apt-get update` 的 `NO_PUBKEY`

## Rollout / Rollback

- 单 `cp` 落盘到 `data/`，`rm /root/LitePan/data/install-docker-fixed.sh` 即回滚
- 若 `LitePan` 的 `LocalUpload` 映射含 `/app/data`，则 `http://IP:5211` 直链即生效，无需 `docker` 重启

## File Map

1. `/tmp/install-docker.sh`（49 行，原）
2. `/tmp/install-docker-fixed.sh`（~70 行，新）
3. `/root/LitePan/data/install-docker-fixed.sh`（落盘）
4. `LitePan` 的 `LocalUpload` 直链（`http://IP:5211/api/files/download?...`）
