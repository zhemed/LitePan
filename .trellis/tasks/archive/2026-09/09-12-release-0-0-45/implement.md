# Implementation Plan: 发布 v0.0.45

## Overview

顺序：**版本号与文档（P1）→ 质量门（P2）→ 提交推送（P3）→ 构建推镜像（P4）→ tag 与 release（P5）→ 备份 + 本机部署 + 验收（P6）→ 归档（P7）**。

不可逆动作（GHCR 推送 / tag / release / 容器重建）全部排在**本地验证之后**。任一步失败即**停在原地**，后续步骤不得执行。

本机 `danger-full-access`、审批关闭 —— 不请求 escalation。

---

## Phase 0: 前置快照

- [x] 0.1 `git status --short`（期望：仅本任务目录未跟踪）、`git log --oneline origin/main -1`（期望 `0d41eb4`）
- [x] 0.2 记录发版基线：`docker inspect litepan` 的 `Mounts/PortBindings/Devices/Privileged/PidMode/RestartPolicy/Env` + `Image`
- [x] 0.3 记录 GHCR 现状：`gh api /users/zhemed/packages/container/litepan/versions --jq '.[0]'`（`0.0.44` 三 tag 同 digest）
- [x] 0.4 记录实例库现状（只读）：`schema_migrations` = 23、`api_keys` 存在 0 行、`configs` 7 行；`5211` 被容器占用

**回滚点 R0**：以上即后续比对依据

---

## Phase 1: D1 版本号 → v0.0.45

- [x] 1.1 `internal/buildinfo/version.go`：`v0.0.44` → `v0.0.45`
- [x] 1.2 `README.md` 2 处、`docker-compose.yml` 1 处、`docker-compose.fnos.yml` 1 处：同改
- [x] 1.3 **验证门 G1**：
      - `grep -rn "v0\.0\.44" internal/ web/src README.md docker-compose*.yml` → **零命中**
      - `grep -rn "v0\.0\.45" internal/buildinfo README.md docker-compose*.yml` → **命中 5 处**
      - `git diff --stat` 只含这 4 个文件、且**不含** `internal/api/web/**`
- **回滚点 R1**：`git checkout HEAD -- internal/buildinfo/version.go README.md docker-compose.yml docker-compose.fnos.yml`

---

## Phase 2: D2 发布前质量门

- [x] 2.1 `make lint` → `0 issues.`
- [x] 2.2 `GOWORK=off go vet ./...` → exit=0
- [x] 2.3 `GOWORK=off go test ./...` → 全包 `ok`、`FAIL` 包数 0
- [x] 2.4 `cd web && npm run type-check` → exit=0
- [x] 2.5 **不重建 embed**（KD2）：确认 `git status --short internal/api/web` **无输出**
- **回滚点 RQ**：任一不过 → 回 P1 修，**不得进入 P3**

---

## Phase 3: D3 提交并推送 main

- [x] 3.1 提交（message 说明：版本号 → `v0.0.45`，本轮发布内容为 API 秘钥功能移除）
- [x] 3.2 `git push origin main`
- [x] 3.3 **验证门 G3**：`git status --short` 干净；`git log origin/main -1 --oneline` = 本次提交
- **回滚点 R3**：`git revert <commit> && git push`

---

## Phase 4: D4 构建并推送镜像（3 tag 同 digest）

- [x] 4.1 构建（**后台任务 + 日志**，首次拉基础镜像较慢）：
      `make docker-build DOCKER_IMAGE=ghcr.io/zhemed/litepan:v0.0.45`
- [x] 4.2 **验证门 G4a**：`docker images ghcr.io/zhemed/litepan` 有 `v0.0.45` 且**不是** `v0.0.44` 的 ID；
      `docker run --rm --entrypoint /app/litepan ghcr.io/zhemed/litepan:v0.0.45 --help` 可执行
- [x] 4.3 `docker tag` 补 `:0.0.45` 与 `:latest`
- [x] 4.4 `gh auth token | docker login ghcr.io -u zhemed --password-stdin`
- [x] 4.5 `docker push` ×3
- [x] 4.6 **验证门 G4b**：`gh api /users/zhemed/packages/container/litepan/versions --jq '.[0] | {tags: .metadata.container.tags, digest: .name}'`
      → 单条 version 同时含 `v0.0.45`/`0.0.45`/`latest`（= 同 digest）
- **回滚点 R4**：删该 GHCR version 的三个 tag（`0.0.44` 不动）

---

## Phase 5: D5 git tag + GitHub release

- [x] 5.1 确认 `main` 已是最新（`git log origin/main -1` = 版本提交）
- [x] 5.2 `git tag v0.0.45`
- [x] 5.3 `git push origin v0.0.45`
- [x] 5.4 `gh release create v0.0.45 --title "v0.0.45" --notes "<本轮说明>"`（说明须写明：完全移除 API 秘钥功能、迁移 0024、如何升级）
- [x] 5.5 **验证门 G5（0.0.39 教训的直接防御）**：
      - `gh api repos/zhemed/LitePan/git/ref/tags/v0.0.45 --jq .object.sha` == `git rev-parse v0.0.45`
      - `gh release view v0.0.45 --json tagName,name,isDraft,url` 存在且 `isDraft=false`
