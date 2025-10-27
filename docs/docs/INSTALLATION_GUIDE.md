# 临时邮箱系统安装与部署指南

本文档详细说明如何在开发、测试与生产环境中部署临时邮箱系统。请严格按照步骤执行，并结合 `PRD.md` 与 `TECH_STACK.md` 获取更多设计与实现细节。

---

## 1. 项目结构与服务划分

| 服务/组件 | 作用 | 技术栈 | 运行方式 |
|-----------|------|--------|----------|
| API 服务 | 提供 REST API、WebSocket、认证授权 | Go 1.25+（推荐 1.25.2）+ Gin | 二进制/容器 |
| SMTP 服务 | 接收外部邮件、解析 MIME | Go 1.25+（推荐 1.25.2）+ go-smtp | 二进制/容器 |
| 定时任务 | 邮件清理、配额重置、统计汇总 | Go 1.25+（推荐 1.25.2）+ cron | 二进制/容器 |
| 用户前台 | 游客/注册用户界面 | Vue 3.5.22 + Vite + Arco | 静态资源/容器 |
| 管理后台 | 管理员配置域名、用户、监控 | Vue 3.5.22 + Vite + Arco | 静态资源/容器 |
| PostgreSQL | 持久化用户、域名、日志等 | PostgreSQL 18 | 托管/自建/容器 |
| Redis Cluster | 邮件缓存、限流、PubSub | Redis 8.2 | 托管/自建/容器 |
| Nginx/Caddy | 统一反向代理、SSL、WebSocket 升级 | Nginx 1.26+ / Caddy 2 | 二进制/容器 |

---

## 2. 环境与依赖要求

### 2.1 系统要求

- 操作系统：Linux (Ubuntu 22.04+ 推荐) / macOS / Windows Server 2022
- CPU：至少 4 核
- 内存：至少 8 GB（生产建议 16 GB+）
- 磁盘：SSD，预留 50 GB+

### 2.2 软件依赖

| 依赖 | 最低版本 | 用途 |
|------|----------|------|
| Go | 1.25+（推荐 1.25.2） | 编译后端、SMTP、任务服务 |
| Node.js | 20 LTS | 构建前端 |
| pnpm | 9.x | 前端包管理 |
| PostgreSQL | 18.0 | 数据库 |
| Redis | 8.2 | 缓存/队列 |
| Docker | 26.x | 容器化部署（可选） |
| Docker Compose | v2.27+ | 本地多服务编排 |
| Make | 4.x | 若使用脚本自动化 |
| Nginx/Caddy | 最新稳定版 | 网关层 |

---

## 3. 快速开始（Docker Compose）

1. 安装 Docker 与 Docker Compose。
2. 克隆代码仓库：
   ```bash
   git clone <your_repo_url> tempmail
   cd tempmail
   ```
3. 复制环境变量模板：
   ```bash
   cp .env.example .env
   cp deploy/docker-compose.example.yml docker-compose.yml
   ```
4. 按需修改 `.env` 中的以下关键配置：
   - `POSTGRES_PASSWORD`
   - `POSTGRES_DB`
   - `REDIS_PASSWORD`
   - `JWT_SECRET`
   - `SMTP_HOSTNAME` / `SMTP_DOMAINS`
5. 拉取并启动服务：
   ```bash
   docker compose pull
   docker compose up -d
   ```
6. 验证运行状态：
   ```bash
   docker compose ps
   docker compose logs -f api
   ```
7. 访问地址：
   - 用户前台：`https://<your-domain>`
   - 管理后台：`https://admin.<your-domain>`
   - API：`https://api.<your-domain>/v1`

> **提示**：首次部署完成后请立即进入管理后台，添加可用的邮箱域名并进行 DKIM/SPF/DMARC 校验。

---

## 4. 手动部署流程

### 4.1 代码准备

```bash
git clone <your_repo_url> tempmail
cd tempmail
go mod download
pnpm install --filter frontend-user --filter frontend-admin --filter frontend-public
```

推荐创建独立配置目录：
```
config/
  |- env/
      |- api.env
      |- smtp.env
      |- cron.env
  |- nginx/
      |- sites-enabled/
```

### 4.2 数据库初始化（PostgreSQL）

1. 创建数据库与用户：
   ```sql
   CREATE DATABASE tempmail;
   CREATE USER tempmail WITH ENCRYPTED PASSWORD '<strong_password>';
   GRANT ALL PRIVILEGES ON DATABASE tempmail TO tempmail;
   ```
2. 执行迁移（假设使用 `golang-migrate`）：
   ```bash
   migrate -path migrations -database "postgres://tempmail:<strong_password>@localhost:5432/tempmail?sslmode=disable" up
   ```
