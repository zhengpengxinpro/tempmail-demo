-- 回滚字符集修复
-- 注意: 通常不需要回滚字符集更改

-- 如果需要回滚到 utf8 (不推荐)
-- ALTER DATABASE tempmail CHARACTER SET = utf8 COLLATE = utf8_general_ci;
-- ALTER TABLE messages CONVERT TO CHARACTER SET utf8 COLLATE utf8_general_ci;
-- ... 其他表类似

-- 实际上不执行任何操作,因为回滚字符集可能导致数据丢失
SELECT 'Character set rollback skipped - would cause data loss' AS warning;
