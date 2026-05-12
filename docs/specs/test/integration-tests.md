# 集成测试规格

> 版本：v1.0  
> 状态：Draft  
> 测试环境：Docker Compose (API + Redis + Mock Upstream)

---

## 1. 测试环境配置

### 1.1 Docker Compose

```yaml
# docker-compose.test.yml
version: "3.8"

services:
  api:
    build: .
    environment:
      - ENV=test
      - PORT=3000
      - REDIS_URL=redis://redis:6379/15
      - ADMIN_KEY=test-admin-key
      - ANTHROPIC_BASE_URL=http://mock-upstream:8080/anthropic
      - OPENAI_BASE_URL=http://mock-upstream:8080/openai
      - ANTHROPIC_API_KEYS=test-key-1,test-key-2
      - OPENAI_API_KEYS=test-key-1
    ports:
      - "3000:3000"
    depends_on:
      - redis
      - mock-upstream

  redis:
    image: redis:7-alpine
    command: redis-server --appendonly yes

  mock-upstream:
    image: mockserver/mockserver:latest
    environment:
      - MOCKSERVER_INITIALIZATION_JSON_PATH=/config/init.json
    volumes:
      - ./test/mocks:/config
```

### 1.2 Mock 上游配置

```json
// test/mocks/init.json
[
  {
    "httpRequest": {
      "method": "POST",
      "path": "/anthropic/v1/messages"
    },
    "httpResponse": {
      "statusCode": 200,
      "headers": {
        "Content-Type": ["application/json"]
      },
      "body": {
        "id": "msg_mock_001",
        "type": "message",
        "role": "assistant",
        "content": [{"type": "text", "text": "Hello from mock!"}],
        "model": "claude-3-5-sonnet-20241022",
        "usage": {"input_tokens": 10, "output_tokens": 5}
      }
    }
  }
]
```

---

## 2. E2E 测试用例

### 2.1 完整调用流程

#### TC-E2E-001: 完整调用闭环

**目标:** 验证「创建用户 → 调用 API → 扣费 → 查余额」完整流程

**Steps:**

```bash
# Step 1: 创建用户并获取 Key
RESPONSE=$(curl -s -X POST http://localhost:3000/admin/keys \
  -H "X-Admin-Key: test-admin-key" \
  -H "Content-Type: application/json" \
  -d '{"user_id": "e2e_user_001", "balance": 10.00}')

API_KEY=$(echo $RESPONSE | jq -r '.key')
echo "API Key: $API_KEY"

# Step 2: 查询初始余额
BALANCE_BEFORE=$(curl -s http://localhost:3000/v1/usage \
  -H "Authorization: Bearer $API_KEY" | jq '.balance')
echo "Balance before: $BALANCE_BEFORE"  # Expected: 10.0

# Step 3: 调用 Chat API
curl -s -X POST http://localhost:3000/v1/chat/completions \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "claude-3-5-sonnet-20241022",
    "messages": [{"role": "user", "content": "Hello"}],
    "max_tokens": 100
  }'

# Step 4: 查询调用后余额
BALANCE_AFTER=$(curl -s http://localhost:3000/v1/usage \
  -H "Authorization: Bearer $API_KEY" | jq '.balance')
echo "Balance after: $BALANCE_AFTER"

# Step 5: 验证扣费
# Expected: balance_after < balance_before
# Expected: balance_after ≈ 10.0 - 0.00396 (based on mock usage)
```

**验收标准:**
- [ ] Step 1: 返回 201，包含完整 API Key
- [ ] Step 2: 余额 = 10.00
- [ ] Step 3: 返回 200，包含 AI 响应
- [ ] Step 4: 余额 < 10.00
- [ ] Step 5: 扣费金额符合 token 消耗计算

---

#### TC-E2E-002: 余额不足拒绝

**Steps:**

```bash
# Step 1: 创建低余额用户
RESPONSE=$(curl -s -X POST http://localhost:3000/admin/keys \
  -H "X-Admin-Key: test-admin-key" \
  -d '{"user_id": "e2e_user_002", "balance": 0.0005}')
API_KEY=$(echo $RESPONSE | jq -r '.key')

# Step 2: 尝试调用 API
RESULT=$(curl -s -w "\n%{http_code}" -X POST http://localhost:3000/v1/chat/completions \
  -H "Authorization: Bearer $API_KEY" \
  -d '{"model": "claude-3-5-sonnet-20241022", "messages": [{"role": "user", "content": "Hi"}]}')

HTTP_CODE=$(echo "$RESULT" | tail -n1)
BODY=$(echo "$RESULT" | head -n-1)

echo "HTTP Code: $HTTP_CODE"  # Expected: 402
echo "Error: $(echo $BODY | jq '.error.code')"  # Expected: insufficient_balance
```

