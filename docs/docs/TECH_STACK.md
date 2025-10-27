# 临时邮箱系统 - 技术栈文档

## 1. 技术架构总览

### 1.1 架构图

```
┌─────────────────────────────────────────────────────────────┐
│                    前端层 (3个独立应用)                      │
├─────────────────────────────────────────────────────────────┤
│  用户端          │  管理端          │  公共端（游客）        │
│  Vue3 + Arco     │  Vue3 + Arco     │  Vue3 + Arco          │
│  TypeScript      │  TypeScript      │  TypeScript           │
└─────────────────────────────────────────────────────────────┘
                          ↓ HTTPS
┌─────────────────────────────────────────────────────────────┐
│                    Web 服务器层                              │
│                  Nginx / Caddy                               │
│  - SSL/TLS 终止                                              │
│  - 反向代理                                                  │
│  - 负载均衡                                                  │
│  - WebSocket 升级                                            │
└─────────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────────┐
│                   后端服务层 (Go 1.25.2)                     │
├─────────────────────────────────────────────────────────────┤
│  API 服务        │  SMTP 服务       │  定时任务服务         │
│  (Gin)           │  (go-smtp)       │  (cron)               │
│  - REST API      │  - 邮件接收      │  - 邮件清理           │
│  - WebSocket     │  - MIME 解析     │  - 配额重置           │
│  - 认证授权      │  - 多域名支持    │  - 统计汇总           │
└─────────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────────┐
│                      数据存储层                              │
├─────────────────────────────────────────────────────────────┤
│  PostgreSQL 18.0        │  Redis 8.2 Cluster               │
│  - 用户数据              │  - 邮件内容 (TTL)                │
│  - 域名配置              │  - 会话缓存                      │
│  - 权限规则 (Casbin)     │  - 限流计数                      │
│  - 审计日志              │  - PubSub 消息                   │
└─────────────────────────────────────────────────────────────┘
```

### 1.2 技术选型原则

**后端**：
- ✅ **性能优先**：选择 Go 语言，高并发、低延迟
- ✅ **成熟稳定**：使用生产验证的库和框架
- ✅ **轻量可控**：避免过度设计，代码完全可控
- ✅ **易于扩展**：无状态设计，支持水平扩展

**前端**：
- ✅ **现代化**：Vue 3.5.22 + TypeScript 5.9.3 + Vite 7.1.9
- ✅ **美观高效**：Arco Design Pro 模板
- ✅ **开发体验**：完善的工具链和代码规范

**数据库**：
- ✅ **PostgreSQL**：ACID 保证，适合关系型数据
- ✅ **Redis**：高性能缓存，适合临时数据和消息

---

## 2. 后端技术栈

### 2.1 核心框架

```yaml
语言: Go 1.25.2

Web 框架:
  - github.com/gin-gonic/gin v1.11.0
    用途: HTTP 服务器、REST API、路由
    选择理由: 性能优异、社区活跃、中间件丰富

配置管理:
  - github.com/spf13/viper
    用途: 配置文件管理 (支持 YAML/JSON/ENV)

日志:
  - go.uber.org/zap
    用途: 结构化日志、高性能
```

### 2.2 认证授权

```yaml
认证框架:
  - github.com/shaj13/go-guardian/v2 v2.11.6
    用途: 认证策略框架，支持多种认证方式
    Star: 9000+

JWT:
  - github.com/golang-jwt/jwt/v5 v5.3.0
    用途: JWT Token 生成与验证
    Star: 6000+

密码加密:
  - golang.org/x/crypto/bcrypt
    用途: 密码 Hash（官方库）

权限控制:
  - github.com/casbin/casbin/v2 v2.128.0
    用途: RBAC 权限模型
    Star: 17000+
  - github.com/casbin/gorm-adapter/v3 v3.37.0
    用途: Casbin 的 GORM 适配器
```

### 2.3 数据库

```yaml
ORM:
  - gorm.io/gorm v1.31.0
    用途: ORM 框架
    Star: 36000+
  - gorm.io/driver/postgres
    用途: PostgreSQL 驱动

数据库迁移:
  - github.com/golang-migrate/migrate/v4 v4.19.0
    用途: 数据库版本管理和迁移

Redis:
  - github.com/redis/go-redis/v9 v9.14.0
    用途: Redis 客户端（支持 Cluster）
    Star: 19000+
```

