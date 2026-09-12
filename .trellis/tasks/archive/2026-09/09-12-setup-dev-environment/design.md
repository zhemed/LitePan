# Design: 本机完整开发环境（Docker + Go + make + golangci-lint）

## Overview

本任务只动**宿主机工具链**，不动仓库源码。四个交付物互不依赖编译关系，但共享同一批前置事实（root、systemd running、`/dev/fuse` 存在、apt 为南大镜像、外网可达 Docker 官方源与 goproxy.cn），因此排成线性序列执行：先 Docker（耗时最长、失败面最大、需 daemon 常驻），再 Go（后续 golangci-lint 的前置），然后 make 与 golangci-lint（`go install` 依赖 Go 已就绪），最后跑端到端验证。

版本不是自由选择，而是被仓库既有契约钉死：

| 组件 | 版本 | 钉死来源 |
|---|---|---|
| Docker Engine | `29.7.2` | `install-docker.sh:8 REQUIRED_DOCKER` |
| Compose 插件 | `v5.4.0` | `install-docker.sh:9 REQUIRED_COMPOSE` |
| Go | `1.26.6` | `go.mod:3` + `Dockerfile` `golang:1.26.6-bookworm`、`GOTOOLCHAIN=local` |
| golangci-lint | `v2.12.2` | `Makefile GOLANGCI_LINT_VERSION ?= v2.12.2` |

## Boundaries

| 层 | 改 | 不改 |
|---|---|---|
| **宿主 apt** | 新增 Docker 官方源 `/etc/apt/sources.list.d/docker.list` + `/etc/apt/keyrings/docker.gpg`；装 `docker-ce`/`docker-ce-cli`/`containerd.io`/`docker-buildx-plugin`/`docker-compose-plugin`/`make`/`build-essential`；`apt-mark hold` 锁 Docker 版本 | 既有南大镜像源；不 `apt upgrade`；不装 `sqlite3`/`pnpm` 等未要求组件 |
| **宿主 /usr/local** | `/usr/local/go`（官方 tarball 解包） | 不动 `/usr/local` 其他内容 |
| **宿主 shell PATH** | 使 `go` 对非交互 shell 可见 | 不改 DSH 进程环境；不重启 DSH（`3080/3081`） |
| **仓库工作区** | `web/node_modules/`（gitignored）、`web/*.tsbuildinfo`（gitignored） | `internal/`、`drivers/`、`pkg/`、`cmd/`、`web/src`、`go.mod`、`Makefile`、`.golangci.yml`，以及受跟踪的 `internal/api/web/**`（135 个构建产物） |
| **任务记录** | `.trellis/tasks/09-12-setup-dev-environment/` | 其他 `.trellis/` 内容 |

## Data Flow

```
① Docker
   install-docker.sh (仓库自带, 105 行, 幂等: 版本≠29.7.2 则重建官方源)
     → 校验 root + 识别 ubuntu → REPO_URL=download.docker.com/linux/ubuntu
     → gpg --dearmor → /etc/apt/keyrings/docker.gpg
     → docker.list (signed-by, $VERSION_CODENAME=jammy, stable)
     → apt-get install --allow-downgrades docker-ce=5:29.7.2* docker-ce-cli=5:29.7.2*
                              docker-compose-plugin=5.4.0* docker-buildx-plugin containerd.io
     → apt-mark hold (锁版本, 防后续 apt 升级漂移)
     → systemctl enable --now docker   ← 本机 PID 1 确为 systemd, 该步可成功
     → 自校验: docker --version == 29.7.2 && docker compose version == v5.4.0

② Go 1.26.6
   https://go.dev/dl/go1.26.6.linux-amd64.tar.gz
     → 校验收到的 sha256 == go.dev/dl/?mode=json 发布值 (不匹配即中止)
     → rm -rf /usr/local/go && tar -C /usr/local -xzf
     → PATH: /usr/local/go/bin
     → go version == go1.26.6 linux/amd64

③ make + C 工具链   apt-get install -y make build-essential
     → make --version; gcc --version
     → 为何需要 gcc: quality-guidelines.md:44 要求 `go test -race ./...`, -race 走 cgo,
       侦察确认本机 gcc/cc/libc6-dev 全缺 → 不装则 `make test` 直接失败
④ golangci-lint   go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.12.2
     → $(go env GOPATH)/bin/golangci-lint == /root/go/bin/golangci-lint
     → 该路径由 Makefile 的 GOLANGCI_LINT 回退分支直接命中, make lint 无需再装

⑤ 端到端验证（证明环境真能用, 而非只看版本号）
   GOWORK=off go vet ./...            # Go 侧质量门 1
   GOWORK=off go test ./...           # Go 侧质量门 2（结果如实记录）
   cd web && npm ci && npm run type-check   # 前端质量门（不跑 build）
   docker run --rm hello-world        # daemon + registry 链路
   ss -ltnp                           # 3080/3081 仍在, 5211 仍空
```

## Compatibility

