# FR-002: 计费与余额管理

> 版本：v1.0  
> 状态：Draft  
> 优先级：P0 (Critical)

---

## 1. 功能概述

实现基于 Token 的实时计费系统，支持预扣费、实际扣费、余额查询，确保并发安全不超扣。

## 2. 用户故事

| ID | 角色 | 故事 | 验收标准 |
|----|------|------|----------|
| US-002-1 | 用户 | 我希望调用前知道是否有足够余额 | 余额不足时返回 402 |
| US-002-2 | 用户 | 我希望扣费基于实际用量 | 扣费金额 = 实际 tokens × 单价 |
| US-002-3 | 用户 | 我希望查询实时余额 | 余额数据延迟 < 1s |
| US-002-4 | 用户 | 我希望查看消费明细 | 每次调用有流水记录 |
| US-002-5 | 管理员 | 我希望为用户充值 | 充值后余额立即增加 |

## 3. 功能需求

### 3.1 计费流程

```
                        请求开始
                            │
                            ▼
                    ┌───────────────┐
                    │   余额预检    │
                    │ balance >= min│
                    └───────┬───────┘
                            │
               ┌────────────┴────────────┐
               │                         │
          余额充足                    余额不足
               │                         │
               ▼                         ▼
       ┌───────────────┐         ┌───────────────┐
       │    预扣费     │         │   返回 402    │
       │ DECRBY 预估   │         │ Insufficient  │
       └───────┬───────┘         └───────────────┘
               │
               ▼
       ┌───────────────┐
       │   转发请求    │
       │  到上游 API   │
       └───────┬───────┘
               │
       ┌───────┴───────┐
       │               │
    成功返回        请求失败
       │               │
       ▼               ▼
┌─────────────┐  ┌─────────────┐
│ 解析 usage  │  │  退回预扣   │
│ 计算实际费用│  │ INCRBY full │
└──────┬──────┘  └──────┬──────┘
       │                │
       ▼                ▼
┌─────────────┐  ┌─────────────┐
│  调整扣费   │  │  不记流水   │
│ diff=预扣-实│  │  或记失败   │
└──────┬──────┘  └─────────────┘
       │
       ▼
┌─────────────┐
│  记录流水   │
│ 异步写入 DB │
└─────────────┘
```

### 3.2 价格配置

#### 3.2.1 模型定价表

| 模型 | 供应商 | Input ($/1M tokens) | Output ($/1M tokens) | 加价率 |
|------|--------|---------------------|----------------------|--------|
| claude-3-5-sonnet-20241022 | Anthropic | 3.00 | 15.00 | 1.2x |
| claude-3-opus-20240229 | Anthropic | 15.00 | 75.00 | 1.2x |
| claude-3-haiku-20240307 | Anthropic | 0.25 | 1.25 | 1.2x |
| gpt-4o | OpenAI | 2.50 | 10.00 | 1.2x |
| gpt-4o-mini | OpenAI | 0.15 | 0.60 | 1.2x |
| deepseek-chat | DeepSeek | 0.14 | 0.28 | 1.5x |

#### 3.2.2 用户实际价格计算

```
用户价格 = 上游价格 × 加价率 (markup_rate)

示例：Claude 3.5 Sonnet
- Input: $3.00 × 1.2 = $3.60 / 1M tokens
- Output: $15.00 × 1.2 = $18.00 / 1M tokens
```

#### 3.2.3 单次请求费用计算

```
cost = (input_tokens / 1,000,000) × input_price × markup_rate
     + (output_tokens / 1,000,000) × output_price × markup_rate
```

### 3.3 预扣费策略

| 参数 | 值 | 说明 |
|------|-----|------|
| 最小余额 | $0.001 | 低于此值拒绝请求 |
| 预扣金额 | $0.05 | 默认预扣（可按模型调整） |
| 预扣超时 | 60s | 超时后自动退回 |

#### 3.3.1 预扣金额分级

| 模型类别 | 预扣金额 | 说明 |
|----------|----------|------|
| 便宜模型 (haiku, mini, deepseek) | $0.01 | 减少资金占用 |
| 标准模型 (sonnet, 4o) | $0.05 | 默认 |
| 昂贵模型 (opus) | $0.20 | 避免超扣风险 |

### 3.4 并发安全

使用 Redis Lua 脚本保证原子性：

```lua
-- check_and_deduct.lua
-- KEYS[1] = balance:{user_id}
-- ARGV[1] = 预扣金额
-- ARGV[2] = 最小余额

local balance = tonumber(redis.call('GET', KEYS[1]) or '0')
local amount = tonumber(ARGV[1])
local min_balance = tonumber(ARGV[2])

if balance < min_balance then
    return {-1, balance}  -- 余额不足
end

if balance < amount then
    -- 余额够最小但不够预扣，扣除全部可用
    amount = balance - min_balance
end

local new_balance = balance - amount
redis.call('SET', KEYS[1], tostring(new_balance))
return {amount, new_balance}  -- 返回实际预扣金额和新余额
```

### 3.5 流水记录

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| id | UUID | Y | 流水 ID |
| user_id | string | Y | 用户 ID |
| api_key_id | string | Y | 使用的 Key |
| type | enum | Y | topup/consume/refund/adjust |
| amount | decimal(20,6) | Y | 金额（consume 为负） |
| balance_after | decimal(20,6) | Y | 交易后余额 |
| model | string | N | 使用的模型 |
| input_tokens | int | N | 输入 token 数 |
| output_tokens | int | N | 输出 token 数 |
| request_id | string | N | 请求 ID |
| created_at | timestamp | Y | 创建时间 |

