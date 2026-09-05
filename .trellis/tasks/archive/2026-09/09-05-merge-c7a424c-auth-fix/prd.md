# 合入c7a424c认证修复并发版0.0.14

## Goal

按排除清单挑拣合入c7a424c认证相关26源文件+测试，go test验证，bump到0.0.14发版推送部署

## Requirements

1. 以 `git cherry-pick -n c7a424c` 后 revert 排除文件的方式合入（等价 `git apply` 过滤补丁）。合入集：调查报告中的 26 源文件 + 7 测试文件。
2. 排除集：其他驱动 13 文件、announcement×4、cacheretention×2、strm×3、`wire_http.go`、`user_agent.go`、`httpx/oauth_test.go`、`settings/service.go`、版本文件（buildinfo/version.ts/README/index.html）。
3. 验证：`go build + go vet + go test ./internal/auth/... ./internal/driver/... ./drivers/115_Open/... ./drivers/189Cloud/... ./internal/core/driverexec/... ./internal/account/...` 全过。
4. 版本 `0.0.13 → 0.0.14`（README + 2 compose 标签），前端无变更则复用现有 `internal/api/web`（如构建产物变化则一并提交）。
5. 发版：docker 构建 `0.0.14/v0.0.14/latest` 推 GHCR，`git tag v0.0.14`，推 `github/main`，本地容器换 `latest` 并健康检查。
6. 真机双账号（115+天翼）刷新验证由用户在 UI 侧确认；任务内完成自动化部分并明确标注待用户验证项。

## Acceptance Criteria

- [ ] 合入集文件与上游一致，排除集零残留（`git diff` 抽查）
- [ ] `go build/vet/test` 全过
- [ ] `v0.0.14` 三标签可拉取，`github/main` 同步，容器健康且为新镜像
- [ ] 待用户真机验证项已书面列出

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.
