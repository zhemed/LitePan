# 审计报告：添加执行动作 5 项能否仅保留本地上传

**任务**：`08-31-audit-automation-keep-localupload-only`  
**时间**：2026-08-31  
**截图**：`de6ccfb 512x440` 5 项 `刷新目录 / 执行整理任务 / 延迟等待 / 本地上传 / Emby全局刷库`  
**基线**：`LitePan main 66337e3 v0.0.3`（`domain 3枚 delay/emby/local_upload`，`service_run 2分支`）

---

## 一、映射

| # | 截图项 | 颜色/图标 | 对应 `domain.AutomationAction*` | 当前后端状态 | 前端 `AutomationPanel ACTION_DEFINITIONS` |
|---|---|---|---|---|---|
| 1 | **刷新目录** | 绿刷 `刷新目录 自动清理后续任务涉及账号的目录缓存` | `cache_clear`（历史 `CacheClear`） | **已删**：`domain` 无常量，`service_run` 无 `runCacheClear`，`service_validate` 无分支 | **仍存**：`ACTION_DEFINITIONS.刷新目录` `657 行` 定义完整（含 `label/optionLabel`） |
| 2 | **执行整理任务** | 蓝整理 `生成计划并执行整理…` | `organize`（历史 `Organize`） | **已删**：`domain` 无常量，`service_run` 无 `runOrganize`，`validate` 无 | **仍存**：`667 行` `label:整理任务 optionLabel:执行整理任务` |
| 3 | **延迟等待** | 橙钟 `等待一段时间后…` | `delay` | **存活**：`domain delay` + `service_run runDelay` + `validate case delay` | **存活**：`680 行` |
| 4 | **本地上传** | 蓝上传 `将服务器映射目录…` | `local_upload` | **存活**：`local_upload` 完整 `fileHash/B mode/云检查` | **存活**：`690 行` **拟保留** |
| 5 | **Emby全局刷库** | 紫 `通知 Emby 扫描…` | `emby_refresh` | **半存活**：`domain emby_refresh` 常量仍在，但 `service_run executeAction` **无分支** → 运行时 `“动作类型不支持”`；`service.go` 仍 `emby_configs: []` 空桩；`validate` 无 `emby` 校验；`normalizeInput` 仅允许 `delay/local_upload` 2 种 → `emby_refresh` 提交时 `存在不支持的动作` | **仍存**：`727 行` `Emby刷库` 完整（含 `fetchEmbyLibraries` `Api emby.ts 53`） |

**来源**：`internal/domain/automation.go:22` `internal/automation/service_run.go:113` `service_validate.go:15` `web/.../AutomationPanel.vue:655` `internal/automation/service.go:278`

---

## 二、依赖与外溢

### 2.1 刷新目录 `cache_clear`
- **后端**：历史 `runCacheClear` 依赖 `collectCacheClearTargets` 需后置 `organize/strm` 任务，已随 `organize/strm` 精简而逻辑空转；`service_run` 已删，`mediaorganize` 已不被自动化调用。
- **前端**：`AutomationPanel` 单独定义，无 `import` 外溢。
- **配置/DB**：无独立 `KeyCache*` 关联（`KeyCache*` 为 `internal/cache` 全局缓存，与 `cache_clear` 动作无关）；`automation_rules` 若含 `cache_clear` 则现 `normalizeInput` 会拒。
- **外溢**：`internal/cache Service` 本体为 `文件列表缓存`（`metadata cache`），与 `cache_clear` 动作解耦，**保留缓存服务不影响删除动作**。

### 2.2 执行整理任务 `organize`
- **后端**：`internal/mediaorganize` 服务仍存在（`ls internal/mediaorganize` 有 `service.go` 等），但 `service_run runOrganize` 已删，`wire_services.go` 已无 `organize` 字段，`domain` 已无常量，自动化已不调用它；`mediaorganize` 仍可通过 `任务管理 → 目录整理` 独立使用（若前端仍有），但自动化链路已断。
- **前端**：`AutomationPanel` `organizeTaskOptions` 来自 `api/automation.ts:70 organize_tasks`，`findTaskLabel organize` 等。
- **DB**：`media_organize_tasks` 表仍存，与自动化解耦。
- **外溢**：彻底移除需决定是否连 `mediaorganize` 目录整理功能一起删（用户 `08-30-remove-cache-organize` 已部分移除 `cacheretention`，但 `mediaorganize` 目录仍在）。

