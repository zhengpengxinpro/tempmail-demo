# 📧 TempMail 临时邮箱系统 - 项目结构详解

> 版本：v0.8.2-beta  
> 数据库：PostgreSQL 15 + Redis 7  
> 语言：Go 1.25  
> 架构：微服务架构 + RESTful API + WebSocket + SMTP

---

## 📑 目录
- [项目概述](#项目概述)
- [技术栈](#技术栈)
- [目录结构](#目录结构)
- [核心模块详解](#核心模块详解)
- [数据流向](#数据流向)
- [部署架构](#部署架构)

---

## 项目概述

临时邮箱系统后端服务，提供：
- ✅ 临时邮箱创建和管理
- ✅ SMTP 邮件接收服务
- ✅ RESTful API 接口
- ✅ WebSocket 实时通知
- ✅ 用户认证和权限管理
- ✅ Webhook 集成
- ✅ 邮件搜索和标签
- ✅ 多域名支持

**流量承载**：可承受 10,000+ QPS（当前部署：300K/月 = 0.12 QPS）

---

## 技术栈

### 核心框架
- **Web 框架**：Gin (HTTP 路由)
- **SMTP 服务器**：go-smtp
- **数据库 ORM**：GORM
- **WebSocket**：gorilla/websocket
- **日志**：Zap
- **配置管理**：Viper

### 数据存储
- **主数据库**：PostgreSQL 15 (支持 MySQL)
- **缓存**：Redis 7
- **文件存储**：本地文件系统 (可扩展到 S3/OSS)

### 监控和安全
- **监控**：Prometheus + Grafana
- **限流**：令牌桶算法
- **认证**：JWT
- **安全**：CORS、XSS 防护、SQL 注入防护

---

## 目录结构

```
go/
├── 📝 根目录配置文件
│   ├── go.mod                      # Go 模块依赖声明
│   ├── go.sum                      # 依赖版本锁定（确保构建一致性）
│   ├── .air.toml                   # Air 热重载配置（开发环境自动重启）
│   ├── .env.database               # 数据库环境变量示例
│   ├── .env.production.example     # 生产环境变量模板（需复制为 .env.production）
│   ├── .gitignore                  # Git 忽略文件规则
│   ├── Makefile                    # 构建脚本（make build/test/run/docker）
│   └── README.md                   # 项目说明文档
│
├── 🐳 Docker 部署配置
│   ├── Dockerfile                  # 多阶段构建镜像（编译 + 运行）
│   ├── docker-compose.yml          # 服务编排（PostgreSQL + Redis + App）
│   ├── deploy.sh                   # 服务器手动部署脚本
│   └── setup-server.sh             # 服务器初始化脚本（安装 Docker、克隆代码）
│
├── 🤖 CI/CD 自动化
│   └── .github/workflows/
│       └── deploy.yml              # GitHub Actions 自动部署工作流
│
├── 🚀 cmd/ - 应用程序入口
│   ├── server/
│   │   └── main.go                 # 🌟 主服务入口（HTTP + SMTP 双服务器）
│   ├── api/
│   │   └── main.go                 # 纯 HTTP API 服务入口（不启动 SMTP）
│   ├── migrate/
│   │   └── main.go                 # 数据库迁移工具（执行 SQL 迁移脚本）
│   └── create-admin/
│       └── main.go                 # 创建管理员用户工具
│
├── 🗄️ migrations/ - 数据库迁移脚本
│   ├── mysql/
│   │   ├── 001_initial_schema.up.sql       # 创建基础表结构
│   │   ├── 001_initial_schema.down.sql     # 回滚脚本
│   │   ├── 002_fix_charset.up.sql          # 修复字符集问题
│   │   └── 003_move_to_filesystem.up.sql   # 邮件内容迁移到文件系统
│   └── postgres/
│       ├── 001_initial_schema.up.sql       # 🌟 PostgreSQL 初始化（当前使用）
│       ├── 002_add_tags_tables.up.sql      # 添加标签功能表
│       └── 003_move_to_filesystem.up.sql   # 邮件内容迁移到文件系统
│
└── 🔧 internal/ - 核心业务代码（私有包）
    │
    ├── 🔐 auth/ - 认证授权模块
    │   ├── service.go              # 用户认证服务（注册、登录、Token 刷新）
    │   ├── password.go             # 密码加密（bcrypt）和验证
    │   └── jwt/
    │       ├── manager.go          # JWT Token 生成和验证
    │       └── claims.go           # JWT 声明结构（用户 ID、角色、过期时间）
    │
    ├── ⚙️ config/ - 配置管理
    │   └── config.go               # 🌟 环境变量加载（Viper）、配置结构定义
    │
    ├── 📦 domain/ - 领域模型（数据结构）
    │   ├── user.go                 # 用户模型（ID、邮箱、密码、角色、等级）
    │   ├── mailbox.go              # 邮箱模型（地址、Token、过期时间、IP）
    │   ├── message.go              # 邮件模型（发件人、主题、内容、附件）
    │   ├── alias.go                # 邮箱别名模型
    │   ├── webhook.go              # Webhook 配置和投递记录
    │   ├── tag.go                  # 邮件标签模型
    │   ├── domain.go               # 域名模型（用户域名、系统域名）
    │   └── apikey.go               # API Key 模型
    │
    ├── 💾 storage/ - 数据存储层（仓储模式）
    │   ├── store.go                # 🌟 存储接口定义（所有数据操作的契约）
    │   │
    │   ├── memory/                 # 内存存储实现
    │   │   ├── store.go            # 纯内存存储（开发/测试用，重启丢数据）
    │   │   ├── mailbox.go          # 邮箱内存存储
    │   │   ├── message.go          # 邮件内存存储
    │   │   └── user.go             # 用户内存存储
    │   │
    │   ├── postgres/               # PostgreSQL 实现
    │   │   ├── store.go            # 🌟 PostgreSQL 主存储（GORM ORM）
    │   │   ├── client.go           # 数据库连接池管理
    │   │   ├── mailbox.go          # 邮箱表操作
    │   │   ├── message.go          # 邮件表操作
    │   │   ├── user.go             # 用户表操作
    │   │   ├── webhook.go          # Webhook 表操作
    │   │   └── search_webhook.go   # 全文搜索实现
    │   │
    │   ├── redis/                  # Redis 缓存实现
    │   │   ├── cache.go            # 🌟 Redis 缓存（邮箱、邮件列表）
    │   │   └── pubsub.go           # Redis 发布订阅（实时通知）
    │   │
    │   ├── filesystem/             # 文件系统存储
    │   │   ├── store.go            # 邮件内容和附件存储到文件
    │   │   └── path.go             # 文件路径生成和管理
    │   │
    │   ├── hybrid/                 # 混合存储实现
    │   │   ├── store.go            # 🌟 组合 PostgreSQL + Redis（当前使用）
    │   │   ├── mailbox.go          # 邮箱：PostgreSQL 持久化 + Redis 缓存
    │   │   ├── message.go          # 邮件：PostgreSQL 存储 + Redis 缓存列表
    │   │   └── user.go             # 用户：PostgreSQL 存储 + Redis 缓存
    │   │
    │   └── sql/                    # 通用 SQL 实现
    │       └── store.go            # 通用 SQL 存储（支持 MySQL/PostgreSQL）
    │
    ├── 🎯 service/ - 业务逻辑层
    │   ├── mailbox.go              # 🌟 邮箱管理（创建、删除、查询、验证）
    │   ├── message.go              # 🌟 邮件管理（接收、读取、删除、搜索）
    │   ├── alias.go                # 别名管理（创建邮箱别名）
    │   ├── search.go               # 邮件搜索服务（全文搜索、过滤）
    │   ├── webhook.go              # Webhook 管理和异步投递
    │   ├── tag.go                  # 标签管理（创建、分配、查询）
    │   ├── admin.go                # 管理员功能（用户管理、统计）
    │   ├── user_domain.go          # 用户自定义域名管理
    │   ├── system_domain.go        # 系统域名管理（域名验证、MX 记录）
    │   └── apikey.go               # API Key 管理（生成、验证、撤销）
    │
    ├── 🌐 transport/http/ - HTTP 传输层
    │   ├── router.go               # 🌟 路由配置（所有 API 端点定义）
    │   ├── response.go             # 统一 JSON 响应格式
    │   │
    │   ├── handler_mailbox.go      # 邮箱 API 处理器
    │   │   # - POST   /v1/mailboxes          创建邮箱
    │   │   # - GET    /v1/mailboxes/:id      获取邮箱详情
    │   │   # - DELETE /v1/mailboxes/:id      删除邮箱
    │   │
    │   ├── handler_message.go      # 邮件 API 处理器
    │   │   # - GET    /v1/mailboxes/:id/messages           获取邮件列表
    │   │   # - GET    /v1/mailboxes/:id/messages/:msgId    获取邮件详情
    │   │   # - DELETE /v1/mailboxes/:id/messages/:msgId    删除邮件
    │   │   # - POST   /v1/mailboxes/:id/messages/:msgId/read  标记已读
    │   │
    │   ├── handler_auth.go         # 认证 API 处理器
    │   │   # - POST /v1/auth/register    用户注册
    │   │   # - POST /v1/auth/login       用户登录
    │   │   # - POST /v1/auth/refresh     刷新 Token
    │   │   # - GET  /v1/auth/me          获取当前用户信息
    │   │
    │   ├── handler_admin.go        # 管理员 API 处理器
    │   │   # - GET    /v1/admin/users            用户列表
    │   │   # - DELETE /v1/admin/users/:id        删除用户
    │   │   # - GET    /v1/admin/stats            系统统计
    │   │   # - GET    /v1/admin/domains          域名管理
    │   │
    │   ├── handler_webhook.go      # Webhook API 处理器
    │   ├── handler_tag.go          # 标签 API 处理器
    │   ├── handler_apikey.go       # API Key 处理器
    │   └── handler_public.go       # 公开 API（无需认证）
    │
    ├── 📧 smtp/ - SMTP 服务器
    │   ├── backend.go              # 🌟 SMTP 后端实现（邮件接收）
    │   └── session.go              # SMTP 会话处理（MAIL FROM、RCPT TO、DATA）
    │
    ├── 🔌 websocket/ - WebSocket 实时通知
    │   ├── hub.go                  # 🌟 WebSocket Hub（连接管理、消息广播）
    │   └── client.go               # WebSocket 客户端连接封装
    │
    ├── 🛡️ middleware/ - HTTP 中间件
    │   ├── auth.go                 # JWT 认证中间件（验证 Token）
    │   ├── mailbox_auth.go         # 邮箱 Token 认证（X-Mailbox-Token 头）
    │   ├── admin_auth.go           # 管理员权限检查
    │   ├── apikey_auth.go          # API Key 认证
    │   ├── rate_limit.go           # 🌟 限流中间件（IP/用户/邮箱级别）
    │   ├── abuse_prevention.go     # 防滥用中间件（检测异常行为）
    │   ├── cors.go                 # CORS 跨域处理
    │   ├── logger.go               # 请求日志记录
    │   ├── recovery.go             # Panic 恢复中间件
    │   ├── security.go             # 安全头部（X-Frame-Options、CSP）
    │   └── monitoring.go           # Prometheus 指标采集
    │
    ├── 📊 monitoring/ - 监控和告警
    │   ├── metrics.go              # 🌟 Prometheus 指标定义和收集
    │   │   # - HTTP 请求统计
    │   │   # - 邮箱创建/删除数量
    │   │   # - 邮件接收/读取数量
    │   │   # - 数据库连接数
    │   │   # - 内存/CPU 使用率
    │   ├── alert.go                # 告警规则（高错误率、高内存使用）
    │   └── health.go               # 健康检查实现
    │
    ├── 📝 logger/ - 日志系统
    │   └── logger.go               # Zap 日志封装（结构化日志、日志轮转）
    │
    ├── 🏥 health/ - 健康检查
    │   └── health.go               # 健康检查端点（/health、/health/live、/health/ready）
    │
    ├── 🔒 security/ - 安全模块
    │   ├── xss.go                  # XSS 攻击防护（HTML 转义）
    │   ├── sql_injection.go        # SQL 注入防护（参数化查询）
    │   └── rate_limit.go           # 限流算法（令牌桶、滑动窗口）
    │
    ├── 💾 cache/ - 缓存层
    │   └── cache.go                # 缓存接口和实现
    │
    └── 🧰 pool/ - 协程池
        └── worker_pool.go          # Worker Pool（并发任务处理）
```

---

## 核心模块详解

### 1️⃣ cmd/server/main.go - 应用启动入口

**职责**：
- 加载配置（环境变量）
- 初始化数据库连接（PostgreSQL + Redis）
- 初始化各个服务层
- 启动 HTTP 服务器（端口 8080）
- 启动 SMTP 服务器（端口 25）
- 启动 WebSocket Hub
- 启动定时任务（清理过期邮箱、重试 Webhook）
- 优雅关闭处理

**关键代码片段**：
```go
// 初始化数据库存储
store := hybrid.NewStore(postgresURL, redisURL)

// 初始化服务
mailboxService := service.NewMailboxService(store)
messageService := service.NewMessageService(store)

// 创建 HTTP 路由
router := http.NewRouter(...)

// 启动 HTTP 服务器（8080）
httpServer.ListenAndServe()

// 启动 SMTP 服务器（25）
smtpServer.ListenAndServe()
```

---

### 2️⃣ internal/storage/ - 数据存储层

**设计模式**：仓储模式（Repository Pattern）

**接口定义** (`store.go`)：
```go
type Store interface {
    // 邮箱操作
    SaveMailbox(*Mailbox) error
    GetMailbox(id string) (*Mailbox, error)
    DeleteMailbox(id string) error
    
    // 邮件操作
    SaveMessage(*Message) error
    ListMessages(mailboxID string) ([]*Message, error)
    GetMessage(mailboxID, messageID string) (*Message, error)
    
    // 用户操作
    CreateUser(*User) error
    GetUserByEmail(email string) (*User, error)
}
```

**实现层次**：
- **memory/** - 开发/测试用，数据在内存，重启丢失
- **postgres/** - PostgreSQL 持久化存储
- **redis/** - Redis 缓存层
- **hybrid/** - **当前使用**，组合 PostgreSQL（持久化）+ Redis（缓存）
- **filesystem/** - 邮件内容和附件存储到文件系统

**Hybrid 存储策略**：
```
写入流程：
1. 写入 PostgreSQL（持久化）
2. 写入 Redis（缓存）

读取流程：
1. 先查 Redis 缓存
2. 缓存命中 → 直接返回
3. 缓存未命中 → 查 PostgreSQL → 写入 Redis → 返回
```

---

### 3️⃣ internal/service/ - 业务逻辑层

**职责**：封装业务规则，不直接操作数据库

#### **mailbox.go - 邮箱管理服务**

**核心功能**：
```go
// 创建邮箱
func (s *MailboxService) CreateMailbox(req CreateMailboxRequest) (*Mailbox, error) {
    // 1. 验证域名是否允许
    // 2. 检查 IP 限制（每个 IP 最多创建 N 个邮箱）
    // 3. 生成邮箱地址和 Token
    // 4. 设置过期时间
    // 5. 保存到数据库
    // 6. 返回邮箱信息
}

// 删除邮箱
func (s *MailboxService) DeleteMailbox(id, token string) error {
    // 1. 验证 Token
    // 2. 删除所有邮件
    // 3. 删除邮箱
}
```

#### **message.go - 邮件管理服务**

**核心功能**：
```go
// SMTP 接收邮件
func (s *MessageService) ReceiveMessage(from, to string, data []byte) error {
    // 1. 解析邮件（主题、正文、附件）
    // 2. 查找目标邮箱
    // 3. 保存邮件到数据库
    // 4. 保存附件到文件系统
    // 5. 更新邮箱未读数
    // 6. 触发 WebSocket 通知
    // 7. 触发 Webhook
}

// 获取邮件列表
func (s *MessageService) ListMessages(mailboxID string) ([]*Message, error) {
    // 1. 验证邮箱权限
    // 2. 从数据库查询（分页）
    // 3. 返回邮件列表
}
```

---

### 4️⃣ internal/transport/http/ - HTTP API 层

**router.go - 路由定义**：

```go
func NewRouter(deps RouterDependencies) *gin.Engine {
    router := gin.New()
    
    // 中间件
    router.Use(middleware.CORS())
    router.Use(middleware.Logger())
    router.Use(middleware.Recovery())
    
    // API v1
    v1 := router.Group("/v1")
    {
        // 公开 API（无需认证）
        v1.GET("/public/domains", handler.GetDomains)
        
        // 邮箱 API
        mailboxes := v1.Group("/mailboxes")
        mailboxes.POST("", handler.CreateMailbox)                    // 创建邮箱
        mailboxes.GET("/:id", mailboxAuth, handler.GetMailbox)       // 需要邮箱 Token
        mailboxes.GET("/:id/messages", mailboxAuth, handler.ListMessages)
        
        // 认证 API
        auth := v1.Group("/auth")
        auth.POST("/register", handler.Register)
        auth.POST("/login", handler.Login)
        auth.POST("/refresh", handler.Refresh)
        auth.GET("/me", jwtAuth, handler.Me)                         // 需要 JWT
        
        // 管理员 API
        admin := v1.Group("/admin")
        admin.Use(jwtAuth, adminAuth)                                // 需要 JWT + 管理员权限
        admin.GET("/users", handler.ListUsers)
        admin.GET("/stats", handler.GetStats)
    }
    
    // 健康检查
    router.GET("/health", handler.Health)
    
    // Prometheus 指标
    router.GET("/metrics", promhttp.Handler())
    
    return router
}
```

---

### 5️⃣ internal/smtp/ - SMTP 服务器

**backend.go - SMTP 邮件接收**：

```go
// 实现 go-smtp 的 Backend 接口
type Backend struct {
    mailboxService *service.MailboxService
    messageService *service.MessageService
}

// 接收邮件流程
func (b *Backend) NewSession(conn *smtp.Conn) (smtp.Session, error) {
    return &Session{backend: b}, nil
}

type Session struct {
    backend *Backend
    from    string
    to      []string
}

// MAIL FROM 命令
func (s *Session) Mail(from string) error {
    s.from = from
    return nil
}

// RCPT TO 命令
func (s *Session) Rcpt(to string) error {
    // 检查收件人邮箱是否存在
    s.to = append(s.to, to)
    return nil
}

// DATA 命令（接收邮件内容）
func (s *Session) Data(r io.Reader) error {
    // 1. 读取邮件数据
    data, _ := io.ReadAll(r)
    
    // 2. 调用 messageService 保存邮件
    for _, to := range s.to {
        s.backend.messageService.ReceiveMessage(s.from, to, data)
    }
    
    return nil
}
```

---

### 6️⃣ internal/websocket/ - WebSocket 实时通知

**hub.go - 连接管理和消息广播**：

```go
type Hub struct {
    clients    map[string]*Client            // 所有连接的客户端
    mailboxes  map[string]map[string]*Client // 邮箱订阅关系
    broadcast  chan *BroadcastMessage        // 广播消息队列
    register   chan *Client                  // 注册新客户端
    unregister chan *Client                  // 注销客户端
}

// 运行 Hub（主循环）
func (h *Hub) Run(ctx context.Context) {
    for {
        select {
        case client := <-h.register:
            // 注册新客户端
            h.clients[client.ID] = client
            
        case client := <-h.unregister:
            // 注销客户端
            delete(h.clients, client.ID)
            close(client.send)
            
        case message := <-h.broadcast:
            // 广播消息到订阅该邮箱的所有客户端
            for _, client := range h.mailboxes[message.MailboxID] {
                select {
                case client.send <- message.Data:
                default:
                    // 客户端阻塞，关闭连接
                    close(client.send)
                }
            }
        }
    }
}

// 广播新邮件通知
func (h *Hub) BroadcastNewMail(mailboxID string, message *Message) {
    h.broadcast <- &BroadcastMessage{
        Type:      "new_mail",
        MailboxID: mailboxID,
        Data:      marshalJSON(message),
    }
}
```

**使用场景**：
```
1. 前端连接 WebSocket: ws://154.40.43.194:8080/ws
2. 发送订阅消息: {"action":"subscribe", "mailbox_id":"xxx", "token":"xxx"}
3. 当邮箱收到新邮件 → Hub 自动推送通知
4. 前端实时显示新邮件
```

---

### 7️⃣ internal/middleware/ - 中间件

#### **rate_limit.go - 限流中间件**

**实现**：令牌桶算法
```go
func RateLimitByIP(store RateLimitStore, logger *zap.Logger, limit int, window time.Duration) gin.HandlerFunc {
    return func(c *gin.Context) {
        ip := c.ClientIP()
        
        // 检查限流
        allowed, err := store.AllowRequest(ip, limit, window)
        if !allowed {
            c.JSON(429, gin.H{"error": "Too many requests"})
            c.Abort()
            return
        }
        
        c.Next()
    }
}
```

**限流策略**：
- **IP 级别**：每分钟 100 请求
- **用户级别**：每分钟 200 请求
- **邮箱创建**：每小时 50 个
- **邮件接收**：每小时 1000 封

---

### 8️⃣ migrations/ - 数据库迁移

**PostgreSQL 表结构** (`001_initial_schema.up.sql`)：

```sql
-- 用户表
CREATE TABLE users (
    id VARCHAR(36) PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    username VARCHAR(100),
    password_hash VARCHAR(255),
    role VARCHAR(20) DEFAULT 'user',      -- user/admin/super
    tier VARCHAR(20) DEFAULT 'free',      -- free/basic/pro/enterprise
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 邮箱表
CREATE TABLE mailboxes (
    id VARCHAR(36) PRIMARY KEY,
    address VARCHAR(255) UNIQUE NOT NULL,
    local_part VARCHAR(100) NOT NULL,
    domain VARCHAR(100) NOT NULL,
    token VARCHAR(255) NOT NULL,
    user_id VARCHAR(36),                  -- 关联用户（可选）
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP,                 -- 过期时间
    ip_source VARCHAR(45),                -- 创建者 IP
    total_count INTEGER DEFAULT 0,        -- 邮件总数
    unread INTEGER DEFAULT 0,             -- 未读数
    FOREIGN KEY (user_id) REFERENCES users(id)
);

-- 邮件表
CREATE TABLE messages (
    id VARCHAR(36) PRIMARY KEY,
    mailbox_id VARCHAR(36) NOT NULL,
    from_address VARCHAR(255) NOT NULL,
    to_address VARCHAR(255) NOT NULL,
    subject TEXT,
    -- 内容字段已迁移到文件系统
    is_read BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (mailbox_id) REFERENCES mailboxes(id) ON DELETE CASCADE
);

-- 附件表
CREATE TABLE attachments (
    id VARCHAR(36) PRIMARY KEY,
    message_id VARCHAR(36) NOT NULL,
    mailbox_id VARCHAR(36) NOT NULL,
    filename VARCHAR(255) NOT NULL,
    content_type VARCHAR(100),
    size INTEGER,
    storage_path VARCHAR(500),            -- 文件存储路径
    FOREIGN KEY (message_id) REFERENCES messages(id) ON DELETE CASCADE
);

-- 邮箱别名表
CREATE TABLE mailbox_aliases (
    id VARCHAR(36) PRIMARY KEY,
    mailbox_id VARCHAR(36) NOT NULL,
    address VARCHAR(255) UNIQUE NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,
    FOREIGN KEY (mailbox_id) REFERENCES mailboxes(id) ON DELETE CASCADE
);

-- 系统域名表
CREATE TABLE system_domains (
    id VARCHAR(36) PRIMARY KEY,
    domain VARCHAR(100) UNIQUE NOT NULL,
    status VARCHAR(20) DEFAULT 'pending', -- pending/verified
    is_active BOOLEAN DEFAULT FALSE,
    is_default BOOLEAN DEFAULT FALSE,
    mailbox_count INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Webhook 配置表
CREATE TABLE webhooks (
    id VARCHAR(36) PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL,
    url VARCHAR(500) NOT NULL,
    events JSONB,                         -- 订阅的事件类型
    is_active BOOLEAN DEFAULT TRUE,
    FOREIGN KEY (user_id) REFERENCES users(id)
);

-- 标签表
CREATE TABLE tags (
    id VARCHAR(36) PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL,
    name VARCHAR(50) NOT NULL,
    color VARCHAR(20),
    FOREIGN KEY (user_id) REFERENCES users(id),
    UNIQUE(user_id, name)
);

-- 邮件标签关联表
CREATE TABLE message_tags (
    message_id VARCHAR(36) NOT NULL,
    tag_id VARCHAR(36) NOT NULL,
    PRIMARY KEY (message_id, tag_id),
    FOREIGN KEY (message_id) REFERENCES messages(id) ON DELETE CASCADE,
    FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE
);
```

---

## 数据流向

### 📨 接收邮件流程

```
外部 SMTP 发送邮件
        ↓
SMTP 服务器 (smtp/backend.go)
    - 端口 25 监听
    - 解析 MAIL FROM、RCPT TO、DATA
        ↓
邮件服务 (service/message.go)
    - 解析邮件内容（主题、正文、附件）
    - 验证收件人邮箱是否存在
        ↓
存储层 (storage/hybrid/)
    - 保存邮件到 PostgreSQL
    - 保存附件到文件系统
    - 更新邮箱未读数
    - 写入 Redis 缓存
        ↓
通知系统（并发执行）
    ├─→ WebSocket Hub 广播新邮件通知
    └─→ Webhook 异步投递
```

### 🌐 HTTP API 请求流程

```
客户端请求
    ↓
HTTP 服务器 (Gin)
    - 端口 8080 监听
        ↓
中间件链 (middleware/)
    - CORS 处理
    - 请求日志
    - 认证检查 (JWT/邮箱 Token)
    - 限流检查
    - 指标采集
        ↓
路由分发 (transport/http/router.go)
        ↓
Handler 处理器 (handler_*.go)
    - 参数验证
    - 调用 Service 层
        ↓
业务逻辑 (service/)
    - 业务规则检查
    - 调用 Storage 层
        ↓
存储层 (storage/)
    - 优先查询 Redis 缓存
    - 缓存未命中 → 查询 PostgreSQL
    - 写入操作同时更新缓存
        ↓
返回响应
    - 统一 JSON 格式
    - 错误处理
```

### 🔌 WebSocket 实时通知流程

```
前端建立 WebSocket 连接
    ↓
WebSocket Hub (websocket/hub.go)
    - 验证 Token
    - 注册客户端
    - 订阅邮箱
        ↓
等待新邮件...
        ↓
SMTP 收到新邮件
    ↓
Message Service 保存邮件
    ↓
调用 Hub.BroadcastNewMail()
    ↓
Hub 查找订阅该邮箱的所有客户端
    ↓
推送消息到客户端
    ↓
前端实时显示新邮件
```

---

## 部署架构

### 当前生产环境

```
服务器: 154.40.43.194
    │
    ├── Docker Compose
    │   │
    │   ├── tempmail-postgres (PostgreSQL 15)
    │   │   - 端口: 5432
    │   │   - 数据卷: postgres_data
    │   │   - 内存: ~150-250MB
    │   │
    │   ├── tempmail-redis (Redis 7)
    │   │   - 端口: 6379
    │   │   - 数据卷: redis_data
    │   │   - 内存: ~50-100MB
    │   │
    │   └── tempmail-app (Go 应用)
    │       - HTTP 端口: 8080
    │       - SMTP 端口: 25
    │       - 数据卷: mail_storage (邮件文件)
    │       - 内存: ~50-150MB
    │
    └── 自动部署
        - GitHub Actions
        - SSH 自动连接
        - 自动构建、重启
```

### 服务依赖关系

```
tempmail-app
    ├── depends_on: tempmail-postgres (健康检查)
    └── depends_on: tempmail-redis (健康检查)
```

### 数据持久化

```
Docker Volumes:
    ├── postgres_data       # PostgreSQL 数据文件
    ├── redis_data          # Redis 持久化文件
    └── mail_storage        # 邮件内容和附件文件
```

**重启不丢数据**：所有数据存储在 Docker Volume，容器删除后数据依然保留。

---

## 监控和健康检查

### Prometheus 指标端点

```
http://154.40.43.194:8080/metrics
```

**采集的指标**：
- `http_requests_total` - HTTP 请求总数
- `http_request_duration_seconds` - 请求耗时
- `mailboxes_created_total` - 邮箱创建数
- `messages_received_total` - 邮件接收数
- `database_connections` - 数据库连接数
- `memory_usage_bytes` - 内存使用量
- `cpu_usage_percent` - CPU 使用率

### 健康检查端点

```
http://154.40.43.194:8080/health           # 基础健康检查
http://154.40.43.194:8080/health/live      # Kubernetes LivenessProbe
http://154.40.43.194:8080/health/ready     # Kubernetes ReadinessProbe
```

---

## 安全特性

### 认证方式

1. **JWT Token**（用户认证）
   - Access Token: 24小时
   - Refresh Token: 7天
   - 算法: HS256

2. **邮箱 Token**（邮箱访问）
   - 32字符随机字符串
   - 请求头: `X-Mailbox-Token`

3. **API Key**（第三方集成）
   - 前缀标识
   - Scopes 权限控制

### 安全防护

- ✅ **CORS** - 跨域资源共享控制
- ✅ **限流** - IP/用户/邮箱级别限流
- ✅ **XSS 防护** - HTML 转义
- ✅ **SQL 注入防护** - 参数化查询（GORM）
- ✅ **密码加密** - bcrypt (cost=10)
- ✅ **HTTPS** - 支持 TLS（需配置）
- ✅ **速率限制** - 防止暴力破解

---

## 性能优化

### 缓存策略

```
邮箱信息:
    - Redis TTL: 1小时
    - 缓存键: mailbox:{id}

邮件列表:
    - Redis TTL: 5分钟
    - 缓存键: messages:{mailbox_id}
    - 分页缓存

用户信息:
    - Redis TTL: 30分钟
    - 缓存键: user:{id}
```

### 数据库优化

```sql
-- 关键索引
CREATE INDEX idx_mailboxes_address ON mailboxes(address);
CREATE INDEX idx_mailboxes_expires_at ON mailboxes(expires_at);
CREATE INDEX idx_messages_mailbox_id ON messages(mailbox_id);
CREATE INDEX idx_messages_created_at ON messages(created_at);
CREATE INDEX idx_users_email ON users(email);
```

### 并发优化

- **Worker Pool** - 限制并发 Goroutine 数量
- **连接池** - 数据库连接池（最大 25 连接）
- **异步处理** - Webhook 投递、WebSocket 通知

---

## 开发指南

### 本地运行

```bash
# 1. 安装依赖
go mod download

# 2. 配置环境变量（复制模板）
cp .env.database .env

# 3. 启动 PostgreSQL 和 Redis（Docker）
docker compose up -d postgres redis

# 4. 运行数据库迁移
go run ./cmd/migrate up

# 5. 启动服务
go run ./cmd/server

# 或使用 Air 热重载
air
```

### 测试

```bash
# 运行所有测试
go test ./...

# 运行特定模块测试
go test ./internal/service

# 生成覆盖率报告
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### 构建

```bash
# 本地构建
make build

# Docker 构建
make docker

# 或
docker compose build
```

---

## API 文档

### 公开 API（无需认证）

```
GET  /health                     健康检查
GET  /metrics                    Prometheus 指标
GET  /v1/public/domains          获取可用域名列表
```

### 邮箱 API

```
POST   /v1/mailboxes             创建邮箱
GET    /v1/mailboxes/:id         获取邮箱详情 (需要邮箱Token)
DELETE /v1/mailboxes/:id         删除邮箱 (需要邮箱Token)
```

### 邮件 API

```
GET    /v1/mailboxes/:id/messages              获取邮件列表
GET    /v1/mailboxes/:id/messages/:msgId       获取邮件详情
DELETE /v1/mailboxes/:id/messages/:msgId       删除邮件
POST   /v1/mailboxes/:id/messages/:msgId/read  标记已读
```

### 认证 API

```
POST /v1/auth/register           用户注册
POST /v1/auth/login              用户登录
POST /v1/auth/refresh            刷新Token
GET  /v1/auth/me                 获取当前用户信息
```

### 管理员 API

```
GET    /v1/admin/users           用户列表
DELETE /v1/admin/users/:id       删除用户
GET    /v1/admin/stats           系统统计
```

---

## 环境变量配置

```bash
# JWT 配置
TEMPMAIL_JWT_SECRET=your-secret-key-at-least-32-characters
TEMPMAIL_JWT_ISSUER=tempmail-production
TEMPMAIL_JWT_ACCESS_EXPIRY=24h
TEMPMAIL_JWT_REFRESH_EXPIRY=168h

# PostgreSQL 配置
TEMPMAIL_DATABASE_TYPE=postgres
TEMPMAIL_DATABASE_DSN=postgresql://user:pass@host:5432/dbname?sslmode=disable

# Redis 配置
TEMPMAIL_REDIS_ADDRESS=localhost:6379
TEMPMAIL_REDIS_PASSWORD=
TEMPMAIL_REDIS_DB=0

# 服务器配置
TEMPMAIL_SERVER_HOST=0.0.0.0
TEMPMAIL_SERVER_PORT=8080

# SMTP 配置
TEMPMAIL_SMTP_BIND_ADDR=:25
TEMPMAIL_SMTP_DOMAIN=temp.mail

# 邮箱配置
TEMPMAIL_MAILBOX_ALLOWED_DOMAINS=temp.mail,tempmail.dev
TEMPMAIL_MAILBOX_DEFAULT_TTL=24h
TEMPMAIL_MAILBOX_MAX_PER_IP=10

# 文件存储
TEMPMAIL_STORAGE_PATH=./data/mail-storage

# 日志配置
TEMPMAIL_LOG_LEVEL=info
TEMPMAIL_LOG_DEVELOPMENT=false
```

---

## 常见问题

### Q1: 如何添加新的 API 端点？

1. 在 `internal/transport/http/handler_*.go` 添加处理函数
2. 在 `router.go` 注册路由
3. 添加必要的中间件（认证、限流等）

### Q2: 如何修改数据库结构？

1. 创建新的迁移文件 `migrations/postgres/00X_description.up.sql`
2. 编写 SQL 语句
3. 创建回滚文件 `00X_description.down.sql`
4. 运行 `go run ./cmd/migrate up`

### Q3: 如何开启限流？

修改 `internal/transport/http/router.go`，取消注释限流中间件：
```go
// 开启 IP 限流
ipRateLimit := middleware.RateLimitByIP(rateLimitStore, logger, 100, 1*time.Minute)
v1.Use(ipRateLimit)
```

### Q4: 如何切换到 MySQL？

1. 修改 `docker-compose.yml`，替换 postgres 为 mysql
2. 修改 `.env.production`：
   ```
   TEMPMAIL_DATABASE_TYPE=mysql
   TEMPMAIL_DATABASE_DSN=user:pass@tcp(host:3306)/dbname?parseTime=true
   ```
3. 使用 MySQL 迁移脚本：`migrations/mysql/`

---

## 版本历史

### v0.8.2-beta (当前)
- ✅ PostgreSQL + Redis 混合存储
- ✅ WebSocket 实时通知
- ✅ Webhook 集成
- ✅ 邮件标签功能
- ✅ 多域名支持
- ✅ 文件系统存储（邮件内容/附件）
- ✅ Prometheus 监控
- ✅ Docker 部署支持
- ✅ GitHub Actions 自动部署

---

## 相关链接

- **生产环境**: http://154.40.43.194:8080
- **健康检查**: http://154.40.43.194:8080/health
- **监控指标**: http://154.40.43.194:8080/metrics
- **GitHub**: https://github.com/zhengpengxinpro/tempmail-demo

---

**文档更新时间**: 2025-10-27  
**维护者**: zhengpengxinpro
