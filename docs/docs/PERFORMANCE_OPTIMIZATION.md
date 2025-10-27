# 性能优化实施指南

## 当前系统性能

### 基准性能（单机 4C8G）

| 指标 | 当前值 | 优化后目标 |
|------|--------|-----------|
| HTTP QPS | ~2,000 | 10,000+ |
| SMTP 并发 | ~200 | 1,000+ |
| WebSocket 连接 | ~5,000 | 50,000+ |
| API 响应时间 (P95) | ~200ms | < 100ms |
| 数据库连接 | 25 | 100-200 |

## 优化实施计划

### Phase 1: 立即优化（1-2天）

#### 1.1 数据库连接池调优

**文件**: `cmd/server/main.go`

```go
// 根据服务器规格调整
db.SetMaxOpenConns(100)              // 4C8G: 100, 8C16G: 200
db.SetMaxIdleConns(20)               // MaxOpen 的 20%
db.SetConnMaxLifetime(1 * time.Hour)
db.SetConnMaxIdleTime(10 * time.Minute)
```

#### 1.2 启用本地缓存

```go
// internal/service/mailbox.go
type MailboxService struct {
    store       domain.Store
    localCache  *cache.LocalCache // 新增
    // ...
}

func NewMailboxService(store domain.Store, cfg *config.Config) *MailboxService {
    return &MailboxService{
        store:      store,
        localCache: cache.NewLocalCache(10000, 5*time.Second), // 缓存1万条，5秒TTL
        // ...
    }
}

func (s *MailboxService) GetByAddress(address string) (*domain.Mailbox, error) {
    // 先查本地缓存
    if val, ok := s.localCache.Get("mb:" + address); ok {
        return val.(*domain.Mailbox), nil
    }
    
    // 查询数据库
    mailbox, err := s.store.GetMailboxByAddress(address)
    if err != nil {
        return nil, err
    }
    
    // 写入缓存
    s.localCache.Set("mb:" + address, mailbox, 10*time.Second)
    return mailbox, nil
}
```

#### 1.3 优化 SMTP 连接处理

```go
// cmd/server/main.go
smtpServer := gosmtp.NewServer(smtpBackend)
smtpServer.Addr = cfg.SMTP.BindAddr
smtpServer.Domain = cfg.SMTP.Domain
smtpServer.MaxConnections = 1000        // 新增：最大并发连接
smtpServer.MaxRecipients = 10
smtpServer.MaxMessageBytes = 10 * 1024 * 1024
smtpServer.ReadTimeout = 30 * time.Second
smtpServer.WriteTimeout = 30 * time.Second
```

### Phase 2: 中期优化（1周）

#### 2.1 实现读写分离

**配置文件**: `config/config.go`

```go
type DatabaseConfig struct {
    Type            string
    MasterDSN       string   // 主库（写）
    SlaveDSNs       []string // 从库（读）
    MaxOpenConns    int
    MaxIdleConns    int
    ConnMaxLifetime time.Duration
}
```

**环境变量**：

```bash
# 主库
TEMPMAIL_DATABASE_MASTER_DSN=mysql://user:pass@master:3306/tempmail

# 从库（多个用逗号分隔）
TEMPMAIL_DATABASE_SLAVE_DSNS=mysql://user:pass@slave1:3306/tempmail,mysql://user:pass@slave2:3306/tempmail
```

#### 2.2 消息队列集成

**安装依赖**：

```bash
go get github.com/rabbitmq/amqp091-go@latest
```

**邮件异步处理**：

```go
// internal/queue/mail_queue.go
type MailQueue struct {
    conn    *amqp.Connection
    channel *amqp.Channel
}

// SMTP 接收后快速入队
func (b *Backend) Data(r io.Reader) error {
    rawBytes, _ := io.ReadAll(r)
    
    // 快速入队，立即返回
    err := mailQueue.Enqueue(&RawMail{
        From:       s.fromAddress,
        Recipients: s.recipients,
        Data:       rawBytes,
    })
    
    return err // 200 OK
}

// 后台 Workers 异步处理
func StartMailWorkers(n int, store domain.Store) {
    for i := 0; i < n; i++ {
        go func(workerID int) {
            for mail := range mailQueue.Consume() {
                // 解析邮件
                parsed, _ := ParseEmail(mail.Data)
                
                // 存储到数据库
                for _, rcpt := range mail.Recipients {
                    store.SaveMessage(&domain.Message{...})
                }
                
                // 发送 WebSocket 通知
                wsHub.NotifyNewMail(...)
            }
        }(i)
    }
}
```

#### 2.3 Redis 集群部署

**Docker Compose**：

