# 全新生产部署（无备份 · Ubuntu 22.04 · Git 维护）

> **适用你的情况**：重装系统、生产上**没有**旧 `.env`、**没有**用户/数据库数据、不需要备份。  
> **目标**：本机 `sub2api-full-code` → GitHub → 服务器 `/opt/sub2api` → 正常访问 `https://cloudtoken.uk`。  
> **仓库**：`https://github.com/heyangshnu/sub2api.git`  
> **源站 IP**：`43.134.8.202`（Cloudflare 橙云时 `dig` 显示 CF IP 属正常）

---

## 操作在哪里做

| 符号 | 位置 | 你会看到什么 |
|------|------|----------------|
| ☁️ | 浏览器：腾讯云、Cloudflare | 网页表单、按钮 |
| 🖥️ | Mac 终端 | `heyang@...MacBook-Air sub2api-full-code %` |
| 🐧 | SSH 连服务器 | `root@VM-xxx:/opt/sub2api#` |

---

## 最终目录（与本地一致，便于 Git 扩展）

```text
/opt/sub2api/                    ← git clone 根目录（以后 git pull）
├── sub2api-go/
│   ├── .env                     ← 仅服务器上有，不进 Git
│   ├── bin/sub2api              ← go build 产物
│   └── data/sub2api.db          ← 首次启动自动创建（空库）
└── dashboard/
    ├── .env.production          ← 仅服务器上有，不进 Git
    └── .next/                   ← npm run build 产物
```

---

# 阶段 0：本机准备（🖥️，重装服务器之前）

## 0.1 确认代码能编译

**在做什么**：保证推到 GitHub 的代码本身没问题，避免在服务器上才发现编译错误。

```bash
cd /Users/heyang/Desktop/myProject/sub2api-full-code/sub2api-go
go build -o /tmp/sub2api-test ./cmd/server && echo "Go OK"

cd ../dashboard
npm ci
NODE_ENV=production NEXT_PUBLIC_API_URL=https://api.cloudtoken.uk npm run build
```

**成功标志**：`Finished TypeScript`，无 `Failed to compile`。

---

## 0.2 提交并推到 GitHub

**在做什么**：建立「本机 → GitHub → 服务器」唯一代码通道，后续只维护这一条。

```bash
cd /Users/heyang/Desktop/myProject/sub2api-full-code
git status
# 确认列表里没有 sub2api-go/.env、dashboard/.env.local、*.db

git add -A
git commit -m "chore: production greenfield deploy"
git push origin main
```

**解决什么问题**：服务器用 `git clone` / `git pull` 与线上一致；密钥永不进仓库。

---

## 0.3 确认 Cloudflare DNS（🖥️）

```bash
dig +short cloudtoken.uk
dig +short api.cloudtoken.uk
```

**☁️ Cloudflare** → 域名 `cloudtoken.uk` → **DNS**：

| 类型 | 名称 | 内容 | 代理 |
|------|------|------|------|
| A | `@` | `43.134.8.202` | 保持现状 |
| A | `api` | `43.134.8.202` | 保持现状 |

**SSL/TLS** → **完全** 或 **完全（严格）**（源站配好 HTTPS 后）。

**说明**：`dig` 若显示 `172.67.x` / `104.21.x` 是 Cloudflare 代理 IP，不是错误。

---

# 阶段 1：重装 Ubuntu 22.04（☁️ 腾讯云）

| 步骤 | 界面操作 | 解决什么问题 |
|------|----------|----------------|
| 1 | 轻量应用服务器 → 你的实例 | 进对机器 |
| 2 | **重装系统** → **基于操作系统镜像** | 不用带 WordPress 等模板 |
| 3 | 选 **Ubuntu 22.04 LTS** → 确认清空系统盘 | 干净 apt，避免旧 dpkg 损坏 |
| 4 | 等待状态 **运行中** | 系统可用 |
| 5 | **重置密码** 或 **绑定 SSH 密钥** → 必要时重启 | 能 SSH 登录 |
| 6 | **防火墙** 放行 **22、80、443** | 外网访问与证书申请 |

---

# 阶段 2：登录并安装运行环境（🐧）

## 2.1 SSH 登录

**🖥️ Mac：**

```bash
ssh root@43.134.8.202
```

---

## 2.2 安装依赖（一条命令装齐）

**在做什么**：装 Git、反代、Redis、证书工具、Go、Node。

```bash
apt update && apt upgrade -y

apt install -y git curl ufw nginx redis-server \
  certbot python3-certbot-nginx golang-go

curl -fsSL https://deb.nodesource.com/setup_20.x | bash -
apt install -y nodejs

systemctl enable --now redis-server
redis-cli ping
```

期望：`PONG`。

```bash
node -v    # v20.x
go version
git --version
```

| 组件 | 解决什么问题 |
|------|----------------|
| redis-server | 生产计费/限流；`APP_ENV=production` 时 Redis 必须可用 |
| nodejs | 构建与运行 dashboard |
| golang-go | 服务器上编译后端 |
| nginx + certbot | 域名 HTTPS 反代 |
| git | clone / pull 维护代码 |

