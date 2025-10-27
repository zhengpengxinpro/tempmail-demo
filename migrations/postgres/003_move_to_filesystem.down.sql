-- PostgreSQL Migration Rollback: 恢复邮件内容和附件到数据库存储

-- 1. 恢复 messages 表的内容字段
ALTER TABLE messages
    ADD COLUMN text_content TEXT,
    ADD COLUMN html_content TEXT,
    ADD COLUMN raw_content TEXT;

-- 2. 恢复 attachments 表的 content 字段
ALTER TABLE attachments
    ADD COLUMN content BYTEA;

-- 3. 移除文件系统标记字段
ALTER TABLE messages
    DROP COLUMN IF EXISTS has_raw,
    DROP COLUMN IF EXISTS has_html,
    DROP COLUMN IF EXISTS has_text;

-- 4. 移除文件存储路径字段
ALTER TABLE attachments
    DROP COLUMN IF EXISTS storage_path;








