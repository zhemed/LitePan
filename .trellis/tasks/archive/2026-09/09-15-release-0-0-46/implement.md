# Implementation Plan: 发布 v0.0.46 + 非特权部署

## Overview

顺序：**版本号（P1）→ 质量门（P2）→ 推送源码（P3）→ 镜像（P4）→ tag/release（P5）→ 备份（P6）→ 重建容器（P7）→ 部署面验证（P8）→ 清残留（P9）→ 归档（P10）**。

不可逆动作（GHCR 推送 / tag / release / 容器重建 / 删目录）全部排在验证之后；**P6 备份未通过不得进 P7**。

本机 `danger-full-access`、审批关闭 —— 不请求 escalation。

---

## Phase 0: 前置快照

- [ ] 0.1 `git status --short`（期望仅本任务目录）、`git log --oneline origin/main -1`（期望 `4958e78`）
- [ ] 0.2 记录容器基线（部署前）：`docker inspect litepan` 的 `Image/Privileged/PidMode/Devices/Binds/Mounts/Env/RestartPolicy` → 存 `/tmp/rel46/` 供 P8 比对
- [ ] 0.3 记录 GHCR 现状：`gh api /users/zhemed/packages/container/litepan/versions --jq '.[0]'`（`0.0.45` 三 tag 同 digest）
- [ ] 0.4 记录实例库现状（只读）：version=24、`fuse_mounts` 存在 0 行、`configs` 7 行、表数 10
- [ ] 0.5 记录残留目录现状：`data/fuse_read_cache/`（内容与体积）、`mounts/`（应为空）

**回滚点 R0**：以上即后续比对依据

---

## Phase 1: D1 版本号 → v0.0.46

- [ ] 1.1 `internal/buildinfo/version.go`：`v0.0.45` → `v0.0.46`
- [ ] 1.2 `README.md` 2 处、`docker-compose.yml` 1 处、`docker-compose.fnos.yml` 1 处
- [ ] 1.3 **验证门 G1**：`grep -rn "v0\.0\.45" internal/ web/src README.md docker-compose*.yml` **零命中**；`grep -rn "v0\.0\.46"` 命中 5 处；`git diff --stat` 只含这 4 个文件
- **回滚点 R1**：`git checkout HEAD -- internal/buildinfo/version.go README.md docker-compose.yml docker-compose.fnos.yml`

---

## Phase 2: D2 发布前质量门

- [ ] 2.1 `make lint` → `0 issues.`
- [ ] 2.2 `GOWORK=off go vet ./...` → exit=0
- [ ] 2.3 `GOWORK=off go test ./...` → 全包 ok、FAIL=0（期望 26 包 ok）
- [ ] 2.4 `cd web && npm run type-check` → exit=0
- [ ] 2.5 **验证门 G2**：`git status --short internal/api/web` **无输出**（不重建 embed）
- **回滚点 RQ**：任一不过 → 回 P1 修，不得进 P3

---

## Phase 3: D3 提交并推送 main

- [ ] 3.1 提交（说明本轮发布内容 = FUSE 移除 + 部署特权收窄）
- [ ] 3.2 `git push origin main`
- [ ] 3.3 **验证门 G3**：`git status --short` 干净；`git log origin/main -1` = 本次提交
- **回滚点 R3**：`git revert <commit> && git push`

---

## Phase 4: D4 构建并推送镜像（三 tag 同 digest）

- [ ] 4.1 **后台**构建：`make docker-build DOCKER_IMAGE=ghcr.io/zhemed/litepan:v0.0.46`
- [ ] 4.2 **验证门 G4a**：本地镜像存在且 ID ≠ `v0.0.45` 的 ID；**镜像内二进制** `grep -c v0.0.46 /app/litepan` = 1、`grep -c v0.0.45` = 0；`--help` 可执行
- [ ] 4.3 `docker tag` 补 `:0.0.46`、`:latest`
- [ ] 4.4 `gh auth token | docker login ghcr.io -u zhemed --password-stdin`
- [ ] 4.5 `docker push` ×3
- [ ] 4.6 **验证门 G4b**：GHCR 单条 version 同时含 `v0.0.46`/`0.0.46`/`latest`，且 `v0.0.45` 仍指向旧 digest
- **回滚点 R4**：删该 GHCR version 的三个 tag

