# 可观测性与健康检查

本文说明 Sub2API Go 服务自带的运维端点，便于负载均衡探活、编排就绪探针与简易指标采集。

## HTTP 端点

| 路径 | 方法 | 说明 |
|------|------|------|
| `/health` | GET | JSON：进程存活、存储类型、Redis/SQLite ping 与延迟（毫秒级）。 |
| `/health/ready` | GET | 就绪探针：Redis 不可用时返回 **503**（若业务强依赖 Redis）。 |
| `/metrics` | GET | **OpenMetrics 文本**（`text/plain; version=0.0.4`），当前包含 `sub2api_http_requests_total` 等计数（按路径聚合，部分路径不计入，见下）。 |

公开端点无需认证；请勿在公网暴露敏感信息时把管理接口与调试端口一并放开。

## 指标说明

- **HTTP 请求计数**：中间件对进入路由的请求递增计数。为减少噪声与高基数，**以下路径默认不计入**：`/metrics`、`/health`、`/health/ready`、以及 Stripe 等 **`/webhook/*`**。
- 指标实现为手写 OpenMetrics 文本，**不依赖** `prometheus/client_golang`，便于在无外网环境下构建。

### 抓取示例

```bash
curl -sS http://127.0.0.1:3000/metrics
```

生产可将 Prometheus `scrape_configs` 指向该路径，或使用任何支持文本指标的采集器。

## 与 Nginx / 云 LB 的配合

- **存活（liveness）**：可指向 `/health`，只要进程与基本依赖可响应即可。
- **就绪（readiness）**：建议使用 **`/health/ready`**，在 Redis 故障时让编排系统停止向该实例转发流量。
- **真实客户端 IP**：请在 `.env` 中配置 **`TRUSTED_PROXIES`**（反代 CIDR 列表），以便 Gin 正确解析 `X-Forwarded-For`；否则 IP 白名单可能被伪造头绕过。

## Dashboard 与审计

用户侧「请求日志」与「按日消费」来自业务接口（JWT），与 `/metrics` 正交：`/metrics` 服务运维聚合，Dashboard 服务租户自助分析。详见 `HANDOVER.md` 与 `docs/operations/SCALING.md`。
