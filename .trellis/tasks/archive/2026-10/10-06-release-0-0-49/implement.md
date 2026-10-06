# Implementation Plan：发布 v0.0.49

## Overview

前置快照（P0）→ 版本号（P1）→ 质量门（P2）→ 推源码（P3）→ 镜像与 GHCR（P4）→ tag/Release（P5）
→ 更新容器（P6）→ 部署后验证（P7）→ 归档（P8）。不可逆动作全部在验证之后。

---

## Phase 0: 前置快照

- [x] 0.1 `git status --short`、`main == origin/main`、`v0.0.48 = 778f5b87`
- [x] 0.2 发布内容核对：`git log --merges v0.0.48..main` 为空；列全部提交并分类
- [x] 0.3 容器基线（`docker inspect litepan` → `/tmp/rel49/`，9 个非镜像字段）
- [x] 0.4 库基线（只读）：`schema_migrations`=25、表数=9、`configs`=7 行
- [x] 0.5 GHCR 现状：当前 tagged version 的 tags/digest

## Phase 1: 版本号 → v0.0.49

- [x] 1.1 5 处替换；**G1**：`v0.0.48` 源码/部署文件零命中、`v0.0.49` 恰 5 处、`git diff --stat` 只 4 文件

## Phase 2: 质量门

- [x] 2.1 `make lint` 0 / `go vet` exit=0 / `go test ./...` 全包 ok / `cd web && npm run type-check` exit=0
- [x] 2.2 **G2**：`git status --short internal/api/web web/src` 无输出

## Phase 3: 提交并推送

- [x] 3.1 `git add -A --` 4 文件 → 提交 `chore(release): 版本号推进到 v0.0.49（…）` → `git show --stat` 回读
- [x] 3.2 `git push origin main`；**G3** `origin/main` == 新提交

## Phase 4: 镜像

- [x] 4.1 `make docker-build DOCKER_IMAGE=ghcr.io/zhemed/litepan:v0.0.49`
- [x] 4.2 **G4a**：镜像内 `v0.0.49`=1、`v0.0.48`=0、`同步盘`≥1；镜像 ID ≠ v0.0.48 的 ID
- [x] 4.3 `docker tag` 补 `:0.0.49`/`:latest`；`gh auth token | docker login ghcr.io -u zhemed --password-stdin`；push ×3
- [x] 4.4 **G4b**：`gh api .../versions` 单条 version 含三 tag 同 digest；`v0.0.48` 仍指旧 digest

## Phase 5: tag + Release

- [x] 5.1 `git tag v0.0.49` → `git push origin v0.0.49` → `gh release create v0.0.49 --title … --notes-file /tmp/rel49/release-notes.md`
- [x] 5.2 **G5**：远端 tag sha == 本地；release 非草稿非预发布

## Phase 6: 更新容器

- [x] 6.1 `docker compose pull litepan && docker compose up -d`；**G6** 容器镜像为 v0.0.49，
      且容器镜像 ID == 本地构建 ID

## Phase 7: 部署后验证

- [x] 7.1 **G7a**：health 200、登录 200、`public/system-config.version` = `v0.0.49`
- [x] 7.2 **G7b**：`schema_migrations`=25、表数=9、`configs`=7 行
- [x] 7.3 **G7c**：9 个非镜像字段与 0.3 逐项一致（仅 Image/ImageName 变）
- [x] 7.4 **G7d**：0 ERROR；上传任务三端点 + 本地映射配置 + 日志接口 200；
      **A4 线上抽验**：`/api/logs` 非 ERROR 条目带 `details`
- [x] 7.5 **G7e**：浏览器 —— 页脚 `v0.0.49`、`window.__err` 空、日志页「复制日志」按钮存在

## Phase 8: 归档

- [x] 8.1 `mark-check` → 勾选验收 → `pre-archive` → `archive`（auto-commit 缺锚点 → 手工补交）
- [x] 8.2 `add_session.py --no-commit` + 手工提交；`git push`

## Validation Commands

```bash
export PATH=/usr/local/go/bin:/root/go/bin:$PATH GOWORK=off GOPROXY=https://goproxy.cn,direct
cd /root/LitePan
grep -rn "v0\.0\.48" internal/ web/src README.md docker-compose*.yml   # 应为空
make lint && go vet ./... && go test ./... && (cd web && npm run type-check)
make docker-build DOCKER_IMAGE=ghcr.io/zhemed/litepan:v0.0.49
docker run --rm --entrypoint /bin/sh ghcr.io/zhemed/litepan:v0.0.49 -c \
  'grep -c v0.0.49 /app/litepan; grep -c v0.0.48 /app/litepan; grep -c 同步盘 /app/litepan'
gh api /users/zhemed/packages/container/litepan/versions --jq '.[0] | {tags: .metadata.container.tags, digest: .name}'
git tag v0.0.49 && git push origin v0.0.49
gh api repos/zhemed/LitePan/git/ref/tags/v0.0.49 --jq .object.sha
docker compose pull litepan && docker compose up -d
curl -s -o /dev/null -w '%{http_code}\n' http://127.0.0.1:5211/api/health
python3 -c "import sqlite3;c=sqlite3.connect('file:data/litepan.db?mode=ro',uri=True);print(c.execute('SELECT MAX(version) FROM schema_migrations').fetchone()[0], c.execute(\"SELECT count(*) FROM sqlite_master WHERE type='table'\").fetchone()[0], c.execute('SELECT count(*) FROM configs').fetchone()[0])"
bw open http://127.0.0.1:5211/ --wait=3000
```