- **apt 源共存**：Docker 官方源与既有南大 jammy 镜像并存；`docker.list` 显式 `signed-by`，不会引发 `NO_PUBKEY`。`install-docker.sh` 已按 `grep -qi ubuntu /etc/os-release` 选择 `download.docker.com/linux/ubuntu`，本机 `VERSION_CODENAME=jammy` 有对应发行版目录。
- **`allow-downgrades`**：本机为全新环境、不存在低版本 `docker.io`，该参数仅为脚本自带容错，不触发降级。
- **Go 与 Dockerfile 的细微不一致**：`Dockerfile` 注释写"与 go.mod 的 go 1.26.4 对齐"，而 `go.mod` 实际是 `1.26.6`（`FROM golang:1.26.6-bookworm` 已对齐 1.26.6）。本任务以 `go.mod` 为准装 `1.26.6`，**不修改该注释**（属源码改动，超出本任务边界）。
- **`web/node_modules` 与 `web/*.tsbuildinfo`**：均已在 `.gitignore`（`web/node_modules/`、`web/*.tsbuildinfo`），故 `npm ci` 与 `vue-tsc -b` 不污染工作区。
- **`internal/api/web/**` 受跟踪**：这是本设计刻意绕开 `npm run build` 的原因；若执行 build，会覆写 135 个受跟踪产物，制造与"环境准备"无关的巨大 diff。
- **FUSE**：`/dev/fuse` 存在且 `fuse3` 已装，`make build`（`-tags fuse`）与后续容器 `--device /dev/fuse` 均具备条件；本任务只做 `go vet`/`go test`，不实际挂载。
- **DSH 共存**：`3080/3081` 由 DSH 持有，安装过程不触碰其服务、端口、进程。

## Tradeoffs

- **复用仓库 `install-docker.sh` vs 手写 apt 命令**：复用脚本可与仓库契约（`29.7.2`/`v5.4.0`）保持单一事实源，且该脚本已在历史任务中按"已装/未装/非目标版本"三种分支加固；代价是脚本按 `ubuntu` 分支写死 `download.docker.com/linux/ubuntu`，若未来换发行版需改脚本（本任务不涉及）。
- **官方 tarball vs `apt install golang`**：jammy 的 `golang` 包远低于 `1.26.6`，无法满足 `go.mod`；tarball 可精确锁版并校验 sha256。代价是需手动维护 PATH。
- **`go install` vs 下载 golangci-lint release 二进制**：`go install` 与 `Makefile lint-install` 完全一致，且落在 Makefile 默认查找路径 `/root/go/bin`；代价是需先装 Go 并走 goproxy 下载依赖。
- **验证 `npm run type-check` 而非 `npm run build`**：type-check 已覆盖 `vue-tsc -b` 这一重头戏，足以判定前端工具链就绪，同时避免覆写受跟踪产物。代价是"`vite build` 是否能成功"本任务不给出结论（`08-30-deploy-local` 已有 build 成功的历史记录，风险低）。
- **装齐五项 vs 只装 Docker**：用户选择路线 D。多装 Go 的成本约 150MB 下载，但解除的是"无法跑质量门"这一根本阻塞 —— 后续任意代码任务都需要它。`build-essential`（约 100MB）同理由 `go test -race` 的 cgo 依赖决定，属规划期补漏。

## Rollout / Rollback

- **分步可停**：四步各自独立，任一步失败即停止，已完成部分不需回退即可保留使用（例如 Go 装好而 Docker 失败，`go vet`/`go test` 仍可用）。
- **回滚 Docker**：`apt-mark unhold docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin` → `apt-get remove --purge docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin` → 删除 `/etc/apt/sources.list.d/docker.list` 与 `/etc/apt/keyrings/docker.gpg`；宿主无既有数据卷，`/var/lib/docker` 可删。
- **回滚 Go**：`rm -rf /usr/local/go` 并移除 PATH 行。
- **回滚 make**：`apt-get remove --purge make`。
- **回滚 golangci-lint**：`rm -f /root/go/bin/golangci-lint`。
- **仓库回滚**：本任务不改受跟踪文件，故 `git checkout .` 无内容可还原；`rm -rf web/node_modules web/*.tsbuildinfo` 即可回到 clone 后原状。
- **无需回滚的场景**：`install-docker.sh` 自带幂等判断（版本已等于目标则跳过重建源），重复执行安全。

## File Map

| 路径 | 动作 | 说明 |
|---|---|---|
| `/etc/apt/sources.list.d/docker.list` | 新增 | Docker 官方源（`signed-by`） |
| `/etc/apt/keyrings/docker.gpg` | 新增 | Docker 官方 GPG key |
| `/usr/local/go/` | 新增 | Go 1.26.6 工具链 |
| `/root/go/bin/golangci-lint` | 新增 | golangci-lint v2.12.2 |
| `/tmp/install-docker.log` | 新增 | Docker 安装日志（回溯用） |
| `/tmp/go-install.log` | 新增 | Go 安装日志（回溯用） |
| `web/node_modules/` | 新增（gitignored） | `npm ci` 产物 |
| `.trellis/tasks/09-12-setup-dev-environment/` | 新增 | 本任务 PRD / design / implement / 验收记录 |
| 仓库受跟踪文件 | **零改动** | 硬约束：`git status` 除任务目录外必须干净 |
