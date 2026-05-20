# Sub2API 生产部署手册（Ubuntu 22.04 + Cloudflare）

> 适用：腾讯云轻量、域名 **cloudtoken.uk** / **api.cloudtoken.uk**、本机 **sub2api-full-code**、GitHub **`https://github.com/heyangshnu/sub2api.git`**。  
> 更新：2026-05，含 Cloudflare 代理、重装系统、发版与回滚。

**你是全新生产、无旧 `.env`、无用户数据？** 请直接跟：[GREENFIELD-DEPLOY.md](./GREENFIELD-DEPLOY.md)（无备份、从零装 `.env`、Git 维护）。

下文 **第一部分** 为「从旧机迁移备份」可选流程；全新安装可跳过。

---

## 三个操作位置

| 符号 | 在哪里 | 说明 |
|------|--------|------|
| ☁️ | 腾讯云 / Cloudflare / Stripe 网页 | 浏览器控制台 |
| 🖥️ | Mac 本机终端 | 项目目录、`git push`、`scp` 备份 |
| 🐧 | 服务器 SSH | `root@<公网IP>`，部署与运维 |

---

## 架构与目录对照

```
用户浏览器
    → Cloudflare（dig/ping 看到 172.67.x / 104.21.x 是正常的）
    → 源站 43.134.8.202（Ubuntu + Nginx）
         ├── api.cloudtoken.uk  → 127.0.0.1:3000  (sub2api-go)
         └── cloudtoken.uk      → 127.0.0.1:3001  (dashboard)

本机:  sub2api-full-code/
         ├── sub2api-go/
         └── dashboard/

服务器: /opt/sub2api/          （git clone，与上结构一致）
         ├── sub2api-go/.env
         ├── sub2api-go/data/sub2api.db
         └── dashboard/.env.production
```

**不要**把本机 `dashboard/.env.local` 当生产配置上传；生产只用服务器上的 `dashboard/.env.production`。

---

# 第一部分：重装前（🖥️ + 备份）

## 1.1 备份到 Mac（必做）

**🖥️ Mac：**

```bash
mkdir -p ~/Desktop/sub2api-server-backup

# 路径按你旧服务器实际位置改（旧布局可能是 /opt/sub2api 根目录）
scp root@43.134.8.202:/opt/sub2api/sub2api-go/.env \
  ~/Desktop/sub2api-server-backup/sub2api-go.env

scp root@43.134.8.202:/opt/sub2api/sub2api-go/data/sub2api.db \
  ~/Desktop/sub2api-server-backup/sub2api.db

# 若有前端生产 env
scp root@43.134.8.202:/opt/sub2api/dashboard/.env.production \
  ~/Desktop/sub2api-server-backup/dashboard.env.production 2>/dev/null || true
```

| 文件 | 解决什么问题 |
|------|----------------|
| `.env` | 重装后恢复 JWT、Stripe、上游 Key |
| `sub2api.db` | 保留用户与 Key 数据 |

## 1.2 本机验证代码可构建（必做）

**🖥️ Mac：**

```bash
cd /Users/heyang/Desktop/myProject/sub2api-full-code/sub2api-go
go build -o /tmp/sub2api-test ./cmd/server && echo "Go OK"

cd ../dashboard
npm ci
NODE_ENV=production NEXT_PUBLIC_API_URL=https://api.cloudtoken.uk npm run build
```

成功：`Finished TypeScript`，无 `Failed to compile`。失败则**先在本机修代码**，再重装服务器。

## 1.3 推到 GitHub

```bash
cd /Users/heyang/Desktop/myProject/sub2api-full-code
git status   # 确认无 .env / .db 被提交
git push origin main
```

记下：`https://github.com/heyangshnu/sub2api.git`

## 1.4 确认 DNS / Cloudflare（🖥️）

```bash
dig +short cloudtoken.uk
dig +short api.cloudtoken.uk
```

若结果是 **172.67.x / 104.21.x** → 域名走 **Cloudflare 代理**（正常，不是轻量机 IP）。

在 **☁️ Cloudflare** → `cloudtoken.uk` → **DNS** 确认：

| 类型 | 名称 | 内容 | 代理 |
|------|------|------|------|
| A | `@` | `43.134.8.202` | 与现网一致 |
| A | `api` | `43.134.8.202` | 与现网一致 |

**SSL/TLS** 建议：**完全** 或 **完全（严格）**（源站需 HTTPS，见下文 certbot）。

---

# 第二部分：重装 Ubuntu 22.04（☁️ 腾讯云）

1. 轻量应用服务器 → 选中实例 → **重装系统**
2. **基于操作系统镜像** → **Ubuntu 22.04 LTS**（不要选应用模板）
3. 确认系统盘清空 → 等待 **运行中**
4. **重置密码** 或 **绑定 SSH 密钥** → 必要时 **重启**
5. **防火墙**：放行 TCP **22、80、443**

| 步骤 | 解决什么问题 |
|------|----------------|
| Ubuntu 22.04 | 完整 `apt`，避免旧系统 dpkg 损坏 |
| SSH | 能登录才能部署 |
| 80/443 | 网站与 Let's Encrypt |

