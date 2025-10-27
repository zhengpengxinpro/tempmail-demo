-- MySQL 5.7+ 初始化脚本
-- TempMail 临时邮箱系统数据库结构

-- 设置数据库默认字符集
ALTER DATABASE tempmail CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- 创建用户表
CREATE TABLE IF NOT EXISTS `users` (
    `id` VARCHAR(36) PRIMARY KEY COMMENT '用户ID (UUID)',
    `email` VARCHAR(255) UNIQUE NOT NULL COMMENT '邮箱地址',
    `username` VARCHAR(100) COMMENT '用户名',
    `password_hash` VARCHAR(255) COMMENT '密码哈希',
    `role` VARCHAR(20) DEFAULT 'user' COMMENT '角色: user/admin/super',
    `tier` VARCHAR(20) DEFAULT 'free' COMMENT '等级: free/basic/pro/enterprise',
    `is_active` BOOLEAN DEFAULT TRUE COMMENT '是否激活',
    `is_email_verified` BOOLEAN DEFAULT FALSE COMMENT '邮箱是否验证',
    `created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `last_login_at` TIMESTAMP NULL COMMENT '最后登录时间',
    INDEX `idx_users_email` (`email`),
    INDEX `idx_users_role` (`role`),
    INDEX `idx_users_tier` (`tier`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户表';

-- 创建邮箱表
CREATE TABLE IF NOT EXISTS `mailboxes` (
    `id` VARCHAR(36) PRIMARY KEY COMMENT '邮箱ID (UUID)',
    `address` VARCHAR(255) UNIQUE NOT NULL COMMENT '完整邮箱地址',
    `local_part` VARCHAR(100) NOT NULL COMMENT '本地部分（@前）',
    `domain` VARCHAR(100) NOT NULL COMMENT '域名部分（@后）',
    `token` VARCHAR(255) NOT NULL COMMENT '访问令牌',
    `user_id` VARCHAR(36) COMMENT '所属用户ID（游客为NULL）',
    `created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `expires_at` TIMESTAMP NULL COMMENT '过期时间',
    `ip_source` VARCHAR(45) COMMENT '创建IP地址',
    `total_count` INT DEFAULT 0 COMMENT '总邮件数',
    `unread` INT DEFAULT 0 COMMENT '未读邮件数',
    INDEX `idx_mailboxes_address` (`address`),
    INDEX `idx_mailboxes_user_id` (`user_id`),
    INDEX `idx_mailboxes_domain` (`domain`),
    INDEX `idx_mailboxes_expires_at` (`expires_at`),
    INDEX `idx_mailboxes_created_at` (`created_at`),
    FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='邮箱表';

-- 创建邮件表
CREATE TABLE IF NOT EXISTS `messages` (
    `id` VARCHAR(36) PRIMARY KEY COMMENT '邮件ID (UUID)',
    `mailbox_id` VARCHAR(36) NOT NULL COMMENT '所属邮箱ID',
    `from_address` VARCHAR(255) NOT NULL COMMENT '发件人地址',
    `to_address` VARCHAR(255) NOT NULL COMMENT '收件人地址',
    `subject` TEXT COMMENT '邮件主题',
    `text_content` MEDIUMTEXT COMMENT '纯文本内容',
    `html_content` MEDIUMTEXT COMMENT 'HTML内容',
    `raw_content` MEDIUMTEXT COMMENT '原始邮件内容',
    `is_read` BOOLEAN DEFAULT FALSE COMMENT '是否已读',
    `created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '接收时间',
    INDEX `idx_messages_mailbox_id` (`mailbox_id`),
    INDEX `idx_messages_created_at` (`created_at`),
    INDEX `idx_messages_is_read` (`is_read`),
    FOREIGN KEY (`mailbox_id`) REFERENCES `mailboxes`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='邮件表';

-- 创建附件表
CREATE TABLE IF NOT EXISTS `attachments` (
    `id` VARCHAR(36) PRIMARY KEY COMMENT '附件ID (UUID)',
    `message_id` VARCHAR(36) NOT NULL COMMENT '所属邮件ID',
    `mailbox_id` VARCHAR(36) NOT NULL COMMENT '所属邮箱ID',
    `filename` VARCHAR(255) NOT NULL COMMENT '文件名',
    `content_type` VARCHAR(100) COMMENT 'MIME类型',
    `size` INT COMMENT '文件大小（字节）',
    `content` MEDIUMBLOB COMMENT '文件内容',
    `created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    INDEX `idx_attachments_message_id` (`message_id`),
    INDEX `idx_attachments_mailbox_id` (`mailbox_id`),
    FOREIGN KEY (`message_id`) REFERENCES `messages`(`id`) ON DELETE CASCADE,
    FOREIGN KEY (`mailbox_id`) REFERENCES `mailboxes`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='附件表';

-- 创建邮箱别名表
CREATE TABLE IF NOT EXISTS `mailbox_aliases` (
    `id` VARCHAR(36) PRIMARY KEY COMMENT '别名ID (UUID)',
    `mailbox_id` VARCHAR(36) NOT NULL COMMENT '主邮箱ID',
    `address` VARCHAR(255) UNIQUE NOT NULL COMMENT '别名地址',
    `is_active` BOOLEAN DEFAULT TRUE COMMENT '是否激活',
    `created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    INDEX `idx_aliases_address` (`address`),
    INDEX `idx_aliases_mailbox_id` (`mailbox_id`),
    INDEX `idx_aliases_is_active` (`is_active`),
    FOREIGN KEY (`mailbox_id`) REFERENCES `mailboxes`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='邮箱别名表';

-- 创建用户域名表
CREATE TABLE IF NOT EXISTS `user_domains` (
    `id` VARCHAR(36) PRIMARY KEY COMMENT '域名ID (UUID)',
    `user_id` VARCHAR(36) NOT NULL COMMENT '所属用户ID',
    `domain` VARCHAR(100) UNIQUE NOT NULL COMMENT '域名',
    `mode` VARCHAR(20) DEFAULT 'shared' COMMENT '模式: shared/exclusive',
    `status` VARCHAR(20) DEFAULT 'pending' COMMENT '状态: pending/verified/failed/expired',
    `verify_token` VARCHAR(255) COMMENT 'DNS验证令牌',
    `verify_method` VARCHAR(20) DEFAULT 'dns_txt' COMMENT '验证方式',
    `verified_at` TIMESTAMP NULL COMMENT '验证时间',
    `last_check_at` TIMESTAMP NULL COMMENT '最后检查时间',
    `expires_at` TIMESTAMP NULL COMMENT '过期时间（独享模式）',
    `is_active` BOOLEAN DEFAULT FALSE COMMENT '是否激活',
    `mailbox_count` INT DEFAULT 0 COMMENT '邮箱数量',
    `monthly_fee` DECIMAL(10,2) DEFAULT 0.00 COMMENT '月费',
    `mx_records` JSON COMMENT 'MX记录配置',
    `notes` TEXT COMMENT '备注',
    `created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    INDEX `idx_user_domains_domain` (`domain`),
    INDEX `idx_user_domains_user_id` (`user_id`),
    INDEX `idx_user_domains_status` (`status`),
    INDEX `idx_user_domains_is_active` (`is_active`),
    FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户域名表';

-- 创建系统域名表
CREATE TABLE IF NOT EXISTS `system_domains` (
    `id` VARCHAR(36) PRIMARY KEY COMMENT '域名ID (UUID)',
    `domain` VARCHAR(100) UNIQUE NOT NULL COMMENT '域名',
    `status` VARCHAR(20) DEFAULT 'pending' COMMENT '状态: pending/verified/failed',
    `verify_token` VARCHAR(255) COMMENT 'DNS验证令牌',
    `verify_method` VARCHAR(20) DEFAULT 'dns_txt' COMMENT '验证方式',
    `verified_at` TIMESTAMP NULL COMMENT '验证时间',
    `last_check_at` TIMESTAMP NULL COMMENT '最后检查时间',
    `is_active` BOOLEAN DEFAULT FALSE COMMENT '是否激活',
    `is_default` BOOLEAN DEFAULT FALSE COMMENT '是否默认域名',
    `mailbox_count` INT DEFAULT 0 COMMENT '邮箱数量',
    `mx_records` JSON COMMENT 'MX记录配置',
    `created_by` VARCHAR(36) COMMENT '创建者用户ID',
    `notes` TEXT COMMENT '备注',
    `created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    INDEX `idx_system_domains_domain` (`domain`),
    INDEX `idx_system_domains_status` (`status`),
    INDEX `idx_system_domains_is_active` (`is_active`),
    INDEX `idx_system_domains_is_default` (`is_default`),
    INDEX `idx_system_domains_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='系统域名表';

-- 创建API密钥表
CREATE TABLE IF NOT EXISTS `api_keys` (
    `id` VARCHAR(36) PRIMARY KEY COMMENT 'API Key ID (UUID)',
    `user_id` VARCHAR(36) NOT NULL COMMENT '所属用户ID',
    `name` VARCHAR(100) NOT NULL COMMENT 'API Key名称',
    `key_hash` VARCHAR(255) NOT NULL COMMENT 'Key哈希值',
    `key_prefix` VARCHAR(20) NOT NULL COMMENT 'Key前缀（用于识别）',
    `scopes` JSON COMMENT '权限范围',
    `is_active` BOOLEAN DEFAULT TRUE COMMENT '是否激活',
    `last_used_at` TIMESTAMP NULL COMMENT '最后使用时间',
    `expires_at` TIMESTAMP NULL COMMENT '过期时间',
    `created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    INDEX `idx_api_keys_user_id` (`user_id`),
    INDEX `idx_api_keys_key_prefix` (`key_prefix`),
    INDEX `idx_api_keys_is_active` (`is_active`),
    FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='API密钥表';

-- 插入默认超级管理员用户（密码：admin123）
INSERT INTO `users` (`id`, `email`, `username`, `password_hash`, `role`, `tier`, `is_active`, `is_email_verified`) 
VALUES (
    UUID(),
    'admin@tempmail.local',
    'admin',
    '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', -- bcrypt hash of "admin123"
    'super',
    'enterprise',
    TRUE,
    TRUE
) ON DUPLICATE KEY UPDATE `id`=`id`;