### 2.3 延迟等待 `delay`
- **后端**：`runDelay` 纯 `time.Sleep`，`clampInt 1-86400`，无外部依赖，`service.go` 无字段。
- **前端**：`ACTION_DEFINITIONS.延迟` 单独。
- **外溢**：纯工具动作，无 DB/配置，无外溢；若 `LocalUpload` 单独定时 `daily 02:00` 则无需 `delay` 串联。

### 2.4 本地上传 `local_upload`（保留）
- **后端**：依赖 `filesvc.Service` `settings KeyLocalUploadMappings` `upload.Manager` `dataDir` `fileHash/loadSaveState/B mode/云检查` 4 环节完整，已 `0.0.3` 融合。
- **前端**：依赖 `AppSelect multiple`、`fetchLocalMappings /api/admin/tools/local-upload/config`。
- **外溢**：无周期性依赖 `delay/emby`。

### 2.5 Emby全局刷库 `emby_refresh`
- **后端**：`domain` 常量残留但 `service_run` 无 handler，`service_validate` 无校验，`normalizeInput` 白名单仅 `delay/local_upload` 会拒，`internal/embyproxy`（若存在）与 `internal/api/emby.go` `web/api/emby.ts` 仍存，`automation/service.go:278 emby_configs: []` 为空桩。
- **前端**：`AutomationPanel 591 import fetchEmbyLibraries` `web/api/emby.ts 40 GET /admin/emby/configs + 53 libraries + 57 refresh`，`SystemSettings` 仍可能管理 Emby 配置（需核对 `settings.KeyEmby*` 是否已删——此前 `08-30-remove-aux` 已删 `KeyEmby*` 但前端仍有）。
- **外溢**：`Emby` 为 `PMS` 通知，独立于 `LocalUpload`，无强依赖；彻底移除需删 `api/emby` + `embyproxy` + `settings emby` 若仍残留。

### 2.6 公共依赖

- **`internal/automation/service.go`**：当前 `278 emby_configs: []` 仅为 `ListOptions` 空桩，若仅留 `local_upload` 可删。
- **`internal/cache`**：与 5 项动作解耦，**建议保留**（非动作相关，是 `file.List` 加速）。
- **`web/api/automation.ts:70`**：`organize_tasks/emply_configs` 字段若删 `organize/emby` 可同步清，但当前 `service.go ListOptions` 仍返回空数组，删除后需同步改 `api` DTO。

---

## 三、逐项“能否仅留本地上传”判定

| # | 项 | 能否只留 `local_upload` 而删它？ | 安全彻底移除？ | 风险/备注 |
|---|---|---|---|---|
| 1 | **刷新目录** | **能** | **安全** | 后端已删仅前端残留 `657行`，删 `ACTION_DEFINITIONS.刷新目录` + 模板 `select` 选项即可；无 DB/配置残留；`validate` 已拒，零运行中依赖 |
| 2 | **执行整理任务** | **能** | **安全（自动化层面）**，若连 `mediaorganize` 功能一起删则需额外删 `internal/mediaorganize` 目录与前端 `目录整理` 页 | 当前 `organize` 仅自动化动作已删，后端 `mediaorganize` 仍孤立；彻底删需决定是否保留“目录整理”独立功能（建议随动作一起删，因用户已 `精简 cache/organize`） |
| 3 | **延迟等待** | **能** | **安全** | 纯 `sleep`，删 `domain delay` + `service_run runDelay` + `validate case delay` + `frontend 680行` 即可；`LocalUpload` 定时无需 `delay` |
| 4 | **本地上传** | **保留** | — | 唯一保留，`10.0.0.99 3映射` 场景核心 |
| 5 | **Emby全局刷库** | **能** | **安全**，但需多文件联动 | 后端 `domain emby_refresh` + `service.go emby_configs 空桩` + `api/emby.ts` + `internal/api/emby*` + `frontend Emby刷库 727行 + fetchEmbyLibraries` 关联；当前后端已半残，彻底删反而修复“前端可选但后端不支持”的不一致 |

**总体**：**能**。5 项中 4 项与 `local_upload` 无强依赖，可安全彻底移除；当前代码已有 `cache_clear/organize` 后端残缺，正好通过彻底移除修复前后端不一致。

---

## 四、若仅留 `local_upload` 需删除清单（彻底）

### 4.1 后端 `backend`

