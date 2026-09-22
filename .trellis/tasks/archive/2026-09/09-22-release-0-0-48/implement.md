# Implementation Plan: 发布 v0.0.48

## Overview

顺序：**前置快照（P0）→ 版本号（P1）→ 质量门（P2）→ 推源码（P3）→ 镜像（P4）→ tag/release（P5）
→ 更新容器（P6）→ 部署后验证（P7）→ 归档（P8）**。不可逆动作全部排在验证之后。

网络动作仅限 `ghcr.io` 与 `github.com`（本轮已获用户授权）；本机 `danger-full-access`、审批关闭 —— 不请求 escalation。

---

## Phase 0: 前置快照

- [x] 0.1 `git status --short`（期望仅任务目录）、`git rev-parse --short main origin/main v0.0.47`
- [x] 0.2 发布内容核对（design KD3 表）：无 merge、6 提交、20 文件、无越界、无新迁移
- [x] 0.3 容器基线：`docker inspect litepan` → `/tmp/rel48/container-before.json`，提取 9 个非镜像字段
- [x] 0.4 库基线（只读）：`schema_migrations`=25、表数=9、`configs`=7 行
- [x] 0.5 GHCR 现状：当前 version 的 tags（应为 `v0.0.47`/`0.0.47`/`latest`）+ digest

**回滚点 R0**：以上即后续比对依据

---

## Phase 1: D1 版本号 → v0.0.48

- [x] 1.1 `internal/buildinfo/version.go`
- [x] 1.2 `README.md`×2、`docker-compose.yml`、`docker-compose.fnos.yml`
- [x] 1.3 **G1**：`grep -rn "v0\.0\.47"` 在源码/部署文件零命中；`v0.0.48` 恰 5 处；`git diff --stat` 只含 4 文件
- **回滚点 R1**：`git checkout HEAD -- <4 文件>`

---

## Phase 2: D2 质量门

- [x] 2.1 `make lint` → 0 issues
- [x] 2.2 `GOWORK=off go vet ./...` → exit=0
- [x] 2.3 `GOWORK=off go test ./...` → 全包 ok（期望 26）
- [x] 2.4 `cd web && npm run type-check` → exit=0
- [x] 2.5 **G2**：`git status --short internal/api/web web/src` 无输出（embed 与前端零改动）

---

## Phase 3: D3 提交并推送

- [x] 3.1 `git add -A -- <4 文件>` → 提交 `chore(release): 版本号推进到 v0.0.48（上游三批移植上线）`；`git show --stat` 回读恰 4 文件
- [x] 3.2 `git push origin main`
- [x] 3.3 **G3**：工作区干净（仅任务目录）、`origin/main` = 本次提交
- **回滚点 R3**：`git revert <版本提交>`

---

## Phase 4: D4 镜像

- [x] 4.1 后台构建：`make docker-build DOCKER_IMAGE=ghcr.io/zhemed/litepan:v0.0.48`
- [x] 4.2 **G4a（KD2 双证据）**：镜像内 `/app/litepan`：`v0.0.48` ≥1、`v0.0.47` = 0；`启动失败`/`OAuth 转发重试均失败`/`映射目录` 各 ≥1
- [x] 4.3 `docker tag` 补 `:0.0.48`、`:latest`
- [x] 4.4 `gh auth token | docker login ghcr.io -u zhemed --password-stdin`
- [x] 4.5 `docker push` ×3
- [x] 4.6 **G4b**：`gh api` 读回单个 version 含三 tag 同 digest；`v0.0.47` 仍指向旧 digest
- **回滚点 R4**：删该 GHCR version

---

## Phase 5: D5 tag + release

