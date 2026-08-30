# Implementation Plan: Fix coverextract nil panic

## Overview

追认已在 `2f1b620` 中完成的 18 行恢复，需补齐 Trellis 产出与验证。

## Phase 1: Restore wiring (wire_http.go)

- [x] 1.1 `internal/app/wire_http.go:1` 恢复 `import "litepan/internal/coverextract"`
- [x] 1.2 `wireHTTPServer` 中 `backupRestoreSvc` 后插入 `coverExtractSvc, err := coverextract.New(Options{DataDir: cfg.DataDir, ListenAddr: cfg.ListenAddr, Files: svc.files, Playback: svc.playback})`
- [x] 1.3 `spacecleanup.New` 的 `Options` 补 `CoverExtractStats: coverExtractSvc.Stats()` 与 `ClearCoverExtract: coverExtractSvc.ClearWithStats()`
- [x] 1.4 `api.NewRouter` 的 `Deps` 补 `CoverExtract: coverExtractSvc`

## Phase 2: Verification

- [x] 2.1 `GOWORK=off go vet ./...` PASS
- [x] 2.2 `GOWORK=off go build -o /tmp/litepan-fix ./cmd/litepan` PASS 39M
- [x] 2.3 `cd web && npm run type-check` PASS, `npm run build` PASS 109 files
- [x] 2.4 `docker build -t litepan-go:nocache-organize-fix .` PASS 125MB (go build 13.5s)
- [x] 2.5 `docker run -d --name litepan -p 5211:5211 -v ./data:/app/data litepan-go:nocache-organize-fix` → `docker logs` 无 panic
- [x] 2.6 `POST /api/auth/login -d "username=admin&password=123456" → 200`
- [x] 2.7 `GET /api/admin/tools/cover-extract/files -b cookie → 200 {"files":[]}`
- [x] 2.8 `GET /api/admin/tools/cover-extract/runtime -b cookie → 200 {"ready":false,...}`
- [x] 2.9 `curl -i /api/admin/tools/cover-extract/images/{dummy} -b cookie → 404` 非 500

## Phase 3: Trellis closure

- [ ] 3.1 `task.py start`（将 `planning → in_progress`，实现已在 `2f1b620` 完成，本步为追认）
- [ ] 3.2 `task.py archive --skip-branch-validation` 与 `git add .trellis/tasks/archive/...`
- [ ] 3.3 `add_session.py --commit 2f1b620` 记录 Session 8

## Rollback

- `git revert 2f1b620` 回到 panic，可 `docker build` 复现 `500`

## Validation Commands

```bash
GOWORK=off go vet ./...
GOWORK=off go build -o /tmp/litepan ./cmd/litepan
curl -s -b /tmp/c.txt http://127.0.0.1:5211/api/admin/tools/cover-extract/files | grep files
docker logs litepan --tail 20 | grep -i panic
```
