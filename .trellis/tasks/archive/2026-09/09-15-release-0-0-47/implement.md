# Implementation Plan: 发布 v0.0.47

## Overview

顺序：**版本号（P1）→ 质量门（P2）→ 推源码（P3）→ 镜像（P4）→ tag/release（P5）→ 更新容器（P6）→ 验证（P7）→ 归档（P8）**。不可逆动作全部排在验证之后。

本机 `danger-full-access`、审批关闭 —— 不请求 escalation。

---

## Phase 0: 前置快照

- [ ] 0.1 `git status --short`（期望仅本任务目录）、`git log --oneline origin/main -1`
- [ ] 0.2 容器基线：`docker inspect litepan` 存 `/tmp/rel47/container-before.json` 并提取 11 个关键字段（Privileged/PidMode/Devices/Mounts/Binds/PortBindings/RestartPolicy/Env/NetworkMode/Image/ImageName）
- [ ] 0.3 库基线（只读）：`schema_migrations`=25、表数=9、`configs`=7 行
- [ ] 0.4 GHCR 现状：`.[0]` = `["0.0.46","v0.0.46","latest"]` + 其 digest
- [ ] 0.5 确认**无新迁移**（`ls internal/store/migrations | tail` → `0025` 仍为最新）

**回滚点 R0**：以上即后续比对依据

---

## Phase 1: D1 版本号 → v0.0.47

- [ ] 1.1 `internal/buildinfo/version.go`
- [ ] 1.2 `README.md` ×2、`docker-compose.yml`、`docker-compose.fnos.yml`
- [ ] 1.3 **验证门 G1**：`grep -rn "v0\.0\.46" internal/ web/src README.md docker-compose*.yml` 零命中；`v0.0.47` 命中 5 处；`git diff --stat` 只含 4 文件
- **回滚点 R1**：`git checkout HEAD -- internal/buildinfo/version.go README.md docker-compose.yml docker-compose.fnos.yml`

---

## Phase 2: D2 质量门

- [ ] 2.1 `make lint` → 0 issues
- [ ] 2.2 `GOWORK=off go vet ./...` → exit=0
- [ ] 2.3 `GOWORK=off go test ./...` → 全包 ok（期望 26）
- [ ] 2.4 `cd web && npm run type-check` → exit=0
- [ ] 2.5 **G2**：`git status --short internal/api/web` 无输出
- **回滚点 RQ**：不过则回 P1

---

## Phase 3: D3 提交并推送

- [ ] 3.1 提交（说明本轮发布内容 = 仪表盘三处假任务展示删除）
- [ ] 3.2 `git push origin main`
- [ ] 3.3 **G3**：工作区干净、`origin/main` = 本次提交
- **回滚点 R3**：`git revert`

---

## Phase 4: D4 镜像

- [ ] 4.1 **后台**构建：`make docker-build DOCKER_IMAGE=ghcr.io/zhemed/litepan:v0.0.47`
- [ ] 4.2 **G4a**：镜像存在且 ID ≠ v0.0.46；镜像内二进制 `grep -c v0.0.47` = 1、`v0.0.46` = 0；如镜像内可 grep 到嵌入资源则**额外**断言三处文案零命中（按 design KD2 的两种路径择一记录）
- [ ] 4.3 `docker tag` 补 `:0.0.47`、`:latest`
- [ ] 4.4 `gh auth token | docker login ghcr.io -u zhemed --password-stdin`
- [ ] 4.5 `docker push` ×3
- [ ] 4.6 **G4b**：GHCR 单条 version 含三 tag；`v0.0.46` 仍指向旧 digest
- **回滚点 R4**：删该 GHCR version

---

## Phase 5: D5 tag + release

- [ ] 5.1 `git tag v0.0.47` → `git push origin v0.0.47`
- [ ] 5.2 写 `/tmp/rel47/release-notes.md` → `gh release create v0.0.47 --title "v0.0.47" --notes-file …`
- [ ] 5.3 **G5**：远端 tag sha == 本地；release 非草稿
- **回滚点 R5**：`gh release delete v0.0.47 --cleanup-tag`

---

## Phase 6: D6 更新容器

- [ ] 6.1 `docker compose pull litepan`
- [ ] 6.2 `docker compose up -d`
- [ ] 6.3 **G6**：`docker ps` 显示 `v0.0.47`
- **回滚点 R6**：compose 回 `v0.0.46` → `docker compose up -d`（无需数据动作）

