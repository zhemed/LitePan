# 安装本机完整开发环境：Docker 29.7.2 + Compose v5.4.0、Go 1.26.6、make、golangci-lint v2.12.2

## Goal

全新 Ubuntu 22.04.5 主机（16 核 / 7.8G / 236G 可用）在 `git clone` 后**只有 Node v22.23.2 + npm 10.9.8**：没有 Docker、没有 Go、没有 make、没有 golangci-lint。这导致两件事同时不可行：① 无法按 README/`docker-compose.yml` 运行 LitePan；② 无法执行本项目的 Trellis 质量门（`go vet ./...`、`go test ./...`、`cd web && npm run type-check && npm run build`）。

本任务把四项宿主工具链装齐并逐项验证，使仓库进入"可构建、可测试、可运行"状态，为后续部署任务与代码开发扫除环境障碍。

## Background

- 只读侦察（2026-09-12）确认：`docker: command not found`；无 `/usr/local/go`；`make`/`golangci-lint`/`sqlite3` 均缺失；`5211` 空闲，`3080/3081` 为 DSH GUI（不可占用）；无 `data/`、`mounts/`、`data/litepan.db`。
- 网络可达：`github.com`、`download.docker.com`、`get.docker.com`、`ghcr.io`、`registry-1.docker.io` 全部 200/401（401 为正常鉴权响应）；apt 走 `mirror.nju.edu.cn`。
- 宿主基础良好：PID 1 为 `systemd`（`is-system-running` = `running`），`/dev/fuse` 存在（`crw-rw-rw-`），`fuse3`/`fusermount3` 已装 —— 故 `systemctl enable --now docker` 可用，容器路线与 `-tags fuse` 构建路线均可行。
- 版本由仓库内既有契约决定，非自由选择：`go.mod` → `go 1.26.6`；`Dockerfile` → `golang:1.26.6-bookworm` + `GOTOOLCHAIN=local`；根目录 `install-docker.sh` → `REQUIRED_DOCKER=29.7.2`、`REQUIRED_COMPOSE=v5.4.0`；`Makefile` → `GOLANGCI_LINT_VERSION=v2.12.2`。
- 历史沿用：`08-30-install-docker-newapi`（装 Docker）、`08-30-run-container`（容器跑 `:5211`）为同类任务的既有先例。

## Requirements

- **Docker Engine 29.7.2**：优先复用仓库自带 `install-docker.sh`（已固化 `29.7.2` + `v5.4.0`，含显式 `signed-by` 建源、`rootless-extras` 容错、`apt-mark hold`）。装齐 `docker-ce`、`docker-ce-cli`、`containerd.io`、`docker-buildx-plugin`、`docker-compose-plugin`，并 `systemctl enable --now docker` 使 daemon 常驻。
- **Docker Compose v5.4.0**：以 `docker compose`（Compose V2 插件）形式可用。
- **Go 1.26.6**：官方 tarball 安装到 `/usr/local/go`（不装 apt 的 `golang`，jammy 版本过旧），并使 `go` 在非交互 shell 中可用。
- **make**：`apt-get install -y make`（`Makefile` 的 lint/test/build 入口依赖它）。
- **C 工具链（gcc）**：`apt-get install -y build-essential`。侦察发现本机 `gcc`/`cc`/`glibc 开发头` 全部缺失，而项目质量门明确要求 `go test -race`（`spec/backend/backend/quality-guidelines.md:44` "-race required"），`-race` 依赖 cgo/gcc；不装则 `make test` 无法执行。此项为原路线 D 清单的补漏，属"完整开发环境"的必要组成。
- **golangci-lint v2.12.2**：用 `Makefile` 自带的安装方式 `go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.12.2`，落在 `$(go env GOPATH)/bin`，使 `make lint` 的默认查找路径直接命中。
- **前端依赖就绪**：`cd web && npm ci`（`web/package-lock.json` 已在），并跑通 `npm run type-check`（`vue-tsc -b`），证明前端质量门可用。
- **端到端验证**：安装后必须实际跑一次 Go 侧质量门（`go vet`、`go test`）与 Docker 侧自检（`docker run --rm hello-world`），而不是只看版本号。

## Constraints

