# 系统域名管理 API 文档

## 概述

系统域名管理功能允许管理员通过后台 API 动态添加、验证和管理邮箱域名，而无需重启服务。SMTP 服务器会自动加载已激活的系统域名。

## 功能特性

- ✅ **动态添加域名** - 无需重启服务即可添加新域名
- ✅ **DNS 验证** - 通过 TXT 记录验证域名所有权
- ✅ **自动清理** - 未验证的域名在 24 小时后自动删除
- ✅ **域名找回** - 误删除的域名可以通过 DNS 验证找回
- ✅ **状态管理** - 支持启用/禁用域名
- ✅ **默认域名** - 可设置默认域名供用户创建邮箱
- ✅ **SMTP 集成** - SMTP 服务器动态加载激活的域名，只接收邮件

## API 端点列表

所有端点都需要管理员权限，基础路径：`/v1/admin/domains`

| 方法 | 端点 | 权限 | 说明 |
|------|------|------|------|
| GET | `/v1/admin/domains` | Admin | 获取所有系统域名列表 |
| POST | `/v1/admin/domains` | Super | 添加新的系统域名 |
| POST | `/v1/admin/domains/recover` | Super | 找回已删除的域名 |
| GET | `/v1/admin/domains/:id` | Admin | 获取域名详情 |
| POST | `/v1/admin/domains/:id/verify` | Admin | 验证域名所有权 |
| GET | `/v1/admin/domains/:id/instructions` | Admin | 获取 DNS 配置说明 |
| PATCH | `/v1/admin/domains/:id/toggle` | Admin | 启用/禁用域名 |
| POST | `/v1/admin/domains/:id/set-default` | Super | 设置为默认域名 |
| DELETE | `/v1/admin/domains/:id` | Super | 删除域名 |

## 完整使用流程

### 步骤 1: 添加域名

**请求**：
```bash
POST /v1/admin/domains
Authorization: Bearer {admin_token}
Content-Type: application/json

{
  "domain": "mail.example.com",
  "notes": "公司邮箱域名"
}
```

**响应**：
```json
{
  "success": true,
  "data": {
    "id": "domain-uuid-123",
    "domain": "mail.example.com",
    "status": "pending",
    "verifyToken": "a1b2c3d4e5f6...",
    "verifyMethod": "dns_txt",
    "isActive": false,
    "isDefault": false,
    "mailboxCount": 0,
    "mxRecords": ["10 mail.tempmail.dev"],
    "createdAt": "2024-01-15T10:30:00Z",
    "createdBy": "admin-user-id",
    "notes": "公司邮箱域名"
  }
}
```

### 步骤 2: 获取 DNS 配置说明

**请求**：
```bash
GET /v1/admin/domains/{domain_id}/instructions
Authorization: Bearer {admin_token}
```

**响应**：
```json
{
  "success": true,
  "data": {
    "domain": "mail.example.com",
    "status": "pending",
    "steps": [
      {
        "step": 1,
        "title": "添加 TXT 记录验证域名所有权",
        "description": "在您的 DNS 提供商处添加以下 TXT 记录：",
        "record": {
          "type": "TXT",
          "name": "@",
          "value": "tempmail-verify=a1b2c3d4e5f6...",
          "ttl": "3600"
        }
      },
      {
        "step": 2,
        "title": "添加 MX 记录接收邮件",
        "description": "添加以下 MX 记录以接收邮件：",
        "records": [
          {
            "type": "MX",
            "name": "@",
            "priority": "10",
            "value": "mail.tempmail.dev",
            "ttl": "3600"
          }
        ]
      },
      {
        "step": 3,
        "title": "等待 DNS 生效并验证",
        "description": "DNS 记录通常需要 5-30 分钟生效，请耐心等待后点击验证按钮。"
      }
    ]
  }
}
```

### 步骤 3: 在 DNS 服务商配置记录

根据步骤 2 获取的配置说明，在您的 DNS 服务商（如 Cloudflare、阿里云、腾讯云等）处添加以下记录：

#### TXT 记录（验证域名所有权）
```
类型: TXT
主机记录: @
记录值: tempmail-verify=a1b2c3d4e5f6...
TTL: 3600（或保持默认）
```

#### MX 记录（接收邮件）
```
类型: MX
主机记录: @
优先级: 10
记录值: mail.tempmail.dev
TTL: 3600（或保持默认）
```

### 步骤 4: 验证域名

等待 DNS 生效（通常 5-30 分钟），然后进行验证：

**请求**：
```bash
POST /v1/admin/domains/{domain_id}/verify
Authorization: Bearer {admin_token}
```