---

## Phase 7: D7 部署后验证

- [ ] 7.1 **G7a**：`/api/health` 200；登录 200；`system-config.version` = `v0.0.47`
- [ ] 7.2 **G7b（库未变）**：`schema_migrations` = 25、表数 = 9、`configs` = 7 行、`fuse_mounts` 仍不存在
- [ ] 7.3 **G7c（形态未变）**：`docker inspect` 11 字段与 0.2 逐项一致（只允许 Image/ImageName 变）
- [ ] 7.4 **G7d**：日志 `level=ERROR` = 0；上传任务 API 三端点 200；自动联动页可渲染
- [ ] 7.5 **G7e（核心：用户可见面）**：`bw` DOM 断言 —— hero 指标 **3**、`.overview-card` **2**、`.dashboard-panel` **1**、innerText 不含四组文案、`window.__err` 空；首页页脚 `v0.0.47`；截图目视布局无塌陷
- **回滚点 R7**：见 R6

---

## Phase 8: D8 归档

- [ ] 8.1 `skill trellis-check` 走查
- [ ] 8.2 `mark-check --note`（含"本轮无迁移故无备份"的判据说明）
- [ ] 8.3 勾选 `prd.md` 全部验收项
- [ ] 8.4 `pre-archive` → `task.py archive --skip-branch-validation`（auto-commit 已知会失败 → 手工提交）
- [ ] 8.5 `add_session.py` → `git push`

---

## Validation Commands（汇总）

```bash
export PATH=/usr/local/go/bin:/root/go/bin:$PATH GOWORK=off GOPROXY=https://goproxy.cn,direct
cd /root/LitePan

# G1
grep -rn "v0\.0\.46" internal/ web/src README.md docker-compose*.yml ; grep -rn "v0\.0\.47" internal/buildinfo README.md docker-compose*.yml

# GQ
make lint && go vet ./... && go test ./... && (cd web && npm run type-check)

# G4
make docker-build DOCKER_IMAGE=ghcr.io/zhemed/litepan:v0.0.47
docker run --rm --entrypoint /bin/sh ghcr.io/zhemed/litepan:v0.0.47 -c 'grep -c v0.0.47 /app/litepan; grep -c v0.0.46 /app/litepan'
gh api /users/zhemed/packages/container/litepan/versions --jq '.[0] | {tags: .metadata.container.tags, digest: .name}'

# G5
gh api repos/zhemed/LitePan/git/ref/tags/v0.0.47 --jq .object.sha ; git rev-parse v0.0.47

# G6/G7
docker compose pull litepan && docker compose up -d
curl -s -o /dev/null -w '%{http_code}\n' http://127.0.0.1:5211/api/health
docker inspect litepan --format '{{.HostConfig.Privileged}} {{.HostConfig.PidMode}} {{.HostConfig.Devices}} {{json .Mounts}}'
python3 -c "import sqlite3;c=sqlite3.connect('file:data/litepan.db?mode=ro',uri=True);print(c.execute('SELECT MAX(version) FROM schema_migrations').fetchone()[0])"
bw open http://127.0.0.1:5211/admin?page=dashboard\&tab=overview --wait=3000
bw eval "JSON.stringify({hero:[...document.querySelectorAll('.hero-metric')].length,cards:[...document.querySelectorAll('.overview-card')].length,panels:[...document.querySelectorAll('.dashboard-panel')].length})"
```

## Review Gates

| 门 | 位置 | 判据 |
|---|---|---|
| G1 | 1.3 | `v0.0.46` 零命中、`v0.0.47` 5 处、diff 只 4 文件 |
| GQ | 2 | lint 0 / vet 0 / test 全 ok / type-check 0 / embed 零改动 |
| G3 | 3.3 | main 已推、工作区干净 |
| G4a/b | 4.2/4.6 | 镜像含 v0.0.47 不含 v0.0.46；GHCR 三 tag 同 digest |
| G5 | 5.3 | 远端 tag sha == 本地；release 非草稿 |
| G6 | 6.3 | 容器跑 v0.0.47 |
| G7a–e | 7.1–7.5 | health/登录/版本；**库未变**；**形态未变**；日志 0 ERROR + 反例回归；**浏览器：3 指标 / 2 卡片 / 1 面板 / 无任务文案 / 无 JS 错误** |

