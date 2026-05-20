# 账户钱包体系（USD）

> 定稿版本：v3 · 更新日期：2026-05-20

## 核心规则

- **币种**：对内、对外统一 **USD**。
- **月赠**：每用户每自然月 **$0.5**（仅首页 JWT 对话路径触发 `TryMonthlyGrant`）。
- **首充**：Stripe 成功 → 金额进入 **用户账户** → `has_paid=true` → 解锁创建 API Key（**不自动建 Key**）。
- **扣费**：首页对话、API Key 调用均扣 **账户余额**。
- **Key 上限**：可选 `spend_limit`（累计终身），须满足 `spend_limit <= 账户余额`（创建/更新时校验）。

## 路由（Dashboard）

| 路径 | 说明 |
|------|------|
| `/` | AI 对话（JWT，流式） |
| `/account` | 余额、充值、Key、用量、日志 |
| `/profile` | 昵称、改密 |

## API 摘要

| 方法 | 路径 |
|------|------|
| GET | `/auth/config` |
| GET/PATCH | `/dashboard/me` |
| POST | `/dashboard/change-password` |
| POST | `/dashboard/payment/checkout` |
| GET | `/dashboard/account/transactions` |
| POST | `/dashboard/chat/completions` |
| POST | `/dashboard/keys`（需 `has_paid`） |

## 配置（`.env`）

```env
ACCOUNT_MONTHLY_GRANT_USD=0.5
REQUIRE_PAYMENT_BEFORE_CREATE_KEY=true
CHAT_ENABLED_MODELS=deepseek-chat
STRIPE_SUCCESS_URL=https://cloudtoken.uk/account?paid=1
```

## Redis 键

- `account:balance:{user_id}` — 账户余额
- `account:grant:{user_id}:{YYYY-MM}` — 月赠已发放
- `key:spent:{key_id}` — Key 累计消费（上限用）

## 迁移

旧 `balance:{key_hash}` 余额需在上线前运行 `scripts/migrate_key_balance_to_account.go`（待补充）或手工对账。

详见 `HANDOVER.md` 账户钱包章节。
