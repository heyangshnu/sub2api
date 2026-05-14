# 规模化与多节点部署指引

当前默认架构：**单节点 Gin + Redis（计费与 Key）+ SQLite（用户/审计辅助）**。在流量与数据量上升时，可按下列顺序演进，避免一次性大改。

## 1. 入口与边缘（Cloudflare）

- **WAF / Bot 管理**：在公网前启用 Cloudflare（或同类）规则，缓解滥用与撞库；与后端 **IP 白名单**、**频次限制** 互补。
- **CDN**：对 Dashboard 静态资源有效；API 长连接与流式响应一般不走缓存。
- **TLS 终止**：可在边缘终止 HTTPS，回源使用证书或内网隧道；注意 **WebSocket / SSE** 与 **流式 chat** 的超时与缓冲设置。

## 2. 多实例 API（无状态层）

- 多副本 Gin 前挂 **负载均衡**。要求：
  - 所有实例共享同一 **`REDIS_URL`**（计费、Key、流水、请求日志列表等均依赖 Redis）。
  - **JWT_SECRET**、**STRIPE_WEBHOOK_SECRET** 等密钥各实例一致。
- 使用 **`GET /health/ready`** 作为就绪探针，Redis 不可用时摘除实例。
- 在 `.env` 配置 **`TRUSTED_PROXIES`**（反代 CIDR），与 Nginx `real_ip` / `X-Forwarded-For` 策略一致，避免 IP 白名单被伪造。

## 3. Redis 高可用

- 生产建议使用托管 Redis（主从/集群）与持久化策略；评估 **key 过期** 对「交易流水扫描」「按日聚合」的影响（当前部分分析依赖 Redis 内 key 的存活时间）。
- 监控：内存、连接数、延迟、主从切换次数。

## 4. SQLite → PostgreSQL（可选）

- SQLite 适合单机低并发写入；多节点写同一 SQLite 文件**不可取**。
- 迁移路径概要：
  1. 引入 PostgreSQL 连接与迁移（用户表、邀请码等当前落在 SQLite 的部分）。
  2. `RedisStore` 构造中保留 Redis 为主存储，SQLite 替换为 PG 或逐步下线 SQLite 依赖。
  3. 双写/回填窗口内做数据校验（可用 `go run ./cmd/sub2api-check -key <raw>` 做抽样对账，见 `sub2api-go/README.md`）。

具体表结构以 `docs/specs/data/entities.md` 与代码为准。

## 5. 观测与容量

- 抓取 **`/metrics`** 与 LB 状态码分布。
- 压测脚本见仓库 **`scripts/loadtest/`**（k6）；先在预发环境固定 **`SUB2API_URL`** 与测试 Key，再逐步提高并发。

## 6. Stripe Webhook

- 多实例时 webhook 可能被任一实例处理；依赖 **幂等键**（Stripe `event.id` 等）避免重复入账。确保各实例共享 Redis 且幂等逻辑一致。

---

更细的端口与 systemd 示例见根目录 **`HANDOVER.md`**。
