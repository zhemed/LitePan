# Implementation Plan: Correct install-docker hosting to repo and update README

## Overview

按 `design.md` 将 `89 行` 修复版提交到仓库根并更新 `README`。

## Phase 1: 仓库提交

- [ ] 1.1 `cp /tmp/install-docker-fixed.sh ./install-docker.sh && chmod +x ./install-docker.sh && wc -l ./install-docker.sh && bash -n ./install-docker.sh && echo ok`
  - `verify: ls -lh ./install-docker.sh && grep -c "signed-by" ./install-docker.sh >=1`
- [ ] 1.2 `git add install-docker.sh && git commit -m "feat: add install-docker.sh for zhemed/LitePan (from new-api-own fixed)"`
  - `verify: git diff --cached --stat`

## Phase 2: README 更新

- [ ] 2.1 `README.md` 在 `## 快速开始` 前或后新增 `### 一键安装 Docker` 小节：
  ```markdown
  ### 一键安装 Docker（Debian/Ubuntu）
  ```bash
  curl -fsSL https://raw.githubusercontent.com/zhemed/LitePan/main/install-docker.sh | bash
  # → Docker 29.7.2 + Compose v5.4.0（已 hold）
  ```
  ```
  - `verify: grep -c "raw.githubusercontent.com/zhemed/LitePan/main/install-docker.sh" README.md >=1`

## Phase 3: Sweep & Verification

- [ ] 3.1 `grep -c "new-api-own" README.md` ==0（除 `WARNING` 外）
- [ ] 3.2 `curl -fsSL https://raw.githubusercontent.com/zhemed/LitePan/main/install-docker.sh | head -n 5`（`git push` 后）
- [ ] 3.3 `GOWORK=off go vet ./...` PASS
- [ ] 3.4 `git status --porcelain` 仅 `README.md` + `install-docker.sh`

## Phase 4: Commit & Archive

- [ ] 4.1 `git add README.md && git commit -m "docs(readme): add one-click docker install via LitePan raw"`
- [ ] 4.2 `task.py archive 08-30-correct-install-docker-hosting --skip-branch-validation && git add ... && git commit`
- [ ] 4.3 `add_session.py --commit <hash>`

## Rollback

- `git rm install-docker.sh && git revert <docs commit>`
