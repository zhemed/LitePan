# Implementation Plan: 本机完整开发环境（Docker + Go + make + golangci-lint）

## Overview

单会话顺序执行：Docker → Go → make → golangci-lint → 端到端验证 → `trellis-check` → archive。
每阶段都有**独立验证命令**与**回滚点**；任一阶段验证不过即停在该阶段，不进入后续步骤。

前置已确认（只读侦察，2026-09-12）：root 权限、PID 1 = systemd（`running`）、`/dev/fuse` 存在、`3080/3081` DSH 在listen、`5211` 空闲、`download.docker.com`/`go.dev`/`goproxy.cn` 可达、apt = 南大镜像。

---

## Phase 1: Docker Engine 29.7.2 + Compose v5.4.0

- [ ] 1.1 记录安装前基线：`docker --version`（预期 missing）、`apt list --installed 2>/dev/null | grep -c docker`
- [ ] 1.2 执行仓库自带脚本并落日志：
      `cd /root/LitePan && bash install-docker.sh 2>&1 | tee /tmp/install-docker.log; echo "EXIT=${PIPESTATUS[0]}"`
- [ ] 1.3 **验证门 G1**：
      - `docker --version` → 必须 `Docker version 29.7.2`
      - `docker compose version` → 必须含 `v5.4.0`
      - `systemctl is-active docker` → 必须 `active`
      - `apt-mark showhold` → 必须含 `docker-ce`、`docker-compose-plugin`
      - `docker run --rm hello-world` → EXIT 0
- [ ] 1.4 失败处理：读 `/tmp/install-docker.log` 定位（源/gpg/候选版本/daemon）；**不静默换版本**，如实记录后停在此阶段
- **回滚点 R1**：`bash install-docker.sh` 幂等，可重复执行；若需彻底回滚见 `design.md` Rollback 段

## Phase 2: Go 1.26.6

- [ ] 2.1 取官方 sha256：`curl -fsSL 'https://go.dev/dl/?mode=json' | python3 -c "..."` 提取 `go1.26.6.linux-amd64.tar.gz` 的 sha256（取不到则停止并报告，不猜测）
- [ ] 2.2 下载 + 校验：
      `curl -fsSL -o /tmp/go1.26.6.linux-amd64.tar.gz https://go.dev/dl/go1.26.6.linux-amd64.tar.gz`
      `sha256sum -c` 比对发布值，**不匹配立即中止且不执行 2.3**
- [ ] 2.3 解包到 `/usr/local/go`（`rm -rf /usr/local/go` 后 `tar -C /usr/local -xzf`），日志落 `/tmp/go-install.log`
- [ ] 2.4 让 `go` 对非交互 shell 可见（写入 `/etc/profile.d/go.sh`，并在当前会话 `export PATH=/usr/local/go/bin:$PATH`）
- [ ] 2.5 **验证门 G2**：`bash -lc 'go version'` → 必须 `go1.26.6 linux/amd64`；`bash -lc 'go env GOPATH'` → `/root/go`
- **回滚点 R2**：`rm -rf /usr/local/go /etc/profile.d/go.sh /tmp/go1.26.6.linux-amd64.tar.gz`

## Phase 3: make + C 工具链（cgo / `-race` 前置）

- [ ] 3.1 `apt-get install -y make build-essential`
      （`build-essential` 为规划期补漏项：`spec/backend/backend/quality-guidelines.md:44` 要求 `go test -race`，`-race` 走 cgo，而本机 `gcc`/`cc`/`libc6-dev` 全缺）
- [ ] 3.2 **验证门 G3**：`make --version` 可用；`gcc --version` 可用；`cd /root/LitePan && make -n lint` 能解析出 golangci-lint 路径（此时二进制可能尚未装，仅验 Makefile 可解析）
- **回滚点 R3**：`apt-get remove --purge -y make build-essential`

## Phase 4: golangci-lint v2.12.2

- [ ] 4.1 `export GOPROXY=https://goproxy.cn,direct && go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.12.2`
- [ ] 4.2 **验证门 G4**：`/root/go/bin/golangci-lint --version` → 必须含 `v2.12.2`；且 `command -v golangci-lint` 或 Makefile 回退路径能命中该文件
- **回滚点 R4**：`rm -f /root/go/bin/golangci-lint`

