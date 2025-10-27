-- MySQL Migration: 将邮件内容和附件迁移到文件系统
-- 移除 messages 表中的大字段，保留元数据

-- 1. 移除 messages 表的内容字段（保留元数据）
ALTER TABLE `messages` 
    DROP COLUMN `text_content`,
    DROP COLUMN `html_content`,
    DROP COLUMN `raw_content`;

-- 2. 移除 attachments 表的 content 字段
ALTER TABLE `attachments` 
    DROP COLUMN `content`;

-- 3. 为 messages 表添加文件系统标记字段（可选）
ALTER TABLE `messages`
    ADD COLUMN `has_raw` BOOLEAN DEFAULT FALSE COMMENT '是否有原始邮件文件',
    ADD COLUMN `has_html` BOOLEAN DEFAULT FALSE COMMENT '是否有HTML文件',
    ADD COLUMN `has_text` BOOLEAN DEFAULT FALSE COMMENT '是否有文本文件';

-- 4. 为 attachments 表添加文件系统标记字段（可选）
ALTER TABLE `attachments`
    ADD COLUMN `storage_path` VARCHAR(500) COMMENT '文件存储路径（相对路径）';








