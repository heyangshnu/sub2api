# 数据模型规格

> 版本：v1.0  
> 状态：Draft

---

## 1. 实体关系图 (ER Diagram)

```
┌──────────────────┐       1:N       ┌──────────────────┐
│      users       │────────────────▶│     api_keys     │
├──────────────────┤                 ├──────────────────┤
│ id (PK)          │                 │ id (PK)          │
│ email            │                 │ key_hash         │
│ balance          │                 │ key_prefix       │
│ level            │                 │ user_id (FK)     │
│ created_at       │                 │ name             │
│ updated_at       │                 │ quota            │
└──────────────────┘                 │ used_amount      │
        │                            │ status           │
        │                            │ expires_at       │
        │ 1:N                        │ created_at       │
        │                            └──────────────────┘
        │                                    │
        ▼                                    │ 1:N
┌──────────────────┐                         │
│   transactions   │◀────────────────────────┘
├──────────────────┤
│ id (PK)          │
│ user_id (FK)     │
│ api_key_id (FK)  │
│ type             │
│ amount           │
│ balance_after    │
│ model            │
│ input_tokens     │
│ output_tokens    │
│ request_id       │
│ metadata         │
│ created_at       │
└──────────────────┘

┌──────────────────┐
│      models      │
├──────────────────┤
│ id (PK)          │
│ provider         │
│ display_name     │
│ input_price      │
│ output_price     │
│ markup_rate      │
│ is_active        │
│ created_at       │
└──────────────────┘
```

---

## 2. 实体详细定义

### 2.1 users - 用户表

存储用户账户信息和余额。

| 字段 | 类型 | 约束 | 默认值 | 说明 |
|------|------|------|--------|------|
| id | VARCHAR(36) | PK | - | UUID v4 |
| email | VARCHAR(255) | UNIQUE, NULL | NULL | 用户邮箱（可选） |
| balance | DECIMAL(20,6) | NOT NULL | 0.000000 | 账户余额（USD） |
| level | INTEGER | NOT NULL | 1 | 用户等级 (1=普通, 2=VIP, 3=企业) |
| created_at | TIMESTAMP | NOT NULL | NOW() | 创建时间 |
| updated_at | TIMESTAMP | NOT NULL | NOW() | 更新时间 |

**业务规则:**
- `balance` 精度为 6 位小数，最小单位 0.000001 USD
- `balance` 在 Redis 中维护实时值，DB 为持久化备份
- 删除用户时软删除（添加 deleted_at 字段）

**索引:**
```sql
CREATE UNIQUE INDEX idx_users_email ON users(email) WHERE email IS NOT NULL;
```

---

### 2.2 api_keys - API Key 表

存储用户的 API 访问密钥。

| 字段 | 类型 | 约束 | 默认值 | 说明 |
|------|------|------|--------|------|
| id | VARCHAR(36) | PK | - | UUID v4 |
| key_hash | VARCHAR(64) | UNIQUE, NOT NULL | - | SHA256(原始Key) |
| key_prefix | VARCHAR(20) | NOT NULL | - | 展示用前缀 `sk-sub2api-xxxx...` |
| user_id | VARCHAR(36) | FK(users.id), NOT NULL | - | 所属用户 |
| name | VARCHAR(100) | NULL | NULL | Key 名称 |
| quota | DECIMAL(20,6) | NULL | NULL | 额度限制（NULL=无限） |
| used_amount | DECIMAL(20,6) | NOT NULL | 0.000000 | 已使用额度 |
| status | VARCHAR(20) | NOT NULL | 'active' | 状态 |
| expires_at | TIMESTAMP | NULL | NULL | 过期时间 |
| created_at | TIMESTAMP | NOT NULL | NOW() | 创建时间 |

**字段枚举值:**

| 字段 | 枚举值 | 说明 |
|------|--------|------|
| status | `active` | 正常可用 |
| status | `disabled` | 已禁用 |
| status | `expired` | 已过期（由定时任务更新） |

**业务规则:**
- `key_hash` 使用 SHA256，原始 Key 不存储
- `key_prefix` 格式: `sk-sub2api-{前4字符}...`
- 原始 Key 格式: `sk-sub2api-{32位随机字符}`
- `quota` 为 NULL 时表示无限额度
- 禁用 Key 后立即生效，不可恢复（只能创建新 Key）