## Phase 5: 端到端验证（环境真能用的证明）

- [ ] 5.1 `cd /root/LitePan && GOWORK=off go vet ./...` → EXIT 0
- [ ] 5.2 `cd /root/LitePan && GOWORK=off go test ./...` → 记录完整结果；**若有失败，判定归属**（环境性 vs 既有缺陷）并写入本任务记录，不擅自修代码
- [ ] 5.2b `cd /root/LitePan && GOWORK=off go test -race ./internal/store` → 验证 cgo/-race 链路可编译可执行（不报 `gcc: not found`）
- [ ] 5.3 `cd /root/LitePan/web && npm ci` → EXIT 0
- [ ] 5.4 `cd /root/LitePan/web && npm run type-check` → EXIT 0（**不跑 `npm run build`**，见 Constraints）
- [ ] 5.5 `git status --short` → 除 `.trellis/tasks/09-12-setup-dev-environment/` 外**无任何改动**（含 `internal/api/web/**` 零 diff）
- [ ] 5.6 `ss -ltnp` → `3080/3081` 仍由 DSH 监听，`5211` 仍空闲
- [ ] 5.7 可选端到端：`cd /root/LitePan && make lint`（验证 Makefile + golangci-lint 联动；若因既有 lint 债务失败，如实记录归属，属既有状态而非环境问题）

## Phase 6: Commit & Archive

- [ ] 6.1 `python3 ./.trellis/scripts/flow_gate.py mark-check 09-12-setup-dev-environment --note "<质量门摘要>"`
- [ ] 6.2 逐条勾选 `prd.md` 的 Acceptance Criteria（**每条都必须实际执行过**）
- [ ] 6.3 `python3 ./.trellis/scripts/flow_gate.py pre-archive 09-12-setup-dev-environment` → 通过
- [ ] 6.4 `python3 ./.trellis/scripts/task.py archive 09-12-setup-dev-environment`
- [ ] 6.5 `python3 ./.trellis/scripts/add_session.py --title "安装本机完整开发环境" --summary "..."`（会话号以 journal 为准）

---

## Validation Commands（汇总，供 check 阶段复核）

```bash
docker --version && docker compose version && systemctl is-active docker && apt-mark showhold
docker run --rm hello-world
bash -lc 'go version && go env GOPATH'
make --version
gcc --version
/root/go/bin/golangci-lint --version
cd /root/LitePan && GOWORK=off go vet ./...
cd /root/LitePan && GOWORK=off go test ./...
cd /root/LitePan && GOWORK=off go test -race ./internal/store
cd /root/LitePan/web && npm run type-check
cd /root/LitePan && git status --short
ss -ltnp | grep -E '3080|3081|5211'
```

## Review Gates

| 门 | 位置 | 判据 |
|---|---|---|
| G1 | Phase 1.3 | Docker 29.7.2 + Compose v5.4.0 + daemon active + hello-world 成功 |
| G2 | Phase 2.5 | `go1.26.6 linux/amd64`，且非交互 shell 可见 |
| G3 | Phase 3.2 | `make --version` + `gcc --version` 可用 |
| G4 | Phase 4.2 | `golangci-lint v2.12.2` |
| G5 | Phase 5 | vet/test/type-check 结果齐全 + 工作区零污染 + DSH 端口未受影响 |
| G6 | Phase 6.1-6.3 | `flow_gate.py mark-check` + 验收全勾选 + `pre-archive` 通过 |

## Rollback

见 `design.md` → **Rollout / Rollback**。要点：各阶段独立可回滚；`install-docker.sh` 幂等可重跑；仓库因零改动而无需回滚，`rm -rf web/node_modules web/*.tsbuildinfo` 即回 clone 原状。

## Out of Scope（后续任务）

- LitePan 部署与运行（拉取 `ghcr.io/zhemed/litepan:v0.0.43` 或本地 `docker build`）→ 需用户另行确认路线
- 首次启动管理员口令处理 → 部署任务处理
- 任何仓库源码改动
