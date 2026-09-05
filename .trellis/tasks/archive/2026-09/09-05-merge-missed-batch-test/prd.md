# 补合入遗漏的batch持久化单测

## Goal

移植de83b46遗漏的TestUploadTaskBatchFieldsPersist到store_test，跑通测试后提交，不发版

## Requirements

1. 将上游 `de83b46` 的 `TestUploadTaskBatchFieldsPersist`（25 行）合入 `internal/store/store_test.go`（审计已验证 `apply --check` 通过、`newTestStore` 存在）。
2. `go test ./internal/store/...` 通过；`go vet` 相关包干净。
3. 仅测试文件变更，不 bump 版本、不打 tag、不重建镜像（搭下次顺风车）；推送 `github/main`。
4. 不碰 4 个可选微补丁（默认不做，用户未点名）。

## Acceptance Criteria

- [ ] 新单测在位且通过
- [ ] 无生产代码变更（`git diff` 仅测试文件）
- [ ] 已推 `github/main`，无版本/tag/镜像变动

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.
