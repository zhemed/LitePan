# Thoroughly refactor README from scratch

## Goal

抛弃基于 `Ponphil/LitePan v0.5.2-Beta` 全功能版的旧 `README.md` 结构，按 `zhemed/LitePan`（`litepan-go:three-drivers 118M`，已移除 STRM/共享/缓存整理/增强工具 7 项/跨盘秒传/8 驱动，仅保留 115/天翼/本机存储等）的**当前真实代码**从零重写一篇可直接 `git clone` 后部署的精简版 `README.md`。

## Requirements

- **先彻底移除旧 README 的不符段落**：
  - 删除 `功能简述` 中已删的 `跨盘秒传`、`目录整理` 的图文与 `挂载与更多功能` 中 `WebDAV/302/缓存保持` 等描述
  - 删除 `extra_hosts` 的 `tmdb` 段落与 `feature-crosstransfer.png/feature-organize.png` 的引用（`docs/pictures` 中对应图片若不再引用可保留文件但不引）
- **再从零重写，按实际分 7 章**：
  1. **顶部**：`banner` 保留，`badge` 的 `docker-pulls/version` 指向 `zhemed/LitePan`，`version` 为 `v0.6.0-lite` 或 `three-drivers`，`license` 保留
  2. **项目简介**：一句话说明“基于 `Ponphil/LitePan` 精简的私有部署版，仅保留 3 驱动与核心文件管理，用于个人聚合”，并用表格或列表明确 `已移除（2026-08-30 精简）` 与 `仍保留` 的对比（`STRM 44文件 / WebDAV dav 16文件 / cacheretention 15文件 / mediaorganize 30+ / 7 增强工具 / crosstransfer 4文件 / 8 驱动 13012 行` 已删）
  3. **功能清单（按实际）**：仅列 `多网盘聚合（115/189/LocalFs）`、`文件管理（浏览/上传/下载/移动/复制/重命名/收藏）`、`FUSE 本地挂载`、`从服务器上传（LocalUpload 唯一增强工具）`、`离线下载`、`自动联动（仅 delay）`、`备份恢复`、`系统设置/日志`，每项 1 行说明，不再出现 `跨盘/整理/STRM/WebDAV` 等
  4. **支持网盘**：独立小节，表格列 `115网盘Open` / `天翼云盘 (189Cloud)` / `本机存储 (LocalFs)` 3 项，含 `display_name / auth_type / 备注`，并单段注明“其余 8 驱动已移除：123_Open/139Cloud/Baidu_Open/Guangya/OneDrive/OpenList/Quark/WebDAV”
  5. **快速开始**：`Docker Compose` 完整示例 `image: zhemed/litepan:latest`（或 `litepan-go:three-drivers` 118M），`ports 5211/42069`、`volumes data/mounts:shared`、`devices /dev/fuse`、`pid: host + privileged`，`TZ=Asia/Shanghai`，`volumes` 仅 `data/mounts`（`fuse_read_cache` 可选），`打开 http://IP:5211 默认 admin/123456`，`FUSE` 权限说明，`WARNING` 段保留但更新为 `不要用 ponphil/litepan:latest`
  6. **技术栈与构建**：`Go 1.26.6 / chi v5 / modernc.org/sqlite / Vue 3.5.41 / Vite 8.2.2 / pnpm`，`GOWORK=off go vet/build`、`web npm run build → internal/api/web`，`docker build -t litepan-go:three-drivers .` 118M
  7. **许可与致谢**：保留 `PolyForm Noncommercial 1.0.0`、`THIRD_PARTY_NOTICES.md`、`ACKNOWLEDGEMENTS.md`，链接改为 `zhemed/LitePan`，`B 站` 等保留

## Constraints

- 仅改 `README.md`（与 `docs/pictures` 中已删功能的图片引用），不改 `internal/*` / `web/*` 代码
- 全文 `zh-CN` 为主，保留 `a name="readme-top"` 锚点、`AGENTS.md` 的 `所有操作必须调用 trellis` 不受影响
- 新 `README.md` 约 150-220 行，结构清晰，`grep -c "跨盘秒传" ==0`（除“已移除”对比表外可有 1 处说明）、`grep -c "feature-crosstransfer" ==0`
- `cat README.md | head` 能直接看到 `zhemed/LitePan` 与 3 驱动列表

## Acceptance Criteria

- [ ] `README.md` 无 `跨盘秒传`/`目录整理` 的功能图文，仅在“已移除”对比表中可提及一次
- [ ] `README.md` 无 `feature-crosstransfer.png` / `feature-organize.png` 的 `<img>` 引用
- [ ] `README.md` 的 `快速开始` 的 `image:` 为 `zhemed/litepan` 非 `ponphil/litepan:beta`，`version-shield` 为 `v0.6.0-lite` 或 `three-drivers`
- [ ] `README.md` 有独立 `支持网盘` 小节明确列 3 驱动，其余 8 标注已移除
- [ ] `README.md` 的 `功能清单` 与当前 `118M` 镜像能力一致，无 `STRM/WebDAV/缓存保持/7 增强工具/跨盘` 等已删描述
- [ ] `GOWORK=off go vet ./...` 仍 PASS（文档不影响构建）

## Notes

- 本任务为文档彻底重构，非补丁；`docs/pictures/feature-crosstransfer.png` 等图片文件本身可保留但不被 `README` 引用
- 基于 `litepan-go:three-drivers` 的 `drivers/all.go 3 导入` 与 `CloudToolsPanel 8→1` 现状
