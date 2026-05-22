# 控制台完整优化方案（v1）

> 整合三项需求：**公开浏览 + 操作时登录**、**对标 DeepSeek 四模块**、**订阅档位（env 配置）**。  
> 更新日期：2026-05 · 状态：后端订阅已落地，**前端与信息架构待实施**。

---

## 一、目标与原则

| 原则 | 说明 |
|------|------|
| 先进用量页 | 去掉「进站即登录/注册全屏」，默认进入 **用量信息** |
| 公开可逛 | 未登录可打开各模块页面，看布局、说明、空状态/演示数据 |
| 操作需登录 | 充值、建 Key、发对话、改资料等 **点击时弹窗** 引导登录或注册 |
| 注册不变 | 现有邮箱/邀请码/验证码流程、接口 **不改** |
| 登录后看本人数据 | JWT 后展示真实余额、用量、Key、账单、订阅 |
| 对标 DeepSeek | 左侧（或顶栏）**用量 / Keys / 充值 / 账单**，另加 **订阅** |
| 配置驱动订阅 | 档位、上限、模型范围继续用 `.env` 的 `SUBSCRIPTION_PLANS` |

**不改动** SQLite 表结构（订阅状态在 Redis；见 `SUBSCRIPTION.md`、`DATABASE-SCHEMA.md`）。

---

## 二、现状对照

| 能力 | 后端 | 前端 |
|------|------|------|
| 账户钱包 + Stripe 充值 | ✅ | ✅ `TopupDialog` 在 `/account` |
| 订阅档位 + 校验 | ✅ | ❌ 无页面、无 API 封装 |
| 用量 / Key / 账单 | ✅ 部分 API | ⚠️ 全挤在 `/account` |
| 首页对话 | ✅ | ✅ `/` 需登录才显示 |
| 进站登录墙 | — | ✅ `/` 未登录只显示 `LoginForm` |
| DeepSeek 四栏导航 | — | ❌ |
| 公开浏览 + 操作拦截 | — | ❌ |

---

## 三、信息架构（改后）

### 3.1 路由表

| 路径 | 名称 | 未登录 | 登录后 |
|------|------|--------|--------|
| `/` | 用量信息（**默认首页**） | 静态概览 + 功能说明 + 示例图/占位 | 真实余额、本月消费、14 日图、订阅摘要 |
| `/keys` | API Keys | 说明 + 空表格样式 | `ApiKeysCard` 完整能力 |
| `/topup` | 充值 | 档位说明 +「登录后充值」 | Stripe 充值 |
| `/billing` | 账单 | Tab 结构预览（假数据/空） | 充值/消费分 Tab + 分页 |
| `/subscription` | 订阅 | 展示 `auth/config` 各档位价格与模型 | 当前订阅 + 订阅/续费按钮 |
| `/chat` | 对话（可选保留） | 界面预览或简短说明 | 原 `ChatPage`（从 `/` 迁出） |
| `/profile` | 个人中心 | 只读说明 | 昵称、改密 |
| `/login` | 登录 | 独立页（可选，主要靠弹窗） | 重定向 `/` |
| `/register` | 注册 | **保持现有 `LoginForm` 注册 Tab/流程** | — |

**废弃/重定向**

- 原 `/` 登录墙 → 删除；`/` = 用量页。
- `/account` → **301 或链接重定向到 `/`**，避免双入口。

### 3.2 顶栏（全局 `ConsoleShell`）

```
┌─────────────────────────────────────────────────────────────────┐
│ Sub2API    [用量] [API Keys] [充值] [账单] [订阅] [对话?]     [登录] 或 [邮箱 ▾ 退出] │
└─────────────────────────────────────────────────────────────────┘
```

- **未登录**：右侧主按钮「登录」；点击打开 **AuthDialog**（Tab：登录 | 注册）。
- **已登录**：显示邮箱/昵称下拉：个人中心、退出；不再占满屏登录。

### 3.3 侧栏（可选，桌面端）

与 DeepSeek 一致时，左侧固定：

- 用量信息  
- API Keys  
- 充值  
- 账单  
- 订阅  