### 2.4 SMTP 与邮件

```yaml
SMTP 服务器:
  - github.com/emersion/go-smtp v0.24.0
    用途: SMTP 协议实现，接收邮件
    Star: 2000+
    特点: 轻量、易用、支持 TLS

邮件解析:
  - github.com/emersion/go-message v0.18.2
    用途: MIME 邮件解析（HTML/文本/附件）
    Star: 300+
    替代: postal-mime (JS库的 Go 版本)
```

### 2.5 限流与并发

```yaml
限流器:
  - github.com/ulule/limiter/v3 v3.11.2
    用途: 令牌桶限流算法
    Star: 2000+
    特点: 支持 Redis 存储、灵活配置

并发控制:
  - golang.org/x/sync/semaphore
    用途: 信号量，控制并发数
    官方库
```

### 2.6 WebSocket

```yaml
WebSocket:
  - github.com/gorilla/websocket
    用途: WebSocket 协议实现
    Star: 22000+
    特点: 标准化、稳定可靠
```

### 2.7 工具库

```yaml
UUID:
  - github.com/google/uuid
    用途: 生成唯一标识符

参数验证:
  - github.com/go-playground/validator/v10 v10.28.0
    用途: 结构体参数验证
    Star: 16000+

定时任务:
  - github.com/robfig/cron/v3
    用途: Cron 定时任务调度
    Star: 13000+

CLI 工具:
  - github.com/spf13/cobra
    用途: 命令行工具框架
    Star: 37000+
```

### 2.8 测试

```yaml
测试框架:
  - github.com/stretchr/testify
    用途: 断言和 Mock
    Star: 23000+

数据库 Mock:
  - github.com/DATA-DOG/go-sqlmock
    用途: SQL Mock

Redis Mock:
  - github.com/go-redis/redismock/v9 v9.2.0
    用途: Redis Mock
```

---

## 3. 前端技术栈

### 3.1 基础框架

```yaml
模板: Arco Design Pro (字节跳动)
仓库: https://github.com/arco-design/arco-design-pro-vue

核心框架:
  - Vue 3.5.22
    用途: 渐进式前端框架

  - TypeScript 5.9.3
    用途: 类型安全、代码提示

  - Vite 7.1.9
    用途: 构建工具、开发服务器
    特点: 极速热更新、ESM 原生支持
```

### 3.2 UI 组件库

```yaml
UI 库:
  - Arco Design Vue 2.57.0
    用途: 企业级 UI 组件库
    特点: 现代设计、性能优秀、Tree Shaking
    Star: 2500+

图标:
  - @arco-design/web-vue/es/icon
    用途: 图标库（集成在 Arco Design 中）
```

### 3.3 路由与状态

```yaml
路由:
  - Vue Router 4.5.1
    用途: 官方路由管理
    特点: 支持动态路由、路由守卫

状态管理:
  - Pinia 3.0.3
    用途: 官方状态管理（Vue 团队推荐的 Pinia）
    特点: TypeScript 友好、轻量
```

### 3.4 HTTP 与实时通信

```yaml
HTTP 客户端:
  - Axios 1.6+
    用途: HTTP 请求库
    特点: 拦截器、请求/响应转换

WebSocket:
  - Native WebSocket API
    用途: 实时推送
    特点: 浏览器原生支持
```

### 3.5 工具库

```yaml
组合式工具:
  - @vueuse/core 13.9.0
    用途: Vue 组合式 API 工具集
    Star: 19000+
    常用: useWebSocket, useIntervalFn, useTitle

时间处理:
  - dayjs 1.11+
    用途: 轻量级时间库（Moment.js 替代）
    大小: 2KB

工具函数:
  - lodash-es 4.17+
    用途: 常用工具函数（ES 模块版本）

HTML 安全:
  - dompurify 3+
    用途: HTML 清理，防 XSS
    Star: 13000+
```

### 3.6 样式

```yaml
预处理器:
  - Scss
    用途: CSS 预处理器
    特点: 变量、嵌套、Mixin

CSS 工具:
  - autoprefixer
    用途: 自动添加浏览器前缀
```

### 3.7 代码规范

