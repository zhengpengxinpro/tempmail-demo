# API 响应格式说明

TempMail API v0.8.0+ 统一响应格式完整指南

## 📋 目录

- [概述](#概述)
- [响应结构](#响应结构)
- [状态码说明](#状态码说明)
- [成功响应示例](#成功响应示例)
- [错误响应示例](#错误响应示例)
- [迁移指南](#迁移指南)
- [兼容性说明](#兼容性说明)

---

## 🎯 概述

从 **v0.8.0** 开始，TempMail API 采用**面向失败设计（Design for Failure）**理念，所有 API 端点使用统一的响应格式。

### 核心改进

- ✅ **统一响应结构** - 所有端点使用 `{code, msg, data}` 格式
- ✅ **标准 HTTP 状态码** - 保持 RESTful 规范
- ✅ **中文错误消息** - 提升用户体验
- ✅ **业务码与HTTP码分离** - 更灵活的错误处理
- ✅ **向后兼容** - 兼容API保持旧格式

---

## 📦 响应结构

### 统一格式

```json
{
  "code": 200,           // 业务状态码（整数）
  "msg": "成功",          // 响应消息（字符串，中文）
  "data": { ... }        // 响应数据（对象/数组/null）
}
```

### 字段说明

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `code` | int | ✅ | 业务状态码，与HTTP状态码一致 |
| `msg` | string | ✅ | 响应消息，成功或错误描述（中文） |
| `data` | any | ✅ | 响应数据，成功时包含实际数据，失败时为 `null` |

---

## 📊 状态码说明

### 业务状态码

| Code | HTTP状态 | 类型 | 说明 | 示例 |
|------|---------|------|------|------|
| 200  | 200     | 成功 | 操作成功 | 获取数据、更新成功 |
| 201  | 201     | 成功 | 创建成功 | 创建邮箱、注册用户 |
| 204  | 204     | 成功 | 成功无内容 | 删除操作 |
| 400  | 400     | 客户端错误 | 请求参数错误 | 参数格式错误、缺少必填字段 |
| 401  | 401     | 客户端错误 | 未认证 | 缺少令牌、令牌过期 |
| 403  | 403     | 客户端错误 | 无权限 | 权限不足、禁止访问 |
| 404  | 404     | 客户端错误 | 资源不存在 | 邮箱不存在、邮件不存在 |
| 409  | 409     | 客户端错误 | 资源冲突 | 邮箱已存在、域名已存在 |
| 422  | 422     | 客户端错误 | 无法处理 | 域名验证失败 |
| 500  | 500     | 服务器错误 | 内部错误 | 数据库错误、系统异常 |

### HTTP 状态码保持标准

- ✅ 2xx - 成功
- ✅ 4xx - 客户端错误
- ✅ 5xx - 服务器错误

---

## ✅ 成功响应示例

### 1. 获取数据（200）

```json
{
  "code": 200,
  "msg": "成功",
  "data": {
    "id": "mailbox-123",
    "address": "test@temp.mail",
    "createdAt": "2025-10-14T12:00:00Z"
  }
}
```

### 2. 创建资源（201）

```json
{
  "code": 201,
  "msg": "邮箱创建成功",
  "data": {
    "id": "mailbox-456",
    "address": "newuser@temp.mail",
    "token": "eyJhbGc...",
    "expiresAt": "2025-10-15T12:00:00Z"
  }
}
```

### 3. 列表查询（200）

```json
{
  "code": 200,
  "msg": "成功",
  "data": {
    "items": [
      { "id": "1", "subject": "Welcome" },
      { "id": "2", "subject": "Notification" }
    ],
    "total": 2,
    "page": 1,
    "pageSize": 20
  }
}
```

### 4. 删除操作（204）

**响应体为空**（HTTP 204 No Content）

---

## ❌ 错误响应示例

### 1. 请求参数错误（400）

```json
{
  "code": 400,
  "msg": "邮箱前缀格式无效：长度必须在3-32字符之间",
  "data": null
}
```

### 2. 未认证（401）

```json
{
  "code": 401,
  "msg": "认证令牌已过期",
  "data": null
}
```

### 3. 无权限（403）

```json
{
  "code": 403,
  "msg": "无权访问此邮箱",
  "data": null
}
```

### 4. 资源不存在（404）

```json
{
  "code": 404,
  "msg": "邮箱不存在",
  "data": null
}
```

### 5. 资源冲突（409）

```json
{
  "code": 409,
  "msg": "邮箱地址已存在",
  "data": null
}
```

### 6. 无法处理（422）

```json
{
  "code": 422,
  "msg": "域名验证失败，请检查DNS记录",
  "data": null
}
```

### 7. 服务器错误（500）

```json
{
  "code": 500,
  "msg": "服务器内部错误",
  "data": null
}
```

---

## 🔄 迁移指南

### 旧格式（v0.7.0 及之前）

```json
{
  "id": "mailbox-123",
  "address": "test@temp.mail",
  "token": "eyJhbGc...",
  "expiresAt": "2025-10-15T12:00:00Z"
}
```

**错误响应：**
```json
{
  "error": "mailbox not found"
}
```

### 新格式（v0.8.0+）

```json
{
  "code": 200,
  "msg": "成功",
  "data": {
    "id": "mailbox-123",
    "address": "test@temp.mail",
    "token": "eyJhbGc...",
    "expiresAt": "2025-10-15T12:00:00Z"
  }
}
```

**错误响应：**
```json
{
  "code": 404,
  "msg": "邮箱不存在",
  "data": null
}
```

### 客户端代码适配示例

#### JavaScript/TypeScript

**旧代码：**
```javascript
const response = await fetch('/v1/mailboxes');
const mailbox = await response.json();
console.log(mailbox.address);
```

**新代码：**
```javascript
const response = await fetch('/v1/mailboxes');
const result = await response.json();

if (result.code === 200) {
  console.log(result.data.address);
} else {
  console.error(result.msg);
}
```

#### Python

**旧代码：**
```python
response = requests.get('/v1/mailboxes')
mailbox = response.json()
print(mailbox['address'])
```

**新代码：**
```python
response = requests.get('/v1/mailboxes')
result = response.json()

if result['code'] == 200:
    print(result['data']['address'])
else:
    print(result['msg'])
```

#### Go

**旧代码：**
```go
var mailbox Mailbox
json.Unmarshal(body, &mailbox)
fmt.Println(mailbox.Address)
```

**新代码：**
```go
var response Response
json.Unmarshal(body, &response)

if response.Code == 200 {
    var mailbox Mailbox
    json.Unmarshal(response.Data, &mailbox)
    fmt.Println(mailbox.Address)
} else {
    fmt.Println(response.Msg)
}
```

---

## 🔌 兼容性说明

### 兼容 API 格式说明（v0.8.2 更新）

从 v0.8.2 开始，**兼容 API 全部使用旧格式**（直接返回数据），与 mail.ry.edu.kg 完全一致：

| 端点 | 格式 | 说明 |
|------|------|------|
| `/api/config` | 旧格式（直接返回数据） | 与 mail.ry.edu.kg 一致 |
| `/api/emails/generate` | 旧格式（直接返回数据） | v0.8.2+ |
| `/api/emails` | 旧格式（直接返回数据） | v0.8.2+ |
| `/api/emails/{emailId}` | 旧格式（直接返回数据） | v0.8.2+ |
| `/api/emails/{emailId}/{messageId}` | 旧格式（直接返回数据） | v0.8.2+ |

### HTTP 状态码保持不变

- ✅ 成功仍然返回 2xx
- ✅ 客户端错误仍然返回 4xx
- ✅ 服务器错误仍然返回 5xx

**这意味着基于 HTTP 状态码的错误处理逻辑无需修改！**

### 渐进式迁移策略

1. **阶段 1**：继续使用兼容 API，无需修改
2. **阶段 2**：新功能使用新格式 API
3. **阶段 3**：逐步迁移旧代码到新格式
4. **阶段 4**：完全迁移后移除兼容 API 依赖

---

## 📝 常见问题

### Q1: 为什么要改变响应格式？

**A:** 采用统一格式有以下优势：
- 更清晰的成功/失败判断
- 更友好的中文错误消息
- 更灵活的业务状态码
- 更好的API一致性
- 符合"面向失败设计"理念

### Q2: 旧客户端会受影响吗？

**A:** 
- ✅ 兼容API (`/compat/*`) 保持旧格式，无影响
- ⚠️ 主API (`/v1/*`) 已更改，需要适配
- ✅ HTTP状态码不变，基础错误处理无需修改

### Q3: 如何快速判断成功或失败？

**A:** 三种方式：
1. 检查 `response.code === 200/201/204`
2. 检查 HTTP 状态码 `response.status === 200/201/204`
3. 检查 `response.data !== null`

### Q4: data 字段什么时候是 null？

**A:** 
- 所有错误响应（4xx, 5xx）的 `data` 都是 `null`
- 删除操作（204）通常 `data` 为 `null`
- 其他成功响应 `data` 包含实际数据

### Q5: 业务状态码和HTTP状态码有什么区别？

**A:** 
- **HTTP状态码**：传输层状态，RESTful标准
- **业务状态码**：应用层状态，与HTTP码一致但可扩展
- 当前版本两者完全一致，未来可能扩展业务码

---

## 📚 相关文档

- [API 参考文档](./API.md) - 完整API文档
- [兼容API说明](./API_COMPATIBILITY.md) - 兼容层详细说明
- [更新日志](./CHANGELOG.md) - 版本更新记录

---

**最后更新**: 2025-10-14
**适用版本**: v0.8.0+