移动端：顶栏 Tab 或汉堡菜单。

---

## 四、认证与拦截（核心交互）

### 4.1 `AuthContext` 扩展

```ts
isAuthenticated: boolean   // JWT 为主；不再把 api_key 当「已登录控制台」
isGuest: boolean           // !isAuthenticated
openAuthDialog(tab?: 'login' | 'register')
requireAuth(action: () => void | Promise<void>)
```

- **游客模式**：`isAuthenticated === false` 仍可渲染各 `page.tsx`。
- **恢复会话**：仅 JWT（`sub2api_token`）；API Key 仅登录后绑定用于用量查询，不作为进站凭证。

### 4.2 `AuthDialog` 组件

- 复用 `LoginForm` 内 **登录 / 注册** 表单逻辑（抽成 `LoginPanel` / `RegisterPanel`）。
- 注册流程、字段、邀请码、验证码：**与原 `login-form.tsx` 一致**。
- 登录成功：关闭弹窗、`refreshProfile()`、可选 toast「登录成功」。
- 支持 `?auth=register` 深链打开注册 Tab。

### 4.3 需拦截的操作（`requireAuth`）

| 模块 | 操作 | 未登录行为 |
|------|------|------------|
| 用量 | 刷新个人数据、切换 Key 图表 | 弹窗登录 |
| Keys | 创建 / 删除 / 设置 / 检测连通性 | 弹窗登录 |
| 充值 | 选择金额、去支付 | 弹窗登录 |
| 账单 | 翻页、导出 | 弹窗登录 |
| 订阅 | 订阅某档位、免费开通 | 弹窗登录 |
| 对话 | 发送消息 | 弹窗登录 |
| 个人中心 | 改密、保存昵称 | 弹窗登录 |

**不拦截**：切换导航、阅读说明、查看定价表（订阅档位可从 `GET /auth/config` 公开拉取）。

### 4.4 公开接口（无需 JWT）

| 接口 | 用途 |
|------|------|
| `GET /auth/config` | 月赠、币种、`subscription_plans`、`subscriptions_enabled` |
| `GET /health` | 存活探测 |

其余 `GET /dashboard/*` 保持 JWT；前端游客页用 **占位 UI**，不调用（避免 401 报错）。

---

## 五、分模块页面规格

### 5.1 用量信息 `/`（首页）

**游客态**

- 标题 + 产品一句话说明。  
- 三张卡片（灰色占位）：账户余额、本月消费、请求次数。  
- 图表区：示例柱状图或「登录后查看您的用量」。  
- 主按钮：「登录查看我的用量」→ `openAuthDialog('login')`。

**登录态**

- 数据来自 `GET /dashboard/me` + `usage-daily` + 可选 `usage/summary`（P1 后端）。  
- 展示：`balance`、`spendable_balance`、订阅 `remaining_cap_usd`（若启用订阅）。  
- 按 Key 14 日图（现有逻辑从 `dashboard.tsx` 迁入）。  
- 链接：请求日志 `/account/logs` 或迁到 `/usage/logs`。

### 5.2 API Keys `/keys`

**游客态**：API 使用说明、Base URL `https://api.cloudtoken.uk/v1`、表格列名预览。  
**登录态**：迁入 `ApiKeysCard`；创建 Key 走 `requireAuth`。

### 5.3 充值 `/topup`

**游客态**：充值档位卡片（$5–$100）+ 文案「登录后支付」。  
**登录态**：迁入 `TopupDialog` 内容改为整页；Stripe 返回仍用 `payment/success`。

与 **订阅** 区分文案：

- **充值** = 增加账户余额（按量扣费的资金）。  
- **订阅** = 开通档位（模型 + 周期上限）。

### 5.4 账单 `/billing`

**游客态**：充值账单 / 消费账单 两个 Tab 空表 + 字段说明。  
**登录态**：`getAccountTransactions`；按 `type` 分 Tab：

- 充值：`topup`、`admin_topup`、`subscription_grant`  
- 消费：`chat_consume`、`api_consume`  

