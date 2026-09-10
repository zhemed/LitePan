# cleanup-p1-residuals

## Goal

完成残留盘点报告的 **P1 清理**（用户已批准）：云端测试树、本地测试目录、关联暂停任务记录、残留容器与镜像 tag 治理。来源：`09-10-investigate-residual-maintenance` research.md §P1。

## 执行计划（顺序有依赖）

| 步 | 操作 | 前置 | 核验 |
|---|---|---|---|
| A | 删除云端 `bulk-test-2000` 测试树（外层目录 + 内层 190 文件） | 记录目录 ID 与文件数 | 重新 List：目录消失 |
| B | 删除关联 **1810 个暂停任务记录**（它们引用即将删除的本地文件） | 备份 DB | `upload_tasks` 行数下降、无残留 paused |
| C | 删除本地 `/root/LitePan/mounts/LitePan-123/bulk-test-2000`（2G） | B 完成 | 目录消失、磁盘释放 ≈2G |
| D | `docker rm litepan-auto` + `docker rmi litepan-own:0.0.8` | 记录基线 | 容器/镜像消失 |
| E | 镜像 tag 治理：保留 `litepan:latest`、`litepan:0.0.27`、`litepan:0.0.26`、`litepan-go:dev`；清其余 litepan/litepan-go 历史 tag | Dockerfile 已核实不依赖 litepan-go | 保留清单在位、docker system df 下降 |
| F | 收尾核验：服务三连、容器 StartedAt 未变、磁盘对比 | — | 全绿 |

## Requirements

1. B 步前必须备份 DB（`data/backups/`）
2. 云端删除走 API（`/api/files/delete`）；任务记录清理走 `/api/files/upload/tasks/batch-delete`（分块）
3. **硬性禁止**：不删运行容器/其镜像/卷；不删 `litepan:latest|0.0.27|0.0.26` 与 `litepan-go:dev`；不碰用户非测试数据（root 下仅 `bulk-test-2000` 属测试残留）
4. 每步即时核验并记录前后数字

## Acceptance Criteria

- [x] 云端测试树删除，root List 复查无残留（190 文件入 189 回收站，账号为 trash 模式）
- [x] DB 已备份（manual-pre-p1-20260910-192849.db）+ 1810 记录清理（7814→6004，失败 0）
- [x] 本地目录删除，磁盘 11G→8.8G（释放 2.0G）
- [x] 残留容器与 litepan-own:0.0.8 均已删除（容器 2→1）
- [x] 75 个历史 tag 清理，镜像 36→3、1.059GB→169.7MB（回收 ≈890MB），保留清单在位
- [x] 服务三连 ✓、StartedAt 未变、质量门 + 门禁 + 归档 + journal + push
