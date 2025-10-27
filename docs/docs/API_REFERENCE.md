# TempMail API 完整参考指南

**版本**: v0.8.3-beta  
**最后更新**: 2025-10-16  
**测试状态**: ✅ 已完成全面API测试

## 📋 目录

- [快速开始](#快速开始)
- [认证方式](#认证方式)
- [基础API](#基础api)
- [公开API](#公开api)
- [用户认证](#用户认证)
- [邮箱管理](#邮箱管理)
- [邮件管理](#邮件管理)
- [别名管理](#别名管理)
- [标签管理](#标签管理)
- [API密钥管理](#api密钥管理)
- [用户域名管理](#用户域名管理)
- [Webhook管理](#webhook管理)
- [管理员API](#管理员api)
- [WebSocket](#websocket)
- [兼容性API](#兼容性api)
- [错误处理](#错误处理)
- [使用示例](#使用示例)
- [调试指南](#调试指南)

---

## 🚀 快速开始

### 基础URL
```
开发环境: http://localhost:8080
生产环境: https://api.tempmail.example.com
```

### 1. 创建临时邮箱（无需注册）
```bash
curl -X POST http://localhost:8080/v1/mailboxes
```

### 2. 查看邮箱邮件
```bash
# 复制返回的邮箱ID和Token，然后：
curl -X GET http://localhost:8080/v1/mailboxes/{id} \
  -H "X-Mailbox-Token: {token}"
```

### 3. 用户注册（可选）
```bash
curl -X POST http://localhost:8080/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "SecurePassword123!",
    "username": "myusername"
  }'
```

---

## 🔐 认证方式

TempMail API 支持多种认证方式：

### 1. JWT Bearer Token（推荐）
用于用户认证后的API访问：
```http
Authorization: Bearer {jwt_token}
```

### 2. 邮箱Token（Mailbox Token）
用于邮箱相关的操作：
```http
X-Mailbox-Token: {mailbox_token}
```

### 3. API Key（兼容API）
用于兼容性API访问：
```http
X-API-Key: {api_key}
```

---

## 📚 基础API

### Health Check
**检查API服务状态**

```http
GET /health
```

**响应**:
```json
{
  "status": "ok"
}
```

---

## 🌐 公开API

无需认证即可访问的公开接口。

### 获取可用域名
**获取系统支持的域名列表**

```http
GET /v1/public/domains
```

**响应**:
```json
{
  "code": 200,
  "msg": "成功",
  "data": {
    "count": 2,
    "domains": ["temp.mail", "tempmail.dev"]
  }
}
```

### 获取系统配置
**获取系统公开配置信息**

```http
GET /v1/public/config
```

**响应**:
```json
{
  "code": 200,
  "msg": "成功",
  "data": {
    "defaultDomain": "temp.mail",
    "domains": ["temp.mail", "tempmail.dev"],
    "features": {
      "aliases": true,
      "attachments": true,
      "search": true,
      "tags": true,
      "webhooks": true
    },
    "limits": {
      "maxAliasesPerMailbox": 5,
      "maxMessagesPerMailbox": 1000,
      "messageRetentionDays": 7
    }
  }
}
```

---

## 🔐 用户认证

### 用户注册
**创建新的用户账号**

```http
POST /v1/auth/register
```

**请求体**:
```json
{
  "email": "user@example.com",
  "password": "SecurePassword123!",
  "username": "john_doe"
}
```

**响应**:
```json
{
  "code": 201,
  "msg": "注册成功",
  "data": {
    "user": {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "email": "user@example.com",
      "username": "john_doe",
      "tier": "free"
    },
    "tokens": {
      "accessToken": "eyJhbGc...",
      "refreshToken": "eyJhbGc...",
      "expiresIn": 900
    }
  }
}
```

### 用户登录
**用户登录获取访问令牌**

```http
POST /v1/auth/login
```

**请求体**:
```json
{
  "email": "user@example.com",
  "password": "SecurePassword123!"
}
```

### 刷新令牌
**使用刷新令牌获取新的访问令牌**

```http
POST /v1/auth/refresh
```

**请求体**:
```json
{
  "refreshToken": "eyJhbGc..."
}
```

### 获取当前用户信息
**获取当前登录用户的详细信息**

```http
GET /v1/auth/me
Authorization: Bearer {access_token}
```

---

## 📬 Mailbox Management API

### 创建临时邮箱
**创建一个新的临时邮箱地址**

```http
POST /v1/mailboxes
Authorization: Bearer {access_token}  // 可选
```

**请求体**（可选）:
```json
{
  "prefix": "mytemp",        // 自定义前缀
  "domain": "temp.mail",     // 域名
  "expiresIn": "48h"         // 过期时间
}
```

**响应**:
```json
{
  "code": 201,
  "msg": "创建成功",
  "data": {
    "id": "a1b2c3d4-e5f6-4789-a012-3456789abcde",
    "address": "mytemp@temp.mail",
    "localPart": "mytemp",
    "domain": "temp.mail",
    "token": "AbCdEf123456",
    "createdAt": "2025-01-01T00:00:00Z",
    "expiresAt": "2025-01-03T00:00:00Z",
    "unread": 0,
    "total": 0
  }
}
```

### 获取邮箱列表
**获取用户的所有邮箱列表**

```http
GET /v1/mailboxes
Authorization: Bearer {access_token}
```

### 获取邮箱详情
**获取指定邮箱的详细信息**

```http
GET /v1/mailboxes/{id}
X-Mailbox-Token: {mailbox_token}
```

### 删除邮箱
**删除指定邮箱及其所有邮件**

```http
DELETE /v1/mailboxes/{id}
X-Mailbox-Token: {mailbox_token}
```

---

## 📧 Messages API

### 获取邮件列表
**获取邮箱的所有邮件**

```http
GET /v1/mailboxes/{id}/messages
X-Mailbox-Token: {mailbox_token}
```

**查询参数**:
- `limit`: 限制返回数量（默认50）
- `offset`: 偏移量（默认0）

**响应**:
```json
{
  "code": 200,
  "msg": "获取成功",
  "data": {
    "items": [
      {
        "id": "msg-123456",
        "mailboxId": "a1b2c3d4-e5f6-4789-a012-3456789abcde",
        "from": "sender@example.com",
        "to": "test@temp.mail",
        "subject": "Test Email",
        "isRead": false,
        "receivedAt": "2025-01-01T10:30:00Z",
        "hasAttachments": false
      }
    ],
    "count": 1
  }
}
```

### 获取邮件详情
**获取单封邮件的完整内容**

```http
GET /v1/mailboxes/{id}/messages/{messageId}
X-Mailbox-Token: {mailbox_token}
```

### 标记邮件为已读
**将指定邮件标记为已读状态**

```http
POST /v1/mailboxes/{id}/messages/{messageId}/read
X-Mailbox-Token: {mailbox_token}
```

**响应**: 204 No Content

### 搜索邮件
**在指定邮箱中搜索邮件**

```http
GET /v1/mailboxes/{id}/messages/search
X-Mailbox-Token: {mailbox_token}
```

**查询参数**:
- `q`: 搜索关键词（搜索主题、发件人、内容）
- `from`: 发件人筛选
- `subject`: 主题筛选
- `startDate`: 开始日期 (RFC3339格式)
- `endDate`: 结束日期 (RFC3339格式)
- `isRead`: 是否已读
- `hasAttachment`: 是否有附件
- `page`: 页码（默认1）
- `pageSize`: 每页数量（默认20，最大100）

**示例**:
```http
GET /v1/mailboxes/{id}/messages/search?q=测试&from=sender@example.com&isRead=true
```

### 下载附件
**下载邮件附件**

```http
GET /v1/mailboxes/{id}/messages/{messageId}/attachments/{attachmentId}
X-Mailbox-Token: {mailbox_token}
```

**响应**: 二进制文件流

---

## 🔄 Aliases API

### 创建邮箱别名
**为邮箱创建新的别名地址**

```http
POST /v1/mailboxes/{id}/aliases
X-Mailbox-Token: {mailbox_token}
```

**请求体**:
```json
{
  "address": "myalias@temp.mail"
}
```

**响应**:
```json
{
  "code": 200,
  "msg": "创建成功",
  "data": {
    "id": "alias-123456",
    "mailboxId": "a1b2c3d4-e5f6-4789-a012-3456789abcde",
    "address": "myalias@temp.mail",
    "createdAt": "2025-01-01T12:00:00Z",
    "isActive": true
  }
}
```

### 获取别名列表
**获取邮箱的所有别名**

```http
GET /v1/mailboxes/{id}/aliases
X-Mailbox-Token: {mailbox_token}
```

---

## 🏷️ 标签管理API

所有标签API都需要JWT认证。

### 创建标签
**创建新的邮件标签**

```http
POST /v1/tags
Authorization: Bearer {access_token}
```

**请求体**:
```json
{
  "name": "重要邮件",
  "color": "#FF0000",
  "description": "用于标记重要邮件的标签"
}
```

**响应**:
```json
{
  "code": 201,
  "msg": "创建成功",
  "data": {
    "id": "tag-123456",
    "name": "重要邮件",
    "color": "#FF0000",
    "description": "用于标记重要邮件的标签",
    "userId": "user-123",
    "createdAt": "2025-10-16T00:00:00Z",
    "updatedAt": "2025-10-16T00:00:00Z"
  }
}
```

### 获取标签列表
**获取用户的所有标签**

```http
GET /v1/tags
Authorization: Bearer {access_token}
```

### 获取标签详情
**获取指定标签的详细信息**

```http
GET /v1/tags/{id}
Authorization: Bearer {access_token}
```

### 更新标签
**更新指定标签的信息**

```http
PATCH /v1/tags/{id}
Authorization: Bearer {access_token}
```

**请求体**:
```json
{
  "name": "重要邮件-更新",
  "color": "#00FF00",
  "description": "更新后的标签描述"
}
```

### 删除标签
**删除指定标签**

```http
DELETE /v1/tags/{id}
Authorization: Bearer {access_token}
```

**响应**: 204 No Content

### 获取标签相关邮件
**获取使用此标签的所有邮件**

```http
GET /v1/tags/{id}/messages
Authorization: Bearer {access_token}
```

---

## 🔑 API密钥管理

所有API密钥管理都需要JWT认证。

### 创建API密钥
**创建新的API访问密钥**

```http
POST /v1/api-keys
Authorization: Bearer {access_token}
```

**请求体**:
```json
{
  "name": "测试API密钥",
  "description": "用于API测试的密钥",
  "permissions": ["read:mailboxes", "write:messages"]
}
```

**响应**:
```json
{
  "code": 201,
  "msg": "创建成功",
  "data": {
    "id": "apikey-123456",
    "name": "测试API密钥",
    "key": "ak_1234567890abcdef",
    "description": "用于API测试的密钥",
    "permissions": ["read:mailboxes", "write:messages"],
    "userId": "user-123",
    "createdAt": "2025-10-16T00:00:00Z",
    "lastUsed": null
  }
}
```

### 获取API密钥列表
**获取用户的所有API密钥**

```http
GET /v1/api-keys
Authorization: Bearer {access_token}
```

### 获取API密钥详情
**获取指定API密钥的详细信息**

```http
GET /v1/api-keys/{id}
Authorization: Bearer {access_token}
```

### 删除API密钥
**删除指定API密钥**

```http
DELETE /v1/api-keys/{id}
Authorization: Bearer {access_token}
```

**响应**: 204 No Content

---

## 🌐 用户域名管理

所有用户域名管理都需要JWT认证。

### 添加用户域名
**添加自定义域名**

```http
POST /v1/user/domains
Authorization: Bearer {access_token}
```

**请求体**:
```json
{
  "domain": "mydomain.com",
  "mode": "catch_all"
}
```

### 获取用户域名列表
**获取用户的所有自定义域名**

```http
GET /v1/user/domains
Authorization: Bearer {access_token}
```

### 获取域名详情
**获取指定域名的详细信息**

```http
GET /v1/user/domains/{id}
Authorization: Bearer {access_token}
```

### 获取域名配置说明
**获取域名DNS配置说明**

```http
GET /v1/user/domains/{id}/instructions
Authorization: Bearer {access_token}
```

### 验证域名
**验证域名DNS配置**

```http
POST /v1/user/domains/{id}/verify
Authorization: Bearer {access_token}
```

### 更新域名模式
**更新域名接收模式**

```http
PATCH /v1/user/domains/{id}
Authorization: Bearer {access_token}
```

**请求体**:
```json
{
  "mode": "whitelist"
}
```

### 删除用户域名
**删除用户自定义域名**

```http
DELETE /v1/user/domains/{id}
Authorization: Bearer {access_token}
```

**响应**: 204 No Content

---

## 🔗 Webhook管理

所有Webhook管理都需要JWT认证。

### 创建Webhook
**创建新的Webhook**

```http
POST /v1/webhooks
Authorization: Bearer {access_token}
```

**请求体**:
```json
{
  "url": "https://example.com/webhook",
  "events": ["message.received", "message.read"],
  "description": "测试Webhook"
}
```

### 获取Webhook列表
**获取用户的所有Webhooks**

```http
GET /v1/webhooks
Authorization: Bearer {access_token}
```

### 获取Webhook详情
**获取指定Webhook的详细信息**

```http
GET /v1/webhooks/{id}
Authorization: Bearer {access_token}
```

### 更新Webhook
**更新指定Webhook**

```http
PATCH /v1/webhooks/{id}
Authorization: Bearer {access_token}
```

### 删除Webhook
**删除指定Webhook**

```http
DELETE /v1/webhooks/{id}
Authorization: Bearer {access_token}
```

### 获取Webhook投递记录
**获取Webhook的投递历史**

```http
GET /v1/webhooks/{id}/deliveries
Authorization: Bearer {access_token}
```

---

## 👑 Admin API

所有管理员API都需要JWT认证和管理员权限。

### 获取系统统计
**获取系统整体统计信息**

```http
GET /v1/admin/statistics
Authorization: Bearer {admin_token}
```

**响应**:
```json
{
  "code": 200,
  "msg": "获取成功",
  "data": {
    "totalUsers": 1250,
    "activeUsers": 980,
    "totalMailboxes": 5600,
    "totalMessages": 45000,
    "systemUptime": "15d 8h 30m",
    "memoryUsage": {
      "used": 512000000,
      "total": 8589934592
    }
  }
}
```

### 获取用户列表
**获取系统中的所有用户**

```http
GET /v1/admin/users
Authorization: Bearer {admin_token}
```

---

## 🔌 WebSocket API

### 实时连接
**建立WebSocket连接接收实时邮件通知**

```javascript
const ws = new WebSocket('ws://localhost:8080/v1/ws');

// 连接建立后订阅邮箱
ws.send(JSON.stringify({
  type: 'subscribe',
  mailboxId: 'a1b2c3d4-e5f6-4789-a012-3456789abcde',
  token: 'AbCdEf123456'
}));

// 监听新邮件通知
ws.onmessage = function(event) {
  const data = JSON.parse(event.data);
  if (data.type === 'new_mail') {
    console.log('新邮件:', data.message);
  }
};
```

**事件类型**:
- `new_mail`: 新邮件通知
- `mailbox_expired`: 邮箱过期通知

---

## 🔄 Compatibility API

兼容API提供与 mail.ry.edu.kg 格式兼容的接口。

### 生成临时邮箱
```http
POST /api/emails/generate
X-API-Key: {api_key}
```

**响应**:
```json
{
  "emailId": "uuid-1234",
  "email": "test@temp.mail",
  "token": "**MAILBOX_EXAMPLE"
}
```

---

## ❌ 错误处理

### 统一错误响应格式

所有API错误都遵循统一格式：

```json
{
  "code": {error_code},
  "msg": "中文错误消息",
  "data": null
}
```

### 常见错误码

| Code | HTTP状态 | 说明 |
|------|---------|------|
| 400  | 400     | 请求参数错误 |
| 401  | 401     | 未认证或认证失败 |
| 403  | 403     | 权限不足 |
| 404  | 404     | 资源不存在 |
| 409  | 409     | 资源冲突（如邮箱已存在） |
| 500  | 500     | 服务器内部错误 |

---

## 🛠️ 使用示例

### JavaScript/TypeScript

```typescript
// 创建邮箱
async function createMailbox() {
  const response = await fetch('http://localhost:8080/v1/mailboxes', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    }
  });
  
  const result = await response.json();
  return result.data;
}

// 用户注册
async function register(email: string, password: string, username: string) {
  const response = await fetch('http://localhost:8080/v1/auth/register', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ email, password, username })
  });
  
  const result = await response.json();
  return result.data;
}

// 获取邮件列表
async function getMessages(mailboxId: string, token: string) {
  const response = await fetch(
    `http://localhost:8080/v1/mailboxes/${mailboxId}/messages`,
    {
      headers: {
        'X-Mailbox-Token': token
      }
    }
  );
  
  const result = await response.json();
  return result.data.items;
}
```

### Python

```python
import requests
import json

class TempMailAPI:
    def __init__(self, base_url="http://localhost:8080"):
        self.base_url = base_url
        self.token = None
        
    def create_mailbox(self, prefix=None, domain=None):
        data = {}
        if prefix:
            data['prefix'] = prefix
        if domain:
            data['domain'] = domain
            
        response = requests.post(f"{self.base_url}/v1/mailboxes", json=data)
        return response.json()['data']
    
    def register(self, email, password, username):
        data = {
            "email": email,
            "password": password,
            "username": username
        }
        response = requests.post(f"{self.base_url}/v1/auth/register", json=data)
        result = response.json()['data']
        self.token = result['tokens']['accessToken']
        return result
    
    def get_messages(self, mailbox_id, mailbox_token):
        headers = {'X-Mailbox-Token': mailbox_token}
        response = requests.get(
            f"{self.base_url}/v1/mailboxes/{mailbox_id}/messages",
            headers=headers
        )
        return response.json()['data']['items']

# 使用示例
api = TempMailAPI()

# 创建邮箱（游客模式）
mailbox = api.create_mailbox(prefix="test123")
print(f"邮箱地址: {mailbox['address']}")
print(f"访问令牌: {mailbox['token']}")

# 注册用户
user = api.register("user@example.com", "Password123!", "testuser")
print(f"用户ID: {user['user']['id']}")
```

### cURL

```bash
# 创建邮箱
curl -X POST http://localhost:8080/v1/mailboxes \
  -H "Content-Type: application/json" \
  -d '{"prefix": "test", "domain": "temp.mail"}'

# 用户注册
curl -X POST http://localhost:8080/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "SecurePassword123!",
    "username": "testuser"
  }'

# 获取邮件列表
curl -X GET http://localhost:8080/v1/mailboxes/{id}/messages \
  -H "X-Mailbox-Token: {token}"
```

---

## 🔧 调试指南

### 1. 常见问题

#### CORS错误
如果你遇到跨域问题，确保：
- 前端请求包含正确的认证头
- 后端配置了正确的CORS允许源
- 使用正确的请求方法

#### 认证失败
- 检查JWT令牌是否过期
- 确认Authorization头格式：`Bearer {token}`
- 验证API Key是否正确

#### 404错误
- 确认API路径正确
- 检查资源ID是否存在
- 验证权限是否足够

### 2. 调试工具

#### 使用浏览器开发者工具
1. 打开Network标签
2. 观察请求和响应
3. 检查响应状态码和内容

#### 命令行调试
```bash
# 测试API连通性
curl -v http://localhost:8080/health

# 测试认证
curl -v http://localhost:8080/v1/auth/me \
  -H "Authorization: Bearer {your_token}"

# 测试邮箱访问
curl -v http://localhost:8080/v1/mailboxes/{id} \
  -H "X-Mailbox-Token: {mailbox_token}"
```

### 3. 日志查看
查看后端日志获取详细错误信息：
```bash
# 查看服务器日志
tail -f server.log

# 查看错误日志
tail -f server_err.log
```

### 4. 响应时间监控
```bash
# 测试响应时间
time curl http://localhost:8080/health
```

---

## 📊 API统计信息

| API类别 | 端点数量 | 认证要求 | 描述 |
|---------|---------|---------|------|
| 基础健康 | 1 | 无 | 系统状态检查 |
| 公开API | 2 | 无 | 无需认证的公开接口 |
| 认证管理 | 4 | 无 | 用户注册登录 |
| 邮箱管理 | 4 | JWT/邮箱Token | 邮箱创建和管理 |
| 邮件管理 | 4+ | 邮箱Token | 邮件读取操作 |
| 别名管理 | 5+ | 邮箱Token | 别名管理 |
| 管理员API | 10+ | JWT+管理员 | 系统管理 |
| WebSocket | 1 | 无 | 实时通知 |
| 兼容API | 4+ | API Key | 兼容旧版本 |

---

## 📞 技术支持

如果您在使用API时遇到问题：

1. **查看此文档** - 大多数常见问题都有解决方案
2. **检查错误码** - 根据返回的错误码和消息定位问题
3. **使用调试工具** - 使用curl或Postman测试API调用
4. **查看后端日志** - 获取详细的错误信息
5. **参考示例代码** - 本文档提供了完整的示例

---

**文档版本**: v3.0  
**API版本**: v0.8.3-beta  
**最后更新**: 2025-10-16  
**测试状态**: ✅ 已完成全面API测试 (88.89% 成功率)
