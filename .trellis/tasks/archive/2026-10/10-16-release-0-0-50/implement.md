# Implementation Plan：v0.0.50

## Phase 1: 版本号
- [x] 1.1 5 处替换；**G1** 旧串零残留、新串恰 5 处、diff 恰 4 文件

## Phase 2: 质量门
- [x] 2.1 lint / vet / test / type-check；**G2** `git status --short internal/api/web web/src` 无输出

## Phase 3: 提交与推送
- [x] 3.1 版本提交（锚点本任务）→ `git show --stat` 回读
- [x] 3.2 `git push origin main`（一并推送此前 4 个提交）；**G3** `origin/main` == 本地

## Phase 4: 镜像与 GHCR
- [x] 4.1 `make docker-build DOCKER_IMAGE=ghcr.io/zhemed/litepan:v0.0.50`
- [x] 4.2 **G4a** 镜像内：`v0.0.50`=1、`v0.0.49`=0、`HeyTapBrowser`≥1；镜像 ID ≠ v0.0.49 的 ID
- [x] 4.3 tag 补 `:0.0.50`/`:latest` → login → push ×3
- [x] 4.4 **G4b** GHCR 单条 version 三 tag 同 digest；v0.0.49 旧 digest 未动

## Phase 5: tag + Release
- [x] 5.1 `git tag v0.0.50` → push → `gh release create`（notes 不含内网信息）
- [x] 5.2 **G5** 远端 tag sha == 本地；release 非草稿

## Phase 6: 部署到飞牛生产机
- [x] 6.1 基线：容器 inspect（非镜像字段）+ 库计数（`schema_migrations`/表数/`configs`）
- [x] 6.2 `docker compose pull && docker compose up -d`（不手搓 `docker run`）
- [x] 6.3 **G6** 容器镜像 = v0.0.50，容器镜像 ID == 本地构建 ID

## Phase 7: 部署后验证
- [x] 7.1 **G7a** health 200 / 登录 200 / `public/system-config.version` = `v0.0.50`
- [x] 7.2 **G7b** 库计数与 6.1 一致；**G7c** 容器非镜像字段未变
- [x] 7.3 **G7d 服务端两跳模拟**：安卓浏览器 UA → 200 空附件（无 Location、0 字节）；
      组件 UA → 302 且 Location 指向 115；组件 UA 取 CDN → 206
- [x] 7.4 **G7e 不回归**：桌面 UA 首次 → 直接 302；0 ERROR

## Phase 8: 归档
- [x] 8.1 `mark-check` → 勾选 → `pre-archive` → `archive`
- [x] 8.2 `add_session.py` + 提交 + `git push`

## Validation Commands
```bash
export PATH=/usr/local/go/bin:/root/go/bin:$PATH GOWORK=off GOPROXY=https://goproxy.cn,direct
cd /root/LitePan
grep -rn "v0\.0\.49" internal/ web/src README.md docker-compose*.yml   # 应为空
make lint && go vet ./... && go test -count=1 ./... && (cd web && npm run type-check)
make docker-build DOCKER_IMAGE=ghcr.io/zhemed/litepan:v0.0.50
docker run --rm --entrypoint /bin/sh ghcr.io/zhemed/litepan:v0.0.50 -c \
  'grep -c v0.0.50 /app/litepan; grep -c v0.0.49 /app/litepan; grep -c HeyTapBrowser /app/litepan'
# 飞牛侧（SSH）：pull/up -d → 版本与库核对 → 两跳模拟（浏览器 UA / 组件 UA / 组件 UA 取 CDN）
```

## Review Gates
| 门 | 判据 |
|---|---|
| G1 | 版本串 5 处、旧串零残留 |
| G2 | 质量门全绿、前端与 embed 零改动 |
| G3 | main 已推送、origin/main 一致 |
| G4a/b | 镜像含 v0.0.50 与握手特征串；GHCR 三 tag 同 digest |
| G5 | tag 远端 sha == 本地；release 非草稿 |
| G6/G7 | 飞牛容器 = v0.0.50；三连通过；库/形态未变；两跳模拟三条全过；桌面不回归 |

## Out of Scope
- 本机 `:5211` 实例更新、GHCR 历史清理、安卓真机操作（由用户执行）
- 若真机发现系统组件不带 cookie 导致 401 → 另开任务补短期令牌

## 执行记录（实施后回填）


---

## 执行记录（2026-10-16 实测）

| 门 | 结果 |
|---|---|
| G0 发布内容 | `v0.0.49..main` 共 7 提交、0 merge；非记账改动仅 `internal/playback/{handoff.go,handoff_test.go,service.go}` |
| G1 版本 | 5 处替换；`v0.0.49` 源码/部署文件**零残留**；diff 恰 4 文件 |
| GQ 质量门 | `make lint` 0 issues、`go vet` exit=0、`go test ./...` 26 包 ok、`vue-tsc -b` exit=0；G2 前端/embed 零改动（`git status` 无输出） |
| G3 | 版本提交 `6ce9004d`；`git push` 一并推送此前 4 个提交 → `origin/main = 6ce9004d` |
| G4a 镜像 | ID `eeda8ce7d49a` ≠ v0.0.49 的 `6ddd6e7973da`；镜像内 `v0.0.50`=1、`v0.0.49`=0、握手特征串 `HeyTapBrowser`=1 |
| G4b GHCR | 单条 version `["0.0.50","v0.0.50","latest"]` 同 digest `eeda8ce7d49a`；`0.0.49` 仍指旧 digest |
| G5 tag/Release | tag `v0.0.50` 远端 sha == 本地 `6ce9004d`；release 非草稿非预发布 |
| G6 部署 | 飞牛生产机容器 = `ghcr.io/zhemed/litepan:v0.0.50`（18:25 启动） |
| G7a | health 200、version = **v0.0.50** |
| G7b/G7c | 库 `migration=25 / tables=9 / accounts=2` 与部署前**逐项一致**；容器网络模式/挂载/环境未变 |
| G7d **两跳模拟** | ① 安卓浏览器 UA → `200` + `attachment` + `Content-Length: 0` + 无 `Location` + 空体；② 同资源系统组件 UA → `302` + `Location` 指向 115 CDN；③ 用组件 UA 取该链接 → **`206`** |
| G7e 不回归 | ④ 桌面 UA 同资源 → `302`；⑤ `level=ERROR` = 0 |

### 关于生产机升级的如实记录

- 用户在该步骤中途提出「生产机的升级我去做，你不要动」。经**只读确认**：那条更新命令（备份 compose → 改 tag → `docker compose pull && up -d`）
  **在被中止前已经执行完毕** —— 容器已在跑 v0.0.50、部署文件 tag 已是 v0.0.50、备份 `docker-compose.yml.bak-v049` 在位。
  之后我只做了**用户逐项许可的只读核对**（状态确认 + 上述 G7 验证），未再做任何写入。
- 部署方式：compose 项目 `litepan`，配置文件为 NAS 上的 `docker-compose.yml`（非仓库内文件）；**未手搓 `docker run`**。

### 未做/限制

- **真机验证未做**（由用户在手机上执行）：预期点下载即可正常下载；若安卓不发起第二跳或组件不带登录态（401），需回退或另开任务补短期令牌。
- 未清理 GHCR 历史版本、未动本机 `:5211` 实例（仍 v0.0.49）。
