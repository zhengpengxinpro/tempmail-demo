-- PostgreSQL 初始化脚本
-- TempMail 临时邮箱系统数据库结构

-- 创建用户表
CREATE TABLE IF NOT EXISTS users (
    id VARCHAR(36) PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    username VARCHAR(100),
    password_hash VARCHAR(255),
    role VARCHAR(20) DEFAULT 'user',
    tier VARCHAR(20) DEFAULT 'free',
    is_active BOOLEAN DEFAULT TRUE,
    is_email_verified BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_login_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);
CREATE INDEX IF NOT EXISTS idx_users_tier ON users(tier);

COMMENT ON TABLE users IS '用户表';
COMMENT ON COLUMN users.id IS '用户ID (UUID)';
COMMENT ON COLUMN users.email IS '邮箱地址';
COMMENT ON COLUMN users.role IS '角色: user/admin/super';
COMMENT ON COLUMN users.tier IS '等级: free/basic/pro/enterprise';

-- 创建邮箱表
CREATE TABLE IF NOT EXISTS mailboxes (
    id VARCHAR(36) PRIMARY KEY,
    address VARCHAR(255) UNIQUE NOT NULL,
    local_part VARCHAR(100) NOT NULL,
    domain VARCHAR(100) NOT NULL,
    token VARCHAR(255) NOT NULL,
    user_id VARCHAR(36),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP,
    ip_source VARCHAR(45),
    total_count INTEGER DEFAULT 0,
    unread INTEGER DEFAULT 0,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_mailboxes_address ON mailboxes(address);
CREATE INDEX IF NOT EXISTS idx_mailboxes_user_id ON mailboxes(user_id);
CREATE INDEX IF NOT EXISTS idx_mailboxes_domain ON mailboxes(domain);
CREATE INDEX IF NOT EXISTS idx_mailboxes_expires_at ON mailboxes(expires_at);
CREATE INDEX IF NOT EXISTS idx_mailboxes_created_at ON mailboxes(created_at);

COMMENT ON TABLE mailboxes IS '邮箱表';

-- 创建邮件表
CREATE TABLE IF NOT EXISTS messages (
    id VARCHAR(36) PRIMARY KEY,
    mailbox_id VARCHAR(36) NOT NULL,
    from_address VARCHAR(255) NOT NULL,
    to_address VARCHAR(255) NOT NULL,
    subject TEXT,
    text_content TEXT,
    html_content TEXT,
    raw_content TEXT,
    is_read BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (mailbox_id) REFERENCES mailboxes(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_messages_mailbox_id ON messages(mailbox_id);
CREATE INDEX IF NOT EXISTS idx_messages_created_at ON messages(created_at);
CREATE INDEX IF NOT EXISTS idx_messages_is_read ON messages(is_read);

COMMENT ON TABLE messages IS '邮件表';

-- 创建附件表
CREATE TABLE IF NOT EXISTS attachments (
    id VARCHAR(36) PRIMARY KEY,
    message_id VARCHAR(36) NOT NULL,
    mailbox_id VARCHAR(36) NOT NULL,
    filename VARCHAR(255) NOT NULL,
    content_type VARCHAR(100),
    size INTEGER,
    content BYTEA,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (message_id) REFERENCES messages(id) ON DELETE CASCADE,
    FOREIGN KEY (mailbox_id) REFERENCES mailboxes(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_attachments_message_id ON attachments(message_id);
CREATE INDEX IF NOT EXISTS idx_attachments_mailbox_id ON attachments(mailbox_id);

COMMENT ON TABLE attachments IS '附件表';

-- 创建邮箱别名表
CREATE TABLE IF NOT EXISTS mailbox_aliases (
    id VARCHAR(36) PRIMARY KEY,
    mailbox_id VARCHAR(36) NOT NULL,
    address VARCHAR(255) UNIQUE NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (mailbox_id) REFERENCES mailboxes(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_aliases_address ON mailbox_aliases(address);
CREATE INDEX IF NOT EXISTS idx_aliases_mailbox_id ON mailbox_aliases(mailbox_id);
CREATE INDEX IF NOT EXISTS idx_aliases_is_active ON mailbox_aliases(is_active);

COMMENT ON TABLE mailbox_aliases IS '邮箱别名表';

-- 创建用户域名表
CREATE TABLE IF NOT EXISTS user_domains (
    id VARCHAR(36) PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL,
    domain VARCHAR(100) UNIQUE NOT NULL,
    mode VARCHAR(20) DEFAULT 'shared',
    status VARCHAR(20) DEFAULT 'pending',
    verify_token VARCHAR(255),
    verify_method VARCHAR(20) DEFAULT 'dns_txt',
    verified_at TIMESTAMP,
    last_check_at TIMESTAMP,
    expires_at TIMESTAMP,
    is_active BOOLEAN DEFAULT FALSE,
    mailbox_count INTEGER DEFAULT 0,
    monthly_fee DECIMAL(10,2) DEFAULT 0.00,
    mx_records JSONB,
    notes TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_user_domains_domain ON user_domains(domain);
CREATE INDEX IF NOT EXISTS idx_user_domains_user_id ON user_domains(user_id);
CREATE INDEX IF NOT EXISTS idx_user_domains_status ON user_domains(status);
CREATE INDEX IF NOT EXISTS idx_user_domains_is_active ON user_domains(is_active);

COMMENT ON TABLE user_domains IS '用户域名表';

-- 创建系统域名表
CREATE TABLE IF NOT EXISTS system_domains (
    id VARCHAR(36) PRIMARY KEY,
    domain VARCHAR(100) UNIQUE NOT NULL,
    status VARCHAR(20) DEFAULT 'pending',
    verify_token VARCHAR(255),
    verify_method VARCHAR(20) DEFAULT 'dns_txt',
    verified_at TIMESTAMP,
    last_check_at TIMESTAMP,
    is_active BOOLEAN DEFAULT FALSE,
    is_default BOOLEAN DEFAULT FALSE,
    mailbox_count INTEGER DEFAULT 0,
    mx_records JSONB,
    created_by VARCHAR(36),
    notes TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_system_domains_domain ON system_domains(domain);
CREATE INDEX IF NOT EXISTS idx_system_domains_status ON system_domains(status);
CREATE INDEX IF NOT EXISTS idx_system_domains_is_active ON system_domains(is_active);
CREATE INDEX IF NOT EXISTS idx_system_domains_is_default ON system_domains(is_default);
CREATE INDEX IF NOT EXISTS idx_system_domains_created_at ON system_domains(created_at);

COMMENT ON TABLE system_domains IS '系统域名表';

-- 创建API密钥表
CREATE TABLE IF NOT EXISTS api_keys (
    id VARCHAR(36) PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL,
    name VARCHAR(100) NOT NULL,
    key_hash VARCHAR(255) NOT NULL,
    key_prefix VARCHAR(20) NOT NULL,
    scopes JSONB,
    is_active BOOLEAN DEFAULT TRUE,
    last_used_at TIMESTAMP,
    expires_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_api_keys_user_id ON api_keys(user_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_key_prefix ON api_keys(key_prefix);
CREATE INDEX IF NOT EXISTS idx_api_keys_is_active ON api_keys(is_active);

COMMENT ON TABLE api_keys IS 'API密钥表';

-- 创建触发器函数用于更新 updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- 创建触发器
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_user_domains_updated_at BEFORE UPDATE ON user_domains
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- 插入默认超级管理员用户（密码：admin123）
INSERT INTO users (id, email, username, password_hash, role, tier, is_active, is_email_verified) 
VALUES (
    gen_random_uuid()::text,
    'admin@tempmail.local',
    'admin',
    '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', -- bcrypt hash of "admin123"
    'super',
    'enterprise',
    TRUE,
    TRUE
) ON CONFLICT (email) DO NOTHING;
