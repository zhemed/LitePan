# Implementation Plan: Create GitHub repo LitePan and sync

## Overview

按 `design.md` 先建库后推。

## Phase 1: 建库

- [ ] 1.1 `gh repo view zhemed/LitePan --json nameWithOwner,visibility 2>&1 | head` 确认 404 则 `gh repo create zhemed/LitePan --public --description "LitePan - 精简版（仅115/189/LocalFs）" --confirm`，200 则跳过
  - `verify: gh api repos/zhemed/LitePan --jq .html_url` 返回 `https://github.com/zhemed/LitePan`

## Phase 2: Remote 与推送

- [ ] 2.1 `git remote -v` 与 `git log --oneline origin/main..HEAD | head -n 20` 展示待推
  - `verify: git rev-parse HEAD` 与 `git rev-parse origin/main` 差
- [ ] 2.2 `git remote` 处理（二选一，默认新增 `github`）：
  - `git remote add github https://github.com/zhemed/LitePan.git` 或 `git remote rename origin upstream && git remote add origin https://github.com/zhemed/LitePan.git`
  - `verify: git remote -v | grep zhemed/LitePan`
- [ ] 2.3 `git push -u github main`（或 `origin main`）含 `70ee23d` 等
  - `verify: git ls-remote https://github.com/zhemed/LitePan.git HEAD | cut -f1` == `git rev-parse HEAD`

## Phase 3: 验证

- [ ] 3.1 `gh repo view zhemed/LitePan --json nameWithOwner,visibility,url` → `zhemed/LitePan PUBLIC`
- [ ] 3.2 `curl -I https://github.com/zhemed/LitePan 2>&1 | head -n 5` → `200`
- [ ] 3.3 `gh api repos/zhemed/LitePan/commits/main --jq .sha` == `git rev-parse HEAD`

## Phase 4: Commit & Archive

- [ ] 4.1 本任务为 infra，无代码 `vet`，但 `git status` 仍 `Clean`
- [ ] 4.2 `task.py archive 08-30-create-github-repo-litepan --skip-branch-validation && git add .trellis/tasks/archive/... && git commit`
- [ ] 4.3 `add_session.py --commit <push-sha>`

## Rollback

- `gh repo delete zhemed/LitePan --confirm`（需用户二次确认）
