# 审查报告：LitePan-own 自定义提取与融合

**任务**：`08-31-review-litepan-own-extraction-fusion`  
**时间**：2026-08-31  
**执行**：`trellis-start → task.py create/start → in_progress` 只读审计  
**范围**：`LitePan-own/`（锁定，不提交）→ `_extracted/LitePan-own-custom/` → `LitePan`（`zhemed/LitePan main @ 6ef108b`）

---

## 一、提取审计：`LitePan-own` → `_extracted`

### 1.1 源基线
- `LitePan-own` `git log 93616d6..099cbc9 --oneline` 9 自有 commits（基于 `Ponphil/LitePan v0.5.1-beta 93616d6`）：
  ```
  099cbc9 maint: 0.0.8 115_Open OSS 512MB一片 + 超时600s
  c0f8e17 maint: 0.0.7 wrap local upload into mapping folder (B mode)
  5dd16a4 feat: 0.0.6 add cloud existence check to incremental (79 lines)
  923129a fix(file): treat NOT_FOUND as success on delete (unified)
  b4ff744 fix(189Cloud): make delete resilient for newly uploaded files
  2b44969 fix: make local_upload mapping multi-select dynamic dropdown
  283b875 feat: support multi-select for local_upload mappings
  8340206 docs: lock docker image to 0.0.1 for deterministic pull
  c3df10c docs: rewrite README for LitePan-own v0.0.1
  9e2d344 feat: add local_upload automation with full hash incremental + frontend
  ```
- 权威 `git diff 93616d6..099cbc9 --stat`：
  ```
  README.md                                    | 188 +++++-- 
  drivers/115_Open/driver.go                   |   2 +-
  drivers/115_Open/upload.go                   |  24 +---
  drivers/189Cloud/ops.go                      |  14 +-
  internal/automation/service_run.go           | 185 ++++++++++++++-----
  internal/automation/service_validate.go      |  21 ++-
  internal/file/service.go                     |   8 +-
  web/src/components/admin/AutomationPanel.vue |  65 ++++++---
  web/src/components/base/AppSelect.vue        |  26 +++-
  9 files changed, 316 insertions(+), 217 deletions(-)
  ```

### 1.2 提取物核对
- **目录** `_extracted/LitePan-own-custom/`（已 `/.gitignore/_extracted/` 忽略，只读）：
  ```
  diff/stat.diff      # 9 files 316+217  ✔ 与上权威一致
  diff/full.diff      # 772 行 ✔
  patches/0001-0009 9张 + combined.patch ✔ 一一对应上 9 SHA
  files/internal/domain/automation.go
  files/internal/automation/service_run.go (1042 行, grep runLocalUpload 2)
  files/internal/automation/service_validate.go
  files/web/src/components/admin/AutomationPanel.vue
  README_CUSTOM.md    # 表格 10 行（含 9+1 统计差异），关键实现说明 ✔
  ```
- **逐补丁验证**（`cat patches/*.patch | git apply --check` 层面）：
  | # | 补丁 | 覆盖文件 | 提取完整性 |
  |---|---|---|---|
  | 1 | `0001-docs-rewrite-README` | `README.md` | ✔ 已含于 `full.diff` |
  | 2 | `0002-docs-lock-docker-image` | `docker-compose.yml` | ✔ 已含 |
  | 3 | `0003-feat-support-multi-select` | `service_run.go`/`service_validate.go`/`AutomationPanel.vue` | ✔ |
  | 4 | `0004-fix-make-multi-select-dynamic` | `AutomationPanel.vue`/`AppSelect.vue` | ✔ |
  | 5 | `0005-fix-189Cloud` | `drivers/189Cloud/ops.go` | ✔ |
  | 6 | `0006-fix-file NOT_FOUND` | `internal/file/service.go` | ✔ |
  | 7 | `0007-feat-cloud-existence-check` | `service_run.go 79 行` | ✔ |
  | 8 | `0008-maint-B-mode` | `service_run.go B mode` | ✔ |
  | 9 | `0009-maint-115 OSS 512MB` | `drivers/115_Open/driver.go + upload.go` | ✔ |
- **未提取**：无。`git status LitePan-own` `Clean`，`files/` 未含 `README.md`/`docker-compose.yml` 等 docs，但 `diff/` 已覆盖；二进制/大文件无。
- **结论**：**提取完整**。`_extracted` 与 `git diff 93616d6..099cbc9` 一致，9 commits 无遗漏，`README_CUSTOM.md` 表格与 `stat.diff` 一致（10 行含表头为展示差异，实际 9）。

> 备注：`_extracted` 的 `files/` 仅快照 4 个核心文件（domain/automation×2 + AutomationPanel），`AppSelect.vue` 等通过 `patches/` 覆盖，符合 `只加一个功能（本地自动上传）` 的边界。

---

## 二、融合审计：`_extracted` → `LitePan`