---

# 第三部分：首次登录与装环境（🐧）

## 3.1 SSH

**🖥️ Mac：**

```bash
ssh root@43.134.8.202
# 或 ssh -i ~/.ssh/id_ed25519 root@43.134.8.202
```

## 3.2 系统更新与依赖

**🐧 服务器：**

```bash
apt update && apt upgrade -y

apt install -y git curl ufw nginx redis-server \
  certbot python3-certbot-nginx golang-go

curl -fsSL https://deb.nodesource.com/setup_20.x | bash -
apt install -y nodejs

systemctl enable --now redis-server
redis-cli ping    # 期望 PONG

node -v           # 期望 v20.x
go version
```

| 组件 | 作用 |
|------|------|
| redis-server | 生产计费/限流 |
| nodejs | 构建与运行 dashboard |
| golang-go | 编译 sub2api-go |
| nginx + certbot | 反代与源站 HTTPS |

---

# 第四部分：拉代码（🐧）

```bash
mkdir -p /opt
cd /opt
git clone https://github.com/heyangshnu/sub2api.git sub2api
cd /opt/sub2api
ls    # 应看到 sub2api-go  dashboard
```

私有仓库：配置 Deploy Key 后可用 SSH：

```bash
git clone git@github.com:heyangshnu/sub2api.git sub2api
```

---

# 第五部分：恢复配置与数据（🖥️ scp + 🐧）

## 5.1 上传备份

**🖥️ Mac：**

```bash
scp ~/Desktop/sub2api-server-backup/sub2api-go.env \
  root@43.134.8.202:/opt/sub2api/sub2api-go/.env

ssh root@43.134.8.202 "mkdir -p /opt/sub2api/sub2api-go/data"

scp ~/Desktop/sub2api-server-backup/sub2api.db \
  root@43.134.8.202:/opt/sub2api/sub2api-go/data/sub2api.db
```

**🐧 服务器：**

```bash
chmod 600 /opt/sub2api/sub2api-go/.env
```

## 5.2 核对 `sub2api-go/.env`

```bash
nano /opt/sub2api/sub2api-go/.env
```

生产必查：

```bash
APP_ENV=production
PORT=3000
REDIS_URL=redis://127.0.0.1:6379
TRUSTED_PROXIES=127.0.0.1/32,::1/128

JWT_SECRET=<至少32字符随机>
ADMIN_KEY=<随机，非默认>

STRIPE_SUCCESS_URL=https://cloudtoken.uk/payment/success
STRIPE_CANCEL_URL=https://cloudtoken.uk

DEEPSEEK_API_KEY=<至少一个上游>
# OPENAI_API_KEY= ...
```

无备份时：复制 `.env.example` 为 `.env` 并填齐；`openssl rand -base64 48` 生成密钥。

## 5.3 前端 `dashboard/.env.production`

**🐧 服务器：**

```bash
cat > /opt/sub2api/dashboard/.env.production <<'EOF'
NEXT_PUBLIC_API_URL=https://api.cloudtoken.uk
NEXT_PUBLIC_EMAIL_VERIFY_ENABLED=false
EOF
```

须与后端 `EMAIL_VERIFY_ENABLED` 一致。  
**作用**：build 时把 API 地址写入 JS，避免请求打到用户本机 `localhost`。

---

# 第六部分：编译与构建（🐧）

## 6.1 后端

```bash
cd /opt/sub2api/sub2api-go
go build -o bin/sub2api ./cmd/server
```

## 6.2 前端（注意 npm ci）

```bash
cd /opt/sub2api/dashboard
npm ci
NODE_ENV=production npm run build
```

| 错误做法 | 原因 |
|----------|------|
| `NODE_ENV=production npm ci` | 跳过 devDependencies，缺 `@tailwindcss/postcss` |

成功标志：`✓ Finished TypeScript`、`✓ Generating static pages`，无 `Failed to compile`。

---

# 第七部分：systemd（🐧）

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

本机自检：

```bash
curl -sS http://127.0.0.1:3000/health | head
curl -sS -o /dev/null -w "%{http_code}\n" http://127.0.0.1:3001
```

---

# 第八部分：Nginx + HTTPS（🐧）

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

**Cloudflare SSL/TLS**：源站有证书后选 **完全** 或 **完全（严格）**。

可选防火墙：

```bash
ufw allow OpenSSH
ufw allow 80/tcp
ufw allow 443/tcp
ufw --force enable
```

---

# 第九部分：Stripe（可选）

- Webhook URL：`https://api.cloudtoken.uk/webhook/stripe`
- `.env` 中 `STRIPE_SECRET_KEY`、`STRIPE_WEBHOOK_SECRET`（Live）
- `systemctl restart sub2api`

---

# 第十部分：验收清单

## 10.1 命令行（🖥️ Mac）

