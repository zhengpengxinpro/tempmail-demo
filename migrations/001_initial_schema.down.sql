-- 删除触发器
DROP TRIGGER IF EXISTS update_user_domains_updated_at ON user_domains;
DROP TRIGGER IF EXISTS update_users_updated_at ON users;

-- 删除触发器函数
DROP FUNCTION IF EXISTS update_updated_at_column();

-- 删除表（按依赖关系逆序）
DROP TABLE IF EXISTS attachments;
DROP TABLE IF EXISTS user_domains;
DROP TABLE IF EXISTS mailbox_aliases;
DROP TABLE IF EXISTS messages;
DROP TABLE IF EXISTS mailboxes;
DROP TABLE IF EXISTS users;