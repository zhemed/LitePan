# Implementation Plan: 版本号单一来源 + 发布 v0.0.44

## Overview

顺序：**先做代码与版本号（D1–D4）→ 质量门 → 重建 embed 并提交推送（D5）→ 构建发布（D6）→ 临时容器验证（D7）→ 归档**。

发布是不可逆对外动作，因此把**所有验证都放在 push 之前**：代码质量门不绿不进 D5，镜像未构建成功不推 GHCR，tag 未推送不建 release。

本机 `danger-full-access` 且审批关闭 —— **不请求任何 escalation**。

---

## Phase 0: 前置快照

- [ ] 0.1 `git status --short` 记录基线（期望：仅本任务目录）
- [ ] 0.2 记录当前远端状态：`git fetch origin && git log origin/main -1 --oneline`、`gh api /users/zhemed/packages/container/litepan/versions --jq '.[0]'`（当前 `latest` 指向哪个 digest）
- [ ] 0.3 记录 `5211` 空闲、`18081/3080/3081` 无冲突：`ss -ltnp | grep -E ':(5211|5212|3080|3081)\b'`

**回滚点 R0**：基线写入本任务记录；`latest` 原 digest 记下（回滚时可指回）

---

## Phase 1: D1 后端成为唯一来源

- [ ] 1.1 `internal/api/router.go`：`Deps` 加 `Version string`；`Handler` 加 `version string`；`NewRouter` 赋值 `version: d.Version`
- [ ] 1.2 `internal/api/public.go`：`publicSystemConfig` 响应加 `"version": h.version`（**保留**既有 3 个字段）
- [ ] 1.3 `internal/app/wire_http.go`：`api.Deps{... Version: buildinfo.Version ...}`
- [ ] 1.4 新增 `internal/api/public_version_test.go`：直接构造 `Handler{version: "vX.Y.Z-test"}` 调 `publicSystemConfig`，断言响应 JSON 含该版本且 3 个既有字段仍在
- [ ] 1.5 **验证门 G1**：`GOWORK=off go vet ./...` 通过；`GOWORK=off go test ./internal/api/` 通过（含新测试）
- **回滚点 R1**：`git checkout -- internal/api/router.go internal/api/public.go internal/app/wire_http.go` + 删新测试文件

---

## Phase 2: D2+D3 前端运行期读取

- [ ] 2.1 `web/src/api/public.ts`：`PublicSystemConfig` 加 `version: string`
- [ ] 2.2 新增 `web/src/stores/appInfo.ts`：
      ```ts
      export const useAppInfoStore = defineStore("appInfo", () => {
        const version = ref("");
        let inflight: Promise<void> | null = null;
        async function load() {
          if (version.value) return;
          if (inflight) return inflight;
          inflight = (async () => {
            try { version.value = (await publicApi.systemConfig()).version ?? ""; }
            finally { inflight = null; }
          })();
          return inflight;
        }
        return { version, load };
      });
      ```
- [ ] 2.3 `web/src/components/layout/AppFooter.vue`：删 `APP_VERSION_BADGE` 导入；`onMounted` 调 `appInfo.load()`；徽标 `value` 改为 `APP_NAME` + 版本（版本为空只显示 `APP_NAME`）
- [ ] 2.4 `web/src/components/admin/AdminAccountChip.vue`：`APP_VERSION` → `appInfo.version`，`onMounted` 触发 `load()`；空值时不渲染版本行
- [ ] 2.5 `web/src/version.ts`：删 `APP_VERSION` 与 `APP_VERSION_BADGE`；`GITHUB_URL` → `https://github.com/zhemed/LitePan`
- [ ] 2.6 **验证门 G2**：
      - `grep -rn "APP_VERSION" web/src` **零命中**
      - `grep -rn "v0\.5\.2" web/src internal/` **零命中**
      - `cd web && npm run type-check` 通过
- **回滚点 R2**：`git checkout -- web/src` + 删 `stores/appInfo.ts`

---

## Phase 3: D4 版本号升到 v0.0.44

