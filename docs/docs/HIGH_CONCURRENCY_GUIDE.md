# 高并发架构优化指南

## 目标性能指标

| 指标 | 目标值 | 说明 |
|------|--------|------|
| API QPS | 10,000+ | 每秒 API 请求数 |
| SMTP 并发 | 1,000+ | 同时处理的 SMTP 连接 |
| WebSocket 连接 | 100,000+ | 同时在线 WebSocket 连接 |
| 邮件接收延迟 | < 2s | SMTP 接收到前端通知 |
| API 响应时间 | < 100ms | P95 响应时间 |
| 数据库 TPS | 50,000+ | 每秒事务数 |

## 架构优化方案

### 1. 整体架构演进

#### 当前架构（单体）
```
┌────────────┐
│   用户     │
└─────┬──────┘
      │
┌─────▼──────────────────────┐
│    Nginx (负载均衡)         │
└─────┬──────────────────────┘
      │
┌─────▼──────────────────────┐
│  Go Server (单实例)         │
│  • HTTP API                 │
│  • SMTP Server              │
│  • WebSocket Hub            │
└─────┬──────────────────────┘
      │
┌─────▼──────┐  ┌───────────┐
│  MySQL/PG  │  │   Redis   │
└────────────┘  └───────────┘
```

#### 推荐架构（高并发）
```
                   ┌─────────────┐
                   │   用户      │
                   └──────┬──────┘
                          │
        ┌─────────────────┼─────────────────┐
        │                 │                 │
   ┌────▼─────┐    ┌─────▼──────┐   ┌────▼─────┐
   │ CloudFlare│    │   Nginx    │   │  SMTP LB │
   │    CDN    │    │   (HTTP)   │   │  (TCP)   │
   └────┬─────┘    └─────┬──────┘   └────┬─────┘
        │                │                │
        │         ┌──────┴──────┐         │
        │         │             │         │
   ┌────▼───┐ ┌──▼──┐  ┌────▼──┐  ┌────▼───┐
   │ Static │ │ API │  │ API   │  │ SMTP   │
   │ Assets │ │ Pod1│  │ Pod2  │  │ Server │
   └────────┘ └──┬──┘  └────┬──┘  └────┬───┘
              │  WebSocket   │         │
              └──────┬───────┘         │
                     │                 │
        ┌────────────┼─────────────────┼──────┐
        │            │                 │      │
   ┌────▼────┐  ┌───▼────┐  ┌────────▼──┐  ┌▼──────┐
   │ Redis   │  │ Redis  │  │  MySQL    │  │ RabbitMQ│
   │ Cluster │  │ PubSub │  │  Cluster  │  │ Queue   │
   └─────────┘  └────────┘  └───────────┘  └─────────┘
```

## 核心优化策略

### 2. 数据库优化

#### 2.1 读写分离

**配置 MySQL 主从复制**：

```yaml
# docker-compose.yml
services:
  mysql-master:
    image: mysql:5.7
    environment:
      MYSQL_ROOT_PASSWORD: root
    volumes:
      - ./mysql-master.cnf:/etc/mysql/conf.d/custom.cnf
    ports:
      - "3306:3306"

  mysql-slave1:
    image: mysql:5.7
    environment:
      MYSQL_ROOT_PASSWORD: root
    volumes:
      - ./mysql-slave.cnf:/etc/mysql/conf.d/custom.cnf
    ports:
      - "3307:3306"
    depends_on:
      - mysql-master

  mysql-slave2:
    image: mysql:5.7
    environment:
      MYSQL_ROOT_PASSWORD: root
    ports:
      - "3308:3306"
    depends_on:
      - mysql-master
```

**应用层读写分离**：

```go
// internal/storage/mysql/cluster.go
type ClusterStore struct {
    master *sql.DB  // 写操作
    slaves []*sql.DB // 读操作
    current int
}

func (s *ClusterStore) GetDB(write bool) *sql.DB {
    if write {
        return s.master
    }
    
    // 轮询从库
    s.current = (s.current + 1) % len(s.slaves)
    return s.slaves[s.current]
}
```

