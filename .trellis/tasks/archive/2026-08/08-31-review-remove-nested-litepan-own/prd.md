# 审查是否可移除 LitePan-own 嵌套

## Goal
只读审查 `/root/LitePan/LitePan-own`（`nested`）是否可安全 `rm -rf`，明确依赖、替代与风险，给出可/不可移除结论与清理建议。

## Background
- 用户已明确 `工作区只有 /root/LitePan`，`另外的是另外的`，`LitePan-own` 有两处：`sibling /root/LitePan-own` 与 `nested /root/LitePan/LitePan-own`（后者在 `/.gitignore` 中 `LitePan-own/` 已忽略，不入库）。
- `nested` 的创建为 `08-30-clone-litepan-own` 任务（`mkdir -p LitePan-own && git clone ...`），后续 `extract` 与 `adapt` 均以 `_extracted/` 为交付物，不应长期依赖 `nested` 本体。
- 需核实是否有 `trellis task`、`_extracted` 生成、`docker`、`grep`、`path` 引用仍指向 `nested`。

## Requirements
- **现状**：`ls -ld /root/LitePan/LitePan-own` `_extracted` `sibling` 的存在性、大小、`git remote -v`、`git status`、`du -sh`、`/.gitignore` 命中情况。
- **依赖**：`grep -R "LitePan-own" --exclude-dir=.git --exclude-dir=node_modules` 在 `LitePan` 工作区内的所有引用（`docs`、`tasks`、`scripts`、`review.md` 等），区分“历史记录提及”与“运行时依赖”。
- **替代**：确认 `sibling /root/LitePan-own` 是否可 100% 替代 `nested`（`git log 93616d6..099cbc9` 一致、`remote` 同、`_extracted` 已静态化无需再 `git diff`）。
- **产出**：`review.md` 含 ① 现状清单 ② 依赖清单 ③ 可/不可移除结论 ④ 若可则给出 `rm -rf` 命令与 `.gitignore` 是否保留。

## Constraints
- 只读审查，不实际 `rm -rf`；写仅限任务目录 `review.md`。
- 遵循 `AGENTS.md 项目强制规则`：`task.py start` 后执行。

## Acceptance Criteria
- [ ] `review.md` 存在，含三表（现状、依赖、替代）与明确结论（可/不可）
- [ ] 每项有来源（`ls/grep/git` 输出）与风险评估
- [ ] 若结论为可，给出精确清理命令与验证步骤
