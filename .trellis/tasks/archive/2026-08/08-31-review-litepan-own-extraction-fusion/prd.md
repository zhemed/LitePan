# 审查 LitePan-own 自定义提取与融合

## Goal
审计工作区 `LitePan-own`（锁定，不允许提交/修改）到 `_extracted/LitePan-own-custom` 的提取完整性，以及到 `zhemed/LitePan` 的融合完整性，产出差异清单与修复建议。

## Background
- 用户在 `LitePan` 工作区内 `LitePan-own/` 目录拉取 `https://github.com/zhemed/LitePan-own`（嵌套 clone，已加入 `.gitignore`），要求锁定不提交。
- 已执行 `08-30-extract-litepan-own-custom` 提取到 `_extracted/LitePan-own-custom/`（含 `README_CUSTOM.md`、`diff/stat.diff`、9 个 `patches/`、`files/internal/automation/service_run.go` 等），并有 `08-30-adapt-litepan-own-localupload` 将 `local_upload` 适配到 `LitePan`。
- 需验证两阶段无遗漏：① 提取是否覆盖 `LitePan-own` 相对 `Ponphil/LitePan` 基线的全部自定义（9 commits `9e2d344..099cbc9`）；② 融合是否覆盖提取物到当前 `LitePan`（automation 定时/状态/前端等）。

## Requirements
- **提取审计**：以 `LitePan-own` 相对基线 `4c160d9`（或 `Ponphil/LitePan` 同步点）的 `git log --oneline` 与 `git diff --stat` 为权威，核对 `_extracted/LitePan-own-custom/README_CUSTOM.md`、`.trellis/tasks/archive/...` 记录、`patches/` 数量与内容、`stat.diff`/`full.diff` 是否一致，检查是否遗漏未提交变更或二进制/大文件。
- **融合审计**：以 `_extracted` 内容为输入，核对 `LitePan` 现有 `internal/automation/service*.go`（`AutomationActionLocalUpload`、`Options`、`runLocalUpload`、`fileHash`/`loadSaveState` 等）、`internal/domain/automation.go`、`web/src/api/automation.ts`、`web/src/components/admin/AutomationPanel.vue` 等是否等价实现，是否处理了与精简后（STRM/share/cache/organize 清理）冲突、`.trellis/spec` 是否同步。
- **产出**：`review.md`（或追加到任务目录）包含 ① 提取清单与差异、② 融合清单与差异、③ 结论（已完整/缺口）与建议（含是否需补丁）。

## Constraints
- 只读审计，不改 `LitePan-own/` 与业务代码；写操作仅限任务目录 `review.md` / 报告。
- 所有操作在 `task.py start` 后（`in_progress`）执行，遵循 `AGENTS.md 项目强制规则`。

## Acceptance Criteria
- [ ] 生成 `review.md`：含提取项逐条核对表（9 commits、文件清单、patch 完整性）
- [ ] 生成 `review.md`：含融合项逐条核对表（backend 字段/逻辑、前端 API/组件、状态文件行为 `B mode/sha256/512M` 等）
- [ ] 明确结论：提取是否完整、融合是否完整，若有缺口列出待办（文件:行）
- [ ] 报告路径告知用户，若缺口可一键创建后续修复任务
