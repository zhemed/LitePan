# Design: 移植上游三批修复

## 目标与不变量

**目标**：把调研报告 §7 的三批修复按**本方架构适配**移植进来，每批都能独立验证、独立回滚。

**不变量**

1. **不引入上游的已删依赖**：`drivers/template`、`internal/strm`、`internal/embyproxy`、`internal/fnosproxy`、FUSE 相关一律不碰（上游删了 template，直接 apply 会连带删除）。
2. 每批结束后 `go build`/`vet`/`test` 全绿；三批之间无相互依赖（可单独 revert）。
3. **行为等价类改动必须给出等价性证据**（C2/C3 依赖 `FormatAPITime` 零值行为与 `WriteHeader` 幂等语义 —— 两者已实测/读码确认）。
4. 不动部署面与数据：不改 `:5211`、不改 `data/`、不加迁移、不 bump 版本。

## 关键决策

### KD1 为什么三批合并为一个任务、但分三个 Phase

三批来自同一份调研报告、同一批上游提交，且共用同一套质量门与验证手段；拆成三个任务会让"又一次全量质量门+归档"的仪式成本三倍于实际改动量（合计约 300~350 行）。因此一个任务、三个 Phase，**每个 Phase 自带验证点与回滚点**，任一 Phase 失败可只回滚该批。

### KD2 移植方式：craft patch，不 apply

上游 patch 的上下文与本方**大面积不同**（本方删了 STRM/媒体整理/缓存整理/FUSE/API 秘钥/跨盘/分享，且改了版本单一来源、上传批次化、冷却语义、非特权部署）。
→ 逐项按本方代码改，**用上游 diff 当规格说明**（"应该达到什么行为"），不用它当补丁。

### KD3 A 组（115 驱动）的验证策略

驱动逻辑无法在本机对真实 115 账号端到端验证（无凭据、且不该触碰用户账号）。
→ 采用**单元测试 + 纯函数化**：
- A1/A2/A3 的核心逻辑（分页收集、路径拼装、段名消毒）都以 `fullListPageFetcher` / `buildDirPath` / `pathSegmentName` 形式可注入或纯函数 → 用**假 fetch 函数**构造"空页 → 非空页"、"Count 大于实际"、"目录名含 `/`"、"越出挂载根"等场景断言。
- A4 的缓存治理可在 `drivers/115_Open` 包内用**小上限常量**构造（或直接断言 `clear` 行为与删除后清理）。
- A5 只断言"不再每次调用 `regexp.MustCompile`"（改为读包级变量）——用读码 + `grep` 证据。

### KD4 B1（adminauth 缓存）的失效语义必须写清

缓存一旦引入，"外部改库"就会陈旧。本方所有配置写入路径都经过服务层（UI/API/备份恢复），**没有外部写入者**；且 `setConfig` 会同步回写内存。
→ 代码注释必须写明该前提；`All()` 读取失败时**不置 `configLoaded`**，下次再试（上游同款）。

### KD5 C5（删僵尸键）的迁移口径

先用 `git show 924e4c75` 核对**上游是否也清了 `internal/store/backup.go` 清洗名单与 `internal/backuprestore/service.go` 清空名单**：
- 若上游保留 → 本方保留（旧库/旧备份里的残留键无害，且清空名单是"清历史残留"的安全网）
- 若上游也清 → 本方同步清
**本轮不新增迁移**：删代码后旧库里的 `admin_temp_password_last_reset_at` 变成孤儿键，无害；如需彻底清库由用户决定是否随下次迁移处理（调研报告 §3 G4 已记录）。

## 风险与处置

| 风险 | 处置 |
|---|---|
| A2 改动 `buildDirPath` 签名影响其它调用方 | 先 `grep -rn buildDirPath` 找全调用点（本方与上游一致：仅 `ResolveDirPath`）|
| A1 行为变化导致"清单模式"在正常网络下误报不完整 | 保留"必须先连续空页"语义；`expectedCount` 只在终局比较；异常路径返回明确错误而非部分数据 |
| B1 缓存与备份恢复冲突（恢复期间直接写库）| 恢复流程走 `store`/`backuprestore`，不经 `adminauth`；缓存只在进程内、且 `setConfig` 回写；补一条"`All()` 失败不置 loaded"的用例 |
| B2 把日志降到 Debug 导致排查时看不到 | 这是上游刻意的取舍（诊断信息不该刷用户日志）；`LITEPAN_LOG_LEVEL=debug` 可恢复可见性；README/spec 若提到该日志再同步 |
| B3 删 `statLocalFile` 影响其它调用点 | 先 `grep -rn statLocalFile`（预期仅 `local_upload.go` 内）|
| C 组"行为等价"判断错 | C2 依赖 `FormatAPITime` 零值语义（已读码确认返回 `""`）；C3 依赖 `WriteHeader` 幂等（已读码确认已提交则原样返回）|
| 误跟着上游删东西 | 三批都不碰 `drivers/template`；提交前 `git status` 逐文件核对 |

## Rollout / Rollback

**Rollout**：Phase A（115）→ 门禁 → Phase B（P1）→ 门禁 → Phase C（P2）→ 门禁 → 端到端临时实例验证 → 归档。

**Rollback**：三批在**一个提交**里落地（若用户希望可拆三个提交）→ `git revert` 即全回；若拆提交，可**逐批 revert**（B1 缓存是最需要留意的单点）。不涉及数据与部署，无迁移。

## File Map

| 路径 | 动作 |
|---|---|
| `drivers/115_Open/full_list.go` | A1/A2/A3 |
| `drivers/115_Open/ops.go` | A4 |
| `drivers/115_Open/full_list_test.go` | A1/A2/A3 用例 |
| `drivers/189Cloud/ops.go` | A5 |
| `internal/adminauth/service.go`（+ 测试）| B1、C5 |
| `internal/api/slow_dashboard_log.go`（+ 测试）、`internal/api/router.go` | B2 |
| `internal/api/local_upload.go`（+ 测试）| B3 |
| `internal/api/oauth.go` | B4 |
| `Dockerfile` | C1 |
| `internal/api/{accounts.go,files.go}` | C2 |
| `internal/api/commit_writer.go` | C3 |
| `cmd/litepan/main.go` | C4 |

**零改动（越界即失败）**：`web/src/**`、`internal/api/web/**`（本批不改前端，embed 不重建）、`go.mod`/`go.sum`、`.golangci.yml`、`internal/buildinfo/version.go`、`drivers/template/**`、`data/**`。