**索引:**
```sql
CREATE UNIQUE INDEX idx_api_keys_hash ON api_keys(key_hash);
CREATE INDEX idx_api_keys_user ON api_keys(user_id);
CREATE INDEX idx_api_keys_status ON api_keys(status);
```

---

### 2.3 transactions - 流水表

记录所有账户变动。

| 字段 | 类型 | 约束 | 默认值 | 说明 |
|------|------|------|--------|------|
| id | VARCHAR(36) | PK | - | UUID v4 |
| user_id | VARCHAR(36) | FK(users.id), NOT NULL | - | 用户 ID |
| api_key_id | VARCHAR(36) | FK(api_keys.id), NULL | NULL | 使用的 Key |
| type | VARCHAR(20) | NOT NULL | - | 交易类型 |
| amount | DECIMAL(20,6) | NOT NULL | - | 金额（consume 为负） |
| balance_after | DECIMAL(20,6) | NOT NULL | - | 交易后余额 |
| model | VARCHAR(100) | NULL | NULL | 使用的模型 |
| input_tokens | INTEGER | NULL | NULL | 输入 token 数 |
| output_tokens | INTEGER | NULL | NULL | 输出 token 数 |
| request_id | VARCHAR(36) | NULL | NULL | 请求 ID |
| metadata | JSONB | NULL | NULL | 额外信息 |
| created_at | TIMESTAMP | NOT NULL | NOW() | 创建时间 |

**字段枚举值:**

| 字段 | 枚举值 | 说明 |
|------|--------|------|
| type | `topup` | 充值 |
| type | `consume` | 消费（amount 为负） |
| type | `refund` | 退款 |
| type | `adjust` | 调整（管理员操作） |

**metadata 结构示例:**

```json
// topup 类型
{
  "channel": "stripe",
  "payment_id": "pi_xxx",
  "note": "Manual topup"
}

// consume 类型
{
  "provider": "anthropic",
  "stream": true,
  "duration_ms": 1234
}

// refund 类型
{
  "reason": "upstream_error",
  "original_txn_id": "txn_xxx"
}
```

**索引:**
```sql
CREATE INDEX idx_transactions_user ON transactions(user_id);
CREATE INDEX idx_transactions_created ON transactions(created_at);
CREATE INDEX idx_transactions_type ON transactions(type);
CREATE INDEX idx_transactions_api_key ON transactions(api_key_id);
```

**分区策略（规模化后）:**
```sql
-- 按月分区
CREATE TABLE transactions_2026_04 PARTITION OF transactions
    FOR VALUES FROM ('2026-04-01') TO ('2026-05-01');
```

---

### 2.4 models - 模型配置表

存储支持的模型及定价。

| 字段 | 类型 | 约束 | 默认值 | 说明 |
|------|------|------|--------|------|
| id | VARCHAR(100) | PK | - | 模型 ID |
| provider | VARCHAR(50) | NOT NULL | - | 供应商 |
| display_name | VARCHAR(100) | NULL | NULL | 展示名称 |
| input_price | DECIMAL(10,6) | NOT NULL | - | 输入价格 ($/1M tokens) |
| output_price | DECIMAL(10,6) | NOT NULL | - | 输出价格 ($/1M tokens) |
| markup_rate | DECIMAL(5,2) | NOT NULL | 1.20 | 加价比例 |
| is_active | BOOLEAN | NOT NULL | true | 是否启用 |
| created_at | TIMESTAMP | NOT NULL | NOW() | 创建时间 |

**字段枚举值:**

| 字段 | 枚举值 | 说明 |
|------|--------|------|
| provider | `anthropic` | Anthropic (Claude) |
| provider | `openai` | OpenAI (GPT) |
| provider | `deepseek` | DeepSeek |

**初始化数据:**

