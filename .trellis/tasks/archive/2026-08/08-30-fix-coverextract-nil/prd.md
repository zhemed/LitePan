# Fix coverextract nil panic after cache organize removal

## Goal

修复 `08-30-remove-cache-organize`（`1bcfac8`）误删 `internal/app/wire_http.go` 中 `coverextract` 装配导致的运行时 `nil pointer dereference`，使 `Image 500（197x50 sha256:53c5c...）`、`GET /api/admin/tools/cover-extract/files` 与 `runtime` 等封面相关接口不再 `panic`，恢复 `coverextract` 正常可用。

## Requirements

- **复现确认**：
  - `docker logs litepan` 含 `panic: runtime error: invalid memory address or nil pointer dereference` 在 `coverextract.(*Service).List:220` 与 `Runtime:292`，经 `Recoverer` 转 `500`
  - 前端任意封面/清理页触发 `GET /api/admin/tools/cover-extract/files` 或 `runtime` 即 `500`，`197x50` 小图 `请求失败(500)`
- **修复**：
  - 恢复 `internal/app/wire_http.go` 中 `coverextract.New(Options{DataDir, ListenAddr, Files: svc.files, Playback: svc.playback})` 的创建
  - 恢复 `spacecleanup.New` 的 `CoverExtractStats: coverExtractSvc.Stats()` 与 `ClearCoverExtract: coverExtractSvc.ClearWithStats()`
  - 恢复 `api.NewRouter` 的 `Deps{CoverExtract: coverExtractSvc}` 注入
  - 保留 `internal/coverextract` 目录（`rules` 依赖 `mediaorganize/rules`，但服务本身独立于 `mediaorganize`）
- **不回归**：
  - 已删除的 `cacheretention / mediaorganize / classifyorganize / aiorganize` 仍保持删除，`grep -r -i "cacheretention|mediaorganize" 0` 不变
  - `internal/cache` 核心与 `mediaorganize/rules` 仍保留

## Constraints

- 仅改 `internal/app/wire_http.go`（18 行），不改 `internal/coverextract/service.go` 本身
- 关联 `issue` 为 `08-30-remove-cache-organize` 的回归，非新功能，故为 `P1` hotfix

## Acceptance Criteria

- [ ] `docker logs litepan | grep -i panic` 无 `coverextract` panic
- [ ] `POST /api/auth/login -d "username=admin&password=123456" → 200`
- [ ] `GET /api/admin/tools/cover-extract/files -b cookie → 200 {"files":[]}`
- [ ] `GET /api/admin/tools/cover-extract/runtime -b cookie → 200 {"ready":false,"ffmpeg":"", "error":"未找到 ffmpeg"}`
- [ ] `GET /api/admin/tools/cover-extract/images/{id}` 对不存在 id 返回 `404` 而非 `500`
- [ ] `GOWORK=off go vet ./...` PASS，`go build` PASS，`docker build -t litepan-go:nocache-organize` PASS，`/api/health 200`

## Notes

- 根因：`wire_http.go` 的 `coverextract.New` 紧邻 `mediaorganize` 代码，批量删除时被误带走；`go vet` 无法捕获运行时 `nil`，需运行时 `curl` 覆盖
- 该修复已在 `2f1b620` 中直推，本任务为追认 Trellis 痕迹，原 `2f1b620` 的 `diff` 即本任务的 `implement` 产出
