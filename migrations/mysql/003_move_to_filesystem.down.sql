-- MySQL Migration Rollback: 恢复邮件内容和附件到数据库存储

-- 1. 恢复 messages 表的内容字段
ALTER TABLE `messages`
    ADD COLUMN `text_content` MEDIUMTEXT COMMENT '纯文本内容',
    ADD COLUMN `html_content` MEDIUMTEXT COMMENT 'HTML内容',
    ADD COLUMN `raw_content` MEDIUMTEXT COMMENT '原始邮件内容';

-- 2. 恢复 attachments 表的 content 字段
ALTER TABLE `attachments`
    ADD COLUMN `content` MEDIUMBLOB COMMENT '文件内容';

-- 3. 移除文件系统标记字段
ALTER TABLE `messages`
    DROP COLUMN `has_raw`,
    DROP COLUMN `has_html`,
    DROP COLUMN `has_text`;

-- 4. 移除文件存储路径字段
ALTER TABLE `attachments`
    DROP COLUMN `storage_path`;








