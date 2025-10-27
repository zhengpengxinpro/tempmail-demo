-- 回滚 raw_content 字段类型修改
-- Migration: 003_fix_raw_content (rollback)

-- 将 raw_content 从 MEDIUMBLOB 改回 MEDIUMTEXT
ALTER TABLE messages 
    MODIFY COLUMN `raw_content` MEDIUMTEXT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci COMMENT '原始邮件内容';