- [ ] 3.1 `internal/buildinfo/version.go`：`v0.5.2-Beta` → `v0.0.44`
- [ ] 3.2 `README.md` 2 处、`docker-compose.yml` 1 处：`v0.0.43` → `v0.0.44`
- [ ] 3.3 **验证门 G4**：`grep -rn "v0\.0\.43"`（排除 node_modules/归档/workspace）零命中；`grep -rn "v0\.0\.44"` 命中 4 处（version.go + README×2 + compose×1）
- **回滚点 R4**：`git checkout -- internal/buildinfo/version.go README.md docker-compose.yml`

---

## Phase 4: 质量门（发布前最后一道）

- [ ] 4.1 `cd /root/LitePan && make lint` → `0 issues.`
- [ ] 4.2 `GOWORK=off go vet ./...` → exit=0
- [ ] 4.3 `GOWORK=off go test ./...` → 全包 `ok`
- [ ] 4.4 `cd web && npm run type-check` → exit=0
- **回滚点 RQ**：任一不过则回到对应 Phase 修，**不得进入 Phase 5**

---

## Phase 5: D5 重建 embed → 提交 → push main

- [ ] 5.1 `cd web && npm run build`（会写 `../internal/api/web`，135 个受跟踪产物）
- [ ] 5.2 **验证门 G5a**：`git status --short internal/api/web | wc -l` > 0（确有更新）；`ls internal/api/web/index.html` 存在
- [ ] 5.3 **验证门 G5b（关键）**：构建产物中**不含**上游版本号 —— `grep -rl "v0.5.2\|Beta" internal/api/web/ 2>/dev/null | wc -l` = 0（证明硬编码已被移除，而不是只改了源码）
- [ ] 5.4 `git add -A` 并提交（message 说明 D1–D5）
- [ ] 5.5 `git push origin main`
- [ ] 5.6 **验证门 G5c**：`git status -sb` 与 origin 同步；`git log origin/main -1 --oneline` = 本次提交
- **回滚点 R5**：`git revert <commit> && git push`（embed 大 diff 一并回滚）

---

## Phase 6: D6 发布 v0.0.44

- [ ] 6.1 构建镜像（走项目自己的入口）：`make docker-build DOCKER_IMAGE=ghcr.io/zhemed/litepan:v0.0.44`
      → 需拉 `node:20-bookworm-slim` / `golang:1.26.6-bookworm` / `debian:bookworm-slim`，**耗时较长**，以后台任务 + 日志运行
- [ ] 6.2 **验证门 G6a**：`docker run --rm --entrypoint /app/litepan ghcr.io/zhemed/litepan:v0.0.44 --help` 可执行（或 `docker inspect` 镜像存在且架构为 amd64/linux）
- [ ] 6.3 本地补 tag：`docker tag ghcr.io/zhemed/litepan:v0.0.44 ghcr.io/zhemed/litepan:0.0.44`、`...:latest`
- [ ] 6.4 登录：`gh auth token | docker login ghcr.io -u zhemed --password-stdin`
- [ ] 6.5 推送 3 tag（`docker push` ×3）
- [ ] 6.6 **验证门 G6b**：`gh api /users/zhemed/packages/container/litepan/versions --jq '.[0].metadata.container.tags'` 返回含全部三个 tag → 单条 version = 单 digest，**即三 tag 同 digest**
- [ ] 6.7 **严格顺序发 tag 与 release**：
      ① `git push origin main`（确认已是最新）
      ② `git tag v0.0.44`
      ③ `git push origin v0.0.44`
      ④ `gh release create v0.0.44 --title "v0.0.44" --notes "<说明>"`
- [ ] 6.8 **验证门 G6c**：`gh api repos/zhemed/LitePan/git/ref/tags/v0.0.44 --jq '.object.sha'` 与本地 `git rev-parse v0.0.44` **一致**（0.0.39 教训的直接防御）；`gh release view v0.0.44` 存在

**回滚点 R6**：见 design.md Rollback 表（删 GHCR 版本 / 删 tag / 删 release）

---

## Phase 7: D7 临时容器验证（不构成部署）