```yaml
# docker-compose-redis-cluster.yml
version: '3.8'

services:
  redis-node-1:
    image: redis:7-alpine
    command: redis-server --cluster-enabled yes --appendonly yes
    ports:
      - "7000:6379"
    volumes:
      - redis1:/data

  redis-node-2:
    image: redis:7-alpine
    command: redis-server --cluster-enabled yes --appendonly yes
    ports:
      - "7001:6379"
    volumes:
      - redis2:/data

  redis-node-3:
    image: redis:7-alpine
    command: redis-server --cluster-enabled yes --appendonly yes
    ports:
      - "7002:6379"
    volumes:
      - redis3:/data

  redis-node-4:
    image: redis:7-alpine
    command: redis-server --cluster-enabled yes --appendonly yes
    ports:
      - "7003:6379"
    volumes:
      - redis4:/data

  redis-node-5:
    image: redis:7-alpine
    command: redis-server --cluster-enabled yes --appendonly yes
    ports:
      - "7004:6379"
    volumes:
      - redis5:/data

  redis-node-6:
    image: redis:7-alpine
    command: redis-server --cluster-enabled yes --appendonly yes
    ports:
      - "7005:6379"
    volumes:
      - redis6:/data

volumes:
  redis1:
  redis2:
  redis3:
  redis4:
  redis5:
  redis6:
```

**创建集群**：

```bash
docker-compose -f docker-compose-redis-cluster.yml up -d

# 创建集群（3主3从）
docker exec -it redis-node-1 redis-cli --cluster create \
  127.0.0.1:7000 127.0.0.1:7001 127.0.0.1:7002 \
  127.0.0.1:7003 127.0.0.1:7004 127.0.0.1:7005 \
  --cluster-replicas 1
```

### Phase 3: 长期优化（1个月）

#### 3.1 分库分表

**按业务垂直拆分**：

```
tempmail_user_db    - 用户、认证、权限
tempmail_mailbox_db - 邮箱、域名
tempmail_message_db - 邮件、附件
```

**按时间水平拆分**：

```sql
-- 按月分表
CREATE TABLE messages_202401 LIKE messages;
CREATE TABLE messages_202402 LIKE messages;
...

-- 自动路由
func getMessageTable(createdAt time.Time) string {
    return fmt.Sprintf("messages_%s", createdAt.Format("200601"))
}
```

#### 3.2 CDN 加速

```nginx
# 静态资源 CDN
location ~* \.(jpg|jpeg|png|gif|ico|css|js)$ {
    expires 30d;
    add_header Cache-Control "public, immutable";
}

# API 缓存
location /v1/mailboxes {
    proxy_cache api_cache;
    proxy_cache_valid 200 10s;
    proxy_cache_key $request_uri;
}
```

#### 3.3 服务拆分

**微服务架构**：

```
┌──────────────┐  ┌──────────────┐  ┌──────────────┐
│  Auth Service│  │ Mailbox Svc  │  │ Message Svc  │
│  端口: 8081  │  │ 端口: 8082   │  │ 端口: 8083   │
└──────┬───────┘  └──────┬───────┘  └──────┬───────┘
       │                 │                 │
       └─────────────────┼─────────────────┘
                         ↓
                  ┌──────────────┐
                  │  API Gateway │
                  │  端口: 8080  │
                  └──────────────┘
```

## 配置建议

### 小型部署（< 1万用户）

```yaml
配置: 1台服务器 (4C8G)
数据库: MySQL 单机 (2C4G)
Redis: 单机 (2C2G)
```

**配置参数**：

```bash
TEMPMAIL_DATABASE_MAX_OPEN_CONNS=50
TEMPMAIL_DATABASE_MAX_IDLE_CONNS=10
```

### 中型部署（1-10万用户）

```yaml
API 服务器: 3台 (4C8G)
SMTP 服务器: 2台 (2C4G)
数据库: MySQL 1主2从 (4C8G)
Redis: 集群 3主3从 (2C4G)
负载均衡: Nginx (2C4G)
```

**配置参数**：

```bash
TEMPMAIL_DATABASE_MAX_OPEN_CONNS=100
TEMPMAIL_DATABASE_MAX_IDLE_CONNS=20
```

### 大型部署（10万+ 用户）

```yaml
API 服务器: 10台 (8C16G) + 自动扩缩容
SMTP 服务器: 5台 (4C8G)
数据库: MySQL 1主4从 (8C16G) + 分库分表
Redis: 集群 6主6从 (4C8G)
消息队列: RabbitMQ 集群 (4C8G)
负载均衡: 阿里云 SLB
CDN: CloudFlare
```

**配置参数**：

