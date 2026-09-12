# 09-12-remove-dead-code-p1

## Goal

执行死代码排查报告（`09-12-investigate-dead-code-sweep` §9）中的 **P1 清理**：删除 3 个零消费者包（`internal/proxybase`、`internal/taskauth`、`pkg/strutil`）与 **24 个真死函数**（跨 20 个文件）；删除后重跑 `deadcode` 验证，跑全量质量门，发布 `0.0.41` 并完成本地部署验证。

## Background

- 报告依据：`.trellis/tasks/archive/2026-09/09-12-investigate-dead-code-sweep/research.md`（判据＝可达性：生产依赖图 + `deadcode` prod 与 `-test` 差集）。
- 24 个真死函数分布在 20 个文件：`backuprestore/maintenance.go`(3)、`apikey/service.go`(2)、`automation/helpers.go`(2)、`driver/uploadutil/{hash,progress,resume}.go`(3)、`upload/{lifecycle,manager,target_dir,worker}.go`(4)、`httpx/*` 不在本批、`file/name_align.go`(1)、`settings/registry.go`(1)、`api/sse.go`(1)、`notification/service.go`(1)、`driver/{delay,upload}.go`(2)、`189Cloud/transport.go`(1)、`pkg/jsonvalue/flexible_string.go`(1)、`pkg/security/origin.go`(1)、`pkg/speedsmoother/smoother.go`(1)。
- ⚠️ **反射/接口语义必须先行复核**：`deadcode` 基于 RTA，看不到 `encoding/json` 等**反射调用**。已知风险点：`pkg/jsonvalue.FlexibleString.UnmarshalJSON` 实现 `json.Unmarshaler`，即便"无人调用"也**必须保留**（否则该类型的 JSON 解码语义改变）。同类（实现标准接口的方法）一律先复核再决定。

## Requirements

- **R1 删包**：删除 `internal/proxybase/`、`internal/taskauth/`、`pkg/strutil/`（含各自测试文件）。删除前逐包复核零 import（`grep` + `go list -deps ./cmd/litepan`）。
- **R2 删函数**：删除 §4.1 表中 24 个真死函数。执行前对每个符号做两项复核：
  1. 是否实现标准/自有接口（`Read`/`UnmarshalJSON`/`MarshalJSON` 等反射可达方法）；
  2. 是否被接口声明（`grep` 接口定义）。
  复核不通过者 **不删**，并在 PRD/journal 记录原因（预期至少有 `FlexibleString.UnmarshalJSON` 一项）。
- **R3 验证**：删除后重跑 `deadcode ./cmd/litepan` 与 `deadcode -test ./...`，确认目标符号消失、且**未引入新发现**（留存两份输出对比）。
- **R4 质量门**：`go vet ./...`、`go test ./...`、`go build ./...`、`cd web && npm run type-check && npm run build && npm run check:memo` 全绿；`go mod tidy` 若有依赖因此变为未使用则一并处理（先 `-diff` 预演）。
- **R5 发版**：bump `0.0.41`（README + docker-compose）；镜像三 tag；**顺序：push main → tag → push tag → 最后 `gh release`**（会话内已犯过一次，严格遵守）；本地容器重建 + health/登录/任务汇总三连。
- **R6 记录**：PRD 勾选附证据；journal 记录删除清单、排除项及原因、验证结果。

## Constraints

- **不扩张范围**：本任务只做 P1；前端 12 个零引用文件（P2）、`drivers/template`+httpx OAuth 链路（P3）、`refresh-auth` 端点（P4）不在本轮。
- 不触碰生产机；不改 DB；不改任何在线行为（被删代码均生产不可达，预期零行为变化）。
- 版本规则：`0.0.x` 递增 → `0.0.41`。
- 门禁：`pre-start` / `mark-check` / `pre-archive` 全过；**pre-archive 被拒时不得继续 archive**。

## Acceptance Criteria

- [ ] `internal/proxybase`、`internal/taskauth`、`pkg/strutil` 已删除，且删除前有零 import 复核证据
- [ ] 24 个真死函数已按复核结果处理：删除或（复核不通过者）保留并记录原因
- [ ] `deadcode ./cmd/litepan` 复跑：目标符号消失且无新增发现（留存对比证据）
- [ ] `go vet ./...`、`go test ./...`、`go build ./...`、web 三连全绿
- [ ] 版本号 `0.0.41` 已更新（`README.md` + `docker-compose.yml`）
- [ ] 镜像 `ghcr.io/zhemed/litepan:0.0.41`（含 `:v0.0.41`、`:latest`）已推送；tag 指向修复提交（API 复核）；release 已创建（顺序正确）
- [ ] 本地容器已重建到 0.0.41，health / form 登录 / 任务汇总三连通过
- [ ] 删除清单、排除项与原因已写入 journal

## Notes

- `scope=lightweight`：纯删除 + 发版，无接口/契约/行为变更。
- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
