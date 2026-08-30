# Extract LitePan-own custom parts

## Goal

`LitePan-own`（`https://github.com/zhemed/LitePan-own`，`main` 099cbc9）已锁定为只读（`maint: 0.0.8`），不允许 `push` 或直接修改；需将其在 `Ponphil/LitePan v0.5.1-beta` 上游基础上新增的**自定义部分**完整提取到本工作区（`/root/LitePan`）的只读对比目录，供后续 `zhemed/LitePan`（`three-drivers`）按需移植，且**不向 `LitePan-own` 提交任何变更**。

## Requirements

- **只读**：`git -C /root/LitePan/LitePan-own` 与 `git -C /root/LitePan-own` 均 `fetch` 即可，不 `push`、`commit`、`amend` 到 `zhemed/LitePan-own`；`git config` 不改 `origin`
- **提取范围（以 `git log --oneline` 的 9 个自有 commits 为准）**：
  - `9e2d344 feat: add local_upload automation with full hash incremental + frontend` 及其后续 `283b875 / 2b44969 / ... / 099cbc9` 的增量
  - 关键文件：`internal/domain/automation.go`（`AutomationActionLocalUpload`）、`internal/automation/service_run.go`（`runLocalUpload/fileHash/loadLocalUploadState/saveLocalUploadState`）、`service_validate.go`、`internal/settings/registry.go`（`KeyLocalUpload*`）、`web/src/components/admin/AutomationPanel.vue`（`本地上传` 动作 UI）、`drivers/115_Open/upload.go` 等的 `OSS 512M` 优化
- **落盘**：在 `/root/LitePan` 内创建 `_extracted/LitePan-own-custom/`（或 `LitePan-own-custom/`），`gitignore` 需忽略该目录（`_extracted/`），内含：
  - `README_CUSTOM.md`（自定义功能总览表）
  - `diff/`（`git -C LitePan-own diff Ponphil/main..HEAD --stat` 与 `git format-patch` 归档）
  - `patches/`（按 commit 的 `*.patch`）
  - `files/`（按原路径拷贝的关键自定义文件快照，供 `diff` 对比）
- **溯源**：`README_CUSTOM.md` 需列 `9` 个自有 commits 的 `sha / subject / files`，并标注每个自定义与上游的边界（`automation local_upload` vs `crosstransfer` 等已删功能无关）

## Constraints

- 不向 `zhemed/LitePan-own` 的 `origin` 推送任何分支/标签；`git push` 目标仅为 `zhemed/LitePan`（本仓库）
- 提取目录 `_extracted/` 必须被 `/.gitignore` 的 `/_extracted/` 忽略，避免误提交到 `zhemed/LitePan`
- 提取过程 `GOWORK=off go vet` 不需通过（仅拷贝），但 `ls _extracted` 需可验证

## Acceptance Criteria

- [ ] `ls /root/LitePan/_extracted/LitePan-own-custom/README_CUSTOM.md` 存在且含 `9e2d344` 等 9 个 `sha` 的表格
- [ ] `ls /root/LitePan/_extracted/LitePan-own-custom/patches/ | wc -l` == 9（或 `git format-patch` 产物的数量）
- [ ] `ls /root/LitePan/_extracted/LitePan-own-custom/files/internal/automation/service_run.go` 存在且 `grep -c "runLocalUpload" | wc -l` >=1
- [ ] `git -C /root/LitePan status --porcelain | grep _extracted` == 0（已被 `.gitignore` 忽略）
- [ ] `git -C /root/LitePan/LitePan-own status` 仍 `Clean`，`git -C /root/LitePan-own status` 亦 `Clean`（未被污染）

## Notes

- `LitePan-own` 的 `README.md` 已说明“只加一个功能：本地自动上传”，提取后可供 `zhemed/LitePan` 的 `three-drivers` 评估是否移植 `local_upload` 到 `LocalUpload`（当前 `aux-keep-upload` 仅保留 `LocalUpload` 映射上传，非 `local_upload automation`）