- [x] 5.1 `git tag v0.0.48` → `git push origin v0.0.48`
- [x] 5.2 写 `/tmp/rel48/release-notes.md`（无内网信息）→ `gh release create v0.0.48 --title "v0.0.48" --notes-file …`
- [x] 5.3 **G5**：`gh api repos/zhemed/LitePan/git/ref/tags/v0.0.48 --jq .object.sha` == `git rev-parse v0.0.48`；release 非草稿
- **回滚点 R5**：`gh release delete v0.0.48 --cleanup-tag`

---

## Phase 6: D6 更新容器

- [x] 6.1 `docker compose pull litepan`
- [x] 6.2 `docker compose up -d`
- [x] 6.3 **G6**：`docker ps` 显示 `v0.0.48`
- **回滚点 R6**：compose 回 `v0.0.47` → `docker compose up -d`（无数据动作）

---

## Phase 7: D7 部署后验证

- [x] 7.1 **G7a**：health 200；登录 200；`system-config.version` = `v0.0.48`
- [x] 7.2 **G7b（库未变）**：`schema_migrations`=25、表数=9、`configs`=7 行
- [x] 7.3 **G7c（形态未变）**：9 个非镜像字段与 0.3 逐项一致（只允许 Image/ImageName 变）
- [x] 7.4 **G7d**：日志 `level=ERROR` = 0（WARN 逐条判读）；上传任务 API 三端点 200；本地映射配置 200
- [x] 7.5 **G7e（浏览器）**：`bw` 打开 `:5211` 首页 + 仪表盘 → 正常渲染、页脚 `v0.0.48`、`window.__err` 空；截图目视
- [x] 7.6 **G7f（B1 回归）**：改一次设置后回读（非独占键不陈旧）+ 重新登录仍 200
- **回滚点 R7**：见 R6

---

## Phase 8: D8 归档

- [x] 8.1 `skill trellis-check` 走查
- [x] 8.2 `mark-check`（含"本轮无迁移故无备份"的判据说明）
- [x] 8.3 勾选 `prd.md` 全部验收项
- [x] 8.4 `pre-archive` → `task.py archive --skip-branch-validation`（auto-commit 若缺锚点则手工补）
- [x] 8.5 `add_session.py` → `git push`

---

## Validation Commands（汇总）

```bash
export PATH=/usr/local/go/bin:/root/go/bin:$PATH GOWORK=off GOPROXY=https://goproxy.cn,direct
cd /root/LitePan

# 发布内容核对（P0 / design KD3）
git log --merges --oneline v0.0.47..main
git log --oneline v0.0.47..main
git diff --name-only v0.0.47..main -- . ':!.trellis'
ls internal/store/migrations | tail -1

# G1
grep -rn "v0\.0\.47" internal/ web/src README.md docker-compose*.yml
grep -rn "v0\.0\.48" internal/buildinfo README.md docker-compose*.yml

# GQ
make lint && go vet ./... && go test ./... && (cd web && npm run type-check)

# G4
make docker-build DOCKER_IMAGE=ghcr.io/zhemed/litepan:v0.0.48
docker run --rm --entrypoint /bin/sh ghcr.io/zhemed/litepan:v0.0.48 -c \
  'grep -c v0.0.48 /app/litepan; grep -c v0.0.47 /app/litepan; grep -c 启动失败 /app/litepan; grep -c "OAuth 转发重试均失败" /app/litepan; grep -c 映射目录 /app/litepan'
gh api /users/zhemed/packages/container/litepan/versions --jq '.[0] | {tags: .metadata.container.tags, digest: .name}'

# G5
gh api repos/zhemed/LitePan/git/ref/tags/v0.0.48 --jq .object.sha ; git rev-parse v0.0.48

# G6/G7
docker compose pull litepan && docker compose up -d
curl -s -o /dev/null -w '%{http_code}\n' http://127.0.0.1:5211/api/health
docker inspect litepan --format '{{.Config.Image}} {{.HostConfig.Privileged}} {{.HostConfig.PidMode}} {{.HostConfig.Devices}} {{json .Mounts}}'
python3 -c "import sqlite3;c=sqlite3.connect('file:data/litepan.db?mode=ro',uri=True);print(c.execute('SELECT MAX(version) FROM schema_migrations').fetchone()[0], c.execute(\"SELECT count(*) FROM sqlite_master WHERE type='table'\").fetchone()[0], c.execute('SELECT count(*) FROM configs').fetchone()[0])"
bw open http://127.0.0.1:5211/ --wait=3000
bw eval "JSON.stringify({ver:(document.body.innerText.match(/v0\\.0\\.\\d+/)||['none'])[0], err:window.__err||null})"
```

