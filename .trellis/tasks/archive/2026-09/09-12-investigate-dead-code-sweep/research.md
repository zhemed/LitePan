# 死代码全面排查报告（09-12-investigate-dead-code-sweep）

> 口径：**只读**（零代码改动；未触碰生产机、未改容器/DB）
> 工具：`go list -deps`、`golang.org/x/tools/cmd/deadcode`（RTA 可达性）、五维自定义脚本（Go 符号引用计数 / 前端引用计数 / package.json 依赖 / 构建产物引用 / 路由使用）
> 判据：**可达性优先**——「文件存在」不算活着；生产不可达再分「仅测试可达」与「连测试都不可达（真死）」

---

## 1. 结论摘要

| 类别 | 结果 |
|---|---|
| **真死代码（可清理）** | **3 个包**（`internal/proxybase` 163 行、`internal/taskauth` 128+100 行、`pkg/strutil` 13 行）+ **24 个函数**（跨 20 个文件）+ **12 个前端文件**（2,237 行） |
| 仅测试可达（生产死） | 8 个函数（httpx OAuth 代理 5 个 + domain 暂停原因 2 个 + upload 离线 handoff 1 个）与 `drivers/template`（438 行，测试专用骨架） |
| 干净项（无需清理） | DB 表结构（11 张表全在用，已删功能表由迁移 0022 清除）、API 路由（无已删功能残留）、驱动注册表、设置项枚举、构建产物（109 个 asset 全部被引用）、前端依赖（30 个依赖 0 未用） |
| 疑似未使用端点 | 1 个：`POST /accounts/{id}/refresh-auth`（后端有 handler、前端与测试均无调用；可能留给人工/外部调用） |
| 不建议清理 | 迁移历史文件（9 个属于已删功能，见 §7）、`drivers/template`（测试骨架）、`upload_tasks` 的跨盘遗留列（仍被本机上传路径使用） |

**一句话**：精简总体做得干净（DB/路由/注册表/构建产物/依赖零残留），残留集中在**三个孤儿包**与**少量已删功能的辅助函数/前端文件**。

---

## 2. 方法（可复现）

```bash
# Go 包级：仓库内包 vs 生产依赖图
GOWORK=off go list ./... | sed 's|^litepan/||' | sort > /tmp/all_pkgs.txt
GOWORK=off go list -deps ./cmd/litepan | grep '^litepan/' | sed 's|^litepan/||' | sort -u > /tmp/reach_pkgs.txt
comm -23 /tmp/all_pkgs.txt /tmp/reach_pkgs.txt

# Go 符号级：生产视角 + 含测试视角（差集 = 仅测试可达）
deadcode ./cmd/litepan      > /tmp/deadcode_prod.txt
deadcode -test ./...        > /tmp/deadcode_test.txt

# 前端：basename/stem 引用计数（vue/ts/js/css；排除入口与声明文件）
# 依赖：package.json ∩ web/**  文本检索
# 构建产物：解压 .gz 后检查 asset 互引 + index.html
# 路由：router.go 叶子路径 × web/src 文本检索
```

**防误报**：`deadcode` 用 RTA（含接口分派），比 grep 引用计数可靠；凡 grep 与 deadcode 冲突的，逐个核查调用链是否本身是死链（例：`uploadutil.HashMD5` 的"5 处引用"实为 `domain.HashMD5` 常量同名，真实调用 0 处）。

---

## 3. Go 包级：3 个零消费者包 + 1 个测试专用包

| 包 | 规模 | 引用情况（证据） | 结论 |
|---|---|---|---|
| `internal/proxybase` | 2 文件 / 163 行 | `grep -rn "litepan/internal/proxybase\""` → **0**；`go list -deps ./cmd/litepan` 不含 | **真死**（曾服务已删的 embyproxy/fnosproxy） |
| `internal/taskauth` | 1 文件 / 128 行 + 测试 100 行 | 同上 → **0** import | **真死**（曾服务已删的 STRM/缓存保持任务） |
| `pkg/strutil` | 1 文件 / 13 行 | 同上 → **0** import | **真死** |
| `drivers/template` | 6 文件 / 438 行 | 仅 `internal/auth/oauth_integration_test.go:14` 空导入（测试专用） | **保留**：驱动骨架/测试替身，属有意保留的脚手架 |

