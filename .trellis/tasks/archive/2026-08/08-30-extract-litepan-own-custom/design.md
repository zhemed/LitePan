# Design: Extract LitePan-own custom parts

## Overview

`LitePan-own` 在 `Ponphil/LitePan` 上以 `9` 个 `maint/feat` commits 叠加了“本地自动上传”全量 hash 增量能力，需在不污染其 `origin` 的前提下，将 `diff` 与关键文件快照提取到本仓库的 `_extracted/` 隔离目录，供 `zhemed/LitePan` 后续评估移植。

## Boundaries

| 源 | 提取 | 不提取 |
|---|---|---|
| `LitePan-own` 的 `9` 个自有 commits（`9e2d344` 起） | `diff --stat`、`format-patch`、`files/` 快照 | `Ponphil` 上游的 `~200` 个 commits |
| `internal/automation` | `service_run.go` 的 `runLocalUpload/fileHash`、`service_validate.go` 的 `LocalUpload` 分支 | `crosstransfer` 等已删功能 |
| `internal/domain` | `automation.go` 的 `AutomationActionLocalUpload` | 其他 `domain` |
| `web` | `AutomationPanel.vue` 的 `本地上传` 动作 UI | `CrossDriveTransfer` 等已删 |
| `drivers/115_Open` | `upload.go` 的 `OSS 512M` 优化（如有） | 其他驱动 |

## Data Flow

```
Ponphil/LitePan v0.5.1-beta (4c160d9)
  ↓ 9 commits (zhemed/LitePan-own main 099cbc9)
LitePan-own custom
  ↓ git diff Ponphil..HEAD + git format-patch + cp files/
zhemed/LitePan/_extracted/LitePan-own-custom/
  ├── README_CUSTOM.md (9 commits 表)
  ├── diff/stat.patch
  ├── patches/0001-*.patch (9)
  └── files/internal/automation/service_run.go 等快照
```

## Compatibility

- `_extracted/` 需 `/.gitignore` 的 `/_extracted/` 忽略，避免 `zhemed/LitePan` 的 `git status` 污染与误 `push`
- 提取目录为只读快照，不参与 `go vet` / `vite build`

## Tradeoffs

- **全量 `format-patch` vs 仅 `diff`**：全量保留 9 个 `patch` 便于 `git am` 按需挑选移植，`diff` 仅总览
- **`_extracted/` vs `LitePan-own/` 嵌套**：已有 `LitePan/LitePan-own` 嵌套克隆（`08-30-clone-litepan-own`），本次提取到 `_extracted/LitePan-own-custom` 与之并存，前者为完整克隆，后者为自定义切片

## Rollout / Rollback

- 单 `git add .gitignore` 的 `/_extracted/` 忽略，无代码提交；`rm -rf _extracted/` 即回滚
- 若需移植，`cp _extracted/.../files/internal/automation/service_run.go` 到 `zhemed/LitePan` 后按 `three-drivers` 现状适配

## File Map

1. `/.gitignore` 追加 `/_extracted/`
2. `_extracted/LitePan-own-custom/README_CUSTOM.md`
3. `_extracted/LitePan-own-custom/diff/`
4. `_extracted/LitePan-own-custom/patches/`
5. `_extracted/LitePan-own-custom/files/`
