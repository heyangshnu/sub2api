# Sub2API Go Server

轻量级 API 聚合中转平台，提供 OpenAI 兼容格式的统一接口。

## 快速开始

### 1. 配置环境变量

```bash
cp .env.example .env
# 编辑 .env，填入你的 API Key
```

### 2. 启动服务

```bash
# 开发模式
go run ./cmd/server

# 或编译后运行
go build -o bin/sub2api ./cmd/server
./bin/sub2api
```

### 3. 创建 API Key

```bash
curl -X POST http://localhost:3000/admin/keys \
  -H "X-Admin-Key: sk-admin-sub2api-secret" \
  -H "Content-Type: application/json" \
  -d '{"user_id":"user001","balance":10}'
```

响应示例：
```json
{
  "key": "sk-sub2api-abc123...",
  "key_id": "xxx",
  "user_id": "user001",
  "balance": 10
}
```

### 4. 测试调用

```bash
# 非流式
curl -X POST http://localhost:3000/v1/chat/completions \
  -H "Authorization: Bearer sk-sub2api-xxx" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "claude-3-5-sonnet-20241022",
    "messages": [{"role": "user", "content": "Hello"}],
    "stream": false
  }'

# 流式
curl -X POST http://localhost:3000/v1/chat/completions \
  -H "Authorization: Bearer sk-sub2api-xxx" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "claude-3-5-sonnet-20241022",
    "messages": [{"role": "user", "content": "Hello"}],
    "stream": true
  }'

# 查看余额
curl http://localhost:3000/v1/usage \
  -H "Authorization: Bearer sk-sub2api-xxx"
```

## API 端点

| 方法 | 路径 | 说明 | 认证 |
|------|------|------|------|
| POST | /v1/chat/completions | Chat API | API Key |
| GET | /v1/models | 列出可用模型 | API Key |
| GET | /v1/usage | 查看使用统计 | API Key |
| POST | /admin/keys | 创建 API Key | Admin Key |
| GET | /admin/keys | 列出所有 Key | Admin Key |
| POST | /admin/keys/:id/topup | 充值 | Admin Key |
| GET | /health | 详细健康（Redis/SQLite ping） | 无 |
| GET | /health/ready | 就绪探针（Redis 异常时 503） | 无 |
| GET | /metrics | OpenMetrics 文本指标 | 无 |
| GET | /dashboard/usage-daily | 按天消费聚合（JWT，需 `key_id`） | JWT |
| GET | /dashboard/request-logs | 请求审计日志（JWT，需 `key_id`） | JWT |

## 可观测与对账

- 抓取指标：`curl -sS http://127.0.0.1:3000/metrics`（详见仓库 `docs/operations/OBSERVABILITY.md`）。
- 离线抽样对账（需 `REDIS_URL` 与可选 `./data/sub2api.db`）：

```bash
go run ./cmd/sub2api-check -key sk-sub2api-你的原始Key
```

## 支持的模型

### Anthropic
- claude-3-5-sonnet-20241022
- claude-3-5-haiku-20241022
- claude-3-opus-20240229

### OpenAI
- gpt-4o
- gpt-4o-mini

### DeepSeek
- deepseek-chat
- deepseek-coder

## 定价 (USD per 1K tokens)

| 模型 | Input | Output |
|------|-------|--------|
| claude-3-5-sonnet | $0.003 | $0.015 |
| claude-3-5-haiku | $0.001 | $0.005 |
| claude-3-opus | $0.015 | $0.075 |
| gpt-4o | $0.005 | $0.015 |
| gpt-4o-mini | $0.00015 | $0.0006 |
| deepseek-chat | $0.00014 | $0.00028 |

## 项目结构

```
sub2api-go/
├── cmd/server/          # HTTP 服务入口
├── cmd/sub2api-check/ # Redis 侧余额与流水抽样检查
├── internal/
│   ├── config/          # 配置管理
│   ├── handler/         # HTTP 处理器
│   ├── middleware/      # 中间件 (认证、CORS)
│   ├── model/           # 数据模型
│   ├── service/         # 业务逻辑 (Provider、Billing)
│   └── store/           # 存储层 (Memory → Redis)
├── migrations/          # 数据库迁移
├── .env                 # 环境变量
└── go.mod
```

## 开发路线

- [x] Phase 1: MVP - 单供应商可商用
- [x] Phase 2: Redis 原子扣费 + SQLite 用户数据
- [x] Phase 3: 用户自助 Dashboard（用量图、请求日志、健康与指标）
- [x] Phase 4: Stripe 支付集成（测试模式；生产需 Live Key）
- [ ] Phase 5: 多节点 + 托管 Redis + PG 迁移（见 `docs/operations/SCALING.md`）
