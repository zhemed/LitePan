-- 0024: 清理已删功能的孤儿表 api_keys。
-- API 密钥功能（后台「API 密钥」页、/api/api-keys 五个接口、internal/apikey 包、
-- domain.ApiKeyRepository 与仓储实现）已在精简版中整体移除：无任何入口中间件读取密钥，
-- 表内无任何活代码读写（备份恢复的清洗语句与组件清单同步移除）。
-- 幂等：DROP TABLE IF EXISTS，可重复执行。

DROP TABLE IF EXISTS api_keys;
