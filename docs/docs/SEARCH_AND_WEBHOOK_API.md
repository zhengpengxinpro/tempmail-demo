# 邮件搜索和 Webhook API 文档

## 版本

**版本**: v0.9.0  
**更新日期**: 2024-01-15

---

## 一、邮件搜索功能

### 1.1 功能概述

邮件搜索功能允许用户在指定邮箱中搜索邮件，支持多种筛选条件：
- 关键词搜索（搜索主题、发件人、内容）
- 按发件人筛选
- 按主题筛选
- 按时间范围筛选
- 按已读状态筛选
- 按是否有附件筛选
- 分页支持

### 1.2 API 端点

#### 搜索邮件

```http
GET /v1/mailboxes/:id/messages/search
```

**请求头**:
```
X-Mailbox-Token: {邮箱访问令牌}
```

**查询参数**:

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| q | string | 否 | 搜索关键词（搜索主题、发件人、内容） |
| from | string | 否 | 发件人筛选 |
| subject | string | 否 | 主题筛选 |
| startDate | string | 否 | 开始日期（RFC3339格式） |
| endDate | string | 否 | 结束日期（RFC3339格式） |
| isRead | boolean | 否 | 是否已读 |
| hasAttachment | boolean | 否 | 是否有附件 |
| page | integer | 否 | 页码（默认1） |
| pageSize | integer | 否 | 每页数量（默认20，最大100） |

**响应示例**:
```json
{
  "success": true,
  "data": {
    "messages": [
      {
        "id": "msg_xxx",
        "mailboxId": "mb_xxx",
        "from": "sender@example.com",
        "to": ["recipient@temp.mail"],
        "subject": "Test Email",
        "text": "Email content...",
        "html": "<p>Email content...</p>",
        "isRead": false,
        "attachments": [],
        "createdAt": "2024-01-15T10:00:00Z",
        "receivedAt": "2024-01-15T10:00:00Z"
      }
    ],
    "total": 1,
    "page": 1,
    "pageSize": 20,
    "totalPages": 1
  }
}
```

### 1.3 使用示例

#### cURL 示例

```bash
# 搜索包含关键词 "verification" 的邮件
curl "http://localhost:8080/v1/mailboxes/mb_xxx/messages/search?q=verification" \
  -H "X-Mailbox-Token: token_xxx"

# 搜索特定发件人的邮件
curl "http://localhost:8080/v1/mailboxes/mb_xxx/messages/search?from=noreply@example.com" \
  -H "X-Mailbox-Token: token_xxx"

# 搜索有附件的未读邮件
curl "http://localhost:8080/v1/mailboxes/mb_xxx/messages/search?hasAttachment=true&isRead=false" \
  -H "X-Mailbox-Token: token_xxx"

# 按时间范围搜索
curl "http://localhost:8080/v1/mailboxes/mb_xxx/messages/search?startDate=2024-01-01T00:00:00Z&endDate=2024-01-15T23:59:59Z" \
  -H "X-Mailbox-Token: token_xxx"
```

#### JavaScript 示例

```javascript
// 搜索邮件
async function searchMessages(mailboxId, token, criteria) {
  const params = new URLSearchParams(criteria);
  
  const response = await fetch(
    `http://localhost:8080/v1/mailboxes/${mailboxId}/messages/search?${params}`,
    {
      headers: {
        'X-Mailbox-Token': token
      }
    }
  );
  
  return await response.json();
}

// 使用示例
const result = await searchMessages('mb_xxx', 'token_xxx', {
  q: 'verification',
  isRead: false,
  page: 1,
  pageSize: 20
});