## Review Gates

| 门 | 位置 | 判据 |
|---|---|---|
| G0 | P0 | 无 merge、6 提交、20 文件、无越界、无新迁移 |
| G1 | 1.3 | `v0.0.47` 零命中、`v0.0.48` 5 处、diff 只 4 文件 |
| GQ | 2 | lint 0 / vet 0 / test 全 ok / type-check 0 / embed 与 web/src 零改动 |
| G3 | 3.3 | main 已推、工作区干净、提交恰 4 文件 |
| G4a/b | 4.2/4.6 | 镜像含 `v0.0.48` 不含 `v0.0.47` 且含本轮文案；GHCR 三 tag 同 digest |
| G5 | 5.3 | 远端 tag sha == 本地；release 非草稿 |
| G6 | 6.3 | 容器跑 `v0.0.48` |
| G7a–f | 7.1–7.6 | 版本/登录；库未变；形态未变；0 ERROR + 反例回归；浏览器无 JS 错误且页脚新版本；B1 写读一致 |

## Rollback

见 design.md → **Rollback**（compose 回 `v0.0.47` + `up -d`；对外删 release/tag/GHCR version；无数据动作）。

## Out of Scope

- 上游继续同步、「高级定时」能力、GHCR 历史清理
- 115 清单模式的接线（该路径当前无调用方，接线需另开任务并用真实账号核对 `Count` 口径）
- 任何前端改动（本轮零前端改动，故不发前端相关说明）

## 执行记录（实施后回填）

---

## 执行记录（2026-09-21/22 实测）

### G0 发布内容核对（"别合并错"的机械核对）

| 检查 | 结果 |
|---|---|
| `git log --merges v0.0.47..main` | **空**（无 merge 提交） |
| `v0.0.47..main` 提交数 | 10（3 个代码/文档 + 7 个任务归档与 journal 记账） |
| 非 `.trellis` 改动文件数 | **恰 20**，全部在移植任务 File Map 内 |
| 越界文件（`web/src`/`go.mod`/`go.sum`/`.golangci.yml`/`Makefile`/`internal/api/web`） | 全 0 |
| `internal/**` 除 `version.go` 的意外文件 | 0 |
| 迁移最新 | `0025_drop_fuse.sql`（**无新迁移**） |
| `main` / `origin/main` | 一致（发布前 `d37945c5`） |

**修正一处口径**：初稿把 `v0.0.47..main` 写成 6 个提交，实测为 **10** 个（含上一轮调研任务的 3 个归档/journal 与 v0.0.47 发布任务的 1 个归档提交）；PRD Background 已按实测改正。

### 各门结果

- **G1**：源码/部署文件里 `v0.0.47` **零残留**、`v0.0.48` **恰 5 处**、`git diff --stat` **恰 4 文件**。
- **GQ**：`make lint` 0 issues、`go vet` exit=0、`go test ./...` **26 包全 ok 0 FAIL**、`vue-tsc -b` exit=0；
  `git status internal/api/web web/src` 无输出（embed 与前端零改动）。
