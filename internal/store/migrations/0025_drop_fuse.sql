-- 0025: 清理已删功能的孤儿表与配置键（FUSE 本地挂载 + 读缓存）。
-- 「本地挂载」（把网盘账号挂成本地目录）与 FUSE 读缓存已在精简版中整体移除：
-- 前端只有一张只读计数卡片、没有创建入口，且它曾是容器需要 privileged/pid host//dev/fuse
-- 的唯一原因；相关包与接线全部删除后，表与配置键无任何活代码读写。
-- 通知一并清理：NotificationCategoryFuseMountWarn 常量已随代码删除，
-- 旧库若残留该类别通知会变成前端渲染不出的未知类别。
-- 幂等：DROP TABLE IF EXISTS / DELETE，可重复执行。

DROP TABLE IF EXISTS fuse_mounts;

DELETE FROM configs WHERE key IN (
  'fuse_enabled',
  'fuse_mount_root',
  'fuse_read_cache_enabled',
  'fuse_read_cache_max_gb',
  'fuse_read_cache_retention_days',
  'fuse_read_cache_eviction_policy'
);

DELETE FROM notifications WHERE category = 'fuse_mount_warn';