**验证成功响应**：
```json
{
  "success": true,
  "data": {
    "id": "domain-uuid-123",
    "domain": "mail.example.com",
    "status": "verified",
    "isActive": true,
    "verifiedAt": "2024-01-15T11:00:00Z",
    "lastCheckAt": "2024-01-15T11:00:00Z"
  }
}
```

**验证失败响应**：
```json
{
  "success": false,
  "error": "DNS 验证失败，请检查 TXT 记录是否正确配置",
  "code": "DOMAIN_VERIFY_FAILED"
}
```

### 步骤 5: 设置为默认域名（可选）

**请求**：
```bash
POST /v1/admin/domains/{domain_id}/set-default
Authorization: Bearer {admin_token}
```

**响应**：
```json
{
  "success": true,
  "message": "默认域名设置成功"
}
```

## 域名状态说明

| 状态 | 说明 | 是否激活 | 可以接收邮件 |
|------|------|----------|--------------|
| `pending` | 待验证 | ❌ | ❌ |
| `verified` | 已验证 | ✅ | ✅ |
| `failed` | 验证失败 | ❌ | ❌ |

## 其他管理操作

### 获取域名列表

```bash
GET /v1/admin/domains
Authorization: Bearer {admin_token}
```

**响应**：
```json
{
  "success": true,
  "data": [
    {
      "id": "domain-1",
      "domain": "mail.example.com",
      "status": "verified",
      "isActive": true,
      "isDefault": true,
      "mailboxCount": 150
    },
    {
      "id": "domain-2",
      "domain": "temp.example.org",
      "status": "pending",
      "isActive": false,
      "isDefault": false,
      "mailboxCount": 0
    }
  ]
}
```

### 启用/禁用域名

```bash
PATCH /v1/admin/domains/{domain_id}/toggle
Authorization: Bearer {admin_token}
Content-Type: application/json

{
  "isActive": false
}
```

### 删除域名

**注意**：
- 不能删除默认域名
- 不能删除还有邮箱的域名
- 删除操作不可恢复（可通过找回功能恢复）

```bash
DELETE /v1/admin/domains/{domain_id}
Authorization: Bearer {admin_token}
```

## 域名找回功能

如果域名被误删除或验证失败，可以通过找回功能恢复：

**前提条件**：
- DNS TXT 记录仍然配置正确
- 验证令牌仍然有效

**请求**：
```bash
POST /v1/admin/domains/recover
Authorization: Bearer {admin_token}
Content-Type: application/json

{
  "domain": "mail.example.com"
}
```

**成功响应**：
```json
{
  "success": true,
  "data": {
    "id": "new-domain-uuid",
    "domain": "mail.example.com",
    "status": "verified",
    "isActive": true,
    "notes": "通过找回功能恢复"
  }
}
```

**失败响应**：
```json
{
  "success": false,
  "error": "未找到有效的验证令牌，请先添加 DNS TXT 记录",
  "code": "DOMAIN_RECOVER_FAILED"
}
```

## 自动清理机制

系统会自动清理未验证的域名：

- **清理周期**：每小时检查一次
- **清理条件**：创建超过 24 小时且状态仍为 `pending` 的域名
- **日志记录**：清理操作会在服务器日志中记录

## SMTP 服务器集成

### 动态域名加载

SMTP 服务器会自动加载所有已激活（`status=verified` 且 `isActive=true`）的系统域名：

```go
// 内部实现（开发者参考）
activeDomains, err := systemDomainService.GetAllActiveDomains()
// 返回: ["mail.example.com", "temp.example.org", ...]
```

### 收件验证流程

当 SMTP 服务器收到邮件时：

1. **提取收件人域名**
   - 例如：`user@mail.example.com` → `mail.example.com`

2. **检查域名是否激活**
   - 在系统域名列表中查找
   - 在用户自定义域名列表中查找（如果存在）

3. **验证邮箱存在性**
   - 查找主邮箱
   - 查找别名

4. **拒绝策略**
   - 域名不在管理列表 → `550 Relay access denied`
   - 邮箱不存在 → `550 Recipient mailbox not found`

### 安全特性

✅ **只接收邮件** - 不支持对外发送邮件
✅ **严格域名验证** - 只接收管理域名的邮件
✅ **防止邮件中继** - 拒绝所有发往外部地址的邮件
✅ **动态配置** - 无需重启即可生效

## 错误处理

### 常见错误码

