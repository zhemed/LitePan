<a name="readme-top"></a>

<div align="center">

<img src="docs/pictures/banner.png" alt="LitePan" width="100%">

<br>

<a href="https://github.com/zhemed/LitePan"><img src="https://img.shields.io/badge/GitHub-zhemed%2FLitePan-1B1B2F?style=for-the-badge&logo=github&logoColor=white" alt="GitHub"></a>
&nbsp;
<a href="https://hub.docker.com/r/zhemed/litepan"><img src="https://img.shields.io/badge/Docker-zhemed%2Flitepan-2496ED?style=for-the-badge&logo=docker&logoColor=white&labelColor=1B1B2F" alt="Docker"></a>
&nbsp;
<a href="https://space.bilibili.com/1501989416"><img src="https://img.shields.io/badge/Bilibili-交流与演示-00A1D6?style=for-the-badge&logo=bilibili&logoColor=white&labelColor=1B1B2F" alt="Bilibili"></a>

[![docker-pulls][docker-pulls-shield]][docker-url]
[![version][version-shield]][docker-url]
[![license][license-shield]][license-url]

</div>

<br>

> [!CAUTION]
> 本仓库为基于 `Ponphil/LitePan` 精简的私有部署版，已深度裁剪，仅保留 3 驱动与核心能力。
> 上游全功能 Go 版见 `Ponphil/LitePan`，Python 旧版已归档至 `LitePan-old`。

<br>

## ▎ 项目简介

**LitePan（精简版）** 是单二进制云盘聚合网盘，基于 `Ponphil/LitePan` 的 Go 版（`Go 1.26.6 / chi v5 / modernc.org/sqlite / Vue 3.5.41`）做减法：**一个 `litepan` 二进制 + 一个 `web` 构建产物（`internal/api/web`）**，`SQLite` 单库，`FUSE` 可选挂载，镜像 `118M`。

| 已移除（2026-08-30 精简，`+13012 -11000` 行） | 仍保留 |
|---|---|
| `STRM`（`internal/strm 44文件 + strmscrape 24`）/ `WebDAV` 本地共享（`internal/share/dav 16`） | `文件管理`（浏览/上传/下载/移动/复制/重命名/收藏） |
| `缓存保持`（`cacheretention 15`）/ `目录整理`（`mediaorganize 30+ / classifyorganize / aiorganize`） | `FUSE 本地挂载`（`fusemount` + `fusereadcache`） |
| `增强工具 7 项`（`Emby/飞牛/夸克TV/AI/分类/清理/海报`）/ `跨盘秒传`（`crosstransfer 4`） | `从服务器上传`（`LocalUpload`，增强工具唯一保留） |
| `8 驱动`（`123_Open/139Cloud/Baidu_Open/Guangya/OneDrive/OpenList/Quark/WebDAV`） | `115网盘Open / 天翼云盘 (189Cloud) / 本机存储 (LocalFs)` 3 驱动 + `离线下载` + `自动联动（仅 delay）` + `备份恢复` + `系统设置/日志` |

> **定位**：个人自用聚合，非全功能发行版；`STRM/整理/秒传/WebDAV` 等已删，若需全功能请用上游。

## ▎ 功能清单（按实际 118M 镜像）

* **多网盘聚合（仅 3 驱动）**：`115网盘Open`（`115_open`）、`天翼云盘`（`189_cloud`）、`本机存储`（`localfs`），`GET /admin/drivers → 3`。
* **文件管理**：`FileBrowser` 网格/列表、`Breadcrumb`、`AccountSelector`，支持 `List/Download/Move/Copy/Rename/Delete/Favorites/NameAlign/CreateFolder/Upload`，`FUSE` 挂载后可本地访问。
* **FUSE 本地挂载**：`fusemount` + `fusereadcache`（`7 天` / `LRU`），`mounts:shared` 传播。
* **从服务器上传**：前台 `新建 → 上传 → 从服务器上传`，`local_upload` 映射宿主机目录到容器内路径（`docker-compose` 中映射一侧），不暴露其他路径。
* **离线下载**：`offlinedownload`（`aria2` 风格 `335M` 磁力/BT，`builtin_offline` 临时目录）。
* **自动联动**：仅 `delay` 等待动作（`Emby/整理` 等已随增强工具移除）。
* **备份恢复**：`backuprestore`（`DB` 快照与回滚，`zstd`）。
* **系统设置/日志**：`settings`（`cacheTTL/logLevel` 等）、`logx` 自动清理。

## ▎ 支持网盘

| 驱动 | `Config.Name` | 认证 | 备注 |
|---|---|---|---|
| **115网盘Open** | `115_open` | `OAuth` `115网盘Open` | 官方 API，`SHA1` 秒传（保留 `ProvideHashes`） |
| **天翼云盘** | `189_cloud` | `扫码登录` | 个人云/家庭云，`MD5`，支持家庭云 |
| **本机存储** | `localfs` | `无` | 本地目录 `root_path`（`local_dir` 类型），容器挂载 |

> **已移除 8 驱动**：`123_Open`（123网盘）/ `139Cloud` / `Baidu_Open`（百度）/ `Guangya` / `OneDrive` / `OpenList` / `Quark`（夸克）/ `WebDAV`（远端挂载，与已删 `internal/share/dav` 的本地 WebDAV 不同）—— `drivers/all.go` 11→3，`drivers/template` 保留为脚手架。

## ▎ 挂载与更多功能

