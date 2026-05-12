# FR-003: 请求路由与转发

> 版本：v1.0  
> 状态：Draft  
> 优先级：P0 (Critical)

---

## 1. 功能概述

实现多供应商 API 的统一路由，将 OpenAI 兼容格式请求转发到对应上游供应商，支持流式/非流式响应透传。

## 2. 用户故事

| ID | 角色 | 故事 | 验收标准 |
|----|------|------|----------|
| US-003-1 | 用户 | 我希望用统一格式调用不同模型 | 同一接口支持 Claude/GPT/DeepSeek |
| US-003-2 | 用户 | 我希望流式输出实时显示 | SSE 事件无延迟透传 |
| US-003-3 | 用户 | 我希望自动选择可用供应商 | 某供应商故障时自动切换 |
| US-003-4 | 管理员 | 我希望配置多个 API Key 池 | 单供应商多 Key 负载均衡 |

## 3. 功能需求

### 3.1 模型路由映射

| 用户请求模型 | 路由目标 | 上游模型 |
|--------------|----------|----------|
| `claude-3-5-sonnet-20241022` | Anthropic | claude-3-5-sonnet-20241022 |
| `claude-3-opus-20240229` | Anthropic | claude-3-opus-20240229 |
| `claude-3-haiku-20240307` | Anthropic | claude-3-haiku-20240307 |
| `gpt-4o` | OpenAI | gpt-4o |
| `gpt-4o-mini` | OpenAI | gpt-4o-mini |
| `gpt-4-turbo` | OpenAI | gpt-4-turbo |
| `deepseek-chat` | DeepSeek | deepseek-chat |
| `deepseek-coder` | DeepSeek | deepseek-coder |

### 3.2 供应商配置

```yaml
providers:
  anthropic:
    base_url: "https://api.anthropic.com"
    api_version: "2023-06-01"
    keys:
      - "sk-ant-xxx"
      - "sk-ant-yyy"
    rate_limit: 60  # RPM per key
    timeout: 120s
    
  openai:
    base_url: "https://api.openai.com"
    keys:
      - "sk-xxx"
    rate_limit: 500
    timeout: 60s
    
  deepseek:
    base_url: "https://api.deepseek.com"
    keys:
      - "sk-xxx"
    rate_limit: 100
    timeout: 60s
```

### 3.3 路由流程

```
                    用户请求
                        │
                        ▼
            ┌───────────────────────┐
            │   解析 model 参数      │
            └───────────┬───────────┘
                        │
                        ▼
            ┌───────────────────────┐
            │  查询路由表 → Provider │
            └───────────┬───────────┘
                        │
        ┌───────────────┴───────────────┐
        │                               │
   模型存在                          模型不存在
        │                               │
        ▼                               ▼
┌───────────────┐               ┌───────────────┐
│  选择 Key     │               │   返回 400    │
│  (轮询/随机)  │               │ invalid_model │
└───────┬───────┘               └───────────────┘
        │
        ▼
┌───────────────┐
│  协议转换     │
│  (如需要)     │
└───────┬───────┘
        │
        ▼
┌───────────────┐
│  转发请求     │
│  到上游       │
└───────┬───────┘
        │
    ┌───┴───┐
    │       │
  成功     失败
    │       │
    ▼       ▼
┌───────┐ ┌─────────────┐
│ 透传  │ │ 重试/降级   │
│ 响应  │ │ 或返回错误  │
└───────┘ └─────────────┘
```

### 3.4 协议转换

#### 3.4.1 OpenAI → Anthropic 转换

**OpenAI 格式（入参）:**
```json
{
  "model": "claude-3-5-sonnet-20241022",
  "messages": [
    {"role": "system", "content": "You are helpful."},
    {"role": "user", "content": "Hello"}
  ],
  "max_tokens": 1000,
  "temperature": 0.7,
  "stream": true
}
```

**Anthropic 格式（转换后）:**
```json
{
  "model": "claude-3-5-sonnet-20241022",
  "system": "You are helpful.",
  "messages": [
    {"role": "user", "content": "Hello"}
  ],
  "max_tokens": 1000,
  "temperature": 0.7,
  "stream": true
}
```

