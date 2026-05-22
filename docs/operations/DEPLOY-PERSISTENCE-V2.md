# 部署教程：SQLite 持久化 + Admin 后台（v2）

> 适用：已跑通的生产环境（`43.134.8.202`、`cloudtoken.uk` / `api.cloudtoken.uk`）。  
> 本次发版**主要是后端** `sub2api-go`；未改前端时可跳过 dashboard 编译。  
> 通用发版流程仍见 [DEPLOY-FROM-LOCAL.md](./DEPLOY-FROM-LOCAL.md)。

---

## 你会得到什么

- 启动时自动建表 / 迁移：`account_ledger`、`request_logs`、`admin_audit_log`、`sync_outbox` 等
- 充值、扣费、建 Key、请求日志 **写穿** 进 `sub2api-go/data/sub2api.db`
- 新 Admin 接口：查用户、改余额、改状态、SQLite → Redis 恢复

---

## 三个位置

| 位置 | 做什么 |
|------|--------|
| **Mac** | 提交代码、`git push`、可选本地 `go build` 自测 |
| **GitHub** | `heyangshnu/sub2api` |
| **服务器 SSH** | `git pull`、编译、重启 `sub2api` |

---

## 第一步：Mac 本机（约 5 分钟）

### 1.1 确认改动已提交

```bash
cd /Users/heyang/Desktop/myProject/sub2api-full-code
git status
```

确认 **没有** 把 `sub2api-go/.env`、`*.db` 加入提交。

### 1.2 本地编译（可选但建议）

```bash
cd sub2api-go
go build -o bin/server ./cmd/server
echo "Go 编译 OK"
```

### 1.3 推到 GitHub

```bash
cd /Users/heyang/Desktop/myProject/sub2api-full-code
git add -A
git commit -m "feat: SQLite 持久化写穿、Admin 用户/余额管理"
git push origin main
```

记下提交号，便于回滚：

```bash
git rev-parse HEAD
```

---

## 第二步：登录服务器

```bash
ssh root@43.134.8.202
```

---

## 第三步：发版前备份（服务器，必做）

```bash
cd /opt/sub2api

# 记录当前可用版本
git rev-parse HEAD | tee /root/last-good-commit.txt

# 备份数据库（含即将迁移的 users / transactions）
cp -a sub2api-go/data/sub2api.db /root/backup-$(date +%Y%m%d-%H%M).db

# 备份 .env（勿提交 Git）
cp -a sub2api-go/.env /root/sub2api-go.env.backup-$(date +%Y%m%d-%H%M)
```

---

## 第四步：拉代码（服务器）

```bash
cd /opt/sub2api
git pull --ff-only origin main
```

若 `git pull` 失败且提示 `dashboard/next-env.d.ts` 冲突：

```bash
git restore dashboard/next-env.d.ts
git pull --ff-only origin main
```

---

## 第五步：核对 `.env`（服务器）

```bash
nano /opt/sub2api/sub2api-go/.env
```

**本次发版必须有的项**（没有就追加，不要用 Mac 上的 `.env` 整文件覆盖）：

```env
APP_ENV=production
PORT=3000
REDIS_URL=redis://127.0.0.1:6379

# 管理后台 API（改余额/状态用这个，务必强随机）
ADMIN_KEY=你的强管理密钥

JWT_SECRET=至少32位随机字符串

# 账户钱包
ACCOUNT_MONTHLY_GRANT_USD=0.1
REQUIRE_PAYMENT_BEFORE_CREATE_KEY=true
```

确认 Redis 在跑：

```bash
systemctl status redis-server --no-pager
```

---

## 第六步：编译并重启后端（服务器，核心）

```bash
cd /opt/sub2api/sub2api-go
go build -o bin/sub2api ./cmd/server

systemctl restart sub2api
systemctl status sub2api --no-pager
```

应看到 **`Active: active (running)`**。

### 6.1 看启动日志（确认迁移成功）

```bash
journalctl -u sub2api -n 50 --no-pager
```

正常情况：

- 有 `Store: redis+sqlite`（或含 sqlite）
- 有 `[Syncer] Started`
- **没有** `failed to run migrations` / `panic`

### 6.2 确认新表已创建（可选）

```bash
sqlite3 /opt/sub2api/sub2api-go/data/sub2api.db ".tables"
```

应能看到：`account_ledger`、`request_logs`、`admin_audit_log`、`sync_outbox` 等。

```bash
sqlite3 /opt/sub2api/sub2api-go/data/sub2api.db "SELECT COUNT(*) FROM account_ledger;"
```

