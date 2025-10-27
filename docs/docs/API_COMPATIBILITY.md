# API 兼容层说明

本系统提供了 `mail.ry.edu.kg` 风格的API接口，让其他系统可以无需修改代码直接对接使用。

## ⚠️ 重要说明（v0.8.2 更新）

**兼容 API 完全保持旧格式！**

从 v0.8.2 开始，**兼容 API (`/api/*`) 全部保持旧格式**（直接返回数据），与 mail.ry.edu.kg 完全一致。

### 格式对比

| API类型 | 端点 | 响应格式 | 说明 |
|---------|------|---------|------|
| 主API | `/v1/*` | 新格式 `{code, msg, data}` | v0.8.0+ |
| 兼容API | `/api/*` 所有端点 | 旧格式（直接返回数据） | 与 mail.ry.edu.kg 一致 |

### 兼容API响应格式（所有端点）

**所有兼容API端点都使用旧格式，直接返回数据：**

- `/api/config` - 获取配置
- `/api/emails/generate` - 生成邮箱
- `/api/emails` - 邮箱列表
- `/api/emails/{emailId}` - 邮件列表
- `/api/emails/{emailId}/{messageId}` - 邮件详情

**旧格式示例：**

```json
{
  "emailId": "xxx",
  "email": "test@temp.mail",
  "name": "test",
  "domain": "temp.mail",
  "createdAt": "2025-10-14T12:00:00Z",
  "expiresAt": "2025-10-14T13:00:00Z",
  "token": "xxx"
}
```

**错误响应示例：**

```json
{
  "error": "invalid request"
}
```

**详见**：[API响应格式说明](./API_RESPONSE_FORMAT.md)

## 使用场景

如果你的项目原本使用 `mail.ry.edu.kg` 的临时邮箱服务，现在想切换到我们的系统，只需要：

1. 将API域名改为你的服务器域名
2. 将API Key改为从我们系统获取的Key
3. 其他代码完全不用动！

## API 端点

所有兼容API端点都以 `/api` 开头，并需要在请求头中包含 `X-API-Key`。

### 1. 获取系统配置

```bash
curl https://your-domain.com/api/config \
  -H "X-API-Key: YOUR_API_KEY"
```

**响应示例：**
```json
{
  "domains": ["temp.mail", "tempmail.dev"]
}
```

### 2. 生成临时邮箱

```bash
curl -X POST https://your-domain.com/api/emails/generate \
  -H "X-API-Key: YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test",
    "expiryTime": 3600000,
    "domain": "temp.mail"
  }'
```

**请求参数：**
- `name` (string): 邮箱前缀
- `expiryTime` (number): 过期时间（毫秒）
  - `3600000` - 1小时
  - `86400000` - 1天
  - `604800000` - 7天
  - `0` - 永久（默认100年）
- `domain` (string): 邮箱域名（从 `/api/config` 获取可用域名）

**响应示例：**
```json
{
  "emailId": "uuid-xxx",
  "email": "test@temp.mail",
  "name": "test",
  "domain": "temp.mail",
  "createdAt": "2025-10-14T12:00:00Z",
  "expiresAt": "2025-10-14T13:00:00Z",
  "token": "mailbox-token-xxx"
}
```

### 3. 获取邮箱列表

```bash
curl https://your-domain.com/api/emails?cursor=CURSOR&limit=20 \
  -H "X-API-Key: YOUR_API_KEY"
```

**查询参数：**
- `cursor` (string, 可选): 分页游标
- `limit` (number, 可选): 每页数量，默认20，最大100

**响应示例：**
```json
{
  "emails": [
    {
      "emailId": "uuid-xxx",
      "email": "test@temp.mail",
      "name": "test",
      "domain": "temp.mail",
      "createdAt": "2025-10-14T12:00:00Z",
      "expiresAt": "2025-10-14T13:00:00Z"
    }
  ],
  "nextCursor": "20"
}
```

### 4. 获取邮件列表

```bash
curl https://your-domain.com/api/emails/{emailId}?cursor=CURSOR&limit=20 \
  -H "X-API-Key: YOUR_API_KEY"
```

**路径参数：**
- `emailId` (string): 邮箱ID

**查询参数：**
- `cursor` (string, 可选): 分页游标
- `limit` (number, 可选): 每页数量，默认20，最大100

**响应示例：**
```json
{
  "messages": [
    {
      "messageId": "msg-xxx",
      "from": "sender@example.com",
      "subject": "Welcome",
      "createdAt": "2025-10-14T12:05:00Z",
      "isRead": false,
      "hasAttachment": false
    }
  ],
  "nextCursor": "20"
}
```

### 5. 获取单封邮件

```bash
curl https://your-domain.com/api/emails/{emailId}/{messageId} \
  -H "X-API-Key: YOUR_API_KEY"
```

**路径参数：**
- `emailId` (string): 邮箱ID
- `messageId` (string): 邮件ID

