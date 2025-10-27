# 部署指南

本文档提供临时邮箱系统的完整部署指南。

## 📋 目录

- [系统要求](#系统要求)
- [开发环境部署](#开发环境部署)
- [生产环境部署](#生产环境部署)
- [Docker 部署](#docker-部署)
- [安全配置](#安全配置)
- [监控和维护](#监控和维护)

---

## 🖥️ 系统要求

### 最低配置

- **CPU**: 2 核心
- **内存**: 2GB RAM
- **存储**: 20GB 可用空间
- **操作系统**: Linux (推荐 Ubuntu 20.04+) / macOS / Windows

### 推荐配置（生产环境）

- **CPU**: 4 核心
- **内存**: 8GB RAM
- **存储**: 100GB SSD
- **操作系统**: Ubuntu 22.04 LTS

### 软件依赖

- **Go**: 1.21+ (开发和编译)
- **PostgreSQL**: 14+ (生产环境)
- **Redis**: 6+ (生产环境缓存)
- **Nginx**: 1.18+ (反向代理)
- **Docker**: 20.10+ (容器化部署，可选)

---

## 🛠️ 开发环境部署

### 1. 克隆项目

```bash
git clone https://github.com/your-org/tempmail.git
cd tempmail/backend
```

### 2. 安装依赖

```bash
go mod download
```

### 3. 配置环境变量

创建 `.env` 文件：

```bash
# JWT 密钥（至少32字符）
export TEMPMAIL_JWT_SECRET="dev-secret-key-please-change-in-production-32chars"

# 服务器配置
export TEMPMAIL_SERVER_HOST=0.0.0.0
export TEMPMAIL_SERVER_PORT=8080

# SMTP 配置
export TEMPMAIL_SMTP_BIND_ADDR=:25
export TEMPMAIL_SMTP_DOMAIN=temp.mail

# 邮箱配置
export TEMPMAIL_MAILBOX_ALLOWED_DOMAINS=temp.mail,tempmail.dev
export TEMPMAIL_MAILBOX_DEFAULT_TTL=24h

# 日志配置
export TEMPMAIL_LOG_LEVEL=debug
export TEMPMAIL_LOG_DEVELOPMENT=true
```

### 4. 启动服务

```bash
# 加载环境变量
source .env

# 启动综合服务（HTTP + SMTP + WebSocket）
go run ./cmd/server
```

### 5. 验证部署

```bash
# 健康检查
curl http://localhost:8080/health

# 创建测试邮箱
curl -X POST http://localhost:8080/v1/mailboxes
```

---

## 🚀 生产环境部署

### 方案 A: 二进制部署

#### 1. 编译生产版本

```bash
cd backend

# 编译优化的二进制文件
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
  -ldflags="-w -s" \
  -o tempmail-server \
  ./cmd/server
```

#### 2. 创建系统用户

```bash
sudo useradd -r -s /bin/false tempmail
```

#### 3. 部署文件

```bash
# 创建目录结构
sudo mkdir -p /opt/tempmail/{bin,logs,data}

# 复制二进制文件
sudo cp tempmail-server /opt/tempmail/bin/
sudo chmod +x /opt/tempmail/bin/tempmail-server

# 设置权限
sudo chown -R tempmail:tempmail /opt/tempmail
```

#### 4. 配置环境变量

创建 `/etc/tempmail/config.env`：

```bash
# JWT 密钥（生产环境必须修改！）
TEMPMAIL_JWT_SECRET="CHANGE-THIS-TO-A-SECURE-RANDOM-STRING-AT-LEAST-32-CHARS"

# 服务器配置
TEMPMAIL_SERVER_HOST=0.0.0.0
TEMPMAIL_SERVER_PORT=8080

# SMTP 配置
TEMPMAIL_SMTP_BIND_ADDR=:25
TEMPMAIL_SMTP_DOMAIN=temp.example.com

# 邮箱配置
TEMPMAIL_MAILBOX_ALLOWED_DOMAINS=temp.example.com,mail.example.com
TEMPMAIL_MAILBOX_DEFAULT_TTL=24h

# CORS 配置
TEMPMAIL_CORS_ALLOWED_ORIGINS=https://www.example.com,https://app.example.com

# 日志配置
TEMPMAIL_LOG_LEVEL=info
TEMPMAIL_LOG_DEVELOPMENT=false

# PostgreSQL 配置（生产环境）
TEMPMAIL_DATABASE_DSN=postgres://tempmail:password@localhost:5432/tempmail?sslmode=require

# Redis 配置（生产环境）
TEMPMAIL_REDIS_ADDRESS=localhost:6379
TEMPMAIL_REDIS_PASSWORD=your-redis-password
TEMPMAIL_REDIS_DB=0
```

#### 5. 创建 Systemd 服务

创建 `/etc/systemd/system/tempmail.service`：

```ini
[Unit]
Description=TempMail Backend Service
After=network.target postgresql.service redis.service
Wants=postgresql.service redis.service

[Service]
Type=simple
User=tempmail
Group=tempmail
WorkingDirectory=/opt/tempmail
EnvironmentFile=/etc/tempmail/config.env
ExecStart=/opt/tempmail/bin/tempmail-server
Restart=on-failure
RestartSec=10s

# 安全加固
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/opt/tempmail/logs /opt/tempmail/data

# 资源限制
LimitNOFILE=65536
LimitNPROC=512

[Install]
WantedBy=multi-user.target
```

#### 6. 启动服务

```bash
# 重新加载 systemd
sudo systemctl daemon-reload

# 启动服务
sudo systemctl start tempmail

# 设置开机自启
sudo systemctl enable tempmail

# 查看状态
sudo systemctl status tempmail

# 查看日志
sudo journalctl -u tempmail -f
```

### 方案 B: Docker 部署

参见 [Docker 部署](#docker-部署) 章节。

---

## 🐳 Docker 部署

### 1. 使用预构建镜像

```bash
# 拉取镜像
docker pull your-registry/tempmail:v0.8.2-beta

# 运行容器
docker run -d \
  --name tempmail \
  --restart unless-stopped \
  -p 8080:8080 \
  -p 25:25 \
  -e TEMPMAIL_JWT_SECRET="your-production-secret-key-32-chars-min" \
  -e TEMPMAIL_SMTP_DOMAIN=temp.example.com \
  -e TEMPMAIL_MAILBOX_ALLOWED_DOMAINS=temp.example.com \
  -v /opt/tempmail/logs:/app/logs \
  your-registry/tempmail:v0.8.2-beta
```

### 2. 使用 Docker Compose

创建 `docker-compose.yml`：

```yaml
version: '3.8'

services:
  tempmail:
    image: your-registry/tempmail:v0.8.2-beta
    container_name: tempmail
    restart: unless-stopped
    ports:
      - "8080:8080"
      - "25:25"
    environment:
      - TEMPMAIL_JWT_SECRET=${JWT_SECRET}
      - TEMPMAIL_SERVER_HOST=0.0.0.0
      - TEMPMAIL_SERVER_PORT=8080
      - TEMPMAIL_SMTP_BIND_ADDR=:25
      - TEMPMAIL_SMTP_DOMAIN=${SMTP_DOMAIN}
      - TEMPMAIL_MAILBOX_ALLOWED_DOMAINS=${ALLOWED_DOMAINS}
      - TEMPMAIL_LOG_LEVEL=info
      - TEMPMAIL_DATABASE_DSN=postgres://tempmail:${DB_PASSWORD}@postgres:5432/tempmail
      - TEMPMAIL_REDIS_ADDRESS=redis:6379
      - TEMPMAIL_REDIS_PASSWORD=${REDIS_PASSWORD}
    depends_on:
      - postgres
      - redis
    volumes:
      - ./logs:/app/logs
    networks:
      - tempmail-network

  postgres:
    image: postgres:15-alpine
    container_name: tempmail-postgres
    restart: unless-stopped
    environment:
      - POSTGRES_DB=tempmail
      - POSTGRES_USER=tempmail
      - POSTGRES_PASSWORD=${DB_PASSWORD}
    volumes:
      - postgres-data:/var/lib/postgresql/data
    networks:
      - tempmail-network

  redis:
    image: redis:7-alpine
    container_name: tempmail-redis
    restart: unless-stopped
    command: redis-server --requirepass ${REDIS_PASSWORD}
    volumes:
      - redis-data:/data
    networks:
      - tempmail-network

  nginx:
    image: nginx:alpine
    container_name: tempmail-nginx
    restart: unless-stopped
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf:ro
      - ./ssl:/etc/nginx/ssl:ro
    depends_on:
      - tempmail
    networks:
      - tempmail-network

volumes:
  postgres-data:
  redis-data:

networks:
  tempmail-network:
    driver: bridge
```

创建 `.env` 文件：

```bash
JWT_SECRET=your-production-jwt-secret-at-least-32-characters
SMTP_DOMAIN=temp.example.com
ALLOWED_DOMAINS=temp.example.com,mail.example.com
DB_PASSWORD=secure-database-password
REDIS_PASSWORD=secure-redis-password
```

启动服务：

```bash
docker-compose up -d
```

---

## 🔒 安全配置

### 1. Nginx 反向代理

创建 `nginx.conf`：

```nginx
upstream tempmail_backend {
    server tempmail:8080;
}

# HTTP 重定向到 HTTPS
server {
    listen 80;
    server_name temp.example.com;
    return 301 https://$server_name$request_uri;
}

# HTTPS 服务
server {
    listen 443 ssl http2;
    server_name temp.example.com;

    # SSL 证书
    ssl_certificate /etc/nginx/ssl/cert.pem;
    ssl_certificate_key /etc/nginx/ssl/key.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;
    ssl_prefer_server_ciphers on;

    # 安全头
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;
    add_header Referrer-Policy "no-referrer-when-downgrade" always;

    # 限速
    limit_req_zone $binary_remote_addr zone=api_limit:10m rate=10r/s;
    limit_req zone=api_limit burst=20 nodelay;

    # 代理配置
    location / {
        proxy_pass http://tempmail_backend;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # WebSocket 支持
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";

        # 超时配置
        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
    }

    # 静态文件缓存
    location ~* \.(jpg|jpeg|png|gif|ico|css|js)$ {
        expires 7d;
        add_header Cache-Control "public, immutable";
    }
}
```

### 2. 防火墙配置

```bash
# Ubuntu/Debian (UFW)
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw allow 25/tcp  # SMTP（仅限内网或信任IP）
sudo ufw enable

# CentOS/RHEL (firewalld)
sudo firewall-cmd --permanent --add-service=http
sudo firewall-cmd --permanent --add-service=https
sudo firewall-cmd --permanent --add-port=25/tcp
sudo firewall-cmd --reload
```

### 3. 邮件服务器 MX 记录配置

在 DNS 中添加 MX 记录：

```
temp.example.com.   IN  MX  10  mail.example.com.
mail.example.com.   IN  A       your.server.ip
```

---

## 📊 监控和维护

### 1. 健康检查

```bash
# 手动检查
curl https://temp.example.com/health

# 自动监控脚本
cat > /opt/tempmail/monitor.sh << 'EOF'
#!/bin/bash
if ! curl -sf http://localhost:8080/health > /dev/null; then
    echo "TempMail service is down!" | mail -s "Alert" admin@example.com
    systemctl restart tempmail
fi
EOF

chmod +x /opt/tempmail/monitor.sh

# 添加到 crontab（每5分钟检查）
(crontab -l; echo "*/5 * * * * /opt/tempmail/monitor.sh") | crontab -
```

### 2. 日志管理

```bash
# 查看实时日志
sudo journalctl -u tempmail -f

# 查看错误日志
sudo journalctl -u tempmail -p err -n 100

# 日志轮转配置 /etc/logrotate.d/tempmail
/opt/tempmail/logs/*.log {
    daily
    rotate 30
    compress
    delaycompress
    missingok
    notifempty
    create 0640 tempmail tempmail
    sharedscripts
    postrotate
        systemctl reload tempmail > /dev/null 2>&1 || true
    endscript
}
```

### 3. 备份策略

```bash
# 备份脚本
cat > /opt/tempmail/backup.sh << 'EOF'
#!/bin/bash
BACKUP_DIR="/backup/tempmail"
DATE=$(date +%Y%m%d-%H%M%S)

# 备份 PostgreSQL
pg_dump -h localhost -U tempmail tempmail | gzip > "$BACKUP_DIR/db-$DATE.sql.gz"

# 清理30天前的备份
find "$BACKUP_DIR" -name "*.gz" -mtime +30 -delete
EOF

chmod +x /opt/tempmail/backup.sh

# 每天凌晨2点备份
(crontab -l; echo "0 2 * * * /opt/tempmail/backup.sh") | crontab -
```

### 4. 性能监控

使用 Prometheus 和 Grafana（待实现）：

```bash
# TODO: 添加 Prometheus metrics 端点
# GET /metrics
```

---

## 🔧 故障排查

### 常见问题

#### 1. 服务无法启动

```bash
# 检查日志
sudo journalctl -u tempmail -n 100

# 检查配置
cat /etc/tempmail/config.env

# 检查端口占用
sudo lsof -i :8080
sudo lsof -i :25
```

#### 2. 无法接收邮件

```bash
# 测试 SMTP 连接
telnet localhost 25

# 检查 DNS MX 记录
dig MX temp.example.com

# 查看 SMTP 日志
sudo journalctl -u tempmail | grep smtp
```

#### 3. 数据库连接失败

```bash
# 检查 PostgreSQL 状态
sudo systemctl status postgresql

# 测试数据库连接
psql -h localhost -U tempmail -d tempmail
```

---

## 📚 相关文档

- [README.md](README.md) - 项目概述
- [CLAUDE.md](CLAUDE.md) - 开发指南
- [CHANGELOG.md](CHANGELOG.md) - 更新日志

---

**最后更新**: 2025-10-12
**版本**: v0.8.2-beta