- **不改仓库源码**：本任务不修改 `internal/`、`drivers/`、`pkg/`、`cmd/`、`web/src`、`go.mod`、`go.sum`、`Makefile`、`.golangci.yml` 或任何受跟踪文件；改动范围仅限宿主系统工具链 + 本任务目录 `.trellis/tasks/09-12-setup-dev-environment/`。
- **不执行 `npm run build`**：`internal/api/web/**` 有 135 个受跟踪的构建产物（`go:embed` 目标），`vite build` 会整批覆写并制造巨大无关 diff；环境就绪以 `npm ci` + `npm run type-check` 判定，完整构建留待真正改前端时执行。
- **版本精确锁定**：Docker `29.7.2`、Compose `v5.4.0`、Go `1.26.6`、golangci-lint `v2.12.2`。不得用"最新版/最新稳定版"替代；若某个精确版本无法获取，必须停下来报告，不得静默换版本。
- **不占用 `3080/3081`**：DSH Web GUI 端口；不得停止、重启或改写 DSH 相关进程与服务。
- **不做部署**：不 `docker compose up`、不 `docker run litepan`、不写 `data/`、不建 `litepan` 容器 —— LitePan 运行属后续独立任务。
- **不装无关组件**：不装 `sqlite3` CLI、`pnpm`、`goimports` 等本次未要求的工具；不升级系统既有包（除为满足依赖必需者）。`build-essential` 因 `go test -race` 的 cgo 依赖而属**在范围内**，非无关组件。
- **镜像与代理**：`GOPROXY=https://goproxy.cn,direct`；npm registry `https://registry.npmmirror.com`；apt 保持既有南大镜像，仅新增 Docker 官方源。
- **权限**：以 root 执行；`install-docker.sh` 自带 root 校验。
- **失败不得静默**：任一组件失败即停止后续可能破坏状态的步骤，保留日志（`/tmp/install-docker.log`、`/tmp/go-install.log`），并在任务内如实记录。

## Acceptance Criteria

- [x] `docker --version` 输出 `Docker version 29.7.2`，且 `docker info` 正常返回（daemon 可达）—— 实测 `Server=29.7.2 Driver=overlayfs Root=/var/lib/docker`
- [x] `docker compose version` 输出包含 `v5.4.0` —— 实测 `Docker Compose version v5.4.0`
- [x] `systemctl is-active docker` 返回 `active`，且 `apt-mark showhold` 含 `docker-ce` 与 `docker-compose-plugin`
- [x] `docker run --rm hello-world` 执行成功（证明 daemon 与镜像拉取链路可用）—— 实测输出 `Hello from Docker!`，镜像 `hello-world:latest` 已落地
- [x] `go version` 输出 `go1.26.6 linux/amd64`，且在非交互 shell 中同样可用 —— 实测 `bash -lc` 与**普通非交互 shell**均可（另建 `/usr/local/bin/go`、`gofmt` 软链，因 DSH 的 bash 工具不加载 `/etc/profile.d`）
- [x] `make --version` 可用 —— 实测 `GNU Make 4.3`
- [x] `gcc --version` 可用（`go test -race` 的 cgo 前置），且 `GOWORK=off go test -race ./internal/store` 能进入编译/执行 —— 实测 `ok litepan/internal/store 3.518s`
- [x] `golangci-lint --version` 输出包含 `2.12.2`（**修订**：二进制自报版本为 `2.12.2` 不带 `v` 前缀，Makefile 钉的是 `v2.12.2`，语义同一版本）
- [x] `cd /root/LitePan && GOWORK=off go vet ./...` 通过 —— 实测 exit=0，无输出
- [x] `cd /root/LitePan && GOWORK=off go test ./...` 执行完毕，结果如实记录 —— 实测 exit=0，全部包 `ok`（115_Open 5.057s / upload 3.313s / adminauth 1.901s / api 1.057s 等），无失败项、无环境性失败
- [x] `cd /root/LitePan/web && npm ci` 成功且 `npm run type-check` 通过；`internal/api/web/**` 零 diff —— 实测 `added 145 packages in 13s`，`vue-tsc -b` exit=0，受跟踪产物 0 改动
- [x] `git status --short` 除本任务目录外无其他改动 —— 实测仅 `?? .trellis/tasks/09-12-setup-dev-environment/`
- [x] `ss -ltnp` 确认 `3080/3081` 仍由 DSH 监听、`5211` 未被占用
- [x] **附加（质量门联动）**：`make lint` 实测 `0 issues.` exit=0，验证 `Makefile` → `$(go env GOPATH)/bin/golangci-lint` 回退路径与 `depguard` 规则链可用

## Notes