**响应示例：**
```json
{
  "messageId": "msg-xxx",
  "emailId": "uuid-xxx",
  "from": "sender@example.com",
  "to": "test@temp.mail",
  "subject": "Welcome",
  "text": "Plain text content",
  "html": "<p>HTML content</p>",
  "createdAt": "2025-10-14T12:05:00Z",
  "isRead": true,
  "attachments": [
    {
      "attachmentId": "att-xxx",
      "filename": "document.pdf",
      "contentType": "application/pdf",
      "size": 12345
    }
  ]
}
```

## API Key 管理

### 创建 API Key

目前API Key需要直接在数据库中创建。在内存存储模式下，可以通过以下方式添加：

```go
// 创建用户
user := &domain.User{
    ID:       "user-123",
    Email:    "user@example.com",
    Username: "testuser",
    IsActive: true,
}
store.CreateUser(user)

// 创建API Key
apiKey := &domain.APIKey{
    ID:        "key-123",
    UserID:    "user-123",
    Key:       "your-api-key-here",  // 建议使用UUID或随机字符串
    Name:      "My API Key",
    IsActive:  true,
    CreatedAt: time.Now(),
}
store.SaveAPIKey(apiKey)
```

### 后续计划

计划在后续版本中添加：
1. 用户管理界面中的API Key管理功能
2. API Key自动生成和轮换
3. API Key权限控制
4. API Key使用统计

## 注意事项

1. **认证要求**：所有 `/api` 端点都需要 `X-API-Key` 请求头
2. **分页支持**：使用 `cursor` 和 `limit` 参数进行分页
3. **自动标记已读**：获取单封邮件时会自动标记为已读
4. **过期时间**：设置为 `0` 时表示永久有效（实际为100年）
5. **域名验证**：邮箱域名必须在系统配置的允许列表中

## 错误响应

所有错误都返回以下格式：

```json
{
  "error": "error message"
}
```

常见错误码：
- `400` - 请求参数错误
- `401` - 未授权（API Key无效或缺失）
- `404` - 资源不存在
- `500` - 服务器内部错误

## 兼容性说明

本系统提供了与 `mail.ry.edu.kg` 相同格式的API接口，让第三方系统可以快速对接。

### 迁移指南

如果你正在使用 `mail.ry.edu.kg`，迁移到我们的系统非常简单：

**原来的代码**:
```bash
curl https://mail.ry.edu.kg/api/config \
  -H "X-API-Key: their-api-key"
```

**迁移后的代码**:
```bash
curl https://your-domain.com/api/config \
  -H "X-API-Key: your-api-key"
```

仅需修改两处：
1. **域名**: `mail.ry.edu.kg` → `your-domain.com`
2. **API Key**: 使用从我们系统获取的Key

其他参数、请求格式、响应格式完全一致！

## 示例：第三方系统对接流程

假设你的第三方系统需要临时邮箱功能，可以通过以下步骤对接：

```bash
# 1. 获取可用的邮箱域名
CONFIG=$(curl -s https://your-domain.com/api/config \
  -H "X-API-Key: YOUR_API_KEY")
echo "Available domains: $CONFIG"

# 2. 生成临时邮箱
EMAIL=$(curl -s -X POST https://your-domain.com/api/emails/generate \
  -H "X-API-Key: YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "mytest",
    "expiryTime": 3600000,
    "domain": "temp.mail"
  }')
echo "Created email: $EMAIL"

# 提取emailId (使用jq)
EMAIL_ID=$(echo $EMAIL | jq -r '.emailId')

# 3. 检查邮件
MESSAGES=$(curl -s "https://your-domain.com/api/emails/$EMAIL_ID" \
  -H "X-API-Key: YOUR_API_KEY")
echo "Messages: $MESSAGES"

# 4. 读取单封邮件
MESSAGE_ID=$(echo $MESSAGES | jq -r '.messages[0].messageId')
MESSAGE=$(curl -s "https://your-domain.com/api/emails/$EMAIL_ID/$MESSAGE_ID" \
  -H "X-API-Key: YOUR_API_KEY")
echo "Message content: $MESSAGE"
```

## 技术实现

本系统提供的兼容API采用以下设计：

- **路由前缀**：`/api`（与标准v1 API的 `/v1` 区分）
- **认证方式**：API Key通过 `X-API-Key` 请求头
- **数据转换**：自动转换内部数据模型到兼容格式
- **分页实现**：基于偏移量的简单分页（cursor为数字偏移）

### 优势

1. **无缝对接** - 第三方系统无需修改业务逻辑代码
2. **标准格式** - 遵循常见的RESTful API设计
3. **独立部署** - 可以独立部署和扩展
4. **易于维护** - 与内部API分离，互不影响

## 对接优势

选择我们的系统对接，你可以获得：

- ✅ **开源免费** - 完全开源，可自主部署
- ✅ **数据私有** - 数据存储在你自己的服务器
- ✅ **可定制化** - 可根据需求修改和扩展
- ✅ **无需担心服务商跑路** - 完全掌控
- ✅ **高性能** - Go语言实现，响应快速
- ✅ **生产就绪** - 完整的监控、日志、健康检查

## 更新日志

### v0.8.2-beta (2025-10-14)
- ✅ 实现 mail.ry.edu.kg 风格的API接口
- ✅ 支持所有5个核心端点
- ✅ API Key认证机制
- ✅ 分页支持
- ✅ 完整的文档和示例