## 4. 接口规格

### 4.1 查询余额

**Request:**
```http
GET /v1/usage
Authorization: Bearer {api_key}
```

**Response (200 OK):**
```json
{
  "balance": 9.876543,
  "currency": "USD",
  "usage_today": {
    "requests": 42,
    "input_tokens": 15000,
    "output_tokens": 8000,
    "cost": 0.123456
  },
  "usage_month": {
    "requests": 500,
    "input_tokens": 200000,
    "output_tokens": 100000,
    "cost": 1.500000
  }
}
```

### 4.2 查询流水

**Request:**
```http
GET /v1/transactions?limit=20&offset=0&type=consume
Authorization: Bearer {api_key}
```

**Response (200 OK):**
```json
{
  "data": [
    {
      "id": "txn_abc123",
      "type": "consume",
      "amount": -0.003456,
      "balance_after": 9.996544,
      "model": "claude-3-5-sonnet-20241022",
      "input_tokens": 100,
      "output_tokens": 200,
      "created_at": "2026-04-22T10:00:00Z"
    }
  ],
  "total": 42,
  "limit": 20,
  "offset": 0
}
```

### 4.3 管理员充值

**Request:**
```http
POST /admin/topup
Content-Type: application/json
X-Admin-Key: {admin_secret}

{
  "user_id": "user_001",
  "amount": 10.00,
  "note": "Manual topup"
}
```

**Response (200 OK):**
```json
{
  "user_id": "user_001",
  "amount": 10.00,
  "balance_before": 5.00,
  "balance_after": 15.00,
  "transaction_id": "txn_xyz789"
}
```

## 5. 错误码

| HTTP 状态 | 错误码 | 说明 | 场景 |
|-----------|--------|------|------|
| 402 | `insufficient_balance` | 余额不足 | balance < min_balance |
| 402 | `quota_exceeded` | Key 额度超限 | key.used >= key.quota |
| 500 | `billing_error` | 计费系统错误 | Redis 不可用等 |

## 6. 测试用例

### TC-BILL-001: 正常扣费

**Given:**
- 用户余额 = 10.000000 USD
- 模型 = claude-3-5-sonnet-20241022
- 定价: input = $3.60/1M, output = $18.00/1M

**When:**
- POST /v1/chat/completions
- 请求成功，usage: input_tokens=100, output_tokens=200

**Then:**
- 扣费 = (100/1M)×3.60 + (200/1M)×18.00 = 0.000360 + 0.003600 = 0.003960 USD
- 余额 = 10.000000 - 0.003960 = 9.996040 USD
- 流水记录: type=consume, amount=-0.003960

---

### TC-BILL-002: 余额不足拒绝

**Given:**
- 用户余额 = 0.0005 USD (低于 min_balance 0.001)

**When:**
```http
POST /v1/chat/completions
```

**Then:**
- HTTP 402
- `{"error": {"code": "insufficient_balance", "message": "Insufficient balance"}}`
- 余额不变 = 0.0005 USD

---

### TC-BILL-003: 请求失败退回预扣

**Given:**
- 用户余额 = 10.000000 USD
- 预扣金额 = 0.05 USD

**When:**
- POST /v1/chat/completions
- 预扣成功，余额变为 9.950000 USD
- 上游返回 500 错误

**Then:**
- 退回预扣 0.05 USD
- 余额恢复 = 10.000000 USD
- 无 consume 类型流水（或记录 failed 流水）

---

### TC-BILL-004: 并发扣费不超扣

**Given:**
- 用户余额 = 1.000000 USD
- 每次请求预估消耗 = 0.05 USD

**When:**
- 同时发起 100 个并发请求

**Then:**
- 成功请求数 ≈ 20 (1.0 / 0.05)
- 最终余额 ≥ 0 USD（绝不为负）
- 无任何请求出现超扣

---

### TC-BILL-005: 流式响应中断计费

**Given:**
- 用户余额 = 10.000000 USD
- 预扣金额 = 0.05 USD

**When:**
- POST /v1/chat/completions (stream=true)
- 预扣成功
- 流式传输中途客户端断开（已传输 output_tokens=50）

**Then:**
- 按实际传输 tokens 计费（非预扣全额）
- 扣费基于 usage 或估算的已传输量
- 多扣部分退回

---

### TC-BILL-006: 管理员充值

**Given:**
- 用户 user_001 余额 = 5.000000 USD

**When:**
```http
POST /admin/topup
X-Admin-Key: valid-admin-key
{"user_id": "user_001", "amount": 10.00}
```

**Then:**
- HTTP 200
- 余额 = 15.000000 USD
- 流水记录: type=topup, amount=+10.00

---

### TC-BILL-007: Key 额度限制

**Given:**
- Key 配置: quota = 5.00 USD, used_amount = 4.99 USD
- 本次请求预估消耗 = 0.05 USD

**When:**
```http
POST /v1/chat/completions
Authorization: Bearer {this_key}
```

**Then:**
- HTTP 402
- `{"error": {"code": "quota_exceeded", "message": "API key quota exceeded"}}`

---

*文档结束*
