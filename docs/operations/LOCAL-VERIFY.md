# 本地验证保姆级教程

在 Mac 上从零跑通「账户钱包 + 对话首页」全流程。预计 30–45 分钟（含 Stripe 测试配置）。

---

## 0. 你需要准备什么

| 工具 | 检查命令 | 说明 |
|------|----------|------|
| Go ≥ 1.21 | `go version` | 编译后端 |
| Node ≥ 20 | `node -v` | 跑 Dashboard |
| Redis（推荐） | `redis-cli ping` → `PONG` | 不用也可，会退化为内存库（重启丢数据） |
| DeepSeek Key | [platform.deepseek.com](https://platform.deepseek.com) | 对话必须，否则 chat 会 502 |
| Stripe 测试账号（测充值时） | Dashboard 测试模式 | 可选；也可用下文「跳过 Stripe」 |

项目根目录示例：

```text
/Users/heyang/Desktop/myProject/sub2api-full-code/
├── sub2api-go/     # 后端，端口 3000
└── dashboard/      # 前端，端口 3001（避免与后端冲突）
```

---

## 1. 开 3 个终端窗口

建议用 iTerm / Terminal 分屏，分别命名为 **Redis**、**API**、**Web**。

### 终端 A：Redis（推荐）

```bash
# 若未安装：brew install redis
redis-server
# 另开一行验证：redis-cli ping
```

没有 Redis 也能跑：后端会打印 `using memory store`，但**重启后用户/余额会清空**。

### 终端 B：Go 后端

```bash
cd /Users/heyang/Desktop/myProject/sub2api-full-code/sub2api-go

# 首次：拉依赖并编译
go mod download
go build -o bin/server ./cmd/server

# 配置 .env（见下一节）
./bin/server
# 或直接：go run ./cmd/server
```

成功标志：日志里有 `Listening on :3000`，且无 `Redis required` 致命错误。

健康检查：

```bash
curl -s http://127.0.0.1:3000/health | head -c 500
curl -s http://127.0.0.1:3000/auth/config
```

`auth/config` 应返回 `currency: USD`、`monthly_grant_usd: 0.5` 等字段。

### 终端 C：Next.js 前端

```bash
cd /Users/heyang/Desktop/myProject/sub2api-full-code/dashboard
npm install
npm run dev -- -p 3001
```

浏览器打开：**http://localhost:3001**

> 必须用 `-p 3001`，因为后端已占用 **3000**。

---

## 2. 配置文件（照抄再改密钥）

### 2.1 后端 `sub2api-go/.env`

在现有 `.env` 上确认或追加（**不要提交真实密钥到 Git**）：

```env
PORT=3000
APP_ENV=development

JWT_SECRET=local-dev-jwt-secret-change-me
ADMIN_KEY=sk-admin-sub2api-secret
INVITE_CODE=cloudtoken2026

REDIS_URL=redis://127.0.0.1:6379
ALLOW_MEMORY_STORE=true

# 必填：否则无法对话
DEEPSEEK_API_KEY=sk-你的deepseek密钥

# 本地建议关闭邮箱验证（海外 SMTP 也常失败）
EMAIL_VERIFY_ENABLED=false

# 账户钱包
ACCOUNT_MONTHLY_GRANT_USD=0.5
REQUIRE_PAYMENT_BEFORE_CREATE_KEY=true
CHAT_ENABLED_MODELS=deepseek-chat
STRIPE_SUCCESS_URL=http://localhost:3001/account?paid=1
STRIPE_CANCEL_URL=http://localhost:3001/account

# Stripe 测试（测充值时再填）
STRIPE_SECRET_KEY=sk_test_xxx
STRIPE_WEBHOOK_SECRET=whsec_xxx
```

### 2.2 前端 `dashboard/.env.local`

```env
NEXT_PUBLIC_API_URL=http://127.0.0.1:3000
NEXT_PUBLIC_EMAIL_VERIFY_ENABLED=false
```

改完后**重启** `npm run dev`。

---

## 3. 浏览器验流程（推荐顺序）

### 3.1 注册 + 登录

1. 打开 http://localhost:3001  
2. 切到「注册」  
3. 邀请码：`cloudtoken2026`（与 `.env` 中 `INVITE_CODE` 一致）  
4. 注册并登录  

**预期**：进入首页 **AI 对话**，顶栏显示余额（可能为 `$0.0000`）。

### 3.2 月赠 $0.5（首页对话）

1. 在首页输入一句：`你好`  
2. 发送，应看到**流式**回复  

**预期**：

- 顶栏余额约 **$0.5** 减去本次消费  
- 再次打开 http://localhost:3001/account ，绿色横幅显示账户余额  

用 API 复核（先在前端登录后，从浏览器 DevTools → Application → Local Storage 找 `token`，或登录响应里的 token）：

```bash
TOKEN="粘贴你的JWT"
curl -s http://127.0.0.1:3000/dashboard/me \
  -H "Authorization: Bearer $TOKEN" | python3 -m json.tool
```

应看到 `"balance"`、`"has_paid": false`、`"can_create_key": false`。

### 3.3 首次充值（Stripe 测试）

**前置**：已配置 `STRIPE_SECRET_KEY`，并安装 [Stripe CLI](https://stripe.com/docs/stripe-cli)。

**终端 D**（转发 Webhook 到本地）：

```bash
stripe login
stripe listen --forward-to http://127.0.0.1:3000/webhook/stripe
```

复制 CLI 输出的 `whsec_...` 到 `sub2api-go/.env` 的 `STRIPE_WEBHOOK_SECRET`，**重启后端**。

**浏览器**：

1. 右上角「充值」→ 选金额 → 跳转 Stripe 测试页  
2. 测试卡：`4242 4242 4242 4242`，任意未来日期 / CVC  
3. 支付成功后回到 `/account?paid=1`  

**预期**：

- `GET /dashboard/me` 中 `has_paid: true`、`can_create_key: true`  
- 余额增加对应美元数  

### 3.4 创建 API Key

1. 进入 http://localhost:3001/account  
2. 「+ Create Key」应**可点击**（未充值时为灰色）  
3. 填写密码、可选「消费上限 USD」  
4. 创建成功后复制 `sk-sub2api-...`  

### 3.5 用 Key 调 OpenAI 兼容 API

```bash
API_KEY="sk-sub2api-你的key"

curl -s http://127.0.0.1:3000/v1/chat/completions \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "deepseek-chat",
    "messages": [{"role": "user", "content": "say hi"}],
    "stream": false
  }' | head -c 800
```

**预期**：返回 JSON；账户余额减少（在 `/account` 或 `/dashboard/me` 查看）。

### 3.6 个人中心

http://localhost:3001/profile — 改昵称、改密码后重新登录验证。

---

## 4. 不配置 Stripe 时的快速解锁（仅本地调试）

仅用于跳过支付测 Key，**不要用于生产**。

**方式 A**：临时关闭门禁，重启后端：

```env
REQUIRE_PAYMENT_BEFORE_CREATE_KEY=false
```

**方式 B**：用 Stripe CLI 触发测试事件（仍需 Stripe 账号）：

```bash
stripe trigger checkout.session.completed
```

注意：测试事件里的 metadata 可能不是 `account_topup`，入账可能失败；完整流程仍建议走 3.3 的真实 Checkout。

---

## 5. 常见问题

| 现象 | 原因 | 处理 |
|------|------|------|
| 前端请求 `localhost:3000` 失败 | API 未启动或 `.env.local` 错 | 确认后端在跑、`NEXT_PUBLIC_API_URL` 正确 |
| 注册要验证码 | 邮箱验证开着 | 后端 `EMAIL_VERIFY_ENABLED=false`，前端 `NEXT_PUBLIC_EMAIL_VERIFY_ENABLED=false` |
| 对话 502 / provider error | 无 `DEEPSEEK_API_KEY` | 填入有效 Key 并重启后端 |
| 创建 Key 灰色 | 未首充 | 完成 Stripe 测试或临时 `REQUIRE_PAYMENT_BEFORE_CREATE_KEY=false` |
| 充值成功余额不变 | Webhook 未到达 | 必须 `stripe listen` 且 `STRIPE_WEBHOOK_SECRET` 与 CLI 一致 |
| 登录后白屏 / keys 报错 | 旧缓存 | 硬刷新；确认 `GET /dashboard/keys` 返回 `[]` 而非 `null` |
| 端口占用 | 3000/3001 被占 | `lsof -i :3000` 杀掉进程；前端务必 `-p 3001` |

---

## 6. 命令行自检清单（打勾即通过）

```bash
# 1) 健康
curl -sf http://127.0.0.1:3000/health/ready

# 2) 公开配置
curl -sf http://127.0.0.1:3000/auth/config | grep -q USD

# 3) 前端构建（可选）
cd dashboard && npm run build

# 4) 后端编译
cd sub2api-go && go build -o bin/server ./cmd/server
```

---

## 7. 相关文档

- 产品规则：`docs/product/ACCOUNT-WALLET.md`
- Dashboard API：`docs/api/DASHBOARD-API.md`
- **本地推生产（保姆级）**：`docs/operations/DEPLOY-FROM-LOCAL.md`
- 生产全量装机：`docs/operations/PRODUCTION-DEPLOY.md`
