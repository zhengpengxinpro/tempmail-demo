# TempMail API 使用示例与调试指南

**版本**: v0.8.2-beta  
**最后更新**: 2025-01-15  

## 📋 目录

- [快速上手示例](#快速上手示例)
- [前端集成示例](#前端集成示例)
- [后端集成示例](#后端集成示例)
- [调试工具与技巧](#调试工具与技巧)
- [常见问题排查](#常见问题排查)
- [性能优化建议](#性能优化建议)

---

## 🚀 快速上手示例

### 最简单的使用流程

```bash
# 1. 创建临时邮箱（无需注册）
curl -X POST http://localhost:8080/v1/mailboxes

# 2. 查看返回的数据，记录邮箱ID和Token
# 响应示例：
# {
#   "code": 200,
#   "data": {
#     "id": "abc-123",
#     "address": "test@temp.mail",
#     "token": "**MAILBOX_EXAMPLE"
#   }
# }

# 3. 查看邮件（使用返回的邮箱ID和Token）
curl -X GET http://localhost:8080/v1/mailboxes/abc-123/messages \
  -H "X-Mailbox-Token: AbCdEf123456"
```

---

## 🌐 前端集成示例

### React + TypeScript

```typescript
// api.ts
interface ApiConfig {
  baseURL: string;
}

class TempMailAPI {
  private baseURL: string;
  
  constructor(config: ApiConfig) {
    this.baseURL = config.baseURL;
  }

  // 创建邮箱
  async createMailbox(prefix?: string, domain?: string) {
    const response = await fetch(`${this.baseURL}/v1/mailboxes`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(prefix || domain ? {
        prefix,
        domain,
      } : {})
    });
    
    const result = await response.json();
    if (result.code !== 200) {
      throw new Error(result.msg);
    }
    return result.data;
  }

  // 获取邮件列表
  async getMessages(mailboxId: string, token: string) {
    const response = await fetch(
      `${this.baseURL}/v1/mailboxes/${mailboxId}/messages`,
      {
        headers: {
          'X-Mailbox-Token': token
        }
      }
    );
    
    const result = await response.json();
    if (result.code !== 200) {
      throw new Error(result.msg);
    }
    return result.data.items;
  }

  // 用户注册
  async register(email: string, password: string, username: string) {
    const response = await fetch(`${this.baseURL}/v1/auth/register`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ email, password, username })
    });
    
    const result = await response.json();
    if (result.code !== 201) {
      throw new Error(result.msg);
    }
    return result.data;
  }

  // 用户登录
  async login(email: string, password: string) {
    const response = await fetch(`${this.baseURL}/v1/auth/login`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ email, password })
    });
    
    const result = await response.json();
    if (result.code !== 200) {
      throw new Error(result.msg);
    }
    return result.data;
  }

  // 获取用户信息
  async getUserInfo(accessToken: string) {
    const response = await fetch(`${this.baseURL}/v1/auth/me`, {
      headers: {
        'Authorization': `Bearer ${accessToken}`
      }
    });
    
    const result = await response.json();
    if (result.code !== 200) {
      throw new Error(result.msg);
    }
    return result;
  }

  // WebSocket连接
  connectWebSocket(mailboxId: string, token: string) {
    const wsURL = `${this.baseURL.replace('http', 'ws')}/v1/ws`;
    const ws = new WebSocket(wsURL);
    
    ws.onopen = () => {
      // 订阅邮箱
      ws.send(JSON.stringify({
        type: 'subscribe',
        mailboxId,
        token
      }));
    };
    
    return ws;
  }
}

// 使用示例
const api = new TempMailAPI({
  baseURL: 'http://localhost:8080'
});

// React组件示例
export function MailboxComponent() {
  const [mailbox, setMailbox] = useState(null);
  const [messages, setMessages] = useState([]);
  const [loading, setLoading] = useState(false);

  const createMailbox = async () => {
    setLoading(true);
    try {
      const result = await api.createMailbox('mytemp', 'temp.mail');
      setMailbox(result);
      // 自动获取消息
      loadMessages(result.id, result.token);
    } catch (error) {
      console.error('创建邮箱失败:', error.message);
    } finally {
      setLoading(false);
    }
  };

  const loadMessages = async (mailboxId: string, token: string) => {
    try {
      const messageList = await api.getMessages(mailboxId, token);
      setMessages(messageList);
    } catch (error) {
      console.error('获取消息失败:', error.message);
    }
  };

  return (
    <div>
      <button onClick={createMailbox} disabled={loading}>
        {loading ? '创建中...' : '创建邮箱'}
      </button>
      
      {mailbox && (
        <div>
          <h3>邮箱地址: {mailbox.address}</h3>
          <h4>邮件列表 ({messages.length})</h4>
          <ul>
            {messages.map((msg: any) => (
              <li key={msg.id}>
                <strong>{msg.subject}</strong> - {msg.from}
                <br />
                <small>{new Date(msg.receivedAt).toLocaleString()}</small>
              </li>
            ))}
          </ul>
        </div>
      )}
    </div>
  );
}
```

### Vue.js 3 + TypeScript

```typescript
// composables/useTempMail.ts
import { ref, computed } from 'vue';

export function useTempMail() {
  const baseURL = 'http://localhost:8080';
  const mailbox = ref(null);
  const messages = ref([]);
  const loading = ref(false);
  const error = ref('');

  const createMailbox = async (prefix?: string, domain?: string) => {
    loading.value = true;
    error.value = '';
    
    try {
      const response = await fetch(`${baseURL}/v1/mailboxes`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(prefix || domain ? { prefix, domain } : {})
      });
      
      const result = await response.json();
      if (result.code === 200) {
        mailbox.value = result.data;
        await loadMessages(result.data.id, result.data.token);
      } else {
        error.value = result.msg;
      }
    } catch (e) {
      error.value = '网络错误';
    } finally {
      loading.value = false;
    }
  };

  const loadMessages = async (mailboxId: string, token: string) => {
    try {
      const response = await fetch(
        `${baseURL}/v1/mailboxes/${mailboxId}/messages`,
        {
          headers: {
            'X-Mailbox-Token': token
          }
        }
      );
      
      const result = await response.json();
      if (result.code === 200) {
        messages.value = result.data.items;
      }
    } catch (e) {
      console.error('加载消息失败:', e);
    }
  };

  return {
    mailbox,
    messages,
    loading,
    error,
    createMailbox,
    loadMessages
  };
}

// 组件使用
<template>
  <div>
    <button @click="createMailbox('test123')" :disabled="loading">
      {{ loading ? '创建中...' : '创建邮箱' }}
    </button>
    
    <div v-if="error" class="error">{{ error }}</div>
    
    <div v-if="mailbox">
      <h3>邮箱: {{ mailbox.address }}</h3>
      <p>邮箱ID: {{ mailbox.id }}</p>
      <p>Token: {{ mailbox.token }}</p>
      
      <h4>邮件 ({{ messages.length }})</h4>
      <div v-if="messages.length === 0">暂无邮件</div>
      <div v-else>
        <div v-for="msg in messages" :key="msg.id" class="message">
          <strong>{{ msg.subject }}</strong>
          <span class="from">{{ msg.from }}</span>
          <span class="time">{{ formatTime(msg.receivedAt) }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useTempMail } from '../composables/useTempMail';

const { mailbox, messages, loading, error, createMailbox } = useTempMail();

const formatTime = (timeStr: string) => {
  return new Date(timeStr).toLocaleString();
};
</script>
```

---

## ⚙️ 后端集成示例

### Node.js + Express

```javascript
// tempmail-api.js
const axios = require('axios');

class TempMailService {
  constructor(apiURL = 'http://localhost:8080') {
    this.apiURL = apiURL;
    this.axios = axios.create({
      baseURL: apiURL,
      timeout: 10000,
    });
  }

  // 错误处理
  handleResponse(response) {
    const { data } = response;
    if (data.code !== 200 && data.code !== 201) {
      throw new Error(data.msg || 'API请求失败');
    }
    return data.data;
  }

  // 创建邮箱
  async createMailbox(options = {}) {
    const { prefix, domain } = options;
    try {
      const response = await this.axios.post('/v1/mailboxes', {
        prefix,
        domain,
      });
      return this.handleResponse(response);
    } catch (error) {
      if (error.response) {
        throw new Error(error.response.data.msg || '创建邮箱失败');
      }
      throw error;
    }
  }

  // 获取邮件列表
  async getMessages(mailboxId, token) {
    try {
      const response = await this.axios.get(
        `/v1/mailboxes/${mailboxId}/messages`,
        {
          headers: {
            'X-Mailbox-Token': token,
          },
        }
      );
      return this.handleResponse(response);
    } catch (error) {
      if (error.response) {
        throw new Error(error.response.data.msg || '获取邮件失败');
      }
      throw error;
    }
  }

  // 用户注册
  async register(email, password, username) {
    try {
      const response = await this.axios.post('/v1/auth/register', {
        email,
        password,
        username,
      });
      return this.handleResponse(response);
    } catch (error) {
      if (error.response) {
        throw new Error(error.response.data.msg || '注册失败');
      }
      throw error;
    }
  }

  // 用户登录
  async login(email, password) {
    try {
      const response = await this.axios.post('/v1/auth/login', {
        email,
        password,
      });
      return this.handleResponse(response);
    } catch (error) {
      if (error.response) {
        throw new Error(error.response.data.msg || '登录失败');
      }
      throw error;
    }
  }

  // WebSocket服务
  createWebSocketServer(mailboxId, token) {
    const WebSocket = require('ws');
    const wsURL = this.apiURL.replace('http', 'ws');
    
    const ws = new WebSocket(`${wsURL}/v1/ws`);
    
    ws.on('open', () => {
      console.log('WebSocket连接已建立');
      ws.send(JSON.stringify({
        type: 'subscribe',
        mailboxId,
        token,
      }));
    });

    return ws;
  }
}

// Express路由示例
const express = require('express');
const TempMailService = require('./tempmail-api');

const app = express();
app.use(express.json());

const tempMail = new TempMailService();

// 创建临时邮箱路由
app.post('/api/mailbox/create', async (req, res) => {
  try {
    const { prefix, domain } = req.body;
    const mailbox = await tempMail.createMailbox({ prefix, domain });
    
    res.json({
      success: true,
      data: mailbox,
    });
  } catch (error) {
    res.status(400).json({
      success: false,
      error: error.message,
    });
  }
});

// 获取邮件列表路由
app.get('/api/mailbox/:id/messages', async (req, res) => {
  try {
    const { id } = req.params;
    const { token } = req.query;
    
    const messages = await tempMail.getMessages(id, token);
    
    res.json({
      success: true,
      data: messages,
    });
  } catch (error) {
    res.status(400).json({
      success: false,
      error: error.message,
    });
  }
});

// 用户注册路由
app.post('/api/auth/register', async (req, res) => {
  try {
    const { email, password, username } = req.body;
    const userData = await tempMail.register(email, password, username);
    
    res.json({
      success: true,
      data: userData,
    });
  } catch (error) {
    res.status(400).json({
      success: false,
      error: error.message,
    });
  }
});

// 启动服务器
const PORT = process.env.PORT || 3001;
app.listen(PORT, () => {
  console.log(`服务器运行在端口 ${PORT}`);
});
```

### Python Flask

```python
# tempmail_service.py
import requests
from typing import Optional, List, Dict, Any
import json

class TempMailService:
    def __init__(self, api_url: str = "http://localhost:8080"):
        self.api_url = api_url.rstrip('/')
        self.session = requests.Session()
        self.session.headers.update({
            'Content-Type': 'application/json'
        })

    def _make_request(self, method: str, endpoint: str, **kwargs) -> Dict[str, Any]:
        """发送HTTP请求并处理响应"""
        url = f"{self.api_url}{endpoint}"
        
        try:
            response = self.session.request(method, url, **kwargs)
            response.raise_for_status()
            
            data = response.json()
            if data.get('code') not in [200, 201]:
                raise Exception(data.get('msg', 'API请求失败'))
                
            return data.get('data')
        except requests.exceptions.RequestException as e:
            raise Exception(f"网络请求失败: {str(e)}")

    def create_mailbox(self, prefix: Optional[str] = None, 
                       domain: Optional[str] = None) -> Dict[str, Any]:
        """创建临时邮箱"""
        data = {}
        if prefix:
            data['prefix'] = prefix
        if domain:
            data['domain'] = domain
            
        return self._make_request('POST', '/v1/mailboxes', json=data)

    def get_messages(self, mailbox_id: str, token: str, 
                     limit: Optional[int] = None,
                     offset: Optional[int] = None) -> List[Dict[str, Any]]:
        """获取邮件列表"""
        headers = {'X-Mailbox-Token': token}
        params = {}
        if limit:
            params['limit'] = limit
        if offset:
            params['offset'] = offset
            
        return self._make_request(
            'GET', 
            f'/v1/mailboxes/{mailbox_id}/messages',
            headers=headers,
            params=params
        )

    def register(self, email: str, password: str, username: str) -> Dict[str, Any]:
        """用户注册"""
        data = {
            'email': email,
            'password': password,
            'username': username
        }
        return self._make_request('POST', '/v1/auth/register', json=data)

    def login(self, email: str, password: str) -> Dict[str, Any]:
        """用户登录"""
        data = {
            'email': email,
            'password': password
        }
        return self._make_request('POST', '/v1/auth/login', json=data)

    def get_user_info(self, access_token: str) -> Dict[str, Any]:
        """获取用户信息"""
        headers = {'Authorization': f'Bearer {access_token}'}
        return self._make_request('GET', '/v1/auth/me', headers=headers)


# Flask应用示例
from flask import Flask, request, jsonify
from tempmail_service import TempMailService

app = Flask(__name__)
tempmail = TempMailService()

@app.route('/api/mailbox', methods=['POST'])
def create_mailbox():
    try:
        data = request.get_json()
        prefix = data.get('prefix')
        domain = data.get('domain')
        
        mailbox = tempmail.create_mailbox(prefix, domain)
        return jsonify({
            'success': True,
            'data': mailbox
        })
    except Exception as e:
        return jsonify({
            'success': False,
            'error': str(e)
        }), 400

@app.route('/api/mailbox/<mailbox_id>/messages')
def get_messages(mailbox_id):
    try:
        token = request.args.get('token')
        if not token:
            return jsonify({
                'success': False,
                'error': '缺少token参数'
            }), 400
            
        messages = tempmail.get_messages(mailbox_id, token)
        return jsonify({
            'success': True,
            'data': messages
        })
    except Exception as e:
        return jsonify({
            'success': False,
            'error': str(e)
        }), 400

@app.route('/api/auth/register', methods=['POST'])
def register():
    try:
        data = request.get_json()
        user_data = tempmail.register(
            data['email'], 
            data['password'], 
            data['username']
        )
        return jsonify({
            'success': True,
            'data': user_data
        })
    except Exception as e:
        return jsonify({
            'success': False,
            'error': str(e)
        }), 400

if __name__ == '__main__':
    app.run(debug=True, port=5000)
```

---

## 🛠️ 调试工具与技巧

### 1. 使用Postman调试

#### 导入API集合
创建以下集合和环境：

**环境变量**:
```
base_url: http://localhost:8080
mailbox_id: {{mailbox_id}}
mailbox_token: {{mailbox_token}}
access_token: {{access_token}}
```

**基础请求示例**:

1. **创建邮箱**
```http
POST {{base_url}}/v1/mailboxes
Content-Type: application/json

{
  "prefix": "test",
  "domain": "temp.mail"
}
```

2. **获取邮件**
```http
GET {{base_url}}/v1/mailboxes/{{mailbox_id}}/messages
X-Mailbox-Token: {{mailbox_token}}
```

### 2. 使用curl调试

#### 批量测试脚本
```bash
#!/bin/bash

API_URL="http://localhost:8080"

echo "=== TempMail API 测试脚本 ==="

# 1. 健康检查
echo "1. 健康检查..."
curl -s "$API_URL/health" | jq

# 2. 创建邮箱
echo "2. 创建邮箱..."
RESPONSE=$(curl -s -X POST "$API_URL/v1/mailboxes" \
  -H "Content-Type: application/json" \
  -d '{"prefix": "test", "domain": "temp.mail"}')

echo $RESPONSE | jq

# 提取邮箱信息
MAILBOX_ID=$(echo $RESPONSE | jq -r '.data.id')
MAILBOX_TOKEN=$(echo $RESPONSE | jq -r '.data.token')

echo "邮箱ID: $MAILBOX_ID"
echo "Token: $MAILBOX_TOKEN"

# 3. 获取邮件列表
echo "3. 获取邮件列表..."
curl -s "$API_URL/v1/mailboxes/$MAILBOX_ID/messages" \
  -H "X-Mailbox-Token: $MAILBOX_TOKEN" | jq

# 4. 获取邮箱详情
echo "4. 获取邮箱详情..."
curl -s "$API_URL/v1/mailboxes/$MAILBOX_ID" \
  -H "X-Mailbox-Token: $MAILBOX_TOKEN" | jq

# 5. 删除邮箱
echo "5. 删除邮箱..."
curl -s -X DELETE "$API_URL/v1/mailboxes/$MAILBOX_ID" \
  -H "X-Mailbox-Token: $MAILBOX_TOKEN" -w "%{http_code}"

echo "=== 测试完成 ==="
```

### 3. 使用浏览器开发者工具

#### 网络监控
```javascript
// 在浏览器控制台中执行的调试脚本

// 1. 测试API调用
async function testAPI() {
  try {
    // 创建邮箱
    const createResponse = await fetch('/v1/mailboxes', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      }
    });
    
    const createData = await createResponse.json();
    console.log('创建邮箱结果:', createData);
    
    // 获取邮件列表
    const messageResponse = await fetch(
      `/v1/mailboxes/${createData.data.id}/messages`,
      {
        headers: {
          'X-Mailbox-Token': createData.data.token
        }
      }
    );
    
    const messageData = await messageResponse.json();
    console.log('邮件列表:', messageData);
    
  } catch (error) {
    console.error('API测试失败:', error);
  }
}

// 2. 监控WebSocket连接
function debugWebSocket(mailboxId, token) {
  const ws = new WebSocket(`ws://localhost:8080/v1/ws`);
  
  ws.onopen = () => {
    console.log('WebSocket连接成功');
    ws.send(JSON.stringify({
      type: 'subscribe',
      mailboxId,
      token
    }));
  };
  
  ws.onmessage = (event) => {
    console.log('WebSocket消息:', JSON.parse(event.data));
  };
  
  ws.onerror = (error) => {
    console.error('WebSocket错误:', error);
  };
  
  ws.onclose = (event) => {
    console.log('WebSocket关闭:', event.code, event.reason);
  };
  
  return ws;
}
```

### 4. 使用API测试工具

#### Newman (Postman CLI) Newman测试脚本
```json
{
  "info": {
    "name": "TempMail API Tests",
    "schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
  },
  "item": [
    {
      "name": "Create Mailbox",
      "request": {
        "method": "POST",
        "header": [
          {
            "key": "Content-Type",
            "value": "application/json"
          }
        ],
        "body": {
          "mode": "raw",
          "raw": "{\"prefix\": \"test\", \"domain\": \"temp.mail\"}"
        },
        "url": {
          "raw": "{{base_url}}/v1/mailboxes"
      }
    },
    "test": [
      {
        "script": {
          "exec": [
            "pm.test("邮箱创建成功", function () {",
            "  pm.expect(pm.response.code).to.be.oneOf([200, 201]);",
            "  const response = pm.response.json();",
            "  pm.collectionVariables.set('mailbox_id', response.data.id);",
            "  pm.collectionVariables.set('mailbox_token', response.data.token);",
            "  pm.globals.set('mailbox_address', response.data.address);",
            "});"
          ]
        }
      }
    ]
  ]
}
```

---

## 🔍 常见问题排查

### 1. 连接问题

#### 问题：无法连接到API
**症状**：
```
curl: (7) Failed to connect to localhost port 8080: Connection refused
```

**排查步骤**：
```bash
# 1. 检查服务是否运行
netstat -an | grep :8080

# 2. 检查进程
ps aux | grep tempmail

# 3. 检查防火墙
telnet localhost 8080

# 4. 检查日志
tail -f server.log
```

### 2. 认证问题

#### 问题：JWT令牌无效
**症状**：
```json
{
  "code": 401,
  "msg": "无效令牌"
}
```

**排查工具**：
```javascript
// JWT调试脚本
function decodeJWT(token) {
  try {
    const payload = token.split('.')[1];
    const decoded = JSON.parse(atob(payload));
    console.log('JWT Payload:', decoded);
    console.log('过期时间:', new Date(decoded.exp * 1000));
    console.log('当前时间:', new Date());
    console.log('是否过期:', Date.now() > decoded.exp * 1000);
  } catch (error) {
    console.error('JWT解码失败:', error);
  }
}

// 使用示例
decodeJWT('your.jwt.token.here');
```

### 3. 邮箱Token问题

#### 问题：邮箱Token无效
**症状**：
```json
{
  "code": 404,
  "msg": "邮箱不存在"
}
```

**排查步骤**：
```bash
# 1. 验证邮箱Token是否存在
curl -X GET "http://localhost:8080/v1/mailboxes/{id}" \
  -H "X-Mailbox-Token: {token}" -v

# 2. 检查邮箱是否被删除
# 3. 确认Token格式正确（AbCdEf123456格式）
# 4. 验证ID和Token是否匹配
```

### 4. CORS问题

#### 问题：跨域请求被拒绝
**症状**：
```
Access to fetch at 'http://localhost:8080/...' from origin '...' 
has been blocked by CORS policy
```

**解决方案**：
1. 检查后端CORS配置
2. 确保前端域名在允许列表中
3. 检查请求头格式

```javascript
// 前端请求配置示例
const config = {
  headers: {
    'Content-Type': 'application/json',
    // 如果需要认证
    'Authorization': `Bearer ${token}`
  }
};

// 开发环境代理配置（React）
const { createProxyMiddleware } = require('http-proxy-middleware');

module.exports = function(app) {
  app.use(
    '/api',
    createProxyMiddleware({
      target: 'http://localhost:8080',
      changeOrigin: true,
    })
  );
};
```

### 5. WebSocket连接问题

#### 问题：WebSocket连接失败
**症状**：
```
WebSocket connection failed: Error during WebSocket handshake: Unexpected response code: 400
```

**调试工具**：
```javascript
function debugWebSocketConnection(url, subscriptionData) {
  console.log('开始WebSocket调试...');
  
  const ws = new WebSocket(url);
  
  // 连接事件
  ws.onopen = () => {
    console.log('✅ WebSocket连接成功');
    console.log('发送订阅数据:', subscriptionData);
    ws.send(JSON.stringify(subscriptionData));
  };
  
  ws.onmessage = (event) => {
    console.log('📨 收到消息:', JSON.parse(event.data));
  };
  
  ws.onerror = (error) => {
    console.error('❌ WebSocket错误:', error);
  };
  
  ws.onclose = (event) => {
    console.log('🔌 WebSocket关闭:', {
      code: event.code,
      reason: event.reason,
      wasClean: event.wasClean
    });
  };
  
  return ws;
}

// 使用示例
const ws = debugWebSocketConnection(
  'ws://localhost:8080/v1/ws',
  {
    type: 'subscribe',
    mailboxId: 'your-mailbox-id',
    token: 'your-mailbox-token'
  }
);
```

---

## ⚡ 性能优化建议

### 1. 前端优化

#### 请求优化
```javascript
// 1. 请求合并
class BatchAPICalls {
  constructor() {
    this.pendingRequests = new Map();
    this.batchTimeout = null;
  }

  async batchRequest(requests) {
    // 批量处理多个API调用
    const promises = requests.map(request => this.makeRequest(request));
    return Promise.all(promises);
  }

  // 2. 请求缓存
  async getCachedData(key, fetchFn, ttl = 60000) {
    const cached = localStorage.getItem(key);
    const timestamp = localStorage.getItem(`${key}_timestamp`);
    
    if (cached && timestamp) {
      const age = Date.now() - parseInt(timestamp);
      if (age < ttl) {
        return JSON.parse(cached);
      }
    }
    
    const data = await fetchFn();
    localStorage.setItem(key, JSON.stringify(data));
    localStorage.setItem(`${key}_timestamp`, Date.now().toString());
    return data;
  }
}

// 3. 自动重试机制
class APIClient {
  async request(url, options = {}, retries = 3) {
    try {
      const response = await fetch(url, options);
      if (response.ok) return response;
      
      if (retries > 0 && response.status >= 500) {
        await this.delay(1000);
        return this.request(url, options, retries - 1);
      }
      
      throw new Error(`HTTP ${response.status}: ${response.statusText}`);
    } catch (error) {
      if (retries > 0) {
        await this.delay(1000);
        return this.request(url, options, retries - 1);
      }
      throw error;
    }
  }
  
  delay(ms) {
    return new Promise(resolve => setTimeout(resolve, ms));
  }
}
```

### 2. 后端优化

#### 连接池管理
```python
# Python aiohttp示例
import aiohttp
import asyncio

class TempMailClient:
    def __init__(self):
        self.session = None

    async def __aenter__(self):
        connector = aiohttp.TCPConnector(
            limit=100,  # 连接池大小
            limit_per_host=30,
            force_close=True,
            enable_cleanup_closed=True
        )
        
        timeout = aiohttp.ClientTimeout(total=30, connect=10)
        self.session = aiohttp.ClientSession(
            connector=connector,
            timeout=timeout
        )
        return self

    async def __aexit__(self, exc_type, exc_val, exc_tb):
        await self.session.close()
        await self.session.connector.close()
```

### 3. 监控和分析

#### 性能监控
```javascript
// 性能监控装饰器
function performanceMonitor(apiName) {
  return function(target, propertyKey, descriptor) {
    const originalMethod = descriptor.value;
    
    descriptor.value = async function(...args) {
      const start = performance.now();
      const startMemory = performance.memory?.usedJSHeapSize || 0;
      
      try {
        const result = await originalMethod.apply(this, args);
        const duration = performance.now() - start;
        
        // 记录性能数据
        console.log(`[性能监控] ${apiName}:`, {
          duration: `${duration.toFixed(2)}ms`,
          memory: ((performance.memory?.usedJSHeapSize - startMemory) / 1024 / 1024).toFixed(2) + 'MB'
        });
        
        // 发送到分析服务
        if (window.analytics) {
          window.analytics('timing', 'api_call', {
            name: apiName,
            value: duration
          });
        }
        
        return result;
      } catch (error) {
        const duration = performance.now() - start;
        console.error(`[性能监控] ${apiName} 失败:`, {
          duration: `${duration.toFixed(2)}ms`,
          error: error.message
        });
        throw error;
      }
    };
    
    return descriptor;
  };
}

// 使用示例
class TempMailAPI {
  @performanceMonitor('createMailbox')
  async createMailbox(options = {}) {
    // API实现
  }
}
```

---

## 📋 检查清单

### 测试前检查
- [ ] API服务是否正常运行
- [ ] 网络连接是否正常
- [ ] 认证Token是否有效
- [ ] 请求头格式是否正确

### 集成后检查
- [ ] 错误处理是否完善
- [ ] 超时设置是否合理
- [ ] 重试机制是否实现
- [ ] 性能监控是否添加

### 生产部署检查
- [ ] HTTPS配置是否正确
- [ ] 生产环境API URL是否正确
- [ ] 日志记录是否完善
- [ ] 监控告警是否设置

---

**文档版本**: v1.0  
**API版本**: v0.8.2-beta  
**最后更新**: 2025-01-15