```bash
TEMPMAIL_DATABASE_MAX_OPEN_CONNS=200
TEMPMAIL_DATABASE_MAX_IDLE_CONNS=50
```

## 性能测试

### 测试工具

```bash
# 安装测试工具
go install github.com/rakyll/hey@latest
go install github.com/tsenart/vegeta@latest

# API 压测
hey -z 30s -c 100 http://localhost:8080/v1/mailboxes

# 生成压测报告
echo "GET http://localhost:8080/v1/mailboxes" | \
  vegeta attack -duration=30s -rate=1000 | \
  vegeta report -type=text
```

### 监控指标

**Grafana Dashboard 配置**：

```json
{
  "dashboard": {
    "title": "TempMail Performance",
    "panels": [
      {
        "title": "HTTP Requests/s",
        "targets": [
          {
            "expr": "rate(http_requests_total[1m])"
          }
        ]
      },
      {
        "title": "API Response Time (P95)",
        "targets": [
          {
            "expr": "histogram_quantile(0.95, http_request_duration_seconds)"
          }
        ]
      },
      {
        "title": "SMTP Active Connections",
        "targets": [
          {
            "expr": "smtp_connections_active"
          }
        ]
      }
    ]
  }
}
```

## 快速优化 Checklist

### 立即可做（0成本）

- [ ] 启用 Gin Release 模式
- [ ] 优化数据库连接池参数
- [ ] 添加本地缓存（已提供代码）
- [ ] 启用 HTTP Keep-Alive
- [ ] 开启 GZIP 压缩
- [ ] 优化日志级别（生产环境用 info）

### 短期优化（低成本）

- [ ] 部署 Redis 单机缓存
- [ ] 配置 Nginx 反向代理
- [ ] 添加 CDN（静态资源）
- [ ] 数据库索引优化
- [ ] 启用 HTTP/2

### 中期优化（中等成本）

- [ ] API 服务器水平扩展（3-5台）
- [ ] 数据库主从复制（1主2从）
- [ ] Redis 集群部署
- [ ] 消息队列集成（RabbitMQ）
- [ ] 实施监控告警（Prometheus + Grafana）

### 长期优化（高成本）

- [ ] 微服务拆分
- [ ] 分库分表
- [ ] 跨地域部署
- [ ] 多活架构
- [ ] 智能 DNS 调度

## Go 代码优化

### 1. 减少内存分配

```go
// 错误示例
func processEmails() {
    for _, mail := range mails {
        result := fmt.Sprintf("%s: %s", mail.From, mail.Subject) // 每次分配
    }
}

// 优化示例
func processEmails() {
    var buf strings.Builder
    for _, mail := range mails {
        buf.Reset()
        buf.WriteString(mail.From)
        buf.WriteString(": ")
        buf.WriteString(mail.Subject)
        result := buf.String()
    }
}
```

### 2. 对象池

```go
var messagePool = sync.Pool{
    New: func() interface{} {
        return &domain.Message{}
    },
}

func ProcessMessage(data []byte) {
    msg := messagePool.Get().(*domain.Message)
    defer messagePool.Put(msg)
    
    // 处理消息
    parseMessage(msg, data)
}
```

### 3. 并发控制

```go
// 使用 errgroup 控制并发
import "golang.org/x/sync/errgroup"

func processMailboxes(mailboxes []*domain.Mailbox) error {
    g, ctx := errgroup.WithContext(context.Background())
    g.SetLimit(10) // 最多10个并发
    
    for _, mb := range mailboxes {
        mb := mb // 捕获循环变量
        g.Go(func() error {
            return processSingleMailbox(ctx, mb)
        })
    }
    
    return g.Wait()
}
```

## MySQL 优化配置

### my.cnf 配置

```ini
[mysqld]
# 基础配置
max_connections = 500
max_connect_errors = 100
wait_timeout = 600
interactive_timeout = 600

# InnoDB 配置
innodb_buffer_pool_size = 4G          # 物理内存的 50-80%
innodb_log_file_size = 256M
innodb_log_buffer_size = 16M
innodb_flush_log_at_trx_commit = 2   # 性能优先
innodb_flush_method = O_DIRECT

# 查询缓存（MySQL 5.7）
query_cache_type = 1
query_cache_size = 256M
query_cache_limit = 2M

# 连接相关
thread_cache_size = 100
table_open_cache = 4096

# 慢查询日志
slow_query_log = 1
slow_query_log_file = /var/log/mysql/slow.log
long_query_time = 1
```

### 索引优化

