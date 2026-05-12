# 单元测试规格

> 版本：v1.0  
> 状态：Draft  
> 覆盖率目标：≥ 80%

---

## 1. 测试模块划分

| 模块 | 文件 | 覆盖重点 |
|------|------|----------|
| Auth | `internal/middleware/auth_test.go` | Key 校验、状态检查 |
| Billing | `internal/service/billing_test.go` | 扣费计算、并发安全 |
| Router | `internal/service/router_test.go` | 模型路由、协议转换 |
| Provider | `internal/provider/*_test.go` | 各供应商适配 |
| Redis | `internal/service/redis_test.go` | 原子操作、Lua 脚本 |

---

## 2. Auth 模块测试

### 2.1 Key 格式校验

```go
func TestValidateKeyFormat(t *testing.T) {
    tests := []struct {
        name    string
        key     string
        wantErr error
    }{
        {
            name:    "valid key",
            key:     "sk-sub2api-aBcDeFgHiJkLmNoPqRsTuVwXyZ012345",
            wantErr: nil,
        },
        {
            name:    "missing prefix",
            key:     "aBcDeFgHiJkLmNoPqRsTuVwXyZ012345",
            wantErr: ErrInvalidKeyFormat,
        },
        {
            name:    "wrong prefix",
            key:     "sk-openai-aBcDeFgHiJkLmNoPqRsTuVwXyZ012345",
            wantErr: ErrInvalidKeyFormat,
        },
        {
            name:    "too short",
            key:     "sk-sub2api-short",
            wantErr: ErrInvalidKeyFormat,
        },
        {
            name:    "too long",
            key:     "sk-sub2api-aBcDeFgHiJkLmNoPqRsTuVwXyZ012345extra",
            wantErr: ErrInvalidKeyFormat,
        },
        {
            name:    "empty",
            key:     "",
            wantErr: ErrInvalidKeyFormat,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateKeyFormat(tt.key)
            assert.Equal(t, tt.wantErr, err)
        })
    }
}
```

### 2.2 Key 状态检查

```go
func TestCheckKeyStatus(t *testing.T) {
    now := time.Now()
    
    tests := []struct {
        name    string
        key     *ApiKey
        wantErr error
    }{
        {
            name: "active key",
            key: &ApiKey{
                Status:    "active",
                ExpiresAt: nil,
            },
            wantErr: nil,
        },
        {
            name: "disabled key",
            key: &ApiKey{
                Status: "disabled",
            },
            wantErr: ErrKeyDisabled,
        },
        {
            name: "expired key",
            key: &ApiKey{
                Status:    "active",
                ExpiresAt: timePtr(now.Add(-1 * time.Hour)),
            },
            wantErr: ErrKeyExpired,
        },
        {
            name: "active with future expiry",
            key: &ApiKey{
                Status:    "active",
                ExpiresAt: timePtr(now.Add(24 * time.Hour)),
            },
            wantErr: nil,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := CheckKeyStatus(tt.key, now)
            assert.Equal(t, tt.wantErr, err)
        })
    }
}
```

---

## 3. Billing 模块测试

### 3.1 费用计算

```go
func TestCalculateCost(t *testing.T) {
    model := &Model{
        ID:          "claude-3-5-sonnet-20241022",
        InputPrice:  3.00,  // $/1M tokens
        OutputPrice: 15.00,
        MarkupRate:  1.20,
    }

    tests := []struct {
        name         string
        inputTokens  int
        outputTokens int
        wantCost     float64
    }{
        {
            name:         "zero tokens",
            inputTokens:  0,
            outputTokens: 0,
            wantCost:     0.0,
        },
        {
            name:         "input only",
            inputTokens:  1000,
            outputTokens: 0,
            // (1000/1M) * 3.00 * 1.2 = 0.0036
            wantCost: 0.0036,
        },
        {
            name:         "output only",
            inputTokens:  0,
            outputTokens: 1000,
            // (1000/1M) * 15.00 * 1.2 = 0.018
            wantCost: 0.018,
        },
        {
            name:         "both input and output",
            inputTokens:  100,
            outputTokens: 200,
            // (100/1M)*3.00*1.2 + (200/1M)*15.00*1.2
            // = 0.00036 + 0.0036 = 0.00396
            wantCost: 0.00396,
        },
        {
            name:         "large request",
            inputTokens:  50000,
            outputTokens: 10000,
            // (50000/1M)*3.00*1.2 + (10000/1M)*15.00*1.2
            // = 0.18 + 0.18 = 0.36
            wantCost: 0.36,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            cost := CalculateCost(model, tt.inputTokens, tt.outputTokens)
            assert.InDelta(t, tt.wantCost, cost, 0.000001)
        })
    }
}
```

