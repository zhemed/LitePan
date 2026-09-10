# 09-11-fix-terminal-bucket-with-batch-grouping

## Goal

修复既有缺陷「**批次分组下终态桶恒为空**」：任务面板在「已完成 / 失败」桶里必须能列出批次内的终态文件（用户 2026-09-11 选择方案 A：保留「进行中」折叠、修好终态桶）。

## Background

- 现象（用户截图，生产机 `10.0.0.11`）：「已完成 **13**」徽标有数字，列表却显示「暂无已完成上传任务」。
- 机理：
  - 面板把同一 `batch_id` 的任务折叠为一个文件夹节点（`web/src/composables/upload/uploadTaskTree.ts`）；
  - 批次行状态取**聚合值**：任一成员 active ⇒ `active`（`TaskPanel.vue:563-567`）；
  - 桶过滤只看顶层行状态（`TaskPanel.vue:629`）⇒ 批次行永远只进「进行中」，其内的终态文件在根层级不可见。
- 该缺陷自 0.0.18 引入批次分组即存在；自动化批次在 0.0.35 首次带上 `batch_id` 后暴露（0.0.36 已回退自动化分组，但**浏览器批量/文件夹上传仍会触发**）。

## Requirements

- **R1 仅进行中折叠**：`buildUploadTaskLevel()` 新增 `options.groupBatches`（默认 `true` 保持兼容）；`TaskPanel` 仅在「进行中」桶传 `true`，终态桶传 `false`（展开为逐文件行）。
- **R2 终态可见**：切到「已完成」/「失败」时，批次内的对应文件以文件行出现；「进行中」仍是一行批次（含总进度 `x/y`）。
- **R3 不改计数口径**：徽标数字仍来自汇总（`summary.counts`），不受行展开影响。
- **R4 可断言**：`npm run check:memo` 新增用例——默认折叠（含批次节点）、`groupBatches:false` 展开为逐文件行且批次内文件可见。
- **R5 范围**：仅前端（`uploadTaskTree.ts`、`TaskPanel.vue`、断言脚本）；不改后端、不改缓存/分页/SSE。
- **R6 发布**：版本 `0.0.37`，质量门 + 镜像推送 + tag/release + 本地部署三连；不部署生产机。

## Constraints

- 进入批次目录（`currentBatchId` 非空）时的路径级构建行为不变。
- 不引入配置开关；不改变任务数据本身（纯展示层）。

## Acceptance Criteria

- [x] R1：`buildUploadTaskLevel` 支持 `groupBatches` 且默认行为与旧版一致
- [x] R2：`TaskPanel` 的 `uploadRootRows` 按 `uploadStateFilter` 决定是否分组
- [x] R3：徽标口径未改动（`uploadTaskTotals` 无 diff）
- [x] R4：`npm run check:memo` 新增断言全过
- [x] R5：`git diff --name-only` 仅含前端两文件 + 断言脚本 + spec + 版本文件
- [x] R6：`npm run type-check`/`build`、`go vet`/`go test` 全绿；`0.0.37` 镜像推送 + tag/release + 本地部署三连
- [x] spec §8.4 的「已知缺陷」条目更新为「已修复（0.0.37）」

## Notes

- Lightweight（PRD-only）：改动集中在展示层的一个开关 + 一个参数透传。
- **过程偏差（如实记录）**：本任务的两处代码改动发生在 `task.py start` **之前**（我在建任务后直接动手，未先跑门禁），PRD 仍先于 `start` 完成（门禁要求满足）。偏差写入 journal，后续严格按「PRD → 门禁 → start → 实施」执行。
- 与被 0.0.36 回退的自动化分组不冲突：自动化批次此后无 `batch_id`，天然平铺；本修复保障浏览器批量上传场景。