```yaml
ESLint:
  - eslint 9.37.0
  - @typescript-eslint/parser
  - @typescript-eslint/eslint-plugin
  - eslint-plugin-vue
    用途: JavaScript/TypeScript/Vue 代码检查

Prettier:
  - prettier 3.6.2
    用途: 代码格式化

Stylelint:
  - stylelint 16.25.0
    用途: CSS/SCSS 代码检查

Git Hooks:
  - husky 9.1.7
    用途: Git 钩子管理

  - lint-staged
    用途: 只对暂存文件执行 lint

Commit 规范:
  - @commitlint/cli
  - @commitlint/config-conventional
    用途: Commit 消息检查

  - cz-git
    用途: Commitizen 适配器（中文友好）
```

### 3.8 构建优化

```yaml
Vite 插件:
  - @vitejs/plugin-vue
    用途: Vue 3 支持

  - vite-plugin-compression
    用途: Gzip/Brotli 压缩

  - unplugin-auto-import
    用途: 自动导入 API

  - unplugin-vue-components
    用途: 组件自动导入

  - vite-plugin-vue-devtools
    用途: Vue DevTools 集成
```

---

## 4. 数据库设计

### 4.1 PostgreSQL 18.0

**选择理由**：
- ✅ ACID 保证，数据可靠
- ✅ 强大的 JSON 支持（审计日志）
- ✅ 全文搜索功能
- ✅ 成熟的生态系统

**核心表**：
```
- users              # 用户表
- user_levels        # 用户等级配置
- domains            # 域名表
- mailboxes          # 邮箱表
- quota_usage        # 配额使用记录
- api_keys           # API 密钥表
- webhooks           # Webhook 配置（可选）
- audit_logs         # 审计日志
- casbin_rule        # Casbin 权限规则
- system_config      # 系统配置
```

**连接配置**：
```yaml
max_open_conns: 50      # 最大连接数
max_idle_conns: 10      # 最大空闲连接
conn_max_lifetime: 3600 # 连接最大生命周期（秒）
```

**备份策略**：
- 每日全量备份（凌晨 2 点）
- 实时 WAL 归档（增量备份）
- 保留 7 天的备份

### 4.2 Redis 8.2 Cluster

**选择理由**：
- ✅ 高性能（内存数据库）
- ✅ 数据结构丰富（String/Hash/List/Set/ZSet）
- ✅ TTL 自动过期（适合临时数据）
- ✅ PubSub 支持（实时消息）
- ✅ Cluster 模式（高可用）

**数据结构**：
```redis
# 邮件内容存储
mail:{id}                      # Hash, TTL: 24h
mailbox:{email}                # List (邮件 ID 列表)

# 认证相关
token:blacklist:{jti}          # String, 已登出的 Token
refresh:{token_id}             # String, Refresh Token

# 限流计数
ratelimit:user:{id}:min:{ts}  # String, 每分钟限流
ratelimit:user:{id}:day:{date} # String, 每天限流
login:attempts:{ip}            # String, 登录尝试次数

# 权限缓存
casbin:policy                  # JSON, Casbin 策略缓存

# 实时推送
PubSub: new_mail:{email}       # 新邮件通知频道
```

**集群配置**：
```yaml
cluster_mode: true
nodes: 6                   # 3 主 + 3 从
max_redirects: 3           # 最大重定向次数
pool_size: 50              # 连接池大小
```

---

## 5. 部署架构

### 5.1 容器化 (Docker)

**Docker 镜像**：
```dockerfile
# API 服务
FROM golang:1.25.2-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o api cmd/api/main.go

FROM alpine:latest
COPY --from=builder /app/api /app/api
EXPOSE 8080
CMD ["/app/api"]

# SMTP 服务
FROM golang:1.25.2-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o smtp cmd/smtp/main.go

FROM alpine:latest
COPY --from=builder /app/smtp /app/smtp
EXPOSE 25 587
CMD ["/app/smtp"]

# 前端（多阶段构建）
FROM node:20-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm install
COPY . .
RUN npm run build

FROM nginx:alpine
COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
```

### 5.2 编排 (Docker Compose)