### 3.2 预扣金额策略

```go
func TestGetPreauthAmount(t *testing.T) {
    tests := []struct {
        name      string
        modelID   string
        wantAmount float64
    }{
        {
            name:       "cheap model - haiku",
            modelID:    "claude-3-haiku-20240307",
            wantAmount: 0.01,
        },
        {
            name:       "cheap model - mini",
            modelID:    "gpt-4o-mini",
            wantAmount: 0.01,
        },
        {
            name:       "cheap model - deepseek",
            modelID:    "deepseek-chat",
            wantAmount: 0.01,
        },
        {
            name:       "standard model - sonnet",
            modelID:    "claude-3-5-sonnet-20241022",
            wantAmount: 0.05,
        },
        {
            name:       "standard model - gpt4o",
            modelID:    "gpt-4o",
            wantAmount: 0.05,
        },
        {
            name:       "expensive model - opus",
            modelID:    "claude-3-opus-20240229",
            wantAmount: 0.20,
        },
        {
            name:       "unknown model - default",
            modelID:    "unknown-model",
            wantAmount: 0.05,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            amount := GetPreauthAmount(tt.modelID)
            assert.Equal(t, tt.wantAmount, amount)
        })
    }
}
```

### 3.3 余额检查

```go
func TestCheckBalance(t *testing.T) {
    minBalance := 0.001

    tests := []struct {
        name       string
        balance    float64
        preauth    float64
        wantErr    error
        wantDeduct float64
    }{
        {
            name:       "sufficient balance",
            balance:    10.0,
            preauth:    0.05,
            wantErr:    nil,
            wantDeduct: 0.05,
        },
        {
            name:       "exactly enough",
            balance:    0.051,
            preauth:    0.05,
            wantErr:    nil,
            wantDeduct: 0.05,
        },
        {
            name:       "partial deduct",
            balance:    0.03,
            preauth:    0.05,
            wantErr:    nil,
            wantDeduct: 0.029, // balance - minBalance
        },
        {
            name:       "below minimum",
            balance:    0.0005,
            preauth:    0.05,
            wantErr:    ErrInsufficientBalance,
            wantDeduct: 0,
        },
        {
            name:       "zero balance",
            balance:    0,
            preauth:    0.05,
            wantErr:    ErrInsufficientBalance,
            wantDeduct: 0,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            deduct, err := CheckBalance(tt.balance, tt.preauth, minBalance)
            assert.Equal(t, tt.wantErr, err)
            if err == nil {
                assert.InDelta(t, tt.wantDeduct, deduct, 0.0001)
            }
        })
    }
}
```

---

## 4. Router 模块测试

### 4.1 模型路由

```go
func TestRouteModel(t *testing.T) {
    tests := []struct {
        name         string
        modelID      string
        wantProvider string
        wantErr      error
    }{
        {
            name:         "claude sonnet",
            modelID:      "claude-3-5-sonnet-20241022",
            wantProvider: "anthropic",
            wantErr:      nil,
        },
        {
            name:         "gpt-4o",
            modelID:      "gpt-4o",
            wantProvider: "openai",
            wantErr:      nil,
        },
        {
            name:         "deepseek",
            modelID:      "deepseek-chat",
            wantProvider: "deepseek",
            wantErr:      nil,
        },
        {
            name:         "unknown model",
            modelID:      "gpt-5-super",
            wantProvider: "",
            wantErr:      ErrInvalidModel,
        },
        {
            name:         "empty model",
            modelID:      "",
            wantProvider: "",
            wantErr:      ErrInvalidModel,
        },
    }

    router := NewRouter(testModels)
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            provider, err := router.Route(tt.modelID)
            assert.Equal(t, tt.wantErr, err)
            if err == nil {
                assert.Equal(t, tt.wantProvider, provider.Name())
            }
        })
    }
}
```

