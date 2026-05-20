# 本地代码推到生产环境（保姆级）

> 适用：你已在 Mac 本地测通，生产机已装好（`43.134.8.202`、`cloudtoken.uk`）。  
> 只做**日常发版**；全新装机见 [PRODUCTION-DEPLOY.md](./PRODUCTION-DEPLOY.md) 或 [GREENFIELD-DEPLOY.md](./GREENFIELD-DEPLOY.md)。

---

## 0. 三个地方要分清

| 位置 | 做什么 |
|------|--------|
| **Mac 本机** | 改代码、`git push`、发版前编译自测 |
| **GitHub** | 代码中转，`heyangshnu/sub2api` |
| **腾讯云服务器** | `git pull`、编译、重启 `sub2api` / `dashboard` |

**千万不要**把本机 `sub2api-go/.env`、`dashboard/.env.local`、`*.db` 提交或 `scp` 覆盖到生产（除非你有意恢复备份）。

---

## 1. 发版前自检（Mac，约 5 分钟）

### 1.1 确认没有误提交密钥

```bash
cd /Users/heyang/Desktop/myProject/sub2api-full-code
git status
```

应**看不到** `.env`、`.db` 在待提交列表（已在 `.gitignore`）。

### 1.2 本地编译通过

```bash
cd sub2api-go
go build -o bin/server ./cmd/server
echo "Go 编译 OK"

cd ../dashboard
npm ci
npm run build
echo "前端编译 OK"
```

任一步报错 → **先在本机修完**，再推生产。

### 1.3 提交并推到 GitHub

```bash
cd /Users/heyang/Desktop/myProject/sub2api-full-code
git add -A
git status                    # 再看一眼，确认无 .env
git commit -m "你的发版说明，例如：账户钱包与对话首页"
git push origin main
```

若 `git push` 要密码：用 **GitHub 用户名 + Personal Access Token（PAT）**，不是邮箱登录密码。

记下当前提交号（方便回滚）：

```bash
git rev-parse HEAD
```

---

## 2. 登录服务器（Mac 新开一个终端）

```bash
ssh root@43.134.8.202
```

连不上时检查：腾讯云防火墙是否放行 **22**；IP/密码是否正确。

---

## 3. 发版前备份（服务器，约 1 分钟）

```bash
cd /opt/sub2api

# 记录「上一版好使」的 commit
git rev-parse HEAD | tee /root/last-good-commit.txt

# 备份 SQLite（用户数据）
cp -a sub2api-go/data/sub2api.db /root/backup-$(date +%Y%m%d-%H%M).db

# 可选：备份生产 .env（勿提交到 Git）
cp -a sub2api-go/.env /root/sub2api-go.env.backup-$(date +%Y%m%d-%H%M)
```

---

## 4. 拉取最新代码（服务器）

```bash
cd /opt/sub2api
git pull --ff-only origin main
```

若提示冲突或 `not possible to fast-forward`：先 `git status`，不要强行覆盖；回 Mac 处理分支后再 pull。

---

## 5. 核对生产环境变量（服务器，重要）

账户钱包等新功能需要 `.env` 里有这些项（没有就**追加**，不要整文件用本机 `.env` 覆盖）：

```bash
nano /opt/sub2api/sub2api-go/.env
```

建议确认存在：

```env
APP_ENV=production
PORT=3000
REDIS_URL=redis://127.0.0.1:6379

# 账户钱包（USD）
ACCOUNT_MONTHLY_GRANT_USD=0.5
REQUIRE_PAYMENT_BEFORE_CREATE_KEY=true
CHAT_ENABLED_MODELS=deepseek-chat
STRIPE_SUCCESS_URL=https://cloudtoken.uk/account?paid=1
STRIPE_CANCEL_URL=https://cloudtoken.uk/account

# 生产密钥（务必用 Live / 长随机串，不要用开发默认值）
JWT_SECRET=至少32位随机字符串
ADMIN_KEY=你的强管理密钥
DEEPSEEK_API_KEY=sk-...

# 邀请码（与注册页一致，勿写进前端）
INVITE_CODE=你的邀请码

# 海外服务器建议关闭邮箱验证（163 等国内邮箱 SMTP 常被拒）
EMAIL_VERIFY_ENABLED=false
```

前端 API 地址（**改这里后必须重新 build 前端**）：

```bash
cat /opt/sub2api/dashboard/.env.production
```

应为一行：

```env
NEXT_PUBLIC_API_URL=https://api.cloudtoken.uk
```

若没有该文件：

```bash
echo 'NEXT_PUBLIC_API_URL=https://api.cloudtoken.uk' > /opt/sub2api/dashboard/.env.production
```

---

## 6. 编译并重启后端（服务器）

```bash
cd /opt/sub2api/sub2api-go
go build -o bin/sub2api ./cmd/server

systemctl restart sub2api
systemctl status sub2api --no-pager
```

