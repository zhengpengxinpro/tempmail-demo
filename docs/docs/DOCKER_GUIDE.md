# Docker 部署指南

本文档说明如何使用 Docker Compose 一键部署临时邮箱系统。

## 前置要求

- Docker 26.x+
- Docker Compose v2.27+
- 至少 4GB 内存
- 至少 20GB 磁盘空间

## 快速启动

### 1. 克隆代码

```bash
git clone <your-repo-url> tempmail
cd tempmail
```

### 2. 启动所有服务

```bash
docker compose up -d
```

这将启动以下服务：
- **PostgreSQL** (端口 5432) - 数据库
- **Redis** (端口 6379) - 缓存和消息队列
- **Backend** (端口 8080, 25) - HTTP API 和 SMTP 服务
- **Frontend User** (端口 5173) - 用户前台
- **Frontend Admin** (端口 5273) - 管理后台

### 3. 验证服务状态

```bash
# 查看所有服务状态
docker compose ps

# 查看后端日志
docker compose logs -f backend

# 查看所有日志
docker compose logs -f
```

### 4. 访问应用

- 用户前台: http://localhost:5173
- 管理后台: http://localhost:5273
- API 文档: http://localhost:8080/health
- PostgreSQL: localhost:5432 (用户名: tempmail, 密码: tempmail123)
- Redis: localhost:6379 (密码: tempmail123)

## 初始化数据库

首次启动时，数据库会自动创建表结构。如果需要手动运行迁移：

```bash
# 进入 postgres 容器
docker compose exec postgres psql -U tempmail -d tempmail

# 或使用 migrate 工具
docker compose exec backend migrate -path /app/migrations -database "postgres://tempmail:tempmail123@postgres:5432/tempmail?sslmode=disable" up
```

## 默认账户

系统会自动创建一个管理员账户：

- 邮箱: `admin@tempmail.dev`
- 密码: `admin123`

**⚠️ 生产环境请立即修改密码！**

## 环境配置

编辑 `docker-compose.yml` 中的环境变量来自定义配置：

### 关键配置项

```yaml
# JWT 密钥（必须修改）
TEMPMAIL_JWT_SECRET: your-super-secret-jwt-key-change-in-production

# 允许的邮箱域名
TEMPMAIL_MAILBOX_ALLOWED_DOMAINS: temp.mail,tempmail.dev,your-domain.com

# CORS 允许的源
TEMPMAIL_CORS_ALLOWED_ORIGINS: http://localhost:5173,https://yourdomain.com

# 数据库连接
TEMPMAIL_DATABASE_DSN: postgres://tempmail:tempmail123@postgres:5432/tempmail?sslmode=disable

# Redis 连接
TEMPMAIL_REDIS_ADDRESS: redis:6379
TEMPMAIL_REDIS_PASSWORD: tempmail123
```

## 常用命令

### 启动服务

```bash
# 启动所有服务
docker compose up -d

# 启动特定服务
docker compose up -d backend redis postgres

# 查看实时日志
docker compose logs -f backend
```

### 停止服务

```bash
# 停止所有服务
docker compose stop

# 停止特定服务
docker compose stop backend
```

### 重启服务

```bash
# 重启所有服务
docker compose restart

# 重启特定服务
docker compose restart backend
```

### 清理和重建

```bash
# 停止并删除容器（保留数据卷）
docker compose down

# 停止并删除容器及数据卷（⚠️ 会删除所有数据）
docker compose down -v

# 重新构建镜像
docker compose build

# 重新构建并启动
docker compose up -d --build
```

### 数据备份

```bash
# 备份 PostgreSQL
docker compose exec -T postgres pg_dump -U tempmail tempmail > backup.sql

# 恢复 PostgreSQL
docker compose exec -T postgres psql -U tempmail tempmail < backup.sql

# 备份 Redis
docker compose exec redis redis-cli --raw SAVE
docker cp tempmail-redis:/data/dump.rdb ./redis-backup.rdb
```

## 生产环境部署

### 1. 使用外部数据库和 Redis

编辑 `docker-compose.yml`，移除 postgres 和 redis 服务，修改 backend 环境变量：