#### 2.2 连接池优化

```go
// 优化连接池配置
db.SetMaxOpenConns(100)           // 最大连接数
db.SetMaxIdleConns(20)            // 最大空闲连接
db.SetConnMaxLifetime(1 * time.Hour)
db.SetConnMaxIdleTime(10 * time.Minute)
```

**推荐配置**：

| 场景 | MaxOpenConns | MaxIdleConns |
|------|--------------|--------------|
| 小型 (< 1000 QPS) | 25 | 5 |
| 中型 (< 10000 QPS) | 100 | 20 |
| 大型 (10000+ QPS) | 200-500 | 50-100 |

#### 2.3 分库分表

**垂直拆分**（按业务模块）：

```
tempmail_user    - 用户相关表
tempmail_mailbox - 邮箱相关表
tempmail_message - 邮件相关表
tempmail_domain  - 域名相关表
```

**水平拆分**（按邮箱 ID）：

```go
// 按邮箱 ID 取模分表
func getTableName(mailboxID string) string {
    hash := crc32.ChecksumIEEE([]byte(mailboxID))
    tableIndex := hash % 16 // 16 个分表
    return fmt.Sprintf("messages_%02d", tableIndex)
}
```

**分表配置**：

```sql
-- 创建 16 个消息表
CREATE TABLE messages_00 LIKE messages;
CREATE TABLE messages_01 LIKE messages;
...
CREATE TABLE messages_15 LIKE messages;
```

#### 2.4 索引优化

```sql
-- 复合索引（按查询模式设计）
CREATE INDEX idx_messages_mailbox_time 
ON messages(mailbox_id, created_at DESC);

-- 覆盖索引（避免回表）
CREATE INDEX idx_messages_list 
ON messages(mailbox_id, id, subject, from_address, is_read, created_at);

-- 分区表（按时间分区）
ALTER TABLE messages 
PARTITION BY RANGE (UNIX_TIMESTAMP(created_at)) (
    PARTITION p202401 VALUES LESS THAN (UNIX_TIMESTAMP('2024-02-01')),
    PARTITION p202402 VALUES LESS THAN (UNIX_TIMESTAMP('2024-03-01')),
    ...
);
```

### 3. 缓存优化

#### 3.1 多级缓存架构

```
用户请求
    ↓
┌─────────────────┐
│ L1: 本地缓存     │ (sync.Map, 容量限制)
│ TTL: 5s         │
└────────┬────────┘
         ↓ (miss)
┌─────────────────┐
│ L2: Redis 缓存  │ (Redis Cluster)
│ TTL: 1h         │
└────────┬────────┘
         ↓ (miss)
┌─────────────────┐
│ L3: 数据库      │ (MySQL/PostgreSQL)
└─────────────────┘
```

**实现代码**：

```go
// internal/cache/multilevel.go
type MultiLevelCache struct {
    l1 *sync.Map              // 本地缓存
    l2 *redis.Client          // Redis
    db storage.Store          // 数据库
}

func (c *MultiLevelCache) GetMailbox(id string) (*domain.Mailbox, error) {
    // L1: 本地缓存
    if val, ok := c.l1.Load(id); ok {
        return val.(*domain.Mailbox), nil
    }
    
    // L2: Redis
    var mailbox domain.Mailbox
    err := c.l2.Get(ctx, "mailbox:"+id).Scan(&mailbox)
    if err == nil {
        c.l1.Store(id, &mailbox)
        return &mailbox, nil
    }
    
    // L3: 数据库
    mailbox, err = c.db.GetMailbox(id)
    if err != nil {
        return nil, err
    }
    
    // 回写缓存
    c.l2.Set(ctx, "mailbox:"+id, mailbox, 1*time.Hour)
    c.l1.Store(id, &mailbox)
    
    return &mailbox, nil
}
```

#### 3.2 Redis 集群

