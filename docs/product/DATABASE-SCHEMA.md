# SQLite 持久化 schema（v2）

运行时余额以 **Redis** 为准；SQLite 通过 **写穿（Write-Through）** 与 Redis 同步，供查询、对账与后台 `reload-from-db`。

## 表一览

| 表 | 用途 |
|----|------|
| `users` | 注册用户 + 账户快照 |
| `account_ledger` | 充值 / 月赠 / 消费 / 后台调账流水 |
| `payments` | Stripe 订单（待接入写穿） |
| `api_keys` | Key 当前状态 |
| `request_logs` | 请求消耗明细 |
| `admin_audit_log` | 后台改余额/状态审计 |
| `sync_outbox` | SQLite 写失败重试队列 |
| `transactions` | 遗留表，启动时迁移到 `account_ledger` |

## Admin API（`X-Admin-Key`）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/admin/users` | 用户列表 |
| GET | `/admin/users/:id` | 用户详情 |
| PATCH | `/admin/users/:id/balance` | 调余额（先 Redis 后 SQLite） |
| PATCH | `/admin/users/:id/status` | 改状态 `active/disabled/banned/...` |
| POST | `/admin/users/:id/reload-from-db` | 应急：SQLite → Redis |

### 调余额示例

```bash
curl -X PATCH "https://api.example.com/admin/users/USER_ID/balance" \
  -H "X-Admin-Key: $ADMIN_KEY" \
  -H "Content-Type: application/json" \
  -d '{"adjust_amount": 5.0, "note": "客服补偿"}'
```

或指定目标余额：`{"spendable_balance": 10.5, "recharged_balance": 8.0}`

### 应急改库后恢复线上

```bash
curl -X POST "https://api.example.com/admin/users/USER_ID/reload-from-db" \
  -H "X-Admin-Key: $ADMIN_KEY"
```

## 同步机制

1. **写穿**：`AccountTopup`、扣费、`AppendRequestLog`、`CreateKey` 成功后立即写 SQLite。
2. **Syncer**（默认 1 分钟）：补同步 keys、ledger、users，并处理 `sync_outbox`。
3. **禁止**生产环境只改 SQLite 而不 `reload-from-db`（余额类字段）。

迁移文件：`sub2api-go/migrations/002_persistence.sql`（由 `SQLiteStore.migrate()` 自动应用）。
