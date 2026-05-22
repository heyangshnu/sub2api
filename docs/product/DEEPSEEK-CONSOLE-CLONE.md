# 对标 DeepSeek 开放平台：控制台功能复制方案

> 参考：[DeepSeek 开放平台](https://platform.deepseek.com/)、[API 文档 - 模型与价格](https://api-docs.deepseek.com/zh-cn/quick_start/pricing)、[常见问题](https://api-docs.deepseek.com/zh-cn/faq)  
> 本文将官方控制台拆成 **用量信息 / API Keys / 充值 / 账单** 四块，对照 **cloudtoken.uk（Sub2API）** 现状，给出可复制的产品与实现清单。

---

## 一、DeepSeek 四块功能在做什么

| 模块 | 官方典型能力 | 用户目的 |
|------|----------------|----------|
| **用量信息** | 总览余额、本月消耗、按模型/按 Key 统计；图表；导出 CSV | 知道钱花在哪、哪把 Key 用得多 |
| **API Keys** | 创建 Key（仅显示一次）、列表、删除；建议分项目分 Key | 安全调用、分项目统计 |
| **充值** | 在线支付（支付宝/微信）、余额展示、企业汇款 | 账户有钱才能调 API |
| **账单** | 充值记录、消费明细、发票（企业） | 对账、报销、审计 |

官方扣费逻辑（文档）：`费用 = token 数 × 模型单价`，从充值余额或赠送余额扣，**优先扣赠送余额**。  
你的项目：统一 **USD 账户钱包**，另有 **月赠 $0.1**，展示上「充值余额」与「可消费总额」已分离（见 `ACCOUNT-WALLET.md`）。

---

## 二、与你项目的对照总表

| DeepSeek 模块 | 你项目 **已有** | 你项目 **缺失 / 可加强** | 建议路由 |
|---------------|-----------------|---------------------------|----------|
| 用量信息 | 账户余额、累计消费、请求次数；按 Key 近 14 日柱状图；请求日志页 | **账户级**用量总览；按**模型**拆分；按月筛选；导出 CSV | `/usage` |
| API Keys | 创建/列表/删/设置限额与 IP；连通性检测；首充解锁 | Key **命名规范**引导；批量禁用；创建时间筛选 | `/keys` |
| 充值 | Stripe Checkout 档位充值；月赠 | 充值**记录列表**（订单号/状态）；充值成功页优化 | `/topup` |
| 账单 | `account/transactions` 混合流水表 | **充值账单**与**消费账单**分 Tab；`payments` 表展示；发票（可选） | `/billing` |

当前实现：**四块全挤在 `/account` 一个页面**（`dashboard.tsx`），功能有但信息架构不像 DeepSeek 左侧菜单四分法。

---

## 三、分模块复制规格（可直接当需求写进迭代）

### 3.1 用量信息（对标 DeepSeek「用量信息」）

**DeepSeek 典型界面**

- 顶部：可用余额、本月已用、赠送余额（若有）
- 中部：时间范围 + 按模型 / 按 API Key 切换
- 图表：日/月消耗趋势
- 操作：导出用量 CSV（含分 Key 的 amount）

**复制到你项目 — 页面 `/usage`**

| 区块 | 内容 | 后端数据来源（已有/待做） |
|------|------|---------------------------|
| 概览卡片 | 充值结余 `balance`、可消费 `spendable_balance`、本月消费、本月请求数 | `GET /dashboard/me`；聚合 `account_ledger` 本月 `chat_consume`+`api_consume` |
| 趋势图 | 近 7/14/30 日消费（**账户级**，不仅单 Key） | 扩展 `GET /dashboard/usage-daily?scope=account` 或按 `user_id` 聚合 ledger |
| 按模型 | 表格：模型名、请求数、Token、金额 | 从 `request_logs` + `account_ledger` GROUP BY `model` |
| 按 Key | 下拉选 Key，柱状图（**已有**） | `GET /dashboard/usage-daily?key_id=` ✅ |
| 导出 | CSV：`日期, key_prefix, model, input_tokens, output_tokens, amount` | 新 `GET /dashboard/usage/export?month=2026-05` |

**与现状差异**

- 你已有 **按 Key** 14 日图；缺 **账户级总览** 和 **按模型**、**导出**。
- 月赠建议在用量页单独一行「本月赠送 +$0.1」，对标 DeepSeek「赠送余额」。

---

### 3.2 API Keys（对标 DeepSeek「API keys」）

**DeepSeek 典型界面**

- 列表：名称、Key 前缀、创建时间、状态
- 创建：一键生成，`sk-` 只显示一次
- 文档提示：不同项目用不同 Key 便于统计

**复制到你项目 — 页面 `/keys`**

| 能力 | 状态 | 说明 |
|------|------|------|
| 列表 + 前缀展示 | ✅ `ApiKeysCard` | 迁到独立页，保留表格 |
| 创建 Key + 仅一次明文 | ✅ | 保留复制/检测连通性 |
| 删除 / 设置限额 / IP 白名单 | ✅ | |
| 首充后才可创建 | ✅ `can_create_key` | DeepSeek 无此规则，你可保留 |
| Key 累计已用 / spend_limit | ✅ | 对标「单 Key 预算」 |
| 分 Key 用量跳转 | ✅ 链到 `/usage?key_id=` | 加强导航即可 |

**待做（体验对齐 DeepSeek）**

- 页顶简短说明 + 链接「OpenAI 兼容 Base URL：`https://api.cloudtoken.uk/v1`」
- 空状态：未首充时引导去 `/topup`（DeepSeek 是先去充值）
- 可选：Key 备注、最后使用时间列（数据在 `api_keys.last_used_at`，需写穿更新）

---

### 3.3 充值（对标 DeepSeek「充值」）

**DeepSeek 典型界面**

- 当前余额大字展示
- 快捷金额 + 支付宝/微信
- 充值说明、余额不过期

**复制到你项目 — 页面 `/topup`**

| 能力 | 状态 | 说明 |
|------|------|------|
| 余额展示 | ✅ `userProfile.balance`（充值结余） | 页顶加大字号 |
| 档位充值 $5–$100 | ✅ `TopupDialog` / Stripe | 从弹窗改为整页 |
| Stripe 支付 | ✅ `POST /dashboard/payment/checkout` | 替代支付宝/微信 |
| 月赠说明 | ✅ 配置 `ACCOUNT_MONTHLY_GRANT_USD` | 文案：每月登录赠送 $0.1 |
| 充值到账 | ✅ Webhook → `AccountTopup` | |
| 最近充值记录 | ⚠️ 弱 | 应用 `payments` 表 + `GET /dashboard/payments` |

**待做**

- `payments` 写穿（Stripe session → `payments` 行 + ledger）
- 充值页底部「最近 5 笔充值」：时间、金额、状态（pending/completed）

---

### 3.4 账单（对标 DeepSeek「账单」）

**DeepSeek 典型界面**

- Tab：**充值账单** / **消费账单**
- 充值：时间、渠道、金额、状态
- 消费：时间、模型、Token、金额；支持按月导出
- 企业：发票申请（你可 Phase 2）

**复制到你项目 — 页面 `/billing`**

| Tab | 内容 | API / 表 |
|-----|------|----------|
| 充值账单 | Stripe 订单列表 | `payments` + `type in (topup, admin_topup)` 的 ledger |
| 消费账单 | 扣费明细 | `GET /dashboard/account/transactions` 过滤 consume 类 |
| 全部流水 | 可选第三 Tab | 现有 transactions 表 ✅ |

**字段建议（消费账单行）**

| 列 | 来源 |
|----|------|
| 时间 | `created_at` |
| 类型 | `chat_consume` / `api_consume` |
| 模型 | `model` |
| Token | `input_tokens` / `output_tokens` |
| 金额 | `amount` |
| 余额 | `balance_after` |
| Key | `key_id` → 显示 `key_prefix` |

**待做**

- 前端 Tab 拆分（现在混在「最近交易」一张表）
- 类型中文映射：`topup`→充值，`monthly_grant`→月赠，`api_consume`→API 消费…
- 导出：`GET /dashboard/billing/export?type=consume&month=2026-05`

---

## 四、推荐信息架构（像 DeepSeek 左侧菜单）

```
控制台（登录后）
├── 首页 /              → 对话（保留，DeepSeek 无对等，是你的差异化）
├── 用量信息 /usage      → 新建
├── API Keys /keys       → 从 /account 拆出
├── 充值 /topup          → 从弹窗升级为页
├── 账单 /billing        → 新建
└── 个人中心 /profile    → 保留（昵称、改密）
```

`AppShell` 导航改为与上一致；`/account` 可 **301 重定向到 `/usage`** 或做总览仪表盘。

---

## 五、后端 API 增量（复制功能所需）

在现有 Dashboard API 上增加：

| 方法 | 路径 | 用途 |
|------|------|------|
| GET | `/dashboard/usage/summary` | 本月消费、请求数、按模型 Top N |
| GET | `/dashboard/usage/daily` | 扩展：`scope=account`（不限 key_id） |
| GET | `/dashboard/usage/export` | CSV 导出 |
| GET | `/dashboard/payments` | 充值订单列表（读 `payments` 表） |
| GET | `/dashboard/billing/export` | 账单 CSV |

Admin 已有：`/admin/users/:id/balance` 等，**不**对终端用户开放。

数据层：`account_ledger`、`request_logs`、`payments` 已在 schema v2 设计里（见 `DATABASE-SCHEMA.md`），导出类接口主要查 SQLite。

---

## 六、实施优先级（建议三期）

### P0 — 只改前端拆分（1–2 天）

- 新建 `/usage`、`/keys`、`/topup`、`/billing` 四个页面
- 从 `dashboard.tsx` 搬组件，改 `AppShell` 导航
- 账单 Tab 用现有 `getAccountTransactions` 客户端过滤类型

### P1 — 用量与账单增强（3–5 天）

- `usage/summary`、账户级日聚合
- 交易类型中文 + 充值/消费分 Tab
- Stripe `payments` 写穿 + `GET /dashboard/payments`

### P2 — 对标 DeepSeek 进阶（可选）

- CSV 导出
- 按模型用量表
- 发票/对公汇款（若你面向企业客户）

---

## 七、差异说明（不必强行一致）

| 项 | DeepSeek | 你的项目 |
|----|----------|----------|
| 支付 | 支付宝/微信（国内） | Stripe（海外 USD）✅ 合理 |
| 实名 | 需实名才能充值 | 你可邀请码 / 邮箱注册 |
| 定价页 | 官网公示单价 | 你可加 `/pricing` 读 `model_pricing` |
| 对话 | 无网页对话 | 你有 `/` 首页对话 ✅ 差异化 |
| Key 与余额 | 平台账户统一扣 | 账户钱包 + Key spend_limit ✅ 更细 |

---

## 八、总结

- **能力上**：你的后端已覆盖 DeepSeek 四块的 **70%**（Key、充值、流水、按 Key 用量、日志）。
- **体验上**：主要差 **页面拆分**、**账户级用量**、**账单分类**、**充值订单列表**、**导出**。
- **复制方式**：不是抄 UI 皮肤，而是 **同一套左侧四菜单 + 上表字段与 API**，保留 Stripe、月赠、首页对话等你已有优势。

确认后可按 **P0 → P1** 在 `dashboard/` 开分支实现；需要的话我可以直接从 P0 拆页面开始改代码。