```yaml
# docker-compose.yml
services:
  redis-node-1:
    image: redis:7-alpine
    command: redis-server --cluster-enabled yes --cluster-config-file nodes.conf
    ports:
      - "7000:6379"
      
  redis-node-2:
    image: redis:7-alpine
    command: redis-server --cluster-enabled yes --cluster-config-file nodes.conf
    ports:
      - "7001:6379"
      
  redis-node-3:
    image: redis:7-alpine
    command: redis-server --cluster-enabled yes --cluster-config-file nodes.conf
    ports:
      - "7002:6379"
```

**创建集群**：

```bash
redis-cli --cluster create \
  127.0.0.1:7000 \
  127.0.0.1:7001 \
  127.0.0.1:7002 \
  --cluster-replicas 0
```

#### 3.3 缓存预热

```go
// 启动时预热热点数据
func warmupCache(cache *MultiLevelCache) {
    // 1. 预热活跃域名
    domains, _ := systemDomainService.ListActiveSystemDomains()
    for _, d := range domains {
        cache.Set("domain:"+d.Domain, d, 24*time.Hour)
    }
    
    // 2. 预热活跃用户
    activeUsers, _ := userService.GetActiveUsers(1000)
    for _, u := range activeUsers {
        cache.Set("user:"+u.ID, u, 1*time.Hour)
    }
}
```

### 4. 消息队列解耦

#### 4.1 邮件处理异步化

```
SMTP 接收 → 快速响应 (200 OK)
    ↓
消息队列 (RabbitMQ/Kafka)
    ↓
后台 Workers (并发处理)
    ↓
存储 + 通知
```

**实现**：

```go
// internal/queue/mail_processor.go
type MailQueue struct {
    ch *amqp.Channel
}

// SMTP 服务器快速入队
func (q *MailQueue) Enqueue(mail *RawMail) error {
    body, _ := json.Marshal(mail)
    return q.ch.Publish(
        "mail_exchange", // exchange
        "mail.received", // routing key
        false,
        false,
        amqp.Publishing{
            ContentType: "application/json",
            Body:        body,
        },
    )
}

// Worker 并发消费
func (q *MailQueue) StartWorkers(n int) {
    for i := 0; i < n; i++ {
        go q.worker(i)
    }
}

func (q *MailQueue) worker(id int) {
    msgs, _ := q.ch.Consume("mail_queue", "", false, false, false, false, nil)
    
    for msg := range msgs {
        var mail RawMail
        json.Unmarshal(msg.Body, &mail)
        
        // 处理邮件
        err := processEmail(&mail)
        if err != nil {
            msg.Nack(false, true) // 重新入队
        } else {
            msg.Ack(false)
        }
    }
}
```

#### 4.2 批量处理

```go
// 批量插入邮件
func (s *Store) BatchInsertMessages(messages []*domain.Message) error {
    // 100 条一批
    batchSize := 100
    for i := 0; i < len(messages); i += batchSize {
        end := i + batchSize
        if end > len(messages) {
            end = len(messages)
        }
        
        batch := messages[i:end]
        if err := s.insertBatch(batch); err != nil {
            return err
        }
    }
    return nil
}
```

### 5. SMTP 服务器优化

#### 5.1 并发连接限制

```go
// internal/smtp/backend.go
type Backend struct {
    semaphore chan struct{} // 限制并发
}

func NewBackend(maxConcurrent int) *Backend {
    return &Backend{
        semaphore: make(chan struct{}, maxConcurrent),
    }
}

func (b *Backend) NewSession(c *gosmtp.Conn) (gosmtp.Session, error) {
    // 获取信号量
    select {
    case b.semaphore <- struct{}{}:
        return &session{
            backend: b,
        }, nil
    default:
        return nil, &gosmtp.SMTPError{
            Code:    421,
            Message: "too many connections, please try again later",
        }
    }
}

func (s *session) Logout() error {
    <-s.backend.semaphore // 释放信号量
    return nil
}
```

#### 5.2 连接池复用

```go
// SMTP 服务器配置
smtpServer := gosmtp.NewServer(smtpBackend)
smtpServer.MaxRecipients = 10
smtpServer.MaxMessageBytes = 10 * 1024 * 1024
smtpServer.MaxConnections = 1000        // 最大并发连接
smtpServer.ReadTimeout = 30 * time.Second
smtpServer.WriteTimeout = 30 * time.Second
```

