# Record mandatory trellis rule and admin password

## Goal

将用户于 2026-08-30 明确的强制规则“**所有操作必须调用 trellis**”与当前线上管理员凭据 `admin / 123456` 持久化到版本库，使后续所有会话（含新开 `dsh_session`）均可见且可审计。

## Requirements

- **规则原文**：`所有操作必须调用trellis` —— 含代码、配置、数据（`data/litepan.db`）、镜像（`docker build/run`）、前端（`web`）等任何写操作
- **落盘位置**：
  - `AGENTS.md`：在 `<!-- TRELLIS:END -->` 之后追加「项目强制规则」章节，置顶该句，外加当前密码 `admin/123456` 的说明（不暴露 hash）
  - `.trellis/config.yaml`：在 `session_commit_message` 段追加注释块，同样置顶
  - `.trellis/workspace/zhemed/journal-1.md`：已在 Session8 落盘，此任务仅追认 AGENTS/config 的显式化
- **不泄露**：不写 `hash`、`secret.key`、`litepan.db` 路径，仅写 `admin / 123456` 明文（用户已公开）与规则句

## Acceptance Criteria

- [ ] `AGENTS.md` 含 `## 项目强制规则` 且首句为 `所有操作必须调用trellis`
- [ ] `AGENTS.md` 含 `当前管理员：admin / 123456` 且位于同一章节
- [ ] `.trellis/config.yaml` 含同句注释且 `git diff` 可见
- [ ] `python .trellis/scripts/get_context.py` 仍 `Clean`，`GOWORK=off go vet` 不受影响（仅文档）

## Notes

- Lightweight 任务，PRD-only 即可 `task.py start → archive`
- 为满足「所有操作必须调用 trellis」，本任务本身即按 Trellis 创建