- Scope 标注为 `multi-deliverable`（五个可独立验证的交付物：Docker+Compose、Go、make、golangci-lint、C 工具链），故按复杂任务补齐 `design.md` + `implement.md` 后再 `start`。
- **规划期修订记录（2026-09-12，实施前）**：原路线 D 清单为四项（Docker/Go/make/golangci-lint）。实施前侦察发现 `gcc` 缺失会直接卡住项目强制的 `go test -race`，故在**未做任何系统写入之前**回退到 Plan 补入第五项 `build-essential`，并同步 `design.md` / `implement.md`。此为补漏而非范围膨胀。
- 本任务**不产出** LitePan 运行实例；`data/litepan.db` 管理员密码、镜像拉取（`ghcr.io/zhemed/litepan:v0.0.43`）等属后续部署任务。
- 上一台机器的 `/tmp/litepan-backup-20260909-201827.db` 不在本机，本机为新库，部署时管理员将是首次启动默认值 —— 该事项在部署任务中处理，本任务不涉及。
- 部署路线（拉取现成镜像 vs 本地 `docker build`）待本任务完成后另行确认。

## 检查记录（trellis-check，2026-09-12）

**项目质量门（Step 3，全部实测通过）**

| 检查 | 命令 | 结果 |
|---|---|---|
| Lint | `make lint` | `0 issues.` exit=0 |
| Vet | `GOWORK=off go vet ./...` | exit=0 |
| Test | `GOWORK=off go test ./...` | exit=0，全包 `ok`，无失败/跳过 |
| Race | `GOWORK=off go test -race ./internal/store` | ok 3.518s（cgo 链路可用） |
| Type-check | `cd web && npm run type-check` | `vue-tsc -b` exit=0 |
| 前端构建 | `cd web && npm run build` | **刻意未执行** —— 会覆写 135 个受跟踪 embed 产物；理由见 Constraints，属有意偏离并在此留痕 |

**清单核对（Step 4）**

- 代码质量：lint/type-check/test 全绿；无源码改动，故无 debug 日志、无抑制告警、无 `//nolint` 问题。
- 测试覆盖：**N/A** —— 本任务零源码改动，未新增/修改任何 Go/TS 函数或修复任何产品缺陷；新增的 `setup-env.sh` 是宿主安装脚本，仓库无 shell 测试框架。
- 跨层一致性（Step 5）：**N/A** —— 未触及 API/Service/Store/UI 任一层。
- 范围纪律：`git diff --name-only HEAD` = 0 行；`git status --short` 仅本任务目录。**无越界改动。**

**诚实记录：实施期发现与偏离**

1. **我自己的脚本有两个缺陷（已修复，非环境问题）**：① `make --version | head -1 || die` 在 `set -o pipefail` 下因 `head` 提前关闭管道使 `make` 收到 SIGPIPE(141)，产生假失败 —— 同类写法还有 `cmd | grep -q`；已改为"捕获变量 + `case` 匹配"。② golangci-lint 自报版本是 `2.12.2`（不带 `v`），而 Makefile 钉 `v2.12.2`，带 `v` 的断言失败；已改为去 `v` 的数字串比较。
2. **仓库自带 `install-docker.sh` 的副作用（照实记录，未擅自清理）**：`get.docker.com` 分支先装了 `29.8.0` 再由脚本降级到 `29.7.2`，并顺带装了 `docker-model-plugin 1.2.6`；`docker-ce-rootless-extras` 因脚本未钉版本而停在 `5:29.8.0`（其余五项已 `apt-mark hold` 在目标版本）。这些是复用仓库脚本的既定行为，清理它们属超范围。
3. **apt 依赖带动的升级**：`libc6 2.35-0ubuntu3.14→.15`、`libbz2-1.0` 随 `build-essential`/`libc6-dev` 升级，属 Constraints "为满足依赖必需者"。
4. **`/usr/local/bin/go` 软链**：超出原计划的最小实现，但为必要 —— DSH 的 bash 工具运行非登录非交互 shell，不加载 `/etc/profile.d/go.sh`，不加软链则后续每个会话都要手动设 PATH。
5. **spec 同步：建议但未执行（按 trellis-check「不要静默扩大改动」停下）**。`pipefail` + `head`/`grep -q` 的 SIGPIPE 陷阱在本仓库有复用价值（仓库自带 `install-docker.sh` 也用 `set -euo pipefail` + 管道断言），建议后续单独建任务写入 `.trellis/spec/guides/`；本任务 Constraints 声明了"不改任何受跟踪文件"，静默改 spec 会破坏该约束与验收项 11，故交由用户决定。