console.log(`找到 ${result.data.total} 封邮件`);
```

---

## 二、Webhook 回调功能

### 2.1 功能概述

Webhook 回调功能允许开发者配置 HTTP 回调 URL，当特定事件发生时（如新邮件到达），系统会自动向该 URL 发送 HTTP POST 请求。

**特性**：
- 支持多个 Webhook 配置
- 订阅特定事件类型
- HMAC-SHA256 签名验证
- 自动重试机制（指数退避）
- 投递记录查询

### 2.2 支持的事件类型

| 事件类型 | 说明 |
|---------|------|
| `mail.received` | 新邮件到达 |
| `mail.read` | 邮件已读 |
| `mailbox.created` | 邮箱创建 |
| `mailbox.deleted` | 邮箱删除 |

### 2.3 API 端点

#### 2.3.1 创建 Webhook

```http
POST /v1/webhooks
```

**请求头**:
```
Authorization: Bearer {JWT令牌}
Content-Type: application/json
```

**请求体**:
```json
{
  "url": "https://your-domain.com/webhook",
  "events": ["mail.received", "mail.read"]
}
```

**响应示例**:
```json
{
  "success": true,
  "data": {
    "id": "wh_xxx",
    "userId": "user_xxx",
    "url": "https://your-domain.com/webhook",
    "events": ["mail.received", "mail.read"],
    "secret": "wh_secret_xxx",
    "isActive": true,
    "retryCount": 0,
    "lastError": "",
    "lastSuccess": null,
    "createdAt": "2024-01-15T10:00:00Z",
    "updatedAt": "2024-01-15T10:00:00Z"
  }
}
```

#### 2.3.2 列出 Webhooks

```http
GET /v1/webhooks
```

**响应示例**:
```json
{
  "success": true,
  "data": [
    {
      "id": "wh_xxx",
      "userId": "user_xxx",
      "url": "https://your-domain.com/webhook",
      "events": ["mail.received"],
      "isActive": true,
      "createdAt": "2024-01-15T10:00:00Z"
    }
  ]
}
```

#### 2.3.3 获取 Webhook 详情

```http
GET /v1/webhooks/:id
```

#### 2.3.4 更新 Webhook

```http
PATCH /v1/webhooks/:id
```

**请求体**:
```json
{
  "url": "https://your-domain.com/new-webhook",
  "events": ["mail.received", "mailbox.created"],
  "isActive": false
}
```

#### 2.3.5 删除 Webhook

```http
DELETE /v1/webhooks/:id
```

#### 2.3.6 获取投递记录

```http
GET /v1/webhooks/:id/deliveries?limit=20
```

**响应示例**:
```json
{
  "success": true,
  "data": [
    {
      "id": "del_xxx",
      "webhookId": "wh_xxx",
      "event": "mail.received",
      "payload": "{...}",
      "statusCode": 200,
      "response": "OK",
      "duration": 150,
      "success": true,
      "error": "",
      "attempts": 1,
      "nextRetry": null,
      "createdAt": "2024-01-15T10:00:00Z"
    }
  ]
}
```

### 2.4 Webhook 回调格式

当事件发生时，系统会向配置的 URL 发送 POST 请求：

**请求头**:
```
Content-Type: application/json
X-Webhook-Signature: sha256=abc123...
X-Webhook-Event: mail.received
X-Webhook-ID: del_xxx
```

**请求体** (mail.received 事件):
```json
{
  "id": "evt_xxx",
  "event": "mail.received",
  "timestamp": "2024-01-15T10:00:00Z",
  "data": {
    "id": "msg_xxx",
    "mailboxId": "mb_xxx",
    "from": "sender@example.com",
    "to": ["recipient@temp.mail"],
    "subject": "New Email",
    "text": "Email content...",
    "html": "<p>Email content...</p>",
    "createdAt": "2024-01-15T10:00:00Z"
  }
}
```

### 2.5 签名验证

为了验证 Webhook 请求的真实性，系统使用 HMAC-SHA256 对 payload 进行签名。

#### 验证步骤：

**1. 提取签名**:
```
X-Webhook-Signature: sha256=abc123...
```

**2. 计算签名**:
```javascript
const crypto = require('crypto');