### 4.2 OpenAI → Anthropic 转换

```go
func TestConvertOpenAIToAnthropic(t *testing.T) {
    tests := []struct {
        name    string
        input   *OpenAIChatRequest
        want    *AnthropicRequest
    }{
        {
            name: "simple message",
            input: &OpenAIChatRequest{
                Model: "claude-3-5-sonnet-20241022",
                Messages: []Message{
                    {Role: "user", Content: "Hello"},
                },
                MaxTokens: 100,
            },
            want: &AnthropicRequest{
                Model: "claude-3-5-sonnet-20241022",
                Messages: []AnthropicMessage{
                    {Role: "user", Content: "Hello"},
                },
                MaxTokens: 100,
                System:    "",
            },
        },
        {
            name: "with system message",
            input: &OpenAIChatRequest{
                Model: "claude-3-5-sonnet-20241022",
                Messages: []Message{
                    {Role: "system", Content: "You are helpful."},
                    {Role: "user", Content: "Hello"},
                },
                MaxTokens: 100,
            },
            want: &AnthropicRequest{
                Model: "claude-3-5-sonnet-20241022",
                Messages: []AnthropicMessage{
                    {Role: "user", Content: "Hello"},
                },
                MaxTokens: 100,
                System:    "You are helpful.",
            },
        },
        {
            name: "multi-turn conversation",
            input: &OpenAIChatRequest{
                Model: "claude-3-5-sonnet-20241022",
                Messages: []Message{
                    {Role: "system", Content: "Be concise."},
                    {Role: "user", Content: "Hi"},
                    {Role: "assistant", Content: "Hello!"},
                    {Role: "user", Content: "How are you?"},
                },
            },
            want: &AnthropicRequest{
                Model: "claude-3-5-sonnet-20241022",
                Messages: []AnthropicMessage{
                    {Role: "user", Content: "Hi"},
                    {Role: "assistant", Content: "Hello!"},
                    {Role: "user", Content: "How are you?"},
                },
                System: "Be concise.",
            },
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := ConvertOpenAIToAnthropic(tt.input)
            assert.Equal(t, tt.want.Model, got.Model)
            assert.Equal(t, tt.want.System, got.System)
            assert.Equal(t, tt.want.Messages, got.Messages)
        })
    }
}
```

### 4.3 Anthropic → OpenAI 响应转换

```go
func TestConvertAnthropicToOpenAI(t *testing.T) {
    tests := []struct {
        name  string
        input *AnthropicResponse
        want  *OpenAIChatResponse
    }{
        {
            name: "simple response",
            input: &AnthropicResponse{
                ID:    "msg_xxx",
                Model: "claude-3-5-sonnet-20241022",
                Content: []ContentBlock{
                    {Type: "text", Text: "Hello!"},
                },
                Usage: AnthropicUsage{
                    InputTokens:  10,
                    OutputTokens: 5,
                },
                StopReason: "end_turn",
            },
            want: &OpenAIChatResponse{
                ID:      "chatcmpl-xxx",
                Object:  "chat.completion",
                Model:   "claude-3-5-sonnet-20241022",
                Choices: []Choice{
                    {
                        Index: 0,
                        Message: Message{
                            Role:    "assistant",
                            Content: "Hello!",
                        },
                        FinishReason: "stop",
                    },
                },
                Usage: Usage{
                    PromptTokens:     10,
                    CompletionTokens: 5,
                    TotalTokens:      15,
                },
            },
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := ConvertAnthropicToOpenAI(tt.input)
            assert.Equal(t, tt.want.Model, got.Model)
            assert.Equal(t, tt.want.Choices[0].Message.Content, got.Choices[0].Message.Content)
            assert.Equal(t, tt.want.Usage.TotalTokens, got.Usage.TotalTokens)
        })
    }
}
```

---

## 5. Redis 模块测试

### 5.1 原子扣费 Lua 脚本

