# Implementation Plan: Keep only 115 Tianyi LocalFs drivers

## Overview

按 `design.md` 自底向上删除 8 驱动目录并收敛 `all.go`。

## Phase 1: 删除驱动目录

- [ ] 1.1 `rm -rf drivers/123_Open drivers/139Cloud drivers/Baidu_Open drivers/Guangya drivers/OneDrive drivers/OpenList drivers/Quark drivers/WebDAV`
  - `verify: ls drivers/` → `115_Open 189Cloud LocalFs all.go template`（5 项）
- [ ] 1.2 `edit drivers/all.go` 为：
  ```go
  package drivers
  import (
    _ "litepan/drivers/115_Open"
    _ "litepan/drivers/189Cloud"
    _ "litepan/drivers/LocalFs"
  )
  ```
  - `verify: cat drivers/all.go | grep import -A5` 仅 3 行

## Phase 2: 关联清理（可选）

- [ ] 2.1 `grep -r "123_Open|139Cloud|Baidu_Open|Guangya|OneDrive|OpenList|Quark|WebDAV" --include="*.md" README.md docs/ 2>&1 | head` 若提及 8 驱动则删或标注已移除（若无则跳过）
  - `verify: grep -r "Baidu_Open" --include="*.md" | wc -l` == 0 或仅在“已移除”备注

## Phase 3: Sweep & Verification

- [ ] 3.1 `ls drivers/ | wc -l` == 5
- [ ] 3.2 `grep -r "drivers/123" --include="*.go" | wc -l` == 0
- [ ] 3.3 `GOWORK=off go vet ./...` PASS
- [ ] 3.4 `GOWORK=off go build -trimpath -ldflags="-s -w" -o /tmp/litepan ./cmd/litepan` PASS
- [ ] 3.5 `cd web && npm run type-check` PASS, `npm run build` PASS
- [ ] 3.6 `docker build -t litepan-go:three-drivers .` PASS, `docker run -d -p 5218:5211 -v ./data:/app/data litepan-go:three-drivers` + `curl /api/health 200` + `curl -b cookie /api/admin/drivers | jq length` == 3

## Phase 4: Commit & Archive

- [ ] 4.1 `git add -A && git restore --staged .trellis/tasks/08-30-keep-only-three-drivers && git commit -m "refactor(drivers): keep only 115 189 LocalFs"`
- [ ] 4.2 `task.py archive 08-30-keep-only-three-drivers --skip-branch-validation && git add .trellis/tasks/archive/... && git commit`
- [ ] 4.3 `add_session.py --commit <hash>`

## Rollback

- `git revert <refactor commit>` 恢复 8 驱动

## Validation Commands

```bash
ls drivers/
cat drivers/all.go
grep -r "Quark" --include="*.go" drivers/ | wc -l  # 0
GOWORK=off go vet ./...
GOWORK=off go build -o /tmp/litepan ./cmd/litepan && echo ok
docker build -t litepan-go:three-drivers . && echo ok
curl -b /tmp/c.txt http://127.0.0.1:5211/api/admin/drivers | jq
```
