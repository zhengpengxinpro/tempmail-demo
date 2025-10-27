# TempMail Backend API 文档

## OpenAPI 3.0 文档

本项目已集成 Swagger/OpenAPI 3.0 文档，提供完整的 API 接口说明。

## 访问 API 文档

启动后端服务后，可以通过以下方式访问 API 文档：

### Swagger UI（推荐）
访问地址：`http://localhost:8080/swagger/index.html`

Swagger UI 提供交互式的 API 文档界面，可以：
- 查看所有 API 端点
- 查看请求/响应格式
- 在线测试 API
- 查看认证方式

### OpenAPI JSON
访问地址：`http://localhost:8080/swagger/doc.json`

获取原始的 OpenAPI 3.0 JSON 格式文档。

### OpenAPI YAML
文件位置：`docs/swagger.yaml`

YAML 格式的 OpenAPI 规范文档。

## 重新生成文档

当 API 接口有更新时，需要重新生成 Swagger 文档：

```bash
# 安装 swag 工具（首次使用）
go install github.com/swaggo/swag/cmd/swag@latest

# 生成文档
swag init -g cmd/api/main.go -o docs --parseDepth 1
```

## API 认证方式

本项目支持三种认证方式：

### 1. Bearer Token (JWT)
用于用户认证的端点。

**使用方式：**
```
Authorization: Bearer <your_jwt_token>
```

### 2. Mailbox Token
用于邮箱相关操作的端点。

**使用方式：**
```
X-Mailbox-Token: <your_mailbox_token>
```

### 3. API Key
用于兼容层 API 的认证。

**使用方式：**
```
X-API-Key: <your_api_key>
```

## API 分类

### Public (公开接口)
- 获取可用域名列表
- 获取系统配置

### Auth (认证)
- 用户注册
- 用户登录
- 刷新令牌
- 获取当前用户信息

### Mailboxes (邮箱管理)
- 创建临时邮箱
- 获取邮箱列表
- 获取邮箱详情
- 删除邮箱

### Messages (邮件管理)
- 创建邮件
- 获取邮件列表
- 获取邮件详情
- 标记邮件已读
- 搜索邮件
- 批量删除邮件
- 下载附件

### Aliases (别名管理)
- 创建别名
- 列出别名
- 获取别名详情
- 切换别名状态
- 删除别名

### Tags (标签管理)
- 创建标签
- 列出标签
- 获取标签详情
- 更新标签
- 删除标签
- 为邮件添加标签
- 获取邮件标签
- 移除邮件标签

### Webhooks (Webhook管理)
- 创建 Webhook
- 列出 Webhooks
- 获取 Webhook 详情
- 更新 Webhook
- 删除 Webhook
- 获取投递记录

### Admin (管理员功能)
- 用户管理
- 用户配额管理
- 系统域名管理
- 系统统计
- 系统配置管理

### User Domains (用户域名)
- 添加用户域名
- 列出用户域名
- 获取域名详情
- 验证域名
- 更新域名模式
- 删除域名

### API Keys (API密钥管理)
- 创建 API Key
- 列出 API Keys
- 获取 API Key 详情
- 删除 API Key

### Compat (兼容层API)
提供与 mail.ry.edu.kg 完全兼容的 API 格式。

## 版本信息

- **API 版本**: 0.9.0
- **OpenAPI 版本**: 3.0
- **文档生成工具**: swaggo/swag

## 技术栈

- **Web 框架**: Gin
- **文档工具**: Swagger/OpenAPI 3.0
- **文档生成**: swaggo/swag
- **UI**: Swagger UI

## 联系方式

如有 API 相关问题，请联系：support@example.com