> 合计**可清理 304 行非测试代码 + 100 行测试**（≥3 个包），另有 438 行测试专用骨架（保留）。

---

## 4. Go 符号级：32 个生产不可达（24 真死 / 8 仅测试可达）

### 4.1 真死（连测试都不可达，24 个）

| 文件 | 符号 | 归属的已删功能 / 说明 |
|---|---|---|
| `internal/backuprestore/maintenance.go` | `Service.OrphanTempCandidates`、`Service.CleanupOrphanTempCandidates`、`Service.orphanTempCandidatesLocked` | 孤儿临时文件清理，无任何入口调用（3 个） |
| `internal/apikey/service.go` | `Service.Validate`、`Service.ValidateTask` | STRM Token / 任务校验（已删 STRM） |
| `internal/automation/helpers.go` | `normalizePath`、`ternaryStatus` | 已删动作（整理/STRM）的辅助函数 |
| `internal/upload/manager.go` | `offlineHandoffGroupID` | 离线下载交接（已删功能） |
| `internal/upload/lifecycle.go` | `progressForBytes` | 被 `calcProgress` 取代后的遗留 |
| `internal/upload/target_dir.go` | `joinUploadDisplayPath` | 未使用 |
| `internal/upload/worker.go` | `Manager.taskLocalPath` | 未使用 |
| `internal/driver/delay.go` | `WithExtraAPIDelay` | 自动化/STRM 的额外限速上下文 |
| `internal/driver/upload.go` | `LocalUploadEpochMillis` | 未使用 |
| `internal/driver/uploadutil/hash.go` | `HashMD5` | 与 `domain.HashMD5` 常量同名误导，真实调用 0 |
| `internal/driver/uploadutil/progress.go` | `ReadProgress.Read` | 进度读取器被驱动各自实现取代 |
| `internal/driver/uploadutil/resume.go` | `UploadedBytesByPartKeys` | 未使用 |
| `internal/file/name_align.go` | `asAlignInt` | 精简改写（1bcfac8）后的遗留 |
| `internal/settings/registry.go` | `stringSpec` | 已删设置项的类型构造函数 |
| `internal/notification/service.go` | `Service.DeleteByRef` | 服务层方法未被调用（仓储层同名方法仍活） |
| `internal/api/sse.go` | `streamSSEMessages` | 未使用 |
| `drivers/189Cloud/transport.go` | `Driver.signedForm` | 签名表单路径未用（上传已走加密/OSS 流程） |
| `pkg/jsonvalue/flexible_string.go` | `FlexibleString.UnmarshalJSON` | 单点未用 |
| `pkg/security/origin.go` | `RequestBaseURL` | 单点未用 |
| `pkg/speedsmoother/smoother.go` | `NewDefault` | 单点未用 |

### 4.2 仅测试可达（生产死、测试在用；8 个）

| 文件 | 符号 | 说明 |
|---|---|---|
| `internal/httpx/oauth.go` | `PostOAuthProxyJSON`、`OAuthProxyHTTPError`、`OAuthProxyResponseError` | OAuth 代理链路（123/百度/OneDrive 已删）；现仅 `drivers/template` 与自身测试使用 |
| `internal/httpx/do_json.go` | `DoJSON` | 同上（被 template 引用） |
| `internal/httpx/envelope.go` | `ParseDataEnvelope` | 同上 |
| `internal/domain/pause_reason.go` | `PauseReason.AutoResumable`、`ValidAutoPauseReason` | 已删任务类型的暂停原因（仅测试覆盖） |
| `internal/upload/manager.go` | `OfflineHandoffClientID` | 离线交接（仅测试覆盖） |

> 处理建议：与 `drivers/template` 一并决策——若保留 template 作为驱动开发脚手架，则 `httpx` OAuth 一套**也应保留**（否则模板编译失败）；若不保留 template，可整链清理（约 5 个符号 + 438 行）。

---

## 5. 前端：12 个零引用文件（2,237 行）、依赖 0 未用

判定方法：对 `web/src` 全部 223 个 `.vue/.ts/.js/.css` 文件做 basename/stem 引用计数（排除 `main.ts`、`*.d.ts`），再对全部 12 个命中项做**二次全仓检索**（含 `web/index.html`、配置、`package.json`）复核，全部为 0。