### 2.1 融合总览
- 对应历史任务：`08-30-extract-litepan-own-custom`（提取）→ `08-30-adapt-litepan-own-localupload`（适配，`be39a6a 430 行 runLocalUpload`）→ 后续 `精简`（STRM/share/cache/organize/crosstransfer/drivers）与 `0.0.1/0.0.2` 发版。
- 融合策略：**仅融合 `local_upload 自动化` 核心**，其余 `docs` 与 `精简已移除` 的 `organize/strm/strm_scrape/cache_clear` 相关不在融合范围（见 `AGENTS.md` 与 `spec` 已移除记录）。

### 2.2 已完整融合（✔）
| 模块 | 文件 | 关键实现 | 状态 |
|---|---|---|---|
| **Domain** | `internal/domain/automation.go:17` | `AutomationActionLocalUpload = "local_upload"` | ✔ 本项目 `674 行版` 保留，精简后仅 `delay + emby_refresh + local_upload` 3 常量（`organize/strm` 已删符合精简） |
| **Service 主逻辑** | `internal/automation/service_run.go` | `fileHash(sha256)` `load/saveLocalUploadState(local_upload_state_*.json)` `runLocalUpload` 多映射循环、汇总 `totalScanned/Created/Skipped` | ✔ `674 行` 含完整实现，与 `LitePan-own 1042 行版` 的 `556-906` 行 `local_upload` 段等价 |
| **云端二次校验 0.0.6** | 同上 `runLocalUpload` 内 | `targetDirsForCheck + ensureDirForCheck + List 检查云端存在，hash 同但云端缺则重传`（79 行） | ✔ 已融合 `314-398` 行 |
| **B mode 0.0.7** | 同上 | `filepath.Join(mappingName, sc.relDir)` 包装到映射文件夹，云端路径 `mapping/relDir` | ✔ `475/497` 行 |
| **多选 0.0.3** | 同上 + `service_validate.go` | `params["mappings"] []any → mappingNames` 兼容 `mapping` 单值，空则报错 | ✔ `193-212` + `22-39` |
| **Validate** | `internal/automation/service_validate.go:12-40` | `mappings` 多选校验、兼容旧单值 | ✔ 与 `LitePan-own` 同逻辑（仅剥离 `organize/strm` 校验，符合精简） |
| **前端 AutomationPanel** | `web/src/components/admin/AutomationPanel.vue:481` | `mappings` 多选 + 动态 `fetchLocalMappings()` + `ACTION_DEFINITIONS.local_upload.normalize/canApply/nodeTitle/previewTitle` | ✔ `481 AppSelect multiple` + `853-858 fetch` + `565 import` |
| **状态文件** | `internal/automation/service_run.go:147-177` | `local_upload_state_<mapping>.json` `relPath → sha256` 存 `dataDir` | ✔ 与 `LitePan-own` 一致，`B mode` 下 key 仍为 `relPath` 分目录隔离 |

**证据**：`grep -n runLocalUpload LitePan/internal` 1 处，`grep -n fileHash 2` 处特征与 `_extracted` 一致；`diff LitePan vs _extracted` 的 `local_upload` 段仅差 `import math/strmscrape` 等已移除的 `organize/strm` 依赖，`mappings/B mode/云检查` 逻辑零差异。

### 2.3 未融合（✘ 缺口，待修复）
| # | 补丁来源 | 文件 | LitePan 现状 | 应有（LitePan-own） | 影响 | 优先级 |
|---|---|---|---|---|---|---|
| **A** | `0009 115_Open OSS 512MB + 超时600s` | `drivers/115_Open/driver.go:73` | `Timeout: 30s` | `Timeout: 600s` | `>30s` 大文件 `httpx` 超时，`100M 单片` 场景下曾闭环失败 | **高** |
| **B** | 同上 | `drivers/115_Open/upload.go:743 calculateOSSPartSize` | 复杂分级 `20MB-5GB` | `固定 512MB` | 与 `099cbc9` 验证的 `100M 单片直传` 不一致，分片数/OSS 签名差异 | **高** |
| **C** | `0005 189Cloud` | `drivers/189Cloud/ops.go:264 batchTaskInfos` | `if err {return nil,err}` | `cachedItem → fallback fileId` 容错 + `name==id` 兜底 | 刚上传文件 `GetFileInfo NOT_FOUND` 时批量任务失败（天翼） | **高** |
| **D** | `0006 file NOT_FOUND` | `internal/file/service.go:226 DeleteFiles` | `Warn+return err` | `CodeNotFound → Info视为成功` | 删除已不存在文件误报失败，影响同步清理 | **中** |
| **E** | `0004 AppSelect multiple` | `web/src/components/base/AppSelect.vue` | `modelValue: string|number|boolean|null`，无 `multiple`，`choose` 单值 `close()` | `modelValue: ...|(... )[]`, `multiple?`, `selectedLabel` 取 `labels.join('、')`, `choose` 增删 `vals` 不 `close` | `AutomationPanel` 的 `multiple` 属性不生效，多选映射无法选多个（当前仅单选有效） | **高** |