---

## Phase 5: D5 git tag + release

- [ ] 5.1 确认 `main` 已推
- [ ] 5.2 `git tag v0.0.46` → `git push origin v0.0.46`
- [ ] 5.3 写 `/tmp/rel46/release-notes.md`（移除内容 + **升级提示：可去掉 privileged/pid host/devices/mounts** + 迁移 0025 行为 + **完整回滚需恢复备份**）→ `gh release create v0.0.46 --title "v0.0.46" --notes-file …`
- [ ] 5.4 **验证门 G5**：`gh api repos/zhemed/LitePan/git/ref/tags/v0.0.46 --jq .object.sha` == `git rev-parse v0.0.46`；`gh release view v0.0.46 --json tagName,isDraft,url` 非草稿
- **回滚点 R5**：`gh release delete v0.0.46 --cleanup-tag`

---

## Phase 6: D6 部署前备份（硬前置）

- [ ] 6.1 用 SQLite 在线备份 API 落 `data/backups/manual-pre-0025-<YYYYmmdd-HHMMSS>.db`（禁止裸 `cp`）
- [ ] 6.2 **验证门 G6**：只读打开该快照 → `PRAGMA integrity_check` = ok、`version` = 24、`fuse_mounts` 表**存在**且 0 行、`configs` = 7 行、表数 = 10
- **回滚点 R6**：无（这一步只增不删）

---

## Phase 7: D7 用非特权 compose 重建容器

- [ ] 7.1 `docker compose pull litepan`
- [ ] 7.2 `docker compose up -d`（**不得**手搓 `docker run`）
- [ ] 7.3 **验证门 G7**：`docker ps` 显示新容器使用 `ghcr.io/zhemed/litepan:v0.0.46`
- **回滚点 R7**：compose image 改回 `v0.0.45` → `docker compose up -d`（+ 需要完整回滚时恢复 P6 备份）

---

## Phase 8: D8 部署面验证（本任务核心证明）

- [ ] 8.1 **G8a**：`/api/health` 200；登录（表单编码）200；`/api/public/system-config` 的 `version` = `v0.0.46`
- [ ] 8.2 **G8b（迁移真跑）**：实例库只读 → `schema_migrations` = 25、`fuse_mounts` 与 `idx_fuse_mounts_*` **不存在**、`fuse_*` 键 0、`configs` **仍 7 行**、表数 **9**
- [ ] 8.3 **G8c（最高风险点）**：`POST /api/admin/backups`（带密码）→ 2xx「备份创建成功」，响应含 `schema_version=25`
- [ ] 8.4 **G8d（收益兑现）**：`docker inspect` → `Privileged=false`、`PidMode=""`、`Devices=[]`、无 `/app/mounts` 绑定；端口仍 `5211`
- [ ] 8.5 **G8e**：`docker compose logs --since 5m` 中 `level=ERROR` = 0；旧端点 `/api/admin/fuse/{status,mounts,read-cache}` 均 **404**
- [ ] 8.6 **G8f（浏览器）**：`bw` 打开 `:5211` → 仪表盘**3 张卡片**、页面无「FUSE」文案、设置页无 FUSE 读缓存项、布局无塌陷；截图留档
- **回滚点 R8**：见 R7

---

## Phase 9: D9 清理两个残留目录

- [ ] 9.1 前置核对：`mounts/` 确为空且已不在 compose；`data/fuse_read_cache/` 仅空 `blocks/` + 空索引（记录体积）
- [ ] 9.2 删除二者
- [ ] 9.3 **验证门 G9**：容器仍在跑、`/api/health` 200、`docker compose logs` 无新错误、`git status` 无新增跟踪改动
- **回滚点 R9**：两者都是空目录，无需回滚

---

## Phase 10: D10 归档收尾