3. 启用扩展（如需全文搜索/JSON）：
   ```sql
   CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
   CREATE EXTENSION IF NOT EXISTS "pgcrypto";
   ```

### 4.3 缓存与消息队列（Redis）

1. 生产环境建议部署 Redis Cluster（3 主 3 从）。
2. 开发生态可使用单实例：
   ```bash
   redis-server --requirepass "<redis_password>"
   ```
3. 设置持久化（AOF）保障邮件缓存可恢复。
4. 配置防火墙仅允许内网访问。

### 4.4 综合服务（HTTP + SMTP）

1. 创建配置文件 `config/env/server.env`（示例）：
   ```env
   TEMPMAIL_SERVER_HOST=0.0.0.0
   TEMPMAIL_SERVER_PORT=8080

   TEMPMAIL_MAILBOX_ALLOWED_DOMAINS=temp.mail,tempmail.dev
   TEMPMAIL_MAILBOX_DEFAULT_TTL=24h

   TEMPMAIL_SMTP_BIND_ADDR=:25
   TEMPMAIL_SMTP_DOMAIN=temp.mail

   TEMPMAIL_CORS_ALLOWED_ORIGINS=http://localhost:5173,http://localhost:5273
   ```
   以上配置基于内存存储，适合开发与测试环境。若需 PostgreSQL/Redis，请结合 `TECH_STACK.md` 中的持久化方案扩展存储实现。

2. 启动同时包含 HTTP API 与 SMTP 的综合服务：
   ```bash
   go run ./cmd/server
   ```

3. 验证状态：
   ```bash
   curl http://localhost:8080/health
   swaks --to test@temp.mail --server localhost:25 --ehlo localhost
   ```

> 若希望分别运行 HTTP 与 SMTP，可继续使用 `./cmd/api` 与自定义 SMTP 服务；但推荐使用 `cmd/server` 以共享同一内存存储，避免跨进程状态不同步。

### 4.5 定时任务服务（可选）

1. 配置 `config/env/cron.env`：
   ```env
   CRON_SCHEDULE_CLEANUP=0 * * * *
   CRON_SCHEDULE_QUOTA=0 0 * * *
   CRON_SCHEDULE_STATS=*/10 * * * *

   POSTGRES_DSN=postgres://tempmail:<password>@localhost:5432/tempmail?sslmode=disable
   REDIS_ADDR=localhost:6379
   REDIS_PASSWORD=<redis_password>

   LOG_LEVEL=info
   ```
2. 编译与运行：
   ```bash
   go build -o bin/cron ./cmd/cron
   ./bin/cron --config=config/env/cron.env
   ```

### 4.7 前端应用构建

以 `frontend-user` 为例，其他前端目录操作一致：

```bash
cd frontend-user
cp .env.example .env
pnpm install
pnpm build
```

关键环境变量说明：
```env
VITE_APP_API_BASE=https://api.tempmail.dev
VITE_APP_WS_URL=wss://api.tempmail.dev/ws
VITE_APP_ENABLE_PUSH=true
```

构建输出通常位于 `dist/`，可通过 Nginx 提供静态托管。

### 4.8 网关与负载均衡

示例 Nginx 配置（简化版）：
```nginx
server {
    listen 443 ssl http2;
    server_name tempmail.dev;

    ssl_certificate     /etc/nginx/ssl/fullchain.pem;
    ssl_certificate_key /etc/nginx/ssl/privkey.pem;

    location / {
        root /var/www/tempmail/user;
        try_files $uri /index.html;
    }
}

server {
    listen 443 ssl http2;
    server_name api.tempmail.dev;

    ssl_certificate     /etc/nginx/ssl/fullchain.pem;
    ssl_certificate_key /etc/nginx/ssl/privkey.pem;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }
}
```

---

## 5. 配置清单

| 类别 | 配置项 | 说明 |
|------|--------|------|
| 数据库 | `POSTGRES_DSN` | 包含用户名、密码、库名、SSL 模式 |
| Redis | `REDIS_ADDR` / `REDIS_PASSWORD` | 支持哨兵/集群 |
| 认证 | `JWT_SECRET` | 建议 32 字节随机字符串 |
| 限流 | `RATE_LIMIT_REQUESTS` / `RATE_LIMIT_DURATION` | 针对 IP 或用户级别 |
| 邮箱 | `SMTP_ALLOWED_DOMAINS` | 逗号分隔域名列表 |
| 存储 | `MESSAGE_TTL` | 邮件有效期（默认 24h） |
| 推送 | `WEBSOCKET_ORIGINS` | 允许的前端来源 |
| 日志 | `LOG_LEVEL` | debug/info/warn/error |