```sql
-- 分析慢查询
SELECT * FROM mysql.slow_log ORDER BY query_time DESC LIMIT 10;

-- 检查索引使用情况
EXPLAIN SELECT * FROM messages WHERE mailbox_id = 'xxx' ORDER BY created_at DESC;

-- 添加缺失的索引
ALTER TABLE messages ADD INDEX idx_mailbox_time (mailbox_id, created_at DESC);
```

## PostgreSQL 优化配置

### postgresql.conf 配置

```ini
# 内存配置
shared_buffers = 2GB              # 物理内存的 25%
effective_cache_size = 6GB        # 物理内存的 75%
work_mem = 16MB
maintenance_work_mem = 512MB

# 连接配置
max_connections = 500
shared_preload_libraries = 'pg_stat_statements'

# WAL 配置
wal_buffers = 16MB
checkpoint_completion_target = 0.9
max_wal_size = 2GB
min_wal_size = 512MB

# 查询优化
random_page_cost = 1.1  # SSD 磁盘
effective_io_concurrency = 200

# 自动清理
autovacuum = on
autovacuum_max_workers = 4
autovacuum_naptime = 10s
```

## Redis 优化配置

### redis.conf 配置

```ini
# 内存配置
maxmemory 4gb
maxmemory-policy allkeys-lru

# 持久化（根据需求选择）
save ""  # 禁用 RDB（追求性能）
appendonly yes  # 启用 AOF
appendfsync everysec

# 网络配置
tcp-backlog 511
timeout 300
tcp-keepalive 300

# 客户端限制
maxclients 10000
```

## 监控告警

### Prometheus 配置

```yaml
# prometheus.yml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'tempmail'
    static_configs:
      - targets:
        - 'api1:8080'
        - 'api2:8080'
        - 'api3:8080'
    metrics_path: /metrics

  - job_name: 'mysql'
    static_configs:
      - targets: ['mysql-exporter:9104']

  - job_name: 'redis'
    static_configs:
      - targets: ['redis-exporter:9121']
```

### 告警规则

```yaml
# alerts.yml
groups:
  - name: tempmail_alerts
    rules:
      - alert: HighAPILatency
        expr: histogram_quantile(0.95, http_request_duration_seconds) > 0.5
        for: 5m
        annotations:
          summary: "API 响应时间过高"

      - alert: HighErrorRate
        expr: rate(http_requests_total{status=~"5.."}[5m]) > 10
        for: 5m
        annotations:
          summary: "错误率过高"

      - alert: DatabaseConnectionsHigh
        expr: mysql_global_status_threads_connected / mysql_global_variables_max_connections > 0.8
        for: 5m
        annotations:
          summary: "数据库连接数过高"
```

## 成本估算

### 月度成本（阿里云为例）

#### 小型部署（1万用户）
```
ECS (4C8G) x1     : ¥500/月
RDS MySQL (2C4G)  : ¥400/月
Redis (2G)        : ¥200/月
带宽 (10Mbps)     : ¥150/月
----------------------------
总计: ¥1,250/月
```

#### 中型部署（10万用户）
```
ECS (4C8G) x3     : ¥1,500/月
RDS MySQL (4C8G)  : ¥800/月
Redis 集群 (8G)   : ¥600/月
SLB 负载均衡      : ¥200/月
带宽 (50Mbps)     : ¥750/月
CDN 流量          : ¥300/月
----------------------------
总计: ¥4,150/月
```

#### 大型部署（100万用户）
```
ECS (8C16G) x10   : ¥10,000/月
RDS MySQL (8C16G) : ¥2,000/月
Redis 集群 (32G)  : ¥2,000/月
RabbitMQ 集群     : ¥1,000/月
SLB + CDN         : ¥2,000/月
带宽 (200Mbps)    : ¥3,000/月
----------------------------
总计: ¥20,000/月
```

## 实施优先级

### P0（必须做）
1. ✅ 数据库连接池优化
2. ✅ Redis 缓存部署
3. ✅ Nginx 反向代理
4. ✅ 慢查询监控

### P1（推荐做）
1. 数据库读写分离
2. 本地缓存实现
3. API 水平扩展（3台）
4. 性能监控（Prometheus）

### P2（可选做）
1. 消息队列集成
2. Redis 集群
3. 分库分表
4. 微服务拆分

## 预期效果

实施完 Phase 1-2 后：

| 指标 | 优化前 | 优化后 | 提升 |
|------|--------|--------|------|
| API QPS | 2,000 | 10,000+ | 5x |
| 响应时间 | 200ms | 50ms | 4x |
| 数据库负载 | 80% | 30% | 2.7x |
| 并发连接 | 500 | 5,000 | 10x |
| 系统可用性 | 99% | 99.9% | +0.9% |

---

**最后更新**: 2024-01-15  
**适用版本**: v0.8.2+
