<a name="readme-top"></a>

<div align="center">

<img src="docs/pictures/banner.png" alt="LitePan" width="100%">

**LitePan · 私有云盘聚合 · 3 驱动 · 118M**

`115网盘 · 天翼云盘 · 本机存储` 一个界面看完

<br>

<a href="https://github.com/zhemed/LitePan"><img src="https://img.shields.io/badge/GitHub-zhemed%2FLitePan-1B1B2F?style=for-the-badge&logo=github&logoColor=white" alt="GitHub"></a>
<a href="https://github.com/zhemed/LitePan/pkgs/container/litepan"><img src="https://img.shields.io/badge/GHCR-ghcr.io%2Fzhemed%2Flitepan-2496ED?style=for-the-badge&logo=docker&logoColor=white" alt="GHCR"></a>
<a href="./LICENSE"><img src="https://img.shields.io/badge/License-PolyForm%20NC-red?style=for-the-badge" alt="License"></a>

<br>

`ghcr.io/zhemed/litepan:v0.5.2-Beta` · `Go 1.26.6` · `Vue 3.5.41` · `118M`

</div>

---

## 一句话

> **基于 `Ponphil/LitePan` 精简的私有部署版**：只留 `115网盘Open / 天翼云盘 / 本机存储` 3 驱动与文件管理核心，`STRM / WebDAV / 跨盘秒传 / 目录整理 / 7 项增强工具` 已移除，单二进制 `118M`，`FUSE` 可选挂载。

* 上游：`Ponphil/LitePan`（全功能 Go 版）· 旧版：`LitePan-old`（Python）

## 支持网盘

| 驱动 | 认证 | 备注 |
|---|---|---|
| **115网盘Open** `115_open` | `OAuth` | 官方 API |
| **天翼云盘** `189_cloud` | `扫码登录` | 个人云 / 家庭云 |
| **本机存储** `localfs` | `无` | 本地目录 `root_path` |

## 能做什么

* **看**：`FileBrowser` 多账号统一浏览，`Breadcrumb` 导航，`AccountSelector` 切换
* **管**：`List / Download / Move / Copy / Rename / Delete / Favorites / NameAlign / CreateFolder / Upload`
* **挂**：`FUSE` 本地挂载 `mounts:/app/mounts:shared` + `7 天 LRU` 读缓存
* **传**：`从服务器上传`（`LocalUpload` 映射宿主机目录，不暴露其他路径）· `离线下载`（`Magnet/BT`）
* **动**：`自动联动` `delay` + `local_upload`（`LitePan-own` 移植，全量 `sha256` 增量，`local_upload_state_*.json`）
* **备**：`备份恢复` `DB` 快照 `zstd` · `系统设置/日志` 自动清理

## 快速开始

### GHCR（推荐）

```bash
docker pull ghcr.io/zhemed/litepan:v0.5.2-Beta
docker pull ghcr.io/zhemed/litepan:latest
```

### Compose

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
git clone https://github.com/zhemed/LitePan.git && cd LitePan
docker compose up -d
```

打开 `http://IP:5211`，`admin / admin` 首次需改密。需 `FUSE` 确保宿主机有 `/dev/fuse`。

> `ponphil/litepan:latest` 仍为 Python 旧版，请勿混用。

## 验证

```bash
GOWORK=off go vet ./... && GOWORK=off go build -o /tmp/litepan ./cmd/litepan  # 32M
cd web && npm run type-check && npm run build  # 104 files
curl -s http://127.0.0.1:5211/api/health | grep ok
curl -s -b /tmp/c.txt http://127.0.0.1:5211/api/admin/drivers | jq length  # 3
```

## 许可

[PolyForm Noncommercial 1.0.0](./LICENSE) — 个人学习与非商业使用。第三方见 [THIRD_PARTY_NOTICES.md](./THIRD_PARTY_NOTICES.md)。

<p align="center">
  <a href="https://github.com/zhemed/LitePan/issues">Issues</a> · <a href="https://space.bilibili.com/1501989416">B 站</a> · <a href="./ACKNOWLEDGEMENTS.md">致谢</a> · <a href="#readme-top">回顶部</a>
</p>