类型显示中文（映射表在前端）。

### 5.5 订阅 `/subscription`（新增）

**游客态**：卡片列表展示 `auth/config.subscription_plans`（价格、上限、模型标签）。  
**登录态**：

- `GET /dashboard/subscription` 当前周期。  
- 按钮：订阅 / 续费 → `POST /dashboard/subscription/checkout`。  
- 免费档（`monthly_price_usd === 0`）直接激活。  
- 展示：已用 / 上限 / 到期日 / 可用模型列表。

### 5.6 对话 `/chat`（建议从首页迁出）

- 首页改为用量后，对话单独路由，避免游客进站困惑。  
- 游客：输入框 disabled +「登录后开始对话」。  
- 登录：现有 `ChatPage`；模型列表受订阅或 `chat_enabled_models` 约束。

### 5.7 注册 `/register`（可选独立路由）

- 与弹窗内注册 **同一组件**，便于邮件链接、推广落地页。  
- 流程不变：`send-register-code` → `register`。

---

## 六、前端工程拆分（实施清单）

### 6.1 新建/调整文件

```
dashboard/src/
  components/
    console-shell.tsx          # 顶栏导航 + 登录按钮
    auth-dialog.tsx            # 登录/注册弹窗
    auth/login-panel.tsx       # 从 login-form 抽出
    auth/register-panel.tsx
    require-auth-button.tsx    # 包装 onClick → requireAuth
  app/
    page.tsx                   # → 用量页 UsagePage
    keys/page.tsx
    topup/page.tsx
    billing/page.tsx
    subscription/page.tsx
    chat/page.tsx
    profile/page.tsx           # 保留，包 ConsoleShell
    account/page.tsx           # redirect → /
  lib/
    auth-context.tsx           # 扩展 guest + openAuthDialog
    api.ts                     # subscription* 方法
    transaction-labels.ts      # type → 中文
```

### 6.2 从 `dashboard.tsx` 拆解

| 原块 | 迁至 |
|------|------|
| StatCard + 用量图 | `usage-page.tsx` |
| ApiKeysCard | `keys-page.tsx` |
| TopupDialog | `topup-page.tsx` |
| 交易表格 | `billing-page.tsx` |
| 模型列表 | `usage-page` 或 `keys-page` 底部 |

### 6.3 `api.ts` 补充

```ts
getAuthConfig()
getSubscription()
listSubscriptionPlans()  // 或直接用 config
createSubscriptionCheckout(planId)
// UserProfile 增加 subscription 字段类型
```

---

## 七、后端配合（少量增量）

| 项 | 优先级 | 说明 |
|----|--------|------|
| 订阅 API | P0 | 已有，前端对接即可 |
| `GET /dashboard/usage/summary` | P1 | 账户级本月统计，用量首页 |
| `usage-daily?scope=account` | P1 | 不限 key_id 的日聚合 |
| `GET /dashboard/payments` | P2 | 充值账单 Tab |
| `GET /dashboard/usage/export` | P2 | CSV 导出 |

**无需**为本次 UX 改表。

---

## 八、环境变量（运营配置）

```env
# 订阅（见 SUBSCRIPTION.md）
SUBSCRIPTIONS_ENABLED=true
SUBSCRIPTION_PERIOD_DAYS=30
SUBSCRIPTION_PLANS=free:0:0.5:0:deepseek-chat|basic:9.99:30:5:deepseek-chat,gpt-4o-mini|pro:29.99:150:20:deepseek-chat,gpt-4o-mini,claude-3-5-haiku-20241022

# 未开订阅时对话模型
CHAT_ENABLED_MODELS=deepseek-chat

# 账户钱包（不变）
ACCOUNT_MONTHLY_GRANT_USD=0.1
REQUIRE_PAYMENT_BEFORE_CREATE_KEY=true
```

前端 `GET /auth/config` 在 **游客态** 拉取订阅档位用于 `/subscription` 公开展示。

---

## 九、实施阶段与工期（建议）

### Phase A — 壳子 + 登录改造（2–3 天）【优先】

