<a name="readme-top"></a>

<div align="center">

<img src="docs/pictures/banner.png" alt="LitePan" width="100%">

**LitePan · 115 / 天翼 / 本机 · 118M**

`115网盘 · 天翼云盘 · 本机存储` 一个界面

<br>

<a href="https://github.com/zhemed/LitePan"><img src="https://img.shields.io/badge/GitHub-zhemed%2FLitePan-1B1B2F?style=for-the-badge&logo=github&logoColor=white" alt="GitHub"></a>
<a href="https://github.com/zhemed/LitePan/pkgs/container/litepan"><img src="https://img.shields.io/badge/GHCR-ghcr.io%2Fzhemed%2Flitepan-2496ED?style=for-the-badge&logo=docker&logoColor=white" alt="GHCR"></a>
<a href="./LICENSE"><img src="https://img.shields.io/badge/License-PolyForm%20NC-red?style=for-the-badge" alt="License"></a>

`ghcr.io/zhemed/litepan:v0.5.2-Beta` · `118M` · `3 驱动`

</div>

> 精简版 `zhemed` 基于 `Ponphil/LitePan`：**仅 3 驱动与文件管理核心**，`STRM/WebDAV/跨盘/整理/7 项增强` 已移除。

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
# compose.yml
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
# http://IP:5211  admin / admin 首次改密
```

## 能做什么

* **看/管**：多账号浏览 `List/Download/Move/Copy/Rename/Delete/Favorites`
* **挂**：`FUSE` `mounts:shared` + `从服务器上传`（`LocalUpload`）
* **动**：`delay` / `local_upload`（`sha256` 增量，`LitePan-own` 移植）
* **备**：`备份恢复` `zstd` · `离线下载` · `系统设置`

## 验证

```bash
GOWORK=off go vet ./... && cd web && npm run type-check
curl -s http://127.0.0.1:5211/api/health | grep ok  # 3 驱动
```

## 许可

[PolyForm Noncommercial 1.0.0](./LICENSE) · [ACKNOWLEDGEMENTS.md](./ACKNOWLEDGEMENTS.md) · [Issues](https://github.com/zhemed/LitePan/issues)

<p align="center"><a href="#readme-top">回顶部</a></p>