function verifySignature(payload, signature, secret) {
  const expectedSignature = 'sha256=' + 
    crypto
      .createHmac('sha256', secret)
      .update(payload)
      .digest('hex');
  
  return crypto.timingSafeEqual(
    Buffer.from(signature),
    Buffer.from(expectedSignature)
  );
}
```

**3. 验证示例** (Node.js + Express):
```javascript
app.post('/webhook', express.raw({ type: 'application/json' }), (req, res) => {
  const signature = req.headers['x-webhook-signature'];
  const secret = 'your-webhook-secret';
  const payload = req.body.toString();
  
  // 验证签名
  if (!verifySignature(payload, signature, secret)) {
    return res.status(401).send('Invalid signature');
  }
  
  // 解析事件
  const event = JSON.parse(payload);
  
  console.log('Event:', event.event);
  console.log('Data:', event.data);
  
  // 处理事件...
  
  res.status(200).send('OK');
});
```

### 2.6 重试机制

如果 Webhook 投递失败（HTTP 状态码非 2xx），系统会自动重试。

**重试策略**（指数退避）:

| 尝试次数 | 重试间隔 |
|---------|---------|
| 1 | 立即 |
| 2 | 1分钟后 |
| 3 | 5分钟后 |
| 4 | 15分钟后 |
| 5 | 1小时后 |
| 6 | 6小时后 |
| 7+ | 不再重试 |

**重试任务**: 系统每 5 分钟自动检查并重试失败的投递。

### 2.7 最佳实践

#### 1. 处理幂等性

由于重试机制，您的 Webhook 处理程序可能收到重复的事件。建议：

```javascript
const processedEvents = new Set();

