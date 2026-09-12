# 补齐 golangci-lint 全局 PATH 软链（/usr/local/bin）

## Goal

上一任务（`09-12-setup-dev-environment`，已归档）把 `golangci-lint v2.12.2` 装在 `$(go env GOPATH)/bin` 即 `/root/go/bin`，而该目录不在 PATH。项目实际调用路径不受影响（`Makefile` 的 `GOLANGCI_LINT ?= $(shell command -v golangci-lint 2>/dev/null || echo "$(shell go env GOPATH)/bin/golangci-lint")` 回退分支已验证，`make lint` → `0 issues.`），但裸 shell 直接执行 `golangci-lint` 会 `command not found`。

本任务按用户指示补 `/usr/local/bin/golangci-lint` 软链，使该工具在任意 shell（含 DSH 的非登录非交互 bash）中可直接调用，与已建的 `/usr/local/bin/go`、`/usr/local/bin/gofmt` 软链保持一致。

## Background

- `PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/snap/bin` —— `/usr/local/bin` 在默认 PATH 内，故软链即可全局生效，无需改 shell 配置。
- 同类先例：上一任务已用同样方式解决 `go` 在 DSH 非登录 shell 中不可用的问题（`/etc/profile.d/go.sh` 对非登录 shell 不生效）。
- 选软链而非改 PATH 的理由：DSH 的 bash 工具既不加载 `/etc/profile.d/*` 也不加载 `~/.bashrc`，软链是唯一对"任意非交互 shell"都生效的方案。

## Requirements

- 建立软链 `/usr/local/bin/golangci-lint` → `/root/go/bin/golangci-lint`
- 目标必须是指向 `v2.12.2` 的同一二进制；软链不得掩盖版本漂移（即断言**版本**，而不仅断言命令存在）
- 不改 `Makefile`、不改 `.golangci.yml`、不改任何仓库受跟踪文件
- 不动 DSH（`3080/3081`）；不触碰 Docker 与已有工具链配置

## Constraints

- **零仓库改动**：`git diff --name-only HEAD` 必须为 0；改动仅限宿主 `/usr/local/bin` 软链 + 本任务目录
- 用 `ln -sfn` 保证幂等（重复执行不报错、不产生嵌套软链）
- 需宿主写权限（DSH 沙箱对工作区外写入默认拒绝，需用户批准的提权）
- 本任务**不**解决 `$GOPATH/bin` 下未来新工具的 PATH 问题 —— 那属于放开 PATH/`profile.d` 的策略变更，需另行决策

## Acceptance Criteria

- [x] `ls -l /usr/local/bin/golangci-lint` 显示指向 `/root/go/bin/golangci-lint` 的软链 —— 实测 `lrwxrwxrwx ... /usr/local/bin/golangci-lint -> /root/go/bin/golangci-lint`
- [x] 在非登录非交互 shell 中 `golangci-lint --version` 输出 `2.12.2`（裸 shell 直接可用）—— 实测输出 `golangci-lint has version 2.12.2 built with go1.26.6 ...`
- [x] `command -v golangci-lint` 解析到 `/usr/local/bin/golangci-lint` —— 实测命中
- [x] `cd /root/LitePan && make lint` 仍 `0 issues.` exit=0（软链未破坏 Makefile 原路径）—— 实测 `0 issues.` exit=0
- [x] `git status --short` 除本任务目录外无改动；`git diff --name-only HEAD` = 0 —— 实测 0 行 diff，status 仅 `?? .trellis/tasks/09-12-golangci-lint-path-symlink/`
- [x] `ss -ltnp` 确认 `3080/3081` 仍由 DSH 监听、`5211` 仍空闲 —— 实测仅 3080/3081 在听

## Notes

- Lightweight task：PRD-only，无需 `design.md` / `implement.md`（scope 标注 `lightweight`）。
- 本任务由用户在 `09-12-setup-dev-environment` 收尾时以「补软链，让 golangci-lint 全局可用」明确授权发起。
- 复盘候选（本任务不做）：`set -o pipefail` 下 `cmd | head -1` / `cmd | grep -q` 的 SIGPIPE 假失败陷阱值得写入 `.trellis/spec/guides/`，详见上一任务 `prd.md` 检查记录第 5 条。
