# 与 [all-api-hub](https://github.com/qixing-jk/all-api-hub) 的对比

## 产品定位

| 维度 | all-api-hub | 本项目 (sub2api) |
|------|-------------|------------------|
| 形态 | Chrome 浏览器扩展 | 自建 Go API 网关 + Next.js Dashboard |
| 目标用户 | 已在多个 **第三方中转站** 有账号的用户 | 在你站点 **注册、充值、发 Key** 的终端用户 |
| 数据归属 | 用户本地 / 扩展存储各站 Token | 你的 Redis + SQLite + Stripe |
| 是否替代后端 | **否**，只聚合已有 New-API / Sub2API 等 | **是**，完整计费、路由、账户钱包 |

**结论**：all-api-hub 是「多站点账号管家」；本项目是「单站点 SaaS 网关」。不是二选一，而是不同层级的产品。

## all-api-hub 主要能力（来自官方 README）

- 多中转站账号聚合：余额、用量看板
- 自动签到、密钥一键复制/使用
- 模型价格对比、可用性测试（如 `/v1/models`）
- 高级渠道管理（面向站长侧）

## 本项目已有、且与之重叠的部分

- Dashboard：Key 管理、用量曲线、请求日志、账户充值（Stripe）
- 账户钱包：USD 余额、月赠、首充解锁创建 Key
- 首页对话（JWT + 账户余额计费）
- OpenAI 兼容 `/v1/chat/completions`、`/v1/models`

## 本次从 all-api-hub 借鉴并已落地

1. **Key 连通性检测**：控制台「检测连通性」、创建 Key 后「测试 Key 连通性」（请求 `GET /v1/models`）。
2. **模型列表与配置一致**：`/v1/models` 与 `GET /dashboard/models` 均从 `PROVIDERS` 配置生成，不再写死模型名。
3. **注册用户写入 SQLite**：`RedisStore.CreateUser` / `UpdateUser` 成功后镜像到 SQLite，便于运维在 `users` 表查人（Redis 仍为运行时主库）。

## 适合后续择项（需产品确认）

| 能力 | 说明 | 复杂度 |
|------|------|--------|
| 模型价格表 UI | 展示 `model.DefaultPricing` 或配置价 | 中 |
| 多上游健康探测 | 定时 ping 各 Provider | 高 |
| 自动签到 | 仅适用于对接第三方站，不适用自建 | — |
| 多站聚合 | 与当前单租户 SaaS 目标不一致 | — |

## 相关代码

- 模型列表：`sub2api-go/internal/handler/models.go`
- 用户同步：`sub2api-go/internal/store/redis.go` → `syncUserToSQLite`
- 前端检测：`dashboard/src/components/api-keys-card.tsx`、`dashboard/src/lib/api.ts`