### 6. WebSocket 优化

#### 6.1 分布式 WebSocket

```go
// internal/websocket/distributed_hub.go
type DistributedHub struct {
    redis    *redis.Client
    localHub *Hub
}

// 订阅 Redis PubSub 跨节点通知
func (h *DistributedHub) Start() {
    pubsub := h.redis.Subscribe(ctx, "websocket:broadcast")
    
    go func() {
        for msg := range pubsub.Channel() {
            // 转发到本地 WebSocket 客户端
            h.localHub.Broadcast(msg.Payload)
        }
    }()
}

// 发布消息到所有节点
func (h *DistributedHub) NotifyNewMail(mailboxID string, message *domain.Message) {
    data, _ := json.Marshal(message)
    h.redis.Publish(ctx, "websocket:broadcast", data)
}
```

#### 6.2 连接管理优化

```go
// 使用更高效的数据结构
type Hub struct {
    clients    sync.Map // 代替 map + mutex
    register   chan *Client
    unregister chan *Client
    broadcast  chan []byte
}

// 批量广播
func (h *Hub) broadcastBatch(messages [][]byte) {
    h.clients.Range(func(key, value interface{}) bool {
        client := value.(*Client)
        for _, msg := range messages {
            select {
            case client.send <- msg:
            default:
                // 客户端阻塞，关闭连接
                close(client.send)
                h.clients.Delete(key)
            }
        }
        return true
    })
}
```

### 7. API 网关和限流

#### 7.1 Nginx 配置

```nginx
# nginx.conf
upstream api_backend {
    least_conn;  # 最少连接负载均衡
    server api1:8080 max_fails=3 fail_timeout=30s;
    server api2:8080 max_fails=3 fail_timeout=30s;
    server api3:8080 max_fails=3 fail_timeout=30s;
    keepalive 100;
}

server {
    listen 80;
    
    # 限流
    limit_req_zone $binary_remote_addr zone=api_limit:10m rate=100r/s;
    limit_req zone=api_limit burst=200 nodelay;
    
    # 连接限制
    limit_conn_zone $binary_remote_addr zone=conn_limit:10m;
    limit_conn conn_limit 20;
    
    location /v1/ {
        proxy_pass http://api_backend;
        proxy_http_version 1.1;
        proxy_set_header Connection "";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        
        # 超时配置
        proxy_connect_timeout 5s;
        proxy_send_timeout 10s;
        proxy_read_timeout 10s;
    }
}
```

#### 7.2 应用层限流

**令牌桶算法**：

```go
// internal/middleware/rate_limit.go
type TokenBucket struct {
    capacity int64
    tokens   int64
    rate     int64
    lastTime time.Time
    mu       sync.Mutex
}

func (tb *TokenBucket) Allow() bool {
    tb.mu.Lock()
    defer tb.mu.Unlock()
    
    now := time.Now()
    elapsed := now.Sub(tb.lastTime).Seconds()
    
    // 补充令牌
    tb.tokens = min(tb.capacity, tb.tokens + int64(elapsed*float64(tb.rate)))
    tb.lastTime = now
    
    if tb.tokens > 0 {
        tb.tokens--
        return true
    }
    return false
}
```

### 8. 监控和优化

#### 8.1 Prometheus 监控

```go
// internal/monitoring/metrics.go
var (
    httpRequestTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "http_requests_total",
        },
        []string{"method", "endpoint", "status"},
    )
    
    httpRequestDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "http_request_duration_seconds",
            Buckets: prometheus.DefBuckets,
        },
        []string{"method", "endpoint"},
    )
    
    smtpConnectionsActive = promauto.NewGauge(
        prometheus.GaugeOpts{
            Name: "smtp_connections_active",
        },
    )
    
    websocketConnectionsActive = promauto.NewGauge(
        prometheus.GaugeOpts{
            Name: "websocket_connections_active",
        },
    )
)
```

#### 8.2 性能分析

```go
// 启用 pprof
import _ "net/http/pprof"

go func() {
    log.Println(http.ListenAndServe("localhost:6060", nil))
}()
```

