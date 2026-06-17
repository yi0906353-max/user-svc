-- 修改 PostgreSQL 认证为 md5（允许外部连接）
-- 此脚本在 initdb 之后运行
ALTER SYSTEM SET password_encryption = 'md5';
SELECT pg_reload_conf();
