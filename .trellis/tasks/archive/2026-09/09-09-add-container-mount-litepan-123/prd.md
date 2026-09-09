# 09-09-add-container-mount-litepan-123

## Goal

为 litepan 容器创建映射目录 `LitePan-123`。用户选定方案 A：宿主机 `/root/LitePan/mounts/LitePan-123`——该目录位于既有 bind 挂载（`/root/LitePan/mounts → /app/mounts:shared`）之下，**容器内即时可见，无需重建容器**。

## Requirements

1. 宿主机创建 `/root/LitePan/mounts/LitePan-123`（目录已 gitignore 侧的 data/mounts 运行目录，不入 git）。
2. 验证：容器内 `/app/mounts/LitePan-123` 可见；双向读写探测（宿主写文件→容器可见；容器写文件→宿主可见）；health/登录/列表三连不回归。
3. 权限：容器内 root（容器以 root 运行）即可读写，无需额外 chown。
4. 更新任务记录（运行手册性质：容器挂载布局 + 该目录用途位）。

## Constraints

- 不重建容器、不改 compose、不动 DB。
- 目录名区分大小写：`LitePan-123`。

## Acceptance Criteria

- [x] 目录两侧存在且互通（宿主写→容器读 ✓；容器写→宿主读 ✓，探针已清理）
- [x] 容器无重启（boot_id 不变，Up 持续），health/登录/列表三连通过
- [x] 挂载布局核实：binds 仍为 `/root/LitePan/data:/app/data` + `/root/LitePan/mounts:/app/mounts:shared`，新目录经 shared 传播即时可见
- [ ] 归档 + journal + push