**分析命令**：

```bash
# CPU 分析
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30

# 内存分析
go tool pprof http://localhost:6060/debug/pprof/heap

# 协程分析
go tool pprof http://localhost:6060/debug/pprof/goroutine
```

### 9. 部署架构

#### 9.1 Kubernetes 部署

```yaml
# k8s/api-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: tempmail-api
spec:
  replicas: 5  # 5 个副本
  selector:
    matchLabels:
      app: tempmail-api
  template:
    metadata:
      labels:
        app: tempmail-api
    spec:
      containers:
      - name: api
        image: tempmail/api:latest
        resources:
          requests:
            memory: "512Mi"
            cpu: "500m"
          limits:
            memory: "1Gi"
            cpu: "1000m"
        env:
        - name: DATABASE_TYPE
          value: "mysql"
        - name: DATABASE_DSN
          valueFrom:
            secretKeyRef:
              name: db-secret
              key: dsn
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
---
apiVersion: v1
kind: Service
metadata:
  name: tempmail-api
spec:
  selector:
    app: tempmail-api
  ports:
  - port: 80
    targetPort: 8080
  type: LoadBalancer
---
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: tempmail-api-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: tempmail-api
  minReplicas: 3
  maxReplicas: 20
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
```

#### 9.2 Docker Compose 示例

```yaml
version: '3.8'

services:
  # API 服务器（多实例）
  api1:
    image: tempmail/api:latest
    environment:
      - DATABASE_TYPE=mysql
      - DATABASE_DSN=${DB_DSN}
      - REDIS_ADDRESS=redis-cluster:6379
    deploy:
      resources:
        limits:
          cpus: '1'
          memory: 1G

  api2:
    image: tempmail/api:latest
    environment:
      - DATABASE_TYPE=mysql
      - DATABASE_DSN=${DB_DSN}
      - REDIS_ADDRESS=redis-cluster:6379

  api3:
    image: tempmail/api:latest
    environment:
      - DATABASE_TYPE=mysql
      - DATABASE_DSN=${DB_DSN}
      - REDIS_ADDRESS=redis-cluster:6379

  # Nginx 负载均衡
  nginx:
    image: nginx:alpine
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf
    depends_on:
      - api1
      - api2
      - api3
```

## 性能测试

### 压力测试工具

```bash
# 1. API 压测 (wrk)
wrk -t12 -c400 -d30s --latency http://localhost:8080/v1/mailboxes

# 2. SMTP 压测 (smtp-source)
smtp-source -s 100 -m 1000 -c localhost:25

# 3. WebSocket 压测 (websocket-bench)
websocket-bench -c 10000 -s 1000 ws://localhost:8080/v1/ws
```

### 预期性能

| 配置 | QPS | 并发 | 响应时间 (P95) |
|------|-----|------|----------------|
| 单机 (4C8G) | ~3,000 | 500 | 50ms |
| 3节点 (4C8G) | ~10,000 | 1,500 | 80ms |
| 5节点 (8C16G) | ~30,000 | 5,000 | 100ms |
| 10节点 + 优化 | 50,000+ | 10,000+ | 150ms |

## 成本优化建议

1. **使用云服务商的托管服务**：
   - 阿里云 RDS (MySQL)
   - 阿里云 Redis
   - 负载均衡 SLB

2. **按需扩容**：
   - 高峰期自动扩容
   - 低谷期自动缩容

3. **CDN 加速**：
   - 静态资源 CDN
   - API 缓存加速

4. **数据生命周期管理**：
   - 定期归档旧邮件
   - 冷热数据分离

## 检查清单

- [ ] 数据库读写分离
- [ ] Redis 集群部署
- [ ] 多级缓存实现
- [ ] 消息队列解耦
- [ ] API 网关限流
- [ ] SMTP 并发控制
- [ ] WebSocket 分布式
- [ ] 监控告警完善
- [ ] 性能测试通过
- [ ] 自动扩缩容配置

---

**最后更新**: 2024-01-15  
**适用版本**: v0.8.2+