**验收标准:**
- [ ] HTTP 状态码 = 402
- [ ] error.code = "insufficient_balance"
- [ ] 余额未变化

---

#### TC-E2E-003: 无效 Key 拒绝

**Steps:**

```bash
# 使用无效 Key 调用
RESULT=$(curl -s -w "\n%{http_code}" http://localhost:3000/v1/chat/completions \
  -H "Authorization: Bearer sk-sub2api-invalid12345678901234567890" \
  -d '{"model": "claude-3-5-sonnet-20241022", "messages": [{"role": "user", "content": "Hi"}]}')

HTTP_CODE=$(echo "$RESULT" | tail -n1)
echo "HTTP Code: $HTTP_CODE"  # Expected: 401
```

**验收标准:**
- [ ] HTTP 状态码 = 401
- [ ] error.code = "api_key_not_found"

---

### 2.2 流式响应测试

#### TC-E2E-004: 流式响应完整性

**Steps:**

```bash
# 创建用户
RESPONSE=$(curl -s -X POST http://localhost:3000/admin/keys \
  -H "X-Admin-Key: test-admin-key" \
  -d '{"user_id": "e2e_stream_user", "balance": 10.00}')
API_KEY=$(echo $RESPONSE | jq -r '.key')

# 流式调用
curl -s -N http://localhost:3000/v1/chat/completions \
  -H "Authorization: Bearer $API_KEY" \
  -d '{
    "model": "claude-3-5-sonnet-20241022",
    "messages": [{"role": "user", "content": "Count 1 to 5"}],
    "stream": true
  }' | while read line; do
    echo "$line"
    # 检查每行格式
    if [[ $line == data:* ]]; then
        # 验证 JSON 格式
        echo "$line" | sed 's/^data: //' | jq . > /dev/null 2>&1
        if [ $? -ne 0 ] && [[ "$line" != "data: [DONE]" ]]; then
            echo "FAIL: Invalid JSON"
            exit 1
        fi
    fi
done
```

**验收标准:**
- [ ] 响应 Content-Type = text/event-stream
- [ ] 每个 chunk 以 `data: ` 开头
- [ ] 最后一行是 `data: [DONE]`
- [ ] usage 在最后一个有效 chunk 中返回
- [ ] 扣费基于 usage 中的 token 数

---

### 2.3 并发安全测试

#### TC-E2E-005: 并发扣费不超扣

**测试脚本:**

```bash
#!/bin/bash
# test_concurrent.sh

ADMIN_KEY="test-admin-key"
INITIAL_BALANCE=1.0
CONCURRENCY=100
EXPECTED_MAX_SUCCESS=20  # 1.0 / 0.05 = 20

# 创建测试用户
RESPONSE=$(curl -s -X POST http://localhost:3000/admin/keys \
  -H "X-Admin-Key: $ADMIN_KEY" \
  -d "{\"user_id\": \"concurrent_user\", \"balance\": $INITIAL_BALANCE}")
API_KEY=$(echo $RESPONSE | jq -r '.key')

echo "Testing $CONCURRENCY concurrent requests with balance $INITIAL_BALANCE"

# 并发请求
SUCCESS=0
FAIL=0

for i in $(seq 1 $CONCURRENCY); do
  (
    CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST http://localhost:3000/v1/chat/completions \
      -H "Authorization: Bearer $API_KEY" \
      -d '{"model":"claude-3-5-sonnet-20241022","messages":[{"role":"user","content":"Hi"}]}')
    echo $CODE
  ) &
done | while read code; do
  if [ "$code" = "200" ]; then
    ((SUCCESS++))
  else
    ((FAIL++))
  fi
  echo "Success: $SUCCESS, Fail: $FAIL"
done

wait

# 检查最终余额
FINAL_BALANCE=$(curl -s http://localhost:3000/v1/usage \
  -H "Authorization: Bearer $API_KEY" | jq '.balance')

echo "Final balance: $FINAL_BALANCE"
echo "Expected: >= 0"

# 验证
if (( $(echo "$FINAL_BALANCE < 0" | bc -l) )); then
  echo "FAIL: Negative balance detected (overdraft)"
  exit 1
fi

echo "PASS: No overdraft"
```

**验收标准:**
- [ ] 最终余额 ≥ 0
- [ ] 成功请求数 ≤ (初始余额 / 预扣金额)
- [ ] 无数据不一致

---

### 2.4 上游故障处理

#### TC-E2E-006: 上游超时退回预扣

**配置 Mock 超时:**

```json
{
  "httpRequest": {
    "method": "POST",
    "path": "/anthropic/v1/messages",
    "headers": {
      "X-Test-Timeout": ["true"]
    }
  },
  "httpResponse": {
    "delay": {
      "timeUnit": "SECONDS",
      "value": 120
    }
  }
}
```

**Steps:**

