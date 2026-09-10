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

## 项目强制规则（用户于 2026-08-30 明确；2026-09-10 加强为"严格执行，不得忽略"）

> **所有操作必须调用 trellis；强制序列缺一不可、顺序不得颠倒**

- 本项目为 `trellis init --dsh -u zhemed` 托管，**任何**代码、配置、数据（`data/litepan.db` / `secret.key`）、镜像（`docker build/run`）、前端（`web`）、规则文件等写操作，**必须先完成到 `task.py start`**，否则视为违规。

### 强制序列与每步完成判据

| 步 | 命令 | 完成判据（不满足不得进入下一步） |
|---|---|---|
| 1 | `skill trellis-start` | 已执行 `get_context.py`（含 `--mode phase`/`--mode packages`）并读过相关 `.trellis/spec/*/index.md` |
| 2 | `task.py create <name>` | 任务目录生成 |
| 3 | 写 `prd.md` | 需求/约束/验收标准齐全，**无 TBD 占位**；复杂任务另有 `design.md` + `implement.md`；**必须早于 start** |
| 4 | `task.py set-scope <name> <scope>` | 已标注轻/复杂（决定第 3 步产物要求） |
| 5 | `task.py start <name>` | 门禁 `flow_gate.py pre-start <task>` 通过 |
| 6 | 实施 + 项目质量门 | `go vet ./...`、`go test ./...`、`cd web && npm run type-check && npm run build` 全绿 |
| 7 | `skill trellis-check` | 清单全过：lint/类型/测试、测试覆盖（新功能单测、修 bug 回归测试）、**spec 同步**、范围纪律、跨层一致性 |
| 8 | `task.py archive <name>` | 门禁 `flow_gate.py pre-archive <task>` 通过（验收项全勾选 + check 标记存在） |
| 9 | `add_session.py` | journal 记录完成；**会话号一律以 journal 为准，不得自报** |

### 四条硬性禁止（2026-09-10 违规复盘结论）

- ✗ **先实施后补 PRD**（PRD 沦为事后验收记录 = 规划门失效）
- ✗ **跳过 `trellis-check`**，用自建的 vet/test/build 三连顶替质量门
- ✗ **跳过 `trellis-start` 的上下文/spec 索引载入**
- ✗ **未标注 scope、复杂任务缺 `design.md`+`implement.md` 就 start**

### 门禁命令（`.trellis/scripts/flow_gate.py`）

- start 前：`python3 ./.trellis/scripts/flow_gate.py pre-start <task>`
- 质量门通过后：`python3 ./.trellis/scripts/flow_gate.py mark-check <task>`
- archive 前：`python3 ./.trellis/scripts/flow_gate.py pre-archive <task>`
- 显式跳过（须写明原因，记录在案）：加 `--force "<原因>"`

### 其余既有约定（保留）

- 未建任务不得 `edit/write/bash` 改文件、不得 `sqlite3 UPDATE` 改库、不得 `docker` 重建。
- 当前线上管理员：`admin / 123456`（`2026-09-09` 经用户授权重置落库 `pbkdf2:sha256:600000$aa9764...`，重置前备份 `/tmp/litepan-backup-20260909-201827.db`；此前 `08-30` 记录的 `admin/admin` 已作废），后续改密必先建 Trellis 任务并 `ask_user_question`。
- **API 登录注意**：`POST /api/auth/login` 仅接受 **form 表单编码**（`curl -d 'username=admin&password=123456'`），发 JSON 体会被静默解析为空用户名而报"用户名或密码错误"（日志特征 `username=""`）。
- **版本基线**：`0.0.1` 即稳定基线（`118M 3驱动`，`ghcr.io/zhemed/litepan:0.0.1` 已推，`git tag v0.0.1`），后续 `0.0.2` 递增（`fix`→`0.0.2`，`feat`→`0.0.3`），**不跳 `1.0.0`**，仅用户显式说“发 `1.0`”时再 `1.0.0`。
- 关联任务：`08-30-remove-cache-organize` 回归 `2f1b620` 已追认为 `08-30-fix-coverextract-nil`，`journal-1.md Session8` 为证。
