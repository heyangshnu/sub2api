# 05. 技术架构与安全边界

## 1. 技术总原则

MORA Drive 应采用“本地工作区 + 云端模型网关”的混合架构。

- **本地承载**：身份、加密工作区、文件索引、提示词、历史、项目上下文、缓存。
- **云端承载**：模型网关、计费、账号、订阅、可选备份、高成本推理。
- **可选本地能力**：轻量搜索、历史查看、索引维护、小模型 fallback。

不要在 v1 承诺完整离线大模型。应把边界说清楚：

> **Your workspace is portable and local-first. Advanced AI reasoning uses the MORA model gateway unless local mode is explicitly enabled.**

## 2. 推荐架构

```text
┌──────────────────────────────────────────────┐
│                  MORA Drive USB               │
│                                              │
│  ┌──────────────┐   ┌────────────────────┐    │
│  │ Start Page   │   │ Portable App        │    │
│  │ Web/PWA Link │   │ Tauri + React UI    │    │
│  └──────────────┘   └────────────────────┘    │
│                                              │
│  ┌────────────────────────────────────────┐   │
│  │ Local Encrypted Workspace              │   │
│  │ - SQLite state                         │   │
│  │ - Tantivy full-text index              │   │
│  │ - Vector index                         │   │
│  │ - Prompts / history / task packs       │   │
│  │ - Project metadata                     │   │
│  └────────────────────────────────────────┘   │
└──────────────────────────────────────────────┘
                    │
                    ▼
┌──────────────────────────────────────────────┐
│              MORA Cloud / Model Gateway       │
│ - Auth / device binding                       │
│ - Credits / billing                           │
│ - Model routing                               │
│ - Domestic model API relay                    │
│ - Global model fallback                       │
│ - API endpoint                                │
│ - Optional encrypted backup                   │
└──────────────────────────────────────────────┘
```

## 3. 本地组件建议

| 组件 | 推荐 | 说明 |
|---|---|---|
| 桌面壳 | Tauri v2 | 体积小、Rust 后端、比 Electron 更适合安全控制 |
| UI | React/TypeScript | 便于快速迭代 |
| 本地状态 | SQLite WAL | 稳定、可恢复、适合 U 盘拔出场景 |
| 全文检索 | Tantivy | Rust 生态成熟，适合嵌入式检索 |
| 向量索引 | LanceDB / sqlite-vec / Qdrant local 需评估 | MVP 可以保守选型，优先稳定 |
| 文件监听 | notify-rs | 跨平台监听文件变化 |
| 文档解析 | unstructured / pymupdf / markitdown 等组合 | 先覆盖 PDF、docx、txt、md、html |
| 模型调用 | MORA gateway | 产品控制模型路由，不把模型 provider 暴露给用户 |

## 4. U 盘数据结构建议

```text
/MORA-Drive
  /Start.html
  /README.md
  /Apps
    /macOS/MORA.app
    /Windows/MORA.exe
  /Workspace
    /vault.sqlite
    /index
    /projects
    /prompts
    /history
    /task-packs
  /Recovery
    /recovery-info.txt
  /Logs
    /local-diagnostic.log
```

实际发布时，技术目录可隐藏，用户只看到 Start、App、README。

## 5. 加密与设备绑定

最低要求：

- 本地工作区加密。
- 用户设置工作区密码。
- 设备序列号与账号绑定。
- 支持远程 revoke。
- 支持恢复码。
- 支持丢失 U 盘后的账号冻结。
- 不把明文 API key 存在 U 盘。

推荐表达：

```text
No autorun.
Signed app.
Encrypted local workspace.
Transparent network behavior.
Revoke lost drives.
Optional encrypted cloud backup.
```

## 6. USB 拔出安全

U 盘产品最容易出问题的是突然拔出导致索引或数据库损坏。

建议策略：

1. SQLite 使用 WAL 模式。
2. 索引提交前写 journal。
3. App 内提供明显的 “Safe Eject” 按钮。
4. Safe Eject 执行：flush → sync → close handles → OS eject。
5. 下次启动做完整性检查。
6. 如发现未完成写入，回滚到上一个稳定版本。

用户可见文案：

> **Click Safe Eject before unplugging to protect your workspace. If you forget, MORA will run a repair check next time.**

## 7. 模型网关边界

MORA 模型网关的作用：

- 统一国内模型接口；
- 提供 OpenAI-compatible endpoint；
- 管理 credits；
- 做模型路由与 fallback；
- 隐藏供应商复杂性；
- 降低用户配置成本。

但必须透明：

- 哪些请求会发送到云端；
- 文件是否会上传；
- 文件上传后是否保存；
- 是否用于训练；
- 用户如何删除；
- API 失败如何处理；
- 高级模型如何扣费。

## 8. 隐私叙事与真实技术必须一致

如果宣传“数据不离开 U 盘”，技术上就必须真的做到不上传文件内容。

更稳妥的分层表达：

| 模式 | 数据处理 | 可宣传点 |
|---|---|---|
| Local Search Mode | 文件索引和搜索本地完成 | 文件不上传也能搜索 |
| Cloud Reasoning Mode | 用户选中的片段/问题发送到模型网关 | 透明提示，会消耗 credits |
| Web Mode | 云端轻量工作区 | 适合临时使用，不等于完整本地模式 |
| Backup Mode | 加密备份到云端 | 可选、用户控制 |

## 9. 安全风险与缓解

| 风险 | 缓解 |
|---|---|
| 用户害怕陌生 U 盘 | 不 autorun，签名 App，公开安全说明，提供 Web Mode |
| 企业电脑禁用 USB | 提供 Web/PWA 模式，不把企业作为首发主战场 |
| U 盘丢失 | 本地加密、远程 revoke、恢复码 |
| 上游模型不稳定 | fallback、模型状态页、credits 退回策略 |
| API 成本失控 | 模型档位、预算上限、速率限制 |
| 数据误上传 | 明确模式切换、上传前提示、默认最小必要上下文 |
| 跨平台兼容差 | Mac-first + Windows 快速跟进；文件系统用 exFAT 或专用加密容器需评估 |

## 10. 技术路线建议

### v1：可信电脑工作区

- Mac + Windows。
- 本地工作区。
- 模型网关。
- 项目文件夹索引。
- 2 个任务包。

### v1.5：更强安全与恢复

- 加密云备份。
- 丢失设备 revoke。
- 完整诊断工具。
- 安全白皮书。

### v2：团队与本地增强

- Team workspace。
- 小模型本地 fallback。
- 更多数据源连接。
- 更强 API gateway。

## 11. 最重要的技术边界

对外不要说：

> 完全本地私有部署的大模型。

应说：

> 随身、本地优先的私人 AI 工作区；复杂推理通过已配置好的模型网关完成，用户无需自行部署。用户可以清楚选择哪些资料用于云端推理。