**转换规则:**
| OpenAI | Anthropic | 说明 |
|--------|-----------|------|
| messages[role=system] | system (顶层) | 提取为独立字段 |
| max_tokens | max_tokens | 直接映射 |
| temperature | temperature | 直接映射 |
| top_p | top_p | 直接映射 |
| stop | stop_sequences | 重命名 |
| stream | stream | 直接映射 |

#### 3.4.2 Anthropic → OpenAI 响应转换

**Anthropic 响应:**
```json
{
  "id": "msg_xxx",
  "type": "message",
  "role": "assistant",
  "content": [{"type": "text", "text": "Hello!"}],
  "model": "claude-3-5-sonnet-20241022",
  "usage": {
    "input_tokens": 10,
    "output_tokens": 5
  }
}
```

**OpenAI 格式（转换后）:**
```json
{
  "id": "chatcmpl-xxx",
  "object": "chat.completion",
  "created": 1713776400,
  "model": "claude-3-5-sonnet-20241022",
  "choices": [{
    "index": 0,
    "message": {
      "role": "assistant",
      "content": "Hello!"
    },
    "finish_reason": "stop"
  }],
  "usage": {
    "prompt_tokens": 10,
    "completion_tokens": 5,
    "total_tokens": 15
  }
}
```

### 3.5 流式响应处理

#### 3.5.1 SSE 格式

```
data: {"id":"chatcmpl-xxx","object":"chat.completion.chunk","created":1713776400,"model":"claude-3-5-sonnet-20241022","choices":[{"index":0,"delta":{"content":"Hello"},"finish_reason":null}]}

data: {"id":"chatcmpl-xxx","object":"chat.completion.chunk","created":1713776400,"model":"claude-3-5-sonnet-20241022","choices":[{"index":0,"delta":{"content":"!"},"finish_reason":null}]}

data: {"id":"chatcmpl-xxx","object":"chat.completion.chunk","created":1713776400,"model":"claude-3-5-sonnet-20241022","choices":[{"index":0,"delta":{},"finish_reason":"stop"}],"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}}

data: [DONE]
```

#### 3.5.2 流式 Token 统计

流式响应中，usage 在最后一个 chunk（或 message_stop 事件）返回：

```go
// Anthropic stream events
// 1. message_start → 获取 input_tokens
// 2. content_block_delta → 累积内容
// 3. message_delta → 获取 output_tokens
// 4. message_stop → 结束
```

### 3.6 Key 池管理

#### 3.6.1 负载均衡策略

| 策略 | 说明 | 适用场景 |
|------|------|----------|
| round_robin | 轮询 | 默认，均匀分配 |
| random | 随机 | 简单，适合 Key 少 |
| least_used | 最少使用 | 精确控制配额 |

#### 3.6.2 Key 健康检查

```go
type KeyStatus struct {
    Key         string
    IsHealthy   bool
    LastUsed    time.Time
    ErrorCount  int
    RateLimited bool
    ResetAt     time.Time
}

// 标记 Key 不可用条件
// 1. 连续 3 次 5xx 错误
// 2. 返回 429 (Rate Limited)
// 3. 返回 401 (Key 失效)
```

### 3.7 故障处理

| 上游错误 | 处理策略 | 用户响应 |
|----------|----------|----------|
| 429 Rate Limited | 切换其他 Key，该 Key 冷却 | 透明重试 |
| 500 Server Error | 重试 1 次，仍失败则返回错误 | 504 Gateway Timeout |
| 401 Unauthorized | 标记 Key 失效，告警 | 503 Service Unavailable |
| Timeout | 返回超时错误 | 504 Gateway Timeout |
| 网络错误 | 重试 1 次 | 502 Bad Gateway |

## 4. 接口规格

### 4.1 Chat Completions

**Request:**
```http
POST /v1/chat/completions
Content-Type: application/json
Authorization: Bearer {api_key}

{
  "model": "claude-3-5-sonnet-20241022",
  "messages": [
    {"role": "user", "content": "Hello, how are you?"}
  ],
  "max_tokens": 1000,
  "temperature": 0.7,
  "stream": false
}
```