```sql
INSERT INTO models (id, provider, display_name, input_price, output_price, markup_rate) VALUES
('claude-3-5-sonnet-20241022', 'anthropic', 'Claude 3.5 Sonnet', 3.00, 15.00, 1.20),
('claude-3-opus-20240229', 'anthropic', 'Claude 3 Opus', 15.00, 75.00, 1.20),
('claude-3-haiku-20240307', 'anthropic', 'Claude 3 Haiku', 0.25, 1.25, 1.20),
('gpt-4o', 'openai', 'GPT-4o', 2.50, 10.00, 1.20),
('gpt-4o-mini', 'openai', 'GPT-4o Mini', 0.15, 0.60, 1.20),
('deepseek-chat', 'deepseek', 'DeepSeek Chat', 0.14, 0.28, 1.50);
```

---

## 3. Redis 数据结构

### 3.1 用户余额

**Key 格式:** `balance:{user_id}`  
**类型:** STRING  
**值:** 余额数值（字符串格式，保留 6 位小数）  
**TTL:** 无（持久）

```
GET balance:user_001  →  "9.876543"
```

### 3.2 请求预扣锁

**Key 格式:** `preauth:{user_id}:{request_id}`  
**类型:** STRING  
**值:** 预扣金额  
**TTL:** 60 秒

```
SET preauth:user_001:req_abc123 "0.050000" EX 60
```

### 3.3 用户日用量

**Key 格式:** `usage:{user_id}:{date}`  
**类型:** HASH  
**TTL:** 7 天

```
HSET usage:user_001:2026-04-22 input_tokens 15000 output_tokens 8000 requests 42 cost "0.125000"
```

### 3.4 API Key 缓存

**Key 格式:** `key:{key_hash}`  
**类型:** HASH  
**TTL:** 5 分钟（写入时刷新）

```
HSET key:abc123hash user_id "user_001" status "active" quota "100" used "23.5"
```

### 3.5 Key 池状态

**Key 格式:** `provider:{provider}:keys`  
**类型:** SORTED SET  
**Score:** 最后使用时间戳  

```
ZADD provider:anthropic:keys 1713776400 "sk-ant-xxx"
```

### 3.6 Key 健康状态

**Key 格式:** `key_health:{provider}:{key_index}`  
**类型:** HASH  
**TTL:** 无

```
HSET key_health:anthropic:0 is_healthy 1 error_count 0 rate_limited 0 reset_at 0
```

---

## 4. 数据完整性约束

### 4.1 外键约束

```sql
ALTER TABLE api_keys 
  ADD CONSTRAINT fk_api_keys_user 
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE transactions 
  ADD CONSTRAINT fk_transactions_user 
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE transactions 
  ADD CONSTRAINT fk_transactions_api_key 
  FOREIGN KEY (api_key_id) REFERENCES api_keys(id) ON DELETE SET NULL;
```

### 4.2 检查约束

```sql
ALTER TABLE users 
  ADD CONSTRAINT chk_users_balance CHECK (balance >= 0);

ALTER TABLE users 
  ADD CONSTRAINT chk_users_level CHECK (level IN (1, 2, 3));

ALTER TABLE api_keys 
  ADD CONSTRAINT chk_api_keys_status CHECK (status IN ('active', 'disabled', 'expired'));

ALTER TABLE api_keys 
  ADD CONSTRAINT chk_api_keys_quota CHECK (quota IS NULL OR quota >= 0);

ALTER TABLE transactions 
  ADD CONSTRAINT chk_transactions_type CHECK (type IN ('topup', 'consume', 'refund', 'adjust'));

ALTER TABLE models 
  ADD CONSTRAINT chk_models_markup CHECK (markup_rate >= 1.0);
```

---

## 5. 数据迁移脚本

### 5.1 初始化 Schema