```yaml
version: '3.8'

services:
  # PostgreSQL
  postgres:
    image: postgres:18.0-alpine
    environment:
      POSTGRES_USER: mailuser
      POSTGRES_PASSWORD: ${DB_PASSWORD}
      POSTGRES_DB: maildb
    volumes:
      - postgres_data:/var/lib/postgresql/data
    ports:
      - "5432:5432"

  # Redis Cluster (简化为单节点，生产环境用 Cluster)
  redis:
    image: redis:8.2.2-alpine
    command: redis-server --appendonly yes
    volumes:
      - redis_data:/data
    ports:
      - "6379:6379"

  # API 服务
  api:
    build:
      context: ./backend
      dockerfile: Dockerfile.api
    environment:
      - DB_HOST=postgres
      - REDIS_HOST=redis
    depends_on:
      - postgres
      - redis
    ports:
      - "8080:8080"
    deploy:
      replicas: 3  # 3 个实例

  # SMTP 服务
  smtp:
    build:
      context: ./backend
      dockerfile: Dockerfile.smtp
    environment:
      - DB_HOST=postgres
      - REDIS_HOST=redis
    depends_on:
      - postgres
      - redis
    ports:
      - "25:25"
      - "587:587"
    deploy:
      replicas: 2  # 2 个实例

  # Nginx (反向代理 + 负载均衡)
  nginx:
    image: nginx:alpine
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf
      - ./ssl:/etc/nginx/ssl
    ports:
      - "80:80"
      - "443:443"
    depends_on:
      - api
      - smtp

volumes:
  postgres_data:
  redis_data:
```

### 5.3 Web 服务器 (Nginx)

**Nginx 配置**：
```nginx
upstream api_backend {
    least_conn;
    server api1:8080 max_fails=3 fail_timeout=30s;
    server api2:8080 max_fails=3 fail_timeout=30s;
    server api3:8080 max_fails=3 fail_timeout=30s;
}

server {
    listen 80;
    server_name yourdomain.com;
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name yourdomain.com;

    # SSL 证书
    ssl_certificate /etc/nginx/ssl/fullchain.pem;
    ssl_certificate_key /etc/nginx/ssl/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;

    # API 反向代理
    location /api/ {
        proxy_pass http://api_backend;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }

    # WebSocket 升级
    location /api/v1/mailboxes/ {
        proxy_pass http://api_backend;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }

    # 前端静态资源
    location / {
        root /usr/share/nginx/html;
        try_files $uri $uri/ /index.html;
    }

    # Gzip 压缩
    gzip on;
    gzip_types text/plain text/css text/javascript application/json;
}
```

---

## 6. 开发工具

### 6.1 后端开发

```yaml
IDE:
  - GoLand (JetBrains)
  - VS Code + Go 插件

Go 工具:
  - go mod         # 依赖管理
  - gofmt          # 代码格式化
  - golangci-lint  # 代码检查
  - go test        # 单元测试
  - delve          # 调试器

API 测试:
  - Postman
  - cURL
  - HTTPie

数据库工具:
  - DBeaver        # PostgreSQL 客户端
  - RedisInsight   # Redis 客户端
```

### 6.2 前端开发

```yaml
IDE:
  - WebStorm (JetBrains)
  - VS Code

浏览器插件:
  - Vue DevTools
  - React DevTools (备用)

调试工具:
  - Chrome DevTools
  - Firefox DevTools
```

### 6.3 DevOps 工具

```yaml
版本控制:
  - Git
  - GitHub / GitLab

CI/CD:
  - GitHub Actions
  - GitLab CI
  - Jenkins

容器:
  - Docker Desktop
  - Docker Compose

监控:
  - Prometheus    # 指标采集
  - Grafana       # 可视化
  - ELK Stack     # 日志分析（可选）

性能测试:
  - wrk           # HTTP 压测
  - ab            # Apache Bench
  - k6            # 现代化压测工具
```

---

## 7. 性能指标

### 7.1 目标指标

```yaml
API 性能:
  - P50 响应时间: < 50ms
  - P95 响应时间: < 200ms
  - P99 响应时间: < 500ms
  - QPS: > 10000

SMTP 性能:
  - 邮件接收延迟: < 5s
  - 并发连接数: > 1000

WebSocket:
  - 推送延迟: < 1s
  - 同时在线: > 10000

数据库:
  - PostgreSQL 查询: < 10ms (P95)
  - Redis 查询: < 1ms (P95)

前端:
  - 首屏加载: < 2s
  - TTI (可交互时间): < 3s
  - Lighthouse 分数: > 90
```

### 7.2 资源估算

**小规模（1000 用户）**：
```
服务器:
  - API: 2核4G × 2 台
  - SMTP: 2核4G × 1 台
  - PostgreSQL: 2核4G × 1 台
  - Redis: 2核4G × 1 台

成本: $50-100/月
```