```go
func TestLuaCheckAndDeduct(t *testing.T) {
    ctx := context.Background()
    rdb := setupTestRedis(t)
    defer rdb.Close()

    tests := []struct {
        name         string
        initialBal   float64
        deductAmount float64
        minBalance   float64
        wantNewBal   float64
        wantDeducted float64
        wantErr      bool
    }{
        {
            name:         "normal deduct",
            initialBal:   10.0,
            deductAmount: 0.05,
            minBalance:   0.001,
            wantNewBal:   9.95,
            wantDeducted: 0.05,
            wantErr:      false,
        },
        {
            name:         "insufficient balance",
            initialBal:   0.0005,
            deductAmount: 0.05,
            minBalance:   0.001,
            wantNewBal:   0.0005, // unchanged
            wantDeducted: 0,
            wantErr:      true,
        },
        {
            name:         "partial deduct",
            initialBal:   0.03,
            deductAmount: 0.05,
            minBalance:   0.001,
            wantNewBal:   0.001,
            wantDeducted: 0.029,
            wantErr:      false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Setup
            userID := "test_user_" + tt.name
            key := "balance:" + userID
            rdb.Set(ctx, key, fmt.Sprintf("%.6f", tt.initialBal), 0)

            // Execute
            deducted, newBal, err := CheckAndDeduct(ctx, rdb, userID, tt.deductAmount, tt.minBalance)

            // Assert
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
                assert.InDelta(t, tt.wantDeducted, deducted, 0.0001)
                assert.InDelta(t, tt.wantNewBal, newBal, 0.0001)
            }
        })
    }
}
```

### 5.2 并发扣费测试

```go
func TestConcurrentDeduct(t *testing.T) {
    ctx := context.Background()
    rdb := setupTestRedis(t)
    defer rdb.Close()

    userID := "concurrent_test_user"
    initialBalance := 1.0
    deductPerRequest := 0.05
    concurrency := 100

    // Setup
    rdb.Set(ctx, "balance:"+userID, fmt.Sprintf("%.6f", initialBalance), 0)

    // Execute concurrent requests
    var wg sync.WaitGroup
    successCount := int32(0)
    failCount := int32(0)

    for i := 0; i < concurrency; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            _, _, err := CheckAndDeduct(ctx, rdb, userID, deductPerRequest, 0.001)
            if err == nil {
                atomic.AddInt32(&successCount, 1)
            } else {
                atomic.AddInt32(&failCount, 1)
            }
        }()
    }
    wg.Wait()

    // Assert
    finalBalance, _ := rdb.Get(ctx, "balance:"+userID).Float64()
    
    // 最多成功 20 次 (1.0 / 0.05 = 20)
    assert.LessOrEqual(t, int(successCount), 20)
    // 余额不能为负
    assert.GreaterOrEqual(t, finalBalance, 0.0)
    // 余额 = 初始 - 成功次数 * 扣费
    expectedBalance := initialBalance - float64(successCount)*deductPerRequest
    assert.InDelta(t, expectedBalance, finalBalance, 0.001)
    
    t.Logf("Success: %d, Fail: %d, Final Balance: %.6f", successCount, failCount, finalBalance)
}
```

---

## 6. 测试辅助函数

```go
// test_helpers.go

func setupTestRedis(t *testing.T) *redis.Client {
    rdb := redis.NewClient(&redis.Options{
        Addr: "localhost:6379",
        DB:   15, // 使用独立的测试 DB
    })
    
    t.Cleanup(func() {
        rdb.FlushDB(context.Background())
        rdb.Close()
    })
    
    return rdb
}

func timePtr(t time.Time) *time.Time {
    return &t
}

var testModels = map[string]*Model{
    "claude-3-5-sonnet-20241022": {
        ID:          "claude-3-5-sonnet-20241022",
        Provider:    "anthropic",
        InputPrice:  3.00,
        OutputPrice: 15.00,
        MarkupRate:  1.20,
    },
    "gpt-4o": {
        ID:          "gpt-4o",
        Provider:    "openai",
        InputPrice:  2.50,
        OutputPrice: 10.00,
        MarkupRate:  1.20,
    },
    "deepseek-chat": {
        ID:          "deepseek-chat",
        Provider:    "deepseek",
        InputPrice:  0.14,
        OutputPrice: 0.28,
        MarkupRate:  1.50,
    },
}
```

---

*文档结束*
