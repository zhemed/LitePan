<a name="readme-top"></a>

<div align="center">

<img src="docs/pictures/banner.png" alt="LitePan" width="100%">

# LitePan

**115 · 天翼 · 本机 · 一个界面 · 118M**

`Go 1.26.6` · `Vue 3.5.41` · `ghcr.io/zhemed/litepan:v0.5.2-Beta`

<br>

<a href="https://github.com/zhemed/LitePan"><img src="https://img.shields.io/badge/GitHub-zhemed%2FLitePan-1B1B2F?style=for-the-badge&logo=github&logoColor=white" alt="GitHub"></a>
<a href="https://github.com/zhemed/LitePan/pkgs/container/litepan"><img src="https://img.shields.io/badge/GHCR-ghcr.io%2Fzhemed%2Flitepan-2496ED?style=for-the-badge&logo=docker&logoColor=white" alt="GHCR"></a>
<a href="./LICENSE"><img src="https://img.shields.io/badge/License-PolyForm%20NC-red?style=for-the-badge" alt="License"></a>

</div>

> 私有部署版，基于 `Ponphil/LitePan` 精简：**仅 3 驱动**，无 `STRM` 等。

## 支持网盘

| 驱动 | 认证 |
|---|---|
| **115网盘Open** `115_open` | `OAuth` |
| **天翼云盘** `189_cloud` | `扫码登录` |
| **本机存储** `localfs` | `无` |

## 快速开始

```bash
docker pull ghcr.io/zhemed/litepan:v0.5.2-Beta
```

```yaml
services:
  litepan:
    image: ghcr.io/zhemed/litepan:v0.5.2-Beta
    container_name: litepan
    restart: unless-stopped
    ports: ["5211:5211", "42069:42069/tcp", "42069:42069/udp"]
    environment: [TZ=Asia/Shanghai]
    volumes: ["./data:/app/data", "./mounts:/app/mounts:shared"]
    devices: ["/dev/fuse:/dev/fuse"]
    pid: "host"
    privileged: true
```

```bash
git clone https://github.com/zhemed/LitePan.git && cd LitePan && docker compose up -d
# http://IP:5211  admin / admin
```

## 能做什么

* 看 / 管：`FileBrowser` 浏览 `List/Download/Move/Copy/Rename`
* 挂：`FUSE` `mounts:shared` · 传：`从服务器上传` · 离线：`Magnet/BT`
* 动：`delay` · 备：`zstd` 快照

---

[PolyForm Noncommercial 1.0.0](./LICENSE) · [Issues](https://github.com/zhemed/LitePan/issues) · [致谢](./ACKNOWLEDGEMENTS.md)

<p align="center"><a href="#readme-top">回顶部</a></p>
