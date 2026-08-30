# Remove quick start and features sections from README

## Goal

按用户“算了，按照我的要求把快速开始...能做什么...PolyForm...删除就行了”将 `README.md` 中指定的 3 段彻底删除，使 `README.md` 更简洁。

## Requirements

- **删除段落 1 - 快速开始**：
  - `## 快速开始` 标题及以下全部内容：
    ```bash
    docker pull ghcr.io/zhemed/litepan:v0.5.2-Beta
    ```
    ```yaml
    services:
      litepan:
        image: ghcr.io/zhemed/litepan:v0.5.2-Beta
        ...
        privileged: true
    ```
    ```bash
    git clone https://github.com/zhemed/LitePan.git && cd LitePan && docker compose up -d
    # http://IP:5211  admin / admin
    ```
- **删除段落 2 - 能做什么**：
  - `## 能做什么` 标题及以下 3 行：
    ```
    看 / 管：FileBrowser 浏览 List/Download/Move/Copy/Rename
    挂：FUSE mounts:shared · 传：从服务器上传 · 离线：Magnet/BT
    动：delay · 备：zstd 快照
    ```
- **删除段落 3 - 许可行**：
  - `PolyForm Noncommercial 1.0.0 · Issues · 致谢` 所在行（`[PolyForm...` / `<p align="center"><a href="#readme-top">回顶部</a></p>` 前的许可行）

## Constraints

- 仅删 `README.md` 的上述 3 段，不改 `internal/*` / `web/*` 代码
- 保留 `README.md` 的其余部分：`顶部 banner`、`支持网盘` 表格、`许可` 标题后的 `PolyForm` 段（若与删除的许可行重复需保留标题）、`AGENTS.md` 强制句
- `grep -c "docker pull ghcr.io/zhemed/litepan:v0.5.2-Beta" README.md` == 0（快速开始已删）
- `grep -c "能做什么" README.md` == 0
- `grep -c "PolyForm Noncommercial 1.0.0 · Issues" README.md` == 0（若为许可行）

## Acceptance Criteria

- [ ] `README.md` 无 `## 快速开始` 标题
- [ ] `README.md` 无 `docker pull ghcr.io/zhemed/litepan:v0.5.2-Beta` 的 `bash` 段
- [ ] `README.md` 无 `## 能做什么` 标题
- [ ] `README.md` 无 `看 / 管：FileBrowser` 3 行
- [ ] `README.md` 无 `PolyForm Noncommercial 1.0.0 · Issues · 致谢` 行（或仅在 `## 许可` 标题下保留 1 处）
- [ ] `wc -l README.md` 约 40-50（原 66 行删 3 段后）
- [ ] `GOWORK=off go vet ./...` 仍 PASS

## Notes

- 用户明确“算了...删除就行了”，为轻量文档任务，`design.md/implement.md` 可省略
- 备份管理等其他功能不受影响
