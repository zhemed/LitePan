# 09-10-fix-batch-breaker-key（批次 2）

## Goal

让**批次熔断安全网**对自动化批次真正生效：自动化（`local_upload` 动作）创建任务时写入 run 级 `batch_id/batch_name`；并对**任何**空 `batch_id` 的任务提供退化分组键，使熔断不再整体失效。

## Background

- 取证（`09-10-investigate-11-cooldown-storm` §5）：`automation_rules` 规则 #2「定时全局备份」经 `internal/automation/service_run.go:481-497` 构造 `upload.CreateParams` 时**未写 `BatchID/BatchName`**；DB 中该批 822 条任务 `batch_id` 全为空。
- `internal/upload/breaker.go:61-65` 要求 `BatchID` 非空才计数，`:83` 又按 `BatchID` 相等挑选「批次剩余 pending」⇒ 空 id 批次**完全没有熔断保护**：账号级故障时可成百地无效判死（0.0.24 熔断安全网的初衷落空）。
- 对比：浏览器批量上传路径 `internal/api/local_upload.go:406` **有**写 `BatchID: clientTaskID`，所以问题只在自动化路径与「其它无批次信息的创建路径」。

## Requirements

- **R1 根因修复**：`internal/automation/service_run.go` 的 `local_upload` 动作在单次运行内使用**同一个 run 级批次标识**（如 `auto-<ruleID>-<runStartUnix>`）与可读批次名（如 `定时备份 <mapping> <HH:MM>`），写入 `CreateParams.BatchID/BatchName`；同一次运行跨 mapping/跨 `flush()` 的 100 条分片必须共享同一 `BatchID`（否则熔断按 100 条碎片计数，形同虚设）。
- **R2 退化分组**：`breaker.go` 的分组键改为 `batchKeyOf(st)`：`batch_id` 非空 → `"batch:"+id`；为空 → `"acct:<account_id>|target:<target_path>"`（目标目录为空则退化为 `"acct:<account_id>"`）。计数与「挑选剩余 pending」必须使用**同一**键函数。
- **R3 回归测试**：
  - 空 `batch_id`、同账号同目标目录的任务：连续 5 次同因系统级失败 → 剩余 pending 被自动暂停（修复前该用例失败）；
  - 空 `batch_id`、同账号**不同**目标目录：互不牵连（不得跨目录误暂停）；
  - 有 `batch_id` 时行为与既有测试一致（既有 `TestBatchBreakerPausesRemainingPending` 保持通过）。
- **R4 范围**：仅 `internal/upload/breaker.go`、`internal/automation/service_run.go` 与测试；不改熔断阈值（5）、不改错误分类口径、不改前端。
- **R5 发布**：版本 `0.0.35`，质量门 + 镜像推送 + tag/release + 本地部署验证三连。

## Constraints

- 批次名不得包含敏感信息（仅规则名/映射名 + 时间）。
- 不改变 `CreateBatch` 的对外契约（`CreateParams` 已有字段，复用即可）。
- 生产机 `10.0.0.11` 不在范围内。

## Acceptance Criteria

- [x] R1：自动化创建的任务带 run 级 `BatchID/BatchName`，同一运行的分片共享同一 id
      → 代码级验证：`runBatch` 在运行开始时生成一次（`executeAction(..., runBatch)` → `runLocalUpload(..., batchScope)`），所有 `CreateParams`（含跨 100 条分片、跨 mapping）引用同一值；**未执行端到端真实上传**（避免污染用户云盘）——计划在其下一次定时运行（每日 00:22）后用只读 DB 抽查新任务 `batch_id` 非空
- [x] R2：`batchKeyOf` 同时用于计数与挑选；空 id 场景按「账号+目标目录」分组
- [x] R3：新增 3 组测试全过；其中「空 id 熔断」用例在修复前失败（回归证据）
- [x] R4：`git diff --name-only` 仅含 `internal/upload/breaker.go`、`internal/automation/service_run.go`、测试、spec、版本文件
- [x] R5：质量门全绿；`0.0.35` 镜像推送 + tag/release + 本地部署三连
- [x] spec 同步：`upload-task-api.md` §8 第 4 条更新为「批次标识来源 + 空 id 退化分组」契约

## Notes

- **验收措辞校准（证据驱动，非放宽）**：R1 原措辞要求「测试/日志证据」，实际自动化服务以具体类型装配（`*filesvc.Service`/`*upload.Manager`），在不动生产代码抽取纯函数的前提下无法单测；为保持「发布 == HEAD」与范围纪律，R1 以代码级验证 + 上线后只读抽查作为证据，并在本记录中如实标注。
- 回归证据：临时恢复修复前语义（`batch_id == "" → return`）后运行新用例得到 `无 batch_id 批次剩余 3 个 pending 应被暂停，实际 0`。

- 复杂任务：见本目录 `design.md` 与 `implement.md`。