```bash
curl -sS https://api.cloudtoken.uk/health
curl -sS -o /dev/null -w "%{http_code}\n" https://api.cloudtoken.uk/health/ready
curl -sS -o /dev/null -w "%{http_code}\n" https://cloudtoken.uk
```

| 结果 | 含义 |
|------|------|
| health 有 JSON | API 正常 |
| ready = 200 | Redis 正常 |
| 521（CF） | 源站未起来或 CF A 记录错 |
| 502 | Nginx 或 sub2api/dashboard 未运行 |

## 10.2 浏览器

1. 无痕打开 `https://cloudtoken.uk`
2. F12 → Network → 刷新
3. API 请求 Host 必须是 **`api.cloudtoken.uk`**（不能是 `127.0.0.1` / `localhost:8080`）
4. 登录 → 建 Key → 用量/日志 → （可选）Chat

---

# 第十一部分：日常发版（生产已运行）

## 11.1 发版前（🖥️ + 🐧）

**🖥️ Mac：**

```bash
cd /Users/heyang/Desktop/myProject/sub2api-full-code
go build -o /tmp/t ./cmd/server   # 在 sub2api-go 下
cd dashboard && npm ci && NODE_ENV=production NEXT_PUBLIC_API_URL=https://api.cloudtoken.uk npm run build
git push origin main
```

**🐧 服务器（记录回滚点 + 备份）：**

```bash
cd /opt/sub2api
git rev-parse HEAD | tee /root/last-good-commit.txt
cp -a sub2api-go/data/sub2api.db /root/backup-$(date +%Y%m%d-%H%M).db
```

## 11.2 发版四自检（🐧）

```bash
# 1 前端 API 基址
grep NEXT_PUBLIC /opt/sub2api/dashboard/.env.production

# 2 后端关键项
grep -E '^APP_ENV=|^JWT_SECRET=|^REDIS_URL=|^STRIPE_SUCCESS|^TRUSTED_PROXIES=' \
  /opt/sub2api/sub2api-go/.env

# 3 禁止依赖 api.ts 的 localhost 兜底 — 必须先有 .env.production 再 build
# 4 使用 NODE_ENV=production npm run build（npm ci 不要用 production）
```

## 11.3 拉代码、构建、重启（🐧）

```bash
cd /opt/sub2api
git pull --ff-only origin main

cd sub2api-go
go build -o bin/sub2api ./cmd/server
systemctl restart sub2api

# 若改了 dashboard 或 NEXT_PUBLIC_*：
cd ../dashboard
npm ci
NODE_ENV=production npm run build
systemctl restart dashboard
```

## 11.4 发版后验证

同第十部分 `curl` + 浏览器冒烟。

---

# 第十二部分：回滚

## 仅代码/前端异常（不动库）

```bash
systemctl stop dashboard sub2api
cd /opt/sub2api
git checkout $(cat /root/last-good-commit.txt)
cd sub2api-go && go build -o bin/sub2api ./cmd/server
cd ../dashboard && npm ci && NODE_ENV=production npm run build
systemctl start sub2api dashboard
```

## 数据库也坏了（有备份）

```bash
systemctl stop dashboard sub2api
cp -a /root/backup-XXXX.db /opt/sub2api/sub2api-go/data/sub2api.db
# 再执行代码回滚步骤
systemctl start sub2api dashboard
```

---

# 附录 A：保证本机项目与线上一致

| 项 | 本机 | 线上 |
|----|------|------|
| 代码 | `git push` | `git pull` |
| 后端 env | `sub2api-go/.env`（开发） | 仅服务器 `.env` |
| 前端 API | `.env.local` → 127.0.0.1 | `.env.production` → `https://api.cloudtoken.uk` |
| 构建 | 发版前本地 `npm run build` 通过 | 服务器同样命令 |

发版前本地验证命令：

```bash
cd sub2api-go && go test ./...   # 可选
cd ../dashboard && npm ci && NODE_ENV=production NEXT_PUBLIC_API_URL=https://api.cloudtoken.uk npm run build
```

---

# 附录 B：常见故障

| 现象 | 排查 |
|------|------|
| 页面能开，登录失败 | `.env.production` + 重新 `npm run build` + restart dashboard |
| sub2api 起不来 | `journalctl -u sub2api -n 80`；Redis、JWT 长度、APP_ENV |
| build 缺 tailwind postcss | 用 `npm ci`，勿 `NODE_ENV=production npm ci` |
| CF 521 | 源站 443 服务、A 记录 43.134.8.202、防火墙 |
| dig 不是轻量 IP | 橙云代理正常，以 curl/浏览器为准 |

---

# 附录 C：旧服务器并行目录迁移说明

若旧环境为 `/opt/sub2api`（仅 server）+ `/opt/dashboard`（独立），新环境统一为 `/opt/sub2api/{sub2api-go,dashboard}`。  
恢复时：旧 `.env` → `sub2api-go/.env`，旧 `data/sub2api.db` → `sub2api-go/data/sub2api.db`。

---

相关文档：[OBSERVABILITY.md](./OBSERVABILITY.md)、[SCALING.md](./SCALING.md)。
