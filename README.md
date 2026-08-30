<a name="readme-top"></a>

<div align="center">

<img src="docs/pictures/banner.png" alt="LitePan" width="100%">

<br>

<a href="https://www.litepan.top"><img src="https://img.shields.io/badge/官网文档-www.litepan.top-6C63FF?style=for-the-badge&labelColor=1B1B2F" alt="官网文档"></a>
&nbsp;
<a href="https://space.bilibili.com/1501989416"><img src="https://img.shields.io/badge/Bilibili-交流与演示-00A1D6?style=for-the-badge&logo=bilibili&logoColor=white&labelColor=1B1B2F" alt="Bilibili"></a>
&nbsp;
<a href="https://github.com/zhemed/LitePan"><img src="https://img.shields.io/badge/Docker-zhemed%2FLitePan-2496ED?style=for-the-badge&logo=docker&logoColor=white&labelColor=1B1B2F" alt="Docker"></a>


[![docker-pulls][docker-pulls-shield]][docker-url]
[![version][version-shield]][docker-url]
[![license][license-shield]][license-url]

</div>

<br>

> [!CAUTION]
> 当前仓库是正在开发中的 **Go 版 LitePan**，首次发布可能问题较多，请谨慎测试。
> Python 旧版已归档至 [LitePan-old](https://github.com/Ponphil/LitePan-old)。


<br>

## ▎ 功能简述

<table>
  <tr>
    <td width="50%" valign="top" align="center">
      <h3>多网盘聚合（仅 3 驱动）</h3>
      <p align="left">仅支持 <strong>115网盘Open / 天翼云盘 (189Cloud) / 本机存储 (LocalFs)</strong> 3 驱动，其余 8 驱动已移除（2026-08-30 three-drivers）。一个界面看完。</p>
      <img src="docs/pictures/feature-browser.png" alt="多网盘聚合" height="220">
    </td>
    <td width="50%" valign="top" align="center">
      <h3>自动联动（仅 delay）</h3>
      <p align="left">仅保留 <code>delay</code> 等待动作，Emby/整理等联动已移除。</p>
      <img src="docs/pictures/feature-automation.png" alt="自动联动" height="220">
    </td>
  </tr>
</table>

## ▎ 挂载与更多功能

支持 **FUSE 本地挂载** 与 **从服务器上传**（`LocalUpload`，增强工具中唯一保留），另有离线下载等能力。`WebDAV`（`internal/share/dav`）、`跨盘秒传`、`目录整理`、`缓存保持` 等已移除（2026-08-30 精简）。

> **支持网盘**：`115网盘Open` / `天翼云盘 (189Cloud)` / `本机存储 (LocalFs)` 3 项。其余 8 驱动（`123_Open` / `139Cloud` / `Baidu_Open` / `Guangya` / `OneDrive` / `OpenList` / `Quark` / `WebDAV`）已移除，镜像 `128M → 118M`。

---

## ▎ 快速开始

**Docker Compose 部署** · 镜像标签：`latest`（`zhemed/litepan`）或本地 `litepan-go:three-drivers`（118M，仅 3 驱动）

```yaml
services:
  litepan:
    image: zhemed/litepan:latest  # 或 litepan-go:three-drivers（本地构建 118M）
    container_name: litepan
    restart: unless-stopped
    ports:
      - "5211:5211"
      # 内置 Magnet 的 TCP/uTP/DHT 监听端口；若在后台修改，需同步调整映射
      - "42069:42069/tcp"
      - "42069:42069/udp"
    environment:
      - TZ=Asia/Shanghai
    volumes:
      - ./data:/app/data
      - ./mounts:/app/mounts:shared

      # 可选：将 FUSE 读缓存单独映射，建议放到更快的磁盘
      # - ./fuse_read_cache:/app/data/fuse_read_cache
    devices:
      - /dev/fuse:/dev/fuse
    pid: "host"
    privileged: true
```

打开 `http://你的IP:5211`，默认管理员密码均为admin。  
需要 FUSE 时请确保宿主机具备 `/dev/fuse` 权限。

> [!WARNING]
> **不要用 `ponphil/litepan:latest` 部署本仓库对应的 Go 版。**  
> `latest` 仍是 Python 旧版镜像。若你需要旧版程序与 Compose 脚本，请前往归档仓库：[LitePan-old](https://github.com/Ponphil/LitePan-old)。

## ▎ 支持

<table>
  <tr>
    <td width="50%" valign="top">
      <h3>支持 LitePan</h3>
      <p>如果这个项目对你有帮助，欢迎点右上角 <strong>Star</strong>，也欢迎自愿赞赏。</p>
      <img src="docs/pictures/wechat-tip.png" alt="微信赞赏" width="260">
    </td>
    <td width="50%" valign="top">
      <h3>赞助致谢</h3>
      <p>感谢每一位支持 LitePan 的朋友。</p>
      <p>完整致谢名单见官方网站：</p>
      <p>
        <a href="https://www.litepan.top/sponsor.html">https://www.litepan.top/sponsor.html</a>
      </p>
    </td>
  </tr>
</table>

## ▎ 反馈

交流请到 <a href="https://space.bilibili.com/1501989416">B 站主页</a>。  
暂不接受公开 PR；有维护意愿请私信。
外部贡献致谢见 [ACKNOWLEDGEMENTS.md](./ACKNOWLEDGEMENTS.md)。

---

## ▎ 许可

[PolyForm Noncommercial 1.0.0](./LICENSE) — 个人学习与非商业使用，**禁止商用**。  
第三方依赖见 [THIRD_PARTY_NOTICES.md](./THIRD_PARTY_NOTICES.md)。请遵守各网盘服务条款与当地法规。

[docker-pulls-shield]: https://img.shields.io/docker/pulls/ponphil/litepan?logo=docker&logoColor=white&style=flat-square
[version-shield]: https://img.shields.io/badge/Version-v0.6.0--lite-6C63FF?style=flat-square
[license-shield]: https://img.shields.io/badge/License-PolyForm%20NC-red?style=flat-square
[docker-url]: https://github.com/zhemed/LitePan
[license-url]: ./LICENSE
