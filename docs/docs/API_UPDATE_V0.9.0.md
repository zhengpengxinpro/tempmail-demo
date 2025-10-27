# API 端点更新总结 v0.9.0

**更新日期**: 2025-10-16  
**适用版本**: v0.9.0  
**文档类型**: 更新日志与API参考

---

## 目录

- [更新概览](#更新概览)
- [核心问题修复](#核心问题修复)
- [新增功能](#新增功能)
- [完整API端点清单](#完整api端点清单)
- [认证方式说明](#认证方式说明)
- [使用示例](#使用示例)
- [注意事项](#注意事项)
- [待完成功能](#待完成功能)

---

## 更新概览

本次更新（v0.9.0）完成了后端API的全面修复和完善，解决了邮箱令牌认证问题，新增了批量操作、健康检查和管理员创建功能。

### 已完成的更新

#### 1. 修复邮件标签添加认证问题

**问题描述**: 邮件标签端点路由要求邮箱token，但handler需要JWT认证，导致认证不一致

**解决方案**: 添加双重认证中间件（邮箱Token + JWT认证）

**影响端点**:
- `POST /v1/mailboxes/:id/messages/:messageId/tags` 
- `GET /v1/mailboxes/:id/messages/:messageId/tags`
- `DELETE /v1/mailboxes/:id/messages/:messageId/tags/:tagId`

#### 2. 添加完整的健康检查端点

支持 Kubernetes/Docker 等容器编排系统的健康监控。

**新增端点**:
- `GET /health` - 基础健康检查
- `GET /health/live` - 存活检查（Liveness Probe）
- `GET /health/ready` - 就绪检查（Readiness Probe）

#### 3. 实现批量邮件操作

支持一键清空邮箱所有邮件。

**新增端点**: 
- `DELETE /v1/mailboxes/:id/messages` - 清空邮箱所有邮件

**新增Service方法**:
- `MessageService.Delete(mailboxID, messageID)` - 删除单个邮件
- `MessageService.ClearAll(mailboxID)` - 清空所有邮件

**新增存储接口**:
- `DeleteMessage(mailboxID, messageID)` - 删除单个消息
- `DeleteAllMessages(mailboxID)` - 批量删除消息

#### 4. 创建管理员用户功能

提供开发测试用的管理员用户创建接口。

**新增端点**:
- `POST /v1/debug/admin/create` - 创建管理员/超级管理员用户

**请求格式**:
```json
{
  "email": "admin@example.com",
  "password": "securepassword",
  "username": "admin",
  "role": "admin"
}
```

**role 参数**:
- `admin` - 普通管理员
- `super` - 超级管理员

**新增Service方法**:
- `AdminService.CreateAdminUser(input)` - 创建管理员用户

---

## 核心问题修复

### 邮箱令牌认证问题

#### 问题描述

之前所有需要邮箱令牌的端点都返回 `401 Unauthorized` 错误，导致邮箱相关功能无法使用。

#### 原因分析

- 邮箱令牌传递方式不正确
- 测试时使用了错误的请求头名称或参数位置

#### 正确的令牌传递方式

邮箱认证中间件支持以下三种方式（优先级从高到低）：

##### 1. X-Mailbox-Token 请求头（推荐）✅

```http
GET /v1/mailboxes/{id} HTTP/1.1
Host: localhost:8080
X-Mailbox-Token: your-mailbox-token-here
```

**PowerShell 示例**:
```powershell
Invoke-WebRequest -Uri "http://localhost:8080/v1/mailboxes/{id}" `
  -Method GET `
  -Headers @{"X-Mailbox-Token"="your-mailbox-token-here"}
```

##### 2. Authorization Bearer（备选）

```http
GET /v1/mailboxes/{id} HTTP/1.1
Host: localhost:8080
Authorization: Bearer your-mailbox-token-here
```

**PowerShell 示例**:
```powershell
Invoke-WebRequest -Uri "http://localhost:8080/v1/mailboxes/{id}" `
  -Method GET `
  -Headers @{"Authorization"="Bearer your-mailbox-token-here"}
```

##### 3. Query 参数（URL参数）

```http
GET /v1/mailboxes/{id}?token=your-mailbox-token-here HTTP/1.1
Host: localhost:8080
```

**PowerShell 示例**:
```powershell
Invoke-WebRequest -Uri "http://localhost:8080/v1/mailboxes/{id}?token=your-mailbox-token-here" `
  -Method GET
```

#### 测试验证结果

以下端点已通过完整测试验证：

- ✅ 创建邮箱 - `POST /v1/mailboxes`
- ✅ 获取邮箱详情 - `GET /v1/mailboxes/:id`
- ✅ 获取邮件列表 - `GET /v1/mailboxes/:id/messages`
- ✅ 创建邮件 - `POST /v1/mailboxes/:id/messages`
- ✅ 获取邮件详情 - `GET /v1/mailboxes/:id/messages/:messageId`
- ✅ 创建别名 - `POST /v1/mailboxes/:id/aliases`
- ✅ 切换别名状态 - `PATCH /v1/mailboxes/:id/aliases/:aliasId`
- ✅ WebSocket连接 - `GET /v1/ws` (返回101协议切换)
- ✅ 健康检查 - `GET /health`
- ✅ 自定义邮箱前缀 - 支持localPart参数

---

## 新增功能

### 批量邮件清空

允许用户一键清空邮箱中的所有邮件。

**端点**: `DELETE /v1/mailboxes/:id/messages`

**认证**: 需要邮箱Token

**响应示例**:
```json
{
  "code": 200,
  "msg": "成功",
  "data": {
    "message": "邮件清空成功",
    "deleted": 15
  }
}
```

### 健康检查端点

提供三个不同级别的健康检查端点，适用于不同的监控场景。

#### 基础健康检查

**端点**: `GET /health`

**用途**: 快速检查服务是否运行

**响应**:
```json
{
  "status": "ok"
}
```

#### 存活检查（Liveness Probe）

**端点**: `GET /health/live`

**用途**: Kubernetes 存活探针，检查应用是否还活着

**响应**:
```json
{
  "status": "ok",
  "timestamp": "2025-10-16T08:00:00Z",
  "service": "tempmail-backend"
}
```

#### 就绪检查（Readiness Probe）

**端点**: `GET /health/ready`

**用途**: Kubernetes 就绪探针，检查应用是否准备好接收流量

**响应**:
```json
{
  "status": "ready",
  "timestamp": "2025-10-16T08:00:00Z",
  "service": "tempmail-backend",
  "dependencies": {
    "database": "ok",
    "storage": "ok"
  }
}
```

### 管理员用户创建

提供便捷的管理员用户创建接口，用于开发测试。

**端点**: `POST /v1/debug/admin/create`

**认证**: 无（仅用于开发环境）

**请求体**:
```json
{
  "email": "admin@example.com",
  "password": "securepassword123",
  "username": "admin",
  "role": "super"
}
```

**响应**:
```json
{
  "code": 201,
  "msg": "创建成功",
  "data": {
    "id": "user-uuid",
    "email": "admin@example.com",
    "username": "admin",
    "role": "super",
    "tier": "free",
    "isActive": true,
    "isEmailVerified": true,
    "createdAt": "2025-10-16T08:00:00Z"
  }
}
```

---

## 完整API端点清单

### 认证相关（Authentication）

| 方法 | 端点 | 认证要求 | 说明 |
|------|------|----------|------|
| POST | `/v1/auth/register` | 无 | 用户注册 |
| POST | `/v1/auth/login` | 无 | 用户登录 |
| POST | `/v1/auth/refresh` | 无 | 刷新令牌 |
| GET | `/v1/auth/me` | JWT | 获取当前用户信息 |

### 邮箱管理（Mailboxes）

| 方法 | 端点 | 认证要求 | 说明 |
|------|------|----------|------|
| POST | `/v1/mailboxes` | JWT(可选) | 创建临时邮箱 |
| GET | `/v1/mailboxes` | JWT(可选) | 获取邮箱列表 |
| GET | `/v1/mailboxes/:id` | 邮箱Token | 获取邮箱详情 |
| DELETE | `/v1/mailboxes/:id` | 邮箱Token | 删除邮箱 |

### 邮件管理（Messages）

| 方法 | 端点 | 认证要求 | 说明 |
|------|------|----------|------|
| POST | `/v1/mailboxes/:id/messages` | 邮箱Token | 创建邮件 |
| GET | `/v1/mailboxes/:id/messages` | 邮箱Token | 获取邮件列表 |
| DELETE | `/v1/mailboxes/:id/messages` | 邮箱Token | 🆕 清空所有邮件 |
| GET | `/v1/mailboxes/:id/messages/:messageId` | 邮箱Token | 获取邮件详情 |
| POST | `/v1/mailboxes/:id/messages/:messageId/read` | 邮箱Token | 标记邮件已读 |
| GET | `/v1/mailboxes/:id/messages/search` | 邮箱Token | 搜索邮件 |

### 附件管理（Attachments）

| 方法 | 端点 | 认证要求 | 说明 |
|------|------|----------|------|
| GET | `/v1/mailboxes/:id/messages/:messageId/attachments/:attachmentId` | 邮箱Token | 下载附件 |

### 别名管理（Aliases）

| 方法 | 端点 | 认证要求 | 说明 |
|------|------|----------|------|
| POST | `/v1/mailboxes/:id/aliases` | 邮箱Token | 创建别名 |
| GET | `/v1/mailboxes/:id/aliases` | 邮箱Token | 获取别名列表 |
| GET | `/v1/mailboxes/:id/aliases/:aliasId` | 邮箱Token | 获取别名详情 |
| DELETE | `/v1/mailboxes/:id/aliases/:aliasId` | 邮箱Token | 删除别名 |
| PATCH | `/v1/mailboxes/:id/aliases/:aliasId` | 邮箱Token | 切换别名状态 |

### 邮件标签（Tags）

| 方法 | 端点 | 认证要求 | 说明 |
|------|------|----------|------|
| POST | `/v1/tags` | JWT | 创建标签 |
| GET | `/v1/tags` | JWT | 获取标签列表 |
| GET | `/v1/tags/:id` | JWT | 获取标签详情 |
| PATCH | `/v1/tags/:id` | JWT | 更新标签 |
| DELETE | `/v1/tags/:id` | JWT | 删除标签 |
| GET | `/v1/tags/:id/messages` | JWT | 获取标签下的邮件 |
| POST | `/v1/mailboxes/:id/messages/:messageId/tags` | 邮箱Token + JWT | 🔧 为邮件添加标签 |
| GET | `/v1/mailboxes/:id/messages/:messageId/tags` | 邮箱Token + JWT | 🔧 获取邮件标签 |
| DELETE | `/v1/mailboxes/:id/messages/:messageId/tags/:tagId` | 邮箱Token + JWT | 🔧 移除邮件标签 |

### 管理员功能（Admin）

| 方法 | 端点 | 认证要求 | 说明 |
|------|------|----------|------|
| POST | `/v1/debug/admin/create` | 无 | 🆕 创建管理员用户(开发) |
| GET | `/v1/admin/users` | JWT + 管理员 | 获取用户列表 |
| GET | `/v1/admin/users/:id` | JWT + 管理员 | 获取用户详情 |
| PATCH | `/v1/admin/users/:id` | JWT + 管理员 | 更新用户信息 |
| DELETE | `/v1/admin/users/:id` | JWT + 超级管理员 | 删除用户 |
| GET | `/v1/admin/users/:id/quota` | JWT + 管理员 | 获取用户配额 |
| PUT | `/v1/admin/users/:id/quota` | JWT + 管理员 | 更新用户配额 |
| GET | `/v1/admin/statistics` | JWT + 管理员 | 获取系统统计 |

### 系统域名管理（System Domains）

| 方法 | 端点 | 认证要求 | 说明 |
|------|------|----------|------|
| GET | `/v1/admin/domains` | JWT + 管理员 | 获取系统域名列表 |
| POST | `/v1/admin/domains` | JWT + 超级管理员 | 添加系统域名 |
| GET | `/v1/admin/domains/:id` | JWT + 管理员 | 获取域名详情 |
| POST | `/v1/admin/domains/:id/verify` | JWT + 管理员 | 验证域名 |
| PATCH | `/v1/admin/domains/:id/toggle` | JWT + 管理员 | 切换域名状态 |
| POST | `/v1/admin/domains/:id/set-default` | JWT + 超级管理员 | 设置默认域名 |
| DELETE | `/v1/admin/domains/:id` | JWT + 超级管理员 | 删除域名 |

### 用户域名管理（User Domains）

| 方法 | 端点 | 认证要求 | 说明 |
|------|------|----------|------|
| POST | `/v1/user/domains` | JWT | 添加域名 |
| GET | `/v1/user/domains` | JWT | 获取域名列表 |
| GET | `/v1/user/domains/:id` | JWT | 获取域名详情 |
| POST | `/v1/user/domains/:id/verify` | JWT | 验证域名 |
| PATCH | `/v1/user/domains/:id` | JWT | 更新域名模式 |
| DELETE | `/v1/user/domains/:id` | JWT | 删除域名 |

### API Key管理

| 方法 | 端点 | 认证要求 | 说明 |
|------|------|----------|------|
| POST | `/v1/api-keys` | JWT | 创建API Key |
| GET | `/v1/api-keys` | JWT | 获取API Keys列表 |
| GET | `/v1/api-keys/:id` | JWT | 获取API Key详情 |
| DELETE | `/v1/api-keys/:id` | JWT | 删除API Key |

### Webhook管理

| 方法 | 端点 | 认证要求 | 说明 |
|------|------|----------|------|
| POST | `/v1/webhooks` | JWT | 创建Webhook |
| GET | `/v1/webhooks` | JWT | 获取Webhooks列表 |
| GET | `/v1/webhooks/:id` | JWT | 获取Webhook详情 |
| PATCH | `/v1/webhooks/:id` | JWT | 更新Webhook |
| DELETE | `/v1/webhooks/:id` | JWT | 删除Webhook |
| GET | `/v1/webhooks/:id/deliveries` | JWT | 获取投递记录 |

### WebSocket

| 方法 | 端点 | 认证要求 | 说明 |
|------|------|----------|------|
| GET | `/v1/ws` | 无 | WebSocket连接(实时通知) |

### 健康检查（Health）

| 方法 | 端点 | 认证要求 | 说明 |
|------|------|----------|------|
| GET | `/health` | 无 | 基础健康检查 |
| GET | `/health/live` | 无 | 🆕 存活检查 |
| GET | `/health/ready` | 无 | 🆕 就绪检查 |
| GET | `/metrics` | 无 | Prometheus指标 |

### 公开API（Public）

| 方法 | 端点 | 认证要求 | 说明 |
|------|------|----------|------|
| GET | `/v1/public/domains` | 无 | 获取可用域名列表 |
| GET | `/v1/public/config` | 无 | 获取系统配置 |

### 系统配置管理

| 方法 | 端点 | 认证要求 | 说明 |
|------|------|----------|------|
| GET | `/v1/admin/config` | JWT + 管理员 | 获取系统配置 |
| PUT | `/v1/admin/config` | JWT + 超级管理员 | 更新系统配置 |
| POST | `/v1/admin/config/reset` | JWT + 超级管理员 | 重置系统配置 |

### Debug端点（开发环境）

| 方法 | 端点 | 认证要求 | 说明 |
|------|------|----------|------|
| GET | `/v1/debug/config` | 无 | 获取系统配置 |
| PUT | `/v1/debug/config` | 无 | 更新系统配置 |
| POST | `/v1/debug/config/reset` | 无 | 重置系统配置 |
| GET | `/v1/debug/domains` | 无 | 获取系统域名列表 |
| POST | `/v1/debug/domains` | 无 | 添加系统域名 |
| POST | `/v1/debug/admin/create` | 无 | 🆕 创建管理员用户 |

**图例**: 🆕 新增 | 🔧 修复

---

## 认证方式说明

### JWT认证（用户身份）

**使用场景**: 需要用户身份的操作

**传递方式**:
```http
Authorization: Bearer <JWT-access-token>
```

**获取方式**: 
- 通过 `/v1/auth/login` 登录获取
- 通过 `/v1/auth/register` 注册获取

**令牌类型**:
- **访问令牌（Access Token）**: 有效期15分钟，用于API请求
- **刷新令牌（Refresh Token）**: 有效期7天，用于更新访问令牌

### 邮箱Token认证（邮箱访问）

**使用场景**: 访问特定邮箱及其邮件

**传递方式**（按优先级）:
1. `X-Mailbox-Token: <mailbox-token>` （推荐）
2. `Authorization: Bearer <mailbox-token>` （备选）
3. `?token=<mailbox-token>` （URL参数）

**获取方式**: 
- 创建邮箱时返回的 `token` 字段
- 每个邮箱有独立的访问令牌

### API Key认证（兼容API）

**使用场景**: 兼容旧版API或第三方集成

**传递方式**: 通过API Key中间件

**使用端点**: `/api/*` 路由

---

## 使用示例

### 创建邮箱并获取令牌

**请求**:
```bash
curl -X POST http://localhost:8080/v1/mailboxes \
  -H "Content-Type: application/json" \
  -d '{
    "prefix": "test",
    "domain": "temp.mail",
    "expiresIn": "1h"
  }'
```

**响应**:
```json
{
  "code": 201,
  "msg": "创建成功",
  "data": {
    "id": "c6030b00-80a6-402e-a2e9-1ee9e7e37d7b",
    "address": "test@temp.mail",
    "localPart": "test",
    "domain": "temp.mail",
    "token": "snFaBcyCsS4uaoZjdtQYZ8ohrAOtORkM",
    "createdAt": "2025-10-16T08:22:36.523Z",
    "expiresAt": "2025-10-16T09:22:36.523Z",
    "unread": 0,
    "total": 0
  }
}
```

### 使用邮箱令牌获取邮件

**PowerShell**:
```powershell
Invoke-WebRequest `
  -Uri "http://localhost:8080/v1/mailboxes/c6030b00-80a6-402e-a2e9-1ee9e7e37d7b/messages" `
  -Method GET `
  -Headers @{"X-Mailbox-Token"="snFaBcyCsS4uaoZjdtQYZ8ohrAOtORkM"}
```

**cURL**:
```bash
curl -X GET \
  "http://localhost:8080/v1/mailboxes/c6030b00-80a6-402e-a2e9-1ee9e7e37d7b/messages" \
  -H "X-Mailbox-Token: snFaBcyCsS4uaoZjdtQYZ8ohrAOtORkM"
```

### 清空邮箱所有邮件

**PowerShell**:
```powershell
Invoke-WebRequest `
  -Uri "http://localhost:8080/v1/mailboxes/c6030b00-80a6-402e-a2e9-1ee9e7e37d7b/messages" `
  -Method DELETE `
  -Headers @{"X-Mailbox-Token"="snFaBcyCsS4uaoZjdtQYZ8ohrAOtORkM"}
```

**响应**:
```json
{
  "code": 200,
  "msg": "成功",
  "data": {
    "message": "邮件清空成功",
    "deleted": 15
  }
}
```

### 创建管理员用户

**请求**:
```bash
curl -X POST http://localhost:8080/v1/debug/admin/create \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@example.com",
    "password": "securepassword123",
    "username": "admin",
    "role": "super"
  }'
```

**响应**:
```json
{
  "code": 201,
  "msg": "创建成功",
  "data": {
    "id": "cf95d3b6-d47b-4b12-a8a8-feee6e3d62b0",
    "email": "admin@example.com",
    "username": "admin",
    "role": "super",
    "tier": "free",
    "isActive": true,
    "isEmailVerified": true,
    "createdAt": "2025-10-16T08:24:35.915Z"
  }
}
```

### WebSocket 连接

**JavaScript**:
```javascript
const ws = new WebSocket('ws://localhost:8080/v1/ws');

ws.onopen = () => {
  console.log('WebSocket 已连接');
  
  // 订阅邮箱通知
  ws.send(JSON.stringify({
    type: 'subscribe',
    mailboxId: 'c6030b00-80a6-402e-a2e9-1ee9e7e37d7b'
  }));
};

ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log('收到新邮件:', data);
};
```

---

## 注意事项

### 邮箱Token vs JWT

- **邮箱Token**: 用于访问特定邮箱（无需用户登录）
  - 每个邮箱独立的访问凭证
  - 仅能访问该邮箱及其邮件
  - 游客模式下可使用

- **JWT**: 用于用户身份认证（跨邮箱操作）
  - 用户级别的身份凭证
  - 可访问用户拥有的所有邮箱
  - 需要注册/登录获取

### 标签功能需要双重认证

邮件标签相关端点需要同时提供：
1. **邮箱Token** - 验证邮箱访问权限
2. **JWT** - 验证标签所有权

**示例**:
```powershell
Invoke-WebRequest `
  -Uri "http://localhost:8080/v1/mailboxes/{id}/messages/{messageId}/tags" `
  -Method POST `
  -Headers @{
    "X-Mailbox-Token" = "邮箱Token"
    "Authorization" = "Bearer JWT访问令牌"
    "Content-Type" = "application/json"
  } `
  -Body '{"tagId":"标签ID"}'
```

### 开发环境Debug端点

- `/v1/debug/*` 端点仅用于开发测试
- 生产环境应禁用或限制访问
- 不进行权限验证，便于快速测试

### 健康检查端点

- `/health` - 快速检查，响应最快
- `/health/live` - K8s liveness probe，检查服务是否存活
- `/health/ready` - K8s readiness probe，检查服务是否就绪

**Kubernetes 配置示例**:
```yaml
livenessProbe:
  httpGet:
    path: /health/live
    port: 8080
  initialDelaySeconds: 30
  periodSeconds: 10

readinessProbe:
  httpGet:
    path: /health/ready
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 5
```

### 默认域名配置

系统默认允许的域名为 `temp.mail`。要使用其他域名：

1. 通过环境变量配置：
   ```bash
   TEMPMAIL_MAILBOX_ALLOWED_DOMAINS=temp.mail,tempmail.dev
   ```

2. 或通过系统域名管理API添加

---

## 待完成功能

### 前端相关功能

这些功能主要需要前端实现：

1. **30秒智能刷新机制**
   - 实现前端自动轮询
   - 显示邮件到达的温和提示

2. **WebSocket实时通知完善**
   - 实现前端WebSocket客户端
   - 处理连接断开重连逻辑

3. **邮件解析和编码优化**
   - 前端JSON解码处理
   - 修复中文显示问题

### 部署相关

**HTTPS支持**:
- 不在后端直接实现TLS/SSL
- 推荐使用反向代理：
  - Nginx
  - Caddy
  - Traefik
  - Cloudflare

**反向代理配置示例（Nginx）**:
```nginx
server {
    listen 443 ssl http2;
    server_name temp.mail;
    
    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;
    
    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
    
    # WebSocket 支持
    location /v1/ws {
        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }
}
```

---

**文档版本**: v0.9.0  
**最后更新**: 2025-10-16  
**维护者**: Backend Team

---

## 相关文档

- [CLAUDE.md](../CLAUDE.md) - Claude Code 工作指南
- [README.md](../README.md) - 项目主文档
- [backend/README.md](../backend/README.md) - 后端开发文档