**Response (200 OK, non-stream):**
```json
{
  "id": "chatcmpl-abc123",
  "object": "chat.completion",
  "created": 1713776400,
  "model": "claude-3-5-sonnet-20241022",
  "choices": [{
    "index": 0,
    "message": {
      "role": "assistant",
      "content": "Hello! I'm doing well, thank you for asking. How can I assist you today?"
    },
    "finish_reason": "stop"
  }],
  "usage": {
    "prompt_tokens": 12,
    "completion_tokens": 20,
    "total_tokens": 32
  }
}
```

### 4.2 List Models

**Request:**
```http
GET /v1/models
Authorization: Bearer {api_key}
```

**Response (200 OK):**
```json
{
  "object": "list",
  "data": [
    {
      "id": "claude-3-5-sonnet-20241022",
      "object": "model",
      "created": 1713744000,
      "owned_by": "anthropic"
    },
    {
      "id": "gpt-4o",
      "object": "model",
      "created": 1713744000,
      "owned_by": "openai"
    }
  ]
}
```

## 5. 错误码

| HTTP 状态 | 错误码 | 说明 | 场景 |
|-----------|--------|------|------|
| 400 | `invalid_model` | 模型不存在 | 请求不支持的模型 |
| 400 | `invalid_request` | 请求格式错误 | 参数缺失/类型错误 |
| 502 | `upstream_error` | 上游错误 | 上游返回异常 |
| 503 | `service_unavailable` | 服务不可用 | 所有 Key 不可用 |
| 504 | `gateway_timeout` | 网关超时 | 上游响应超时 |

## 6. 测试用例

### TC-ROUTE-001: Claude 模型路由

**Given:**
- 配置了 Anthropic provider
- Key 池有可用 Key

**When:**
```http
POST /v1/chat/completions
{"model": "claude-3-5-sonnet-20241022", "messages": [{"role": "user", "content": "Hi"}]}
```

**Then:**
- 请求转发到 Anthropic API
- 响应格式为 OpenAI 兼容格式
- usage 字段包含 token 统计

---

### TC-ROUTE-002: 不存在的模型

**Given:** 无

**When:**
```http
POST /v1/chat/completions
{"model": "gpt-5-super", "messages": [...]}
```

**Then:**
- HTTP 400
- `{"error": {"code": "invalid_model", "message": "Model 'gpt-5-super' not found"}}`

---

### TC-ROUTE-003: 流式响应透传

**Given:**
- stream = true

**When:**
```http
POST /v1/chat/completions
{"model": "claude-3-5-sonnet-20241022", "messages": [...], "stream": true}
```

**Then:**
- Content-Type: text/event-stream
- 每个 chunk 为 `data: {...}\n\n` 格式
- 最后一行为 `data: [DONE]\n\n`
- usage 在最后一个有效 chunk 返回

---

### TC-ROUTE-004: Key 轮询

**Given:**
- Anthropic 配置 3 个 Key: [A, B, C]
- 负载均衡策略 = round_robin

**When:**
- 连续发起 6 个请求

**Then:**
- 请求使用的 Key 依次为: A, B, C, A, B, C

---

### TC-ROUTE-005: Rate Limit 自动切换

**Given:**
- Key A 返回 429 Rate Limited
- Key B 可用

**When:**
- 使用 Key A 请求 → 429
- 自动重试

**Then:**
- 切换到 Key B 重试
- 请求成功
- Key A 标记冷却，一段时间后恢复

---

### TC-ROUTE-006: system 消息转换

**Given:**
- 请求包含 system role message

**When:**
```http
POST /v1/chat/completions
{
  "model": "claude-3-5-sonnet-20241022",
  "messages": [
    {"role": "system", "content": "You are a helpful assistant."},
    {"role": "user", "content": "Hello"}
  ]
}
```

**Then:**
- 转发到 Anthropic 时，system 提取为顶层字段
- messages 中只保留 user/assistant 消息

---

*文档结束*