- **G2/G3**：版本提交 `778f5b87`（`git show --stat` 回读**恰 4 文件**）；推 `main` 后 `origin/main = 778f5b87`。
- **G4a（KD2 双证据）**：镜像 `f43b75d491f6` ≠ v0.0.47 的 `2ae2b2265243`；镜像内 `/app/litepan`：
  `v0.0.48`=1、`v0.0.47`=**0**，且本轮新增文案 `启动失败`=1、`OAuth 转发重试均失败`=1、`映射目录`=5、
  `后台概况接口响应较慢`=1、`后台概况接口已恢复正常`=1。
  （附带校正：常量名 `slowDashboardLogInterval` 在二进制里是 **0** —— 编译后不保留标识符，只能验字符串，
  故该项不作为判据。）
- **G4b**：GHCR 单条 version `["0.0.48","v0.0.48","latest"]` 同 digest `f43b75d491f6`；
  `v0.0.47`/`0.0.47` 仍指向旧 digest `2ae2b2265243`（回滚退路完好）；历史版本未被清理。
- **G5**：`git tag v0.0.48` 指向版本提交；远端 sha `778f5b87704b…` == 本地；
  release 非草稿、非预发布（`https://github.com/zhemed/LitePan/releases/tag/v0.0.48`）。
- **G6**：`docker compose pull litepan && docker compose up -d` → 容器 `ghcr.io/zhemed/litepan:v0.0.48`。
  **额外一致性证据**：容器使用的镜像 ID `f43b75d491f6` == 本地构建 ID == GHCR manifest digest 前缀。
- **G7a**：health 200、登录 200、`public/system-config` 的 `version` = **`v0.0.48`**。
- **G7b（库未变）**：`schema_migrations`=25、表数=9、`configs`=7 行、`fuse_mounts` 不存在 —— 与部署前逐项一致。
- **G7c（形态未变）**：9 个非镜像字段（Mounts/Binds/Privileged/PidMode/Devices/PortBindings/RestartPolicy/
  Env/NetworkMode）与部署前逐项一致，只有 `Image`/`ImageName` 由 v0.0.47 变为 v0.0.48。
- **G7d**：`docker logs` 与 `data/log/2026-09-22.log` 的 `level=ERROR` 均为 **0**；端点回归 8 项全 200
  （upload tasks / summary / runtime、本地映射配置、admin settings / system-config、
  notifications/unread-count、logs/stats）。
- **G7e（浏览器，真实例 `:5211`）**：首页页脚 **`LitePan v0.0.48`**、`window.__err` = null；
  管理后台仪表盘与 v0.0.47 验收一致（`.hero-metric`=3、`.overview-card`=2、右列 `.dashboard-panel`=1），
  截图 `/tmp/rel48/dashboard-v48.png` 目视无塌陷、无错误横幅（`window.__err` 空）。
- **G7f（B1 回归）**：`PUT /api/admin/settings {"log_retention_days":"30"}`（**同值往返，不改动线上配置语义**）
  → 200，回读仍 30（非独占键不陈旧）；重新登录 200（凭据走缓存路径）。
  写路径的完整验证在移植任务的**数据副本**临时实例上做过（见该任务 implement.md），本机线上不做破坏性写入。

### 本轮未做部署前备份的判据（写在 PRD Background，非事后解释）

迁移最新仍是 `0025`、实例库已在 25、本轮改动无任何迁移文件 → 镜像更新不执行迁移。
G7b 的三项实测（25 / 9 / 7）即该判据的反证。

### 偏差

1. **口径修正**：PRD 初稿的"6 个提交"实测为 10（已改正，见上）。
2. **G4a 探针校正**：常量名不入二进制，改用文案字符串作证据（已在上面记录）。
3. 发布说明二次修订：把"删除僵尸键"改为"**不再写入**该键，备份/恢复的历史键清洗名单保留该键名"，
   与实际实现（保留清洗名单，与上游同口径）一致。
4. 无其它偏差；`web/src`、`internal/**`（除 `version.go`）、`Dockerfile`、`Makefile`、`.golangci.yml`、
   `go.mod`/`go.sum`、`data/**` 本轮均零改动。