## Review Gates

| 门 | 判据 |
|---|---|
| G0 | 无 merge；提交集合与预期一致 |
| G1 | 版本串 5 处、旧串零命中、diff 恰 4 文件 |
| GQ | lint 0 / vet 0 / test 全 ok / type-check 0 / embed+前端零改动 |
| G4a/b | 镜像含 `v0.0.49` 不含 `v0.0.48` 且含 `同步盘`；GHCR 三 tag 同 digest |
| G5 | 远端 tag sha == 本地；release 非草稿 |
| G6/G7 | 容器 v0.0.49 + 库/形态未变 + 0 ERROR + 端点 200 + 浏览器页脚与日志页按钮 |

## Rollback

见 design.md → Rollback（compose 回 `v0.0.48` + `up -d`；对外删 release/tag/GHCR version；无数据动作）。

## Out of Scope

- 上游继续同步、2FA、GHCR 历史清理、任何新功能
- A2/A3 的真实环境验证（需真实账号；本轮以单测为判据，已在移植任务记录里说明）

## 执行记录（实施后回填）


---

## 执行记录（2026-10-06 实测）

| 门 | 结果 |
|---|---|
| G0 发布内容 | `v0.0.48..main` **7 提交、0 merge**；非 .trellis 改动 = 四项移植代码/测试 + embed(94) + 前端 2 文件 |
| G1 版本 | 5 处替换成功；`v0.0.48` 源码/部署文件 **零残留**、`v0.0.49` 恰 5 处、diff 恰 4 文件 |
| GQ 质量门 | `make lint` 0 issues、`go vet` exit=0、`go test ./...` 26 包 ok、`vue-tsc -b` exit=0；embed/前端本版零改动（G2 无输出） |
| G3 | 版本提交 `ba2f2fef`（恰 4 文件），推送后 `origin/main = ba2f2fef` |
| G4a 镜像 | ID `6ddd6e7973da` ≠ v0.0.48 的 `f43b75d491f6`；镜像内 `v0.0.49`=1、`v0.0.48`=0、`同步盘`=1（前端 `复制日志`=0，属 gz 后 embed 的预期） |
| G4b GHCR | 单条 version `["0.0.49","v0.0.49","latest"]` 同 digest `6ddd6e7973da`；`v0.0.48`/`v0.0.47` 仍指各自旧 digest |
| G5 tag/Release | tag `v0.0.49` 远端 sha == 本地 `ba2f2fefb52d…`；release 非草稿非预发布 |
| G6 部署 | 容器 `ghcr.io/zhemed/litepan:v0.0.49`；容器镜像 ID `6ddd6e7973da` == 本地构建 ID |
| G7a | health 200、登录 200、`public/system-config.version` = **v0.0.49** |
| G7b 库 | `migration=25`、`表数=9`、`configs=8` —— 与部署前**基线（25/9/8）逐项一致** |
| G7c 形态 | 只有 `Image`/`ImageName` 变化，其余 9 个非镜像字段全同 |
| G7d | `level=ERROR` = **0**；6 个端点全 200；**A4 线上抽验**：`/api/logs` 40 条中 33 条带 `details`，**全部为非 ERROR 级**（旧实现这些都不带） |
| G7e 浏览器 | 首页页脚 **LitePan v0.0.49**、`window.__err` = null；系统日志页 **43 个「复制日志」按钮 + 43 个详情块**渲染正常，截图 `/tmp/rel49/logs-v49.png` |

### 与上一版发布记录的差异说明（诚实口径）

- **`configs` 行数基线是 8 而不是 v0.0.48 记录的 7**：v0.0.48 部署后我做过一次"同值往返"设置写入
  （`log_retention_days=30`），该键此前不存在 → 插入了一行，7→8。本轮以**部署前实测的 8** 为基线并保持不变，
  不是本轮引入的变化。
- 本版**有用户可见的前端变化**（日志页详情 + 复制按钮），故浏览器验收比 v0.0.48 多验了日志页 DOM 断言。

### 未做/限制

- A2（多段 Range）与 A3（189 同步盘）的**运行时行为未端到端**（需可播放账号与 189 账号），以移植任务的单测为判据。
- 未清理 GHCR 历史版本与旧 tag/release（回滚退路）。
