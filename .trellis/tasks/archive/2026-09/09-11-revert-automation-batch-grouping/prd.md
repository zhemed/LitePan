# 09-11-revert-automation-batch-grouping

## Goal

按用户决策（2026-09-11，方案 B）**回退自动化批次分组**：自动化 `local_upload` 不再为任务写入 `batch_id/batch_name`，任务面板恢复「一文件一行」的平铺列表；同时保留 0.0.35 的熔断保护能力（空批次任务由退化分组键覆盖）。

## Background

- 0.0.35 为自动化运行写入了批次身份（`auto-<ruleID>-<unix>` / `定时任务 <规则名>`），本意是让批次熔断生效。副作用：任务面板按 `batch_id` 折叠（`web/src/composables/upload/uploadTaskTree.ts:54-63`），生产机 819 个文件被收成一行「定时任务 定时全局备份（传输中 13/819）」。
- 用户观察并确认环境：生产机 `10.0.0.11`（已升 0.0.35）；用户选择**恢复平铺列表**。
- 回退不影响熔断：0.0.35 已把分组键改为 `batchKeyOf()`——`batch_id` 为空时退化为 `acct:<account_id>|target:<target_path>`，并已有回归测试（`TestBreakerTripsForBatchlessTasksWithSameTarget` 等）覆盖。
- 附带发现（本次不修，登记为既有缺陷）：批次分组下**终态桶恒为空**——批次行状态取聚合值（`TaskPanel.vue:563-567`：任一成员 active → active），桶过滤只看顶层行状态（`TaskPanel.vue:629`），因此「已完成/失败」桶看不到批次内的终态文件。该缺陷自 0.0.18 批次分组引入即存在（浏览器批量上传同样会中招）。

## Requirements

- **R1 回退批次身份**：`internal/automation/service_run.go` 不再向 `upload.CreateParams` 写入 `BatchID/BatchName`；移除为此引入的 `uploadBatchScope` 类型与 `executeAction/runLocalUpload` 的额外参数（避免保留死参数），恢复为 0.0.34 形态。
- **R2 保留熔断**：`internal/upload/breaker.go` 的 `batchKeyOf` 与退化分组**保持不变**（测试须继续全绿），自动化任务由此仍受熔断保护。
- **R3 前端零改动**：面板分组逻辑不动（无批次即平铺，天然恢复）。
- **R4 记录缺陷**：在 spec §8.4 登记「批次分组下终态桶为空」为**已知缺陷**（触发条件、位置、后续修复方向），避免下次重复排查。
- **R5 发布**：版本 `0.0.36`（`fix` 级），质量门 + 镜像推送 + tag/release + 本地部署三连；不部署生产机。
- **R6 数据存量说明**：已存在于生产机的 819 条任务仍带 `batch_id/batch_name`，本任务**不修改生产数据**；如需其恢复平铺显示，另行单独授权（备份 + 只改这两列）。

## Constraints

- 不引入配置开关（不做「自动化是否分组」的可选项，避免为单一场景加扩展点）。
- 不改熔断阈值/分类、不改冷却日志抑制、不改并发与节流参数。

## Acceptance Criteria

- [x] R1：`grep -n "BatchID\|BatchName\|uploadBatchScope" internal/automation/` 无结果；`git diff` 中该文件与 0.0.34 形态等价（仅回退）
- [x] R2：`internal/upload/breaker_key_test.go` 用例全部通过（空批次退化分组仍生效）
- [x] R3：`web/` 无改动（`git diff --name-only` 不含 `web/`）
- [x] R4：spec §8.4 已更新（自动化不写批次 id + 终态桶缺陷登记）
- [x] R5：`go vet`/`go test`/`-race`/`web type-check+build` 全绿；`0.0.36` 镜像推送 + tag/release + 本地部署三连
- [x] R6：任务记录中写明存量任务影响与「需单独授权」的数据清理路径

## Notes

- Lightweight（PRD-only）：改动是一次定向回退 + 文档同步，逻辑无新设计。
- 决策记录：用户 2026-09-11 明确选择「关掉自动化批次的分组，恢复平铺列表」；未选择「保留分组并修终态桶」（方案 A）——该缺陷仍留在待办中，浏览器批量上传场景仍可能触发。