* **FUSE**：`fusemount` 需宿主机 `/dev/fuse` 与 `privileged: true` + `pid: host`，`mounts:/app/mounts:shared`。
* **从服务器上传**：在 `辅助工具 → 增强工具 → 从服务器上传` 中配置 `映射目录`（如 `媒体库 → /app/data/media`），前台按标签浏览。
* **其他**：`FUSE 读缓存`（`fuse_read_cache` 可选映射到更快磁盘）、离线下载、备份恢复。

## ▎ 快速开始

**Docker Compose 部署** · 镜像 `zhemed/litepan:latest` 或本地 `litepan-go:three-drivers`（`118M`）

```yaml
services:
  litepan:
    image: zhemed/litepan:latest  # 或本地构建的 litepan-go:three-drivers
    container_name: litepan
    restart: unless-stopped
    ports:
      - "5211:5211"
      # 内置离线下载的 Magnet TCP/uTP/DHT 监听；若在后台改过端口需同步映射
      - "42069:42069/tcp"
      - "42069:42069/udp"
    environment:
      - TZ=Asia/Shanghai
    volumes:
      - ./data:/app/data
      - ./mounts:/app/mounts:shared
      # 可选：FUSE 读缓存单独映射到更快磁盘
      # - ./fuse_read_cache:/app/data/fuse_read_cache
    devices:
      - /dev/fuse:/dev/fuse
    pid: "host"
    privileged: true
```

```bash
git clone https://github.com/zhemed/LitePan.git && cd LitePan
docker compose up -d
# 或本地构建
docker build -t litepan-go:three-drivers .  # 118M，Go 1.26.6 + Vue 3.5.41
docker run -d --name litepan -p 5211:5211 -p 42069:42069 -p 42069:42069/udp \
  -v ./data:/app/data -v ./mounts:/app/mounts:shared --device /dev/fuse --cap-add SYS_ADMIN --security-opt apparmor:unconfined \
  litepan-go:three-drivers
```

打开 `http://你的IP:5211`，**默认管理员 `admin / admin`**（`2026-08-30` 已落库）。需 `FUSE` 时确保宿主机具备 `/dev/fuse` 权限。

> [!WARNING]
> **不要用 `ponphil/litepan:latest` 部署本仓库。** `latest` 仍是 Python 旧版镜像。上游 Go 版见 `Ponphil/LitePan`，旧版归档见 `LitePan-old`。

## ▎ 技术栈与构建

* **后端**：`Go 1.26.6` `module litepan` `chi v5` `modernc.org/sqlite` `CGO_ENABLED=0` `GOTOOLCHAIN=local`
* **前端**：`Vue 3.5.41` `Vite 8.2.2` `vue-tsc 3.3.11` `pnpm` `web/vite.config.ts outDir ../internal/api/web` `//go:embed web`
* **质量**：`.golangci.yml`（`depguard: api-no-store` 等）、`GOWORK=off go vet ./...`、`GOWORK=off go build`、`web npm run type-check && npm run build`（`104 files`）、`docker build`（`fuse` 标签）
* **Trellis**：本仓库按 `trellis init --dsh -u zhemed` 托管，**所有操作必须调用 trellis**（`AGENTS.md` 强制），`spec` 在 `.trellis/spec`，`tasks` 在 `archive/2026-08`

```bash
GOWORK=off go vet ./...
GOWORK=off go build -o /tmp/litepan ./cmd/litepan  # 32M
cd web && npm run type-check && npm run build  # 104 files
docker build -t litepan-go:three-drivers .  # 118M
curl -s http://127.0.0.1:5211/api/health | grep ok
```

## ▎ 支持

<table>
  <tr>
    <td width="50%" valign="top">
      <h3>支持 LitePan</h3>
      <p>如果这个项目对你有帮助，欢迎点右上角 <strong>Star</strong> <a href="https://github.com/zhemed/LitePan">zhemed/LitePan</a>。</p>
      <img src="docs/pictures/wechat-tip.png" alt="微信赞赏" width="260">
    </td>
    <td width="50%" valign="top">
      <h3>赞助致谢</h3>
      <p>感谢每一位支持 LitePan 的朋友。</p>
      <p>完整致谢见上游官方：</p>
      <p>
        <a href="https://www.litepan.top/sponsor.html">https://www.litepan.top/sponsor.html</a>
      </p>
    </td>
  </tr>
</table>

## ▎ 反馈

交流请到 <a href="https://github.com/zhemed/LitePan/issues">GitHub Issues</a>（本仓库）或 <a href="https://space.bilibili.com/1501989416">B 站主页</a>（上游）。  
外部贡献致谢见 [ACKNOWLEDGEMENTS.md](./ACKNOWLEDGEMENTS.md)。

---

## ▎ 许可

[PolyForm Noncommercial 1.0.0](./LICENSE) — 个人学习与非商业使用，**禁止商用**。  
第三方依赖见 [THIRD_PARTY_NOTICES.md](./THIRD_PARTY_NOTICES.md)。请遵守各网盘服务条款与当地法规。

[docker-pulls-shield]: https://img.shields.io/docker/pulls/zhemed/litepan?logo=docker&logoColor=white&style=flat-square
[version-shield]: https://img.shields.io/badge/Version-v0.6.0--lite-6C63FF?style=flat-square
[license-shield]: https://img.shields.io/badge/License-PolyForm%20NC-red?style=flat-square
[docker-url]: https://github.com/zhemed/LitePan
[license-url]: ./LICENSE
