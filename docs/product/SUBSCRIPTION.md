# 订阅档位（环境变量配置）

与 **直接充值（Stripe Topup）** 并存：

- **充值**：增加账户 USD 余额（按量扣费的资金来源）。
- **订阅**：决定 **本周期可用模型列表** + **本周期消费上限（USD）**；到期需续订。
- 调用时仍从 **账户余额** 扣费；订阅上限用 `subscription:*` 在 Redis 单独累计。

参考 [DeepSeek 开放平台](https://platform.deepseek.com/) 的分档思路，档位全部由 `.env` 配置，**无需改代码**即可调价、改模型列表。

---

## 环境变量

```env
# 是否启用订阅（false 时行为与旧版一致，仅用 CHAT_ENABLED_MODELS + 账户余额）
SUBSCRIPTIONS_ENABLED=true

# 每个订阅周期天数（默认 30）
SUBSCRIPTION_PERIOD_DAYS=30

# 档位定义（见下方格式）
SUBSCRIPTION_PLANS=free:0:0.5:0:deepseek-chat|basic:9.99:30:5:deepseek-chat,gpt-4o-mini|pro:29.99:150:20:deepseek-chat,gpt-4o-mini,claude-3-5-haiku-20241022
```

### `SUBSCRIPTION_PLANS` 格式

用 `|` 分隔多个档位，每个档位 5 段（`:` 分隔）：

```
档位ID : 月费USD : 周期消费上限USD : 开通赠送余额USD : 允许模型列表
```

- **月费 USD**：`0` 表示免费档，走接口直接开通、不跳转 Stripe。
- **周期消费上限**：本周期内累计消费（API + 首页对话）不得超过该值；`0` 表示不允许消费（仅适合占位）。
- **开通赠送余额**：支付成功或免费开通时，一次性打入账户余额（可选营销）。
- **允许模型**：逗号分隔，须与网关 `PROVIDERS` / 定价表一致。

### 示例档位说明

| ID | 月费 | 周期上限 | 赠送余额 | 模型 |
|----|------|----------|----------|------|
| free | $0 | $0.5 | $0 | deepseek-chat |
| basic | $9.99 | $30 | $5 | deepseek-chat, gpt-4o-mini |
| pro | $29.99 | $150 | $20 | + claude-3-5-haiku |

存在 **free** 档时，用户首次 `GET /dashboard/me` 会自动开通 free（便于试用）。

---

## API

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/auth/config` | 含 `subscriptions_enabled`、`subscription_plans` |
| GET | `/dashboard/me` | 含 `subscription` 当前周期状态 |
| GET | `/dashboard/subscription/plans` | 档位列表 |
| GET | `/dashboard/subscription` | 当前订阅 |
| POST | `/dashboard/subscription/checkout` | `{ "plan_id": "basic" }` → Stripe 或免费开通 |
| POST | `/dashboard/payment/checkout` | **直接充值**（不变） |

Webhook：`checkout.session.completed` 且 `metadata.type=subscription` → 激活档位 + 可选赠送余额。

---

## 校验规则（启用订阅后）

1. 用户须有 **未过期** 的订阅（或有 free 自动开通）。
2. 请求模型须在档位的 **allowed_models** 内。
3. 本周期 `spent + 预估费用` 不得超过 **monthly_spend_cap**。
4. 仍须 **账户余额充足**（与 Key spend_limit 等现有规则叠加）。

错误码：`subscription_required` / `subscription_cap_exceeded` / `model_not_allowed`。

---

## 与数据库表的关系

- **不改表结构**：订阅状态在 Redis `subscription:{user_id}`。
- 扣费流水仍在 `account_ledger`；可选后续把 `payments` 与订阅订单关联。

---

## 关闭订阅

```env
SUBSCRIPTIONS_ENABLED=false
```

恢复为仅 `CHAT_ENABLED_MODELS` + 账户钱包 + 直接充值。
