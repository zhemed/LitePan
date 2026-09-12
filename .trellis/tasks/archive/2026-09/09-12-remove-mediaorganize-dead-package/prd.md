# 09-12-remove-mediaorganize-dead-package

## Goal

按用户指示删除**孤儿包** `internal/mediaorganize/`（19 个文件 / 3,796 行非测试代码 + 4 个测试文件；零消费者、不在 `cmd/litepan` 依赖图内），同步修正 `.trellis/spec` 中四处过期表述，发布 `0.0.40` 并完成本地部署验证。

## Background

- 依据任务 `09-12-investigate-mediaorganize-necessity` 的只读结论：目录整理功能已在 `1bcfac8`（2026-08-30）删除；该提交同时移除了 `internal/file/name_align.go` 对 `mediaorganize/rules` 的依赖（改用自带简化解析），此后该包**无任何消费者**：
  - `grep -rn "mediaorganize" --include="*.go"` 全仓仅命中 1 处注释；
  - `GOWORK=off go list -deps ./cmd/litepan | grep -c mediaorganize` = **0**；
  - 无 API 路由、无前端入口（`web/src` 与构建产物均无）、自动化动作只剩 `local_upload`。
- 该包唯一第三方依赖 `github.com/alde/go-fish`（guessit 移植）**仅被它使用**（`go.mod:6`）。
- spec 仍有四处把 `mediaorganize` 描述为"保留/在用"，是本次 P6 误判的源头，需一并修正。

## Requirements

- **R1 删除**：`rm -rf internal/mediaorganize`（含 `rules/` 与测试）。删除前再次确认零消费者（`grep` + `go list -deps`），删除后 `go build ./...` / `go vet ./...` / `go test ./...` 必须全绿。
- **R2 依赖清理**：`github.com/alde/go-fish` 若在删除后无任何使用者，用 `go mod tidy` 移除（先看 `go mod tidy -diff` 是否只动该依赖；若 tidy 引入大范围改动，则保留依赖并在记录中说明）。
- **R3 spec 同步**（四处）：`.trellis/spec/backend/backend/directory-structure.md` 第 25 行（目录树示意）、第 44 行（"`mediaorganize/rules` 保留"）、第 79 行（示例表引用 `mediaorganize/service.go`）、`.trellis/spec/backend/backend/concurrency-and-scheduling.md` 第 41 行（`mediaorganize.Service` 行）；同时核对 `guides/index.md` 等是否有连带引用。
- **R4 不扩张范围**：不删除其它模块、不做无关重构；如发现其它死代码，记入后续"死代码全面排查"任务（另行建立），不在本任务顺手处理。
- **R5 质量门与发版**：`go vet ./...`、`go test ./...`、`go build ./...`、`cd web && npm run type-check && npm run build && npm run check:memo` 全绿；版本 bump `0.0.40`；镜像三 tag 同 digest；tag + release（**顺序：先 push main → tag → push tag → 最后 gh release**）；本地容器重建 + health/登录/任务汇总三连。
- **R6 记录**：PRD 勾选附证据；journal 记录删除范围、依赖处理、spec 改动与验证结果。

## Constraints

- 生产机不涉及、不触碰。
- 只删死代码 + 同步文档；不改任何在线功能行为（预期零行为变化）。
- 版本规则：`0.0.1` 基线、`0.0.x` 递增、不跳 `1.0.0` → 本任务 `0.0.40`。
- 门禁：`flow_gate.py pre-start / mark-check / pre-archive` 均需通过；**pre-archive 被拒时不得继续 archive**（本会话已犯过一次，本次严格遵守）。

## Acceptance Criteria

- [x] `internal/mediaorganize/` 已删除，且删除前有"零消费者"复核证据（grep + `go list -deps`）
      → 删除前复核：`grep -rn "mediaorganize" --include="*.go"` 仅命中 `internal/file/name_align.go:394` 注释；`go list -deps ./cmd/litepan | grep -c mediaorganize` = **0**；`git rm -r internal/mediaorganize`（20 项变更、27 文件、-4,736 行）
- [x] `go vet ./...`、`go test ./...`、`go build ./...` 全绿；web 三连全绿
      → 全部通过：`go test ./...` 全包 ok（exit 0，`internal/mediaorganize/rules` 已不再出现在测试列表）；web `MEMO-ALL-PASS`
- [x] `github.com/alde/go-fish` 依赖已处理（移除或说明保留原因）
      → 先 `go mod tidy -diff` 预演（仅该依赖 + 其 2 条 go.sum + `google/go-cmp` 2 条），确认无外溢后执行：`go.mod -1`、`go.sum -4`
- [x] spec 四处过期表述已修正，且全仓 `grep -rn "mediaorganize" .trellis/spec` 无残留错误描述
      → `directory-structure.md`（目录树行、已移除清单、Module Ownership 示例行）+ `concurrency-and-scheduling.md`（删 `mediaorganize.Service` 行）；复核后仅剩"历史说明 + 教训"两段（有意保留）
- [x] 版本号 `0.0.40` 已更新（`README.md` + `docker-compose.yml`）
      → README 2 处 + docker-compose 1 处
- [x] 镜像 `ghcr.io/zhemed/litepan:0.0.40`（含 `:v0.0.40`、`:latest`）三 tag 同 digest 已推送
      → 三 tag 同 digest `sha256:1f2b2bb4e24af552f68eef5c3c17054643d3d2a3cd9b005d80c4a3c0fa3c4a1d`（**与 0.0.39 同 digest**：实测两版镜像内 `/app/litepan` sha256 均为 `a00da8e36e8fdcf32d7c5fd2859868077aa19044c371e4a6dc9ed90d79b9793d`，即"死代码不进二进制"的实证，属预期而非缓存误用）
- [x] `git tag v0.0.40` 指向修复提交（GitHub API 复核），release 已创建
      → 本地与远端 tag 均为 `4b2f49b01774d84bccc5f1a70995c5e5ecad0057`（`gh api .../git/ref/tags/v0.0.40` 复核一致）；release https://github.com/zhemed/LitePan/releases/tag/v0.0.40
      → 本次严格遵守顺序：push main → tag → push tag → **最后** gh release（上一任务的顺序错误未重演）
- [x] 本地容器已重建到 0.0.40，health / form 登录 / 任务汇总三连通过
      → ImageID `d7039ebbde26`、`Restarts=0`；health ok；登录 ok；任务汇总 `total=13 success=13`
- [x] 删除范围与验证结果已写入 journal
      → Session 127

## Notes

- 本任务为**纯删除 + 文档同步**，无接口/契约/行为变更（预期零行为差异），scope 记为 `lightweight`。
- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
