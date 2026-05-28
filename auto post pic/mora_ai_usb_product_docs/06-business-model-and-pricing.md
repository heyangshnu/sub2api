# 06. 商业模式与定价设计

## 1. 定价原则

MORA Drive 不能像普通 U 盘一样定价，也不能像普通 token 中转站一样只赚差价。它应该采用：

> **硬件建立信任与拥有感，订阅覆盖持续服务，credits 覆盖高成本消耗，任务包提升感知价值。**

## 2. 为什么不建议免费送 U 盘

早期想法中出现过“U 盘即门票 / 免费试用通行证”的模型，但如果产品主打隐私、数据主权和私人工作区，免费 U 盘会削弱信任。

用户会产生疑问：

- 为什么免费送我一个能插电脑的硬件？
- 里面有没有不安全的东西？
- 为什么一个隐私产品要靠免费硬件传播？

所以推荐：

> **U 盘是 premium privacy anchor，不是免费营销赠品。**

## 3. 推荐 SKU

| SKU | 价格 | 适合用户 | 包含内容 |
|---|---:|---|---|
| Starter Drive | $49 | 尝鲜用户、轻量创作者 | U 盘 + 30 天 Pro + 少量 credits + 基础任务包 |
| Pro Drive | $69 | 独立顾问、自由职业者 | 更高容量/更好外观 + 60 天 Pro + Consultant/Creator Pack |
| Builder Drive | $99 | 开发者、小团队 | API access + 更多 credits + 示例代码 + 模型网关能力 |
| Gift / Referral Card | $19–29 | 礼品与拉新 | 不含完整 U 盘，可作为 credits 激活卡 |

## 4. 订阅设计

### Solo Pro：$15–25/月

包含：

- 多模型聊天；
- 项目工作区；
- 文件索引；
- 基础云备份；
- 任务包；
- 每月基础 credits；
- API 轻量使用。

### Builder：$29–49/月

包含：

- 更高 API 限额；
- 模型路由；
- 使用日志；
- fallback；
- 成本控制。

### Team：后置

不建议 v1 主打 Team。可作为 v2：

- 团队项目空间；
- 成员权限；
- 共享任务包；
- 管理后台；
- 审计日志。

## 5. Credits 设计

Credits 用来覆盖：

- 高级模型调用；
- 长文件分析；
- 多模型对比；
- 图像/视频生成；
- API 调用；
- 超出订阅额度的消耗。

不要把 token 明细直接暴露给普通用户。建议包装成：

- Simple mode；
- Deep mode；
- Compare mode；
- Large file mode。

示例：

```text
Quick answer: ~1 credit
Deep analysis: ~5 credits
Compare 3 models: ~8 credits
Analyze large PDF: ~20 credits
Generate image: ~30 credits
```

## 6. 场景包商业化

任务包不只是功能，也是提高转化的商品包装。

### Consultant Pack

- Meeting brief；
- Proposal builder；
- Client follow-up；
- Risk summary；
- Requirements extractor；
- Client memory。

### Creator Pack

- Video hook generator；
- Script repurposer；
- Brand voice assistant；
- Client feedback digest；
- Reference collector；
- Launch copy pack。

### Builder Pack

- API starter；
- Model selector；
- Prompt evaluator；
- Cost estimator；
- Fallback template；
- OpenAI-compatible examples。

可以作为订阅内置，也可以单独售卖 $9–29。

## 7. 收入结构

MORA Drive 的收入不应只来自硬件。

| 收入项 | 作用 |
|---|---|
| 硬件销售 | 建立拥有感、回收硬件成本、筛选高意向用户 |
| 月订阅 | 主收入来源，覆盖产品服务和基础 AI 成本 |
| Credits | 高消耗用户增收，避免滥用 |
| 任务包 | 提升毛利和场景转化 |
| API 消耗 | 开发者和小团队增量收入 |
| Team Plan | v2 高 ARPU 来源 |

## 8. 定价叙事

不要说：

> 我们 token 便宜。

要说：

> **Less than one billable hour. A private AI workspace you can carry.**

中文理解：

> 不到一个小时服务费的价格，拥有一个能随身带走的私人 AI 工作区。

## 9. 用户购买理由

不同用户应看到不同价值锚点：

| 用户 | 价值锚点 |
|---|---|
| 顾问 | 少花 1 小时准备会议就回本 |
| 创作者 | 少丢一次客户上下文就值回票价 |
| 开发者 | 不用注册/配置多个模型 provider |
| 小团队 | 用更低成本测试 AI 工作流 |
| 礼品用户 | 送别人一套 AI 能力，而不是注册链接 |

## 10. 推荐首发价格

### 最稳方案

- Hardware：$59。
- Solo Pro：$19/月。
- 首月含 $10 credits。
- 额外 credits：$10 / $25 / $50 三档。

### 更激进方案

- Starter Drive：$39，低硬件门槛。
- Pro：$15/月。
- 用 credits 提高 ARPU。

### 更高端方案

- Premium Drive：$79。
- Pro：$25/月。
- 主打隐私、安全、顾问人群。

建议第一轮 A/B 测试 $49、$59、$69 三个硬件价格，观察“购买意愿 vs 信任感”的变化。