**中等规模（10000 用户）**：
```
服务器:
  - API: 4核8G × 3 台
  - SMTP: 4核8G × 2 台
  - PostgreSQL: 8核16G × 1 台 (主) + 4核8G × 1 台 (从)
  - Redis: 4核8G × 3 台 (Cluster)

成本: $300-500/月
```

**大规模（100000+ 用户）**：
```
使用 Kubernetes 集群
成本: $1000+/月
```

---

## 8. 安全措施

### 8.1 应用层安全

```yaml
认证:
  - bcrypt 密码加密（Cost: 12）
  - JWT Token（HMAC-SHA256）
  - API Key 签名验证（HMAC-SHA256）
  - Refresh Token 轮换

权限:
  - Casbin RBAC 权限控制
  - API Scopes 细粒度权限

防护:
  - XSS 防护（DOMPurify）
  - SQL 注入防护（参数化查询）
  - CSRF 防护（SameSite Cookie）
  - 限流（防 DDoS）
  - 请求体大小限制（10MB）
```

### 8.2 网络层安全

```yaml
传输加密:
  - 全站 HTTPS (TLS 1.2+)
  - Let's Encrypt 自动证书

SMTP 安全:
  - STARTTLS 支持
  - SPF/DMARC 验证

防火墙:
  - 仅开放必要端口（80, 443, 25, 587）
  - IP 白名单（管理后台）
```

---

## 9. 监控与日志

### 9.1 应用监控

```yaml
指标:
  - HTTP 请求数、响应时间、错误率
  - SMTP 接收数、失败数
  - WebSocket 连接数
  - 数据库连接池状态
  - Redis 命中率

工具:
  - Prometheus (指标采集)
  - Grafana (可视化)
```

### 9.2 日志管理

```yaml
日志类型:
  - 访问日志 (Access Log)
  - 错误日志 (Error Log)
  - 审计日志 (Audit Log)
  - 慢查询日志 (Slow Query Log)

日志格式:
  - 结构化日志 (JSON)
  - 包含 Trace ID (分布式追踪)

存储:
  - 本地文件 (短期)
  - ELK Stack (长期，可选)
```

---

## 10. 开发环境配置

### 10.1 本地开发

**后端**：
```bash
# 安装 Go 1.25.2
go version

# 安装依赖
go mod download

# 运行 API 服务
go run cmd/api/main.go

# 运行 SMTP 服务
go run cmd/smtp/main.go

# 运行测试
go test ./...
```

**前端**：
```bash
# 安装 Node.js 22+
node -v

# 安装依赖
pnpm install

# 运行开发服务器
pnpm dev

# 构建生产版本
pnpm build
```

**数据库**：
```bash
# 启动 PostgreSQL (Docker)
docker run -d -p 5432:5432 \
  -e POSTGRES_USER=mailuser \
  -e POSTGRES_PASSWORD=password \
  -e POSTGRES_DB=maildb \
  postgres:18.0-alpine

# 启动 Redis (Docker)
docker run -d -p 6379:6379 redis:8.2.2-alpine
```

### 10.2 环境变量

**后端 `.env`**：
```bash
# 服务器
API_PORT=8080
SMTP_PORT=25
MODE=development

# 数据库
DB_HOST=localhost
DB_PORT=5432
DB_USER=mailuser
DB_PASSWORD=password
DB_NAME=maildb

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379

# JWT
JWT_SECRET=your-secret-key

# 日志
LOG_LEVEL=debug
```

**前端 `.env.development`**：
```bash
VITE_API_BASE_URL=http://localhost:8080
VITE_WS_BASE_URL=ws://localhost:8080
VITE_APP_TITLE=临时邮箱
```

---

## 11. 未来技术规划

### 11.1 V1.1 增强功能（第2-3个月）

#### 附件下载支持
```yaml
新增技术:
  - 对象存储: MinIO / AWS S3
    - github.com/minio/minio-go/v7 v7.0.95
    - github.com/aws/aws-sdk-go-v2/service/s3
  - 病毒扫描: ClamAV Go客户端
  - 文件类型检测: github.com/h2non/filetype

前端:
  - 文件下载: FileSaver.js
  - 进度条: @arco-design/web-vue Progress
```