- **回滚点 R5**：`gh release delete v0.0.45 --cleanup-tag` + 删远端 tag

---

## Phase 6: D6 备份 → 部署本机 :5211 → 验收

- [x] 6.1 **部署前备份（SQLite 在线备份，禁止裸 cp）**：
      对 `data/litepan.db` 做一致性快照到 `data/backups/manual-pre-0024-<YYYYmmdd-HHMMSS>.db`
- [x] 6.2 **验证门 G6a（备份可用）**：只读打开该快照 → `schema_migrations` 最大 = 23、`api_keys` 表**存在**、`configs` 7 行
- [x] 6.3 更新实例：`docker compose pull litepan && docker compose up -d`
- [x] 6.4 **验证门 G6b（新版本生效）**：
      - `curl -s http://127.0.0.1:5211/api/health` → 200
      - `curl -s http://127.0.0.1:5211/api/public/system-config` → 含 `"version":"v0.0.45"`
      - `docker ps` 显示容器使用 `ghcr.io/zhemed/litepan:v0.0.45`
- [x] 6.5 **验证门 G6c（迁移真跑）**：只读打开实例库 → `schema_migrations` 最大 = **24**、`api_keys` 表**不存在**、`idx_api_keys_status` **不存在**、`configs` **仍 7 行**、表清单**恰好少 `api_keys` 一张**
- [x] 6.6 **验证门 G6d（部署形态不变）**：`docker inspect` 的 `Mounts/PortBindings/Devices/Privileged/PidMode/RestartPolicy/Env` 与 0.2 基线**逐项一致**
- [x] 6.7 **验证门 G6e（浏览器验收，真实例）**：
      - `bw open http://127.0.0.1:5211/login`（或直接 `bw open .../admin?page=settings` 复用会话）
      - 设置页页签**只有 3 个**（账号安全/首页设置/其他设置），**无「API 秘钥」**
      - 登录/会话仍正常；截图目视确认布局无缺口
- [x] 6.8 **验证门 G6f（日志无异常）**：`docker compose logs --since 5m litepan` 无 `level=ERROR`、无迁移报错
- **回滚点 R6**：compose 改回 `v0.0.44` → `docker compose up -d`；需要时从 6.1 快照恢复

---

## Phase 7: D7 归档收尾

- [x] 7.1 `skill trellis-check` 走查（lint/类型/测试、spec 同步、范围纪律、跨层一致性）
- [x] 7.2 `flow_gate.py mark-check release-0-0-45 --note "<质量门 + 发布 + 部署摘要>"`
- [x] 7.3 勾选 `prd.md` 全部验收项（每条必须实际执行过）
- [x] 7.4 `flow_gate.py pre-archive release-0-0-45`
- [x] 7.5 `task.py archive release-0-0-45 --skip-branch-validation`（auto-commit 会因路径搬迁失败，属已知 → 手工提交 `chore(task): archive ...`）
- [x] 7.6 `add_session.py --commit <work-hash>,<archive-hash>,<journal-hash>`（会话号以 journal 为准）
- [x] 7.7 `git push`（journal 提交）

---

## Validation Commands（汇总）

```bash
export PATH=/usr/local/go/bin:/root/go/bin:$PATH
cd /root/LitePan

# G1 版本号
grep -rn "v0\.0\.44" internal/ web/src README.md docker-compose*.yml
grep -rn "v0\.0\.45" internal/buildinfo README.md docker-compose*.yml
git diff --stat

# GQ 质量门
make lint && GOWORK=off go vet ./... && GOWORK=off go test ./...
cd web && npm run type-check && cd ..
git status --short internal/api/web    # 期望无输出

# G4 镜像
make docker-build DOCKER_IMAGE=ghcr.io/zhemed/litepan:v0.0.45
docker images ghcr.io/zhemed/litepan
gh api /users/zhemed/packages/container/litepan/versions --jq '.[0] | {tags: .metadata.container.tags, digest: .name}'

# G5 tag/release
gh api repos/zhemed/LitePan/git/ref/tags/v0.0.45 --jq .object.sha
git rev-parse v0.0.45
gh release view v0.0.45 --json tagName,isDraft,url

# G6 部署
docker compose pull litepan && docker compose up -d
curl -s http://127.0.0.1:5211/api/health
curl -s http://127.0.0.1:5211/api/public/system-config
docker inspect litepan --format '{{json .Mounts}}'
docker compose logs --since 5m litepan | grep -c level=ERROR
```

## Review Gates

