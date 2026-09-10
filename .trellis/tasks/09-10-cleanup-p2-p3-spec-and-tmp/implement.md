# 执行步骤

1. [ ] 归档旧 DB 备份 → `data/backups/legacy-20260830-before-0022.db`，列目录核验
2. [ ] 写 `.trellis/spec/backend/backend/upload-task-api.md`（7 段式契约）
3. [ ] 在 `.trellis/spec/backend/backend/index.md` 登记新指南
4. [ ] 改 `flow_gate.py`：`find_task_dir` 支持唯一后缀匹配
5. [ ] 扩展自测：短名唯一命中→通过；歧义→拒绝；全名路径回归
6. [ ] /tmp：按清单删除本会话文件并列出保留项
7. [ ] 质量门：`go vet ./...`、`go test ./...`、`cd web && npm run type-check`
8. [ ] `flow_gate.py mark-check` → 勾选验收 → `pre-archive` → `task.py archive` → `add_session.py` → push