app.post('/webhook', async (req, res) => {
  const event = req.body;
  
  // 使用事件 ID 去重
  if (processedEvents.has(event.id)) {
    return res.status(200).send('Already processed');
  }
  
  try {
    // 处理事件
    await handleEvent(event);
    
    // 标记为已处理
    processedEvents.add(event.id);
    
    res.status(200).send('OK');
  } catch (err) {
    // 返回 500 会触发重试
    res.status(500).send('Internal error');
  }
});
```

#### 2. 异步处理

快速响应 200 OK，然后异步处理事件：

```javascript
app.post('/webhook', async (req, res) => {
  const event = req.body;
  
  // 立即响应
  res.status(200).send('OK');
  
  // 异步处理
  processEventAsync(event).catch(err => {
    console.error('Failed to process event:', err);
  });
});
```

#### 3. 错误处理

```javascript
app.post('/webhook', async (req, res) => {
  try {
    const event = req.body;
    
    // 验证签名
    if (!verifySignature(...)) {
      return res.status(401).send('Invalid signature');
    }
    
    // 处理事件
    await handleEvent(event);
    
    res.status(200).send('OK');
  } catch (err) {
    console.error('Webhook error:', err);
    
    // 临时错误返回 500（会重试）
    if (err.temporary) {
      return res.status(500).send('Temporary error');
    }
    
    // 永久错误返回 400（不会重试）
    res.status(400).send('Bad request');
  }
});
```

#### 4. 安全建议

- ✅ **始终验证签名** - 防止伪造请求
- ✅ **使用 HTTPS** - 保护数据传输
- ✅ **限制IP白名单** - 仅允许服务器IP访问
- ✅ **记录日志** - 便于调试和审计
- ✅ **设置超时** - 避免长时间阻塞

---

## 三、使用场景

### 场景 1: 邮件通知系统

用户希望在收到邮件时立即收到通知。

**实现方式**:

1. 配置 Webhook 订阅 `mail.received` 事件
2. 服务器收到回调后，通过推送服务通知用户

```javascript
// Webhook 处理
app.post('/webhook', async (req, res) => {
  const { event, data } = req.body;
  
  if (event === 'mail.received') {
    // 发送推送通知
    await sendPushNotification({
      title: `新邮件：${data.subject}`,
      body: `来自 ${data.from}`,
      data: {
        messageId: data.id,
        mailboxId: data.mailboxId
      }
    });
  }
  
  res.status(200).send('OK');
});
```

### 场景 2: 自动化测试

测试人员需要验证邮件发送功能。

**实现方式**:

1. 测试脚本触发邮件发送
2. 使用搜索 API 查找验证邮件
3. 提取验证码并完成验证

```javascript
// 自动化测试示例
async function testEmailVerification() {
  // 1. 创建临时邮箱
  const mailbox = await createMailbox();
  
  // 2. 触发邮件发送
  await registerWithEmail(mailbox.address);
  
  // 3. 等待邮件到达
  await sleep(5000);
  
  // 4. 搜索验证邮件
  const result = await searchMessages(mailbox.id, mailbox.token, {
    subject: 'Verification',
    pageSize: 1
  });
  
  // 5. 提取验证码
  const verificationCode = extractCode(result.data.messages[0].text);
  
  // 6. 完成验证
  await verifyEmail(verificationCode);
  
  console.log('✅ Email verification test passed');
}
```

### 场景 3: 邮件归档

企业需要归档所有接收到的邮件。

**实现方式**:

1. 配置 Webhook 订阅 `mail.received` 事件
2. 服务器收到回调后，存储到归档系统

```javascript
app.post('/webhook', async (req, res) => {
  const { event, data } = req.body;
  
  if (event === 'mail.received') {
    // 存储到数据库
    await db.emails.create({
      messageId: data.id,
      mailboxId: data.mailboxId,
      from: data.from,
      subject: data.subject,
      content: data.text,
      receivedAt: data.createdAt,
      archived: true
    });
    
    console.log(`Email archived: ${data.id}`);
  }
  
  res.status(200).send('OK');
});
```

---

## 四、性能说明

### 搜索性能

- **内存存储**: 适合中小规模数据（< 10万封邮件），响应时间 < 100ms
- **MySQL 全文索引**: 适合大规模数据，响应时间 < 200ms
- **PostgreSQL FTS**: 适合超大规模数据，响应时间 < 150ms

### Webhook 性能

- **投递超时**: 10秒
- **并发投递**: 支持
- **重试间隔**: 1分钟 ~ 6小时
- **最大重试次数**: 5次

---

## 五、限制说明

### 搜索限制

| 限制项 | 值 |
|-------|---|
| 每页最大数量 | 100 |
| 默认每页数量 | 20 |
| 最大并发查询 | 100 |

### Webhook 限制

| 限制项 | 值 |
|-------|---|
| 每用户最大 Webhook 数 | 10 |
| 投递超时时间 | 10秒 |
| 最大重试次数 | 5次 |
| Payload 最大大小 | 1MB |

---

## 六、故障排查

### 搜索问题

**问题**: 搜索返回空结果

**解决方案**:
1. 检查邮箱 ID 是否正确
2. 检查邮箱 Token 是否有效
3. 尝试不带条件的搜索
4. 检查时间范围是否合理

### Webhook 问题

**问题**: 没有收到 Webhook 回调

**解决方案**:
1. 检查 Webhook 是否启用 (`isActive: true`)
2. 检查 URL 是否可访问
3. 查看投递记录获取错误信息
4. 检查服务器是否返回 2xx 状态码

**问题**: 签名验证失败

**解决方案**:
1. 确认使用正确的 secret
2. 确认使用原始请求体（不要解析 JSON）
3. 确认签名算法正确（HMAC-SHA256）

---

## 七、更新日志

### v0.9.0 (2024-01-15)

#### 新功能
- ✅ 邮件搜索功能
- ✅ Webhook 回调系统
- ✅ 自动重试机制
- ✅ HMAC 签名验证

#### 改进
- 优化搜索性能
- 添加投递记录查询
- 完善错误处理

---

**文档维护**: TempMail 团队  
**技术支持**: support@tempmail.local