| 文件 | 行数 | 归属 |
|---|---|---|
| `web/src/api/spaceCleanup.ts` | 71 | 垃圾清理（已删模块） |
| `web/src/components/admin/AdminStartupBanner.vue` | 23 | 启动横幅（未接线） |
| `web/src/components/admin/CacheRuntimeStats.vue` | 86 | 缓存统计面板（未接线） |
| `web/src/components/admin/CacheSettingsPanel.vue` | 141 | 缓存设置面板（未接线） |
| `web/src/components/admin/FuseManagement.vue` | 1071 | FUSE 管理面板（未接线） |
| `web/src/components/admin/WebDAVSettings.vue` | 122 | WebDAV 设置面板（未接线） |
| `web/src/composables/useConditionalPolling.ts` | 46 | 工具（未用） |
| `web/src/composables/useLiveElapsedClock.ts` | 49 | 工具（未用） |
| `web/src/composables/useStartupCountdown.ts` | 54 | 关闭倒计时（未接线） |
| `web/src/composables/useVirtualPosterWall.ts` | 170 | 海报墙（已删功能） |
| `web/src/utils/coverPoster.ts` | 356 | 海报/封面（已删功能） |
| `web/src/utils/tmdbHit.ts` | 48 | TMDB（已删功能） |
| **合计** | **2,237** | |

**依赖**：`web/package.json` 30 个依赖（含 dev），对 `web/**` 文本检索 → **0 个未使用**。
**注意**：其中 FUSE/WebDAV/缓存相关面板虽"未接线"，但**后端功能仍在**（`internal/fusemount`、`internal/share/fuse`、`internal/cache` 有路由与实现，见 §6）——它们的"无引用"说明当前后台 UI 不再提供这些面板，属**UI 精简遗留**，清理前建议确认是否要保留未接线 UI 以便将来恢复。

---

## 6. API / 注册表 / 配置：无已删功能残留，1 个疑似未使用端点

| 检查 | 证据 | 结果 |
|---|---|---|
| 路由残留 | `grep -nE "strm\|cross\|quark\|emby\|fnos\|scrape\|organize\|retention" internal/api/router.go` | 仅命中日志清理路由（在用的日志功能）→ **干净** |
| 驱动注册 | `drivers/all.go` 仅 `115_Open`/`189Cloud`/`LocalFs`；`grep "Baidu\|OneDrive\|Quark\|Guangya\|123_Open\|139Cloud"` 非测试代码 → 空 | **干净** |
| 自动化动作 | `internal/domain/automation.go` 动作常量只剩 `AutomationActionLocalUpload` | **干净** |
| 设置项 | `internal/settings/registry.go` 键值全部为在用设置（缓存/FUSE/认证/日志/上传保留/本机上传） | **干净** |
| 路由使用 | 87 条叶子路由 × `web/src` 文本检索 | 86 条命中；**1 条无前端调用**：`POST /accounts/{id}/refresh-auth`（`internal/api/auth_refresh.go`，后端 handler 正常，全仓除路由注册外无引用） |

> `refresh-auth` 处置建议：低优先。若确认仅供人工/外部调用，建议在文档标注而非删除（删除会改变已公开的 HTTP 契约）。

---

## 7. DB：结构干净；迁移历史保留；跨盘列仍被使用

- 本地库（只读）共 **11 张表**：`account_auth_states, api_keys, automation_rules, automation_runs, cloud_accounts, configs, fuse_mounts, notifications, schema_migrations, sqlite_sequence, upload_tasks` —— **无已删功能表**（STRM / media_organize / cache_retention / offline_download / emby_proxy / quarktv 均由迁移 `0022_drop_removed_feature_tables.sql` 清除）。
- 迁移文件 23 个，其中 9 个属于已删功能（`0003/0004_strm`、`0007_media_organize`、`0010/0011_cache_retention`、`0014/0016_offline_download`、`0017_strm_dir_cache`、`0018_emby_proxy`、`0019_strm_group_dir`、`0020/0021_quarktv`）→ **不建议删除**：迁移是追加式历史，删除会破坏已部署实例的升级路径与 `schema_migrations` 一致性。
- `upload_tasks` 35 列中含跨盘遗留名（`source_type/source_account_id/…/phase/downloaded_bytes/cleanup_local_*`）→ 经核对**仍被本机上传路径使用**（本地文件清理模式、phase 推进等），**不是死列**。

---

## 8. 构建产物：干净

