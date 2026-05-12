# FR-001: 认证与授权

> 版本：v1.0  
> 状态：Draft  
> 优先级：P0 (Critical)

---

## 1. 功能概述

提供基于 API Key 的认证机制，验证用户身份并关联到对应账户进行计费。

## 2. 用户故事

| ID | 角色 | 故事 | 验收标准 |
|----|------|------|----------|
| US-001-1 | 用户 | 我希望使用 API Key 调用接口 | 请求 Header 携带 Key 即可认证 |
| US-001-2 | 用户 | 我希望创建多个 API Key | 每个 Key 可独立设置额度和名称 |
| US-001-3 | 用户 | 我希望禁用泄露的 Key | 禁用后立即生效，请求返回 401 |
| US-001-4 | 管理员 | 我希望为用户创建初始 Key | 管理接口支持创建 Key 并设置初始余额 |

## 3. 功能需求

### 3.1 API Key 格式

```
sk-sub2api-{random_32_chars}
```

- 前缀：`sk-sub2api-` (固定，便于识别)
- 随机部分：32 位 alphanumeric (a-z, A-Z, 0-9)
- 总长度：42 字符

### 3.2 认证流程

```
请求到达
    │
    ▼
┌──────────────────┐
│ 提取 Authorization │
│ Header            │
└────────┬─────────┘
         │
         ▼
    ┌────────────┐
    │ 格式校验   │──Invalid──▶ 401 Unauthorized
    │ Bearer xxx │            {"error":"invalid_api_key"}
    └────┬───────┘
         │ Valid
         ▼
┌──────────────────┐
│ 计算 Key Hash    │
│ SHA256(key)      │
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│ 查询 Key 信息    │──Not Found──▶ 401 Unauthorized
│ Redis/DB         │              {"error":"api_key_not_found"}
└────────┬─────────┘
         │ Found
         ▼
    ┌────────────┐
    │ 状态检查   │──Disabled──▶ 401 Unauthorized
    │            │             {"error":"api_key_disabled"}
    └────┬───────┘
         │ Active
         ▼
    ┌────────────┐
    │ 过期检查   │──Expired──▶ 401 Unauthorized
    │            │            {"error":"api_key_expired"}
    └────┬───────┘
         │ Valid
         ▼
    认证通过，注入 user_id 到 context
```

### 3.3 Key 存储安全

| 字段 | 存储方式 | 说明 |
|------|----------|------|
| `key` (原文) | **不存储** | 仅创建时返回一次 |
| `key_hash` | SHA256 哈希 | 用于认证查询 |
| `key_prefix` | 明文前 12 字符 | 用于用户识别 `sk-sub2api-xxxx...` |

### 3.4 Key 管理操作

| 操作 | 端点 | 权限 | 说明 |
|------|------|------|------|
| 创建 Key | `POST /v1/keys` | 用户/管理员 | 返回完整 Key（仅一次） |
| 列出 Key | `GET /v1/keys` | 用户 | 返回 prefix + 元信息 |
| 更新 Key | `PATCH /v1/keys/{id}` | 用户 | 修改名称/额度 |
| 禁用 Key | `DELETE /v1/keys/{id}` | 用户 | 软删除，设置 status=disabled |
| 管理员创建 | `POST /admin/keys` | 管理员 | 可设置任意初始余额 |

## 4. 接口规格

### 4.1 用户认证（隐式）

所有 `/v1/*` 接口（除公开接口外）需携带认证头：

```http
Authorization: Bearer sk-sub2api-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
```

### 4.2 创建 API Key

**Request:**
```http
POST /v1/keys
Content-Type: application/json
Authorization: Bearer {existing_key}

{
  "name": "My App Key",
  "quota": 10.00,
  "expires_at": "2026-12-31T23:59:59Z"
}
```

**Response (201 Created):**
```json
{
  "id": "key_abc123",
  "key": "sk-sub2api-aBcDeFgHiJkLmNoPqRsTuVwXyZ012345",
  "key_prefix": "sk-sub2api-aBcD...",
  "name": "My App Key",
  "quota": 10.00,
  "used_amount": 0.00,
  "status": "active",
  "expires_at": "2026-12-31T23:59:59Z",
  "created_at": "2026-04-22T10:00:00Z"
}
```

> ⚠️ `key` 字段仅在创建响应中返回，之后无法再次获取。

### 4.3 管理员创建 Key

**Request:**
```http
POST /admin/keys
Content-Type: application/json
X-Admin-Key: {admin_secret}

{
  "user_id": "user_001",
  "name": "Initial Key",
  "balance": 100.00
}
```

**Response (201 Created):**
```json
{
  "id": "key_xyz789",
  "key": "sk-sub2api-...",
  "user_id": "user_001",
  "balance": 100.00,
  "created_at": "2026-04-22T10:00:00Z"
}
```

## 5. 错误码

| HTTP 状态 | 错误码 | 说明 | 场景 |
|-----------|--------|------|------|
| 401 | `invalid_api_key` | Key 格式错误 | 缺少 Bearer 前缀、长度不对 |
| 401 | `api_key_not_found` | Key 不存在 | Hash 查询无结果 |
| 401 | `api_key_disabled` | Key 已禁用 | status != active |
| 401 | `api_key_expired` | Key 已过期 | expires_at < now |
| 403 | `admin_key_required` | 需要管理员权限 | 访问 /admin/* |

## 6. 测试用例

### TC-AUTH-001: 有效 Key 认证成功

**Given:**
- 数据库中存在 Key，hash = SHA256("sk-sub2api-validkey123...")
- status = "active"，expires_at = NULL

**When:**
```http
GET /v1/models
Authorization: Bearer sk-sub2api-validkey123...
```

**Then:**
- HTTP 200
- 响应包含模型列表

---

### TC-AUTH-002: 无效 Key 格式

**Given:** 无

**When:**
```http
GET /v1/models
Authorization: Bearer invalid-key
```

**Then:**
- HTTP 401
- `{"error": {"code": "invalid_api_key", "message": "Invalid API key format"}}`

---

### TC-AUTH-003: Key 不存在

**Given:**
- 数据库中不存在对应 hash

**When:**
```http
GET /v1/models
Authorization: Bearer sk-sub2api-nonexistent...
```

**Then:**
- HTTP 401
- `{"error": {"code": "api_key_not_found", "message": "API key not found"}}`

---

### TC-AUTH-004: Key 已禁用

**Given:**
- Key 存在，status = "disabled"

**When:**
```http
GET /v1/models
Authorization: Bearer sk-sub2api-disabledkey...
```

**Then:**
- HTTP 401
- `{"error": {"code": "api_key_disabled", "message": "API key has been disabled"}}`

---

### TC-AUTH-005: Key 已过期

**Given:**
- Key 存在，status = "active"，expires_at = "2026-01-01T00:00:00Z" (已过期)

**When:**
```http
GET /v1/models
Authorization: Bearer sk-sub2api-expiredkey...
```

**Then:**
- HTTP 401
- `{"error": {"code": "api_key_expired", "message": "API key has expired"}}`

---

### TC-AUTH-006: 缺少认证头

**Given:** 无

**When:**
```http
GET /v1/models
(无 Authorization header)
```

**Then:**
- HTTP 401
- `{"error": {"code": "invalid_api_key", "message": "Missing Authorization header"}}`

---

*文档结束*