#### 邮件搜索功能
```yaml
搜索引擎:
  方案A - PostgreSQL全文搜索 (小规模):
    - pg_trgm 扩展（模糊搜索）

  方案B - Meilisearch (推荐):
    - github.com/meilisearch/meilisearch-go
    - 特点: 轻量、快速、易部署

  方案C - Elasticsearch (大规模):
    - github.com/elastic/go-elasticsearch/v8 v9.0.0
    - 特点: 功能强大、高性能

前端:
  - @arco-design/web-vue AutoComplete
  - 高亮显示: highlight.js
```

#### Webhook 回调
```yaml
后端:
  - 任务队列: github.com/hibiken/asynq
  - HTTP重试: github.com/hashicorp/go-retryablehttp
  - 签名: HMAC-SHA256

前端:
  - Webhook配置界面
  - 日志查看
  - 测试发送
```

### 11.2 V1.2 商业化功能（第4-5个月）

#### 支付集成
```yaml
支付网关:
  - Stripe (推荐):
    - github.com/stripe/stripe-go/v83.0.1
    - 功能: 订阅、支付、退款
    - Webhook 通知

  - PayPal (备选):
    - github.com/plutov/paypal/v4 v4.17.0

订阅管理:
  - 数据表: subscriptions, payment_transactions
  - 定时任务: 订阅到期检查

前端:
  - Stripe Elements (官方组件)
  - 支付表单
  - 交易历史
```

#### 发票生成
```yaml
后端:
  - PDF生成:
    - github.com/jung-kurt/gofpdf
    - 或: github.com/signintech/gopdf
  - 模板引擎: html/template
  - 邮件发送: gopkg.in/gomail.v2

前端:
  - 发票列表
  - PDF预览
  - 下载按钮
```

### 11.3 V2.0 高级功能（第6-12个月）

#### 自定义域名
```yaml
新增技术:
  - DNS管理:
    - github.com/miekg/dns (DNS查询)
    - Cloudflare API (github.com/cloudflare/cloudflare-go)

  - SSL证书:
    - Let's Encrypt ACME协议
    - github.com/go-acme/lego/v4 v4.26.0
    - 自动证书申请和续期

挑战:
  - 动态SSL证书加载
  - SNI支持
  - 多租户域名隔离
```

#### 邮件发送功能
```yaml
后端:
  - SMTP客户端: github.com/emersion/go-smtp (扩展)
  - 发送队列: github.com/hibiken/asynq
  - SPF/DKIM: github.com/emersion/go-msgauth
  - 反垃圾邮件: 频率限制、内容过滤

前端:
  - 富文本编辑器:
    - @wangeditor/editor-for-vue
    - 或 Quill.js
  - 收件人管理
  - 发送统计
```

#### 多语言支持
```yaml
后端:
  - i18n库:
    - github.com/nicksnyder/go-i18n/v2 v2.6.0
  - 语言包: JSON/YAML格式
  - 支持语言: en, zh-CN, zh-TW, ja, ko

前端:
  - vue-i18n 11.1 (官方)
  - 语言检测: 浏览器语言
  - 动态切换
```

#### 移动端 App
```yaml
技术选型:
方案A - Flutter 3.35.6 (推荐):
    - 语言: Dart
    - 优势: 性能好、UI美观
    - 推送: FCM/APNs

  方案B - React Native:
    - 语言: JavaScript/TypeScript
    - 优势: 复用Web技能

  方案C - PWA (过渡):
    - 基于现有Vue3前端
    - Service Worker
    - 优势: 无需单独开发
```

### 11.4 未来基础设施升级

#### 微服务架构（大规模时）
```yaml
服务拆分:
  - 用户服务、邮箱服务、邮件服务、域名服务、支付服务

服务通信:
  - gRPC: google.golang.org/grpc
  - Protocol Buffers
  - 服务网格: Istio / Linkerd (可选)

服务发现:
  - Consul: github.com/hashicorp/consul/api
  - 或: etcd, Nacos

API网关:
  - Kong 或 Traefik
```

#### 消息队列（大规模时）
```yaml
消息队列:
  - Kafka (大数据量):
    - github.com/segmentio/kafka-go
    - 用途: 邮件处理流水线

  - RabbitMQ (中等规模):
    - github.com/streadway/amqp

  - NATS (轻量级):
    - github.com/nats-io/nats.go
```

#### 可观测性增强
```yaml
分布式追踪:
  - OpenTelemetry:
    - go.opentelemetry.io/otel
  - Jaeger (追踪后端)

日志聚合:
  - ELK Stack:
    - Elasticsearch + Logstash + Kibana
  - 轻量级: Loki + Grafana

告警:
  - AlertManager (Prometheus)
  - 或: PagerDuty
```

