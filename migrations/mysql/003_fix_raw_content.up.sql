-- 修复 raw_content 字段类型
-- Migration: 003_fix_raw_content
-- 原因: MEDIUMTEXT 在某些情况下无法存储特殊字节序列，改为 MEDIUMBLOB

-- 将 raw_content 从 MEDIUMTEXT 改为 MEDIUMBLOB
ALTER TABLE messages 
    MODIFY COLUMN `raw_content` MEDIUMBLOB COMMENT '原始邮件内容（二进制）';









