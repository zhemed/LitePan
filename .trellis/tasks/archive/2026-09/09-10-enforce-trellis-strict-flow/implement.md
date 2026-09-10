# 执行步骤

1. [ ] 写 `~/.dsh/AGENTS.md` 条目（全局：Trellis 托管项目严格执行）
2. [ ] 写 `AGENTS.md` 强制规则段（完整序列 + 完成判据 + 门禁命令 + 会话号口径）
3. [ ] 实现 `.trellis/scripts/flow_gate.py`（pre-start / mark-check / pre-archive / --force）
4. [ ] 自测脚本：构造"合规任务"与"违规任务"各一组，断言退出码与提示
5. [ ] 跑质量门：`go vet ./...`、`go test ./...`、`web: npm run type-check`
6. [ ] `flow_gate.py mark-check` 打标
7. [ ] 更新 PRD 验收勾选 → `task.py archive --skip-branch-validation` → `add_session.py` → push
8. [ ] 会话内容加载：确认已载入 trellis-start / trellis-check / trellis-finish-work 技能与规则文件
