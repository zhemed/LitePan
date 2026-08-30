# Adapt LitePan-own local_upload to LitePan

## Goal

将 `_extracted/LitePan-own-custom` 中 `LitePan-own` 的**本地自动上传**自动化能力（`AutomationActionLocalUpload` + `runLocalUpload` 全量 `sha256` 增量、`load/saveLocalUploadState`、`fileHash`）**适配**到当前 `zhemed/LitePan`（`three-drivers 118M`，`internal/automation` 仅 `delay`），使 `zhemed/LitePan` 的 `自动联动` 中出现 `本地上传` 动作，可按 `mapping/account/target` 定时将宿主机目录增量推到 115/天翼云盘，且**不向 `LitePan-own` 提交任何变更**。

## Requirements

- **Domain**：`internal/domain/automation.go` 新增 `const AutomationActionLocalUpload = "local_upload"`（与 `LitePan-own` 一致），不带回 `organize/strm/cache_clear/emby` 等已删动作
- **Service**：
  - `internal/automation/service.go` 的 `Options` 新增 `Settings *settings.Service`、`DataDir string`、`Uploads *upload.Manager`（`_extracted` 的 `runLocalUpload` 需 `s.settings.String(KeyLocalUploadMappings)`、`s.dataDir`、`s.uploads.CreateBatch`），`Service` 结构新增对应字段，并在 `New` 中赋值
  - `wire_services.go` 的 `automationSvc` 创建时传入 `Settings: st.settings, DataDir: cfg.DataDir, Uploads: uploadSvc, Files: fileSvc`
- **Run**：`internal/automation/service_run.go` 新增 `import`（`crypto/sha256/encoding/hex/encoding/json/io/fs/os/path/filepath` 等）、`case AutomationActionLocalUpload: return s.runLocalUpload(...)`、`runLocalUpload/fileHash/loadLocalUploadState/saveLocalUploadState` 全量移植（`185 行`，含 `B mode` 包装到映射文件夹、`relPath→sha256` 存 `local_upload_state_<mapping>.json`、`cloud existence check` 双重增量、`batchSize 100` 批量创建）
- **Validate**：`internal/automation/service_validate.go` 新增 `case AutomationActionLocalUpload` 的 `mapping/mappings/account_id/target` 校验，及 `normalizeInput` 的 `case LocalUpload` 允许
- **Frontend**：`web/src/api/automation.ts` 的 `AutomationActionType` 追加 `"local_upload"`，`AutomationOptions` 若需则保留；`web/src/components/admin/AutomationPanel.vue` 新增 `本地上传` 动作的 `mapping/account/target/conflict` 三选 UI（复用 `LitePan-own` 的 `AutomationPanel.vue` 片段，但适配 `three-drivers` 的 `115/189` 驱动）
- **只读**：`LitePan-own` 的 2 个克隆均保持 `Clean`，`_extracted/` 仍 `gitignore`

## Constraints

- 仅适配 `local_upload`，不带回 `LitePan-own` 的 `115_Open OSS 512M`、`189Cloud`、`file NOT_FOUND` 等其余 8 个 commits 的 `drivers` 优化（`zhemed/LitePan` 的 `115/189` 已是当前上游 `three-drivers` 状态）
- 保留 `internal/cache` 核心与 `mediaorganize/rules`、`automation` 的 `delay` 原有逻辑
- `runLocalUpload` 依赖的 `s.files.List/CreateFolder` 与 `s.uploads.CreateBatch` 已在 `zhemed/LitePan` 存在，无需新增
- `GOWORK=off go vet`、`web type-check` 必须通过

## Acceptance Criteria

- [ ] `internal/domain/automation.go` 含 `AutomationActionLocalUpload = "local_upload"`
- [ ] `internal/automation/service.go` 的 `Options` 含 `Settings/DataDir/Uploads` 且 `New` 赋值
- [ ] `internal/automation/service_run.go` 含 `runLocalUpload` 且 `grep -c "runLocalUpload" >=1`，`case LocalUpload` 在 `executeAction`
- [ ] `internal/automation/service_validate.go` 含 `case LocalUpload` 校验
- [ ] `web/src/api/automation.ts` 的 `AutomationActionType` 含 `"local_upload"`
- [ ] `web/src/components/admin/AutomationPanel.vue` 含 `本地上传` 动作 UI
- [ ] `GOWORK=off go vet ./...` PASS，`go build` PASS，`web type-check/build` PASS
- [ ] `docker build -t litepan-go:localupload .` PASS，`POST /api/admin/automation/validate` 对 `local_upload` 动作返回 `ok`

## Notes

- 本任务为 `LitePan-own` 提取后的**适配**，与 `08-30-extract-litepan-own-custom` 的只读提取配套
- `LitePan-own` 的 `local_upload` 为自动化增量（`relPath→sha256`），与 `zhemed/LitePan` 现有的 `LocalUpload`（映射目录手动上传）互补，前者为 `automation` 定时，后者为 `tools` 手动
