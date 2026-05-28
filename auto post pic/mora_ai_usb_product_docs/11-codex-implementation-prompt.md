# 11. 可直接给 Codex / Claude Code 的实现提示词

下面提示词用于启动 MORA Drive MVP 的产品资料整理、原型和工程落地。可直接复制给 Codex/Claude Code。

```text
你现在负责把 MORA Drive MVP 从产品文档推进到可运行原型。请严格遵循以下目标和边界。

项目定位：
MORA Drive 是一个装在 U 盘里的随身私人 AI 工作区，不是普通 AI token 中转站。用户插入 U 盘后，可打开自己的 AI 工作区：提示词、历史、项目上下文、本地文件索引、模型额度、模型网关和基础 API 能力。复杂模型推理通过 MORA Gateway 调用，工作区和索引尽量本地优先。

核心魔法时刻：
用户插上 U 盘，打开 App，它知道这是我的 AI、我的提示词、我的历史、我的额度、我的项目文件夹。用户可以打开一个项目文件夹，看到项目摘要，向 AI 提问，并在拔出前安全保存状态。

请先完成以下工作，不要一上来盲目写代码：
1. 阅读并理解 docs 中所有产品文档，特别是定位、功能地图、技术架构、MVP 路线和风险边界。
2. 输出一份 implementation plan，明确目录结构、模块边界、数据结构、技术选型、验收标准。
3. 明确哪些属于 MVP，哪些必须 descoped。
4. 先设计端到端用户路径，再实现模块。

MVP 必须包含：
1. U 盘目录结构原型：Start.html、README、Apps、Workspace、Recovery、Logs。
2. Tauri + React/TypeScript 桌面 App 骨架。
3. 本地加密 workspace 的最小实现。
4. SQLite 状态存储，开启 WAL。
5. 项目文件夹导入与基础 metadata 存储。
6. 文本/Markdown/PDF 的基础解析与全文索引。
7. Project Mode：显示项目摘要、最近文件、可提问入口。
8. Chat/Ask across files：先用 mock model 或可替换 gateway client。
9. Credits 钱包 UI：先 mock，接口保留。
10. Safe Eject 流程：flush、sync、close handles，并显示状态。
11. 基础日志和诊断信息。

MVP 不要做：
1. 完全离线大模型。
2. 自动控制用户电脑。
3. 团队协作。
4. 企业 SSO。
5. 多云盘深度集成。
6. 复杂 Agent 框架。
7. 过多模型市场页。
8. 自动运行 U 盘程序。

架构建议：
- Frontend: React + TypeScript。
- Shell: Tauri v2。
- Backend core: Rust。
- State: SQLite WAL。
- Full-text index: Tantivy 或先用 SQLite FTS5 做 MVP。
- Gateway: 先定义 OpenAI-compatible client interface，可 mock。
- Workspace encryption: MVP 可先做接口和本地密码保护，后续升级为标准加密容器。

验收要求：
1. 在 Mac 和 Windows 至少有明确运行方案。
2. 启动后用户不看到技术配置项。
3. 用户能创建项目、导入文件、搜索、提问。
4. 用户能看到 credits mock 和模型调用记录。
5. 拔出/异常关闭后，下一次启动能恢复并提示 integrity check。
6. 所有关键行为有日志。
7. 文案不能承诺完全本地大模型，只能说 local-first workspace + gateway reasoning。

输出格式：
1. 先输出 implementation plan。
2. 再输出文件树。
3. 再开始创建代码。
4. 每完成一个模块，运行对应自测。
5. 最后输出端到端验收记录和未完成事项。

质量要求：
- 不允许用临时补丁掩盖架构问题。
- 不允许把核心逻辑写死在 UI 里。
- 不允许没有日志和错误处理。
- 不允许跳过安全边界说明。
- 不允许把 mock 伪装成真实完成。
```
