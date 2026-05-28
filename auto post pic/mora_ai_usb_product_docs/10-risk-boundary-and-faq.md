# 10. 风险、边界与 FAQ

## 1. 最大风险列表

| 风险 | 严重度 | 说明 | 缓解 |
|---|---:|---|---|
| 用户害怕陌生 U 盘 | 高 | 海外用户对 USB 安全敏感 | 不 autorun、签名 App、Web Mode、安全说明 |
| 隐私承诺过度 | 高 | 如果宣传“完全本地”，但实际上传数据，会破坏信任 | 分模式透明说明 |
| 技术范围过大 | 高 | 文件索引、模型网关、硬件、跨平台都复杂 | MVP 严格收敛 |
| 硬件供应链拖慢 | 中高 | 硬件质检、物流、售后都会增加成本 | 初期小批量 Founder Edition |
| U 盘损坏/拔出导致数据坏 | 中高 | 工作区产品对数据损坏极敏感 | WAL、journal、Safe Eject、备份 |
| 模型成本失控 | 中高 | 长文档、多模型对比、API 会烧成本 | credits、限额、模型档位 |
| 与普通模型平台同质化 | 中 | 如果 U 盘不承载工作区，就会退化成中转站 | 聚焦身份、记忆、项目上下文 |
| 目标用户太散 | 中 | 每个职业都想服务会导致失败 | 首发顾问+创意自由职业者 |
| 中国模型叙事引发不信任 | 中 | 海外用户可能担心数据和稳定性 | 透明说明供应商、数据流、可选模型 |
| 售后复杂 | 中 | 硬件+软件+AI 的支持复杂度高 | 明确支持范围和诊断工具 |

## 2. 对外必须讲清楚的边界

### 边界一：不是完全离线大模型

推荐回答：

> MORA Drive is a local-first workspace. Your prompts, history, project index and workspace live on the drive. Advanced AI reasoning uses the MORA model gateway unless local mode is explicitly enabled.

### 边界二：不是任何电脑都建议插

推荐回答：

> Use Workspace Mode on computers you trust. For temporary or restricted computers, use Web Mode through the browser.

### 边界三：不是自动控制电脑

推荐回答：

> MORA helps you understand, search and generate work from your project context. It does not automatically control your computer in v1.

### 边界四：不是把所有文件都上传

推荐回答：

> Local indexing happens on the drive. When cloud reasoning is needed, MORA shows what context is being sent and why.

## 3. FAQ

### Q1：这不就是 AI token 中转站吗？

不是。模型网关是底层能力，但用户购买的是一个随身私人 AI 工作区。它包含本地索引、项目上下文、提示词、历史、credits 和模型访问能力。

### Q2：为什么一定要 U 盘？

因为 U 盘提供三个网页账号无法提供的感知：

1. 物理拥有感；
2. 工作区随身携带；
3. 插上/拔下的边界感。

U 盘不是分发渠道，而是工作区的物理化身。

### Q3：我可以不插 U 盘用吗？

可以。Web Mode 用于轻量访问和临时设备。但完整的本地项目索引、加密工作区和拔下即带走体验，需要 Workspace Mode。

### Q4：如果 U 盘丢了怎么办？

本地工作区是加密的。用户可以登录账号 revoke 设备。如果开启了加密备份，可以恢复到新设备。

### Q5：我的文件会被上传吗？

本地索引默认在设备上完成。只有在用户请求云端模型推理时，必要上下文才会发送到 MORA 模型网关。产品应在 UI 中明确提示和记录。

### Q6：为什么不用纯网页？

纯网页适合聊天和模型调用，但很难提供“工作区随身”“项目上下文本地化”“拔下即断开”的强感知。MORA Drive 的核心差异来自实体工作区。

### Q7：为什么不用浏览器插件？

浏览器插件可以做辅助入口，但不能承载完整项目工作区。MORA 可以后续做插件，但 U 盘工作区是主体验。

### Q8：为什么不一开始做企业版？

企业对 USB、安全、审计、合规要求高，销售周期长。MVP 更适合独立顾问、创意自由职业者、indie builder 等个人高意向用户。

### Q9：国内模型对海外用户有吸引力吗？

有，但不应作为第一层主叙事。第一层卖“随身私人 AI 工作区”，第二层卖“无需配置即可访问精选中国和全球模型”。

### Q10：怎么避免用户觉得是噱头？

U 盘必须真实承载：

- 本地加密工作区；
- 用户提示词；
- 项目上下文；
- 文件索引；
- 历史记录；
- 设备身份；
- 安全边界。

如果只放一个网页链接，那就是噱头。

## 4. 内部决策原则

遇到产品分歧时，用这几个问题判断：

1. 这个功能是否强化“随身私人 AI 工作区”？
2. 这个功能是否让用户少配置、少解释、少搜索？
3. 这个功能是否让 U 盘变得必要，而不是可有可无？
4. 这个功能是否能在 24 周 MVP 内稳定交付？
5. 这个功能是否会增加安全/合规承诺风险？

如果答案不清楚，推迟到 v2。
