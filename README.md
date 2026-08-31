<a name="readme-top"></a>

<div align="center">

<img src="docs/pictures/banner.png" alt="LitePan" width="100%">

# LitePan

**115 · 天翼 · 本机 · 一个界面 · 118M**

`Go 1.26.6` · `Vue 3.5.41` · `ghcr.io/zhemed/litepan:v0.0.7`

<br>

<a href="https://github.com/zhemed/LitePan"><img src="https://img.shields.io/badge/GitHub-zhemed%2FLitePan-1B1B2F?style=for-the-badge&logo=github&logoColor=white" alt="GitHub"></a>
<a href="https://github.com/zhemed/LitePan/pkgs/container/litepan"><img src="https://img.shields.io/badge/GHCR-ghcr.io%2Fzhemed%2Flitepan-2496ED?style=for-the-badge&logo=docker&logoColor=white" alt="GHCR"></a>
<a href="./LICENSE"><img src="https://img.shields.io/badge/License-PolyForm%20NC-red?style=for-the-badge" alt="License"></a>

</div>

> 私有部署版，基于 `Ponphil/LitePan` 精简：**仅 3 驱动**，无 `STRM` 等。

### 一键安装 Docker

```bash
curl -fsSL https://raw.githubusercontent.com/zhemed/LitePan/main/install-docker.sh | bash
```

## 快速开始

```yaml
services:
  litepan:
    image: ghcr.io/zhemed/litepan:v0.0.7
    container_name: litepan
    restart: always
    network_mode: host
    pid: "host"
    privileged: true
    environment:
      - TZ=Asia/Shanghai
    volumes:
      # LitePan 核心数据
      - /vol1/1000/docker/litepan/data:/app/data
      - /vol1/1000/docker/litepan/mounts:/app/mounts:shared
      # 映本地目录（不含 docker）
      - /vol1/1000/我的文件:/vol1/1000/我的文件:ro
    devices:
      - /dev/fuse:/dev/fuse
```

## 支持网盘

| 驱动 | 认证 |
|---|---|
| **115网盘Open** `115_open` | `OAuth` |
| **天翼云盘** `189_cloud` | `扫码登录` |
| **本机存储** `localfs` | `无` |

<p align="center"><a href="#readme-top">回顶部</a></p>
