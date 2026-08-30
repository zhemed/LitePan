<!-- TRELLIS:START -->
# Trellis Instructions

These instructions are for AI assistants working in this project.

This project is managed by Trellis. The working knowledge you need lives under `.trellis/`:

- `.trellis/workflow.md` — development phases, when to create tasks, skill routing
- `.trellis/spec/` — package- and layer-scoped coding guidelines (read before writing code in a given layer)
- `.trellis/workspace/` — per-developer journals and session traces
- `.trellis/tasks/` — active and archived tasks (PRDs, research, jsonl context)

If a Trellis command is available on your platform (e.g. `/trellis:finish-work`, `/trellis:continue`), prefer it over manual steps. Not every platform exposes every command.

If you're using Codex or another agent-capable tool, additional project-scoped helpers may live in:
- `.agents/skills/` — reusable Trellis skills
- `.codex/agents/` — optional custom subagents

Managed by Trellis. Edits outside this block are preserved; edits inside may be overwritten by a future `trellis update`.

<!-- TRELLIS:END -->

## 项目强制规则（用户于 2026-08-30 明确，强制）

> **所有操作必须调用 trellis**

- 本项目为 `trellis init --dsh -u zhemed` 托管，**任何**代码、配置、数据（`data/litepan.db` / `secret.key`）、镜像（`docker build/run`）、前端（`web`）等写操作，**必须**先 `skill trellis-start → task.py create → prd/design/implement → task.py start → trellis-check → task.py archive → add_session.py`，否则视为违规。
- 未建任务不得 `edit/write/bash` 改文件、不得 `sqlite3 UPDATE` 改库、不得 `docker` 重建。
- 当前线上管理员：`admin / admin`（`2026-08-30` 已落库 `pbkdf2:sha256:600000$83f88b...`，`must_change:true（首次登录需改密）`），后续改密必先建 Trellis 任务并 `ask_user_question`。
- **版本基线**：`0.0.1` 即稳定基线（`118M 3驱动`，`ghcr.io/zhemed/litepan:0.0.1` 已推，`git tag v0.0.1`），后续 `0.0.2` 递增（`fix`→`0.0.2`，`feat`→`0.0.3`），**不跳 `1.0.0`**，仅用户显式说“发 `1.0`”时再 `1.0.0`。
- 关联任务：`08-30-remove-cache-organize` 回归 `2f1b620` 已追认为 `08-30-fix-coverextract-nil`，`journal-1.md Session8` 为证。