| 错误码 | HTTP 状态码 | 说明 |
|--------|-------------|------|
| `DOMAIN_ALREADY_EXISTS` | 409 | 域名已存在 |
| `DOMAIN_NOT_FOUND` | 404 | 域名不存在 |
| `DOMAIN_VERIFY_FAILED` | 422 | DNS 验证失败 |
| `DOMAIN_NOT_VERIFIED` | 400 | 域名未验证 |
| `DOMAIN_HAS_MAILBOXES` | 409 | 域名下还有邮箱，不能删除 |
| `CANNOT_DELETE_DEFAULT_DOMAIN` | 400 | 不能删除默认域名 |
| `INVALID_DOMAIN_FORMAT` | 400 | 域名格式无效 |

### DNS 验证失败排查

1. **检查 TXT 记录**
   ```bash
   # Linux/Mac
   dig TXT mail.example.com
   
   # Windows
   nslookup -type=TXT mail.example.com
   ```

2. **等待 DNS 生效**
   - 不同 DNS 提供商生效时间不同（5分钟至2小时）
   - 可以使用在线工具检查：https://dnschecker.org

3. **确认记录值正确**
   - 必须完全匹配 `tempmail-verify={token}`
   - 不要有多余的空格或引号

## 最佳实践

1. **添加域名后立即配置 DNS**
   - 避免超过 24 小时导致自动清理

2. **测试验证后再大规模使用**
   - 先用测试域名验证流程

3. **保留验证令牌**
   - 如果需要找回域名，保留 DNS TXT 记录

4. **定期检查域名状态**
   - 确保所有域名都是 `verified` 状态

5. **禁用而不是删除**
   - 临时不用的域名可以禁用，避免数据丢失

## 监控和日志

系统会记录以下操作日志：

```
INFO  domain added: mail.example.com (id=domain-123)
INFO  domain verified: mail.example.com
INFO  unverified system domains cleaned up (count=3)
INFO  domain recovered: mail.example.com
```

## 示例代码

### cURL 示例

```bash
# 1. 登录获取 Token
TOKEN=$(curl -X POST http://localhost:8080/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com","password":"admin123"}' \
  | jq -r '.data.accessToken')

# 2. 添加域名
DOMAIN_ID=$(curl -X POST http://localhost:8080/v1/admin/domains \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"domain":"mail.test.com","notes":"测试域名"}' \
  | jq -r '.data.id')

# 3. 获取配置说明
curl http://localhost:8080/v1/admin/domains/$DOMAIN_ID/instructions \
  -H "Authorization: Bearer $TOKEN"

# 4. 验证域名（等待 DNS 配置完成后）
curl -X POST http://localhost:8080/v1/admin/domains/$DOMAIN_ID/verify \
  -H "Authorization: Bearer $TOKEN"

# 5. 设置为默认域名
curl -X POST http://localhost:8080/v1/admin/domains/$DOMAIN_ID/set-default \
  -H "Authorization: Bearer $TOKEN"
```

### JavaScript 示例

```javascript
// 使用 fetch API
const API_BASE = 'http://localhost:8080/v1';
const token = 'your-admin-token';

async function addDomain(domain, notes) {
  const response = await fetch(`${API_BASE}/admin/domains`, {
    method: 'POST',
    headers: {
      'Authorization': `Bearer ${token}`,
      'Content-Type': 'application/json'
    },
    body: JSON.stringify({ domain, notes })
  });
  
  const result = await response.json();
  if (result.success) {
    console.log('域名添加成功:', result.data);
    return result.data.id;
  } else {
    console.error('添加失败:', result.error);
    throw new Error(result.error);
  }
}

async function verifyDomain(domainId) {
  const response = await fetch(`${API_BASE}/admin/domains/${domainId}/verify`, {
    method: 'POST',
    headers: {
      'Authorization': `Bearer ${token}`
    }
  });
  
  const result = await response.json();
  if (result.success) {
    console.log('验证成功:', result.data);
    return result.data;
  } else {
    console.error('验证失败:', result.error);
    throw new Error(result.error);
  }
}

// 使用示例
(async () => {
  try {
    const domainId = await addDomain('mail.example.com', '测试域名');
    console.log('域名ID:', domainId);
    
    // 等待 DNS 配置...
    console.log('请配置 DNS 记录，然后按任意键继续...');
    
    const verified = await verifyDomain(domainId);
    console.log('验证结果:', verified);
  } catch (error) {
    console.error('操作失败:', error);
  }
})();
```

## 更新日志

- **v0.8.2** - 2024-01-15
  - 首次发布系统域名管理功能
  - 支持 DNS TXT 验证
  - 自动清理机制
  - 域名找回功能

---

**最后更新**: 2024-01-15
**维护者**: 开发团队