#### AI/ML 功能（长期）
```yaml
垃圾邮件过滤:
  - 机器学习模型: 朴素贝叶斯
  - 或: TensorFlow Lite
  - Go ML库: github.com/sjwhitworth/golearn

智能分类:
  - NLP: github.com/jdkato/prose
  - 邮件自动分类
```

---

## 12. 技术选型决策矩阵

| 功能 | 技术方案 | 时间节点 | 复杂度 | 优先级 |
|------|---------|---------|--------|--------|
| 附件下载 | MinIO/S3 | V1.1 | ⭐⭐ | 高 |
| 邮件搜索 | Meilisearch | V1.1 | ⭐⭐⭐ | 高 |
| Webhook | Asynq | V1.1 | ⭐⭐ | 中 |
| 支付集成 | Stripe | V1.2 | ⭐⭐⭐ | 高 |
| 自定义域名 | ACME/DNS API | V2.0 | ⭐⭐⭐⭐ | 中 |
| 邮件发送 | SMTP Client | V2.0 | ⭐⭐⭐ | 中 |
| 多语言 | vue-i18n 11.1 | V2.0 | ⭐⭐ | 中 |
| 移动端 | Flutter 3.35.6 | V2.0+ | ⭐⭐⭐⭐⭐ | 低 |
| 微服务 | gRPC | 按需 | ⭐⭐⭐⭐⭐ | 低 |

---

## 13. 技术决策原则

### 13.1 选型原则

1. **优先成熟稳定**：选择生产验证的技术，避免踩坑
2. **适度超前**：为未来1年的增长留出空间
3. **避免过度设计**：不要为了技术而技术
4. **团队技能匹配**：考虑团队学习成本
5. **开源优先**：避免供应商锁定

### 13.2 技术债务控制

- ✅ 每个版本预留20%时间重构
- ✅ 定期code review
- ✅ 单元测试覆盖率>80%
- ✅ 技术分享会（每月）

### 13.3 版本发布节奏

```
V1.0 (MVP)        → 38-40天    基础功能
V1.0.1            → +7天       Bug修复
V1.1 (增强)       → +30天      高级功能
V1.2 (商业化)     → +30天      支付功能
V2.0 (重大更新)   → +90-180天  自定义域名、移动端
```

---

## 14. 总结

### 14.1 核心技术栈一览

**后端（Go 1.25.2）**:
```
Web框架: Gin v1.11.0
认证: go-guardian v2.11.6 + JWT v5.3.0 + bcrypt
权限: Casbin v2.128.0
数据库: GORM v1.31.0 + PostgreSQL 18 + Redis 8
SMTP: go-smtp v0.24.0 + go-message v0.18.2
限流: ulule/limiter v3.11.2
WebSocket: gorilla/websocket v1.5.3
工具: viper, zap, uuid, validator, cron
```

**前端（Vue3 + TypeScript）**:
```
模板: Arco Design Pro
框架: Vue 3.5.22 + TypeScript 5.9.3 + Vite 7.1.9
UI: Arco Design Vue 2.57.0
路由: Vue Router 4.5.1
状态: Pinia 3.0.3
HTTP: Axios 1.12.2
工具: VueUse 13.9.0, dayjs 1.11.18, lodash-es 4.17.21, dompurify 3.2.7
规范: ESLint + Prettier + Stylelint + Husky
```

**基础设施**:
```
数据库: PostgreSQL 18 + Redis 8 Cluster
Web服务器: Nginx / Caddy
容器: Docker + Docker Compose
监控: Prometheus + Grafana
```

### 14.2 开发周期

- **总工期**：38-40 天
- **团队配置**：
  - 1 全栈：5-6 周
  - 1 后端 + 1 前端：3-4 周
  - 2 后端 + 2 前端：2-3 周

### 14.3 技术亮点

✅ **性能优秀**: Go高并发 + Redis缓存
✅ **安全可靠**: 成熟的认证授权方案
✅ **易于扩展**: 无状态设计 + Docker化
✅ **开发高效**: 完善的工具链和模板
✅ **未来可期**: 清晰的技术演进路线

---

**文档版本**: v1.2
**最后更新**: 2025-10-12
**维护者**: 技术团队