```bash
# 记录初始余额
BALANCE_BEFORE=$(curl -s http://localhost:3000/v1/usage \
  -H "Authorization: Bearer $API_KEY" | jq '.balance')

# 发送会超时的请求
curl -s -m 5 -X POST http://localhost:3000/v1/chat/completions \
  -H "Authorization: Bearer $API_KEY" \
  -H "X-Test-Timeout: true" \
  -d '{"model":"claude-3-5-sonnet-20241022","messages":[{"role":"user","content":"Hi"}]}'

# 等待预扣退回
sleep 2

# 检查余额
BALANCE_AFTER=$(curl -s http://localhost:3000/v1/usage \
  -H "Authorization: Bearer $API_KEY" | jq '.balance')

echo "Before: $BALANCE_BEFORE, After: $BALANCE_AFTER"
```

**验收标准:**
- [ ] 返回 504 Gateway Timeout
- [ ] 余额恢复到请求前状态
- [ ] 无 consume 类型流水记录

---

#### TC-E2E-007: 上游 500 错误处理

**配置 Mock 返回 500:**

```json
{
  "httpRequest": {
    "method": "POST",
    "path": "/anthropic/v1/messages",
    "headers": {
      "X-Test-Error": ["500"]
    }
  },
  "httpResponse": {
    "statusCode": 500,
    "body": {"error": "Internal Server Error"}
  }
}
```

**验收标准:**
- [ ] 返回 502 Bad Gateway 或 504
- [ ] 余额恢复
- [ ] error.code = "upstream_error"

---

### 2.5 Key 管理测试

#### TC-E2E-008: Key 禁用后立即生效

**Steps:**

```bash
# Step 1: 创建 Key
RESPONSE=$(curl -s -X POST http://localhost:3000/admin/keys \
  -H "X-Admin-Key: test-admin-key" \
  -d '{"user_id": "key_test_user", "balance": 10.00}')
API_KEY=$(echo $RESPONSE | jq -r '.key')
KEY_ID=$(echo $RESPONSE | jq -r '.id')

# Step 2: 验证 Key 可用
curl -s http://localhost:3000/v1/models -H "Authorization: Bearer $API_KEY"
# Expected: 200

# Step 3: 禁用 Key
curl -s -X DELETE http://localhost:3000/v1/keys/$KEY_ID \
  -H "Authorization: Bearer $API_KEY"

# Step 4: 验证 Key 不可用
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:3000/v1/models \
  -H "Authorization: Bearer $API_KEY")
echo "HTTP Code after disable: $HTTP_CODE"  # Expected: 401
```

**验收标准:**
- [ ] 禁用后立即返回 401
- [ ] error.code = "api_key_disabled"

---

### 2.6 充值测试

#### TC-E2E-009: 管理员充值

**Steps:**

```bash
# 查询充值前余额
BALANCE_BEFORE=$(curl -s http://localhost:3000/v1/usage \
  -H "Authorization: Bearer $API_KEY" | jq '.balance')

# 充值
curl -s -X POST http://localhost:3000/admin/topup \
  -H "X-Admin-Key: test-admin-key" \
  -d '{"user_id": "e2e_user_001", "amount": 5.00}'

# 查询充值后余额
BALANCE_AFTER=$(curl -s http://localhost:3000/v1/usage \
  -H "Authorization: Bearer $API_KEY" | jq '.balance')

echo "Before: $BALANCE_BEFORE, After: $BALANCE_AFTER"
# Expected: After = Before + 5.00
```

**验收标准:**
- [ ] 返回 200
- [ ] 余额增加正确金额
- [ ] 有 topup 类型流水记录

---

## 3. 测试报告模板

```markdown
# Sub2API 集成测试报告

**测试时间:** YYYY-MM-DD HH:MM  
**测试环境:** Docker Compose v3.8  
**测试版本:** vX.X.X  

## 测试结果汇总

| 测试用例 | 状态 | 耗时 | 备注 |
|----------|------|------|------|
| TC-E2E-001 | ✅ PASS | 1.2s | |
| TC-E2E-002 | ✅ PASS | 0.3s | |
| TC-E2E-003 | ✅ PASS | 0.2s | |
| TC-E2E-004 | ✅ PASS | 2.1s | |
| TC-E2E-005 | ✅ PASS | 5.3s | 100 并发 |
| TC-E2E-006 | ✅ PASS | 6.0s | |
| TC-E2E-007 | ✅ PASS | 1.1s | |
| TC-E2E-008 | ✅ PASS | 0.5s | |
| TC-E2E-009 | ✅ PASS | 0.4s | |

**通过率:** 9/9 (100%)

## 已知问题

无

## 性能数据

- 单节点 QPS: ~850
- P99 延迟: 45ms (不含上游)
- 内存占用: 128MB (空闲)
```

---

*文档结束*