若以前有 `transactions` 且带 `user_id`，启动后会自动回填到 `account_ledger`。

---

## 第七步：前端（本次可跳过）

若你**只发了后端**、没改 `dashboard/`，**不必** `npm run build` / 重启 dashboard。

若同时改了前端，再执行：

```bash
cd /opt/sub2api/dashboard
npm ci
NODE_ENV=production npm run build
systemctl restart dashboard
```

---

## 第八步：发版后验收

### 8.1 健康检查（Mac 或服务器）

```bash
curl -sS https://api.cloudtoken.uk/health
curl -sS -o /dev/null -w "ready=%{http_code}\n" https://api.cloudtoken.uk/health/ready
```

`ready=200` 表示 Redis 正常。

### 8.2 测 Admin 用户列表（把 ADMIN_KEY 换成你 `.env` 里的值）

```bash
export ADMIN_KEY='你的ADMIN_KEY'

curl -sS "https://api.cloudtoken.uk/admin/users?limit=5" \
  -H "X-Admin-Key: $ADMIN_KEY" | head -c 500
```

应返回 JSON，`users` 数组（可能为空，取决于是否已同步进 SQLite）。

### 8.3 浏览器

1. https://cloudtoken.uk 登录
2. 账户页余额、对话、建 Key 仍正常
3. 新注册用户：几分钟后在 SQLite 里能查到（写穿 + Syncer）

```bash
# 服务器上查用户（示例）
sqlite3 /opt/sub2api/sub2api-go/data/sub2api.db \
  "SELECT id, email, status, spendable_balance, created_at FROM users ORDER BY created_at DESC LIMIT 5;"
```

---

## 第九步：后台改余额 / 状态（上线后运维）

### 查用户 ID

```bash
curl -sS "https://api.cloudtoken.uk/admin/users?limit=20" \
  -H "X-Admin-Key: $ADMIN_KEY"
```

记下某个用户的 `id`（UUID）。

### 增加余额 0.1 美元（立即在 Redis 生效）

```bash
curl -sS -X PATCH "https://api.cloudtoken.uk/admin/users/<USER_ID>/balance" \
  -H "X-Admin-Key: $ADMIN_KEY" \
  -H "Content-Type: application/json" \
  -d '{"adjust_amount": 0.1, "note": "客服补偿"}'
```

用户刷新账户页应看到余额变化。

### 禁用用户

```bash
curl -sS -X PATCH "https://api.cloudtoken.uk/admin/users/<USER_ID>/status" \
  -H "X-Admin-Key: $ADMIN_KEY" \
  -H "Content-Type: application/json" \
  -d '{"status": "disabled", "note": "违规"}'
```

该用户再次登录应提示账号已禁用。

### 应急：你直接改了 SQLite 文件

```bash
# 改库后必须执行，否则线上 Redis 仍是旧余额
curl -sS -X POST "https://api.cloudtoken.uk/admin/users/<USER_ID>/reload-from-db" \
  -H "X-Admin-Key: $ADMIN_KEY"
```

---

## 回滚（出问题时）

```bash
cd /opt/sub2api
git checkout "$(cat /root/last-good-commit.txt)"
cd sub2api-go && go build -o bin/sub2api ./cmd/server
systemctl restart sub2api

# 若数据库被 migrate 搞乱，恢复备份
cp -a /root/backup-YYYYMMDD-HHMM.db sub2api-go/data/sub2api.db
systemctl restart sub2api
```

---

## 常见问题

| 现象 | 处理 |
|------|------|
| `git pull` 失败 | `git restore dashboard/next-env.d.ts` 再 pull |
| `go build` 失败 | 看报错；服务器需 `go version` ≥ 1.21 |
| `Store: redis` 无 sqlite | 检查 `data/` 目录权限；看 journal 里 SQLite 报错 |
| Admin 返回 401 | `X-Admin-Key` 与 `.env` 的 `ADMIN_KEY` 完全一致 |
| 用户表仍为空 | 老用户只在 Redis：等 1 分钟 Syncer，或新注册一个用户触发写穿 |
| 只改 `.env` 不生效 | 必须 `systemctl restart sub2api` |

---

## 相关文档

- 表结构说明：[DATABASE-SCHEMA.md](../product/DATABASE-SCHEMA.md)
- 日常发版：[DEPLOY-FROM-LOCAL.md](./DEPLOY-FROM-LOCAL.md)
- 全量装机：[PRODUCTION-DEPLOY.md](./PRODUCTION-DEPLOY.md)