| 文件 | 操作 | 匹配 |
|---|---|---|
| `internal/domain/automation.go:22-23` | 删 `AutomationActionDelay` `AutomationActionEmbyRefresh`，仅留 `local_upload` | `const AutomationAction*` |
| `internal/automation/service_run.go:113-115` | 删 `case delay` | `case domain.AutomationActionDelay` |
| `internal/automation/service_run.go:122` | 删 `runDelay` 函数体 `122-132` | `func runDelay` |
| `internal/automation/service_run.go:666` | 后续 `actionDisplayName` 中 `case delay` | `case domain.AutomationActionDelay:` |
| `internal/automation/service_validate.go:15` | 删 `case delay` 分支 | `case domain.AutomationActionDelay:` |
| `internal/automation/service_validate.go:119` | `switch Type` 白名单 `case delay, local_upload` → 仅 `local_upload` | `case domain.AutomationActionDelay, domain.AutomationActionLocalUpload:` |
| `internal/automation/service.go:278` | 删 `emby_configs: make(...)` 空桩 | `"emby_configs":` |
| `internal/automation/service_test.go:228` | 若存在 `emby_refresh` 用例，改或删 | `emby_refresh` |
| **可选彻底** | `internal/mediaorganize/` 整目录（若连整理功能一起） | `ls internal/mediaorganize` |
| **可选彻底** | `internal/embyproxy/` `internal/api/emby*` `web/src/api/emby.ts`（若 Emby 全删） | `ls internal/embyproxy` `ls web/src/api/emby.ts` |
| `internal/settings/registry.go` | 已删 `KeyEmby*` 无需再动；`KeyCache*` 保留（全局缓存） | `grep KeyEmby` 已 0 |

### 4.2 前端 `web`

| 文件 | 操作 |
|---|---|
| `web/src/components/admin/AutomationPanel.vue:655-742` | 删 `ACTION_DEFINITIONS` 中 `刷新目录 657` `整理任务 667` `延迟 680` `Emby刷库 727` 4 块，仅留 `690 本地上传` + `747 unknown`；删 `actionTypeOptions` 自动收敛；删 `591 import fetchEmbyLibraries`；删 `500-540 Emby配置模板`；删 `456-457 整理任务模板`；删 `657-664 刷新目录模板`；删 `1117/1326/1393-1440 Emby逻辑` |
| `web/src/api/automation.ts:70` | 删 `organize_tasks` `emby_configs` 字段（若对应后端删） |
| `web/src/api/emby.ts` | 删整文件（若 Emby 全删） |

### 4.3 配置/DB

- `settings registry` 已无 `Emby` 键，无需再动。
- `automation_rules.actions_json` 历史若含 `delay/emby/organize/cache_clear`，`normalizeInput` 已会 `存在不支持的动作` 拒，需 `DB` 手动清或 `ValidateRule` 容错（当前已 `default:不支持` 会拒，彻底移除后旧规则需迁移或报错）。

---

## 五、建议

| 方案 | 操作 | 适用 |
|---|---|---|
| **A 激进（用户问的“只保留本地上传”）** | 删 `4 项` 全部（含 `delay`），按 §4 清单 `backend 7 处 + frontend 10 处` 彻底移除，`go vet + vue-tsc` 双过，`docker build 118MB` 不变 | **推荐**：`LocalUpload daily 02:00` 无需 `delay`，前后端彻底一致 |
| **B 保守** | 仅删 `刷新目录/整理/Emby` 3 项，**保留 `delay`**（通用等待） | 若担心某联动仍需 `延迟` 串联 |
| **C 不删** | 保留 5 项，前后端不一致继续（前端可选但后端 `不支持`） | **不推荐**：当前 `emby_refresh` 已半残，用户会遇到“选了但执行失败” |

**风险**：A 方案下历史 `automation_rules` 若含旧类型需 `UPDATE` 清理或首次保存时 `validate` 报错提示用户重配；无其他外溢。

---

## 附：取证命令

```bash
grep -rn "AutomationAction" internal/domain/automation.go
grep -n "case domain.AutomationAction" internal/automation/service_run.go internal/automation/service_validate.go
grep -n "ACTION_DEFINITIONS" web/src/components/admin/AutomationPanel.vue -A 5 | head -n 80
ls internal/mediaorganize 2>&1 | head; ls internal/embyproxy 2>&1 | head
grep -rn "KeyEmby\|KeyCache" internal/settings/registry.go | head
```