---

# 阶段 3：从 GitHub 拉代码（🐧）

**在做什么**：服务器代码树与 GitHub `main` 一致，以后发版只 `git pull`。

```bash
mkdir -p /opt
cd /opt
git clone https://github.com/heyangshnu/sub2api.git sub2api
cd /opt/sub2api
ls
```

应看到 **`sub2api-go`** 和 **`dashboard`**。

**后续扩展**：可加 Deploy Key，改用 `git clone git@github.com:heyangshnu/sub2api.git sub2api`。

---

# 阶段 4：新建生产配置（🐧，无备份场景核心）

## 4.1 后端 `.env`（首次创建）

**在做什么**：配置密钥、Redis、域名相关 URL；文件只留在服务器。

```bash
cd /opt/sub2api/sub2api-go
cp .env.example .env

# 生成两个随机串（复制输出填进 .env）
openssl rand -base64 48
openssl rand -base64 48

nano .env
```

**建议内容**（把 `替换1` `替换2` 换成上面生成的串，上游 Key 换成你的）：

```bash
APP_ENV=production
PORT=3000

REDIS_URL=redis://127.0.0.1:6379
TRUSTED_PROXIES=127.0.0.1/32,::1/128

JWT_SECRET=替换1
ADMIN_KEY=替换2

INVITE_CODE=cloudtoken2026

DEEPSEEK_API_KEY=sk-你的deepseek密钥
# OPENAI_API_KEY=
# ANTHROPIC_API_KEY=

STRIPE_SECRET_KEY=
STRIPE_WEBHOOK_SECRET=
STRIPE_SUCCESS_URL=https://cloudtoken.uk/payment/success
STRIPE_CANCEL_URL=https://cloudtoken.uk

EMAIL_VERIFY_ENABLED=false
```

```bash
chmod 600 .env
mkdir -p data
```

| 配置项 | 解决什么问题 |
|--------|----------------|
| `APP_ENV=production` | 启用生产校验；弱密钥会拒绝启动 |
| `JWT_SECRET` ≥32 字符 | 用户登录 Token 安全 |
| `ADMIN_KEY` 非默认 | 管理接口安全 |
| `TRUSTED_PROXIES` | Nginx 后真实客户端 IP |
| `DEEPSEEK_API_KEY` 等 | Chat 能调上游 |
| `STRIPE_*_URL` 用 cloudtoken.uk | 支付完成不跳 localhost |
| `data/` 目录 | SQLite 首次启动写入 `data/sub2api.db`（空库） |

> **不要**把本机 `sub2api-go/.env` 直接 scp 上传（开发配置与生产混用）。

---

## 4.2 前端 `.env.production`（首次创建）

**在做什么**：构建时把 API 地址写进前端 JS。

```bash
cat > /opt/sub2api/dashboard/.env.production <<'EOF'
NEXT_PUBLIC_API_URL=https://api.cloudtoken.uk
NEXT_PUBLIC_EMAIL_VERIFY_ENABLED=false
EOF
```

**解决什么问题**：用户打开 `cloudtoken.uk` 时请求 `api.cloudtoken.uk`，而不是 `127.0.0.1`。

---

# 阶段 5：编译与构建（🐧）

## 5.1 后端

```bash
cd /opt/sub2api/sub2api-go
go build -o bin/sub2api ./cmd/server
```

## 5.2 前端

```bash
cd /opt/sub2api/dashboard
npm ci
NODE_ENV=production npm run build
```

| 注意 | 原因 |
|------|------|
| 用 `npm ci`，**不要** `NODE_ENV=production npm ci` | 否则会缺 `@tailwindcss/postcss` |
| build 成功无 `Failed to compile` | 与阶段 0.1 一致 |

---

# 阶段 6：systemd 常驻（🐧）

**在做什么**：开机自启、崩溃自动重启；固定工作目录便于扩展。

```bash
cat > /etc/systemd/system/sub2api.service <<'EOF'
[Unit]
Description=Sub2API Go API
After=network.target redis-server.service

[Service]
Type=simple
WorkingDirectory=/opt/sub2api/sub2api-go
EnvironmentFile=/opt/sub2api/sub2api-go/.env
ExecStart=/opt/sub2api/sub2api-go/bin/sub2api
Restart=always
RestartSec=3

[Install]
WantedBy=multi-user.target
EOF

cat > /etc/systemd/system/dashboard.service <<'EOF'
[Unit]
Description=Sub2API Dashboard (Next.js)
After=network.target

[Service]
Type=simple
WorkingDirectory=/opt/sub2api/dashboard
Environment=NODE_ENV=production
ExecStart=/usr/bin/npm run start -- -p 3001
Restart=always
RestartSec=3

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable --now sub2api dashboard
systemctl status sub2api dashboard --no-pager
```

**自检（🐧）：**

```bash
curl -sS http://127.0.0.1:3000/health | head
curl -sS -o /dev/null -w "%{http_code}\n" http://127.0.0.1:3001
ls -la /opt/sub2api/sub2api-go/data/sub2api.db
```

最后一行：首次启动后应出现 **sqlite 数据库文件**（空库）。

