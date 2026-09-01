# 本地构建最新版本

## Goal

在本地 `GOWORK=off` 构建与 `ghcr.io/zhemed/litepan:0.0.11` 等价的最新镜像（`v0.0.11 5f72e15`），并验证 `go vet/type-check/build` 与 `health`。

## Background

- 远端 `0.0.11` 已发布 `sha256:3f4be7f`，本地需 `本地构建` 以验证可复现。

## Requirements

- **构建**：`GOWORK=off go vet ./...` `cd web && npm run type-check && npm run build` `GOWORK=off go build -o /tmp/litepan ./cmd/litepan` `docker build -t ghcr.io/zhemed/litepan:0.0.11-local -t litepan-go:local`。
- **验证**：`docker run` `latest` 与 `local` 对比 `health`，`go test ./internal/automation -run TestCompute` 等。

## Constraints

- 遵循 `AGENTS.md`：`task.py start` 后执行。

## Acceptance Criteria

- [ ] `go vet 0` `type-check 0` `go build 27M`
- [ ] `docker build local` 成功，`health 200`

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.