- [ ] 7.1 起临时容器（**端口 5212、数据目录 `/tmp/litepan-verify`**，不碰 5211、不在仓库建 `data/`）：
      `docker run -d --name litepan-verify -p 127.0.0.1:5212:5211 -v /tmp/litepan-verify:/app/data --device /dev/fuse --privileged --pid host ghcr.io/zhemed/litepan:v0.0.44`
- [ ] 7.2 **验证门 G7（本任务的核心证明）**：
      - `curl -s http://127.0.0.1:5212/api/health` → 200
      - `curl -s http://127.0.0.1:5212/api/public/system-config` → 含 `"version":"v0.0.44"`
      - `curl -s -X POST http://127.0.0.1:5212/api/auth/login -d 'username=admin&password=123456' -c /tmp/litepan-cookie` → 成功（**表单编码**，不得发 JSON）
- [ ] 7.3 清理：`docker rm -f litepan-verify` + `rm -rf /tmp/litepan-verify /tmp/litepan-cookie`
- [ ] 7.4 **验证门 G7b**：`ss -ltnp | grep 5211` **无输出**（部署仍搁置）；`git status --short` 无 `data/`/`mounts/` 新增
- **回滚点 R7**：删容器与临时目录即可

---

## Phase 8: 收尾归档

- [ ] 8.1 `skill trellis-check` 走查（lint/类型/测试、测试覆盖、spec 同步、范围纪律、跨层一致性）
- [ ] 8.2 `flow_gate.py mark-check 09-12-version-single-source-release-0-0-44 --note "<质量门+发版摘要>"`
- [ ] 8.3 勾选 `prd.md` 全部验收项（每条必须实际执行过）
- [ ] 8.4 `flow_gate.py pre-archive` 通过
- [ ] 8.5 `task.py archive ... --skip-branch-validation`（archive 的 auto-commit 会因路径搬迁失败，属已知 → 手工提交）
- [ ] 8.6 `add_session.py --commit <work-hash>`（会话号以 journal 为准）
- [ ] 8.7 `git push`（journal 提交）

---

## Validation Commands（汇总）

```bash
# D1
grep -n "Version" internal/api/router.go internal/api/public.go internal/app/wire_http.go
GOWORK=off go test ./internal/api/ -run Version -v
# D2/D3
grep -rn "APP_VERSION" web/src
grep -rn "v0\.5\.2" web/src internal/
grep -n "GITHUB_URL" web/src/version.ts
# D4
grep -rn "v0\.0\.4[34]" --include=*.go --include=*.md --include=*.yml . | grep -v node_modules | grep -v "/tasks/" | grep -v "/workspace/"
# 质量门
make lint && GOWORK=off go vet ./... && GOWORK=off go test ./...
cd web && npm run type-check && npm run build
# 发版
gh api /users/zhemed/packages/container/litepan/versions --jq '.[0]'
gh api repos/zhemed/LitePan/git/refs/tags/v0.0.44 --jq '.object.sha'
# 验证容器
curl -s http://127.0.0.1:5212/api/health
curl -s http://127.0.0.1:5212/api/public/system-config
```

## Review Gates

| 门 | 位置 | 判据 |
|---|---|---|
| G1 | Phase 1.5 | vet 通过 + `go test ./internal/api/` 通过（含新版本测试） |
| G2 | Phase 2.6 | `APP_VERSION` 零命中 + `v0.5.2` 零命中 + type-check 通过 |
| G4 | Phase 3.3 | `v0.0.43` 零命中、`v0.0.44` 命中 4 处 |
| GQ | Phase 4 | lint/vet/test/type-check 全绿 |
| G5a/b/c | Phase 5 | embed 确有更新 + 产物内不含上游版本号 + 已 push 同步 |
| G6a/b/c | Phase 6 | 镜像可构建 + GHCR 三 tag 同 digest + 远端 tag 指向与本地一致 |
| G7/G7b | Phase 7 | 容器三连通过 + 5211 仍空闲、仓库无 `data/` |

## Out of Scope

- 正式部署 LitePan 到 `:5211`（用户明确搁置）
- 方案 A 的构建期注入（`Dockerfile ARG` + `-ldflags` + Vite define）
- 任何业务逻辑、驱动、上传、缓存、鉴权改动
