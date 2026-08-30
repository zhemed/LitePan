# 补齐 LitePan-own 5处修复

## Goal
补齐 `08-31-review-litepan-own-extraction-fusion` 发现的 5 处高/中优缺口，将 `LitePan-own` 已在 `10.0.0.99` 闭环的修复完整融合到 `LitePan`，并发布 `0.0.3`。

## Background
- 审查报告 `review.md 150 行`：提取完整（9 commits），融合核心 `local_upload` 完整，但 5 处修复未随 `adapt-litepan-own-localupload` 一并移植：
  - A: `drivers/115_Open/driver.go:73 Timeout 30s → 600s`（`httpx` 超时）
  - B: `drivers/115_Open/upload.go:743 calculateOSSPartSize` 固定 `512MB`
  - C: `drivers/189Cloud/ops.go:264 batchTaskInfos` 容错（`cachedItem` + `name==id` 兜底）
  - D: `internal/file/service.go:226 DeleteFiles NOT_FOUND视为成功`
  - E: `web/src/components/base/AppSelect.vue` 支持 `multiple`（`AutomationPanel` 已用 `multiple` 但组件未实现）
- 当前 `LitePan main @ 49e9c39` `0.0.2`（`AdminView stray />` 已修复），`LitePan-own @ 099cbc9` 为源。

## Requirements
- **A** `drivers/115_Open/driver.go`: `Init` 中 `httpx.NewClient(Timeout: 30s)` → `600s`，与 `LitePan-own 099cbc9` 一致（注释保留“用户 2026-08-26 定制”可省略，保 `600s` 即可）。
- **B** `drivers/115_Open/upload.go`: `calculateOSSPartSize` 整体替换为固定 `return 512*mb`，删 `gb/tb` 分级逻辑，注释 `固定 512MB 一片`。
- **C** `drivers/189Cloud/ops.go`: `batchTaskInfos` 内 `GetFileInfo` 失败时 `if cached, ok := d.cachedItem(id)` 容错，否则 `domain.FileItem{Name: id}` 兜底；`fileName` 取 `item.Name` 为空则 `id`。
- **D** `internal/file/service.go`: `DeleteFiles` 后 `if ae.Code == domain.CodeNotFound` 则 `Info` 视为成功不 `return err`，与 `LitePan-own 923129a` 一致。
- **E** `web/src/components/base/AppSelect.vue`: 从 `LitePan-own` 完整拷贝 `multiple` 支持（`modelValue` 联合类型、`multiple` prop、`selectedLabel` 分支、`choose` 增删、`@click` 不 `close` 时的多选、`AppSelect.vue` 模板 `select__option--active` 需支持 `Array.includes`）。
- 版本 `0.0.2 → 0.0.3`（fix 递增），`README` 与 `ghcr.io/zhemed/litepan:0.0.3` / `v0.0.3` / `latest` 同步，`git tag v0.0.3` + `gh release`。

## Constraints
- 只改上述 5 文件 + `README` + 构建产物 `internal/api/web`，不碰 `strm/share/cache/organize` 已精简部分，不改 `domain` 额外常量。
- 所有写操作在 `task.py start` 后，遵循 `AGENTS.md 项目强制规则` 与 `trellis-before-dev / trellis-check`。

## Acceptance Criteria
- [ ] `grep -n "Timeout.*600" drivers/115_Open/driver.go ==1`，`grep -n "30.*Second" ==0`
- [ ] `grep -n "calculateOSSPartSize" drivers/115_Open/upload.go` 仅 `return 512*mb`，无 `5*gb/109951163` 分级
- [ ] `grep -n "cachedItem" drivers/189Cloud/ops.go ==1`，`batchTaskInfos` 含兜底
- [ ] `grep -n "CodeNotFound" internal/file/service.go ==1`，`DeleteFiles` 视成功
- [ ] `grep -n "multiple" web/src/components/base/AppSelect.vue >=3`，`props.multiple` 与 `selectedLabel` 多选分支存在；`AutomationPanel` 多选可实测选 2 映射
- [ ] `cd web && npm run type-check ==0`，`GOWORK=off go vet ./... ==0`，`GOWORK=off go build -o /tmp/litepan ./cmd/litepan ==0`，`docker build -t ghcr.io/zhemed/litepan:0.0.3 . ==0` 且 `118MB`
- [ ] `README v0.0.2 → v0.0.3` 2 处，`docker push 0.0.3/v0.0.3/latest` 成功，`git tag v0.0.3` + `gh release v0.0.3` 完成
- [ ] 本地 `litepan` 重跑 `0.0.3` 镜像，`curl /api/health 200`
