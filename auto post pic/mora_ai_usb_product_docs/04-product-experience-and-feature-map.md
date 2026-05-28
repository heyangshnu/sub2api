# 04. 产品体验与功能地图

## 1. 第一魔法时刻

MVP 的第一魔法时刻必须极其明确：

> **插上 U 盘，打开 App，它知道这是我的 AI、我的提示词、我的历史、我的额度、我的项目文件夹。**

这比“它有很多模型”更重要。

## 2. 首次使用体验

### Step 1：插入 U 盘

用户看到一个极简入口：

```text
Start.html
Open MORA Drive.app / .exe
README - Start Here
```

不应出现大量技术文件、配置文件或复杂目录。

### Step 2：打开启动页

启动页提供两种模式：

```text
Web Mode
No install. Open your workspace in browser.

Workspace Mode
Run the signed portable app on trusted computers.
```

### Step 3：激活

用户输入或扫码激活：

- 设备序列号；
- 账号登录；
- 本地工作区密码；
- 可选恢复码；
- 可选云备份。

### Step 4：选择工作方式

首页不应该是空白聊天框，而是任务入口：

```text
Welcome back.

1. Continue recent project
2. Ask across my files
3. Chat with AI
4. Compare models
5. Use a task pack
6. Check credits
7. API access
```

## 3. 两种模式

### A. Web Mode

适合任何可信电脑，不运行本地程序。

能力：

- 登录 Web/PWA。
- 使用聊天、模型对比、提示词库、credits。
- 查看云端同步的轻量历史。
- 激活和充值。

限制：

- 不读取本机文件。
- 不建立完整本地索引。
- 不承诺拔下后完全无痕，因为浏览器可能有缓存。

### B. Workspace Mode

适合用户自己的电脑或可信电脑，运行签名便携 App。

能力：

- 本地加密工作区。
- 文件索引。
- 项目模式。
- 本地提示词、历史、任务包。
- 模型网关调用。
- 拔出前安全同步与完整性检查。

## 4. 核心功能模块

### 4.1 Workspace Home

显示：

- 最近项目；
- 最近变化；
- 待处理任务；
- 上次 AI 处理结果；
- credits 余额；
- 推荐下一步。

目标：让用户感到“AI 已经在等我”，不是“我又要重新提问”。

### 4.2 Project Mode

每个项目可以绑定一个文件夹或多个数据源。

项目首页应显示：

- 项目摘要；
- 最近更新；
- 关键文件；
- 关键人物/客户要求；
- AI 推荐下一步；
- 常用 prompt。

### 4.3 Route Mode

用户不用知道文件在哪，直接问：

- “上次 Client A 说 logo 颜色不满意具体是哪句话？”
- “帮我找跟这个 proposal 相关的资料。”
- “这个项目还有哪些未处理反馈？”

Route Mode 不应该承诺全网全应用搜索。MVP 范围内只搜索：

- U 盘工作区；
- 用户授权的本地项目文件夹；
- 已接入的数据源；
- 已同步的历史记录。

### 4.4 AI Chat

普通聊天仍然需要，但不能是唯一入口。

建议支持：

- 默认智能模型；
- 快速模式；
- 深度模式；
- 多模型对比；
- 中国模型专区；
- 自定义模型偏好。

### 4.5 Task Packs

任务包让用户不必懂 prompt。

首批建议：

#### Consultant Pack

- 准备客户会议；
- 生成 proposal；
- 整理客户需求；
- 生成 follow-up 邮件；
- 提炼风险点；
- 比较方案优劣。

#### Creator Pack

- 生成短视频 hook；
- 长文拆社媒内容；
- 提案故事线；
- 品牌语气统一；
- 素材摘要；
- 客户反馈整理。

#### Builder Pack

- 生成 API 示例；
- 选择模型；
- 估算成本；
- prompt 测试；
- fallback 配置。

### 4.6 Credits Wallet

不要让普通用户看到 token 细节。

建议显示为：

```text
Simple answer: ~1 credit
Deep reasoning: ~5 credits
Compare 3 models: ~8 credits
Large file analysis: ~20 credits
```

用户需要的是“可控感”，不是 token 账单焦虑。

### 4.7 API Access

面向开发者提供：

- OpenAI-compatible endpoint；
- API key；
- 模型列表；
- 价格与剩余额度；
- 使用日志；
- 简单 fallback；
- 示例代码。

## 5. MVP 功能范围

### 必须做

- U 盘启动页；
- 设备激活和账号绑定；
- 加密本地工作区；
- Tauri 便携 App；
- 多模型聊天；
- credits 钱包；
- 本地历史和提示词库；
- 项目文件夹索引；
- 文件问答；
- 2 个任务包；
- API key 基础能力。

### 暂不做

- 完全离线大模型；
- 自动控制用户电脑；
- 全云盘深度集成；
- 团队协作；
- 企业 SSO；
- 手机 App；
- 复杂 Agent 编排；
- 10 个以上模型的复杂市场页；
- U 盘自动运行。

## 6. 体验原则

1. **不空白**：打开后应有项目、任务、历史和推荐下一步。
2. **不配置**：用户不应该看到 provider、endpoint、temperature 等技术项。
3. **不夸张**：清楚说明哪些在本地，哪些会调用云端模型。
4. **不打扰**：索引、同步、扣费都要可见但不过度干扰。
5. **不留痕**：在 Workspace Mode 中尽量做到拔下后主要状态随盘带走。