```sql
-- migrations/001_init.sql

-- 用户表
CREATE TABLE users (
    id          VARCHAR(36) PRIMARY KEY,
    email       VARCHAR(255) UNIQUE,
    balance     DECIMAL(20,6) NOT NULL DEFAULT 0,
    level       INTEGER NOT NULL DEFAULT 1,
    created_at  TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMP NOT NULL DEFAULT NOW(),
    
    CONSTRAINT chk_users_balance CHECK (balance >= 0),
    CONSTRAINT chk_users_level CHECK (level IN (1, 2, 3))
);

-- API Key 表
CREATE TABLE api_keys (
    id          VARCHAR(36) PRIMARY KEY,
    key_hash    VARCHAR(64) UNIQUE NOT NULL,
    key_prefix  VARCHAR(20) NOT NULL,
    user_id     VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name        VARCHAR(100),
    quota       DECIMAL(20,6),
    used_amount DECIMAL(20,6) NOT NULL DEFAULT 0,
    status      VARCHAR(20) NOT NULL DEFAULT 'active',
    expires_at  TIMESTAMP,
    created_at  TIMESTAMP NOT NULL DEFAULT NOW(),
    
    CONSTRAINT chk_api_keys_status CHECK (status IN ('active', 'disabled', 'expired')),
    CONSTRAINT chk_api_keys_quota CHECK (quota IS NULL OR quota >= 0)
);

CREATE INDEX idx_api_keys_user ON api_keys(user_id);
CREATE INDEX idx_api_keys_status ON api_keys(status);

-- 流水表
CREATE TABLE transactions (
    id            VARCHAR(36) PRIMARY KEY,
    user_id       VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    api_key_id    VARCHAR(36) REFERENCES api_keys(id) ON DELETE SET NULL,
    type          VARCHAR(20) NOT NULL,
    amount        DECIMAL(20,6) NOT NULL,
    balance_after DECIMAL(20,6) NOT NULL,
    model         VARCHAR(100),
    input_tokens  INTEGER,
    output_tokens INTEGER,
    request_id    VARCHAR(36),
    metadata      JSONB,
    created_at    TIMESTAMP NOT NULL DEFAULT NOW(),
    
    CONSTRAINT chk_transactions_type CHECK (type IN ('topup', 'consume', 'refund', 'adjust'))
);

CREATE INDEX idx_transactions_user ON transactions(user_id);
CREATE INDEX idx_transactions_created ON transactions(created_at);
CREATE INDEX idx_transactions_type ON transactions(type);

-- 模型配置表
CREATE TABLE models (
    id            VARCHAR(100) PRIMARY KEY,
    provider      VARCHAR(50) NOT NULL,
    display_name  VARCHAR(100),
    input_price   DECIMAL(10,6) NOT NULL,
    output_price  DECIMAL(10,6) NOT NULL,
    markup_rate   DECIMAL(5,2) NOT NULL DEFAULT 1.20,
    is_active     BOOLEAN NOT NULL DEFAULT true,
    created_at    TIMESTAMP NOT NULL DEFAULT NOW(),
    
    CONSTRAINT chk_models_markup CHECK (markup_rate >= 1.0)
);

-- 初始化模型数据
INSERT INTO models (id, provider, display_name, input_price, output_price, markup_rate) VALUES
('claude-3-5-sonnet-20241022', 'anthropic', 'Claude 3.5 Sonnet', 3.00, 15.00, 1.20),
('claude-3-opus-20240229', 'anthropic', 'Claude 3 Opus', 15.00, 75.00, 1.20),
('claude-3-haiku-20240307', 'anthropic', 'Claude 3 Haiku', 0.25, 1.25, 1.20),
('gpt-4o', 'openai', 'GPT-4o', 2.50, 10.00, 1.20),
('gpt-4o-mini', 'openai', 'GPT-4o Mini', 0.15, 0.60, 1.20),
('deepseek-chat', 'deepseek', 'DeepSeek Chat', 0.14, 0.28, 1.50);
```

---

## 6. 数据同步策略

### 6.1 Redis ↔ DB 同步

| 数据 | 主存储 | 同步方向 | 触发条件 |
|------|--------|----------|----------|
| 用户余额 | Redis | Redis → DB | 每分钟定时 + 服务关闭前 |
| Key 信息 | DB | DB → Redis | 首次查询时缓存 |
| 用量统计 | Redis | Redis → DB | 每小时聚合 |

### 6.2 故障恢复

**Redis 故障恢复:**
1. 从 DB 加载最新余额到 Redis
2. 扫描未完成的预扣（preauth key）并清理
3. 重建 Key 缓存

**DB 故障恢复:**
1. 切换到只读模式（仅用 Redis 余额）
2. 流水暂存本地文件
3. DB 恢复后批量导入

---

*文档结束*