`Active: active (running)` 为正常。

看最近日志：

```bash
journalctl -u sub2api -n 30 --no-pager
```

---

## 7. 编译并重启前端（服务器）

只要改过 `dashboard/` 或 `NEXT_PUBLIC_*`，都要做：

```bash
cd /opt/sub2api/dashboard
npm ci
NODE_ENV=production npm run build
systemctl restart dashboard
systemctl status dashboard --no-pager
```

> 注意：`npm ci` **不要**加 `NODE_ENV=production`（否则 devDependencies 缺失，Tailwind 可能编译失败）。  
> 只在 `npm run build` 前加 `NODE_ENV=production`。

---

## 8. 发版后验收（Mac + 浏览器）

### 8.1 命令行（Mac）

```bash
curl -sS https://api.cloudtoken.uk/health
curl -sS -o /dev/null -w "ready=%{http_code}\n" https://api.cloudtoken.uk/health/ready
curl -sS -o /dev/null -w "web=%{http_code}\n" https://cloudtoken.uk
```

| 结果 | 说明 |
|------|------|
| health 返回 JSON | API 正常 |
| ready=200 | Redis 正常 |
| web=200 | 前端正常 |
| 521 / 502 | 见下方故障表 |

### 8.2 浏览器（建议无痕）

1. 打开 https://cloudtoken.uk  
2. F12 → Network → 刷新，接口 Host 必须是 **api.cloudtoken.uk**  
3. 登录 → 首页对话 → 账户页看余额 → 充值 / 建 Key（按你本次发版功能点测）

---

## 9. 一键发版脚本（熟练后可复制）

**Mac：**

```bash
cd /Users/heyang/Desktop/myProject/sub2api-full-code/sub2api-go && go build -o bin/server ./cmd/server && \
cd ../dashboard && npm run build && \
cd .. && git push origin main
```

**服务器：**

```bash
cd /opt/sub2api && \
git rev-parse HEAD | tee /root/last-good-commit.txt && \
cp -a sub2api-go/data/sub2api.db /root/backup-$(date +%Y%m%d-%H%M).db && \
git pull --ff-only origin main && \
cd sub2api-go && go build -o bin/sub2api ./cmd/server && systemctl restart sub2api && \
cd ../dashboard && npm ci && NODE_ENV=production npm run build && systemctl restart dashboard && \
systemctl status sub2api dashboard --no-pager
```

---

## 10. 出问题了怎么办

### 10.1 仅代码/前端有问题 → 回滚上一版 commit

```bash
systemctl stop dashboard sub2api
cd /opt/sub2api
git checkout $(cat /root/last-good-commit.txt)
cd sub2api-go && go build -o bin/sub2api ./cmd/server
cd ../dashboard && npm ci && NODE_ENV=production npm run build
systemctl start sub2api dashboard
```

### 10.2 数据库坏了 → 用备份 db

```bash
systemctl stop dashboard sub2api
cp -a /root/backup-XXXX.db /opt/sub2api/sub2api-go/data/sub2api.db
# 再执行 10.1 代码回滚
systemctl start sub2api dashboard
```

### 10.3 常见现象

| 现象 | 处理 |
|------|------|
| 页面能开，接口 404/CORS | 检查 `dashboard/.env.production` 后**重新 build + restart dashboard** |
| `sub2api` 起不来 | `journalctl -u sub2api -n 80`；查 Redis、`JWT_SECRET` 长度、`APP_ENV=production` |
| 登录后余额不对 | 生产 Redis 账户键；必要时用 `scripts/dev_account_topup.go` 逻辑对照，勿乱删 Redis |
| Cloudflare 521 | 源站 Nginx/443、A 记录指向 `43.134.8.202` |
| Stripe 充值不到账 | Webhook `https://api.cloudtoken.uk/webhook/stripe`；`STRIPE_WEBHOOK_SECRET` 与 Stripe CLI/Dashboard 一致 |

---

## 11. 发版检查清单（可打印）

```
□ Mac：go build 通过
□ Mac：dashboard npm run build 通过
□ Mac：git push origin main 成功
□ 服务器：备份 sub2api.db + last-good-commit.txt
□ 服务器：git pull 成功
□ 服务器：sub2api-go/.env 含账户钱包与 DEEPSEEK_API_KEY
□ 服务器：dashboard/.env.production → https://api.cloudtoken.uk
□ 服务器：go build + restart sub2api
□ 服务器：npm ci + NODE_ENV=production npm run build + restart dashboard
□ 验收：curl health / ready / cloudtoken.uk
□ 验收：浏览器登录 + 对话 + 账户
```

---

相关：[LOCAL-VERIFY.md](./LOCAL-VERIFY.md)（本地怎么测）、[PRODUCTION-DEPLOY.md](./PRODUCTION-DEPLOY.md)（全量装机）。