失败时：`journalctl -u sub2api -n 80 --no-pager`

---

# 阶段 7：Nginx + HTTPS（🐧）

```bash
cat > /etc/nginx/sites-available/cloudtoken.conf <<'EOF'
server {
    listen 80;
    server_name api.cloudtoken.uk;
    location / {
        proxy_pass http://127.0.0.1:3000;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_read_timeout 3600s;
        proxy_buffering off;
    }
}

server {
    listen 80;
    server_name cloudtoken.uk www.cloudtoken.uk;
    location / {
        proxy_pass http://127.0.0.1:3001;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
EOF

ln -sf /etc/nginx/sites-available/cloudtoken.conf /etc/nginx/sites-enabled/
rm -f /etc/nginx/sites-enabled/default
nginx -t && systemctl reload nginx

certbot --nginx -d api.cloudtoken.uk -d cloudtoken.uk -d www.cloudtoken.uk
```

**☁️ Cloudflare** → SSL/TLS → **完全（严格）**（certbot 成功后）。

可选：

```bash
ufw allow OpenSSH && ufw allow 80/tcp && ufw allow 443/tcp && ufw --force enable
```

---

# 阶段 8：验收（🖥️）

```bash
curl -sS https://api.cloudtoken.uk/health
curl -sS -o /dev/null -w "%{http_code}\n" https://api.cloudtoken.uk/health/ready
```

浏览器：

1. 无痕打开 `https://cloudtoken.uk`
2. F12 → Network → 刷新 → API 请求 Host 为 **`api.cloudtoken.uk`**
3. 用邀请码 **注册** 第一个用户 → 登录 → 创建 Key

| 结果 | 含义 |
|------|------|
| health 有 JSON | API + Nginx + CF 正常 |
| ready = 200 | Redis 正常 |
| 521 | CF 连不上源站 → 查 systemd / 防火墙 / A 记录 |
| 能注册 | 空库 + `.env` 正常 |

---

# 阶段 9：后续用 Git 发版（可扩展工作流）

## 9.1 日常流程（固定四步）

```text
🖥️ 改代码 → 本机构建自测 → git push
🐧 ssh 登录 → /opt/sub2api → git pull → build → restart → curl 验收
```

## 9.2 本机（🖥️）

```bash
cd /Users/heyang/Desktop/myProject/sub2api-full-code
# 可选：改完先本地 build 测一遍

git add -A && git commit -m "feat: xxx" && git push origin main
```

## 9.3 服务器（🐧）

```bash
cd /opt/sub2api

# 记录回滚点（空库阶段也建议养成习惯）
git rev-parse HEAD | tee /root/last-good-commit.txt

git pull --ff-only origin main

cd sub2api-go
go build -o bin/sub2api ./cmd/server
systemctl restart sub2api

# 若改了 dashboard/ 或 dashboard/.env.production 或 NEXT_PUBLIC_*：
cd ../dashboard
npm ci
NODE_ENV=production npm run build
systemctl restart dashboard
```

## 9.4 发版后自检

```bash
curl -sS https://api.cloudtoken.uk/health | head
```

浏览器冒烟：登录、Key、Chat。

## 9.5 有用户数据后的扩展（现在可跳过）

```bash
cp -a /opt/sub2api/sub2api-go/data/sub2api.db \
  /root/backup-sub2api-$(date +%Y%m%d-%H%M).db
```

## 9.6 回滚（仅代码）

```bash
cd /opt/sub2api
git checkout $(cat /root/last-good-commit.txt)
cd sub2api-go && go build -o bin/sub2api ./cmd/server
cd ../dashboard && npm ci && NODE_ENV=production npm run build
systemctl restart sub2api dashboard
```

---

# 阶段 10：后续可扩展方向

| 方向 | 文档/做法 |
|------|-----------|
| 监控 `/health`、`/metrics` | [OBSERVABILITY.md](./OBSERVABILITY.md) |
| 多节点 / PG | [SCALING.md](./SCALING.md) |
| Stripe Live | Dashboard 配 Webhook `https://api.cloudtoken.uk/webhook/stripe` |
| 邮箱注册 | `.env` 开 `EMAIL_VERIFY_ENABLED` + SMTP，前后端开关一致 |
| CI 自动部署 | GitHub Actions ssh 执行 9.3（可选） |

---

# 检查清单（打印勾选）

```text
□ 本机 go build + dashboard npm run build 通过
□ git push 到 github.com/heyangshnu/sub2api
□ 腾讯云 Ubuntu 22.04 重装完成，22/80/443 放行
□ CF DNS: @ 和 api → 43.134.8.202
□ git clone /opt/sub2api
□ sub2api-go/.env 已创建（production + 强密钥 + 上游 Key）
□ dashboard/.env.production → api.cloudtoken.uk
□ npm ci + NODE_ENV=production npm run build
□ systemctl sub2api dashboard active
□ certbot 成功
□ curl health / ready 正常
□ 浏览器注册登录成功
```

---

完整版（含从旧机迁移）：[PRODUCTION-DEPLOY.md](./PRODUCTION-DEPLOY.md)