`internal/api/web/assets` 共 109 个文件（102 个 `.gz` + 7 个不可压缩：4 个 woff2 字体 + 3 个 <100B 的 js/css）。逐文件解压后检查 `index.html` 与其它 chunk 的互引 → **0 个未被引用**。（`web/scripts/compress-build.mjs` 对不可压缩扩展名跳过压缩，属设计行为。）

---

## 9. 清理建议清单（按优先级）

| 优先级 | 范围 | 规模 | 风险 | 说明 |
|---|---|---|---|---|
| **P1** | 删 3 个死包：`internal/proxybase`、`internal/taskauth`、`pkg/strutil` | 304 行 + 100 行测试 | 极低（零 import，删后 `go build/test` 可验证） | 与 `mediaorganize` 同性质的孤儿包 |
| **P1** | 删 24 个真死函数（§4.1 跨 20 个文件） | ~150–250 行 | 低（`deadcode` + 编译/测试复核；注意 `deadcode` 判据含 RTA，删后重跑确认） | 建议按文件分组批量删，配一次全量门禁 |
| **P2** | 前端 12 个零引用文件（§5） | 2,237 行 | 低-中：其中 **FUSE/WebDAV/缓存三块后端功能仍在**，删 UI 前需确认"不再提供这些后台面板"是有意决定 | 建议先删**明确属于已删功能**的 4 个（`spaceCleanup.ts`、`useVirtualPosterWall.ts`、`coverPoster.ts`、`tmdbHit.ts`），其余 8 个单独确认 |
| **P3** | `drivers/template` + `httpx` OAuth 链路（8 个仅测试可达符号） | 438 + ~150 行 | 中：template 是驱动开发脚手架与 auth 集成测试替身，删了会失去新驱动模板 | **默认保留**；若确定不再新增驱动，可整链清理 |
| **P4** | `POST /accounts/{id}/refresh-auth` | 1 端点 | 低（对外契约） | 建议文档标注"预留端点"，不删 |
| 不作处理 | 迁移历史 9 文件、`upload_tasks` 跨盘列 | — | — | 见 §7 |

**建议的执行方式**：P1 两项合并为一个清理任务（纯删除 + 门禁 + 发版），P2 按"已删功能 → 疑似未接线 UI"两步拆分，P3/P4 待用户决策。

---

## 10. 误报排除与未做项

**已排除的误报**：
1. `uploadutil.HashMD5` 的 grep"5 处引用"实为 `domain.HashMD5` 常量（同名），真实调用 0；
2. `drivers/template` 不在生产依赖图但被 `internal/auth` 集成测试使用 → 不是死包；
3. `notification.Service.DeleteByRef` 与仓储层 `DeleteByRef` 同名——服务层方法死、仓储层方法活（`internal/fusemount/service.go:723` 用的是仓储）；
4. 前端 `WebDAVSettings/FuseManagement` 等 5 个面板"未引用"≠"功能已删"——后端路由与实现仍在（§6），属 UI 未接线；
5. `deadcode` 的 12 条测试文件局部桩（`automation/service_test.go` 的 `apiKeyRepo.*`）是接口实现桩，**不是死代码**（本次未计入 24 条）。

**未做 / 待定**：
- 后端**端点使用率**的完整核对（前端路径拼装方式多样，首版脚本因前缀重建错误作废；改用叶子路径匹配后仅剩 1 条，但动态拼接路径无法穷尽）→ 标注"待定"。
- 未评估"删除后二进制体积变化"（预期 ≈0：死代码本不入二进制，0.0.40 已实测二进制逐字节相同）。
- 未做 `web/dist`、`docs/`、`install-docker.sh`、`.github/` 等非源码路径的残留排查（用户未要求；如需可追加）。

---

## 11. 复现命令（关键）

```bash
GOWORK=off go list ./... > /tmp/a; GOWORK=off go list -deps ./cmd/litepan > /tmp/b   # 包级差集
deadcode ./cmd/litepan; deadcode -test ./...                                          # 符号级
grep -rn "litepan/internal/proxybase\"" --include="*.go" .                            # 0 引用
grep -rln "spaceCleanup" web/src                                                       # 0 引用
grep -nE "strm|cross|quark|emby|fnos|organize|retention" internal/api/router.go        # 无残留
python3 - <<'PY' ... sqlite3(file:data/litepan.db?mode=ro) ... PY                      # 表结构只读核对
```
