<a name="readme-top"></a>

<div align="center">

<img src="docs/pictures/banner.png" alt="LitePan" width="100%">

<br>

<a href="https://github.com/zhemed/LitePan"><img src="https://img.shields.io/badge/GitHub-zhemed%2FLitePan-1B1B2F?style=for-the-badge&logo=github&logoColor=white" alt="GitHub"></a>
&nbsp;
<a href="https://github.com/zhemed/LitePan/pkgs/container/litepan"><img src="https://img.shields.io/badge/GHCR-ghcr.io%2Fzhemed%2Flitepan-2496ED?style=for-the-badge&logo=docker&logoColor=white&labelColor=1B1B2F" alt="GHCR"></a>
&nbsp;
<a href="https://space.bilibili.com/1501989416"><img src="https://img.shields.io/badge/Bilibili-交流与演示-00A1D6?style=for-the-badge&logo=bilibili&logoColor=white&labelColor=1B1B2F" alt="Bilibili"></a>

[![ghcr-pulls][ghcr-pulls-shield]][ghcr-url]
[![version][version-shield]][ghcr-url]
[![license][license-shield]][license-url]

</div>

<br>

> [!CAUTION]
> 本仓库为 `zhemed` 基于 `Ponphil/LitePan` 的精简私有部署版，镜像 `ghcr.io/zhemed/litepan:v0.5.2-Beta`（`118M`，仅 **115网盘Open / 天翼云盘 / 本机存储** 3 驱动）。
> 上游全功能 Go 版见 `Ponphil/LitePan`，Python 旧版已归档至 `LitePan-old`。

## 快速开始

**Docker（GHCR）** · 定版 `v0.5.2-Beta`，浮动 `latest` 同步

```bash
docker pull ghcr.io/zhemed/litepan:v0.5.2-Beta
docker pull ghcr.io/zhemed/litepan:latest
```

**Docker Compose** · `ghcr.io/zhemed/litepan:v0.5.2-Beta`（`118M`，`3 驱动`）

```yaml
services:
  litepan:
    image: ghcr.io/zhemed/litepan:v0.5.2-Beta
    container_name: litepan
    restart: unless-stopped
    ports:
      - "5211:5211"
      - "42069:42069/tcp"
      - "42069:42069/udp"
    environment:
      - TZ=Asia/Shanghai
    volumes:
      - ./data:/app/data
      - ./mounts:/app/mounts:shared
    devices:
      - /dev/fuse:/dev/fuse
    pid: "host"
    privileged: true
```

```bash
git clone https://github.com/zhemed/LitePan.git && cd LitePan
docker compose up -d  # 默认读取上例 ghcr.io 镜像
# 本地构建（可选）
docker build -t ghcr.io/zhemed/litepan:v0.5.2-Beta .  # 118M
```

打开 `http://你的IP:5211`，**默认 `admin / admin`**，首次登录需改密。需 `FUSE` 时确保宿主机具备 `/dev/fuse` 权限。

> [!WARNING]
> **不要用 `ponphil/litepan:latest`**，`latest` 仍是 Python 旧版。上游 Go 版见 `Ponphil/LitePan`。

## 支持网盘

| 驱动 | `Config.Name` | 认证 | 备注 |
|---|---|---|---|
| **115网盘Open** | `115_open` | `OAuth` | 官方 API，`SHA1` |
| **天翼云盘** | `189_cloud` | `扫码登录` | 个人云/家庭云，`MD5` |
| **本机存储** | `localfs` | `无` | 本地目录 `root_path` |

> 已移除 8 驱动：`123_Open` / `139Cloud` / `Baidu_Open` / `Guangya` / `OneDrive` / `OpenList` / `Quark` / `WebDAV`（`drivers/all.go` 11→3），`template` 保留。

## 功能清单

* **多网盘聚合**：`115/189/LocalFs` 三合一，`GET /admin/drivers → 3`，`AccountSelector` 切换。
* **文件管理**：`FileBrowser` 网格/列表、`Breadcrumb`，`List/Download/Move/Copy/Rename/Delete/Favorites/NameAlign/CreateFolder/Upload`，`FUSE` 挂载后本地访问。
* **FUSE 本地挂载**：`fusemount` + `fusereadcache`（`7 天 LRU`），`mounts:shared`。
* **从服务器上传**：`辅助工具 → 增强工具 → 从服务器上传`，映射宿主机目录到 `LocalUpload`，为增强工具唯一保留。
* **离线下载**：`offlinedownload`（`Magnet`/`BT`，`builtin_offline`）。
* **自动联动**：`delay` 等待、`local_upload`（`LitePan-own` 移植，全量 `sha256` 增量，`local_upload_state_*.json`）。
* **备份恢复**：`backuprestore`（`DB` 快照 `zstd`）。
* **系统设置/日志**：`settings`/`logx` 自动清理。

> **已移除**：`STRM` / `WebDAV` 本地共享 / `缓存保持` / `目录整理` / `跨盘秒传` / `增强工具 7 项`（`Emby/飞牛/夸克TV/AI/分类/清理/海报`）—— 镜像 `128M → 118M`。

## 技术栈与验证

* **后端**：`Go 1.26.6` `chi v5` `modernc.org/sqlite` `CGO_ENABLED=0` · **前端**：`Vue 3.5.41` `Vite 8.2.2` `vue-tsc 3.3.11` `pnpm` · `//go:embed web`
* **质量**：`GOWORK=off go vet` / `go build -o /tmp/litepan` `32M` / `web type-check && build` `104 files` / `docker build` `fuse` 标签
* **Trellis**：`trellis init --dsh -u zhemed`，**所有操作必须调用 trellis**（`AGENTS.md`），`spec` 在 `.trellis/spec`

```bash
GOWORK=off go vet ./...
GOWORK=off go build -o /tmp/litepan ./cmd/litepan
cd web && npm run type-check && npm run build
docker pull ghcr.io/zhemed/litepan:v0.5.2-Beta && curl -s http://127.0.0.1:5211/api/health | grep ok
```

## 许可与支持

<table>
  <tr>
    <td width="50%" valign="top">
      <h3>支持 LitePan</h3>
      <p>欢迎 Star <a href="https://github.com/zhemed/LitePan">zhemed/LitePan</a>。</p>
      <img src="docs/pictures/wechat-tip.png" alt="微信赞赏" width="260">
    </td>
    <td width="50%" valign="top">
      <h3>赞助致谢</h3>
      <p>完整名单见上游：<a href="https://www.litepan.top/sponsor.html">litepan.top/sponsor</a></p>
    </td>
  </tr>
</table>

交流：<a href="https://github.com/zhemed/LitePan/issues">GitHub Issues</a> · <a href="https://space.bilibili.com/1501989416">B 站</a> · 贡献见 [ACKNOWLEDGEMENTS.md](./ACKNOWLEDGEMENTS.md)

---

[PolyForm Noncommercial 1.0.0](./LICENSE) — 个人学习与非商业使用，**禁止商用**。第三方见 [THIRD_PARTY_NOTICES.md](./THIRD_PARTY_NOTICES.md)。

[ghcr-pulls-shield]: https://img.shields.io/badge/Pulls-GHCR-2496ED?style=flat-square&logo=docker
[version-shield]: https://img.shields.io/badge/Version-v0.5.2--Beta-6C63FF?style=flat-square
[license-shield]: https://img.shields.io/badge/License-PolyForm%20NC-red?style=flat-square
[ghcr-url]: https://github.com/zhemed/LitePan/pkgs/container/litepan
[license-url]: ./LICENSE