> 建议采用 Vault/Secret Manager 管理敏感变量。

---

## 6. DNS 与安全加固

1. **域名解析**
   - `A` 记录：`api.tempmail.dev` → API 服务器
   - `A` 记录：`smtp.tempmail.dev` → SMTP 服务器
   - `CNAME`：`admin.tempmail.dev` → 前端 CDN
   - `MX`：`@` → `smtp.tempmail.dev`
2. **SPF**：`v=spf1 ip4:<smtp_ip> -all`
3. **DKIM**：使用后端脚本生成密钥，添加 `TXT` 记录。
4. **DMARC**：`v=DMARC1; p=quarantine; rua=mailto:dmarc@tempmail.dev`
5. **TLS/HTTPS**：使用 Let’s Encrypt / ACME 自动续期。
6. **防火墙**：
   - 开放端口：`80/443`（HTTP/HTTPS）、`25`（SMTP）、`8080`（内网 API）
   - 禁止外部访问数据库与 Redis。

---

## 7. 功能验证

1. **单元测试**
   ```bash
   go test ./...
   pnpm test --filter frontend-user
   ```
2. **API 自检**
   - `POST /v1/auth/login`
   - `POST /v1/mailbox`
   - `GET /v1/mailbox/:id/messages`
   - `DELETE /v1/mailbox/:id`
3. **SMTP 测试**
   - 使用 `swaks` 或 `openssl s_client` 向临时邮箱发送测试邮件。
   - 确认 API/前端收到实时推送或轮询结果。
4. **前端检查**
   - 游客生成邮箱、查看邮件、删除邮箱。
   - 注册用户多邮箱管理、自定义前缀。
   - 管理后台域名验证、用户封禁。

---

## 8. 运维建议

- **日志**：统一输出 JSON，接入 ELK/Loki。
- **监控**：Prometheus + Grafana，关注 API 延迟、SMTP 队列长度、Redis 命中率。
- **报警**：配置邮件/短信/IM 告警，重点监控服务宕机、队列堆积、MX 失败。
- **备份**：每日备份 PostgreSQL，全量 + WAL；Redis AOF 备份到对象存储。
- **升级**：灰度发布，滚动重启；前端开启版本号强制刷新。
- **安全**：定期更换密钥与证书，启用 WAF/Rate Limit。

---

## 9. 常见问题排查

| 问题 | 可能原因 | 排查建议 |
|------|----------|----------|
| 收不到邮件 | DNS 未生效 / SPF 不匹配 | 使用 `dig` 检查 MX/SPF；查看 SMTP 日志 |
| 邮件丢失 | Redis TTL 过短 / 过期任务异常 | 调整 `MESSAGE_TTL`；检查定时任务 cron |
| API 401 | JWT 过期 / 时钟不同步 | 确认服务器时间；刷新令牌 |
| WebSocket 断开 | Nginx 未转发 Upgrade 头 | 检查网关配置 |
| 前端跨域 | `WEBSOCKET_ORIGINS` 或 CORS 配置缺失 | 更新配置，重新部署 |

---

## 10. 发布流程建议

1. Git 分支策略：`main`（生产）、`develop`（集成）、`feature/*`。
2. CI/CD：
   - 触发：PR → 运行测试 → 构建镜像 → 推送 Registry。
   - 部署：ArgoCD / GitOps 或 GitLab CI。
3. 灰度策略：
   - API：多实例 + 负载均衡。
   - 前端：CDN 回滚、版本标记。
4. 文档与版本：
   - 更新 `CHANGELOG.md`。
   - 标记 Docker 镜像版本。
   - 同步发布公告与用户通知。

---

## 11. 附录

- 相关文档：
  - `docs/PRD.md`：产品需求说明
  - `docs/TECH_STACK.md`：技术选型与架构
- 推荐工具：
  - 邮件调试：`MailHog`、`Papercut`
  - DNS 排错：`dig`、`nslookup`
  - 负载测试：`k6`、`wrk`
- 术语简表：
  - **MX**：邮件交换记录
  - **SPF**：发件人策略框架
  - **DKIM**：域名密钥识别邮件
  - **DMARC**：基于域的消息认证
  - **TTL**：生存时间

---

完成上述步骤后，即可获得一个可用、可扩展且易于维护的临时邮箱系统部署环境。若需进一步的定制与优化，请参考技术栈文档内的扩展建议，并结合团队自身的 CI/CD 与监控体系进行调整。
