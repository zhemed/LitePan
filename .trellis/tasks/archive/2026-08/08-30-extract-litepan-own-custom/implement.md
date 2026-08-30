# Implementation Plan: Extract LitePan-own custom parts

## Overview

按 `design.md` 将 `LitePan-own` 的 9 个自有 commits 的 `diff` 与关键文件快照提取到 `_extracted/`。

## Phase 1: 隔离目录与忽略

- [ ] 1.1 `mkdir -p _extracted/LitePan-own-custom/{diff,patches,files}`
- [ ] 1.2 `echo "/_extracted/" >> .gitignore`（若已存在则跳过）
  - `verify: grep -c "_extracted" .gitignore` >=1 且 `git -C . status --porcelain | grep _extracted` == 0

## Phase 2: 溯源（9 commits）

- [ ] 2.1 `git -C /root/LitePan/LitePan-own log --oneline Ponphil/main..HEAD` 或 `git -C /root/LitePan-own log --oneline 9e2d344^..HEAD` 列 9 个 `sha`
  - `verify: git -C ... log --oneline | wc -l` == 9
- [ ] 2.2 `git -C /root/LitePan/LitePan-own diff --stat Ponphil/main..HEAD > _extracted/LitePan-own-custom/diff/stat.diff`（若无 Ponphil remote 则用 `HEAD~9..HEAD`）
- [ ] 2.3 `git -C /root/LitePan/LitePan-own format-patch --stdout HEAD~9..HEAD > _extracted/LitePan-own-custom/patches/combined.patch` 且 `git format-patch -o _extracted/.../patches HEAD~9..HEAD`
  - `verify: ls _extracted/.../patches | wc -l` == 9

## Phase 3: 关键文件快照

- [ ] 3.1 `mkdir -p _extracted/.../files/internal/automation` 等
- [ ] 3.2 `cp /root/LitePan/LitePan-own/internal/domain/automation.go _extracted/.../files/internal/domain/`
- [ ] 3.3 `cp /root/LitePan/LitePan-own/internal/automation/service_run.go _extracted/.../files/internal/automation/`
- [ ] 3.4 `cp /root/LitePan/LitePan-own/internal/automation/service_validate.go _extracted/.../files/internal/automation/`
- [ ] 3.5 `cp /root/LitePan/LitePan-own/web/src/components/admin/AutomationPanel.vue _extracted/.../files/web/src/components/admin/`（若存在）
  - `verify: grep -c "runLocalUpload" _extracted/.../files/internal/automation/service_run.go` >=1

## Phase 4: 总览文档

- [ ] 4.1 `cat > _extracted/LitePan-own-custom/README_CUSTOM.md <<'EOF'` 含表格：`sha / subject / files` 的 9 行，说明 `LitePan-own` 仅加 `本地自动上传` 全量 hash 增量，与 `zhemed/LitePan` 的 `LocalUpload`（映射上传）不同
  - `verify: cat _extracted/.../README_CUSTOM.md | head -n 30`

## Phase 5: Sweep & Verification

- [ ] 5.1 `ls _extracted/LitePan-own-custom/README_CUSTOM.md` 存在
- [ ] 5.2 `ls _extracted/.../patches | wc -l` == 9
- [ ] 5.3 `grep -c "runLocalUpload" _extracted/.../files/internal/automation/service_run.go` >=1
- [ ] 5.4 `git -C /root/LitePan status --porcelain | grep _extracted` == 0（已忽略）
- [ ] 5.5 `git -C /root/LitePan/LitePan-own status` 仍 `Clean` 且 `git -C /root/LitePan-own status` 亦 `Clean`

## Phase 6: Commit & Archive

- [ ] 6.1 `git add .gitignore && git commit -m "chore: ignore _extracted for LitePan-own custom extraction"`
- [ ] 6.2 `task.py archive ... --skip-branch-validation && git add ... && git commit`
- [ ] 6.3 `add_session.py --commit <hash>`

## Rollback

- `rm -rf _extracted/` + `git checkout HEAD -- .gitignore`