- [ ] 10.1 spec 复核（版本号相关引用若有则同步）
- [ ] 10.2 `skill trellis-check` 走查
- [ ] 10.3 `flow_gate.py mark-check release-0-0-46 --note "<质量门+发布+部署摘要>"`
- [ ] 10.4 勾选 `prd.md` 全部验收项
- [ ] 10.5 `flow_gate.py pre-archive` → `task.py archive release-0-0-46 --skip-branch-validation`（auto-commit 已知会失败 → 手工提交）
- [ ] 10.6 `add_session.py` → `git push`

---

## Validation Commands（汇总）

```bash
export PATH=/usr/local/go/bin:/root/go/bin:$PATH GOWORK=off GOPROXY=https://goproxy.cn,direct
cd /root/LitePan

# G1
grep -rn "v0\.0\.45" internal/ web/src README.md docker-compose*.yml ; grep -rn "v0\.0\.46" internal/buildinfo README.md docker-compose*.yml

# GQ
make lint && go vet ./... && go test ./... && (cd web && npm run type-check)

# G4
make docker-build DOCKER_IMAGE=ghcr.io/zhemed/litepan:v0.0.46
docker run --rm --entrypoint /bin/sh ghcr.io/zhemed/litepan:v0.0.46 -c "grep -c v0.0.46 /app/litepan; grep -c v0.0.45 /app/litepan"
gh api /users/zhemed/packages/container/litepan/versions --jq '.[0] | {tags: .metadata.container.tags, digest: .name}'

# G5
gh api repos/zhemed/LitePan/git/ref/tags/v0.0.46 --jq .object.sha ; git rev-parse v0.0.46

# G6 备份（python3 sqlite3 Connection.backup）+ 只读校验

# G7/G8 部署与验证
docker compose pull litepan && docker compose up -d
curl -s -o /dev/null -w '%{http_code}\n' http://127.0.0.1:5211/api/health
docker inspect litepan --format '{{.HostConfig.Privileged}} {{.HostConfig.PidMode}} {{.HostConfig.Devices}} {{json .Mounts}}'
curl -s -X POST http://127.0.0.1:5211/api/auth/login -d 'username=admin&password=123456' -c /tmp/rel46/c.txt -o /dev/null
curl -s -b /tmp/rel46/c.txt -X POST http://127.0.0.1:5211/api/admin/backups -H 'Content-Type: application/json' -d '{"password":"verify12345","note":"post-0025-verify"}'
docker compose logs --since 5m litepan | grep -c level=ERROR

# G9 残留清理
ls -A mounts/ data/fuse_read_cache/ && rm -rf mounts data/fuse_read_cache
```

## Review Gates

| 门 | 位置 | 判据 |
|---|---|---|
| G1 | 1.3 | `v0.0.45` 零命中、`v0.0.46` 5 处、diff 只 4 文件 |
| GQ | 2 | lint 0 / vet 0 / test 全 ok / type-check 0 / embed 零改动 |
| G3 | 3.3 | main 已推且工作区干净 |
| G4a/b | 4.2/4.6 | 镜像含 v0.0.46 且不含 v0.0.45；GHCR 三 tag 同 digest |
| G5 | 5.4 | 远端 tag sha == 本地；release 非草稿 |
| G6 | 6.2 | 备份可开、version=24、`fuse_mounts` 在、configs=7 |
| G7 | 7.3 | 容器跑 v0.0.46 |
| G8a–f | 8.1–8.6 | health/登录/version；迁移 25 且表消失、configs=7；备份创建成功；**特权项全无**；日志 0 ERROR + 旧端点 404；浏览器 3 卡片 |
| G9 | 9.3 | 删残留后 health 仍 200、无新错误 |

## Rollback

见 design.md → **Rollback**。要点：对外删 release/tag/GHCR version；容器 compose 回 `v0.0.45`；**完整回滚必须恢复 `manual-pre-0025-*` 备份**（迁移已执行，旧镜像查不到 `fuse_mounts`）。

## Out of Scope

- 上游同步、`1.0.0`、GHCR 旧版本清理
- 任何新的功能开发；`offline_handoff` 命名漂移
- 修改 FUSE 无关的部署形态（`network_mode`、端口、restart 策略）

