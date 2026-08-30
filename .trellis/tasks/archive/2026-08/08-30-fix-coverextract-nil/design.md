# Design: Fix coverextract nil panic

## Overview

`1bcfac8` 删除 `cacheretention/mediaorganize` 时误删 `wire_http.go` 中的 `coverextract` 装配，使 `Deps.CoverExtract==nil` → `Handler.listCoverExtractFiles:157` 对 `nil` 的 `s.mu.Lock()` 触发 `panic`，经 `chi Recoverer` 转 `500`。本设计仅恢复 18 行装配，不动 `coverextract` 业务。

## Boundaries

| Layer | Keep | Restore |
|---|---|---|
| `internal/app/wire_services.go` | 已无 `coverextract`（服务在 `wire_http` 创建，非 `servicesBundle`），保持现状 | 无 |
| `internal/app/wire_http.go` | `backupRestoreSvc / spaceCleanupSvc / router` | 恢复 `coverextract.New`、两处 `CoverExtract*` 回调、`router CoverExtract` |
| `internal/coverextract` | `service.go 1006 行 + rules` 均保留 | 无 |
| `internal/api/router.go` | `Deps.CoverExtract *coverextract.Service` 与 `Handler.coverExtract` 均保留，未删 | 无 |
| `internal/spacecleanup` | `Options{CoverExtractStats/ClearCoverExtract}` 字段保留 | 恢复传参 |

## Data Flow

```
Before (broken):
wireHTTPServer → Deps{CoverExtract: nil} → Handler.coverExtract==nil → List() panic → 500

After (fixed):
wireHTTPServer: coverExtractSvc := coverextract.New(DataDir, ListenAddr, svc.files, svc.playback)
           → spacecleanup.New(CoverExtractStats: coverExtractSvc.Stats, ClearCoverExtract: coverExtractSvc.ClearWithStats)
           → api.NewRouter(Deps{CoverExtract: coverExtractSvc})
           → Handler.listCoverExtractFiles → s.mu.Lock() OK → 200 []
```

## Compatibility

- 无 DB/Config 变更，`data/litepan.db` 的 `coverextract` 会话为内存 `map`，`Stats/ClearWithStats` 仅内存
- 关联：`internal/mediaorganize/rules.IsSeasonDirName` 被 `coverextract/service.go:188` 引用，故 `mediaorganize/rules` 目录在 `1bcfac8` 中故意保留，本修复不触 `rules`
- 预览：`playback.Service` 与 `file.Service` 已在 `servicesBundle` 中，`coverextract` 仅消费二者，无循环依赖

## Tradeoffs

- **恢复 vs 删除**：本可彻底删除 `coverextract`（随整理一起下线），但用户未要求，且 `spacecleanup` 仍依赖其统计，故选择恢复而非借机删除
- **最小 diff**：仅 `wire_http.go` 18 行，`go vet` 与 `docker build` 均 `CACHED` 层，仅重建 `go build 13.5s`

## Rollout / Rollback

- 单 `fix` 提交 `2f1b620` 已推，本任务追认；`git revert 2f1b620` 即回到 panic 状态，可快速回滚验证
- 容器：`docker build -t litepan-go:nocache-organize -t litepan-go:nocache-organize-fix` 并 `docker run` 验证

## Risks

- `ffmpeg` 未安装时 `runtime` 返回 `ready:false` 属预期，非回归
- 若后续决定彻底下线封面功能，需另建 `remove-coverextract` 任务并同步删 `router / spacecleanup` 依赖
