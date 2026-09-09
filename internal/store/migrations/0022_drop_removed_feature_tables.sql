-- 0022: 清理已删功能的孤儿表与孤儿配置键。
-- 这些功能（STRM、媒体整理、缓存整理、内置离线下载、夸克TV）已在精简版中整体移除，
-- 表内无任何活代码读写（备份恢复清洗语句同步移除）。
-- 幂等：DROP TABLE IF EXISTS / DELETE，可重复执行。

DROP TABLE IF EXISTS strm_tasks;
DROP TABLE IF EXISTS strm_branches;
DROP TABLE IF EXISTS strm_remote_dir_cache;
DROP TABLE IF EXISTS media_organize_tasks;
DROP TABLE IF EXISTS cache_retention_tasks;
DROP TABLE IF EXISTS offline_download_tasks;
DROP TABLE IF EXISTS quarktv_bindings;

DELETE FROM configs WHERE key IN ('strm_base_url', 'strm_token');