| 门 | 位置 | 判据 |
|---|---|---|
| G1 | P1.3 | `v0.0.44` 零命中、`v0.0.45` 命中 5 处、diff 只含 4 文件且不含 embed |
| GQ | P2 | lint 0 issues、vet 0、test 全 ok、type-check 0、embed 零改动 |
| G3 | P3.3 | main 已推且工作区干净 |
| G4a/b | P4.2/4.6 | 镜像可构建可执行；GHCR 单条 version 三 tag 同 digest |
| G5 | P5.5 | 远端 tag sha == 本地；release 存在非草稿 |
| G6a–f | P6 | 备份可读；v0.0.45 生效；库 migration 24 且 api_keys 消失、configs 7 行；部署形态逐项一致；浏览器验收 3 页签无「API 秘钥」；日志无 ERROR |

## Rollback

见 design.md → **Rollback** 表。三件套：容器回 `v0.0.44`、数据回快照、对外删 tag/release/GHCR version；**`v0.0.44` 全程不动**。

## Out of Scope

- 上游同步、`1.0.0`、GHCR 旧版本清理
- `error-handling.md` 僵尸段落重写（上一任务登记的待办）
- 任何业务代码、驱动、前端源码改动

---

## 执行记录（实施后回填，仅记录事实与偏差）

**全部门禁通过**，且**每一步都是真实执行**（无跳过）：G1 命中 5 处 / `v0.0.44` 零残留 / diff 恰 4 文件；GQ lint 0 issues、vet 0、27 包 ok、type-check 0、embed 零改动；G3 推送后 `origin/main = a6c6c54`；G4a 镜像 `236392d8…`（≠ `v0.0.44` 的 `a864057d…`）且 `--help` 可执行；G4b GHCR 单条 version（id 1240890917）含三 tag 同 digest；G5 远端 tag sha `a6c6c540…` == 本地、release 非草稿且被标为 Latest；G6a–f 全通过（见下）。

**实际做到的事（关键数字）**

- 镜像内二进制校验：`grep -c 'v0.0.45' /app/litepan` = **1**、`grep -c 'v0.0.44'` = **0** —— 直接证明版本注入生效，而不是只改了源码字符串。
- 部署前备份：`data/backups/manual-pre-0024-20260912-143827.db`（229376 B，`PRAGMA integrity_check = ok`、version 23、`api_keys` 在、`configs` 7 行）。用 **SQLite 在线备份 API**（`Connection.backup`）而非 `cp` —— 实测库有 53 KB 未检查点的 `-wal`，裸 cp 会拿到过期快照。
- 实例库迁移：`schema_migrations` 23 → **24**、表数 **11 → 10**、`api_keys` 与 `idx_api_keys_status` 双双消失、`configs` **仍 7 行**。
- 部署形态：`docker inspect` 的 **12 项**（Mounts/PortBindings/Devices/Privileged/PidMode/RestartPolicy/SecurityOpt/NetworkMode/Env/Ports/Cmd/Entrypoint）与部署前**逐项一致**，只有镜像（`a864057d…` → `236392d8…`）与容器 ID 变。容器经 `docker compose`（project=litepan）更新，**没有**手搓 `docker run` —— 否则会丢掉 compose 网络标签，把部署形态悄悄改掉。
- 浏览器验收（真实例 `:5211`）：DOM 断言 `tabs = ["账号安全","首页设置","其他设置"]`、`hasApiKeyTab = false`；首页页脚渲染出 **「当前版本 LitePan v0.0.45」**（后端二进制版本 → API → UI 全链路在真实部署上闭环）；`window.__err = []`。
- 日志：`docker compose logs` 中 `level=ERROR` = **0 行**。

**偏差（1 处，计划外但属 trellis-check 的 Spec Sync 要求）**：改了两行 spec ——
① `api-layering.md` 的发版清单原文只写 `README.md` 与 `docker-compose.yml`，实测 tag 引用是 **3 文件 4 处**（含 `docker-compose.fnos.yml`），已补上并写明"须按 `docker-compose*.yml` 通配搜"；
② `dead-code-guide.md` 的基线只记了 `v0.0.44` 的 9/14，补成两轮对照（`v0.0.45` → 7/0）。
两处都是**同一个事实的更新**，未改任何规则语义。提交 `30e2010`（在镜像/发布之后，不影响已发产物）。

**未做永久回归测试的理由**：本轮零新增函数、零行为分支，唯一源码改动是一个版本字符串；该字符串的链路已有上一轮建立的 `internal/api/public_version_test.go` 锁定（"注入什么就回传什么"），本轮只是把值从 `v0.0.44` 换成 `v0.0.45`，再写一个断言新值的测试等于锁死一个会变的常量。

**回滚演练可行性（未实际执行）**：`v0.0.44` 的镜像、GHCR 三 tag、git tag、release 全程未动；`docker-compose.yml` 改回 `v0.0.44` + `docker compose up -d` 即可退回，本地仍有 `a864057d…` 镜像无需重新拉取。
