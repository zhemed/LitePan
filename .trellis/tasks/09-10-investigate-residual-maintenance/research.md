# 残留维护项盘点报告（2026-09-10）

## 一、结论摘要

**七类中六类有残留，其中"磁盘"最重**：Docker 构建缓存 **17.05GB（可回收 16.03GB）**、39 个镜像（86% 可回收）、1 个两周前的残留容器；此外云端 190 个测试文件、本地 2G 测试文件、1 个旧 DB 备份、1 处 spec 契约缺口、3 项未完成优化。

## 二、逐类实测

| 类别 | 残留项 | 量 | 风险 | 建议 |
|---|---|---|---|---|
| **① 仓库** | 仅本任务目录未跟踪；TODO/FIXME = 0 | 1 项 | 无 | 保留（本次任务档案） |
| **② Docker** | **构建缓存 17.05GB（可回收 16.03GB）** | 16GB | 磁盘 | **P0 清理** |
| | 镜像 39 个（1.15GB，86% 可回收）：litepan 28 个版本 tag + `litepan-go` 20 个历史实验 tag + `litepan-own:0.0.8` | 1.0GB | 无 | P1 保留近 3 版，其余 prune |
| | 悬空镜像 3 个 | 小 | 无 | P0 `image prune` |
| | **残留容器 `litepan-auto`**（Exited，2 周前，镜像 litepan-own:0.0.8） | 1 个 | 混淆 | P1 删除容器+镜像 |
| **③ 数据库** | 任务行 7814（success 6004 + **paused 1810**） | 4.2M | 历史测试记录 | P2 保留策略（另议）或清理 |
| | `data/litepan.db.bak.1788077861`（8-30 旧备份） | 229K | 无 | P2 删或留存档 |
| | WAL 4.0M / shm 32K | 4M | 正常 | 保留（SQLite 机制） |
| **④ 云端（天翼）** | `bulk-test-2000` 内**残留 190 个测试文件**（原始 2000，已删大半） | 190 个 | 占用户云盘 | **P1 待你确认删除** |
| | root 探针文件（p1921/p1987/probe） | 0 | — | 已随旧树清除 ✓ |
| **⑤ 宿主** | `mounts/LitePan-123/bulk-test-2000` | **2.0G** | 磁盘；**1810 个暂停任务正引用其中文件**（删除将导致任务失败） | P1 待决策：续传完再删 / 立刻删（放弃任务） |
| | /tmp 杂物：lp_cookie、report.txt、sse2/3/first.txt、wheelurl.txt | KB 级 | 无 | P3 清理 |
| | /tmp/litepan-backup-*.db | 已不存在 ✓ | — | — |
| **⑥ Trellis** | 归档任务 103 个；journal 3319 行 | 正常规模 | 无 | 保留 |
| | **spec 契约缺口**：0.0.27 窗口化列表/汇总/SSE counts 属跨层 API 契约变更，按 update-spec 强制触发条件应有 code-spec 文档 | 1 处 | 未来接续易踩坑 | **P2 补 7 段式契约文档** |
| **⑦ 未完成事项** | 优化报告剩余：批次树记忆化、工作集保留策略 | 2 项 | 性能 | P2 按需 |
| | 防重复触发：决策记录（暂不实施） | 1 项 | 低 | 保留决策 |
| | 115 删除回环实测：无 115 账号，阻塞 | 1 项 | 无 | 待账号 |
| | flow_gate 短名不识别（本次踩到，需传全名） | 1 项 | 摩擦 | P3 增强后缀匹配 |

## 三、待批准的可执行清单（按优先级）

```bash
# P0 磁盘（预计回收 ≈16GB）
docker builder prune -f                     # 构建缓存 17.05GB → 回收 16.03GB
docker image prune -f                       # 3 个悬空镜像

# P1 Docker 存量
docker rm litepan-auto                      # 2 周前残留容器
docker rmi ghcr.io/zhemed/litepan-own:0.0.8 # 其镜像
docker images --format '{{.Repository}}:{{.Tag}} {{.ID}}' \
  | grep -E 'litepan:(v?0\.0\.(1[0-9]|2[0-6]))|litepan-go:' | awk '{print $2}' | sort -u | xargs -r docker rmi
# ↑ 保留 latest / 0.0.27 / 0.0.26（回滚点），清理其余历史 tag

# P1 数据（需你确认）
# 云端 190 个测试文件：UI 删除 bulk-test-2000，或我调 API 批删
# 本地 2G：rm -rf /root/LitePan/mounts/LitePan-123/bulk-test-2000（会令 1810 暂停任务转为"本地文件缺失"）

# P2 DB / spec
rm data/litepan.db.bak.1788077861           # 8-30 旧备份（可先移到 data/backups/）
# spec：为窗口化任务 API 补 code-spec（7 段式：签名/契约/校验矩阵/用例/测试点/正误对照）

# P3 /tmp
rm -f /tmp/lp_cookie /tmp/report.txt /tmp/sse*.txt /tmp/wheelurl.txt
```

## 四、证据局限

- 镜像"可回收"以 `docker system df` 口径为准（共享层会影响实际回收量）；
- 云端仅抽查了 root 与 bulk-test-2000 两层，未遍历用户其它目录；
- 1810 个暂停任务的本地文件引用按 `local_path` DISTINCT 统计（1810 个不同路径）。