---

## 执行记录（实施后回填，仅记录事实与偏差）

**全部门禁通过（每条都真实执行）**：G1（`v0.0.45` 源码零残留、`v0.0.46` 命中 5 处、diff 恰 4 文件）；GQ（lint 0 / vet 0 / 26 包 ok / type-check 0 / embed 零改动）；G3（`origin/main = 26a96c0`）；G4a（镜像 `31b122a6…`，**镜像内二进制** `v0.0.46`=1、`v0.0.45`=0，且镜像内已无 `fusermount3`、无 `/app/mounts`）；G4b（GHCR 单条 version 含三 tag 同 digest，`v0.0.45` 仍指向旧 digest）；G5（远端 tag sha `26a96c02…` == 本地、release 非草稿）；G6（备份 `data/backups/manual-pre-0025-20260915-131329.db`，`integrity_check=ok`、version 24、`fuse_mounts` 在、`configs` 7 行）；G7（容器 `v0.0.46`）；G8a–f；G9（删残留后 health 仍 200）。

**部署面收益（本任务的核心证明，`docker inspect` 前后逐项比对）**：

| 项 | 部署前 | 部署后 |
|---|---|---|
| `Privileged` | `True` | **`False`** |
| `PidMode` | `host` | **`""`** |
| `Devices` | `/dev/fuse` | **`None`** |
| `Mounts` | `data` + `mounts:shared` | **只剩 `data`** |
| Binds / PortBindings / RestartPolicy / Env / NetworkMode | — | **逐项不变** |

**实例库迁移**：`schema_migrations` 24 → **25**、`fuse_mounts` 与其索引消失、`fuse_*` 键 0、**`configs` 仍 7 行**、表数 10 → **9**。
**最高风险点**：`POST /api/admin/backups` → **201「备份创建成功」**（`app_version=v0.0.46`、`schema_version=25`），证明 `BackupCounts` 的漏改风险未带入发布版。
**浏览器验收（真实例）**：仪表盘恰 **3 张卡片**、页面无「FUSE」、设置页「其他设置」只有两组且无 FUSE 读缓存项、首页页脚显示 **v0.0.46**、`window.__err` 为 null；旧端点 `/api/admin/fuse/*` 三条全 **404**；日志 `level=ERROR` = **0**。
**残留清理**：`./mounts/`（空、已不在 compose）与 `data/fuse_read_cache/`（68K、空 blocks、零代码读者）已删除，删后 health 200、无新错误。

### 偏差 1（我的操作失误，已修正并披露）

上一个任务（`09-15-cleanup-fuse-tails`）的提交 `bcd38f0` **漏加暂存**：只提交了 `maintenance.go` 的删除与新增的行为测试，三个真正的改动（`Dockerfile`、`internal/upload/temp.go`、`internal/upload/manager.go`）仍留在工作区。本轮初始 diff 因此出现"计划外文件"。

处理：① 先确认 `bcd38f0` 那棵树**仍可编译**（临时 worktree 实测 `go build ./...` exit=0，历史里没有坏提交）；② 用 `aa9d244` 补交主体改动并写明原因；③ 本任务的"未越界"验收项随附说明：本任务自身 diff 只有 4 个版本文件，`aa9d244` 属于上一任务的补交。
**教训**：`git commit` 前应看 `git status --short` 的**两列状态**（` M` 与 `M ` 不同），或直接 `git add -A -- <范围>`。

### 偏差 2（无实质影响）

`docker compose` 重建后 `SecurityOpt` 由 `['label=disable']` 变为 `None` —— compose 文件里从未声明该项，属旧容器创建时的历史遗留（本机 SELinux 未启用，无实际影响），且新容器权限更收窄，符合本轮方向。

### 未做永久回归测试的理由

本轮是发布/部署类任务，唯一源码改动是 5 处版本字符串；`BackupCounts` 与临时文件清理的回归测试已在 `09-15-remove-fuse` 与 `09-15-cleanup-fuse-tails` 中补齐。发布本身由 G1–G9 的实测门覆盖。
