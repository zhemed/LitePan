# Update README with host compose deployment

## Goal

将用户于 `2026-08-30` 审阅通过的 `host` 网络 `ghcr.io/zhemed/litepan:v0.5.2-Beta` 部署方式（`network_mode: host` + `3 卷` + `ro 映射`）写入 `README.md` 的 `快速开始`，替换现有的 `bridge + ports` 示例，使 `README` 与 `litepan-go:three-drivers 118M` 的 `host` 最佳实践一致。

## Requirements

- **来源**：用户提供的 `compose`（已审计）：
  ```yaml
  services:
    litepan:
      image: ghcr.io/zhemed/litepan:v0.5.2-Beta
      container_name: litepan
      restart: always
      network_mode: host
      pid: "host"
      privileged: true
      environment: [TZ=Asia/Shanghai]
      volumes:
        - /vol1/1000/docker/litepan/data:/app/data
        - /vol1/1000/docker/litepan/mounts:/app/mounts:shared
        - /vol1/1000/我的文件:/vol1/1000/我的文件:ro
      devices: ["/dev/fuse:/dev/fuse"]
  ```
  经审计：`strm` 卷已删（`STRM` 已移除），`ports` 在 `host` 下无需，`restart: always` 保留（用户指定），`TZ` 保留
- **落盘**：`README.md` 的 `## 快速开始` 整体替换为 `host` 版（`image` 仍 `ghcr.io/zhemed/litepan:v0.5.2-Beta`，`volumes` 用 `host` 绝对路径示例，`ro` 保留）
- **不改**：`README.md` 的其余 4 屏（顶部/支持网盘/能做什么/许可）保持 `66 行` 极简版不变

## Constraints

- 仅改 `README.md` 的 `快速开始` 一节，不改 `internal/*` / `web/*` 代码
- `grep -c "network_mode: host" README.md` ==1，`grep -c "ghcr.io/zhemed/litepan:v0.5.2-Beta" README.md` >=2
- `grep -c "strm:/app/strm" README.md` ==0（已删 `strm`）

## Acceptance Criteria

- [ ] `README.md` 含 `network_mode: host` 且 `image: ghcr.io/zhemed/litepan:v0.5.2-Beta`
- [ ] `README.md` 含 `volumes: - /vol1/1000/docker/litepan/data:/app/data` 的 `host` 绝对路径示例
- [ ] `README.md` 无 `strm:/app/strm`
- [ ] `cat README.md | grep -A2 "快速开始"` 目视 `host`  compose
- [ ] `GOWORK=off go vet ./...` 仍 PASS

## Notes

- 本任务为文档同步，用户已审阅 `host` 部署方式，仅需落盘到 `README`
