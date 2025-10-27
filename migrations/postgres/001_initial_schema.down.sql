-- PostgreSQL 回滚脚本
-- 删除所有表（注意顺序，先删除有外键的表）

DROP TRIGGER IF EXISTS update_user_domains_updated_at ON user_domains;
DROP TRIGGER IF EXISTS update_users_updated_at ON users;
DROP FUNCTION IF EXISTS update_updated_at_column();

DROP TABLE IF EXISTS api_keys CASCADE;
DROP TABLE IF EXISTS attachments CASCADE;
DROP TABLE IF EXISTS mailbox_aliases CASCADE;
DROP TABLE IF EXISTS messages CASCADE;
DROP TABLE IF EXISTS system_domains CASCADE;
DROP TABLE IF EXISTS user_domains CASCADE;
DROP TABLE IF EXISTS mailboxes CASCADE;
DROP TABLE IF EXISTS users CASCADE;
