# Implementation Plan: Adapt LitePan-own local_upload to LitePan

## Overview

按 `design.md` 6 文件移植 `local_upload`。

## Phase 1: Domain & Service Options

- [ ] 1.1 `internal/domain/automation.go`：在 `AutomationActionDelay` 旁新增 `AutomationActionLocalUpload = "local_upload"`
  - `verify: grep -n "LocalUpload" internal/domain/automation.go` >=1
- [ ] 1.2 `internal/automation/service.go`：`Options` 新增 `Settings *settings.Service`、`DataDir string`、`Uploads *upload.Manager`；`Service` 新增 `settings *settings.Service`、`dataDir string`、`uploads *upload.Manager`；`New` 中赋值 `settings: opts.Settings, dataDir: opts.DataDir, uploads: opts.Uploads`
  - `verify: grep -n "Settings.*settings.Service" internal/automation/service.go` >=1
- [ ] 1.3 `internal/app/wire_services.go`：`automationSvc` 创建时传入 `Settings: st.settings, DataDir: cfg.DataDir, Uploads: uploadSvc`（`uploadSvc` 已在同文件创建，需调整顺序使 `uploadSvc` 在 `automationSvc` 前）
  - `verify: grep -n "Settings.*st.settings" internal/app/wire_services.go` >=1

## Phase 2: Run & Validate

- [ ] 2.1 `internal/automation/service_run.go`：在 `import` 追加 `crypto/sha256/encoding/hex/encoding/json/io/fs/os/path/filepath` 等；在 `executeAction` 的 `switch` 新增 `case AutomationActionLocalUpload: return s.runLocalUpload(...)`；文末追加 `fileHash/loadLocalUploadState/saveLocalUploadState/runLocalUpload` 全量（`_extracted/.../service_run.go` 的 511-850 行，含 `B mode`）
  - `verify: grep -c "runLocalUpload" internal/automation/service_run.go` >=1
- [ ] 2.2 `internal/automation/service_validate.go`：`ValidateRule` 的 `switch` 新增 `case AutomationActionLocalUpload` 的 `mapping/mappings/account_id` 校验；在 `normalizeInput` 的 `switch case` 新增 `case AutomationActionLocalUpload`
  - `verify: grep -n "LocalUpload" internal/automation/service_validate.go` >=1

## Phase 3: Frontend

- [ ] 3.1 `web/src/api/automation.ts`：`AutomationActionType = "delay"` → `"delay" | "local_upload"`；若 `AutomationOptions` 需则保留 `organize_tasks/emby_configs` 空数组兼容
  - `verify: grep -n "local_upload" web/src/api/automation.ts` >=1
- [ ] 3.2 `web/src/components/admin/AutomationPanel.vue`：移植 `LitePan-own` 的 `本地上传` 动作 UI（含 `mapping/account/target/conflict` 三选 + `B mode` 提示），复用 `localUploadApi` 的 `getConfig` 拉 `mappings` 列表
  - `verify: grep -n "本地上传" web/src/components/admin/AutomationPanel.vue` >=1

## Phase 4: Sweep & Verification

- [ ] 4.1 `GOWORK=off go vet ./...` PASS
- [ ] 4.2 `GOWORK=off go build -o /tmp/litepan-localupload ./cmd/litepan` PASS
- [ ] 4.3 `cd web && npm run type-check` PASS, `npm run build` PASS
- [ ] 4.4 `docker build -t litepan-go:localupload .` PASS，`docker run -d -p 5219:5211 ...` + `curl /api/health 200` + `curl -b cookie -H "Content-Type: application/json" -d '{"actions":[{"type":"local_upload","params":{"mappings":["test"],"account_id":1}}]}' http://127.0.0.1:5219/api/admin/automation/validate` → `{"ok":true}` 或 `mapping 不存在` 的校验通过

## Phase 5: Commit & Archive

- [ ] 5.1 `git add -A && git restore --staged .trellis/tasks/08-30-adapt-litepan-own-localupload && git commit -m "feat(automation): adapt local_upload from LitePan-own"`
- [ ] 5.2 `task.py archive ... --skip-branch-validation && git add ... && git commit`
- [ ] 5.3 `add_session.py --commit <hash>`

## Rollback

- `git revert <feat commit>`

## Validation Commands

```bash
grep -rn "LocalUpload" --include="*.go" internal/automation | wc -l  # >=4
GOWORK=off go vet ./...
GOWORK=off go build -o /tmp/litepan ./cmd/litepan && echo ok
curl -b /tmp/c.txt -H "Content-Type: application/json" -d '{"actions":[{"type":"local_upload","params":{}}]}' http://127.0.0.1:5211/api/admin/automation/validate | grep ok
```
