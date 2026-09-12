# 09-12-record-local-failed-tasks-archived

## Goal

记录用户 2026-09-12 的确认「**本地那 3 条失败任务确认归档**」，并附只读复核证据。本任务**不修改任何数据**（本地或生产）。

## Background

- 2026-09-11 观察到本地实例有 3 条失败任务：`bulk5k_1609.bin`、`bulk5k_1540.bin`（`天翼云盘 API HTTP 511 S3ClientException`）、`bulk5k_1267.bin`（`HTTP 513 NoSuchUpload`）。
- 期间我方未对本地任务数据做任何写操作（仅 `mode=ro` 查询与容器重建）。
- 用户复核后确认这 3 条已归档。

## Requirements

- **R1 只读复核**：以 `mode=ro` 查询本地 DB，确认当前 `failed` 计数与这 3 条文件的归属。
- **R2 零写入**：不删除/不重试/不修改任何任务记录；不触碰生产机。
- **R3 如实标注观察**：本地任务列表规模已由用户侧清理（15,098 → 13），如属用户操作须在记录中标注（避免与「我方改动」混淆）。
- **R4 证据留档**：把查询结果写进 `research.md`，并与 2026-09-11 的失败清单逐一对照。

## Constraints

- 不因本条记录而扩大范围（不调查 189 S3 异常根因，如需要另开任务）。

## Acceptance Criteria

- [x] R1：已复核（本地 `upload_tasks` = 13 条、全部 `success`、`failed` = 0）
- [x] R2：本轮零写入（仅 `mode=ro` 查询 + `docker inspect`）
- [x] R3：已标注「任务列表规模变化来自用户侧清理」这一观察
- [x] R4：`research.md` 记录三文件对照结果与容器/镜像证据

## Notes

- Lightweight（PRD-only）：纯记录性任务，无代码与数据改动。
