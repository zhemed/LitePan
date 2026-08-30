# Design: Correct install-docker hosting to repo and update README

## Overview

上次 `aceca55` 的 `GET /install-docker.sh` 需先有 `LitePan` 实例，对“先装 Docker 再起 LitePan” 的用户成死循环。本次将 `89 行` 修复版提交到 `zhemed/LitePan` 仓库根，使 `raw.githubusercontent` 直链无需 `LitePlan-IP`。

## Boundaries

| 产出 | 旧 | 新 |
|---|---|---|
| **仓库文件** | 无 `install-docker.sh`，`data/install-docker.sh` 仅宿主机 `bind mount` | `./install-docker.sh` 89 行 `chmod +x`，`git add` 到 `zhemed/LitePan` |
| **README** | `docker pull ghcr.io/...` + `compose` 中 `image: ghcr.io/...`，`install-docker.sh` 仅 `new-api-own` 的 `raw` | `README` 新增 `一键安装 Docker` 小节：`curl -fsSL https://raw.githubusercontent.com/zhemed/LitePan/main/install-docker.sh | bash`，去 `new-api-own` |
| **LitePan 实例直链** | `router.go` 的 `GET /install-docker.sh` 从 `h.dataDir` 读 | 保留（兼容），`README` 主推 `raw.githubusercontent` |

## Data Flow

```
https://raw.githubusercontent.com/zhemed/new-api-own/main/install-docker.sh (49 行)
  ↓ 修复（signed-by + rootless 容错）→ 89 行
/tmp/install-docker-fixed.sh
  ↓ cp → ./install-docker.sh (仓库根)
git add + commit + push → https://github.com/zhemed/LitePan → https://raw.githubusercontent.com/zhemed/LitePan/main/install-docker.sh
  ↓ 他人 curl | bash → Docker 29.7.2 + Compose v5.4.0
```

## Compatibility

- `install-docker.sh` 为 `bash` 独立脚本，不依赖 `LitePan` 的 `Go` 代码，`go vet` 不影响
- `README` 的 `ghcr.io` 段保留，新增 `install-docker.sh` 小节不冲突

## Tradeoffs

- **仓库根 vs docs/**：放根便于 `raw.githubusercontent` 短链，`docs/` 需 `.../docs/install-docker.sh` 长链，故选根
- **保留实例直链**：`router.go` 的 `GET /install-docker.sh` 保留，`data/install-docker.sh` 仍可 `http://IP:5211/install-docker.sh`，与 `raw` 双轨

## File Map

1. `./install-docker.sh`（89 行，新）
2. `README.md`（+5 行 `一键安装 Docker` 小节）