## Rollback

见 design.md → **Rollback**：compose 回 `v0.0.46` + `up -d`；对外删 release/tag/GHCR version；**数据侧无需动作**（本轮不改库、不改权限面）。

## Out of Scope

- 上游同步、`1.0.0`、GHCR 旧版本清理
- `DashboardManagement.vue` 状态句与 `SystemSettings.vue` 帮助文案里的"任务"字样（PRD 已记为遗留观察）
- 恢复"任务"展示的替代方案（接 `GET /api/files/upload/tasks/summary`）—— 仅在 research.md 留档

---

## 执行记录（实施后回填，仅记录事实与偏差）

**全部门禁通过（逐条实测）**：G1（`v0.0.46` 源码/部署文件零残留、`v0.0.47` 命中 5 处、diff 恰 4 文件）；GQ（lint 0 / vet 0 / 26 包 ok / type-check 0 / embed 零改动）；G3（`origin/main = 1c23251`）；G4a（镜像 `2ae2b2265243` ≠ v0.0.46 的 `31b122a678d5`；**镜像内二进制** `v0.0.47`=1、`v0.0.46`=0）；G4b（GHCR 单条 version 含三 tag 同 digest `2ae2b2265…`，`v0.0.46` 仍指向旧 digest）；G5（远端 tag sha `1c23251b…` == 本地、release 非草稿）；G6（容器 `v0.0.47`）；G7a–e 见下。

**关于 design KD2 的镜像内嵌入资源验证**：实测容器内 `grep -c '暂无后台任务' /app/litepan` = **0**，但这条**不能作为判据** —— 前端资源是 `.gz` 压缩后 `go:embed` 进二进制的，明文字符串本就 grep 不到（同一条命令在 v0.0.46 镜像上也会是 0）。因此最终判据落在 **G7e 的部署后浏览器实测**（真实渲染的 DOM 里三处文案消失），这也是 design KD2 预留的 B 路径。

**G7b（库未变）**：`migration=25`、表数 **9**、`configs` **7 行**、`fuse_mounts` 仍不存在 —— 与部署前逐项一致，符合"本轮无迁移"的预期。
**G7c（形态未变）**：`docker inspect` 的 9 个非镜像字段（Mounts/Privileged/PidMode/Devices/Binds/PortBindings/RestartPolicy/Env/NetworkMode）**全部不变**，只有 `Image`/`ImageName` 从 `v0.0.46` 变为 `v0.0.47`。

**G7e（本轮核心：用户可见面）** 真实例 `:5211` 的 DOM 断言：

```json
{"hero":["0=接入","0=在线","0=待确认错误"],"heroCount":3,
 "cards":["0 B=缓存空间=清理","0=未读通知"],"cardCount":2,
 "sidePanels":["日志与通知"],"allPanels":2,
 "hasTaskText":false,"jsErr":null}
```

- 状态行 **3** 项、概况卡片 **2** 张、**右列（`.dashboard-side`）1 个面板**；`.dashboard-panel` 总数 2 是因为左列的「存储账号」**同样使用该类名**（PRD 写"右列 1 个面板"，此处按右列口径核对）
- 页面 innerText 不含 `运行任务`/`任务总数`/`后台任务`/`暂无后台任务`
- 首页页脚显示 **`v0.0.47`**；`window.__err` 为 null
- 截图（`/tmp/rel47/dashboard-v47.png`）目视确认：卡片行左对齐无空洞、右列无塌陷、状态行三指标均匀分布

**反例回归**：上传任务 API 三端点（`tasks`/`tasks/summary`/`runtime`）均 **200**；日志 `level=ERROR` = **0**。

### 偏差（1 处，无实质影响）

`docker compose` 重建后容器 ID 变化（属预期）；无其它偏差。**本轮未做部署前备份**，依据是"无新迁移、库 schema 不变"，且已在执行记录里用 G7b 的三项实测反证库确实未变 —— 这条判据写在 PRD Background 与 design KD1 中，不是事后解释。

### 未做永久回归测试的理由

本轮仅改 5 处版本字符串，无行为代码；前端展示的回归由 **D7 的浏览器 DOM 断言**覆盖（该断言同时锁定了"三处文案必须消失"与"保留项必须仍在"）。