1. `ConsoleShell` + 五栏导航  
2. `/` 用量游客页 + 登录态迁移  
3. `AuthDialog` + `requireAuth` + 去掉进站 `LoginForm`  
4. `/account` → `/` 重定向  
5. `api.ts` 订阅类型与方法（可先不接 UI）

**验收**：游客能点遍导航；点「充值」弹登录；登录后用量有真数据。

### Phase B — 四模块拆页（2 天）

1. `/keys`、`/topup`、`/billing` 拆页  
2. 交易类型中文、账单分 Tab  

**验收**：功能与现 `/account` 等价，无回归。

### Phase C — 订阅页（1 天）

1. `/subscription` 档位卡片 + 当前订阅  
2. 对接 `checkout`、支付成功回跳带 `subscription=1` 提示  

**验收**：`SUBSCRIPTIONS_ENABLED=true` 时能订阅 basic；模型/上限生效。

### Phase D — 对话与增强（1–2 天，可选）

1. `/chat` 迁出首页  
2. 用量账户级汇总 API（P1 后端）  
3. 导出 CSV（P2）  

### Phase E — 上线

按 [DEPLOY-PERSISTENCE-V2.md](../operations/DEPLOY-PERSISTENCE-V2.md) / [DEPLOY-FROM-LOCAL.md](../operations/DEPLOY-FROM-LOCAL.md) 发版；**仅前端变更时也要 `npm run build` + restart dashboard**。

---

## 十、流程图

```mermaid
flowchart TD
  Visit[访问 cloudtoken.uk] --> Usage[/ 用量信息]
  Usage --> Nav{点击导航}
  Nav --> Keys[/keys]
  Nav --> Topup[/topup]
  Nav --> Billing[/billing]
  Nav --> Sub[/subscription]
  Nav --> Chat[/chat]

  Keys --> Action{点击操作}
  Topup --> Action
  Billing --> Action
  Sub --> Action
  Chat --> Action
  Usage --> Action

  Action -->|未登录| Dialog[AuthDialog 登录/注册]
  Action -->|已登录| API[调用 Dashboard API]
  Dialog -->|注册成功/登录成功| API
```

---

## 十一、风险与注意

| 风险 | 缓解 |
|------|------|
| 游客调 Dashboard API 401 | 游客页不调需 JWT 的接口；仅 `auth/config` |
| 旧书签 `/account` | 重定向到 `/` |
| 订阅开启但无免费档 | 游客订阅页可看档；操作需登录后 checkout |
| 首充才能建 Key | 保留；在 Keys 页用文案说明 |
| Stripe 与订阅 success URL | `STRIPE_SUCCESS_URL` 建议指向 `/subscription?success=1` 或 `/topup?paid=1` |

---

## 十二、文档索引

| 文档 | 内容 |
|------|------|
| [DEEPSEEK-CONSOLE-CLONE.md](./DEEPSEEK-CONSOLE-CLONE.md) | 四模块字段级对照 |
| [SUBSCRIPTION.md](./SUBSCRIPTION.md) | 订阅 env 与 API |
| [ACCOUNT-WALLET.md](./ACCOUNT-WALLET.md) | 余额与充值规则 |
| [DATABASE-SCHEMA.md](./DATABASE-SCHEMA.md) | SQLite / 写穿 |
| [DEPLOY-FROM-LOCAL.md](../operations/DEPLOY-FROM-LOCAL.md) | 日常发版 |

---

## 十三、总结

| 维度 | 结论 |
|------|------|
| 登录改造 | 进站用量页 + 右上角登录弹窗 + 操作拦截；注册流程不变 |
| DeepSeek 复制 | 用量 / Keys / 充值 / 账单 四页 + 统一导航 |
| 订阅 | 后端已完成；前端 `/subscription` + config 展示待做 |
| 数据库 | 本方案 **不新增表** |
| 当前缺口 | **几乎全是前端与路由**；按 Phase A→C 约 **5–6 个工作日** 可上线主路径 |

确认本方案后，建议从 **Phase A** 开始改 `dashboard/`（我可按该文档直接落地代码）。