```yaml
services:
  backend:
    environment:
      TEMPMAIL_DATABASE_DSN: postgres://user:pass@external-db:5432/tempmail?sslmode=require
      TEMPMAIL_REDIS_ADDRESS: external-redis:6379
      TEMPMAIL_REDIS_PASSWORD: strong-redis-password
```

### 2. 启用 Nginx 反向代理

```bash
# 使用 production profile 启动
docker compose --profile production up -d
```

这将启动 Nginx 反向代理，统一处理 HTTP/HTTPS 请求。

### 3. SSL/TLS 配置

将 SSL 证书放置在 `deploy/nginx/certs/` 目录，并修改 Nginx 配置。

### 4. 环境变量安全

不要在 `docker-compose.yml` 中硬编码敏感信息。使用 `.env` 文件：

```bash
# 创建 .env 文件
cat > .env <<EOF
POSTGRES_PASSWORD=your-secure-password
REDIS_PASSWORD=your-redis-password
JWT_SECRET=your-jwt-secret-key
EOF

# 修改 docker-compose.yml 使用环境变量
environment:
  TEMPMAIL_DATABASE_DSN: postgres://tempmail:${POSTGRES_PASSWORD}@postgres:5432/tempmail
```

## 监控和日志

### 查看日志

```bash
# 实时查看所有日志
docker compose logs -f

# 查看特定服务日志
docker compose logs -f backend

# 查看最近 100 行日志
docker compose logs --tail=100 backend
```

### 资源使用情况

```bash
# 查看容器资源使用
docker stats

# 查看磁盘使用
docker system df
```

## 性能优化

### 1. 调整资源限制

编辑 `docker-compose.yml`，为服务添加资源限制：

```yaml
services:
  backend:
    deploy:
      resources:
        limits:
          cpus: '2'
          memory: 2G
        reservations:
          cpus: '0.5'
          memory: 512M
```

### 2. 数据库连接池

调整 `TEMPMAIL_DATABASE_MAX_OPEN_CONNS` 和 `TEMPMAIL_DATABASE_MAX_IDLE_CONNS` 根据负载情况。

### 3. Redis 持久化

修改 Redis 配置启用 AOF：

```bash
docker compose exec redis redis-cli CONFIG SET appendonly yes
```

## 故障排查

### 后端服务无法连接数据库

```bash
# 检查 PostgreSQL 是否健康
docker compose ps postgres

# 查看 PostgreSQL 日志
docker compose logs postgres

# 测试连接
docker compose exec backend ping postgres
```

### 前端无法连接后端

1. 检查 `VITE_APP_API_BASE` 环境变量是否正确
2. 检查 CORS 配置是否包含前端地址
3. 查看浏览器控制台错误

### 邮件无法接收

1. 确认 SMTP 端口 25 已开放
2. 检查 `TEMPMAIL_MAILBOX_ALLOWED_DOMAINS` 配置
3. 测试 SMTP 连接：
   ```bash
   telnet localhost 25
   ```

### 容器频繁重启

```bash
# 查看容器退出原因
docker compose logs backend | grep -i error

# 查看容器状态
docker inspect tempmail-backend
```

## 更新和升级

```bash
# 拉取最新代码
git pull origin main

# 重新构建并启动
docker compose up -d --build

# 运行数据库迁移（如有新迁移）
docker compose exec backend migrate -path /app/migrations -database "$DATABASE_URL" up
```

## 卸载

```bash
# 停止并删除所有容器和数据卷
docker compose down -v

# 删除镜像
docker compose down --rmi all

# 清理未使用的 Docker 资源
docker system prune -a --volumes
```

## 安全建议

1. **修改默认密码**：数据库、Redis、管理员账户
2. **使用强 JWT 密钥**：至少 32 字节随机字符串
3. **启用 HTTPS**：生产环境必须使用 SSL/TLS
4. **限制端口暴露**：不要暴露数据库和 Redis 端口到公网
5. **定期备份**：配置自动备份脚本
6. **监控日志**：集成日志聚合工具（如 ELK）
7. **更新依赖**：定期更新 Docker 镜像

## 支持

如遇问题，请查看：
- [技术栈文档](docs/TECH_STACK.md)
- [安装指南](docs/INSTALLATION_GUIDE.md)
- [PRD 文档](docs/PRD.md)
