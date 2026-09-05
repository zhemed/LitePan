# 合入上游353b830目录错位修复

## Goal

cherry-pick上游353b830到本地，vue-tsc验证，按基线递增到0.0.13并按既有发版流程发布

## Requirements

1. 以 `git cherry-pick -n 353b8308`（等价 `git apply`，评估任务已验证 `apply --check PASS`）合入补丁到 `web/src/components/file/CloudLocalUploadPanel.vue`，保持上游注释原样。
2. 前端类型检查 `vue-tsc` 必须 0 错误；Go 侧无改动，`go vet` 抽查通过即可。
3. 按版本基线（fix 递增）将版本 `0.0.12 → 0.0.13`：同步 `internal/buildinfo/version.go`、`web/src/version.ts`、`README.md`、`docker-compose.yml`、`.fnos.yml`（以既有 `0.0.12` 落点为准，grep 确认无遗漏）。
4. 按既有发版流程：前端构建（`internal/api/web`）、`go build`、docker 构建 `0.0.13/v0.0.13/latest` 并推送 GHCR、`git tag v0.0.13`、推送 `github/main`。
5. 不合入 `353b830` 之外的任何上游提交；`de83b46` 批次化、`c7a424c` 认证等不在本任务范围。

## Acceptance Criteria

- [ ] 本地 `CloudLocalUploadPanel.vue` 含上游 8 行修复且注释完整
- [ ] `vue-tsc` 0 错误
- [ ] 所有版本落点一致为 `0.0.13`，`git tag v0.0.13` 已打
- [ ] GHCR `0.0.13/v0.0.13/latest` 可拉取，`github/main` 已同步
- [ ] 本地容器以新版本部署并通过健康检查

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.
