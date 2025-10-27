-- MySQL 5.7+ 回滚脚本
-- 删除所有表（注意顺序，先删除有外键的表）

DROP TABLE IF EXISTS `api_keys`;
DROP TABLE IF EXISTS `attachments`;
DROP TABLE IF EXISTS `mailbox_aliases`;
DROP TABLE IF EXISTS `messages`;
DROP TABLE IF EXISTS `system_domains`;
DROP TABLE IF EXISTS `user_domains`;
DROP TABLE IF EXISTS `mailboxes`;
DROP TABLE IF EXISTS `users`;