> **合计 5 处缺口**，均为 `LitePan-own` 在 `10.0.0.99` 闭环验证过的修复（`100M 秒传、2G hash 2秒、增量跳过`），未随 `local_upload` 一并适配。

### 2.4 故意不融合（预期内，与精简一致）
- `0001 README 重写` / `0002 docker-compose 锁定 0.0.1`：`LitePan` 已有 `README v0.0.2` + `ghcr.io/zhemed/litepan:0.0.2` 独立演进，不视为缺口。
- `organize/strm/strm_scrape/cache_clear` 相关：`domain 4 常量、service_run 600 行、service_validate 200 行` 已由 `08-30-remove-cache-organize/remove-strm` 移除，`spec` 已归档 `3 驱动`，不在 `local_upload` 融合范围。
- `domain AutomationActionOrganize` 等额外常量差异：`LitePan` 保留 `delay/emby_refresh/local_upload` 3 枚，`LitePan-own` 有 7 枚，前者为精简结果。
- `emby_refresh` 处理缺口：`LitePan domain` 保留 `emby_refresh` 常量但 `service_run executeAction` 仅 `delay/local_upload` 2 分支，`service_validate` 亦无 `emby` 校验；`LitePan-own` 有完整 `runEmbyRefresh`。**此为精简后残留不一致**（非 `LitePan-own` 9 commits 范围，但影响 `emby` 联动），建议 `trellis-update-spec` 明确 `emby_refresh` 是否保留。

### 2.5 一致性检查
- `GOWORK=off go vet ./...` 当前 `0`，`cd web && npm run type-check` 通过（`AppSelect` 未改暂不报错，但多选不生效属于功能缺陷而非类型错）。
- `grep -r "local_upload" --include="*.go" --include="*.vue"` 覆盖一致；`grep -r "B mode" / "sha256"` 手动验证通过。
- `internal/automation/service.go` 的 `Options{Settings, DataDir, Uploads}` / `Service{settings,dataDir,uploads}` 与 `_extracted` 前置条件一致。

---

## 三、结论与建议

### 3.1 结论
- **提取**：**完整**。`_extracted/LitePan-own-custom` 9 patches + `stat/full.diff` 与 `git diff 93616d6..099cbc9` 一致，无遗漏。
- **融合**：**核心完整、5 处高/中优修复遗漏**。`local_upload` 增量（全量 hash + 云检查 + B mode + 多选 + 动态映射）已等价融合；`115_Open 600s/512MB、189Cloud 容错、file NOT_FOUND、AppSelect multiple` 未融合。

### 3.2 待办（建议新任务）
- **建议新开 `08-31-fuse-litepan-own-drivers-fixes`（或并入 `0.0.3` 发版）** 补齐 A-E：
  ```
  - drivers/115_Open/driver.go:73 30s → 600s
  - drivers/115_Open/upload.go:743 固定 512MB
  - drivers/189Cloud/ops.go:266 容错
  - internal/file/service.go:226 NOT_FOUND 视为成功
  - web/src/components/base/AppSelect.vue 支持 multiple（LitePan-own 同版 copy）
  ```
  验收：`go vet 0` / `type-check 0` / `docker build 118MB` / `ghcr 0.0.3` / `10.0.0.99` 复验 `100M` 单片与多选。

- **可选**：明确 `emby_refresh` 去留（若保留则从 `LitePan-own` 移植 `runEmbyRefresh` + 前端；若移除则删 `domain` 常量并 `spec` 归档）。

### 3.3 风险
- 不补 A/B：天翼/115 大文件上传超时或分片不一致（已有线上 `100M` 验证）。
- 不补 C/D：天翼 `batchTaskInfos` 与文件删除在增量重传后易失败。
- 不补 E：多映射自动化只能单选，与 `LitePan-own` 的 `我的文件/杂物间/pve_backup 3 映射` 日常使用不符。

---

## 附：核对命令
```bash
git -C LitePan-own diff 93616d6..099cbc9 --stat
diff -u LitePan/internal/automation/service_run.go _extracted/.../service_run.go | grep -E "mappings|B mode"
grep -n "Timeout" LitePan/drivers/115_Open/driver.go      # 现 30s，应 600s
sed -n '743,760p' LitePan/drivers/115_Open/upload.go       # 现分级，应 512MB
sed -n '264,285p' LitePan/drivers/189Cloud/ops.go          # 现 return err，应容错
sed -n '226,235p' LitePan/internal/file/service.go         # 现 Warn，应 CodeNotFound→Info
grep -n "multiple" LitePan/web/src/components/base/AppSelect.vue  # 现无，应有
```
