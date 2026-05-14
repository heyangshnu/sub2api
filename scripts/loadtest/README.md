# 负载与并发压测（k6）

用于在**预发或本地**对 `POST /v1/chat/completions` 做冒烟与并发验证，配合计费与 Redis 监控观察延迟与错误率。

## 前置条件

- 安装 [k6](https://k6.io/docs/getting-started/installation/)
- 一枚有效的 **Sub2API** 原始 Key（`sk-sub2api-...`）
- 后端可访问的 base URL（默认 `http://127.0.0.1:3000`）

## 运行

```bash
export SUB2API_URL="http://127.0.0.1:3000"
export SUB2API_KEY="sk-sub2api-xxxxxxxx"
k6 run scripts/loadtest/k6_chat.js
```

可按需修改 `k6_chat.js` 中的 `vus`、`duration` 与请求体（模型名需为当前环境已配置的模型）。

## 注意

- 压测会产生**真实扣费**（预扣 + 实扣），请使用测试账户与小额度。
- 不要将生产 Key 提交到脚本或 CI 日志中。
