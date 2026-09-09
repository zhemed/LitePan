-- 0023: 上传批次化（移植上游 de83b46，其编号 0022 与本方 09-09-cleanup-orphan-db-tables
-- 的 0022 冲突，故用本方下一个版本号）。
-- 文件夹上传聚合为单条批次任务：batch_id 关联同批任务，batch_name 存批次显示名。
ALTER TABLE upload_tasks ADD COLUMN batch_id TEXT NOT NULL DEFAULT '';
ALTER TABLE upload_tasks ADD COLUMN batch_name TEXT NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_upload_tasks_batch_id ON upload_tasks(batch_id);
