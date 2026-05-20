# Dashboard API（账户钱包）

Base: `https://api.cloudtoken.uk`（生产）

## 公开

- `GET /auth/config` — `monthly_grant_usd`, `chat_enabled_models`, `currency: USD`

## JWT（`Authorization: Bearer <token>`）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/dashboard/me` | 余额、`has_paid`、`can_create_key` |
| PATCH | `/dashboard/me` | `{ "name": "..." }` |
| POST | `/dashboard/change-password` | 改密 |
| POST | `/dashboard/payment/checkout` | `{ "amount": 10 }` → Stripe |
| GET | `/dashboard/account/transactions` | 账户流水 |
| POST | `/dashboard/chat/completions` | 对话（`stream: true`） |
| POST | `/dashboard/keys` | 需 `has_paid`；可选 `spend_limit` |
| PATCH | `/dashboard/keys/:id` | 含 `spend_limit` |

## API Key（`/v1/*`）

扣费从**用户账户**扣除；若 Key 设 `spend_limit`，累计消费不得超过该上限。

## Webhook

`checkout.session.completed` 且 `metadata.type=account_topup` → 账户入账 + `has_paid=true`。
